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
			s.setUpstreamAuth(r)
		},
		ModifyResponse: s.modifyResponse,
	}

	return s, nil
}

func (s *Server) setUpstreamAuth(r *httputil.ProxyRequest) {
	ctx := r.In.Context()
	logger := log.From(ctx)

	token, ok := ctx.Value(contextKey("abac_upstream_token")).(string)
	if !ok || token == "" {
		logger.Debugw("no upstream token in context")
		return
	}

	tokenType, _ := ctx.Value(contextKey("abac_upstream_type")).(*string)
	headerString, _ := ctx.Value(contextKey("abac_upstream_header")).(*string)

	// Default to bearer if not specified
	if tokenType == nil {
		bearer := "bearer"
		tokenType = &bearer
	}

	// Remove client's Authorization header to avoid conflicts
	r.Out.Header.Del("Authorization")

	if *tokenType == "custom" && headerString != nil && *headerString != "" {
		// Use custom header
		logger.Infow("setting custom upstream auth header",
			"header", *headerString,
			"token_preview", token[:20]+"...")
		r.Out.Header.Set(*headerString, token)
	} else {
		// Use Authorization: Bearer
		logger.Infow("setting bearer upstream auth",
			"token_preview", token[:20]+"...")
		r.Out.Header.Set("Authorization", "Bearer "+token)
	}
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

func (s *Server) Start(ctx context.Context, addr string) error {
	logger := log.From(ctx)
	logger.Infow("starting proxy server",
		"addr", addr,
		"allowed_hosts", strings.Join(s.allowlist.GetHostList(), ", "),
	)

	server := &http.Server{
		Addr:    addr,
		Handler: s,
	}

	return server.ListenAndServe()
}
