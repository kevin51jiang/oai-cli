package checks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"oaicheck/internal/config"
)

func TestBuildEnvelopeOK(t *testing.T) {
	cfg := config.Resolved{BaseURL: "https://api.openai.com/v1", APIKey: "x", Model: "gpt-4.1-mini"}
	results := []CheckResult{{Name: CheckPing, OK: true, Message: "reachable"}}
	env := BuildEnvelope("ping", cfg, results, PingData{Reachable: true, Status: 401})

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
	env := BuildEnvelope("models", cfg, results, ModelsData{})

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
