package apitest

import (
	"bytes"
	"net/http"
	"testing"
)

func GetHttp(t *testing.T, url string, token string) *http.Response {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	return resp
}

func PostHttp(t *testing.T, url string, body string, token string) *http.Response {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	return resp
}
