package checks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"oaicheck/internal/config"
)

func TestBuildEnvelopeOK(t *testing.T) {
	cfg := config.Resolved{BaseURL: "https://api.openai.com/v1", APIKey: "x", Model: "gpt-4.1-mini"}
	results := []CheckResult{{Name: CheckPing, OK: true, Message: "reachable"}}
	env := BuildEnvelope("ping", cfg, results, PingData{Reachable: true, Status: 401}, false)

	if !env.OK {
		t.Fatal("expected OK envelope")
	}
	if env.Error != nil {
		t.Fatal("expected nil error payload")
	}
	if !env.Config.APIKeyPresent {
		t.Fatal("expected redacted config to indicate apiKeyPresent")
	}
}

func TestBuildEnvelopeFailure(t *testing.T) {
	cfg := config.Resolved{BaseURL: "https://api.openai.com/v1"}
	results := []CheckResult{{Name: CheckModels, OK: false, Message: "missing API key"}}
	env := BuildEnvelope("models", cfg, results, ModelsData{}, false)

	if env.OK {
		t.Fatal("expected failing envelope")
	}
	if env.Error == nil {
		t.Fatal("expected error payload")
	}
	if env.Error.Message != "missing API key" {
		t.Fatalf("unexpected error message: %q", env.Error.Message)
	}
}

func TestRunProbeFailureIncludesEndpointErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/responses":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"responses disabled"}`))
		case "/chat/completions":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid model for chat"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := config.Resolved{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
	}

	result, _ := RunProbe(t.Context(), cfg)
	if result.OK {
		t.Fatal("expected probe failure")
	}
	if !strings.Contains(result.Message, "/responses: status 404") {
		t.Fatalf("expected /responses failure in message, got %q", result.Message)
	}
	if !strings.Contains(result.Message, "responses disabled") {
		t.Fatalf("expected /responses body snippet in message, got %q", result.Message)
	}
	if !strings.Contains(result.Message, "/chat/completions: status 400") {
		t.Fatalf("expected /chat/completions failure in message, got %q", result.Message)
	}
	if !strings.Contains(result.Message, "invalid model for chat") {
		t.Fatalf("expected /chat/completions body snippet in message, got %q", result.Message)
	}
}

func TestRunDoctorVerboseIncludesInputAndFullModelList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1"},{"id":"gpt-4.1-mini"},{"id":"o4-mini"}]}`))
		case "/responses":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"bad payload"}`))
				return
			}
			_, _ = w.Write([]byte(`{"output_text":"pong"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := config.Resolved{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-4.1-mini",
	}

	_, data := RunDoctor(t.Context(), cfg, true)
	if data.Input == nil {
		t.Fatal("expected verbose input")
	}
	if data.Models == nil || len(data.Models.AllIDs) != 3 {
		t.Fatalf("expected full models list in verbose output, got %#v", data.Models)
	}
	if data.Probe == nil {
		t.Fatal("expected verbose probe data")
	}
	if got := data.Probe.ResponsesRequest["model"]; got != "gpt-4.1-mini" {
		t.Fatalf("expected probe request model, got %#v", got)
	}
	if got := data.Probe.ResponsesOutput["output_text"]; got != "pong" {
		t.Fatalf("expected probe output text, got %#v", got)
	}
}

func TestRunDoctorNonVerboseOmitsVerboseData(t *testing.T) {
	cfg := config.Resolved{
		BaseURL: "https://api.openai.com/v1",
	}
	_, data := RunDoctor(t.Context(), cfg, false)
	if data.Input != nil || data.Models != nil || data.Probe != nil || data.Ping != nil {
		t.Fatalf("expected no verbose details, got %#v", data)
	}
}
