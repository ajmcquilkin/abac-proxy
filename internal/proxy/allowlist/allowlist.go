package allowlist

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

type Allowlist interface {
	FindHost(host string) (scheme string, found bool)
	IsAllowed(host string) bool
	GetHostList() []string
}

type allowlist struct {
	AllowedHosts []HostEntry `json:"allowed_hosts"`
}

var _ Allowlist = (*allowlist)(nil)

func New(path string) (Allowlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read allowlist file: %w", err)
	}

	var al allowlist
	if err := json.Unmarshal(data, &al); err != nil {
		return nil, fmt.Errorf("failed to parse allowlist JSON: %w", err)
	}

	if len(al.AllowedHosts) == 0 {
		return nil, fmt.Errorf("allowlist must contain at least one host")
	}

	for i := range al.AllowedHosts {
		if al.AllowedHosts[i].Scheme == "" {
			al.AllowedHosts[i].Scheme = "https"
		}
	}

	return &al, nil
}

func (a *allowlist) FindHost(host string) (string, bool) {
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

func (a *allowlist) IsAllowed(host string) bool {
	_, found := a.FindHost(host)
	return found
}

func (a *allowlist) GetHostList() []string {
	hosts := make([]string, len(a.AllowedHosts))
	for i, entry := range a.AllowedHosts {
		hosts[i] = entry.Scheme + "://" + entry.Host
	}
	return hosts
}
