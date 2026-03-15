package cmd

import (
	"context"
	"fmt"

	"github.com/abac/proxy/internal/log"
	"github.com/spf13/viper"
)

type Config struct {
	GRPCPort int `mapstructure:"grpcport"`
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
	if o.Config.GRPCPort <= 0 || o.Config.GRPCPort > 65535 {
		return fmt.Errorf("grpcport must be between 1 and 65535, got %d", o.Config.GRPCPort)
	}
	return nil
}

func (o *RootOptions) Run(ctx context.Context) error {
	logger := log.MustInitService("echo")
	defer log.Sync(logger)

	log.From(ctx).Info("hello, world!")

	return nil
}
