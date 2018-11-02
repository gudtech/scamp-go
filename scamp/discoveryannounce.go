package scamp

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
)

// DiscoveryAnnouncer ... TODO: godoc
type DiscoveryAnnouncer struct {
	mu            sync.RWMutex
	services      []*Service
	multicastConn *ipv4.PacketConn
	multicastDest *net.UDPAddr
	stopSig       (chan bool)
	runOnce       sync.Once
	stopOnce      sync.Once
	isStopped     bool
}

// NewDiscoveryAnnouncer creates a DiscoveryAnnouncer
func NewDiscoveryAnnouncer() (a *DiscoveryAnnouncer, err error) {
	a = new(DiscoveryAnnouncer)
	a.services = make([]*Service, 0, 0)
	a.stopSig = make(chan bool)

	config := DefaultConfig()
	a.multicastDest = &net.UDPAddr{
		IP:   config.discoveryMulticastIP(),
		Port: config.discoveryMulticastPort(),
	}
	a.multicastConn, err = localMulticastPacketConn(config)
	if err != nil {
		return
	}

	return
}

// Stop notifies stopSig channel to stop announcer
func (a *DiscoveryAnnouncer) Stop() {
	a.stopOnce.Do(func() {
		a.stopSig <- true
		a.stopped()
	})
}

// Track indicates that announcer should track and announce the service
func (a *DiscoveryAnnouncer) Track(s *Service) {
	a.services = append(a.services, s)
}

func (a *DiscoveryAnnouncer) start() {
	var wg sync.WaitGroup
	a.runOnce.Do(func() {
		wg.Add(1)
		go func() {
			//TODO: should we use waitgroup here?
			a.running()
			a.announceLoop()
			Warning.Println("exited announceloop")
			wg.Done()
		}()
	})
	wg.Wait()
	Error.Println("exiting announcer.start()")
}

// AnnounceLoop runs service announceloop and runs announcer.doAnnounce() at time
// interval configured in defaultAnnounceInterval
// TODO: update to detect when announceloop fails and announcer isn;t actually announcing
func (a *DiscoveryAnnouncer) announceLoop() {
	defer a.stopped()
announceloop:
	for {
		select {
		case <-a.stopSig:
			break announceloop
		default:
			err := a.doAnnounce()
			if err != nil {
				Error.Printf("doAnnounce error: %s\n", err)
				a.stopped()
				break announceloop
			}
		}

		time.Sleep(time.Duration(defaultAnnounceInterval) * time.Second)
	}
	Warning.Println("exiting announceLoop")
}

// TODO: handle there being no services tracked by the announcer and provide a method to update the
// tracked service while the announceloop is running or to restart an announceloop once there are
// services to track. Possibly use agent pattern for announcer
func (a *DiscoveryAnnouncer) doAnnounce() (err error) {
	if len(a.services) == 0 {
		Warning.Println("No services are being tracked")
		//just noop for now. Should eventually return custom error type
		return nil
	}
	for _, s := range a.services {
		var serviceDesc []byte
		// TODO: store serviceDesc
		if s.remarshalForAnnounce {
			serviceDesc, err = s.MarshalText()
			if err != nil {
				return fmt.Errorf("Failed to marshal service as text: `%s`. skipping", err)
			}
			s.mu.Lock()
			s.announceBytes = serviceDesc
			s.remarshalForAnnounce = false
			s.mu.Unlock()
		}

		_, err = a.multicastConn.WriteTo(s.announceBytes, nil, a.multicastDest)
		if err != nil {
			return fmt.Errorf("failed writing to multicastConn: %s", err)
		}
	}

	return
}

// stopped sets isStopped to true
func (a *DiscoveryAnnouncer) stopped() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isStopped = true
}

// hasStopped returns a.isStopped
func (a *DiscoveryAnnouncer) hasStopped() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isStopped
}

func (a *DiscoveryAnnouncer) running() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isStopped = false
}
