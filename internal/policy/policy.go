package policy

import "fmt"

type Policy struct {
	Version       string       `json:"version"`
	User          PolicyUser   `json:"user"`
	BaseURL       string       `json:"baseUrl"`
	Rules         []PolicyRule `json:"policies"`
	DefaultAction string       `json:"default_action"`
}

type PolicyUser struct {
	Token string `json:"token"`
	ID    string `json:"id"`
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

func ValidatePolicy(p *Policy) error {
	if p.Version == "" {
		return fmt.Errorf("version is required")
	}
	if p.User.Token == "" {
		return fmt.Errorf("user token is required")
	}
	if p.DefaultAction == "" {
		return fmt.Errorf("default_action is required")
	}
	if p.DefaultAction != "allow" && p.DefaultAction != "deny" {
		return fmt.Errorf("default_action must be 'allow' or 'deny'")
	}
	return nil
}
