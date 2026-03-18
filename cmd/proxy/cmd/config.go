package cmd

import (
	"context"
	"fmt"
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
	Port      int    `mapstructure:"port"`
	Allowlist string `mapstructure:"allowlist"`
	Policy    string `mapstructure:"policy"`

	DatabaseURL     string `mapstructure:"database_url"`
	PolicyStoreType string `mapstructure:"policy_store_type"`

	RedisURL             string `mapstructure:"redis_url"`
	PolicyCacheTTL       int    `mapstructure:"policy_cache_ttl"`
	PolicyReloadInterval int    `mapstructure:"policy_reload_interval"`
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
	if o.Config.Allowlist == "" {
		return fmt.Errorf("allowlist file is required")
	}

	if o.Config.PolicyStoreType == "" {
		o.Config.PolicyStoreType = "file"
	}

	if o.Config.PolicyStoreType == "file" {
		if o.Config.Policy == "" {
			return fmt.Errorf("policy file is required when policy_store_type is 'file'")
		}
	} else if o.Config.PolicyStoreType == "db" {
		if o.Config.DatabaseURL == "" {
			return fmt.Errorf("database_url is required when policy_store_type is 'db'")
		}
	} else {
		return fmt.Errorf("policy_store_type must be 'file' or 'db', got '%s'", o.Config.PolicyStoreType)
	}

	return nil
}

func (o *RootOptions) Run(ctx context.Context) error {
	logger := log.MustInitService("abac-proxy")
	defer log.Sync(logger)

	var a api.Api
	if o.Config.PolicyStoreType == "db" {
		pool, err := db.NewPool(ctx, o.Config.DatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to create database pool: %w", err)
		}
		store := db.NewStore(pool)
		a = api.NewDBApi(store, 15*time.Second, auth.HashToken, auth.ValidateToken)
	} else {
		var err error
		a, err = api.NewFileApi(o.Config.Policy)
		if err != nil {
			return fmt.Errorf("failed to load policy file: %w", err)
		}
	}

	e := engine.New(a, matcher.New(), filter.New())
	i := interceptor.New(e)

	hosts, err := allowlist.New(o.Config.Allowlist)
	if err != nil {
		return fmt.Errorf("failed to load allowlist: %w", err)
	}

	srv := proxy.New(hosts, i)

	addr := fmt.Sprintf(":%d", o.Config.Port)
	return srv.Start(ctx, addr)
}
