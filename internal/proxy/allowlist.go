package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type HostEntry struct {
	Host   string `json:"host"`
	Scheme string `json:"scheme"`
}

type Allowlist struct {
	AllowedHosts []HostEntry `json:"allowed_hosts"`
}

func LoadAllowlist(path string) (*Allowlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read allowlist file: %w", err)
	}

	var allowlist Allowlist
	if err := json.Unmarshal(data, &allowlist); err != nil {
		return nil, fmt.Errorf("failed to parse allowlist JSON: %w", err)
	}

	if len(allowlist.AllowedHosts) == 0 {
		return nil, fmt.Errorf("allowlist must contain at least one host")
	}

	for i := range allowlist.AllowedHosts {
		if allowlist.AllowedHosts[i].Scheme == "" {
			allowlist.AllowedHosts[i].Scheme = "https"
		}
	}

	return &allowlist, nil
}

func (a *Allowlist) FindHost(host string) (string, bool) {
	host = strings.ToLower(strings.TrimSpace(host))

	for _, entry := range a.AllowedHosts {
		allowedHost := strings.ToLower(strings.TrimSpace(entry.Host))

		if strings.HasPrefix(allowedHost, "*.") {
			suffix := allowedHost[2:]
			if host == suffix[1:] || strings.HasSuffix(host, suffix) {
				return entry.Scheme, true
			}
		} else if host == allowedHost {
			return entry.Scheme, true
		}
	}

	return "", false
}

func (a *Allowlist) IsAllowed(host string) bool {
	_, found := a.FindHost(host)
	return found
}

func (a *Allowlist) GetHostList() []string {
	hosts := make([]string, len(a.AllowedHosts))
	for i, entry := range a.AllowedHosts {
		hosts[i] = entry.Scheme + "://" + entry.Host
	}
	return hosts
}
