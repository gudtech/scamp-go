package scamp

import "testing"

func TestTrack(t *testing.T) {
	Initialize("./../fixtures/sample_soa.conf")

	serv := &Service{}
	a, err := NewDiscoveryAnnouncer()
	if err != nil {
		t.Fatalf("could not create announcer: %s", err)
	}

	a.Track(serv)

	if len(a.services) != 1 {
		t.Fatalf("expected 1 service in s.services, found %v", len(a.services))
	}
}
