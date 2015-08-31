package main

import (
	crand "crypto/rand"
	"encoding/base64"
	"math/rand"
)

// Secure random strings implementation

func secureRandomString(c int) (string, error) {
	var randStr string
	b := make([]byte, c)
	_, randErr := crand.Read(b)
	if randErr != nil {
		log.Printf("Error during random number generation: %s", randErr)
		return "", randErr
	}
	randStr = base64.URLEncoding.EncodeToString(b)
	return randStr, nil
}

const totpChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567" // base32 set

func TotpSecret() string {
	n := 16
	b := make([]byte, n)
	for i := range b {
		b[i] = totpChars[rand.Intn(len(totpChars))]
	}
	return string(b)
}
