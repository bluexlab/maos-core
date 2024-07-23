package apitest

import (
	"bytes"
	"net/http"
	"testing"
)

func PostHttp(t *testing.T, url string, body string, token string) *http.Response {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	return resp
}
