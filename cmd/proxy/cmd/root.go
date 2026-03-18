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
	flags.StringArray("policy-group", nil, "Path to *.policygroup.json file (repeatable)")
	flags.String("policy-group-dir", "", "Directory containing *.policygroup.json files")
	flags.Bool("passthrough-unspecified", false, "Allow unmatched routes instead of denying")
	flags.String("database-url", "", "Database connection URL (mutually exclusive with --policy-group)")

	_ = v.BindPFlags(flags)
	_ = v.BindPFlag("policy_group", flags.Lookup("policy-group"))
	_ = v.BindPFlag("policy_group_dir", flags.Lookup("policy-group-dir"))
	_ = v.BindPFlag("passthrough_unspecified", flags.Lookup("passthrough-unspecified"))
	_ = v.BindPFlag("database_url", flags.Lookup("database-url"))
	_ = v.BindEnv("port", "PROXY_PORT")
	_ = v.BindEnv("database_url", "DATABASE_URL")

	return rootCmd
}

func newConfigViper() *viper.Viper {
	v := viper.New()
	v.SetDefault("port", 8080)
	v.AutomaticEnv()

	return v
}
