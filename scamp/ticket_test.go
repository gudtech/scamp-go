package scamp

import (
	"io/ioutil"
	"testing"
)

var pemPath = "./../fixtures/ticket_verify_public_key.pem"

func TestTicket(t *testing.T) {
	good, err := ioutil.ReadFile("./../fixtures/processor-dispatch.token")
	if err != nil {
		t.Fatalf(err.Error())
	}

	if good == nil {
		t.Fatalf("nil ticket")
	}

	tkt, err := verifyTicket(string(good), pemPath)
	if err != nil {
		t.Errorf("failed to verify correct ticket: %s", err)
	}
	t.Logf("ok %+v, %+v", tkt, err)

	tkt, err = verifyTicket(string(good[:len(good)-1]), pemPath)
	if err == nil {
		t.Errorf("bad ticket accepted")
	}
	t.Logf("ok (should fail) %+v, %+v", tkt, err)

	tkt, err = verifyTicket(string(good[1:]), pemPath)
	if err == nil {
		t.Errorf("bad ticket accepted")
	}
	t.Logf("ok (should fail) %+v, %+v", tkt, err)
}
