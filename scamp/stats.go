package scamp

import (
	"encoding/json"
	"time"
)

type serviceStats struct {
	clientsAccepted uint64 `json:"total_clients_accepted"`
	openConnections uint64 `json:"open_connections"`
}

func gatherStats(service *Service) (stats serviceStats) {
	stats.clientsAccepted = service.connectionsAccepted
	stats.openConnections = uint64(len(service.clients))
	return
}

func printStatsLoop(s *Service, timeout time.Duration, closeChan chan bool) {
forLoop:
	for {
		select {
		case <-time.After(timeout):
			stats := gatherStats(s)
			statsBytes, err := json.Marshal(&stats)
			if err != nil {
				continue
			}

			Trace.Printf("periodic stats (%s): `%s`", s.desc.name, statsBytes)
		case <-closeChan:
			break forLoop
		}
	}
}
