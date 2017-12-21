package scamp

// import "errors"
// import "fmt"

import (
	"bytes"
	"fmt"

	"encoding/base64"

	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
)

var padding = []byte("=")

func verifySHA256(rawPayload []byte, rsaPubKey *rsa.PublicKey, encodedSignature []byte, isURLEncoded bool) (err error) {
	expectedSig, err := decodeUnpaddedBase64(encodedSignature, isURLEncoded)
	if err != nil {
		err = fmt.Errorf("failed to decode base64: `%s`/`%s`", err, encodedSignature)
		return
	}

	h := sha256.New()
	h.Write(rawPayload)
	digest := h.Sum(nil)

	err = rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, digest, expectedSig)
	if err != nil {
		err = fmt.Errorf("failed to verify PKCS1v15: `%s`", err)
		return
	}

	return
}

func signSHA256(rawPayload []byte, priv *rsa.PrivateKey) (base64signature string, err error) {
	h := sha256.New()
	h.Write(rawPayload)
	digest := h.Sum(nil)
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest)
	if err != nil {
		return
	}
	base64signature = base64.StdEncoding.EncodeToString(sig)
	return
}

func decodeUnpaddedBase64(incoming []byte, isURLEncoded bool) (decoded []byte, err error) {
	if isURLEncoded {
		if m := len(incoming) % 4; m != 0 {
			paddingBytes := bytes.Repeat(padding, 4-m)
			incoming = append(incoming, paddingBytes[:]...)
		}
		_, err = base64.URLEncoding.Decode(decoded, incoming)
	} else {
		decoded, err = base64.StdEncoding.DecodeString(string(incoming))
	}

	return
}
