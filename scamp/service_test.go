package scamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"
)

func TestServiceHandlesRequest(t *testing.T) {
	hasStopped := make(chan bool)
	s := spawnTestService(t, hasStopped)
	spec := fmt.Sprintf("%s:%v", s.listenerIP, s.listenerPort)
	connectToTestService(t, spec)
	// time.Sleep(1000 * time.Millisecond)
	s.Stop()
	<-hasStopped
}

func spawnTestService(t *testing.T, hasStopped chan bool) (service *Service) {
	desc := ServiceDesc{
		Sector:      "test",
		ServiceSpec: "0.0.0.0:0",
		HumanName:   "sample",
	}
	opts := &Options{
		SOAConfigPath: "./../../scamp-go/fixtures/soa.conf",
		KeyPath:       "./../../scamp-go/fixtures",
		CertPath:      "./../../scamp-go/fixtures",
	}
	service, err := NewService(desc, opts)
	if err != nil {
		t.Fatalf("error creating new service: `%s`", err)
	}

	type helloResponse struct {
		Test string `json:"test"`
	}

	service.Register("helloworld.hello", func(message *Message, client *Client) {
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
		service.Run() //GR 28
		hasStopped <- true
	}()
	return
}

func connectToTestService(t *testing.T, spec string) {
	Info.Printf("Dialing %s", spec)
	client, err := Dial(spec)
	if err != nil {
		t.Fatalf("could not connect! `%s`\n", err)
	}
	defer client.Close()

	msg := &Message{
		RequestID:   1,
		Action:      "helloworld.hello",
		Envelope:    EnvelopeJSON,
		Version:     1,
		MessageType: MessageTypeRequest,
	}
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
		ServiceSpec: "0.0.0.0:30100",
		HumanName:   "sample",
		Sector:      "main",
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

	re := regexp.MustCompile(`(?m)((\[3,"sample-)|(,"main",1,2500,"beepish\+tls://)|(:30100",\["json"\],\[\["Logging",\["info","",1\]\]\],.*\]))`)
	expected := `[3,"sample-XXXXX","main",1,2500,"beepish+tls://174.10.10.10:30100",["json"],[["Logging",["info","",1]]],10.000000]`
	matches := re.FindAllString(string(b), -1) //{

	if len(matches) < 3 {
		t.Fatalf("\nexpected: \t`%s`,\n\tgot:\t`%s`\n", expected, b)
	}
}

func TestFullServiceMarshal(t *testing.T) {
	desc := ServiceDesc{
		ServiceSpec: "0.0.0.0:0",
		HumanName:   "sample",
		Sector:      "main",
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
	re := regexp.MustCompile(`(?m)((\[3,"sample-)|(,"main",1,2500,"beepish\+tls://)|(",\["json"\],\[\["Logging",\["info","",1\]\]\],.*\])|(-----BEGIN CERTIFICATE-----)|(-----END CERTIFICATE-----))`)
	matches := re.FindAllString(string(b), -1)
	expected := `[3,"sample-38xM9dgjDlRFEJM6g65Xpbvq","main",1,2500,"beepish+tls://192.168.1.22:63408",["json"],[["Logging",["info","",1]]],1541018464.480262]`
	if len(matches) < 5 {
		t.Fatalf("\nexpected: \t`%s`,\n\tgot:\t`%s`\n", expected, b)
	}
}
