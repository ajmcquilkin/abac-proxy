package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/abac/proxy/internal/log"
)

type Server struct {
	allowlist   *Allowlist
	interceptor Interceptor
	proxy       *httputil.ReverseProxy
}

func NewServer(allowlistPath string, interceptor Interceptor) (*Server, error) {
	allowlist, err := LoadAllowlist(allowlistPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load allowlist: %w", err)
	}

	if interceptor == nil {
		interceptor = &PassthroughInterceptor{}
	}

	s := &Server{
		allowlist:   allowlist,
		interceptor: interceptor,
	}

	s.proxy = &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
		},
		ModifyResponse: s.modifyResponse,
	}

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.From(ctx)

	logger.Infow("incoming request",
		"method", r.Method,
		"host", r.Host,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	scheme, allowed := s.allowlist.FindHost(r.Host)
	if !allowed {
		logger.Warnw("host not in allowlist",
			"host", r.Host,
		)
		http.Error(w, `{"error":"host not allowed"}`, http.StatusForbidden)
		return
	}

	r.URL.Scheme = scheme
	r.URL.Host = r.Host

	r = s.interceptor.InterceptRequest(r)
	s.proxy.ServeHTTP(w, r)
}

func (s *Server) modifyResponse(resp *http.Response) error {
	ctx := resp.Request.Context()
	logger := log.From(ctx)

	logger.Infow("response received",
		"status", resp.StatusCode,
		"content_type", resp.Header.Get("Content-Type"),
	)

	return s.interceptor.InterceptResponse(resp)
}

func (s *Server) Start(ctx context.Context, addr string, tlsEnabled bool, certFile, keyFile string) error {
	logger := log.From(ctx)
	logger.Infow("starting proxy server",
		"addr", addr,
		"allowed_hosts", strings.Join(s.allowlist.GetHostList(), ", "),
		"tls", tlsEnabled,
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
