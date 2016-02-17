package data_table

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSearchMatch(t *testing.T) {
	sd := &SearchDefinition{Value: "", Regexp: false}

	assert.True(t, sd.IsMatch(""))
	assert.True(t, sd.IsMatch("test"))

	sd.Value = "test"

	assert.False(t, sd.IsMatch(""))
	assert.False(t, sd.IsMatch("asd"))
	assert.True(t, sd.IsMatch("test"))

}

func TestSearchSimpleApplicable(t *testing.T) {
	s := &SearchDefinition{Value: "", Regexp: false}

	assert.False(t, s.IsApplicable())
	s.Value = "test"
	assert.True(t, s.IsApplicable())
}

func TestSearchRegExpApplicable(t *testing.T) {
	s := &SearchDefinition{Value: "[", Regexp: true}
	assert.False(t, s.IsApplicable())

	s = &SearchDefinition{Value: "test", Regexp: true}
	assert.True(t, s.IsApplicable())
}
