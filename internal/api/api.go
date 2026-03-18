package api

import "context"

type FilterType string

const (
	FilterTypeInclude FilterType = "include_fields"
	FilterTypeExclude FilterType = "exclude_fields"
)

type ResponseFilter struct {
	Type   FilterType `json:"type"`
	Fields []string   `json:"fields"`
}

type PolicyRule struct {
	Route          string          `json:"route"`
	Method         string          `json:"method"`
	Action         string          `json:"action"`
	ResponseFilter *ResponseFilter `json:"response_filter,omitempty"`
}

type Policy struct {
	BaseURL              string
	UpstreamToken        string
	UpstreamTokenType    string
	UpstreamHeaderString string
	Rules                []PolicyRule
}

type HostEntry struct {
	Host   string `json:"host"`
	Scheme string `json:"scheme"`
}

type PolicyGroup struct {
	Version       string
	Policies      []Policy
	DefaultAction string
}

// PolicyData is the resolved per-request view: a single Policy selected by host,
// plus group-level fields (DefaultAction) needed for the interceptor decision.
type PolicyData struct {
	Policy        *Policy
	DefaultAction string
}

type Api interface {
	GetPolicyData(ctx context.Context, token string) (*PolicyGroup, error)
	GetAllowedHosts() []HostEntry
	Invalidate(token string)
	InvalidateAll()
}
