package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/abac/proxy/internal/policy"
)

type FileApi struct {
	data *PolicyData
}

// compile-time interface check
var _ Api = (*FileApi)(nil)

func NewFileApi(policyPath string) (*FileApi, error) {
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var p policy.Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("failed to parse policy JSON: %w", err)
	}

	if err := policy.ValidatePolicy(&p); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	bearerType := "bearer"
	return &FileApi{
		data: &PolicyData{
			Policy:            &p,
			UpstreamToken:     p.User.Token,
			UpstreamTokenType: &bearerType,
		},
	}, nil
}

func (f *FileApi) GetPolicyData(_ context.Context, _ string) (*PolicyData, error) {
	return f.data, nil
}

func (f *FileApi) Invalidate(_ string) {}

func (f *FileApi) InvalidateAll() {}
