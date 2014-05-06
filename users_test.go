package main

// Test users

import (
	"testing"
)

// Test hash
func TestHash(t *testing.T) {
	const pwd string = "my_super_secret_password"
	const salt string = "my_salt"
	const out string = "1241a3bd274dbbe14e7a608d658f7915d5442393051a41a8fa67dc0bd9dc80e5a4f154d39b7c7c9e9ebefe943568e30427bc625b6540096f6cc13d9edd7b6ebb"
	if x := HashPassword(pwd, salt); x != out {
		t.Errorf("HashPassword(%s, %s) = %s, want %s", pwd, salt, x, out)
	}
}
