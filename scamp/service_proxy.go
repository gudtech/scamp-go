package scamp

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	"strconv"

	"errors"
	"fmt"
	"strings"

	"sync"

	"net"
	u "net/url"
)

// Example:
// {"vmin":0,"vmaj":4,"acsec":[[7,"background"]],"acname":["_evaluate","_execute","_evaluate","_execute","_munge","_evaluate","_execute"],"acver":[[7,1]],"acenv":[[7,"json,jsonstore,extdirect"]],"acflag":[[7,""]],"acns":[[2,"Channel.Amazon.FeedInterchange"],[3,"Channel.Amazon.InvPush"],[2,"Channel.Amazon.OrderImport"]]}
type ServiceProxyDiscoveryExtension struct {
	Vmin   int           `json:"vmin"`
	Vmaj   int           `json:"vmaj"`
	AcSec  []interface{} `json:"acsec"`
	AcName []string      `json:"acname"`
	AcVer  []interface{} `json:"acver"`
	AcEnv  []interface{} `json:"acenv"`
	AcFlag []interface{} `json:"acflag"`
	AcNs   []interface{} `json:"acns"`
}

type ServiceProxy struct {
	version          int
	ident            string
	sector           string
	weight           int
	announceInterval int
	connspec         string
	protocols        []string
	classes          []ServiceProxyClass

	extension *ServiceProxyDiscoveryExtension

	rawClassRecords []byte
	rawCert         []byte
	rawSig          []byte

	timestamp HighResTimestamp

	clientM sync.Mutex
	client  *Client
}

func (sp ServiceProxy) GetClient() (client *Client, err error) {
	sp.clientM.Lock()
	defer sp.clientM.Unlock()

	if sp.client == nil {
		var url *u.URL
		url, err = u.Parse(sp.connspec)
		if err != nil {
			return nil, err
		}

		sp.client, err = Dial(url.Host)
	}

	client = sp.client

	return
}

func (sp ServiceProxy) Ident() string {
	return sp.ident
}

func (sp ServiceProxy) BaseIdent() string {
	baseAndRest := strings.SplitN(sp.ident, ":", 2)
	if len(baseAndRest) != 2 {
		return sp.ident
	} else {
		return baseAndRest[0]
	}
}

func (sp ServiceProxy) ShortHostname() string {
	url, err := u.Parse(sp.connspec)
	if err != nil {
		panic("BOMBING OUT")
		return sp.connspec
	}

	hostParts := strings.Split(url.Host, ":")
	if len(hostParts) != 2 {
		return sp.connspec
	}
	host := hostParts[0]

	names, err := net.LookupAddr(host)
	if err != nil {
		return host
	} else if len(names) == 0 {
		return host
	}

	return names[0]
}

func (sp ServiceProxy) ConnSpec() string {
	return sp.connspec
}

func (sp ServiceProxy) Sector() string {
	return sp.sector
}

func (sp *ServiceProxy) Classes() []ServiceProxyClass {
	return sp.classes
}

type ServiceProxyClass struct {
	className string
	actions   []ActionDescription
}

func (spc ServiceProxyClass) Name() string {
	return spc.className
}

func (spc ServiceProxyClass) Actions() []ActionDescription {
	return spc.actions
}

type ActionDescription struct {
	actionName string
	crudTags   string
	version    int
}

func (ad ActionDescription) Name() string {
	return ad.actionName
}

func (ad ActionDescription) Version() int {
	return ad.version
}

func ServiceAsServiceProxy(serv *Service) (proxy *ServiceProxy) {
	proxy = new(ServiceProxy)
	proxy.version = 3
	proxy.ident = serv.name
	proxy.sector = serv.sector
	proxy.weight = 1
	proxy.announceInterval = defaultAnnounceInterval * 500
	proxy.connspec = fmt.Sprintf("beepish+tls://%s:%d", serv.listenerIP.To4().String(), serv.listenerPort)
	proxy.protocols = make([]string, 1, 1)
	proxy.protocols[0] = "json"
	proxy.classes = make([]ServiceProxyClass, 0)
	proxy.rawClassRecords = []byte("rawClassRecords")
	proxy.rawCert = []byte("rawCert")
	proxy.rawSig = []byte("rawSig")

	// { "Logger.info": [{ "name": "blah", "callback": foo() }] }
	for classAndActionName, serviceAction := range serv.actions {
		actionDotIndex := strings.LastIndex(classAndActionName, ".")
		// TODO: this is the only spot that could fail? shouldn't happen in any usage...
		if actionDotIndex == -1 {
			panic(fmt.Sprintf("bad action name: `%s` (no dot found)", classAndActionName))
		}
		className := classAndActionName[0:actionDotIndex]

		actionName := classAndActionName[actionDotIndex+1 : len(classAndActionName)]

		newServiceProxyClass := ServiceProxyClass{
			className: className,
			actions:   make([]ActionDescription, 0),
		}

		newServiceProxyClass.actions = append(newServiceProxyClass.actions, ActionDescription{
			actionName: actionName,
			crudTags:   serviceAction.crudTags,
			version:    serviceAction.version,
		})

		proxy.classes = append(proxy.classes, newServiceProxyClass)

	}

	timestamp, err := Gettimeofday()
	if err != nil {
		Error.Printf("error with high-res timestamp: `%s`", err)
		return nil
	}
	proxy.timestamp = timestamp

	return
}

func NewServiceProxy(classRecordsRaw []byte, certRaw []byte, sigRaw []byte) (proxy *ServiceProxy, err error) {
	proxy = new(ServiceProxy)
	proxy.rawClassRecords = classRecordsRaw
	proxy.rawCert = certRaw
	proxy.rawSig = sigRaw
	proxy.protocols = make([]string, 0)

	var classRecords []json.RawMessage
	err = json.Unmarshal(classRecordsRaw, &classRecords)
	if err != nil {
		return
	}
	if len(classRecords) != 9 {
		err = errors.New(fmt.Sprintf("expected 9 entries in class record. got %d", len(classRecords)))
	}

	// OMG, position-based, heterogenously typed values in an array suck to deal with.
	err = json.Unmarshal(classRecords[0], &proxy.version)
	if err != nil {
		return
	}

	err = json.Unmarshal(classRecords[1], &proxy.ident)
	if err != nil {
		return
	}

	err = json.Unmarshal(classRecords[2], &proxy.sector)
	if err != nil {
		return
	}

	err = json.Unmarshal(classRecords[3], &proxy.weight)
	if err != nil {
		return
	}

	err = json.Unmarshal(classRecords[4], &proxy.announceInterval)
	if err != nil {
		return
	}

	err = json.Unmarshal(classRecords[5], &proxy.connspec)
	if err != nil {
		return
	}

	var rawProtocols []*json.RawMessage
	err = json.Unmarshal(classRecords[6], &rawProtocols)
	if err != nil {
		return
	}

	// Skip object-looking stuff. We only care about strings for now
	for _, rawProtocol := range rawProtocols {
		var tempStr string
		err := json.Unmarshal(*rawProtocol, &tempStr)
		if err != nil {

			var extension ServiceProxyDiscoveryExtension
			err = json.Unmarshal(*rawProtocol, &extension)
			if err != nil {
				fmt.Printf("could not parse rawProtocol: %s\n", string(*rawProtocol))
				continue
			}

			proxy.extension = &extension
		} else {
			proxy.protocols = append(proxy.protocols, tempStr)
		}
	}

	// fmt.Printf("proxy.protocols: %s\n", proxy.protocols)

	var rawClasses [][]json.RawMessage
	err = json.Unmarshal(classRecords[7], &rawClasses)
	if err != nil {
		return
	}
	classes := make([]ServiceProxyClass, len(rawClasses), len(rawClasses))
	proxy.classes = classes

	for i, rawClass := range rawClasses {
		if len(rawClass) < 2 {
			err = errors.New(fmt.Sprintf("expected rawClass to have at least 2 entries. was: `%s`", rawClass))
			return nil, err
		}

		err = json.Unmarshal(rawClass[0], &classes[i].className)
		if err != nil {
			return nil, err
		}

		rawActionsSlice := rawClass[1:]
		classes[i].actions = make([]ActionDescription, len(rawActionsSlice), len(rawActionsSlice))

		for j, rawActionSpec := range rawActionsSlice {
			var actionsRawMessages []json.RawMessage
			err = json.Unmarshal(rawActionSpec, &actionsRawMessages)
			if err != nil {
				Error.Printf("could not parse rawActionSpec: %s", rawActionSpec)
				return nil, err
			} else if len(actionsRawMessages) != 2 && len(actionsRawMessages) != 3 {
				err = errors.New(fmt.Sprintf("expected action spec to have 2 or 3 entries. got `%s` (%d)", actionsRawMessages, len(actionsRawMessages)))
			}

			err = json.Unmarshal(actionsRawMessages[0], &classes[i].actions[j].actionName)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(actionsRawMessages[1], &classes[i].actions[j].crudTags)
			if err != nil {
				return nil, err
			}

			// TODO: it's gross that some of the services announce version
			// as a string.
			if len(actionsRawMessages) < 3 {
				// TODO: safe to assume a version-less thing is version 0?
				classes[i].actions[j].version = 1
			} else {
				err = json.Unmarshal(actionsRawMessages[2], &classes[i].actions[j].version)
				if err != nil {
					var versionStr string
					err = json.Unmarshal(actionsRawMessages[2], &versionStr)
					if err != nil {
						return nil, err
					}

					versionInt, err := strconv.ParseInt(versionStr, 10, 64)
					if err != nil {
						return nil, err
					}

					classes[i].actions[j].version = int(versionInt)
				}
			}
		}
	}

	proxy.client = nil // we connect on demand
	return
}

// 1) Verify signature of classRecords
// 2) Make sure the fingerprint is in authorized_services
// 3) Filter announced actions against authorized actions
func (proxy *ServiceProxy) Validate() (err error) {
	_, err = proxy.validateSignature()
	if err != nil {
		return
	}

	// See if we have this fingerprint in our authorized_services
	// TODO

	return
}

func (proxy *ServiceProxy) validateSignature() (hexSha1 string, err error) {
	decoded, _ := pem.Decode(proxy.rawCert)
	if decoded == nil {
		err = errors.New(fmt.Sprintf("could not find valid cert in `%s`", proxy.rawCert))
		return
	}

	// Put pem in form useful for fingerprinting
	cert, err := x509.ParseCertificate(decoded.Bytes)
	if err != nil {
		err = fmt.Errorf("failed to parse certificate: `%s`", err)
		return
	}

	pkixInterface := cert.PublicKey
	rsaPubKey, ok := pkixInterface.(*rsa.PublicKey)
	if !ok {
		err = errors.New("could not cast parsed value to rsa.PublicKey")
		return
	}

	err = VerifySHA256(proxy.rawClassRecords, rsaPubKey, proxy.rawSig, false)
	if err != nil {
		return
	}

	hexSha1 = sha1FingerPrint(cert)
	return
}

func (proxy *ServiceProxy) GetConnection() (client *Client, err error) {
	if proxy.client != nil {
		client = proxy.client
		return
	}

	proxy.client, err = Dial(proxy.connspec)
	if err != nil {
		return
	}

	return
}

func (proxy *ServiceProxy) MarshalJSON() (b []byte, err error) {
	arr := make([]interface{}, 9)
	arr[0] = &proxy.version
	arr[1] = &proxy.ident
	arr[2] = &proxy.sector
	arr[3] = &proxy.weight
	arr[4] = &proxy.announceInterval
	arr[5] = &proxy.connspec
	arr[6] = &proxy.protocols

	// TODO: move this to two MarshalJSON interfaces for `ServiceProxyClass` and `ActionDescription`
	// doing so should remove manual copies and separate concerns
	//
	// Serialize actions in this format:
	// 	["bgdispatcher",["poll","",1],["reboot","",1],["report","",1]]
	classSpecs := make([][]interface{}, len(proxy.classes), len(proxy.classes))
	for i, class := range proxy.classes {
		entry := make([]interface{}, 1+len(class.actions), 1+len(class.actions))
		entry[0] = class.className
		for j, action := range class.actions {
			actions := make([]interface{}, 3, 3)

			actionNameCopy := make([]byte, len(action.actionName))
			copy(actionNameCopy, action.actionName)
			actions[0] = string(actionNameCopy)
			actions[1] = &action.crudTags
			actions[2] = &action.version
			entry[j+1] = &actions
		}

		classSpecs[i] = entry
	}
	arr[7] = &classSpecs
	arr[8] = &proxy.timestamp

	return json.Marshal(arr)
}
