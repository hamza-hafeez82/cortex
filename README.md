# Cortex

**A three-stage CLI that audits your codebase for security vulnerabilities, architectural issues, and dependency risks — with AI-powered explanations.**

Local-first. No data leaves your machine. Works with Ollama, OpenAI, Anthropic, or any OpenAI-compatible API.

```
$ cortex scan ./

  ██████╗ ██████╗ ██████╗ ████████╗███████╗██╗  ██╗
 ██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝██╔════╝╚██╗██╔╝
 ██║     ██║   ██║██████╔╝   ██║   █████╗   ╚███╔╝
 ██║     ██║   ██║██╔══██╗   ██║   ██╔══╝   ██╔██╗
 ╚██████╗╚██████╔╝██║  ██║   ██║   ███████╗██╔╝ ██╗
  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝

  Stage 1  Reconnaissance ████████████████ done
  Stage 2  Security       ████████████████ done  12 issues
  Stage 3  Architecture   ████████████████ done   5 issues

  CRITICAL  CX-SEC-001  api_key hardcoded          src/config.js:14
  HIGH      CX-SEC-002  SQL injection risk          src/db/users.js:87
  HIGH      CX-DEP-001  lodash < 4.17.21 (CVE-...)  package.json:12
  MEDIUM    CX-ARCH-001  God file detected           src/app.js (892 lines)

  17 issues found across 3 categories
  Run `cortex explain CX-SEC-001` for an AI explanation of any issue
```

---

## Install

**One-line install (Linux/macOS):**
```bash
curl -fsSL https://get.cortex-edr.com | sh
```

**Homebrew:**
```bash
brew install hamza-hafeez82/tap/cortex
```

**Go:**
```bash
go install github.com/hamza-hafeez82/cortex/cmd/cortex@latest
```

**Windows:**
Download the latest binary from [Releases](https://github.com/hamza-hafeez82/cortex/releases).

---

## Quick start

```bash
# Scan current directory
cortex scan .

# Scan a specific repo
cortex scan ./my-project

# Get AI explanation for any issue
cortex explain CX-SEC-001

# Export report as JSON (for CI/CD)
cortex scan . --json > report.json

# Only show critical and high issues
cortex scan . --severity high
```

---

## The three stages

### Stage 1 — Reconnaissance
Cortex first builds a complete picture of your codebase before raising any issues: file structure, tech stack, framework detection, dependency graph, HTTP endpoints, and infrastructure config files.

### Stage 2 — Security scanning
A suite of detectors runs against the recon data, searching for security vulnerabilities: hardcoded secrets, injection risks, insecure dependencies, JWT misconfigurations, CORS issues, and more.

### Stage 3 — Architecture analysis
Cortex measures the health of your architectural decisions: god files, circular dependencies, missing error handling, dead code, and test coverage gaps.

---

## AI configuration

Cortex works with any AI backend. Run `cortex config` to set up interactively.

**Ollama (local, recommended):**
```bash
# Install Ollama and pull a model
ollama pull llama3

# Cortex will detect Ollama automatically
cortex config
```

**OpenAI:**
```bash
cortex config --provider openai --api-key sk-...
```

**Anthropic:**
```bash
cortex config --provider anthropic --api-key sk-ant-...
```

**Any OpenAI-compatible API:**
```bash
cortex config --provider openai-compat --base-url https://your-api/v1 --api-key ...
```

Config is stored in `~/.cortex/config.yaml`.

---

## Issue codes

All findings follow the format `CX-{CATEGORY}-{NUMBER}`:

| Prefix | Category |
|--------|----------|
| `CX-SEC` | Security vulnerabilities |
| `CX-DEP` | Dependency issues |
| `CX-ARCH` | Architecture problems |

See the full registry in [`docs/issue-codes.md`](docs/issue-codes.md).

---

## CI/CD integration

```yaml
# GitHub Actions example
- name: Run Cortex scan
  run: |
    curl -fsSL https://get.cortex-edr.com | sh
    cortex scan . --json --severity high > cortex-report.json

- name: Upload report
  uses: actions/upload-artifact@v4
  with:
    name: cortex-report
    path: cortex-report.json
```

---

## Contributing

We welcome new detectors, language support, and bug fixes. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

---

## License

MIT — see [LICENSE](LICENSE).

---

Built by [Hamza Hafeez](https://github.com/hamza-hafeez82) · [cortex-edr.com](https://cortex-edr.com)