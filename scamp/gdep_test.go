package scamp

import (
	"encoding/json"
	"testing"
)

func TestReadDeps(t *testing.T) {
	expected := []byte(`{"offers":[{"name":"main:action1~1"}],"requires":[{"action":"main:action1~1","deps":["main:Logger.info~1#json"]}]}`)

	deps, err := readDeps("./../../scamp-go/fixtures/gdep.json")
	if err != nil {
		t.Fatalf("falied to open dep file: %s", err)
	}

	if deps == nil {
		t.Fatalf("nil dep struct returned: %s", err)
	}

	if len(deps.Requires) != 1 {
		t.Fatalf("expected 2 required soa actions, received %v", len(deps.Requires))
	}
	if len(deps.Offers) != 1 {
		t.Fatalf("expected 2 offered soa actions, received %v", len(deps.Offers))
	}

	out, err := json.Marshal(deps)
	if err != nil {
		t.Fatalf("cannot marshal deps to JSON: %s", err)
	}

	if string(out) != string(expected) {
		t.Fatalf("expected: \n%s\n received \n%s\n", string(expected), string(out))
	}

}

func TestCheckDependencies(t *testing.T) {
	depPath := "./../../scamp-go/fixtures/gdep.json"
	// overwrite the default "-gdep" flag path for the tests
	depfilePath = &depPath
	err := Initialize("./../../scamp-go/fixtures/sample_soa.conf")
	if err != nil {
		t.Fatalf("%s", err)
	}

	serv, err := NewService("main", "", "sample")
	if err != nil {
		t.Fatalf("falied to create service: %s", err)
	}

	err = serv.CheckDependencies()
	if err != nil {
		t.Fatalf("%s", err)
	}
}

func TestRunningService(t *testing.T) {
	depPath := "./../../scamp-go/fixtures/gdep.json"
	depfilePath = &depPath
	err := Initialize("./../../scamp-go/fixtures/sample_soa.conf")
	if err != nil {
		t.Fatalf("%s", err)
	}

	_, err = NewService("main", "", "sample")
	if err != nil {
		t.Fatalf("falied to create service: %s", err)
	}
}
