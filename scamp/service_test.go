package scamp

import (
	"bytes"
	"encoding/json"
	"regexp"
	"sync"
	"testing"
	"time"
)

func TestServiceHandlesRequest(t *testing.T) {
	s := spawnTestService(t)
	// spec := fmt.Sprintf("%s:%v", s.listenerIP, s.listenerPort)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		connectToTestService(t, s.listener.Addr().String())
		wg.Done()
	}()
	wg.Wait()
	s.Stop()
}

func spawnTestService(t *testing.T) (service *Service) {
	desc := ServiceDesc{
		Sector:      "test",
		ServiceSpec: "0.0.0.0:50100",
		HumanName:   "logger",
	}
	opts := &Options{
		SOAConfigPath:    "./../../scamp-go/fixtures/soa.conf",
		KeyPath:          "./../../scamp-go/fixtures",
		CertPath:         "./../../scamp-go/fixtures",
		LivenessFilePath: "./../../scamp-go/fixtures",
	}
	service, err := NewService(desc, opts)
	if err != nil {
		t.Fatalf("error creating new service: `%s`", err)
	}

	// service.desc.ServiceSpec = fmt.Sprintf("%s:%v", service.listenerIP, service.listenerPort)
	Info.Println("SPEC: ", service.desc.ServiceSpec)
	type helloResponse struct {
		Test string `json:"test"`
	}

	service.Register("helloworld.hello", func(message *Message, client *Client) {
		Info.Println("received helloworld.hello request")
		respMsg := NewMessage()
		if respMsg == nil {
			t.Fatal("newMessage was nil")
		}
		respMsg.RequestID = 2
		respMsg.Envelope = EnvelopeJSON
		respMsg.Version = 1
		respMsg.MessageType = MessageTypeReply
		body := helloResponse{
			Test: "success",
		}

		respMsg.WriteJSON(body)
		_, err := client.Send(respMsg)
		if err != nil {
			t.Fatalf("response send failed: %s", err)
		}
	})

	go func() {
		service.Run()
	}()
	if defaultAnnouncer == nil {
		t.Fatal("announcer is nil")
	}
	return
}

func connectToTestService(t *testing.T, spec string) {
	Info.Printf("Dialing %s", spec)
	client, err := Dial(spec)
	if err != nil {
		t.Fatalf("could not connect! `%s`\n", err)
	}
	if client == nil {
		t.Fatalf("Dial returned nil client")
	}
	msg := &Message{
		RequestID:   1,
		Action:      "helloworld.hello",
		Envelope:    EnvelopeJSON,
		Version:     1,
		MessageType: MessageTypeRequest,
	}
	Info.Println("sending message")
	responseChan, err := client.Send(msg)
	if err != nil {
		t.Fatalf("error initiating session: `%s`", err)
	}
	expected := []byte(`{"test":"success"}`)
	select {
	case msg := <-responseChan:
		// JSON resp includes a newline character that we don't want
		resp := bytes.TrimRight(msg.Bytes(), "\n")
		if !bytes.Equal(resp, expected) {
			Error.Printf("resp: %s", string(resp))
			t.Fatalf("\nExpected:\t%q\nReceived:\t%q", expected, resp)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for response")
	}

	return
}

// TODO: cutting some corners in this test, it tests two complicated things at once:
// 1. Copying `Service` properties to new `ServiceProxy`
// 2. Marshaling `ServiceProxy` to announce format
func TestServiceToProxyMarshal(t *testing.T) {
	desc := ServiceDesc{
		ServiceSpec: "0.0.0.0:30200",
		HumanName:   "logger",
		Sector:      "main",
		name:        "logger-b3/QF6hT+7tEJVVoVkvmxl8n",
	}

	opts := &Options{
		SOAConfigPath: "./../../scamp-go/fixtures/soa.conf",
		KeyPath:       "./../../scamp-go/fixtures",
		CertPath:      "./../../scamp-go/fixtures",
	}

	s, err := NewService(desc, opts)
	if err != nil {
		t.Fatalf("Could not create service: %s", err)
	}

	s.Register("Logging.info", func(_ *Message, _ *Client) {
	})

	serviceProxy := serviceAsServiceProxy(s)
	serviceProxy.timestamp = 10
	b, err := json.Marshal(&serviceProxy)
	if err != nil {
		t.Fatalf("could not serialize service proxy")
	}

	re := regexp.MustCompile(`(?m)((\[3,"logger-b3\/QF6hT\+7tEJVVoVkvmxl8n")|(,"main",1,2500,"beepish\+tls://)|(",\["json"\],\[\["Logging",\["info","",1\]\]\],.*\]))`)
	expected := `[3,"sample-XXXXX","main",1,2500,"beepish+tls://174.10.10.10:30100",["json"],[["Logging",["info","",1]]],10.000000]`
	matches := re.FindAllString(string(b), -1) //{

	if len(matches) < 3 {
		t.Fatalf("\nexpected: \t`%s`\n\tgot:\t`%s`\n", expected, b)
	}
}

func TestFullServiceMarshal(t *testing.T) {
	desc := ServiceDesc{
		ServiceSpec: "0.0.0.0:51055",
		HumanName:   "logger",
		Sector:      "main",
		name:        "logger-b3/QF6hT+7tEJVVoVkvmxl8n",
	}
	opts := &Options{
		SOAConfigPath: "./../../scamp-go/fixtures/soa.conf",
		KeyPath:       "./../../scamp-go/fixtures",
		CertPath:      "./../../scamp-go/fixtures",
	}
	s, err := NewService(desc, opts)
	if err != nil {
		t.Fatalf("could not create service: %s", err)
	}

	s.Register("Logging.info", func(_ *Message, _ *Client) {
	})

	b, err := s.MarshalText()
	if err != nil {
		t.Fatalf("unexpected error serializing service: `%s`", err)
	}
	re := regexp.MustCompile(`(?m)((\[3,"logger-b3\/QF6hT\+7tEJVVoVkvmxl8n")|(,"main",1,2500,"beepish\+tls://)|(",\["json"\],\[\["Logging",\["info","",1\]\]\],.*\])|(-----BEGIN CERTIFICATE-----)|(-----END CERTIFICATE-----))`)
	matches := re.FindAllString(string(b), -1)
	expected := `[3,"logger-b3/QF6hT+7tEJVVoVkvmxl8n","main",1,2500,"beepish+tls://192.168.1.22:63408",["json"],[["Logging",["info","",1]]],1541018464.480262]`
	if len(matches) < 5 {
		t.Fatalf("\nexpected: \t`%s`\n\tgot:\t`%s`\n", expected, b)
	}
}
