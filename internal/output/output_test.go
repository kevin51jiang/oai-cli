package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"oaicheck/internal/checks"
	"oaicheck/internal/config"
)

func TestRenderJSON(t *testing.T) {
	env := checks.Envelope{
		OK:      true,
		Command: "ping",
		Config:  config.SafeView{BaseURL: "https://api.openai.com/v1", APIKeyPresent: true, Model: "gpt-4.1-mini"},
		Checks:  []checks.CheckResult{{Name: checks.CheckPing, OK: true, Message: "reachable"}},
		Data:    checks.PingData{Reachable: true, Status: 401},
		Error:   nil,
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, env); err != nil {
		t.Fatalf("render json: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal rendered json: %v", err)
	}
	if decoded["command"] != "ping" {
		t.Fatalf("expected command ping, got %#v", decoded["command"])
	}
}

func TestRenderHumanDoctorSuccess(t *testing.T) {
	env := checks.Envelope{
		OK:      true,
		Command: "doctor",
		Checks: []checks.CheckResult{
			{Name: checks.CheckPing, OK: true, Message: "reachable", Details: "HTTP 401 from https://api.openai.com/v1/models"},
			{Name: checks.CheckModels, OK: true, Message: "retrieved 50 model(s)"},
			{Name: checks.CheckProbe, OK: true, Message: "probe succeeded"},
		},
		Data:  checks.DoctorData{Passed: 3, Failed: 0},
		Error: nil,
	}

	var buf bytes.Buffer
	if err := RenderHuman(&buf, env); err != nil {
		t.Fatalf("render human: %v", err)
	}

	got := buf.String()
	wantLines := []string{
		"✅ ping   reachable",
		"  ↳ HTTP 401 from https://api.openai.com/v1/models",
		"✅ models retrieved 50 model(s)",
		"✅ probe  probe succeeded",
		"",
		"Summary: 3 passed, 0 failed (healthy)",
	}
	want := strings.Join(wantLines, "\n") + "\n"
	if got != want {
		t.Fatalf("unexpected output\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestRenderHumanDoctorFailure(t *testing.T) {
	env := checks.Envelope{
		OK:      false,
		Command: "doctor",
		Checks: []checks.CheckResult{
			{Name: checks.CheckPing, OK: true, Message: "reachable"},
			{Name: checks.CheckModels, OK: false, Message: "missing API key (use --api-key or OPENAI_API_KEY)"},
			{Name: checks.CheckProbe, OK: false, Message: "missing model (use --model or OPENAI_MODEL)"},
		},
		Data:  checks.DoctorData{Passed: 1, Failed: 2},
		Error: &checks.ErrorPayload{Message: "missing API key (use --api-key or OPENAI_API_KEY)"},
	}

	var buf bytes.Buffer
	if err := RenderHuman(&buf, env); err != nil {
		t.Fatalf("render human: %v", err)
	}

	got := buf.String()
	wantLines := []string{
		"✅ ping   reachable",
		"❌ models missing API key (use --api-key or OPENAI_API_KEY)",
		"❌ probe  missing model (use --model or OPENAI_MODEL)",
		"",
		"Summary: 1 passed, 2 failed (needs attention)",
		"",
		"Error: missing API key (use --api-key or OPENAI_API_KEY)",
	}
	want := strings.Join(wantLines, "\n") + "\n"
	if got != want {
		t.Fatalf("unexpected output\nwant:\n%s\ngot:\n%s", want, got)
	}
}
