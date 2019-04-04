package scamp

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"
)

var verifyKey *rsa.PublicKey
var readVerifyKey sync.Once

// Ticket represents an SOA ticket
type Ticket struct {
	Version   int
	UserID    int
	ClientID  int
	Timestamp int64
	// TTL = Time To Live
	TTL        int64
	Privileges map[int]bool
}

var defaultKeyPath = "/etc/GT/auth/ticket_verify_public_key.pem"

func VerifyTicket(unparsedTicket string, keyPath string) (*Ticket, error) {
	if len(keyPath) == 0 {
		keyPath = defaultKeyPath
	}

	parts := strings.Split(strings.TrimSpace(unparsedTicket), ",")
	if len(parts) < 6 {
		return nil, fmt.Errorf("ticket missing parts, wanted 6 parts, have %d", len(parts))
	}

	sig := parts[len(parts)-1]
	parts = parts[:len(parts)-1]

	readVerifyKey.Do(func() {
		vkStr, err := ioutil.ReadFile(keyPath)
		if err != nil {
			panic(err)
		}

		block, _ := pem.Decode(vkStr)
		if block == nil {
			panic("can't decode nil pem")
		}

		pk, _ := x509.ParsePKIXPublicKey(block.Bytes)
		verifyKey = pk.(*rsa.PublicKey)
	})

	if verifyKey == nil {
		return nil, fmt.Errorf("`%s` verify key not readable", keyPath)
	}

	signature, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %s", err)
	}

	hashed := sha256.Sum256([]byte(strings.Join(parts, ",")))

	verifyErr := rsa.VerifyPKCS1v15(verifyKey, crypto.SHA256, hashed[:], signature)
	if verifyErr != nil {
		return nil, fmt.Errorf("unable to verify ticket: %s", verifyErr)
	}

	if parts[0] != "1" { // version
		return nil, fmt.Errorf("invalid version")
	}

	var ticket Ticket
	ticket.Version = 1

	ticket.UserID, err = strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("parse user id (%v): %s", parts[1], err)
	}

	ticket.ClientID, err = strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("parse client id (%v): %s", parts[2], err)
	}

	ticket.Timestamp, err = strconv.ParseInt(parts[3], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp (%v): %s", parts[3], err)
	}

	ticket.TTL, err = strconv.ParseInt(parts[4], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("parse TTL: %s", err)
	}

	if len(parts) > 5 {
		ticket.Privileges = make(map[int]bool)
		for _, priv := range strings.Split(parts[5], "+") {
			priv, err := strconv.Atoi(priv)
			if err != nil {
				return nil, fmt.Errorf("parse priv (%v): %s", priv, err)
			}
			ticket.Privileges[priv] = true
		}
	}

	if ticket.Expired() {
		return nil, fmt.Errorf("ticket expired")
	}

	return &ticket, nil
}

func (ticket *Ticket) Expired() bool {
	expiryDate := ticket.Timestamp + ticket.TTL
	return expiryDate < time.Now().Unix()
}

func (ticket *Ticket) CheckPrivs(privs []int) error {
	var missingPrivs []int
	for _, priv := range privs {
		found, ok := ticket.Privileges[priv]
		if !found || !ok {
			missingPrivs = append(missingPrivs, priv)
		}
	}

	if len(missingPrivs) > 0 {
		return fmt.Errorf("missing privileges: %v", missingPrivs)
	}

	return nil
}
