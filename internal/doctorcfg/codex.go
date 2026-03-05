package doctorcfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const codexConfigCheckName = "codex-config"

type CodexOptions struct {
	CWD             string
	ProfileOverride string
	ConfigOverride  string
}

type ResolveResult struct {
	BaseURL string
	APIKey  string
	Model   string

	CheckName    string
	CheckOK      bool
	CheckMessage string
	CheckDetails string
}

func ResolveCodex(opts CodexOptions) ResolveResult {
	cwd := strings.TrimSpace(opts.CWD)
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	if cwd == "" {
		cwd = "."
	}

	configPaths, pathNotes := codexConfigPaths(cwd, strings.TrimSpace(opts.ConfigOverride))

	merged := map[string]any{}
	notes := append([]string{}, pathNotes...)
	loaded := 0
	for _, p := range configPaths {
		fileMap, err := decodeTOMLFile(p)
		if err != nil {
			notes = append(notes, fmt.Sprintf("config parse failed at %s: %v", p, err))
			continue
		}
		deepMerge(merged, fileMap)
		loaded++
	}
	if loaded == 0 {
		notes = append(notes, "no Codex config.toml files loaded")
	} else {
		notes = append(notes, fmt.Sprintf("loaded %d config file(s)", loaded))
	}

	activeProfile := strings.TrimSpace(opts.ProfileOverride)
	if activeProfile == "" {
		activeProfile = getStringPath(merged, "profile")
	}
	if activeProfile == "" {
		activeProfile = "default"
	}

	topProfiles := getMapPath(merged, "profiles")
	profileMap := getMapPath(topProfiles, activeProfile)
	if profileMap == nil {
		notes = append(notes, fmt.Sprintf("profile %q not found; using top-level values", activeProfile))
	}

	model := firstNonEmpty(
		getStringPath(profileMap, "model"),
		getStringPath(merged, "model"),
	)

	providerName := firstNonEmpty(
		getStringPath(profileMap, "model_provider"),
		getStringPath(merged, "model_provider"),
		"openai",
	)

	topProvider := getMapPath(getMapPath(merged, "model_providers"), providerName)
	profileProvider := getMapPath(getMapPath(profileMap, "model_providers"), providerName)
	effectiveProvider := map[string]any{}
	deepMerge(effectiveProvider, topProvider)
	deepMerge(effectiveProvider, profileProvider)

	baseURL := firstNonEmpty(
		getStringPath(effectiveProvider, "base_url"),
	)
	if providerName == "openai" {
		baseURL = firstNonEmpty(baseURL, getStringPath(merged, "openai_api_base"))
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")

	envKey := firstNonEmpty(
		getStringPath(effectiveProvider, "env_key"),
	)
	if providerName == "openai" {
		envKey = firstNonEmpty(envKey, getStringPath(merged, "openai_api_key_env_var"), "OPENAI_API_KEY")
	}

	apiKey := ""
	apiKeySource := ""
	if envKey != "" {
		apiKey = strings.TrimSpace(os.Getenv(envKey))
		if apiKey != "" {
			apiKeySource = fmt.Sprintf("env:%s", envKey)
		}
	}
	if apiKey == "" {
		authPath := codexAuthPath()
		authKey, authSource, err := readAuthKey(authPath, envKey)
		if err != nil {
			notes = append(notes, fmt.Sprintf("auth parse failed at %s: %v", authPath, err))
		} else if authKey != "" {
			apiKey = authKey
			apiKeySource = authSource
		}
	}

	notes = append(notes,
		fmt.Sprintf("profile=%s", activeProfile),
		fmt.Sprintf("provider=%s", providerName),
	)
	if envKey != "" {
		notes = append(notes, fmt.Sprintf("provider env_key=%s", envKey))
	}
	if apiKeySource != "" {
		notes = append(notes, fmt.Sprintf("api key source=%s", apiKeySource))
	}

	missing := make([]string, 0, 3)
	if baseURL == "" {
		missing = append(missing, "base URL")
	}
	if model == "" {
		missing = append(missing, "model")
	}
	if apiKey == "" {
		missing = append(missing, "API key")
	}

	ok := len(missing) == 0
	message := "codex config resolved"
	if !ok {
		message = fmt.Sprintf("codex config incomplete: missing %s", strings.Join(missing, ", "))
	}

	return ResolveResult{
		BaseURL:      baseURL,
		APIKey:       apiKey,
		Model:        model,
		CheckName:    codexConfigCheckName,
		CheckOK:      ok,
		CheckMessage: message,
		CheckDetails: strings.Join(notes, "; "),
	}
}

func codexConfigPaths(cwd, configOverride string) ([]string, []string) {
	paths := make([]string, 0, 8)
	notes := make([]string, 0, 4)

	add := func(path string) {
		if path == "" {
			return
		}
		clean := filepath.Clean(path)
		for _, existing := range paths {
			if existing == clean {
				return
			}
		}
		if isFile(clean) {
			paths = append(paths, clean)
		}
	}

	add("/etc/codex/config.toml")
	add(codexConfigPath())

	for _, p := range projectConfigPaths(cwd) {
		add(p)
	}

	if configOverride != "" {
		clean := filepath.Clean(configOverride)
		if isFile(clean) {
			paths = append(paths, clean)
		} else {
			notes = append(notes, fmt.Sprintf("--codex-config not found: %s", clean))
		}
	}

	return paths, notes
}

func projectConfigPaths(cwd string) []string {
	root, ok := gitRoot(cwd)
	if !ok {
		root = cwd
	}

	dirs := make([]string, 0, 16)
	for d := filepath.Clean(cwd); ; d = filepath.Dir(d) {
		dirs = append(dirs, d)
		if d == root || d == filepath.Dir(d) {
			break
		}
	}
	reverseStrings(dirs)

	paths := make([]string, 0, len(dirs))
	for _, d := range dirs {
		paths = append(paths, filepath.Join(d, ".codex", "config.toml"))
	}
	return paths
}

func gitRoot(start string) (string, bool) {
	for d := filepath.Clean(start); ; d = filepath.Dir(d) {
		if isDir(filepath.Join(d, ".git")) || isFile(filepath.Join(d, ".git")) {
			return d, true
		}
		if d == filepath.Dir(d) {
			return "", false
		}
	}
}

func codexHome() string {
	if v := strings.TrimSpace(os.Getenv("CODEX_HOME")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(".", ".codex")
	}
	return filepath.Join(home, ".codex")
}

func codexConfigPath() string {
	return filepath.Join(codexHome(), "config.toml")
}

func codexAuthPath() string {
	return filepath.Join(codexHome(), "auth.json")
}

func readAuthKey(authPath, envKey string) (string, string, error) {
	if !isFile(authPath) {
		return "", "", nil
	}
	b, err := os.ReadFile(authPath)
	if err != nil {
		return "", "", err
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return "", "", err
	}

	if envKey != "" {
		if key := getStringPath(raw, envKey); key != "" {
			return key, fmt.Sprintf("auth:%s", envKey), nil
		}
	}
	if key := getStringPath(raw, "OPENAI_API_KEY"); key != "" {
		return key, "auth:OPENAI_API_KEY", nil
	}
	if key := getStringPath(raw, "tokens", "access_token"); key != "" {
		return key, "auth:tokens.access_token", nil
	}
	if key := getStringPath(raw, "tokens", "id_token"); key != "" {
		return key, "auth:tokens.id_token", nil
	}
	return "", "", nil
}

func decodeTOMLFile(path string) (map[string]any, error) {
	var out map[string]any
	if _, err := toml.DecodeFile(path, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func deepMerge(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src {
		srcMap, srcIsMap := asMap(v)
		if !srcIsMap {
			dst[k] = v
			continue
		}
		dstMap, dstIsMap := asMap(dst[k])
		if !dstIsMap {
			dstMap = map[string]any{}
			dst[k] = dstMap
		}
		deepMerge(dstMap, srcMap)
	}
}

func asMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}

func getMapPath(m map[string]any, path ...string) map[string]any {
	cur := m
	for i, k := range path {
		if cur == nil {
			return nil
		}
		v, ok := cur[k]
		if !ok {
			return nil
		}
		next, ok := v.(map[string]any)
		if !ok {
			if i == len(path)-1 {
				return nil
			}
			return nil
		}
		cur = next
	}
	return cur
}

func getStringPath(m map[string]any, path ...string) string {
	if len(path) == 0 {
		return ""
	}
	cur := m
	for i := 0; i < len(path)-1; i++ {
		if cur == nil {
			return ""
		}
		next, ok := cur[path[i]].(map[string]any)
		if !ok {
			return ""
		}
		cur = next
	}
	if cur == nil {
		return ""
	}
	v, ok := cur[path[len(path)-1]]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func reverseStrings(values []string) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}
