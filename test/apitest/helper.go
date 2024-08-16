package apitest

import (
	"bytes"
	"net/http"
	"testing"

	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

func GetHttp(t *testing.T, url string, token string) (*http.Response, string) {
	return request(t, "GET", url, token)
}

func PostHttp(t *testing.T, url string, body string, token string) (*http.Response, string) {
	return reqeustWithBody(t, "POST", url, body, token)
}

func PatchHttp(t *testing.T, url string, body string, token string) (*http.Response, string) {
	return reqeustWithBody(t, "PATCH", url, body, token)
}

func DeleteHttp(t *testing.T, url string, token string) (*http.Response, string) {
	return request(t, "DELETE", url, token)
}

func request(t *testing.T, method string, url string, token string) (*http.Response, string) {
	req, err := http.NewRequest(method, url, nil)
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

func reqeustWithBody(t *testing.T, method string, url string, body string, token string) (*http.Response, string) {
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
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
