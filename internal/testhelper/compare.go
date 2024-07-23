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
	assertEqualRecursive(t, reflect.ValueOf(expected), reflect.ValueOf(actual), "", ignoredFields)
}

func assertEqualRecursive(t *testing.T, expected, actual reflect.Value, path string, ignoredFields []string) {
	t.Helper()

	if !expected.IsValid() || !actual.IsValid() {
		assert.Equal(t, expected.IsValid(), actual.IsValid(), fmt.Sprintf("Validity mismatch at %s", path))
		return
	}

	switch expected.Kind() {
	case reflect.Map:
		assertEqualMap(t, expected, actual, path, ignoredFields)
	case reflect.Struct:
		assertEqualStruct(t, expected, actual, path, ignoredFields)
	case reflect.Slice, reflect.Array:
		assertEqualSlice(t, expected, actual, path, ignoredFields)
	case reflect.Ptr:
		if !expected.IsNil() && !actual.IsNil() {
			assertEqualRecursive(t, expected.Elem(), actual.Elem(), path, ignoredFields)
		} else {
			assert.Equal(t, expected.IsNil(), actual.IsNil(), fmt.Sprintf("Pointer nullity mismatch at %s", path))
		}
	default:
		assert.EqualValues(t, expected.Interface(), actual.Interface(), fmt.Sprintf("Value mismatch at %s", path))
	}
}

func assertEqualMap(t *testing.T, expected, actual reflect.Value, path string, ignoredFields []string) {
	t.Helper()

	assert.Equal(t, expected.Len(), actual.Len(), fmt.Sprintf("Map length mismatch at %s", path))

	for _, key := range expected.MapKeys() {
		keyStr := fmt.Sprintf("%v", key.Interface())
		if lo.Contains(ignoredFields, keyStr) {
			continue
		}
		newPath := fmt.Sprintf("%s.%s", path, keyStr)
		expectedValue := expected.MapIndex(key)
		actualValue := actual.MapIndex(key)
		if !actualValue.IsValid() {
			t.Errorf("Missing key %s in actual map at %s", keyStr, path)
			continue
		}
		assertEqualRecursive(t, expectedValue, actualValue, newPath, ignoredFields)
	}
}

func assertEqualStruct(t *testing.T, expected, actual reflect.Value, path string, ignoredFields []string) {
	t.Helper()

	for i := 0; i < expected.NumField(); i++ {
		fieldName := expected.Type().Field(i).Name
		if lo.Contains(ignoredFields, fieldName) {
			continue
		}
		newPath := fmt.Sprintf("%s.%s", path, fieldName)
		assertEqualRecursive(t, expected.Field(i), actual.Field(i), newPath, ignoredFields)
	}
}

func assertEqualSlice(t *testing.T, expected, actual reflect.Value, path string, ignoredFields []string) {
	t.Helper()

	assert.Equal(t, expected.Len(), actual.Len(), fmt.Sprintf("Slice length mismatch at %s", path))

	for i := 0; i < expected.Len(); i++ {
		newPath := fmt.Sprintf("%s[%d]", path, i)
		assertEqualRecursive(t, expected.Index(i), actual.Index(i), newPath, ignoredFields)
	}
}
