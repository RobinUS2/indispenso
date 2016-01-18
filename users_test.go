package main
import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestVerifyUserHas2FactorAuth(t *testing.T) {
	user := newUser()
	assert.Equal(t,false,user.HasTwoFactor())
	user.AuthType |= AUTH_TYPE_TWO_FACTOR
	assert.Equal(t,user.HasTwoFactor(),false)
	user.TotpSecret = "test"
	assert.Equal(t,user.HasTwoFactor(),false)
	user.TotpSecretValidated = true
	assert.Equal(t,user.HasTwoFactor(),true)
	user.TotpSecret = ""
	assert.Equal(t,user.HasTwoFactor(),false)
}

func TestUserHasDefinedAnyAuthType(t *testing.T) {
	user := newUser()
	assert.Equal(t,false,user.IsAuthDefined())
	user.AuthType|= AUTH_TYPE_TWO_FACTOR
	assert.Equal(t,true,user.IsAuthDefined())
}


func TestUserHasDefinedAuthType(t *testing.T) {
	user := newUser()
	assert.Equal(t,false,user.IsAuthType(AUTH_TYPE_LOCAL))
	user.AuthType|= AUTH_TYPE_LOCAL
	assert.Equal(t,true,user.IsAuthType(AUTH_TYPE_LOCAL))
	assert.Equal(t,false,user.IsAuthType(AUTH_TYPE_LDAP))
}

func TestUSerDefinitionMigration(t *testing.T) {
	userArray := []*User{ &User{PasswordHash:"dsadsadsad"}, &User{TotpSecret:"dadsafdfa", TotpSecretValidated: true} }
	us := &UserStore{}
	us.MigrateUsers(userArray)

	assert.NotEqual(t, userArray[0].AuthType, 0 )
	assert.Equal(t, userArray[0].AuthType & AUTH_TYPE_LOCAL, AUTH_TYPE_LOCAL)
	assert.NotEqual(t, userArray[0].AuthType & AUTH_TYPE_LDAP, AUTH_TYPE_LDAP)
	assert.NotEqual(t, userArray[0].AuthType & AUTH_TYPE_TWO_FACTOR, AUTH_TYPE_TWO_FACTOR)

	assert.NotEqual(t, userArray[1].AuthType, 0 )
	assert.NotEqual(t, userArray[1].AuthType & AUTH_TYPE_LOCAL, AUTH_TYPE_LOCAL)
	assert.NotEqual(t, userArray[1].AuthType & AUTH_TYPE_LDAP, AUTH_TYPE_LDAP)
	assert.Equal(t, userArray[1].AuthType & AUTH_TYPE_TWO_FACTOR, AUTH_TYPE_TWO_FACTOR)
}