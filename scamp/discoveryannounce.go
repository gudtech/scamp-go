package scamp

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

// DiscoveryAnnouncer ... TODO: godoc
type DiscoveryAnnouncer struct {
	services      []*Service
	multicastConn *ipv4.PacketConn
	multicastDest *net.UDPAddr

	// statsdPeerConn *ipv4.PacketConn // TODO: check if this is correct
	statsdPeerDest *net.UDPAddr // TODO: check if this is correct
	stopSig        (chan bool)
}

// NewDiscoveryAnnouncer creates a DiscoveryAnnouncer
func NewDiscoveryAnnouncer() (announcer *DiscoveryAnnouncer, err error) {
	announcer = new(DiscoveryAnnouncer)
	announcer.services = make([]*Service, 0, 0)
	announcer.stopSig = make(chan bool)

	//TODO: add multicast connection for statsd service (read from soa.conf)
	config := DefaultConfig()
	announcer.multicastDest = &net.UDPAddr{IP: config.DiscoveryMulticastIP(), Port: config.DiscoveryMulticastPort()}
	announcer.multicastConn, err = localMulticastPacketConn(config)
	if err != nil {
		return
	}

	announcer.statsdPeerDest = &net.UDPAddr{IP: config.StatsdPeerAddress(), Port: config.StatsdPeerPort()}

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
// TODO: noop on doAnnounce() if announce address not configured
func (announcer *DiscoveryAnnouncer) AnnounceLoop() {
	for {
		select {
		case <-announcer.stopSig:
			return
		default:
			err := announcer.doAnnounce()
			if err != nil {
				Error.Println(err)
			}
			// TODO: noop on statsd announce (sendQueueDepth()) if statsd address not configured
			err = announcer.sendQueueDepth()
			if err != nil {
				Error.Println("could not send queue depth: ", err)
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

// sendQueueDepth sends the current state of the message queue to the statsd address configured in
// soa.conf (service.statsd_peer_address and service.statsd_peer_port). If this is not configured in
// soa.conf noop and do not send the packets
// statsd packet: "queue_depth.name.sector.ident.address:depth" (depth is int)
func (announcer *DiscoveryAnnouncer) sendQueueDepth() error {
	// no error, just log and noop because in dev there will be no statsd peer and
	// in production we dont; want the service to die because the address was probably
	// missing from soa.conf
	if announcer.statsdPeerDest == nil {
		Warning.Println("noop on sendQueueDepth because statsdPeerDest is nil")
		return nil
	}
	for _, s := range announcer.services {
		sp := serviceAsServiceProxy(s)
		depth := s.getQueueDepth()
		packet := fmt.Sprintf(
			"queue_depth.%s.%s.%s.%s:%v",
			sp.baseIdent(),
			sp.sector,
			sp.ident,
			sp.connspec,
			depth,
		)
		statsdPeerAddr := fmt.Sprintf(
			"%s:%v",
			announcer.statsdPeerDest.IP,
			announcer.statsdPeerDest.Port,
		)
		conn, err := net.Dial("udp", statsdPeerAddr)
		if err != nil {
			return fmt.Errorf("couldn't connect to statsd peer (%s):%s", statsdPeerAddr, err)
		}
		defer conn.Close()
		_, err = conn.Write([]byte(packet))
		if err != nil {
			return fmt.Errorf("couldn't write to statsd peer (%s):%s", statsdPeerAddr, err)
		}
	}
	return nil
}
