package cmd

import (
	"context"
	"fmt"

	"github.com/abac/proxy/internal/log"
	"github.com/abac/proxy/internal/proxy"
	"github.com/spf13/viper"
)

type Config struct {
	Port      int    `mapstructure:"port"`
	Allowlist string `mapstructure:"allowlist"`
	Policy    string `mapstructure:"policy"`

	// Database settings
	DatabaseURL     string `mapstructure:"database_url"`
	PolicyStoreType string `mapstructure:"policy_store_type"` // "file" or "db"

	// Redis settings (future)
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

	// Default to file-based policy store
	if o.Config.PolicyStoreType == "" {
		o.Config.PolicyStoreType = "file"
	}

	// Validate based on policy store type
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

	var interceptor proxy.Interceptor
	var err error

	if o.Config.PolicyStoreType == "db" {
		// Database-based policy loading
		interceptor, err = proxy.NewABACInterceptorFromDB(ctx, o.Config.DatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to create DB-based ABAC interceptor: %w", err)
		}
	} else {
		// File-based policy loading (default)
		interceptor, err = proxy.NewABACInterceptor(o.Config.Policy)
		if err != nil {
			return fmt.Errorf("failed to create file-based ABAC interceptor: %w", err)
		}
	}

	srv, err := proxy.NewServer(o.Config.Allowlist, interceptor)
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	addr := fmt.Sprintf(":%d", o.Config.Port)
	return srv.Start(ctx, addr)
}
