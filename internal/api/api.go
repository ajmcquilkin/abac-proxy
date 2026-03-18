package api

import (
	"context"

	"github.com/abac/proxy/internal/policy"
)

type PolicyData struct {
	Policy               *policy.Policy
	UpstreamToken        string
	UpstreamTokenType    *string
	UpstreamHeaderString *string
}

type Api interface {
	GetPolicyData(ctx context.Context, token string) (*PolicyData, error)
	Invalidate(token string)
	InvalidateAll()
}
