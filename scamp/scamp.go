package scamp

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
)

func main() {
	var keyPath string
	var certPath string
	var fingerprintPath string
	var announcePath string

	gtConfigPathPtr := flag.String("config", "/backplane/discovery/discovery", "path to the discovery file")

	flag.StringVar(&announcePath, "announcepath", "", "payload to be signed")
	flag.StringVar(&certPath, "certpath", "", "path to cert used for signing")
	flag.StringVar(&keyPath, "keypath", "", "path to service private key")
	flag.StringVar(&fingerprintPath, "fingerprintpath", "", "path to cert to fingerprint")
	flag.Parse()

	Initialize(*gtConfigPathPtr)

	if (len(keyPath) == 0 || len(announcePath) == 0 || len(certPath) == 0) && (len(fingerprintPath) == 0) {
		fmt.Printf("fingerprintpath: %s\n", fingerprintPath)
		fmt.Println("not enough options specified, must provide: tcertpath, keypath, and announcepath, OR fingerprintpath")
		return
	}

	if len(keyPath) != 0 {
		doFakeDiscoveryCache(keyPath, certPath, announcePath)
	} else {
		doCertFingerprint(fingerprintPath)
	}

}

func doFakeDiscoveryCache(keyPath, certPath, announcePath string) {
	keyRawBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		Error.Fatalf("could not read key at %s", keyPath)
	}

	block, _ := pem.Decode(keyRawBytes)

	if block == nil {
		Error.Fatalf("could not decode key data (%s)", block.Type)
		return
	} else if block.Type != "RSA PRIVATE KEY" {
		Error.Fatalf("expected key type 'RSA PRIVATE KEY' but got '%s'", block.Type)
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		Error.Fatalf("could not parse key from %s (%s)", keyPath, block.Type)
	}

	announceData, err := ioutil.ReadFile(announcePath)
	if err != nil {
		Error.Fatalf("could not read announce data from %s", announcePath)
	}
	announceSig, err := signSHA256([]byte(announceData), privKey)
	if err != nil {
		Error.Fatalf("could not sign announce data: %s", err)
	}

	certData, err := ioutil.ReadFile(certPath)
	if err != nil {
		Error.Fatalf("could not read cert from %s", certPath)
	}

	fmt.Printf("\n%%%%%%\n%s\n\n%s\n\n%s\n", announceData, bytes.TrimSpace(certData), announceSig)
}

func doCertFingerprint(fingerprintPath string) {
	certData, err := ioutil.ReadFile(fingerprintPath)
	if err != nil {
		Error.Fatalf("could not read cert from %s", fingerprintPath)
	}

	decoded, _ := pem.Decode(certData)
	if decoded == nil {
		Error.Fatalf("could not decode cert. is it PEM encoded?")
	}

	// Put pem in form useful for fingerprinting
	cert, err := x509.ParseCertificate(decoded.Bytes)
	if err != nil {
		Error.Fatalf("could not parse certificate. is it valid x509?")
	}

	fingerprint := GetSHA1FingerPrint(cert)
	if len(fingerprint) > 0 {
		fmt.Printf("fingerprint: %s\n", fingerprint)
	} else {
		Error.Fatalf("could not fingerprint certificate")
	}
}
