package scamp

import (
	"testing"
	"time"
)

func TestNewDiscoveryAnnouncer(t *testing.T) {
	c := NewConfig()
	if c == nil {
		t.Fatalf("could not create config")
	}
	defaultConfig = c

	a, err := NewDiscoveryAnnouncer()
	if err != nil {
		t.Fatalf("could not create discovery announcer: %s", err)
	}
	go a.run()
	time.Sleep(time.Millisecond * 250)
	a.mu.Lock()
	if a.isStopped {
		t.Fatalf("announcer has stopped")
	}
	a.mu.Unlock()
	a.stop()
	time.Sleep(time.Millisecond * 250)
	if !a.hasStopped() {
		return
	}
}
