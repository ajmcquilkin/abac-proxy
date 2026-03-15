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

type ModifyingInterceptor struct {
	OnRequest  func(*http.Request) *http.Request
	OnResponse func(*http.Response) error
}

func (m *ModifyingInterceptor) InterceptRequest(req *http.Request) *http.Request {
	if m.OnRequest != nil {
		return m.OnRequest(req)
	}
	return req
}

func (m *ModifyingInterceptor) InterceptResponse(resp *http.Response) error {
	if m.OnResponse != nil {
		return m.OnResponse(resp)
	}
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
