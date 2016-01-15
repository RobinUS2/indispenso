package main
import "errors"

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
	as.authenticator = newUserStoreAuthenticator(us,newGAuthAuthenticator())
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
}

func newUserStoreAuthenticator(us *UserStore, sfAuth Authenticator) *UserStoreAuthenticator{
	res := new(UserStoreAuthenticator)
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

	authRes := a.userStore.Auth(user.PasswordHash, ar.credential)
	if !authRes {
		return nil, errors.New("Password and hash doesn't match")
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
