package apitest

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func GetHttp(t *testing.T, url, token string) (*http.Response, string) {
	return request(t, http.MethodGet, url, nil, token, nil)
}

func PostHttp(t *testing.T, url, body, token string) (*http.Response, string) {
	return request(t, http.MethodPost, url, bytes.NewBufferString(body), token, nil)
}

func PatchHttp(t *testing.T, url, body, token string) (*http.Response, string) {
	return request(t, http.MethodPatch, url, bytes.NewBufferString(body), token, nil)
}

func DeleteHttp(t *testing.T, url, token string) (*http.Response, string) {
	return request(t, http.MethodDelete, url, nil, token, nil)
}

func GetHttpWithHeader(t *testing.T, url, token string, headers map[string]string) (*http.Response, string) {
	return request(t, http.MethodGet, url, nil, token, headers)
}

func request(t *testing.T, method, url string, body io.Reader, token string, headers map[string]string) (*http.Response, string) {
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+token)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(resBody)
}
