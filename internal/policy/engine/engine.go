package engine

import (
	"context"

	"github.com/abac/proxy/internal/api"
	"github.com/abac/proxy/internal/policy"
	"github.com/abac/proxy/internal/policy/filter"
	"github.com/abac/proxy/internal/policy/matcher"
)

type Engine interface {
	GetPolicyData(ctx context.Context, token string) (*api.PolicyData, error)
	FindMatchingRule(rules []policy.PolicyRule, path, method string) (*policy.PolicyRule, bool)
	GetDefaultAction(p *policy.Policy) string
	ApplyFilter(data any, f policy.ResponseFilter) (any, error)
}

type engine struct {
	api      api.Api
	matcher  matcher.Matcher
	filterer filter.Filterer
}

// compile-time interface check
var _ Engine = (*engine)(nil)

func New(a api.Api, m matcher.Matcher, f filter.Filterer) Engine {
	return &engine{
		api:      a,
		matcher:  m,
		filterer: f,
	}
}

func (e *engine) GetPolicyData(ctx context.Context, token string) (*api.PolicyData, error) {
	return e.api.GetPolicyData(ctx, token)
}

func (e *engine) FindMatchingRule(rules []policy.PolicyRule, path, method string) (*policy.PolicyRule, bool) {
	for i := range rules {
		rule := &rules[i]
		if e.matcher.MatchesWithMethod(rule.Route, rule.Method, path, method) {
			return rule, true
		}
	}
	return nil, false
}

func (e *engine) GetDefaultAction(p *policy.Policy) string {
	return p.DefaultAction
}

func (e *engine) ApplyFilter(data any, f policy.ResponseFilter) (any, error) {
	return e.filterer.Apply(data, f)
}
