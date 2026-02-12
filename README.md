# The Minimal Observability Infrastructure 

[![Tag](https://img.shields.io/github/tag/tschaefer/finch.svg)](https://github.com/tschaefer/finch/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/tschaefer/finch)](https://goreportcard.com/report/github.com/tschaefer/finch)
[![Coverage](https://img.shields.io/codecov/c/github/tschaefer/finch)](https://codecov.io/gh/tschaefer/finch)
[![Contributors](https://img.shields.io/github/contributors/tschaefer/finch)](https://github.com/tschaefer/finch/graphs/contributors)
[![License](https://img.shields.io/github/license/tschaefer/finch)](./LICENSE)

**Finch** is the management service for registering observability agents and
providing related configuration. It is designed to be minimal and easy to use.

The minimal observability stack is based on Docker and consists of the
following services:

- **Grafana** – Visualization and dashboards
- **Loki** – Log aggregation system
- **Mimir** – Metrics backend
- **Pyroscope** – Profiling data aggregation and visualization
- **Alloy** – Client-side agent for logs, metrics, and profiling data
- **Traefik** – Reverse proxy and TLS termination
- **Finch** – Agent manager

See the [Blog post](https://blog.tschaefer.org/posts/2025/08/17/finch-a-minimal-logging-stack/)
for background, motivation, and a walkthrough before you get started.

## gRPC API

Finch provides a gRPC API with three services:

- **AgentService** – Manages observability agents
- **InfoService** – Provides service information
- **DashboardService** – Configures dashboard authentication

The API is protected by mTLS authentication; certificates and keys are provided during stack deployment.

For the API reference, see the [proto definitions](https://github.com/tschaefer/finch/tree/main/api/proto).

### Agent Authentication

Registered agents authenticate using JWT tokens with a **365-day expiration**.
Tokens are generated during agent registration and must be rotated before
expiry. The expiration date is displayed in the dashboard and included as a
comment in generated Alloy configurations. The regeneration happens with any
config request.

## Web Dashboard

Finch provides a lightweight dashboard for visualizing agents with real-time
updates.

**Features:**
- Real-time agent updates
- Service information, statistics, and endpoints
- Agent details (logs, metrics, profiles, labels)
- Download configs, reveal credentials, search/filter
- Dark theme

### Using finchctl (Official CLI)

The **official** way to interact with Finch is via **[finchctl](https://github.com/tschaefer/finchctl)**, a dedicated command-line interface that handles mTLS authentication automatically.

**Install finchctl:**
```bash
# See installation instructions at:
# https://github.com/tschaefer/finchctl#installation
```
## Contributing

Fork, make changes, submit PR. For major changes, open an issue first.

## License

This project is licensed under the [MIT License](LICENSE).
