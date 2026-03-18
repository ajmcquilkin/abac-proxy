package interceptor

import "net/http"

type ContextKey string

const (
	ContextKeyTokenValid     ContextKey = "abac_token_valid"
	ContextKeyRequestPath    ContextKey = "abac_request_path"
	ContextKeyRequestMethod  ContextKey = "abac_request_method"
	ContextKeyUpstreamToken  ContextKey = "abac_upstream_token"
	ContextKeyUpstreamType   ContextKey = "abac_upstream_type"
	ContextKeyUpstreamHeader ContextKey = "abac_upstream_header"
	ContextKeyPolicyData     ContextKey = "abac_policy_data"
)

type Interceptor interface {
	InterceptRequest(req *http.Request) *http.Request
	InterceptResponse(resp *http.Response) error
}
