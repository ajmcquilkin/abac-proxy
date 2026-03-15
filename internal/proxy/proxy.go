package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/abac/proxy/internal/log"
	"go.uber.org/zap"
)

type Server struct {
	target      *url.URL
	interceptor Interceptor
	proxy       *httputil.ReverseProxy
}

func NewServer(targetURL string, interceptor Interceptor) (*Server, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	if interceptor == nil {
		interceptor = &PassthroughInterceptor{}
	}

	s := &Server{
		target:      target,
		interceptor: interceptor,
	}

	s.proxy = &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
		},
		ModifyResponse: s.modifyResponse,
	}

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.From(ctx)

	logger.Info("incoming request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
	)

	r = s.interceptor.InterceptRequest(r)
	s.proxy.ServeHTTP(w, r)
}

func (s *Server) modifyResponse(resp *http.Response) error {
	ctx := resp.Request.Context()
	logger := log.From(ctx)

	logger.Info("response received",
		zap.Int("status", resp.StatusCode),
		zap.String("content_type", resp.Header.Get("Content-Type")),
	)

	return s.interceptor.InterceptResponse(resp)
}

func (s *Server) Start(ctx context.Context, addr string, tlsEnabled bool, certFile, keyFile string) error {
	logger := log.From(ctx)
	logger.Info("starting proxy server",
		zap.String("addr", addr),
		zap.String("target", s.target.String()),
		zap.Bool("tls", tlsEnabled),
	)

	server := &http.Server{
		Addr:    addr,
		Handler: s,
	}

	if tlsEnabled {
		return server.ListenAndServeTLS(certFile, keyFile)
	}
	return server.ListenAndServe()
}
