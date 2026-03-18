package policy

import "fmt"

type PolicyGroup struct {
	Version    string   `json:"version"`
	LocalToken string   `json:"localToken"`
	Policies   []Policy `json:"policies"`
}

type Policy struct {
	BaseURL       string       `json:"baseUrl"`
	UpstreamToken string       `json:"upstreamToken,omitempty"`
	Rules         []PolicyRule `json:"rules"`
}

type PolicyRule struct {
	Route          string          `json:"route"`
	Method         string          `json:"method"`
	Action         string          `json:"action"`
	ResponseFilter *ResponseFilter `json:"response_filter,omitempty"`
}

type FilterType string

const (
	FilterTypeInclude FilterType = "include_fields"
	FilterTypeExclude FilterType = "exclude_fields"
)

type ResponseFilter struct {
	Type   FilterType `json:"type"`
	Fields []string   `json:"fields"`
}

func ValidatePolicyGroup(pg *PolicyGroup) error {
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
