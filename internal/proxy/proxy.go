package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/abac/proxy/internal/log"
	"github.com/abac/proxy/internal/proxy/allowlist"
	"github.com/abac/proxy/internal/proxy/interceptor"
)

type Server struct {
	hosts       allowlist.Allowlist
	interceptor interceptor.Interceptor
	proxy       *httputil.ReverseProxy
}

func New(hosts allowlist.Allowlist, i interceptor.Interceptor) *Server {
	if hosts == nil {
		panic("proxy: hosts must not be nil")
	}
	if i == nil {
		panic("proxy: interceptor must not be nil")
	}

	s := &Server{
		hosts:       hosts,
		interceptor: i,
	}

	s.proxy = &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetXForwarded()
			s.setUpstreamAuth(r)
		},
		ModifyResponse: s.modifyResponse,
	}

	return s
}

func (s *Server) setUpstreamAuth(r *httputil.ProxyRequest) {
	ctx := r.In.Context()
	logger := log.From(ctx)

	token, ok := ctx.Value(interceptor.ContextKeyUpstreamToken).(string)
	if !ok || token == "" {
		logger.Debugw("no upstream token in context")
		return
	}

	tokenType, _ := ctx.Value(interceptor.ContextKeyUpstreamType).(string)
	headerString, _ := ctx.Value(interceptor.ContextKeyUpstreamHeader).(string)

	r.Out.Header.Del("Authorization")

	if tokenType == "custom" && headerString != "" {
		logger.Infow("setting custom upstream auth header",
			"header", headerString)
		r.Out.Header.Set(headerString, token)
	} else {
		logger.Infow("setting bearer upstream auth")
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

	scheme, allowed := s.hosts.FindHost(r.Host)
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
		"allowed_hosts", strings.Join(s.hosts.GetHostList(), ", "),
	)

	server := &http.Server{
		Addr:    addr,
		Handler: s,
	}

	return server.ListenAndServe()
}
