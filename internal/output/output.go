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

	if strings.HasPrefix(env.Command, "doctor") {
		if summary, ok := env.Data.(checks.DoctorData); ok {
			status := "healthy"
			if summary.Failed > 0 {
				status = "needs attention"
			}
			if _, err := fmt.Fprintf(w, "\nSummary: %d passed, %d failed (%s)\n", summary.Passed, summary.Failed, status); err != nil {
				return err
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
