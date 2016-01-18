package main
import (
	"errors"
	"encoding/base64"
	"golang.org/x/crypto/bcrypt"
)


var FirstFactorAuth = map[AuthType]Authenticator {
	AUTH_TYPE_LOCAL : &LocalAuthenticator{},
	//TYPE_LDAP : LdapAuthenticator{},
}

type AuthRequest struct {
	login  string
	credential string
	token string
}

func (ar *AuthRequest) Validate() (bool, error) {
	if len(ar.login) < 1 || len(ar.credential) < 1 {
		return false, errors.New("Empty login or credential")
	}
	return true, nil
}

type Authenticator interface {
	auth(user *User, ar *AuthRequest) (*User, error)
}


type AuthService struct {
	authenticator Authenticator
}

func newAuthService(us *UserStore) (as *AuthService) {
	as = new(AuthService)
	GAuth := newGAuthAuthenticator()
	as.authenticator = newUserStoreAuthenticator(us,FirstFactorAuth, GAuth)
	return
}

func (as* AuthService) authUser( ar *AuthRequest ) (*User, error) {
	if r, err :=ar.Validate(); r {
		return nil, err
	}
	return  as.authenticator.auth( nil, ar)
}

type UserStoreAuthenticator struct {
	userStore        *UserStore
	secondFactorAuth Authenticator
	firstFactorAuth map[AuthType]Authenticator
}

func newUserStoreAuthenticator(us *UserStore, ffAuth map[AuthType]Authenticator, sfAuth Authenticator) *UserStoreAuthenticator{
	res := new(UserStoreAuthenticator)
	res.firstFactorAuth = ffAuth
	res.secondFactorAuth = sfAuth
	res.userStore = us
	return res
}

func (a *UserStoreAuthenticator) auth(user *User, ar *AuthRequest) (*User, error)  {
	if user == nil {
		user = a.userStore.ByName(ar.login)
		if user == nil || !user.Enabled {
			return nil, errors.New("User doesn't exists")
		}
	}

	if !user.IsAuthDefined() {
		return nil, errors.New("User doesn't have auth type configured")
	}

	for k,v := range a.firstFactorAuth {
		if user.IsAuthType(k) {
			_, err := v.auth(user,ar)
			if err == nil {
				break
			}
		}
	}

	if a.secondFactorAuth != nil && user.HasTwoFactor() {
		return a.secondFactorAuth.auth(user, ar)
	}

	return user, nil
}

type GAuthAuthenticator struct {}

func newGAuthAuthenticator() *GAuthAuthenticator{
	res := new(GAuthAuthenticator)
	return res
}

func (a *GAuthAuthenticator) auth(user *User, ar *AuthRequest) (*User, error)  {
	if user == nil {
		return nil, errors.New("User not found")
	}
	//todo extract validation from user
	if res, err := user.ValidateTotp( ar.token ); res {
		return user, nil
	}else{
		return nil, err
	}

}

type LdapAuthenticator struct {
}

func (a *LdapAuthenticator) auth(user *User, ar *AuthRequest) (*User, error)  {
	return nil, errors.New("Not implemented")
}

type LocalAuthenticator struct{}

func (a *LocalAuthenticator) auth(user *User, ar *AuthRequest) (*User, error)  {
	bytes, be := base64.URLEncoding.DecodeString(user.PasswordHash)
	if be != nil {
		log.Printf("%s", be)
		bytes = make([]byte, 0)
	}

	if bcrypt.CompareHashAndPassword(bytes, []byte(ar.credential)) == nil {
		return nil, errors.New("Password and hash doesn't match")
	}
	return user, nil
}