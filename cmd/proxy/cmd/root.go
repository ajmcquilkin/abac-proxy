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
	flags.Int("port", 8080, "HTTP proxy listen port")
	flags.String("allowlist", "", "Path to allowlist JSON file")
	flags.String("policy", "", "Path to ABAC policy JSON file")

	_ = v.BindPFlags(flags)
	_ = v.BindEnv("port", "PROXY_PORT")
	_ = v.BindEnv("allowlist", "PROXY_ALLOWLIST")
	_ = v.BindEnv("policy", "PROXY_POLICY")

	return rootCmd
}

func newConfigViper() *viper.Viper {
	v := viper.New()
	v.SetDefault("port", 8080)
	v.AutomaticEnv()

	return v
}
