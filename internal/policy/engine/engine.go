package engine

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/abac/proxy/internal/api"
	"github.com/abac/proxy/internal/policy"
	"github.com/abac/proxy/internal/policy/filter"
	"github.com/abac/proxy/internal/policy/matcher"
)

type Engine interface {
	GetPolicyData(ctx context.Context, token, host string) (*api.PolicyData, error)
	FindMatchingRule(rules []policy.PolicyRule, path, method string) (*policy.PolicyRule, bool)
	ApplyFilter(data any, f policy.ResponseFilter) (any, error)
}

type engine struct {
	api      api.Api
	matcher  matcher.Matcher
	filterer filter.Filterer
}

var _ Engine = (*engine)(nil)

func New(a api.Api, m matcher.Matcher, f filter.Filterer) Engine {
	return &engine{
		api:      a,
		matcher:  m,
		filterer: f,
	}
}

func (e *engine) GetPolicyData(ctx context.Context, token, host string) (*api.PolicyData, error) {
	groupData, err := e.api.GetPolicyData(ctx, token)
	if err != nil {
		return nil, err
	}

	host = strings.ToLower(strings.TrimSpace(host))

	for i := range groupData.Policies {
		p := &groupData.Policies[i]
		policyHost, err := extractHost(p.BaseURL)
		if err != nil {
			continue
		}
		if strings.ToLower(policyHost) == host {
			return &api.PolicyData{
				Policy:               p,
				DefaultAction:        groupData.DefaultAction,
				UpstreamToken:        p.UpstreamToken,
				UpstreamTokenType:    groupData.UpstreamTokenType,
				UpstreamHeaderString: groupData.UpstreamHeaderString,
			}, nil
		}
	}

	return nil, fmt.Errorf("no policy found for host %q", host)
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

func (e *engine) ApplyFilter(data any, f policy.ResponseFilter) (any, error) {
	return e.filterer.Apply(data, f)
}

func extractHost(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", fmt.Errorf("no host in URL %q", baseURL)
	}
	return u.Host, nil
}
