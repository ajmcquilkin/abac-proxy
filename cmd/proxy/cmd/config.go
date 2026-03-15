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
	TLS       bool   `mapstructure:"tls"`
	Cert      string `mapstructure:"cert"`
	Key       string `mapstructure:"key"`
	Policy    string `mapstructure:"policy"`
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
	if o.Config.Policy == "" {
		return fmt.Errorf("policy file is required")
	}
	if o.Config.TLS {
		if o.Config.Cert == "" {
			return fmt.Errorf("cert file required when TLS is enabled")
		}
		if o.Config.Key == "" {
			return fmt.Errorf("key file required when TLS is enabled")
		}
	}
	return nil
}

func (o *RootOptions) Run(ctx context.Context) error {
	logger := log.MustInitService("abac-proxy")
	defer log.Sync(logger)

	interceptor, err := proxy.NewABACInterceptor(o.Config.Policy)
	if err != nil {
		return fmt.Errorf("failed to create ABAC interceptor: %w", err)
	}

	srv, err := proxy.NewServer(o.Config.Allowlist, interceptor)
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	addr := fmt.Sprintf(":%d", o.Config.Port)
	return srv.Start(ctx, addr, o.Config.TLS, o.Config.Cert, o.Config.Key)
}
