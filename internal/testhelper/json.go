package testhelper

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func SerializeToJson(t *testing.T, value interface{}) string {
	t.Helper()

	jsonBytes, err := json.Marshal(value)
	require.NoError(t, err)

	return string(jsonBytes)
}
