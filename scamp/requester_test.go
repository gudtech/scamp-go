package scamp

import (
	"bytes"
	"testing"
)

// TODO: to get this to work we need to write a dummy discoverycache file
// or directly manipulate the cache in the test
func spawnRequesterTestService(t *testing.T) (service *Service) {
	desc := ServiceDesc{
		Sector:      "test",
		ServiceSpec: "0.0.0.0:30100",
		HumanName:   "logger",
		name:        "logger-b3/QF6hT+7tEJVVoVkvmxl8n",
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
	defaultCache.cacheM.Lock()
	defaultCache.verifyRecords = false
	defaultCache.cacheM.Unlock()

	type helloResponse struct {
		Test string `json:"test"`
	}

	service.Register("Logger.info", func(message *Message, client *Client) {
		Info.Println("Handling client request")
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
	return
}

// TODO: in order to test MakeJSONRequest we will need to make the defaultCache an
// interface so that it is mockable
func TestRequester(t *testing.T) {
	s := spawnRequesterTestService(t)
	msg := NewRequestMessage()
	msg.SetEnvelope(EnvelopeJSON)
	respMsg, err := MakeJSONRequest("main", "Logger.info", 1, msg)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if respMsg == nil || len(respMsg.Bytes()) == 0 {
		t.Fatalf("response message was nil")
	}
	expected := []byte(`{"test":"success"}`)
	resp := bytes.TrimRight(respMsg.Bytes(), "\n")
	if !bytes.Equal(resp, expected) {
		Error.Printf("resp: %s", string(resp))
		t.Fatalf("\nExpected:\t%q\nReceived:\t%q", expected, resp)
	}
	s.Stop()
}

// func TestMain(m *testing.M) {
// 	flag.Parse()
// 	Initialize("/etc/SCAMP/soa.conf")
// 	os.Exit(m.Run())
// }

// fmt.Printf("%#v\n", defaultCache)
// for _, sp := range defaultCache.actionIndex {
// 	Info.Println("num of sp: ", len(sp))
// 	for _, p := range sp {
// 		Info.Println("p.sector: ", p.sector)
// 		Info.Println("p.connspec: ", p.connspec)
// 		Info.Println("p.version: ", p.version)
// 		Info.Println("p.ident: ", p.ident)
// 		Info.Println("p.classes: ", p.classes)
// 		Info.Println("p.client: ", p.client)
// 	}
// }
