# Chitragupta

<p align="center">
  <img src="https://img.shields.io/badge/version-1.0.0--mvp-blue" alt="Version">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  <img src="https://img.shields.io/badge/status-production--ready-brightgreen" alt="Status">
</p>

**Universal package manager for AI development** — version, distribute, and manage AI skills, prompts, agents, and tools across teams and repositories.

Named after [**Chitragupta**](https://en.wikipedia.org/wiki/Chitragupta) (चित्रगुप्त), the Hindu deity who maintains divine records, Chitragupta helps teams treat AI assets (Claude Code skills, MCP tools, Cursor prompts) like code: versioned, dependency-managed, and distributed through a private registry.

```bash
# Install security toolkit across all repos
chitragupta install security-toolkit
# or use short alias
cg install security-toolkit

# Publish your team's prompt library
chitragupta publish ./my-package

# Pin versions, resolve dependencies automatically
cg install code-review@^2.1.0
```

---

## Why I Built This

**The Problem:**

I work with Claude Code across 20+ repositories. Every time I build a useful skill (security scanner, git workflow automation), I copy-paste it to other repos. Then:
- Updates overwrite customizations (my `.claude/` configs get nuked)
- No way to share private skills with my team
- Can't version or roll back broken updates
- Manual dependency tracking (this skill needs that MCP tool)
- Claude Marketplace only handles public, per-machine installs

**It's like npm didn't exist and we copy-pasted `node_modules/` between projects.**

**The Solution:**

Chitragupta is `npm` for AI assets. Publish packages to a team registry, install with dependency resolution, preserve customizations through templates, pin versions, and automate distribution.

---

## Chitragupta vs Claude Marketplace vs APM

### Claude Marketplace
**What it is:** Public skill store, manual per-machine installation  
**What Chitragupta adds:**
- ✅ Private packages (not everything belongs on the internet)
- ✅ Multi-repo installation (`chitragupta install` in 50 repos vs manual clicks)
- ✅ Version pinning + rollbacks
- ✅ Dependency resolution (install `security-toolkit`, auto-get `git-helpers`)
- ✅ Customization via templates (updates preserve your changes)
- ✅ CI/CD automation (install packages in Docker builds)

**Analogy:** Marketplace = Docker Hub. Chitragupta = Artifactory + Helm.

### APM (Agentic Package Manager)
**What it is:** Another package manager for AI assets  
**How Chitragupta differs:**
- **Multi-source support:** Install from git, OCI registries, HTTP tarballs, not just one registry
- **Template rendering:** `{{REPO_NAME}}` variables auto-filled, updates preserve custom edits
- **Security scanning:** Built-in checks for Unicode attacks, command injection
- **Workspace support:** Monorepos with shared dependencies (like Yarn workspaces)
- **Compile targets:** Generate configs for Copilot/Cursor/Claude from one manifest
- **Parallel downloads:** 10× faster with concurrent fetching + caching

---

## Features

### Core
- **📦 Multi-source packages** — Install from registry, Git, OCI (Docker/GHCR), HTTP tarballs
- **🔒 Private registries** — Self-host for internal tools, not public Marketplace
- **📌 Semantic versioning** — Pin versions (`^1.2.0`, `~2.1`, `>=3.0.0`), rollback on break
- **🌳 Dependency graphs** — Auto-resolve transitive deps, detect conflicts, deduplicate
- **⚡ Parallel downloads** — Concurrent fetching + integrity verification
- **🔐 Security scanning** — Detects Unicode attacks, command injection, suspicious patterns
- **🎯 Template rendering** — `{{REPO_NAME}}` variables, preserve custom edits on update
- **🏢 Workspace support** — Monorepos with shared dependencies (Yarn/pnpm model)
- **🔧 Compile targets** — Generate Copilot instructions or Cursor configs from manifest
- **🔍 Lockfiles** — SHA-256 integrity hashes, reproducible installs

### Primitives Supported
- **Skills** — Claude Code `.claude/skills/` definitions
- **Prompts** — Reusable prompt templates
- **Instructions** — System instructions for AI agents
- **Agents** — Multi-step agentic workflows
- **Hooks** — Bash scripts for git/lifecycle events
- **MCP Tools** — Model Context Protocol integrations

---

## Installation

### Binary Release (Recommended)
```bash
# macOS/Linux
curl -fsSL https://get.chitragupta.dev | sh

# Or download from GitHub releases
wget https://github.com/ppanda/chitragupta/releases/latest/download/chitragupta-$(uname -s)-$(uname -m)
chmod +x chitragupta-*
sudo mv chitragupta-* /usr/local/bin/chitragupta

# Create short alias
sudo ln -s /usr/local/bin/chitragupta /usr/local/bin/cg
```

### From Source
```bash
git clone https://github.com/ppanda/chitragupta.git
cd chitragupta
make build
sudo make install
```

### Verify
```bash
chitragupta --version
# chitragupta version 1.0.0-mvp

cg --version
# chitragupta version 1.0.0-mvp (alias)
```

---

## Quick Start

### 1. Install packages
```bash
# Install from registry
chitragupta install security-toolkit
# or use short form
cg install security-toolkit

# Install specific version
cg install code-review@2.1.0

# Install to current repo (creates .claude/)
cg install skill-name

# Install globally (~/.claude/)
cg install -g git-helpers
```

### 2. Create a package
```bash
mkdir my-package && cd my-package

# Create manifest
cat > chitragupta.yml <<EOF
name: my-package
version: 1.0.0
description: My team's AI skills
dependencies:
  registry:
    - git-helpers: ^1.2.0

install:
  global:
    - src: skills/*
      dest: skills/
  repo:
    - src: hooks/pre-commit.sh
      dest: hooks/
      template: true
      vars:
        - REPO_NAME
        - TEAM_NAME
EOF

# Add skills
mkdir -p skills
echo "# My skill" > skills/my-skill.md

# Publish
cg publish .
```

### 3. Use templates
```yaml
# In your package, create templates/ with {{VARS}}
install:
  repo:
    - src: templates/config.yml
      dest: .claude/config.yml
      template: true
      vars:
        - REPO_NAME    # Auto-detected from git
        - TEAM_NAME    # Prompted if not found
```

When installed, `{{REPO_NAME}}` becomes actual repo name. Updates re-render, preserving manual edits.

---

## Package Structure

```
my-package/
├── chitragupta.yml     # Manifest (metadata, deps, install rules)
├── chitragupta.lock    # Lockfile (SHA-256 hashes)
├── skills/             # Claude Code skills
│   └── security-scan.md
├── hooks/              # Git hooks
│   └── pre-commit.sh
├── tools/              # MCP tool definitions
│   └── jira-mcp.json
├── templates/          # Customizable files with {{VARS}}
│   └── claude-config.yml
└── docs/               # CLAUDE.md fragments
    └── architecture.md
```

---

## Manifest Reference

```yaml
name: security-toolkit
version: 1.0.0
description: Security review skills + hooks
author: Team Security
license: MIT
homepage: https://github.com/myorg/security-toolkit

# Dependencies from multiple sources
dependencies:
  registry:
    - git-helpers: ^1.2.0
    - code-review: 2.1.0
  git:
    - github.com/org/internal-tools#main
  oci:
    - ghcr.io/myorg/mcp-tools:latest
  http:
    - https://cdn.example.com/packages/utils.tar.gz

# Installation rules
install:
  global:  # Install to ~/.claude/
    - src: skills/*
      dest: skills/
  repo:    # Install to .claude/
    - src: hooks/pre-commit.sh
      dest: hooks/
      template: true  # Render {{VARS}}
      vars:
        - REPO_NAME
        - TEAM_NAME
```

---

## Self-Hosted Registry

Run your own private registry for team packages.

### Docker Compose (Recommended)
```bash
git clone https://github.com/ppanda/chitragupta.git
cd chitragupta

# Start server (SQLite)
docker-compose up

# Server runs on http://localhost:8080
```

### Manual Setup
```bash
# Start registry server
make run-server

# Configure CLI to use it
export CHITRAGUPTA_REGISTRY=http://localhost:8080

# Publish packages
cg publish ./my-package
```

### API Endpoints
```
POST   /api/v1/packages                   Publish package
GET    /api/v1/packages                   List all
GET    /api/v1/packages/search?q=...      Search
GET    /api/v1/packages/:name             Get latest
GET    /api/v1/packages/:name/:version    Get specific version
GET    /api/v1/packages/:name/:version/download  Download tarball
GET    /health                            Health check
```

---

## Commands

```bash
# Installation
cg install                    Install from manifest (chitragupta.yml)
cg install pkg@1.0.0         Install specific package
cg install -g pkg            Install globally

# Publishing
cg publish <dir>             Publish package to registry

# Discovery
cg list                      List all packages in registry
cg search <query>            Search packages

# Validation
cg verify                    Verify lockfile matches installed packages

# Compilation
cg compile -t copilot        Generate Copilot instructions
cg compile -t cursor         Generate Cursor config

# Workspaces
cg workspace list            Show workspace members
cg workspace install         Install shared deps across workspaces
```

---

## Advanced Usage

### Workspaces (Monorepos)
```yaml
# root/chitragupta.yml
workspaces:
  - packages/*
  - services/*

dependencies:
  registry:
    - shared-utils: ^1.0.0  # Installed for all workspaces
```

Each workspace can have its own manifest, but shared deps install once.

### Security Scanning
```bash
# Scan before install
cg install --scan security-toolkit

# Check reports
cat .claude/security-reports/security-toolkit.json
```

Detects:
- Unicode homoglyphs (ѕ vs s)
- Command injection (`rm -rf`, `curl | sh`)
- Suspicious patterns (base64 decode, eval)

### Template Variables
Built-in vars (auto-detected):
- `{{REPO_NAME}}` — Git repo name
- `{{REPO_OWNER}}` — Org/user from remote URL
- `{{LANGUAGE}}` — Detected from file counts (Go, Python, etc.)

Custom vars (prompted):
- `{{TEAM_NAME}}` — Asked on first install
- `{{API_KEY}}` — Never stored, re-prompt each time

---

## Development

### Build from source
```bash
git clone https://github.com/ppanda/chitragupta.git
cd chitragupta

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run with race detector
go test -race ./...
```

### Project Structure
```
chitragupta/
├── cmd/
│   ├── chitra/          CLI commands
│   └── chitra-server/   Registry server
├── pkg/
│   ├── sources/         Multi-source downloaders
│   ├── installer/       File copying + templates
│   ├── security/        Scanner
│   ├── graph/           Dependency resolver
│   └── lockfile/        Integrity checks
├── internal/
│   └── config/          Config management
└── Makefile
```

## Contributing

Contributions welcome! Please:
1. Fork the repo
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing`)
5. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

MIT License - see [LICENSE](LICENSE) file.

**Star this repo if Chitra helps your workflow!** ⭐
