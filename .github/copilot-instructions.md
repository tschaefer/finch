# Finch Development Guide

Finch is a management service for observability agents (Grafana Alloy), providing agent registration and configuration via gRPC API and a real-time web dashboard.

## Security Philosophy

Finch's security model is built around **no permanent credentials for humans**. This is an intentional design decision.

### Deployment Hardening (Recommended)
```
Network layer: IP allowlisting/private network
SSH layer:     Key-based auth only (no passwords)
Finch layer:   mTLS + short-lived tokens
```

### Security Model
- **Stack deployment/updates** - Via SSH (secrets for bcrypt and dashboard tokens deployed once)
- **Agent management** - Via mTLS (CA deployed via SSH, client cert/key never leave CLI machine)
- **Dashboard access** - Short-lived JWT tokens (no persistent user/session storage)

### Threat Model
**Protected against:**
- Credential leaks from breaches (tokens expire quickly, typically minutes)
- Compromised long-lived credentials (nothing is long-lived for humans)
- Automated attacks (network isolation + SSH keys + mTLS)

**Intentionally accepted:**
- Deliberate credential sharing (rare, detectable in logs, high friction)
- Motivated insider extracting agent configs (too expensive to prevent without major UX complexity)
- Sensitive data in logs/metrics/profiles (observability requires context; data sanitization is application responsibility)

### Observability Data Security
Finch aggregates logs, metrics, and profiling data - which **inherently contains sensitive information**. This cannot be prevented at the stack level:
- Logs need context to be useful (request IDs, user actions, system state)
- Metrics expose system behavior (traffic patterns, resource usage)
- Profiles reveal code execution paths

**Mitigation strategy:**
- Control *access* to observability data (mTLS, network isolation, short-lived tokens)
- Applications must implement their own data sanitization
- Grafana provides basic redacting features (not comprehensive)
- Treat observability backend with same security as production databases

**Do NOT attempt to:**
- Build automatic PII detection/redaction in Finch
- Filter or sanitize agent-submitted data
- Add complex data governance layers

The security model focuses on **access control**, not data sanitization.

### What NOT to Add
- ❌ Persistent user accounts or authentication databases
- ❌ Long-lived API keys
- ❌ Session management with server-side state
- ❌ Password-based authentication for any component

These would contradict the core security philosophy. The design prioritizes minimalism and operational simplicity over theoretical security perfection.

## Build, Test, and Lint

Do not run `go build` directly. Use the Makefile for consistent builds,
testing, and linting if possible. Otherwise, run `go run` to prevent artifacts
leftovers in the repository.

### Build
```bash
make dist                    # Build binary to bin/finch-linux-amd64
make dist GOOS=darwin        # Cross-compile for other platforms
```

### Test
```bash
make test                    # Run all tests (silent, fails on errors)
go test -v ./...             # Run all tests with verbose output
go test -v ./internal/aes    # Run tests for a single package
```

### Format and Lint
```bash
make fmt                     # Check formatting (fails if issues found)
gofmt -w .                   # Auto-format all files
make lint                    # Run golangci-lint (see .golangci.yaml)
golangci-lint run            # Run linter directly
```

### Proto Generation
```bash
make proto                   # Regenerate gRPC code from api/api.proto
```

## Architecture

### Request Flow
```
Client (finchctl/mTLS) → gRPC Server → Controller → Model → Database (SQLite)
                                     ↓
                         HTTP Server (Dashboard) → WebSocket → Real-time updates
```

### Key Components

**Manager** (`internal/manager`)
- Entry point that orchestrates all components
- Initializes config, database, controller, and servers
- Runs both gRPC and HTTP servers concurrently

**Controller** (`internal/controller`)
- Business logic layer between gRPC handlers and model
- Generates Grafana Alloy configs from agent data
- Manages agent credentials and authentication tokens
- Publishes agent events for WebSocket subscribers

**Model** (`internal/model`)
- Database abstraction layer using GORM
- Manages Agent entities with event broadcasting
- Handles CRUD operations and publishes changes to subscribers

**gRPC Server** (`internal/grpc`)
- Implements AgentService, InfoService, DashboardService
- Uses custom interceptors for mTLS auth, logging, headers
- Thin layer delegating to Controller

**HTTP Server** (`internal/http`)
- Serves web dashboard with real-time WebSocket updates
- JWT authentication for dashboard access
- Uses embedded templates from `internal/http/templates/`

**Config** (`internal/config`)
- Loads stack configuration from JSON file (default: `/var/lib/finch/finch.json`)

**Database** (`internal/database`)
- SQLite with single `agents` table storing observability agent metadata
- (Traefik) Basic user auth file modified by locking via `gofrs/flock`

### Agent Data Model
```go
type Agent struct {
    Hostname       string
    ResourceId     string   // Unique identifier (UUID)
    Labels         []string
    LogSources     []string // Log file paths
    Metrics        bool
    MetricsTargets []string // Prometheus scrape targets
    Profiles       bool     // Pyroscope profiling enabled
    Username       string   // Basic auth username
    Password       string   // Plain password (not stored)
    PasswordHash   string   // Bcrypt hash
}
```

## Conventions

### Error Handling
- Use `cobra.CheckErr(err)` in CLI commands
- gRPC handlers return gRPC status codes (e.g., `codes.NotFound`)
- HTTP handlers write appropriate status codes and JSON errors

### Logging
- Use structured logging with `log/slog`
- Debug logs include full context: `slog.Debug("message", "key", value)`
- Configure level and format via CLI flags: `--server.log-level`, `--server.log-format`

### Testing
- Test files named `*_test.go` alongside implementation
- Use `testify/assert` for assertions
- Mock external dependencies (database, config) in tests

### Code Organization
- `cmd/` - CLI commands using Cobra
- `internal/` - Private application packages
- `api/` - Protocol buffer definitions and generated code
- No circular dependencies between internal packages

### Protocol Buffers
- Source of truth: `api/api.proto`
- Generated files: `api/api.pb.go`, `api/api_grpc.pb.go`
- Regenerate with `make proto` after modifying `.proto` file

### Configuration
- Stack config loaded from JSON file path (CLI flag: `--stack.config-file`)
- Contains service URLs, TLS certificates, encryption keys
- See `internal/config/config.go` for structure

### Version Information
- Version injected at build time via ldflags (see Makefile)
- Accessed via `internal/version` package
- Includes version tag and git commit
