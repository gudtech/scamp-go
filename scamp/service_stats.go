package scamp

import (
	"encoding/json"
	"time"
)

type serviceStats struct {
	ClientsAccepted uint64 `json:"total_clients_accepted"`
	OpenConnections uint64 `json:"open_connections"`
}

func gatherStats(service *Service) (stats serviceStats) {
	stats.ClientsAccepted = service.connectionsAccepted
	stats.OpenConnections = uint64(len(service.clients))

	return
}

func printStatsLoop(service *Service, timeout time.Duration, closeChan chan bool) {
forLoop:
	for {
		select {
		case <-time.After(timeout):
			stats := gatherStats(service)
			statsBytes, err := json.Marshal(&stats)
			if err != nil {
				continue
			}

			Trace.Printf("periodic stats (%s): `%s`", service.name, statsBytes)
		case <-closeChan:
			break forLoop
		}
	}

	Trace.Printf("exiting PrintStatsLoop")
}
