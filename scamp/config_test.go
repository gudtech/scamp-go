package scamp

import "testing"
import "bytes"
import "bufio"

var sampleConfigFile = []byte(`
discovery.cache_path = /tmp/discovery.cache
bus.authorized_services = /etc/SCAMP/authorized_services
helloworld.soa_key = /etc/GT_private/services/helloworld.key
helloworld.soa_cert = /etc/GT_private/services/helloworld.crt
scamp.first_port = 30100
scamp.last_port = 30100
bus.address = 127.0.0.1
`)

// TODO: use table tests
func TestConfigHelpers(t *testing.T) {
	reader := bytes.NewReader(sampleConfigFile)
	scanner := bufio.NewScanner(reader)

	conf := NewConfig()
	conf.doLoad(scanner)

	expected := []byte("/etc/GT_private/services/helloworld.key")
	if !bytes.Equal(conf.ServiceKeyPath("helloworld", "/etc/GT_private/services"), expected) {
		t.Fatalf("expected %s, got %s", expected, conf.ServiceKeyPath("helloworld", "/etc/GT_private/services"))
	}

	expected = []byte("/etc/GT_private/services/helloworld.crt")
	if !bytes.Equal(conf.ServiceCertPath("helloworld", "/etc/GT_private/services"), expected) {
		t.Fatalf("expected %s, got %s", expected, conf.ServiceCertPath("helloworld", "/etc/GT_private/services"))
	}

	expected = []byte("/etc/GT_private/services/helloworld.crt")
	if !bytes.Equal(conf.ServiceCertPath("helloworld", ""), expected) {
		t.Fatalf("expected %s, got %s", expected, conf.ServiceCertPath("helloworld", ""))
	}

	expected = []byte("/etc/GT_private/services/helloworld.key")
	if !bytes.Equal(conf.ServiceKeyPath("helloworld", ""), expected) {
		t.Fatalf("expected %s, got %s", expected, conf.ServiceKeyPath("helloworld", ""))
	}

	expected = []byte("/etc/SCAMP/services/helloworld.crt")
	if !bytes.Equal(conf.ServiceCertPath("helloworld", "/etc/SCAMP/services"), expected) {
		t.Fatalf("expected %s, got %s", expected, conf.ServiceCertPath("helloworld", ""))
	}

	expected = []byte("/etc/SCAMP/services/helloworld.key")
	if !bytes.Equal(conf.ServiceKeyPath("helloworld", "/etc/SCAMP/services"), expected) {
		t.Fatalf("expected %s, got %s", expected, conf.ServiceKeyPath("helloworld", "/etc/SCAMP/services"))
	}
}
