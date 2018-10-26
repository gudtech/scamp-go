// Package scamp Copyright 2014-2018 GüdTech, Inc.
// SCAMP provides SOA bus RPC functionality. Please see root SCAMP/README.md for details on configuring environment.
// Basics:
// 	Services and requesters communicate over persistent TLS connections.
//	First, initialize your environment according to the root README.md. You must have a valid certificate and key to present a service.
//	Every program must call `scamp.Initialize()` before doing anything else, to initialize the global configuration.
package scamp

import (
	"fmt"
	"log"
)

// const (
// 	defaultMessageTimeout  = time.Second * 120
// 	defaultLivenessDirPath = "/backplane/running-services/"
// 	defaultConfigPath      = "/backplane/etc"
// )

// // DefaultCache is the default service cache
// var DefaultCache *ServiceCache

// // Options that can be passed at time of service creation
// // TODO: after doing so, define defaultServiceOptions (in Init()) to set these values when
// // a user doesn't
// type Options struct {
// 	// KeyPath path to service private key
// 	KeyPath string
// 	// CertPath path to certificate used for signing
// 	CertPath string
// 	// AnnouncePath payload to be signed
// 	AnnouncePath string
// 	// SOAConfigPath path to the soa.conf file
// 	// the soa.conf file must include the following keys:
// 	//	1) discovery.cache_path
// 	SOAConfigPath string
// 	// LivenessFilePath in Kubernetes environments, scamp services write an empty file to a
// 	// designated directory to facilitate auto-scaling
// 	LivenessFilePath string
// }

// var defaultServiceOptions = Options{
// 	KeyPath:          "",
// 	CertPath:         "",
// 	AnnouncePath:     "",
// 	SOAConfigPath:    defaultConfigPath,
// 	LivenessFilePath: defaultLivenessDirPath,
// }

// setup scamp environment using either passed Options struct or defaultServiceOptions
// TODO: use passed options. These must supercede default options
func init() {
	initSCAMPLogger()
	err := initConfig(defaultServiceOptions.SOAConfigPath)
	if err != nil {
		log.Fatalf("could not initialize scamp environment %s\n", err)
	}

	cachePath, ok := DefaultConfig().Get("discovery.cache_path")
	if !ok {
		err = fmt.Errorf("no such config param `discovery.cache_path`: this key must be present in soa.conf to use scamp-go")
		return
	}

	DefaultCache, err = NewServiceCache(cachePath)
	if err != nil {
		return
	}

	return
}

// TODO: scamp shouldnt have a main function (this is a library) and we shouldn't be using
// flags here. We can substitute an options or config struct with sensible defaults
// func main() {
// var keyPath string
// var certPath string
// var fingerprintPath string
// var announcePath string

// gtConfigPathPtr := flag.String("config", defaultConfigPath, "path to soa.conf")

// flag.StringVar(&announcePath, "announcepath", "", "payload to be signed")
// flag.StringVar(&certPath, "certpath", "", "path to cert used for signing")
// flag.StringVar(&keyPath, "keypath", "", "path to service private key")

// TODO: Fingerprinting should be done in a separate, executable utility we provide with the scamp library
// flag.StringVar(&fingerprintPath, "fingerprintpath", "", "path to cert to fingerprint")
// flag.Parse()

// TODO: move initialize to package init()
// Initialize(*gtConfigPathPtr)

// TODO: Move this logic to Init()
// if (len(keyPath) == 0 || len(announcePath) == 0 || len(certPath) == 0) && (len(fingerprintPath) == 0) {
// 	fmt.Printf("fingerprintpath: %s\n", fingerprintPath)
// 	fmt.Println("not enough options specified, must provide: certpath, keypath, and announcepath, OR fingerprintpath")
// 	return
// }

// if len(keyPath) != 0 {
// 	doFakeDiscoveryCache(keyPath, certPath, announcePath)
// } else {
// 	doCertFingerprint(fingerprintPath)
// }

// }

// func doFakeDiscoveryCache(keyPath, certPath, announcePath string) {
// 	keyRawBytes, err := ioutil.ReadFile(keyPath)
// 	if err != nil {
// 		Error.Fatalf("could not read key at %s", keyPath)
// 	}

// 	block, _ := pem.Decode(keyRawBytes)

// 	if block == nil {
// 		Error.Fatalf("could not decode key data (%s)", block.Type)
// 		return
// 	} else if block.Type != "RSA PRIVATE KEY" {
// 		Error.Fatalf("expected key type 'RSA PRIVATE KEY' but got '%s'", block.Type)
// 	}

// 	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
// 	if err != nil {
// 		Error.Fatalf("could not parse key from %s (%s)", keyPath, block.Type)
// 	}

// 	announceData, err := ioutil.ReadFile(announcePath)
// 	if err != nil {
// 		Error.Fatalf("could not read announce data from %s", announcePath)
// 	}
// 	announceSig, err := signSHA256([]byte(announceData), privKey)
// 	if err != nil {
// 		Error.Fatalf("could not sign announce data: %s", err)
// 	}

// 	certData, err := ioutil.ReadFile(certPath)
// 	if err != nil {
// 		Error.Fatalf("could not read cert from %s", certPath)
// 	}

// 	fmt.Printf("\n%%%%%%\n%s\n\n%s\n\n%s\n", announceData, bytes.TrimSpace(certData), announceSig)
// }

// func doCertFingerprint(fingerprintPath string) {
// 	certData, err := ioutil.ReadFile(fingerprintPath)
// 	if err != nil {
// 		Error.Fatalf("could not read cert from %s", fingerprintPath)
// 	}

// 	decoded, _ := pem.Decode(certData)
// 	if decoded == nil {
// 		Error.Fatalf("could not decode cert. is it PEM encoded?")
// 	}

// 	// Put pem in form useful for fingerprinting
// 	cert, err := x509.ParseCertificate(decoded.Bytes)
// 	if err != nil {
// 		Error.Fatalf("could not parse certificate. is it valid x509?")
// 	}

// 	fingerprint := GetSHA1FingerPrint(cert)
// 	if len(fingerprint) > 0 {
// 		fmt.Printf("fingerprint: %s\n", fingerprint)
// 	} else {
// 		Error.Fatalf("could not fingerprint certificate")
// 	}
// }
