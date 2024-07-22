package testhelper

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func AssertEqualIgnoringFields(t *testing.T, expected, actual interface{}, ignoredFields ...string) {
	t.Helper()

	expectedValue := reflect.ValueOf(expected)
	actualValue := reflect.ValueOf(actual)

	for i := 0; i < expectedValue.NumField(); i++ {
		fieldName := expectedValue.Type().Field(i).Name
		if !lo.Contains(ignoredFields, fieldName) {
			assert.Equal(t, expectedValue.Field(i).Interface(), actualValue.Field(i).Interface(), fmt.Sprintf("Field %s not equal", fieldName))
		}
	}
}
