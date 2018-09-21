package scamp

import "testing"
import "bytes"
import "bufio"

func TestAuthorizedServiceSpec(t *testing.T) {
	err := Initialize("./../../scamp-go/fixtures/sample_soa.conf")
	if err != nil {
		t.Fatalf("%s", err)
	}
	cache := NewAuthorizedServicesCache()

	reader := bytes.NewReader(testAuthorizedServices)

	scanner := bufio.NewScanner(reader)

	err = cache.LoadAuthorizedServices(scanner)
	if err != nil {
		t.Errorf("err loading auth'd services: `%s`", err)
	}
}

func TestNewAuthorizedServicesSpec(t *testing.T) {
	spec, err := NewAuthorizedServicesSpec([]byte(`06:28:FF:2D:85:4D:27:7F:30:39:4D:D1:3C:5A:28:C3:22:2A:85:BD config, constant, feed, device, inventory, media, nav, notes, po, product, receive, user, customer, utility, fulfillment, index, api, web:ALL, reporting, vendor, secproxy, bgdispatcher`))
	if err != nil {
		t.Errorf("error parsing service spec: `%s`", err)
	}

	if !bytes.Equal(spec.Fingerprint, []byte("06:28:FF:2D:85:4D:27:7F:30:39:4D:D1:3C:5A:28:C3:22:2A:85:BD")) {
		t.Errorf("wrong fingerprint")
	}
}

func TestBadeAuthorizedServicesSpec(t *testing.T) {
	var err error

	_, err = NewAuthorizedServicesSpec([]byte(`# don't parse`))
	if err == nil {
		t.Errorf("should not have been parsed")
	}

	_, err = NewAuthorizedServicesSpec([]byte(``))
	if err == nil {
		t.Errorf("should not have been parsed")
	}
}

var testAuthorizedServices = []byte(`
# format: FINGERPRINT PREFIX, PREFIX, PREFIX
# FINGERPRINTs are produced by openssl x509 -fingerprint -sha1 -noout -in CERT
# a PREFIX may, but need not, contain dots; it must not be empty; it is matched without regard for case
06:28:FF:2D:85:4D:27:7F:30:39:4D:D1:3C:5A:28:C3:22:2A:85:BD config, constant, feed, device, inventory, media, nav, notes, po, product, receive, user, customer, utility, fulfillment, index, api, web:ALL, reporting, vendor, secproxy, bgdispatcher
F9:08:C3:66:74:C4:26:76:09:15:A5:0C:CC:25:FF:63:E6:FA:F2:AC auth, user, background:ALL,  compute:ALL, soapoffload:ALL, channelmodule:ALL
`)
