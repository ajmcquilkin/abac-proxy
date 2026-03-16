package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type Interceptor interface {
	InterceptRequest(req *http.Request) *http.Request
	InterceptResponse(resp *http.Response) error
}

type PassthroughInterceptor struct{}

func (p *PassthroughInterceptor) InterceptRequest(req *http.Request) *http.Request {
	return req
}

func (p *PassthroughInterceptor) InterceptResponse(resp *http.Response) error {
	return nil
}

func ReadAndReplaceBody(resp *http.Response, modifier func([]byte) ([]byte, error)) error {
	if resp.Body == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	modifiedBody, err := modifier(body)
	if err != nil {
		return err
	}

	resp.Body = io.NopCloser(bytes.NewReader(modifiedBody))
	resp.ContentLength = int64(len(modifiedBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))

	return nil
}
