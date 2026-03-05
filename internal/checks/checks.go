package checks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"oaicheck/internal/config"
)

const (
	CheckPing   = "ping"
	CheckModels = "models"
	CheckProbe  = "probe"
)

type CheckResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type ErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type Envelope struct {
	OK      bool            `json:"ok"`
	Command string          `json:"command"`
	Verbose bool            `json:"verbose,omitempty"`
	Config  config.SafeView `json:"config"`
	Checks  []CheckResult   `json:"checks"`
	Data    any             `json:"data"`
	Error   *ErrorPayload   `json:"error"`
}

type PingData struct {
	Reachable bool `json:"reachable"`
	Status    int  `json:"status,omitempty"`
}

type ModelsData struct {
	Count              int      `json:"count"`
	SampleIDs          []string `json:"sampleIds"`
	AllIDs             []string `json:"allIds,omitempty"`
	SelectedModelFound *bool    `json:"selectedModelFound,omitempty"`
}

type ProbeData struct {
	SucceededVia     string         `json:"succeededVia,omitempty"`
	Preview          string         `json:"preview,omitempty"`
	ResponsesRequest map[string]any `json:"responsesRequest,omitempty"`
	ResponsesOutput  map[string]any `json:"responsesOutput,omitempty"`
	ResponsesError   string         `json:"responsesError,omitempty"`
	ChatRequest      map[string]any `json:"chatRequest,omitempty"`
	ChatOutput       map[string]any `json:"chatOutput,omitempty"`
	ChatError        string         `json:"chatError,omitempty"`
}

type DoctorData struct {
	Passed int              `json:"passed"`
	Failed int              `json:"failed"`
	Input  *config.SafeView `json:"input,omitempty"`
	Ping   *PingData        `json:"ping,omitempty"`
	Models *ModelsData      `json:"models,omitempty"`
	Probe  *ProbeData       `json:"probe,omitempty"`
}

func RunPing(ctx context.Context, cfg config.Resolved) (CheckResult, PingData) {
	target := strings.TrimRight(cfg.BaseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		msg := fmt.Sprintf("invalid base URL: %v", err)
		return CheckResult{Name: CheckPing, OK: false, Message: msg}, PingData{Reachable: false}
	}

	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		msg := fmt.Sprintf("unreachable: %v", err)
		return CheckResult{Name: CheckPing, OK: false, Message: msg}, PingData{Reachable: false}
	}
	defer resp.Body.Close()

	details := fmt.Sprintf("HTTP %d from %s", resp.StatusCode, target)
	return CheckResult{Name: CheckPing, OK: true, Message: "reachable", Details: details}, PingData{Reachable: true, Status: resp.StatusCode}
}

func RunModels(ctx context.Context, cfg config.Resolved) (CheckResult, ModelsData) {
	return runModels(ctx, cfg, false)
}

func runModels(ctx context.Context, cfg config.Resolved, verbose bool) (CheckResult, ModelsData) {
	if cfg.APIKey == "" {
		msg := "missing API key (use --api-key or OPENAI_API_KEY)"
		return CheckResult{Name: CheckModels, OK: false, Message: msg}, ModelsData{}
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/models"
	if err := doJSONRequest(ctx, http.MethodGet, url, cfg.APIKey, nil, &payload); err != nil {
		msg := fmt.Sprintf("models request failed: %s", err)
		return CheckResult{Name: CheckModels, OK: false, Message: msg}, ModelsData{}
	}

	sample := make([]string, 0, 5)
	for i, m := range payload.Data {
		if i >= 5 {
			break
		}
		sample = append(sample, m.ID)
	}

	data := ModelsData{Count: len(payload.Data), SampleIDs: sample}
	if verbose {
		all := make([]string, 0, len(payload.Data))
		for _, m := range payload.Data {
			all = append(all, m.ID)
		}
		data.AllIDs = all
	}
	if cfg.Model != "" {
		found := false
		for _, m := range payload.Data {
			if m.ID == cfg.Model {
				found = true
				break
			}
		}
		data.SelectedModelFound = &found
	}

	msg := fmt.Sprintf("retrieved %d model(s)", data.Count)
	if data.SelectedModelFound != nil && !*data.SelectedModelFound {
		msg += "; selected model not present"
	}
	return CheckResult{Name: CheckModels, OK: true, Message: msg}, data
}

func RunProbe(ctx context.Context, cfg config.Resolved) (CheckResult, ProbeData) {
	return runProbe(ctx, cfg, false)
}

func runProbe(ctx context.Context, cfg config.Resolved, verbose bool) (CheckResult, ProbeData) {
	if cfg.APIKey == "" {
		msg := "missing API key (use --api-key or OPENAI_API_KEY)"
		return CheckResult{Name: CheckProbe, OK: false, Message: msg}, ProbeData{}
	}
	if cfg.Model == "" {
		msg := "missing model (use --model or OPENAI_MODEL)"
		return CheckResult{Name: CheckProbe, OK: false, Message: msg}, ProbeData{}
	}

	responsesPayload := map[string]any{
		"model": cfg.Model,
		"input": "Reply with exactly: pong",
	}
	data := ProbeData{}
	if verbose {
		data.ResponsesRequest = cloneMap(responsesPayload)
	}
	var responsesBody map[string]any
	responsesURL := strings.TrimRight(cfg.BaseURL, "/") + "/responses"
	responsesCtx, cancelResponses := context.WithTimeout(ctx, 30*time.Second)
	defer cancelResponses()
	responsesErr := doJSONRequest(responsesCtx, http.MethodPost, responsesURL, cfg.APIKey, responsesPayload, &responsesBody)
	if responsesErr == nil {
		data.SucceededVia = "responses"
		data.Preview = extractText(responsesBody)
		if verbose {
			data.ResponsesOutput = cloneMap(responsesBody)
		}
		return CheckResult{Name: CheckProbe, OK: true, Message: "probe succeeded"}, data
	}
	if verbose {
		data.ResponsesError = compactErr(responsesErr)
	}

	chatPayload := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": "Reply with exactly: pong"},
		},
		"max_tokens": 16,
	}
	if verbose {
		data.ChatRequest = cloneMap(chatPayload)
	}
	var chatBody map[string]any
	chatURL := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	chatCtx, cancelChat := context.WithTimeout(ctx, 30*time.Second)
	defer cancelChat()
	chatErr := doJSONRequest(chatCtx, http.MethodPost, chatURL, cfg.APIKey, chatPayload, &chatBody)
	if chatErr == nil {
		data.SucceededVia = "chat.completions"
		data.Preview = extractText(chatBody)
		if verbose {
			data.ChatOutput = cloneMap(chatBody)
		}
		return CheckResult{Name: CheckProbe, OK: true, Message: "probe succeeded via fallback"}, data
	}
	if verbose {
		data.ChatError = compactErr(chatErr)
	}

	msg := fmt.Sprintf(
		"probe failed: /responses: %s; /chat/completions: %s",
		compactErr(responsesErr),
		compactErr(chatErr),
	)
	return CheckResult{Name: CheckProbe, OK: false, Message: msg}, data
}

func RunDoctor(ctx context.Context, cfg config.Resolved, verbose bool) ([]CheckResult, DoctorData) {
	results := make([]CheckResult, 0, 3)

	pingResult, pingData := RunPing(ctx, cfg)
	results = append(results, pingResult)

	modelsResult, modelsData := runModels(ctx, cfg, verbose)
	results = append(results, modelsResult)

	probeResult, probeData := runProbe(ctx, cfg, verbose)
	results = append(results, probeResult)

	failed := 0
	for _, r := range results {
		if !r.OK {
			failed++
		}
	}

	summary := DoctorData{Passed: len(results) - failed, Failed: failed}
	if verbose {
		safe := cfg.Safe()
		summary.Input = &safe
		summary.Ping = &pingData
		summary.Models = &modelsData
		summary.Probe = &probeData
	}

	return results, summary
}

func BuildEnvelope(command string, cfg config.Resolved, results []CheckResult, data any, verbose bool) Envelope {
	ok := true
	for _, r := range results {
		if !r.OK {
			ok = false
			break
		}
	}

	var errPayload *ErrorPayload
	if !ok {
		errPayload = &ErrorPayload{Message: firstFailure(results)}
	}

	return Envelope{
		OK:      ok,
		Command: command,
		Verbose: verbose,
		Config:  cfg.Safe(),
		Checks:  results,
		Data:    data,
		Error:   errPayload,
	}
}

func doJSONRequest(ctx context.Context, method, url, apiKey string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		trimmed := strings.TrimSpace(string(respBody))
		if len(trimmed) > 200 {
			trimmed = trimmed[:200] + "..."
		}
		if trimmed == "" {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return fmt.Errorf("status %d: %s", resp.StatusCode, trimmed)
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func firstFailure(results []CheckResult) string {
	for _, r := range results {
		if !r.OK {
			return r.Message
		}
	}
	return "one or more checks failed"
}

func extractText(body map[string]any) string {
	if body == nil {
		return ""
	}
	if text, ok := body["output_text"].(string); ok {
		return text
	}
	if choicesRaw, ok := body["choices"].([]any); ok && len(choicesRaw) > 0 {
		choice, ok := choicesRaw[0].(map[string]any)
		if !ok {
			return ""
		}
		message, ok := choice["message"].(map[string]any)
		if !ok {
			return ""
		}
		content, _ := message["content"].(string)
		return content
	}
	return ""
}

func compactErr(err error) string {
	if err == nil {
		return "unknown error"
	}
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 180 {
		return msg[:180] + "..."
	}
	return msg
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	b, err := json.Marshal(src)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}
