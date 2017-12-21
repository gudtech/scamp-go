// Package scamp Copyright 2014 GüdTech, Inc.
// SCAMP provides SOA bus RPC functionality. Please see root SCAMP/README.md for details on configuring environment.
// Basics:
// 	Services and requesters communicate over persistent TLS connections.
//	First, initialize your environment according to the root README.md. You must have a valid certificate and key to present a service.
//	Every program must call `scamp.Initialize()` before doing anything else, to initialize the global configuration.
package scamp

import (
	"fmt"
	"time"
)

var defaultCache *serviceCache

//Initialize performs package-level setup. This must be called before calling any other package functionality, as it sets up global configuration.
func Initialize(configPath string) (err error) {
	initSCAMPLogger()
	err = initConfig(configPath)
	if err != nil {
		return
	}

	cachePath, found := DefaultConfig().Get("discovery.cache_path")
	if !found {
		err = fmt.Errorf("no such config param `discovery.cache_path`. must be set to use scamp-go")
		return
	}

	defaultCache, err = newServiceCache(cachePath)
	if err != nil {
		return
	}

	return
}

type stop struct {
	error
}

func retry(attempts int, sleep time.Duration, fn func() error) error {
	if err := fn(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			return retry(attempts, 2*sleep, fn)
		}
		return err
	}
	return nil
}
