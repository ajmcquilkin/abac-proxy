package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"github.com/abac/proxy/internal/log"
)

type ExampleInterceptor struct {
	AddHeaders    map[string]string
	ModifyJSON    bool
	JSONTransform func(map[string]interface{}) map[string]interface{}
}

func (e *ExampleInterceptor) InterceptRequest(req *http.Request) *http.Request {
	ctx := req.Context()
	logger := log.From(ctx)

	for key, value := range e.AddHeaders {
		req.Header.Set(key, value)
		logger.Debug("added header to request",
			zap.String("header", key),
			zap.String("value", value),
		)
	}

	return req
}

func (e *ExampleInterceptor) InterceptResponse(resp *http.Response) error {
	ctx := resp.Request.Context()
	logger := log.From(ctx)

	if !e.ModifyJSON {
		return nil
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil
	}

	return ReadAndReplaceBody(resp, func(body []byte) ([]byte, error) {
		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			logger.Warn("failed to parse JSON response", zap.Error(err))
			return body, nil
		}

		if e.JSONTransform != nil {
			data = e.JSONTransform(data)
		} else {
			data["_intercepted"] = true
			data["_proxy"] = "abac-proxy"
		}

		modified, err := json.Marshal(data)
		if err != nil {
			logger.Error("failed to marshal modified JSON", zap.Error(err))
			return body, nil
		}

		logger.Debug("modified JSON response",
			zap.Int("original_size", len(body)),
			zap.Int("modified_size", len(modified)),
		)

		return modified, nil
	})
}

type LoggingInterceptor struct {
	LogBodies bool
}

func (l *LoggingInterceptor) InterceptRequest(req *http.Request) *http.Request {
	ctx := req.Context()
	logger := log.From(ctx)

	logger.Info("request details",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Any("headers", req.Header),
	)

	return req
}

func (l *LoggingInterceptor) InterceptResponse(resp *http.Response) error {
	ctx := resp.Request.Context()
	logger := log.From(ctx)

	logger.Info("response details",
		zap.Int("status", resp.StatusCode),
		zap.Any("headers", resp.Header),
	)

	if l.LogBodies {
		return ReadAndReplaceBody(resp, func(body []byte) ([]byte, error) {
			logger.Debug("response body", zap.ByteString("body", body))
			return body, nil
		})
	}

	return nil
}

type ChainInterceptor struct {
	Interceptors []Interceptor
}

func (c *ChainInterceptor) InterceptRequest(req *http.Request) *http.Request {
	for _, interceptor := range c.Interceptors {
		req = interceptor.InterceptRequest(req)
	}
	return req
}

func (c *ChainInterceptor) InterceptResponse(resp *http.Response) error {
	for _, interceptor := range c.Interceptors {
		if err := interceptor.InterceptResponse(resp); err != nil {
			return err
		}
	}
	return nil
}

func CloneResponseBody(resp *http.Response) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	if resp.Body == nil {
		return &buf, nil
	}
	defer resp.Body.Close()

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	return &buf, nil
}
