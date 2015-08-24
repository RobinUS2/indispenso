package main

import (
	crand "crypto/rand"
	"encoding/base64"
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
