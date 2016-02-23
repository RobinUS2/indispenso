package data_table

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseSimpleKey(t *testing.T) {
	res := ParseKey("test[0][100][test]")
	assert.Equal(t, []string{"test", "0", "100", "test"}, res)
}

func TestParseFlatKey(t *testing.T) {
	res := ParseKey("test")
	assert.Equal(t, []string{"test"}, res)
}

func TestPostTreeStruct(t *testing.T) {
	testValues := map[string][]string{"test[0][100]": []string{"test100"}}

	expectedResult := map[string]interface{}{
		"test": map[string]interface{}{
			"0": map[string]interface{}{
				"100": "test100",
			},
		},
	}
	assert.Equal(t, expectedResult, TreePostFormValues(testValues))
}
