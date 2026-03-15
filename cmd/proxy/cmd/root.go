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
	flags.String("target", "", "Target base URL to proxy to")
	flags.Bool("tls", false, "Enable TLS for proxy server")
	flags.String("cert", "", "TLS certificate file path")
	flags.String("key", "", "TLS private key file path")

	_ = v.BindPFlags(flags)
	_ = v.BindEnv("port", "PROXY_PORT")
	_ = v.BindEnv("target", "PROXY_TARGET")
	_ = v.BindEnv("tls", "PROXY_TLS")
	_ = v.BindEnv("cert", "PROXY_CERT")
	_ = v.BindEnv("key", "PROXY_KEY")

	return rootCmd
}

func newConfigViper() *viper.Viper {
	v := viper.New()
	v.SetDefault("port", 8080)
	v.SetDefault("tls", false)
	v.AutomaticEnv()

	return v
}
