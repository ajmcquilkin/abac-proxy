package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRootCommand() *cobra.Command {
	v := newConfigViper()
	opts := NewRootOptions(v)

	rootCmd := &cobra.Command{
		Use:           "abac-proxy",
		Short:         "Run ABAC proxy server",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Populate(); err != nil {
				return err
			}
			if err := opts.Validate(); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	flags := rootCmd.Flags()
	flags.Int("grpcport", 50052, "gRPC listen port")

	_ = v.BindPFlags(flags)
	_ = v.BindEnv("grpcport", "GRPC_PORT")

	return rootCmd
}

func newConfigViper() *viper.Viper {
	v := viper.New()
	v.SetDefault("grpcport", 50052)
	v.AutomaticEnv()
	_ = v.BindEnv("grpcport", "GRPC_PORT")

	return v
}
