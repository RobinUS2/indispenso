package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/stretchr/testify/mock"
)

type AuthenticatorMock struct {
	mock.Mock
}

func (m *AuthenticatorMock) auth(user *User, ar *AuthRequest) (*User, error){
	args := m.Called(user,ar)
	userRet := args.Get(0)
	if userRet == nil {
		return user, args.Error(1)
	} else {
		return userRet.(*User), args.Error(1)
	}
}

func TestAuthWithoutSecondFactor(t *testing.T) {
	userStore :=  &UserStore{}
	userStore.CreateUser("test", "test", "test@test.pl", []string{})
	user := userStore.ByName("test")
	user.TotpSecret = "dsafasfasfsafsafa"
	user.TotpSecretValidated = true
	user.Enabled = true

	usAuth := newAuthService(userStore, DefaultFirstFactorAuth, nil)

	res, err := usAuth.authUser(&AuthRequest{login: "test", credential: "test", token: ""})

	assert.NoError(t, err)
	assert.Equal(t, "test", res.Username)
	assert.Equal(t, "test@test.pl", res.EmailAddress)
}

func TestAuthenticateUserOneFactor(t *testing.T) {
	userStore := &UserStore{}
	userStore.CreateUser("test", "test", "test@test.pl", []string{})
	usAuth := newAuthService(userStore, DefaultFirstFactorAuth, nil)

	res, err := usAuth.authUser( &AuthRequest{login: "test", credential: "test", token: ""})
	assert.NoError(t, err)
	assert.Equal(t, "test", res.Username)
	assert.Equal(t, "test@test.pl", res.EmailAddress)
}


func TestExistingUserWithIncorrectPassword(t *testing.T) {
	userStore := &UserStore{}
	userStore.CreateUser("test", "test", "test@test.pl", []string{})
	usAuth := newAuthService(userStore, DefaultFirstFactorAuth, nil)

	res, err := usAuth.authUser( &AuthRequest{login: "test", credential: "test_incorrect", token: ""})
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestPerformFFAuthWithNoAuthenticatorProvided(t *testing.T) {
	as := &AuthService{}
	_, err := as.performFirstFactorAuth(&User{}, &AuthRequest{})
	assert.Error(t,err)
}

func TestValidUserNotFound(t *testing.T) {
	as := newAuthService(&UserStore{},DefaultFirstFactorAuth,nil)
	user,err := as.getValidUser(&AuthRequest{login:"test", credential:"test"})
	assert.NoError(t,err)
	assert.Nil(t,user)
}

func TestFewFirstFactorAuth(t *testing.T) {
	userStore := &UserStore{}
	userStore.CreateUser("test", "test", "test@test.pl", []string{})
	usAuth := newAuthService(userStore, DefaultFirstFactorAuth, nil)
	testUser := userStore.ByName("test")

	testAuth := new(AuthenticatorMock)
	testAuth.On("auth",mock.AnythingOfType("*main.User"),mock.AnythingOfType("*main.AuthRequest")).Return(nil,nil)
	usAuth.appendFirstFactor(testAuth)

	res, err := usAuth.authUser( &AuthRequest{login: "test", credential: "test_incorrect", token: ""})
	assert.NoError(t, err)
	assert.Equal(t, testUser,res)

	testAuth.AssertExpectations(t)
}

func TestLocalAuthenticatorUserNil(t *testing.T) {
	auth := &LocalAuthenticator{}

	user, err := auth.auth(nil, &AuthRequest{})
	assert.Error(t, err)
	assert.Nil(t, user)
}
