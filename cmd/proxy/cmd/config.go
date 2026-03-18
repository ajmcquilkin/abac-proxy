package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/abac/proxy/internal/api"
	"github.com/abac/proxy/internal/auth"
	"github.com/abac/proxy/internal/db"
	"github.com/abac/proxy/internal/log"
	"github.com/abac/proxy/internal/policy/engine"
	"github.com/abac/proxy/internal/policy/filter"
	"github.com/abac/proxy/internal/policy/matcher"
	"github.com/abac/proxy/internal/proxy"
	"github.com/abac/proxy/internal/proxy/allowlist"
	"github.com/abac/proxy/internal/proxy/interceptor"
	"github.com/spf13/viper"
)

type Config struct {
	Port int `mapstructure:"port"`

	PolicyGroup    []string `mapstructure:"policy_group"`
	PolicyGroupDir string   `mapstructure:"policy_group_dir"`

	PassthroughUnspecified bool   `mapstructure:"passthrough_unspecified"`
	DatabaseURL            string `mapstructure:"database_url"`
}

type RootOptions struct {
	Viper  *viper.Viper
	Config Config
}

func NewRootOptions(v *viper.Viper) *RootOptions {
	return &RootOptions{Viper: v}
}

func (o *RootOptions) Populate() error {
	if o.Viper == nil {
		return fmt.Errorf("viper is required")
	}

	return o.Viper.Unmarshal(&o.Config)
}

func (o *RootOptions) Validate() error {
	if o.Config.Port <= 0 || o.Config.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", o.Config.Port)
	}

	hasFileMode := len(o.Config.PolicyGroup) > 0 || o.Config.PolicyGroupDir != ""
	hasDBMode := o.Config.DatabaseURL != ""

	if hasFileMode && hasDBMode {
		return fmt.Errorf("--database-url is mutually exclusive with --policy-group / --policy-group-dir")
	}

	if !hasFileMode && !hasDBMode {
		return fmt.Errorf("either --policy-group / --policy-group-dir or --database-url is required")
	}

	if o.Config.PolicyGroupDir != "" {
		matches, err := filepath.Glob(filepath.Join(o.Config.PolicyGroupDir, "*.policygroup.json"))
		if err != nil {
			return fmt.Errorf("failed to glob policy group dir: %w", err)
		}
		if len(matches) == 0 && len(o.Config.PolicyGroup) == 0 {
			return fmt.Errorf("no *.policygroup.json files found in %s", o.Config.PolicyGroupDir)
		}
		o.Config.PolicyGroup = append(o.Config.PolicyGroup, matches...)
	}

	return nil
}

func (o *RootOptions) Run(ctx context.Context) error {
	l := log.MustInitService("abac-proxy")
	defer log.Sync(l)

	if o.Config.DatabaseURL != "" {
		return o.runDBMode(ctx)
	}

	return o.runFileMode(ctx)
}

func (o *RootOptions) runFileMode(ctx context.Context) error {
	fa, err := api.NewFileApi(o.Config.PolicyGroup)
	if err != nil {
		return fmt.Errorf("failed to load policy group files: %w", err)
	}

	hosts, err := allowlist.FromEntries(fa.GetAllowedHosts())
	if err != nil {
		return fmt.Errorf("failed to build allowlist from policy groups: %w", err)
	}

	e := engine.New(fa, matcher.New(), filter.New())
	i := interceptor.New(e, o.Config.PassthroughUnspecified)
	srv := proxy.New(hosts, i)

	addr := fmt.Sprintf(":%d", o.Config.Port)
	return srv.Start(ctx, addr)
}

func (o *RootOptions) runDBMode(ctx context.Context) error {
	pool, err := db.NewPool(ctx, o.Config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}
	store := db.NewStore(pool)
	_ = api.NewDBApi(store, 15*time.Second, auth.HashToken, auth.ValidateToken)

	// TODO: DB mode allowlist and startup not yet implemented
	return fmt.Errorf("DB mode startup not yet implemented")
}
