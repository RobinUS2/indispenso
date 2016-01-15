package main
import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestVerifyUserHas2FactorAuth(t *testing.T) {
	user := newUser()

	assert.Equal(t,user.HasTwoFactor(),false)
	user.TotpSecret = "test"
	assert.Equal(t,user.HasTwoFactor(),false)
	user.TotpSecretValidated = true
	assert.Equal(t,user.HasTwoFactor(),true)
	user.TotpSecret = ""
	assert.Equal(t,user.HasTwoFactor(),false)
}
