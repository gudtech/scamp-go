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
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Two minute timeout on clients
var msgTimeout = time.Second * 120

// ServiceActionFunc represents a service callback
type ServiceActionFunc interface {
	Call(*Message, *Client)
}

type BasicActionFunc func(*Message, *Client)

func (function BasicActionFunc) Call(message *Message, client *Client) {
	function(message, client)
}

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
	cert     tls.Certificate
	pemCert  []byte // just a copy of what was read off disk at tls cert load time

	// stats
	statsCloseChan      chan bool
	connectionsAccepted uint64
}

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

// NewServiceExplicitCert initializes and returns pointer to a new scamp service,
// with an explicitly specified certificate rather than an implicitly discovered one.
// keypair is a TLS certificate, and pemCert is the raw bytes of an X509 certificate.
func NewServiceExplicitCert(sector string, serviceSpec string, humanName string, keypair tls.Certificate, pemCert []byte) (serv *Service, err error) {
	if len(humanName) > 18 {
		err = fmt.Errorf("name `%s` is too long, must be less than 18 bytes", humanName)
		return
	}

	serv = new(Service)
	serv.sector = sector
	serv.serviceSpec = serviceSpec
	serv.humanName = humanName
	serv.generateRandomName()
	serv.actions = make(map[string]*ServiceAction)
	serv.cert = keypair
	serv.pemCert = bytes.TrimSpace(pemCert)

	err = serv.listen()
	if err != nil {
		return
	}

	serv.statsCloseChan = make(chan bool)
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

	if err != nil {
		return
	}

	serv.listenerPort = serv.listener.Addr().(*net.TCPAddr).Port

	return
}

// Register registers a service handler callback
func (serv *Service) Register(name string, callback func(*Message, *Client), options *ActionOptions) (err error) {
	if serv.isRunning {
		err = errors.New("cannot register handlers while server is running")
		return
	}

	actionOptions := DefaultActionOptions()
	if options != nil {
		actionOptions = *options
	}

	serv.actions[name] = &ServiceAction{
		callback: ServiceOptionsFunc{
			callback: BasicActionFunc(callback),
			options:  actionOptions,
		},
		version: 1,
	}
	return
}

// Run starts a scamp service
func (serv *Service) Run() {
	err := serv.createRunningServiceFile()
	if err != nil {
		fmt.Println(err)
	}

forLoop:
	for {
		netConn, err := serv.listener.Accept()
		if err != nil {
			break forLoop
		}

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

	serv.clientsM.Lock()
	for _, client := range serv.clients {
		client.Close()
	}
	serv.clientsM.Unlock()

	serv.statsCloseChan <- true

	err = serv.removeRunningServiceFile()
	if err != nil {
		fmt.Println("could not remove liveness file: ", err)
	}
	fmt.Println("shutdown done")
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

			Info.Printf(
				"`%s` (request %d, client %d): Action requested",
				msg.Action,
				msg.RequestID,
				msg.ClientID,
			)

			action = serv.actions[msg.Action]

			if action != nil {
				action.callback.Call(msg, client)
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
	if serv.listener != nil {
		serv.listener.Close()
	}
	fmt.Println("shutting down")
}

// MarshalText serializes a scamp service
func (serv *Service) MarshalText() (b []byte, err error) {
	var buf bytes.Buffer

	serviceProxy := serviceAsServiceProxy(serv)

	classRecord, err := serviceProxy.MarshalJSON() // json.Marshal(&serviceProxy) //Marshal is mangling service actions
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

func (serv *Service) createRunningServiceFile() error {
	runningServicesDirPath, configErr := DefaultConfig().RunningServiceFileDirPath()
	if configErr != nil {
		return configErr
	}

	if _, statErr := os.Stat(string(runningServicesDirPath)); os.IsNotExist(statErr) {
		mkdirErr := os.MkdirAll(string(runningServicesDirPath), 0755)
		if mkdirErr != nil {
			return mkdirErr
		}
	}

	runningServiceFilePath := serv.runningServiceFilePath(runningServicesDirPath)

	fmt.Printf("Creating running service file: `%s`\n", runningServiceFilePath)
	file, createErr := os.Create(serv.runningServiceFilePath(runningServicesDirPath))
	if createErr != nil {
		return createErr
	}
	defer file.Close()
	return nil
}

func (serv *Service) removeRunningServiceFile() error {
	runningServicesDirPath, configErr := DefaultConfig().RunningServiceFileDirPath()
	if configErr != nil {
		return configErr
	}

	runningServiceFilePath := serv.runningServiceFilePath(runningServicesDirPath)

	fmt.Printf("Deleting running service file: `%s`\n", runningServiceFilePath)
	removeErr := os.Remove(runningServiceFilePath)
	if removeErr != nil {
		return removeErr
	}
	return nil
}

func (serv *Service) runningServiceFilePath(runningServicesDirPath []byte) string {
	name := strings.Replace(serv.name, "/", "_", -1)

	return string(runningServicesDirPath) + "/" + name
}
