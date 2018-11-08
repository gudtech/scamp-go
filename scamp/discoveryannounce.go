package scamp

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv4"
	_ "gopkg.in/alexcesaro/statsd.v2"
)

// DiscoveryAnnouncer ... TODO: godoc
type DiscoveryAnnouncer struct {
	services      []*Service
	multicastConn *ipv4.PacketConn
	multicastDest *net.UDPAddr
	stopSig       (chan bool)
}

// NewDiscoveryAnnouncer creates a DiscoveryAnnouncer
func NewDiscoveryAnnouncer() (announcer *DiscoveryAnnouncer, err error) {
	announcer = new(DiscoveryAnnouncer)
	announcer.services = make([]*Service, 0, 0)
	announcer.stopSig = make(chan bool)

	config := DefaultConfig()
	announcer.multicastDest = &net.UDPAddr{IP: config.DiscoveryMulticastIP(), Port: config.DiscoveryMulticastPort()}
	announcer.multicastConn, err = localMulticastPacketConn(config)
	if err != nil {
		return
	}

	return
}

// Stop notifies stopSig channel to stop announcer
func (announcer *DiscoveryAnnouncer) Stop() {
	announcer.stopSig <- true
}

// Track indicates that announcer should track and announce service
func (announcer *DiscoveryAnnouncer) Track(s *Service) {
	announcer.services = append(announcer.services, s)
}

// AnnounceLoop runs service announceloop and runs announcer.doAnnounce() at time
// interval configured in defaultAnnounceInterval
// TODO: make defaultAnnounceInterval configurable in the service (rather than hardcoded in scamp)
func (announcer *DiscoveryAnnouncer) AnnounceLoop() {
	// Trace.Printf("starting announcer loop")

	for {
		select {
		case <-announcer.stopSig:
			return
		default:
			err := announcer.doAnnounce()
			if err != nil {
				Error.Println(err)
			}
		}

		time.Sleep(time.Duration(defaultAnnounceInterval) * time.Second)
	}
}

func (announcer *DiscoveryAnnouncer) doAnnounce() (err error) {
	for _, s := range announcer.services {
		serviceDesc, err := s.MarshalText()
		if err != nil {
			return fmt.Errorf("failed to marshal service as text: %s", err)
		}

		_, err = announcer.multicastConn.WriteTo(serviceDesc, nil, announcer.multicastDest)
		if err != nil {
			return fmt.Errorf("failed write to multicast connection: %s", err)
		}
	}
	return
}
