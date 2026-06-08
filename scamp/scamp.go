// Package scamp Copyright 2014-2018 GüdTech, Inc.
// SCAMP provides SOA bus RPC functionality. Please see root SCAMP/README.md for details on configuring environment.
// Basics:
//
//	Services and requesters communicate over persistent TLS connections.
//	First, initialize your environment according to the root README.md. You must have a valid certificate and key to present a service.
//	Every program must call `scamp.Initialize()` before doing anything else, to initialize the global configuration.
package scamp
