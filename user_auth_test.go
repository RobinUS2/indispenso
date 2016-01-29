package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type AuthenticatorMock struct {
	mock.Mock
}

type UserStoreMock struct {
	mock.Mock
}

func (m *AuthenticatorMock) auth(user *User, ar *AuthRequest) (*User, error) {
	args := m.Called(user, ar)
	userRet := args.Get(0)
	if userRet == nil {
		return user, args.Error(1)
	} else {
		return userRet.(*User), args.Error(1)
	}
}

func TestAuthWithoutSecondFactor(t *testing.T) {
	userStore := &UserStore{}
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

	res, err := usAuth.authUser(&AuthRequest{login: "test", credential: "test", token: ""})
	assert.NoError(t, err)
	assert.Equal(t, "test", res.Username)
	assert.Equal(t, "test@test.pl", res.EmailAddress)
}

func TestExistingUserWithIncorrectPassword(t *testing.T) {
	userStore := &UserStore{}
	userStore.CreateUser("test", "test", "test@test.pl", []string{})
	usAuth := newAuthService(userStore, DefaultFirstFactorAuth, nil)

	res, err := usAuth.authUser(&AuthRequest{login: "test", credential: "test_incorrect", token: ""})
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestPerformFFAuthWithNoAuthenticatorProvided(t *testing.T) {
	as := &AuthService{}
	_, err := as.performFirstFactorAuth(&User{}, &AuthRequest{})
	assert.Error(t, err)
}

func TestValidUserNotFound(t *testing.T) {
	as := newAuthService(&UserStore{}, DefaultFirstFactorAuth, nil)
	user, err := as.getValidUser(&AuthRequest{login: "test", credential: "test"})
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestFewFirstFactorAuth(t *testing.T) {
	userStore := &UserStore{}
	userStore.CreateUser("test", "test", "test@test.pl", []string{})
	usAuth := newAuthService(userStore, DefaultFirstFactorAuth, nil)
	testUser := userStore.ByName("test")

	testAuth := new(AuthenticatorMock)
	testAuth.On("auth", mock.AnythingOfType("*main.User"), mock.AnythingOfType("*main.AuthRequest")).Return(nil, nil)
	usAuth.appendFirstFactor(testAuth)

	res, err := usAuth.authUser(&AuthRequest{login: "test", credential: "test_incorrect", token: ""})
	assert.NoError(t, err)
	assert.Equal(t, testUser, res)

	testAuth.AssertExpectations(t)
}

func TestLocalAuthenticatorUserNil(t *testing.T) {
	auth := &LocalAuthenticator{}

	user, err := auth.auth(nil, &AuthRequest{})
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestAuthServiceNotAllowInvalidUsers(t *testing.T) {
	userStore := &UserStore{Users: []*User{&User{Username: "disabled", Enabled: false}, &User{Username: "noauthmethod", AuthType: 0}}}
	auth := &AuthService{userStore: userStore}

	user, err := auth.getValidUser(&AuthRequest{login: "disabled"})
	assert.Nil(t, user)
	assert.Error(t, err)

	user, err = auth.getValidUser(&AuthRequest{login: "noauthmethod"})
	assert.Nil(t, user)
	assert.Error(t, err)
}

func TestAuthServiceValidUsers(t *testing.T) {
	userStore := &UserStore{Users: []*User{&User{Username: "test", Enabled: true, AuthType: AUTH_TYPE_LOCAL}}}
	auth := &AuthService{userStore: userStore}

	user, err := auth.getValidUser(&AuthRequest{login: "test"})
	assert.NotNil(t, user)
	assert.Equal(t, "test", user.Username)
	assert.NoError(t, err)
}

func TestInvalidAuthRequest(t *testing.T) {
	arEmptyLogin := &AuthRequest{login: ""}
	arEmptyPass := &AuthRequest{credential: ""}
	arEmptyLoginAndPass := &AuthRequest{credential: "", login: ""}

	assert.Error(t, arEmptyLogin.Validate())
	assert.Error(t, arEmptyPass.Validate())
	assert.Error(t, arEmptyLoginAndPass.Validate())

}

func TestValidAuthRequest(t *testing.T) {
	authRequest := &AuthRequest{login: "test", credential: "test"}

	assert.NoError(t, authRequest.Validate())
}

func TestLocalAuthenticatorInvalidPasswordHash(t *testing.T) {
	a := &LocalAuthenticator{}

	user, err := a.auth(&User{Username: "test", PasswordHash: "&(^(*^(^(*^(*(*^(^DSA"}, &AuthRequest{login: "test", credential: "test"})

	assert.Nil(t, user)
	assert.Error(t, err)
}

func TestSecondFactorRequireValidUser(t *testing.T) {
	a := GAuthAuthenticator{}

	user, err := a.auth(nil, &AuthRequest{})

	assert.Nil(t, user)
	assert.Error(t, err)
}

func TestInvalidToken(t *testing.T) {
	a := GAuthAuthenticator{}

	user, err := a.auth(&User{TotpSecret: "test"}, &AuthRequest{token: "100000"})

	assert.Nil(t, user)
	assert.Error(t, err)
}
func TestGetUserWithNoAuthDefined(t *testing.T) {
	us := &UserStore{Users: []*User{&User{Enabled: true, AuthType: 0, Username: "test"}}}
	as := &AuthService{userStore: us}

	user, err := as.getValidUser(&AuthRequest{login: "test"})
	assert.Nil(t, user)
	assert.Error(t, err)
}

func TestAuthServiceInvalidRequestOrUser(t *testing.T) {
	as := &AuthService{}

	user, err := as.authUser(&AuthRequest{})
	assert.Error(t, err)
	assert.Nil(t, user)

	user, err = as.authUser(&AuthRequest{login: "test"})
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestPerformSecondFactorAuth(t *testing.T) {
	us := &UserStore{Users: []*User{&User{Enabled: true, AuthType: AUTH_TYPE_TWO_FACTOR}}}

	ffDummyAuth := new(AuthenticatorMock)
	ffDummyAuth.On("auth", mock.Anything, mock.AnythingOfType("*main.AuthRequest")).Return(&User{TotpSecretValidated: true, AuthType: AUTH_TYPE_TWO_FACTOR, TotpSecret: "test"}, nil)

	sfDummyAuth := new(AuthenticatorMock)
	sfDummyAuth.On("auth", mock.Anything, mock.AnythingOfType("*main.AuthRequest")).Return(&User{Username: "sfAuthUser"}, nil)

	as := newAuthService(us, []Authenticator{ffDummyAuth}, sfDummyAuth)

	user, err := as.authUser(&AuthRequest{login: "test", credential: "test"})
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "sfAuthUser", user.Username)

}

func TestAuthServiceInvalidUser(t *testing.T) {
	as := &AuthService{userStore: &UserStore{Users: []*User{&User{Username: "test", Enabled: false}}}}
	user, err := as.authUser(&AuthRequest{login: "test", credential: "test"})

	assert.Error(t, err)
	assert.Nil(t, user)
}
