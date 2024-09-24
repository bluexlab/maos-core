package testhelper

import (
	"bytes"
	"io"
)

func ReadBody(body io.ReadCloser) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
