package main

import (
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"fmt"
)

var DefaultFirstFactorAuth = []Authenticator{ &LocalAuthenticator{} }

type AuthRequest struct {
	login      string
	credential string
	token      string
}

func (ar *AuthRequest) Validate() (error) {
	if len(ar.login) < 1 || len(ar.credential) < 1 {
		return errors.New("Empty login or credential")
	}
	return nil
}

type Authenticator interface {
	auth(user *User, ar *AuthRequest) (*User, error)
}

type AuthService struct {
	userStore        *UserStore
	secondFactorAuth Authenticator
	firstFactorAuth  []Authenticator
}


func newAuthService(us *UserStore, ffAuth []Authenticator, sfAuth Authenticator) (as *AuthService) {
	as = new(AuthService)
	as.userStore = us
	as.firstFactorAuth = ffAuth
	as.secondFactorAuth = sfAuth
	return
}

func (as *AuthService) appendFirstFactor( auth Authenticator ) {
	as.firstFactorAuth = append(as.firstFactorAuth,auth)
}

func (as *AuthService) authUser(ar *AuthRequest) (user *User, err error) {

	if err = ar.Validate(); err != nil {
		return
	}

	user, err = as.getValidUser(ar)
	if err != nil {
		return
	}
	user, err = as.performFirstFactorAuth(user,ar)
	if err != nil || user == nil {
		return nil, fmt.Errorf("User not authenticated, last authenticator error: %s", err)
	}

	if as.secondFactorAuth != nil && user.HasTwoFactor() {
		return as.secondFactorAuth.auth(user, ar)
	}

	return
}

func (as *AuthService) getValidUser(ar *AuthRequest) (user *User, err error){
	user = as.userStore.ByName(ar.login)

	if user != nil {
		if !user.Enabled {
			return nil, errors.New("User not enabled")
		}
		if !user.IsAuthDefined() {
			return nil, errors.New("User doesn't have auth type configured")
		}
	}

	return
}

func (as *AuthService) performFirstFactorAuth(user *User, ar *AuthRequest) (authenticatedUser *User, err error) {
	err = errors.New("Misconfigured Auth Service, there are no First Factor Authenticators defined")
	for _, v := range as.firstFactorAuth {
		authenticatedUser, err = v.auth(user, ar)
		if err == nil {
			return
		}
	}

	return user, err
}


type GAuthAuthenticator struct{}

func newGAuthAuthenticator() *GAuthAuthenticator {
	res := new(GAuthAuthenticator)
	return res
}

func (a *GAuthAuthenticator) auth(user *User, ar *AuthRequest) (*User, error) {
	if user == nil {
		return nil, errors.New("User not found")
	}

	if res, err := user.ValidateTotp(ar.token); res {
		return user, nil
	} else {
		if err == nil{
			err = errors.New("Invalid token/Unknown error")
		}
		return nil, err
	}

}

type LocalAuthenticator struct{}

func (a *LocalAuthenticator) auth(user *User, ar *AuthRequest) (*User, error) {
	if user == nil {
		return nil, errors.New("Cannot authenticate for unknown user.")
	}

	bytes, be := base64.URLEncoding.DecodeString(user.PasswordHash)
	if be != nil {
		log.Printf("%s", be)
		bytes = make([]byte, 0)
	}

	if bcrypt.CompareHashAndPassword(bytes, []byte(ar.credential)) != nil {
		return nil, errors.New("Password and hash doesn't match")
	}
	return user, nil
}
