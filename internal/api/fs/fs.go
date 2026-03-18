package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/abac/proxy/internal/api"
)

type policyGroupFile struct {
	Version    string       `json:"version"`
	LocalToken string       `json:"localToken"`
	Policies   []policyFile `json:"policies"`
}

type policyFile struct {
	BaseURL               string           `json:"baseUrl"`
	LocalUpstreamTokenKey string           `json:"localUpstreamTokenKey,omitempty"`
	UpstreamTokenType     string           `json:"upstreamTokenType,omitempty"`
	UpstreamHeaderString  string           `json:"upstreamHeaderString,omitempty"`
	Rules                 []api.PolicyRule `json:"rules"`
}

type fileApi struct {
	groups map[string]*api.PolicyGroup
	hosts  []api.HostEntry
}

var _ api.Api = (*fileApi)(nil)

func New(paths []string) (api.Api, error) {
	fa := &fileApi{
		groups: make(map[string]*api.PolicyGroup),
	}

	seenHosts := make(map[string]bool)

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read policy group file %s: %w", path, err)
		}

		var pg policyGroupFile
		if err := json.Unmarshal(raw, &pg); err != nil {
			return nil, fmt.Errorf("failed to parse policy group JSON %s: %w", path, err)
		}

		if err := validatePolicyGroupFile(&pg); err != nil {
			return nil, fmt.Errorf("invalid policy group %s: %w", path, err)
		}

		if _, exists := fa.groups[pg.LocalToken]; exists {
			return nil, fmt.Errorf("duplicate localToken %q in %s", pg.LocalToken, path)
		}

		var policies []api.Policy
		for i, pf := range pg.Policies {
			host, scheme, err := extractHostAndScheme(pf.BaseURL)
			if err != nil {
				return nil, fmt.Errorf("policy group %s: policy[%d]: %w", path, i, err)
			}

			if !seenHosts[host] {
				seenHosts[host] = true
				fa.hosts = append(fa.hosts, api.HostEntry{
					Host:   host,
					Scheme: scheme,
				})
			}

			p := api.Policy{
				BaseURL:              pf.BaseURL,
				UpstreamTokenType:    pf.UpstreamTokenType,
				UpstreamHeaderString: pf.UpstreamHeaderString,
				Rules:                pf.Rules,
			}

			if pf.LocalUpstreamTokenKey != "" {
				val := os.Getenv(pf.LocalUpstreamTokenKey)
				if val == "" {
					return nil, fmt.Errorf("policy group %s: policy[%d]: env var %q is not set or empty", path, i, pf.LocalUpstreamTokenKey)
				}
				p.UpstreamToken = val
			}

			policies = append(policies, p)
		}

		fa.groups[pg.LocalToken] = &api.PolicyGroup{
			Version:  pg.Version,
			Policies: policies,
		}
	}

	return fa, nil
}

func (f *fileApi) GetPolicyData(_ context.Context, token string) (*api.PolicyGroup, error) {
	if data, ok := f.groups[token]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("no policy group found for token")
}

func (f *fileApi) GetAllowedHosts() []api.HostEntry {
	return f.hosts
}

func (f *fileApi) Invalidate(_ string) {}

func (f *fileApi) InvalidateAll() {}

func validatePolicyGroupFile(pg *policyGroupFile) error {
	if pg.Version == "" {
		return fmt.Errorf("version is required")
	}
	if pg.LocalToken == "" {
		return fmt.Errorf("localToken is required")
	}
	if len(pg.Policies) == 0 {
		return fmt.Errorf("at least one policy is required")
	}
	for i, p := range pg.Policies {
		if p.BaseURL == "" {
			return fmt.Errorf("policy[%d]: baseUrl is required", i)
		}
	}
	return nil
}

func extractHostAndScheme(baseURL string) (string, string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid baseUrl %q: %w", baseURL, err)
	}
	if u.Host == "" {
		return "", "", fmt.Errorf("baseUrl %q has no host", baseURL)
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return u.Host, scheme, nil
}
