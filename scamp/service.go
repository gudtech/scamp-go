package scamp

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os" // "encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	yaml "gopkg.in/yaml.v2"
)

var livenessDirPath = "/backplane/running-services/"

// Two minute timeout on clients
var msgTimeout = time.Second * 120

// statsdInterval defaults to 500 milliseconds
var statsdInterval = time.Millisecond * 500

// ServiceActionFunc represents a service callback
type ServiceActionFunc func(*Message, *Client)

// ServiceAction interface
type ServiceAction struct {
	callback ServiceActionFunc
	crudTags string
	version  int
}

// Service represents a scamp service
type Service struct {
	serviceSpec string
	sector      string
	name        string
	humanName   string

	listener     net.Listener
	listenerIP   net.IP
	listenerPort int

	actions   map[string]*ServiceAction
	isRunning bool

	clientsM sync.Mutex
	clients  []*Client

	// requests      ClientChan

	cert    tls.Certificate
	pemCert []byte // just a copy of what was read off disk at tls cert load time

	// stats
	statsCloseChan      chan bool
	connectionsAccepted uint64
	dependencies        Resources
	statsStopSig        chan bool
}

type Resources []*Resource

// NewService initializes and returns pointer to a new scamp service
func NewService(sector string, serviceSpec string, humanName string) (*Service, error) {
	crtPath := DefaultConfig().ServiceCertPath(humanName)
	keyPath := DefaultConfig().ServiceKeyPath(humanName)

	var err error

	if crtPath == nil || keyPath == nil {
		err = fmt.Errorf("could not find valid crt/key pair for service %s (`%s`,`%s`)", humanName, crtPath, keyPath)
		return nil, err
	}

	// Load keypair for tls socket library to use
	keypair, err := tls.LoadX509KeyPair(string(crtPath), string(keyPath))
	if err != nil {
		return nil, err
	}

	// Load certificate as bytes
	pemCert, err := ioutil.ReadFile(string(crtPath))
	if err != nil {
		return nil, err
	}

	return NewServiceExplicitCert(sector, serviceSpec, humanName, keypair, pemCert)
}

// NewServiceExplicitCert intializes and returns pointer to a new scamp service,
// with an explicitly specified certificate rather than an implicitly discovered one.
// keypair is a TLS certificate, and pemCert is the raw bytes of an X509 certificate.
func NewServiceExplicitCert(sector string, serviceSpec string, humanName string, keypair tls.Certificate, pemCert []byte) (service *Service, err error) {
	if len(humanName) > 18 {
		err = fmt.Errorf("name `%s` is too long, must be less than 18 bytes", humanName)
		return
	}

	service = new(Service)
	service.sector = sector
	service.serviceSpec = serviceSpec
	service.humanName = humanName
	service.generateRandomName()

	service.actions = make(map[string]*ServiceAction)

	service.cert = keypair

	// Load cert in to memory for announce packet writing
	service.pemCert = bytes.TrimSpace(pemCert)

	// Finally, get ready for incoming requests
	err = service.listen()
	if err != nil {
		return
	}

	service.statsCloseChan = make(chan bool)
	service.statsStopSig = make(chan bool)
	// go PrintStatsLoop(serv, time.Duration(15)*time.Second, serv.statsCloseChan)

	// Trace.Printf("done initializing service")
	return
}

// TODO: port discovery and interface/IP discovery should happen here
// important to set values so announce packets are correct
func (serv *Service) listen() (err error) {
	config := &tls.Config{
		Certificates: []tls.Certificate{serv.cert},
	}

	Info.Printf("starting service on %s", serv.serviceSpec)
	serv.listener, err = tls.Listen("tcp", serv.serviceSpec, config)
	if err != nil {
		return err
	}
	addr := serv.listener.Addr()
	Info.Printf("service now listening to %s", addr.String())

	// TODO: get listenerIP to return 127.0.0.1 or something other than '::'/nil
	// serv.listenerIP = serv.listener.Addr().(*net.TCPAddr).IP
	serv.listenerIP, err = getIPForAnnouncePacket()
	// Trace.Printf("serv.listenerIP: `%s`", serv.listenerIP)

	if err != nil {
		return
	}

	serv.listenerPort = serv.listener.Addr().(*net.TCPAddr).Port

	return
}

// Register registers a service handler callback
func (serv *Service) Register(name string, callback ServiceActionFunc) (err error) {
	if serv.isRunning {
		err = errors.New("cannot register handlers while server is running")
		return
	}

	serv.actions[name] = &ServiceAction{
		callback: callback,
		version:  1,
	}
	return
}

// statsdLoop broadcasts the services queue depth to the metrics server based on the default statsd
// broadcast interval
func (serv *Service) statsdLoop() {
	for {
		select {
		case <-serv.statsStopSig:
			return
		default:
			err := serv.sendQueueDepth()
			if err != nil {
				Error.Println("could not send queue depth: ", err)
			}
		}
		time.Sleep(time.Duration(statsdInterval))
	}
}

//Run starts a scamp service
func (serv *Service) Run() {
	err := serv.createKubeLivenessFile()
	if err != nil {
		fmt.Println(err)
	}

forLoop:
	for {
		netConn, err := serv.listener.Accept()
		if err != nil {
			// Info.Printf("exiting service Run(): `%s`", err)
			break forLoop
		}
		// Trace.Printf("accepted new connection...")

		//var tlsConn (*tls.Conn) = (netConn).(*tls.Conn)
		tlsConn := (netConn).(*tls.Conn)
		if tlsConn == nil {
			Error.Fatalf("could not create tlsConn")
			break forLoop
		}

		conn := NewConnection(tlsConn, "service")
		client := NewClient(conn, "service")

		serv.clientsM.Lock()
		serv.clients = append(serv.clients, client)
		serv.clientsM.Unlock()

		go serv.Handle(client)

		atomic.AddUint64(&serv.connectionsAccepted, 1)
	}

	// Info.Printf("closing all registered objects")

	serv.clientsM.Lock()
	for _, client := range serv.clients {
		client.Close()
	}
	serv.clientsM.Unlock()

	serv.statsCloseChan <- true
}

// Handle handles incoming client messages received via the cient MessageChan
func (serv *Service) Handle(client *Client) {
	var action *ServiceAction
HandlerLoop:
	for {
		select {
		case msg, ok := <-client.Incoming():
			if !ok {
				break HandlerLoop
			}
			action = serv.actions[msg.Action]

			if action != nil {
				action.callback(msg, client)
			} else {
				Error.Printf("do not know how to handle action `%s`", msg.Action)

				reply := NewMessage()
				reply.SetMessageType(MessageTypeReply)
				reply.SetEnvelope(EnvelopeJSON)
				reply.SetRequestID(msg.RequestID)
				reply.Write([]byte(`{"error": "no such action"}`))
				_, err := client.Send(reply)
				if err != nil {
					client.Close()
					break HandlerLoop
				}
			}
		case <-time.After(msgTimeout):
			break HandlerLoop
		}
	}

	client.Close()
	serv.RemoveClient(client)
}

// RemoveClient removes a client from the scamp service
func (serv *Service) RemoveClient(client *Client) (err error) {
	serv.clientsM.Lock()
	defer serv.clientsM.Unlock()

	index := -1
	for i, entry := range serv.clients {
		if client == entry {
			index = i
			break
		}
	}

	if index == -1 {
		Error.Printf("tried removing client that wasn't being tracked")
		return fmt.Errorf("unknown client") // TODO can I get the client's IP?
	}

	client.Close()
	serv.clients = append(serv.clients[:index], serv.clients[index+1:]...)

	return nil
}

// Stop closes the service's net.Listener
func (serv *Service) Stop() {
	fmt.Println("shutting down")
	serv.statsStopSig <- true
	if serv.listener != nil {
		serv.listener.Close()
	}
	err := serv.removeKubeLivenessFile()
	if err != nil {
		fmt.Println("could not remove liveness file: ", err)
	}
	fmt.Println("shutdown done")
}

// MarshalText serializes a scamp service
func (serv *Service) MarshalText() (b []byte, err error) {
	var buf bytes.Buffer

	serviceProxy := serviceAsServiceProxy(serv)

	classRecord, err := serviceProxy.MarshalJSON() //json.Marshal(&serviceProxy) //Marshal is mangling service actions
	if err != nil {
		return
	}
	sig, err := signSHA256(classRecord, serv.cert.PrivateKey.(*rsa.PrivateKey))
	if err != nil {
		return
	}
	sigParts := stringToRows(sig, 76)

	buf.Write(classRecord)
	buf.WriteString("\n\n")
	buf.Write(serv.pemCert)
	buf.WriteString("\n\n")
	// buf.WriteString(sig)
	// buf.WriteString("\n\n")
	for _, part := range sigParts {
		buf.WriteString(part)
		buf.WriteString("\n")
	}
	buf.WriteString("\n")

	b = buf.Bytes()
	return
}

func stringToRows(input string, rowlen int) (output []string) {
	output = make([]string, 0)

	if len(input) <= 76 {
		output = append(output, input)
	} else {
		substr := input[:]
		var row string
		done := false
		for {
			if len(substr) > 76 {
				row = substr[0:76]
				substr = substr[76:]
			} else {
				row = substr[:]
				done = true
			}
			output = append(output, row)
			if done {
				break
			}
		}
	}

	return
}

func (serv *Service) generateRandomName() {
	randBytes := make([]byte, 18, 18)
	read, err := rand.Read(randBytes)
	if err != nil {
		err = fmt.Errorf("could not generate all rand bytes needed. only read %d of 18", read)
		return
	}
	base64RandBytes := base64.StdEncoding.EncodeToString(randBytes)

	var buffer bytes.Buffer
	buffer.WriteString(serv.humanName)
	buffer.WriteString(":")
	buffer.WriteString(base64RandBytes[0:])
	serv.name = string(buffer.Bytes())
}

// Resource represents a "geary" resource listed in service_deps.yml
type Resource struct {
	Type         string
	Sector       string
	Name         string
	Version      string
	Dependencies []*Resource
}

// TODO: we should dicuss movng the path to the liveness file to a config file (like soa.conf) or having it declared
// when creating the service
func (serv *Service) createKubeLivenessFile() error {

	if _, err := os.Stat(livenessDirPath); os.IsNotExist(err) {
		err = os.MkdirAll(livenessDirPath, 0755)
		if err != nil {
			return err
		}
	}

	file, err := os.Create(livenessDirPath + serv.humanName)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

func (serv *Service) removeKubeLivenessFile() error {
	path := livenessDirPath + serv.humanName
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

// sendQueueDepth sends the current state of the message queue to the statsd address configured in
// soa.conf (service.statsd_peer_address and service.statsd_peer_port). If this is not configured in
// soa.conf noop and do not send the packets
// statsd packet: "queue_depth.name.sector.ident.address:depth" (depth is int)
func (serv *Service) sendQueueDepth() error {
	// if statsdPeerDest is nil, just log and noop because in dev there will be no statsd peer and
	// in production we don't want the service to die because the address was probably
	// missing from soa.conf
	statsdPeerDest := &net.UDPAddr{
		IP:   defaultConfig.StatsdPeerAddress(),
		Port: defaultConfig.StatsdPeerPort(),
	}

	if statsdPeerDest == nil {
		Warning.Println("noop on sendQueueDepth because statsdPeerDest is nil")
		return nil
	}

	sp := serviceAsServiceProxy(serv)
	depth := serv.getQueueDepth()
	packet := fmt.Sprintf(
		"queue_depth.%s.%s.%s.%s:%v",
		sp.baseIdent(),
		sp.sector,
		sp.ident,
		sp.connspec,
		depth,
	)
	statsdPeerAddr := fmt.Sprintf(
		"%s:%v",
		statsdPeerDest.IP,
		statsdPeerDest.Port,
	)
	conn, err := net.Dial("udp", statsdPeerAddr)
	if err != nil {
		return fmt.Errorf("couldn't connect to statsd peer (%s):%s", statsdPeerAddr, err)
	}
	defer conn.Close()
	_, err = conn.Write([]byte(packet))
	if err != nil {
		return fmt.Errorf("couldn't write to statsd peer (%s):%s", statsdPeerAddr, err)
	}

	return nil
}

func (serv *Service) getQueueDepth() int {
	depth := 0
	serv.clientsM.Lock()
	defer serv.clientsM.Unlock()
	for _, c := range serv.clients {
		c.openRepliesLock.Lock()
		depth += len(c.openReplies)
		c.openRepliesLock.Unlock()
	}
	return depth
}

// loadDependencies reads the service_deps.yml file amd
// noops if it is missing
func (serv *Service) loadDependencies(path string) (r []*Resource) {
	deps, err := ioutil.ReadFile(path)
	if err != nil {
		Error.Printf("couldn't read service_deps.yml: %s\n", err)
	}
	r, err = UnmarshalResources(deps)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return r
}

func UnmarshalResources(in []byte) (r []*Resource, err error) {
	var m map[string][]string
	if err = yaml.Unmarshal(in, &m); err == nil {
		d := make([]*Resource, 0)
		for k, v := range m {
			r, err := parseResource(k)
			if err != nil {
				Error.Printf("parseResource err: %s\n", err)
				continue
			}
			fmt.Printf("string array size is %v\n", len(v))
			deps := make([]*Resource, 0)
			for _, x := range v {
				d, err := parseResource(x)
				if err != nil {
					Error.Printf("parseResource err: %s\n", err)
					continue
				}
				deps = append(deps, d)
			}
			fmt.Printf("deps: %v\n", deps)
			r.Dependencies = deps
			d = append(d, r)
		}
	}
	return
}

func parseResource(s string) (*Resource, error) {
	// TODO: split into sections <resource.type>:<resource.sector>:<resource.name(soa action)>.<resource.version>
	parts := strings.Split(s, ":")
	if len(parts) < 3 {
		return nil, fmt.Errorf("not enough parts in string for resource")
	}
	r := &Resource{
		Type:    parts[0],
		Sector:  parts[1],
		Name:    parts[2],
		Version: "1", //TODO: parse version
	}
	return r, nil
}
