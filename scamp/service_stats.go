package scamp

import (
	"encoding/json"
	"time"
)

type ServiceStats struct {
	ClientsAccepted uint64 `json:"total_clients_accepted"`
	OpenConnections uint64 `json:"open_connections"`
}

func GatherStats(service *Service) (stats ServiceStats) {
	stats.ClientsAccepted = service.connectionsAccepted
	stats.OpenConnections = uint64(len(service.clients))

	return
}

func PrintStatsLoop(service *Service, timeout time.Duration, closeChan chan bool) {
forLoop:
	for {
		select {
		case <-time.After(timeout):
			stats := GatherStats(service)
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
