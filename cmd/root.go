package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var ErrCheckFailed = errors.New("check failed")

type Options struct {
	BaseURL string
	APIKey  string
	Model   string
	JSON    bool
}

func NewRootCmd() *cobra.Command {
	opts := &Options{}

	rootCmd := &cobra.Command{
		Use:           "oaicheck",
		Short:         "Debug OpenAI API configuration",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVar(&opts.BaseURL, "base-url", "", "OpenAI base URL (or OPENAI_BASE_URL)")
	rootCmd.PersistentFlags().StringVar(&opts.APIKey, "api-key", "", "OpenAI API key (or OPENAI_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&opts.Model, "model", "", "OpenAI model (or OPENAI_MODEL)")
	rootCmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "Output machine-readable JSON")

	rootCmd.AddCommand(newDoctorCmd(opts))
	rootCmd.AddCommand(newPingCmd(opts))
	rootCmd.AddCommand(newModelsCmd(opts))
	rootCmd.AddCommand(newProbeCmd(opts))

	return rootCmd
}

func newDoctorCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run all checks and show summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd, opts)
		},
	}
}

func newPingCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check if the configured server is reachable",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPing(cmd, opts)
		},
	}
}

func newModelsCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List models to verify API key and base URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModels(cmd, opts)
		},
	}
}

func newProbeCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "probe",
		Short: "Run a tiny generation request",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProbe(cmd, opts)
		},
	}
}
