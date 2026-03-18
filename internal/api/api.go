package api

import (
	"context"

	"github.com/abac/proxy/internal/policy"
)

type PolicyGroupData struct {
	Policies             []policy.Policy
	DefaultAction        string
	UpstreamTokenType    *string
	UpstreamHeaderString *string
}

type PolicyData struct {
	Policy               *policy.Policy
	DefaultAction        string
	UpstreamToken        string
	UpstreamTokenType    *string
	UpstreamHeaderString *string
}

type Api interface {
	GetPolicyData(ctx context.Context, token string) (*PolicyGroupData, error)
	Invalidate(token string)
	InvalidateAll()
}
