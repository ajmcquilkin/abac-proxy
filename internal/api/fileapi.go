package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/abac/proxy/internal/policy"
	"github.com/abac/proxy/internal/proxy/allowlist"
)

type FileApi struct {
	groups map[string]*PolicyGroupData
	hosts  []allowlist.HostEntry
}

var _ Api = (*FileApi)(nil)

func NewFileApi(paths []string) (*FileApi, error) {
	fa := &FileApi{
		groups: make(map[string]*PolicyGroupData),
	}

	seenHosts := make(map[string]bool)

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read policy group file %s: %w", path, err)
		}

		var pg policy.PolicyGroup
		if err := json.Unmarshal(raw, &pg); err != nil {
			return nil, fmt.Errorf("failed to parse policy group JSON %s: %w", path, err)
		}

		if err := policy.ValidatePolicyGroup(&pg); err != nil {
			return nil, fmt.Errorf("invalid policy group %s: %w", path, err)
		}

		if _, exists := fa.groups[pg.LocalToken]; exists {
			return nil, fmt.Errorf("duplicate localToken %q in %s", pg.LocalToken, path)
		}

		for i := range pg.Policies {
			host, scheme, err := extractHostAndScheme(pg.Policies[i].BaseURL)
			if err != nil {
				return nil, fmt.Errorf("policy group %s: policy[%d]: %w", path, i, err)
			}

			if !seenHosts[host] {
				seenHosts[host] = true
				fa.hosts = append(fa.hosts, allowlist.HostEntry{
					Host:   host,
					Scheme: scheme,
				})
			}
		}

		fa.groups[pg.LocalToken] = &PolicyGroupData{
			Policies: pg.Policies,
		}
	}

	return fa, nil
}

func (f *FileApi) GetPolicyData(_ context.Context, token string) (*PolicyGroupData, error) {
	if data, ok := f.groups[token]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("no policy group found for token")
}

func (f *FileApi) GetAllowedHosts() []allowlist.HostEntry {
	return f.hosts
}

func (f *FileApi) Invalidate(_ string) {}

func (f *FileApi) InvalidateAll() {}

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
