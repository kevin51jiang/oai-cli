# oaicheck

A tiny CLI for debugging OpenAI API config.

## Commands

- `oaicheck` (shows help)
- `oaicheck doctor`
- `oaicheck ping`
- `oaicheck models`
- `oaicheck probe`

## Flags

- `--base-url` (or `OPENAI_BASE_URL`, default `https://api.openai.com/v1`)
- `--api-key` (or `OPENAI_API_KEY`)
- `--model` (or `OPENAI_MODEL`)
- `--json` (machine-readable output)

Flags override environment variables.

## What each check does

- `ping`: sends a request to `<base-url>/models` and treats any HTTP response as reachable.
- `models`: calls OpenAI models list to verify base URL + API key.
- `probe`: sends a tiny generation request to `/responses`; if that fails, retries with `/chat/completions`.
- `doctor`: runs all three checks and summarizes pass/fail.

## JSON mode

`--json` prints exactly one JSON object to stdout and nothing else.

Example:

```bash
oaicheck doctor --json
```

## Shell completion

Generate completion scripts with:

```bash
oaicheck completion <shell>
```

### Bash (Linux)

```bash
mkdir -p ~/.local/share/bash-completion/completions
oaicheck completion bash > ~/.local/share/bash-completion/completions/oaicheck
```

### Bash (macOS + Homebrew bash-completion)

```bash
oaicheck completion bash > "$(brew --prefix)/etc/bash_completion.d/oaicheck"
```

### Zsh

```bash
mkdir -p "${HOME}/.zfunc"
oaicheck completion zsh > "${HOME}/.zfunc/_oaicheck"
```

Then ensure `${HOME}/.zfunc` is in `fpath`, and initialize completion:

```bash
autoload -U compinit && compinit
```

### Fish

```bash
mkdir -p ~/.config/fish/completions
oaicheck completion fish > ~/.config/fish/completions/oaicheck.fish
```

### PowerShell

For the current session:

```powershell
oaicheck completion powershell | Out-String | Invoke-Expression
```

To persist:

```powershell
oaicheck completion powershell > $PROFILE.CurrentUserAllHosts
```

## Build

```bash
go build -o oaicheck .
```

## Quick run

```bash
OPENAI_API_KEY=... OPENAI_MODEL=gpt-4.1-mini oaicheck doctor
```
