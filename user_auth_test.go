package main
import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestUserStoreAuthWithoutSecondFactor(t *testing.T) {
	userStore := newUserStore()
	userStore.CreateUser("test","test","test@test.pl", []string{})
	user := userStore.ByName("test")
	user.TotpSecret = "dsafasfasfsafsafa"
	user.TotpSecretValidated = true

	usAuth := newUserStoreAuthenticator(userStore,FirstFactorAuth,nil)

	res, err := usAuth.auth(nil,&AuthRequest{login:"test",credential:"test", token:""})

	assert.NoError(t,err)
	assert.Equal(t,"test",res.Username)
	assert.Equal(t,"test@test.pl",res.EmailAddress)
}

func TestAuthenticateUserOneFactor(t *testing.T) {
	userStore := newUserStore()
	userStore.CreateUser("test","test","test@test.pl", []string{})
	usAuth := newUserStoreAuthenticator(userStore,FirstFactorAuth,nil)

	res, err := usAuth.auth(nil,&AuthRequest{login:"test",credential:"test", token:""})
	assert.NoError(t,err)
	assert.Equal(t,"test",res.Username)
	assert.Equal(t,"test@test.pl",res.EmailAddress)
}
