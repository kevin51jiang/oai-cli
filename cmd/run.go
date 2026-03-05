package cmd

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"oaicheck/internal/checks"
	"oaicheck/internal/config"
	"oaicheck/internal/output"
)

func runDoctor(cmd *cobra.Command, opts *Options) error {
	cfg := config.Resolve(opts.BaseURL, opts.APIKey, opts.Model)
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	results, data := checks.RunDoctor(ctx, cfg, opts.Verbose)
	env := checks.BuildEnvelope("doctor", cfg, results, data, opts.Verbose)
	return renderAndExit(env, opts.JSON)
}

func runPing(cmd *cobra.Command, opts *Options) error {
	cfg := config.Resolve(opts.BaseURL, opts.APIKey, opts.Model)
	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	result, data := checks.RunPing(ctx, cfg)
	env := checks.BuildEnvelope("ping", cfg, []checks.CheckResult{result}, data, opts.Verbose)
	return renderAndExit(env, opts.JSON)
}

func runModels(cmd *cobra.Command, opts *Options) error {
	cfg := config.Resolve(opts.BaseURL, opts.APIKey, opts.Model)
	ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
	defer cancel()

	result, data := checks.RunModels(ctx, cfg)
	env := checks.BuildEnvelope("models", cfg, []checks.CheckResult{result}, data, opts.Verbose)
	return renderAndExit(env, opts.JSON)
}

func runProbe(cmd *cobra.Command, opts *Options) error {
	cfg := config.Resolve(opts.BaseURL, opts.APIKey, opts.Model)
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	result, data := checks.RunProbe(ctx, cfg)
	env := checks.BuildEnvelope("probe", cfg, []checks.CheckResult{result}, data, opts.Verbose)
	return renderAndExit(env, opts.JSON)
}

func renderAndExit(env checks.Envelope, asJSON bool) error {
	var err error
	if asJSON {
		err = output.RenderJSON(os.Stdout, env)
	} else {
		err = output.RenderHuman(os.Stdout, env)
	}
	if err != nil {
		return err
	}
	if !env.OK {
		return ErrCheckFailed
	}
	return nil
}
