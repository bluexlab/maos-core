package apitest

import (
	"bytes"
	"net/http"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func GetHttp(t *testing.T, url string, token string) (*http.Response, string) {
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

	resBody, err := testhelper.ReadBody(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	client.CloseIdleConnections()
	return resp, resBody
}

func PostHttp(t *testing.T, url string, body string, token string) (*http.Response, string) {
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

	resBody, err := testhelper.ReadBody(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}
	client.CloseIdleConnections()
	return resp, resBody
}
