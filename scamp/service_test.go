package scamp

import "testing"
import "bytes"
import "encoding/json"
import "net"
import "crypto/tls"
import "os"

// TODO: fix Session API (aka, simplify design by dropping it)
func TestServiceHandlesRequest(t *testing.T) {
	// Initialize("./../fixtures/sample_soa.conf")
	// // TODO: servicekeypath and servicecertpath should allow for mocking
	// // sp that we can test services
	// hasStopped := make(chan bool)
	// service := spawnTestService(hasStopped)
	// // connectToTestService(t)
	// time.Sleep(1000 * time.Millisecond)
	// service.Stop()
	// <-hasStopped

}

// TODO: I'm cutting some corners in this test, it tests two complicated things at once:
// 1. Copying `Service` properties to new `ServiceProxy`
// 2. Marshaling `ServiceProxy` to announce format
func TestServiceToProxyMarshal(t *testing.T) {
	s := Service{
		serviceSpec:  "123",
		humanName:    "a-cool-name",
		name:         "a-cool-name-1234",
		listenerIP:   net.ParseIP("174.10.10.10"),
		listenerPort: 30100,
		sector:       "main",
		actions:      make(map[string]*ServiceAction),
	}
	_ = s.Register("Logging.info", func(_ *Message, _ *Client) {}, nil)

	serviceProxy := serviceAsServiceProxy(&s)
	serviceProxy.timestamp = 10
	b, err := json.Marshal(&serviceProxy)
	if err != nil {
		t.Fatalf("could not serialize service proxy")
	}
	expected := []byte(`[3,"a-cool-name-1234","main",1,2500,"beepish+tls://174.10.10.10:30100",["json"],[["Logging",["info","",1]]],10.000000]`)
	if !bytes.Equal(b, expected) {
		t.Fatalf("expected: `%s`,\n\tgot:\t`%s`\n", expected, b)
	}

}

func TestFullServiceMarshal(t *testing.T) {
	// TODO big assumption that you environment is set up like mine:
	//   root repo `scamp-go` has a sibling folder called `scamp-go-workspace` where `scamp-go`
	//   is symlinked in as such: ../scamp-go-workspace/src/github.com/gudtech/scamp-go
	// it's crazy, I know. thanks GOPATH.
	cert, err := tls.LoadX509KeyPair("./../../scamp-go/fixtures/sample.crt", "./../../scamp-go/fixtures/sample.key")
	if err != nil {
		t.Fatalf("could not load fixture keypair: `%s`", err)
	}

	encodedCert, err := os.ReadFile("./../fixtures/sample.crt")
	if err != nil {
		t.Fatalf("could not load fixture certificate")
	}
	encodedCert = bytes.TrimSpace(encodedCert)

	s := Service{
		serviceSpec:  "123",
		humanName:    "a-cool-name",
		name:         "a-cool-name-1234",
		sector:       "main",
		listenerIP:   net.ParseIP("174.10.10.10"),
		listenerPort: 30100,
		actions:      make(map[string]*ServiceAction),
		pemCert:      encodedCert,
		cert:         cert,
	}
	_ = s.Register("Logging.info", func(_ *Message, _ *Client) {}, nil)

	// TODO: confirm output of marshalling the payload.
	_, err = s.MarshalText()
	if err != nil {
		t.Fatalf("unexpected error serializing service: `%s`", err)
	}
	// t.Fatalf("b: `%s`", b)

}
