package scamp

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	// Get root of the project.
	_, base, _, _ = runtime.Caller(0)
	basePath      = filepath.Dir(base)
	fixturesPath  = basePath + "/../fixtures"
	pemPath       = fixturesPath + "/ticket_verify_public_key.pem"
	dispatchPath  = fixturesPath + "/processor-dispatch.token"
)

func TestTicket(t *testing.T) {
	t.Logf("basepath: %v", basePath)
	t.Logf("fixtures path: %v", fixturesPath)
	t.Logf("pem path: %v", pemPath)
	good, err := ioutil.ReadFile(dispatchPath)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if good == nil {
		t.Fatalf("nil ticket")
	}

	tkt, err := VerifyTicket(string(good), pemPath)
	if err != nil {
		t.Errorf("failed to verify correct ticket: %s", err)
	}
	t.Logf("ok %+v, %+v", tkt, err)

	tkt, err = VerifyTicket(string(good[:len(good)-1]), pemPath)
	if err == nil {
		t.Errorf("bad ticket accepted")
	}
	t.Logf("ok (should fail) %+v, %+v", tkt, err)

	tkt, err = VerifyTicket(string(good[1:]), pemPath)
	if err == nil {
		t.Errorf("bad ticket accepted")
	}
	t.Logf("ok (should fail) %+v, %+v", tkt, err)
}
