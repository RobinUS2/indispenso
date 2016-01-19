package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserStoreAuthWithoutSecondFactor(t *testing.T) {
	userStore := newUserStore()
	userStore.CreateUser("test", "test", "test@test.pl", []string{})
	user := userStore.ByName("test")
	user.TotpSecret = "dsafasfasfsafsafa"
	user.TotpSecretValidated = true
	user.Enabled = true

	usAuth := newUserStoreAuthenticator(userStore, FirstFactorAuth, nil)

	res, err := usAuth.auth(nil, &AuthRequest{login: "test", credential: "test", token: ""})

	assert.NoError(t, err)
	assert.Equal(t, "test", res.Username)
	assert.Equal(t, "test@test.pl", res.EmailAddress)
}

func TestAuthenticateUserOneFactor(t *testing.T) {
	userStore := newUserStore()
	userStore.CreateUser("test", "test", "test@test.pl", []string{})
	usAuth := newUserStoreAuthenticator(userStore, FirstFactorAuth, nil)

	res, err := usAuth.auth(nil, &AuthRequest{login: "test", credential: "test", token: ""})
	assert.NoError(t, err)
	assert.Equal(t, "test", res.Username)
	assert.Equal(t, "test@test.pl", res.EmailAddress)
}

func TestLocalAuthenticatorUserNil(t *testing.T) {
	auth := &LocalAuthenticator{}

	user, err := auth.auth(nil, &AuthRequest{})
	assert.Error(t, err)
	assert.Nil(t, user)
}
