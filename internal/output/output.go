package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"oaicheck/internal/checks"
)

func RenderHuman(w io.Writer, env checks.Envelope) error {
	nameWidth := 0
	for _, check := range env.Checks {
		if len(check.Name) > nameWidth {
			nameWidth = len(check.Name)
		}
	}

	for _, check := range env.Checks {
		icon := "✅"
		if !check.OK {
			icon = "❌"
		}

		if _, err := fmt.Fprintf(w, "%s %-*s %s\n", icon, nameWidth, check.Name, check.Message); err != nil {
			return err
		}
		if check.Details != "" {
			if _, err := fmt.Fprintf(w, "  ↳ %s\n", check.Details); err != nil {
				return err
			}
		}
	}

	if env.Command == "doctor" {
		if summary, ok := env.Data.(checks.DoctorData); ok {
			status := "healthy"
			if summary.Failed > 0 {
				status = "needs attention"
			}
			if _, err := fmt.Fprintf(w, "\nSummary: %d passed, %d failed (%s)\n", summary.Passed, summary.Failed, status); err != nil {
				return err
			}
			if env.Verbose {
				if err := renderDoctorVerbose(w, summary); err != nil {
					return err
				}
			}
		}
	}

	if env.Error != nil {
		if _, err := fmt.Fprintf(w, "\nError: %s\n", env.Error.Message); err != nil {
			return err
		}
	}

	return nil
}

func RenderJSON(w io.Writer, env checks.Envelope) error {
	enc := json.NewEncoder(w)
	return enc.Encode(env)
}

func renderDoctorVerbose(w io.Writer, summary checks.DoctorData) error {
	if _, err := fmt.Fprintln(w, "\nVerbose:"); err != nil {
		return err
	}
	if summary.Input != nil {
		if _, err := fmt.Fprintf(w, "  input: base-url=%s model=%q api-key-present=%t\n", summary.Input.BaseURL, summary.Input.Model, summary.Input.APIKeyPresent); err != nil {
			return err
		}
	}
	if summary.Ping != nil {
		if _, err := fmt.Fprintf(w, "  ping output: reachable=%t status=%d\n", summary.Ping.Reachable, summary.Ping.Status); err != nil {
			return err
		}
	}
	if summary.Models != nil {
		if _, err := fmt.Fprintf(w, "  models output: count=%d\n", summary.Models.Count); err != nil {
			return err
		}
		if len(summary.Models.AllIDs) > 0 {
			if _, err := fmt.Fprintln(w, "  models list:"); err != nil {
				return err
			}
			for _, modelID := range summary.Models.AllIDs {
				if _, err := fmt.Fprintf(w, "    - %s\n", modelID); err != nil {
					return err
				}
			}
		}
	}
	if summary.Probe != nil {
		if _, err := fmt.Fprintf(w, "  probe output: via=%q preview=%q\n", summary.Probe.SucceededVia, summary.Probe.Preview); err != nil {
			return err
		}
		if len(summary.Probe.ResponsesRequest) > 0 {
			if _, err := fmt.Fprintf(w, "  probe input (/responses): %s\n", formatJSON(summary.Probe.ResponsesRequest)); err != nil {
				return err
			}
		}
		if len(summary.Probe.ResponsesOutput) > 0 {
			if _, err := fmt.Fprintf(w, "  probe output (/responses): %s\n", formatJSON(summary.Probe.ResponsesOutput)); err != nil {
				return err
			}
		}
		if summary.Probe.ResponsesError != "" {
			if _, err := fmt.Fprintf(w, "  probe error (/responses): %s\n", summary.Probe.ResponsesError); err != nil {
				return err
			}
		}
		if len(summary.Probe.ChatRequest) > 0 {
			if _, err := fmt.Fprintf(w, "  probe input (/chat/completions): %s\n", formatJSON(summary.Probe.ChatRequest)); err != nil {
				return err
			}
		}
		if len(summary.Probe.ChatOutput) > 0 {
			if _, err := fmt.Fprintf(w, "  probe output (/chat/completions): %s\n", formatJSON(summary.Probe.ChatOutput)); err != nil {
				return err
			}
		}
		if summary.Probe.ChatError != "" {
			if _, err := fmt.Fprintf(w, "  probe error (/chat/completions): %s\n", summary.Probe.ChatError); err != nil {
				return err
			}
		}
	}
	return nil
}

func formatJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "(unavailable)"
	}
	return strings.TrimSpace(string(b))
}
