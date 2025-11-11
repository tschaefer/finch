# The Minimal Observability Infrastructure [![Go Report Card](https://goreportcard.com/badge/github.com/tschaefer/finch)](https://goreportcard.com/report/github.com/tschaefer/finch)

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

For deployment instructions, please refer to the [finchctl repository](https://github.com/tschaefer/finchctl).

## Registering an Observability Agent

Finch provides a REST API to manage agents. The API is protected by basic
authentication; credentials are provided during stack deployment.

To register a new agent, supply a hostname and at least one log source.
Supported log sources include:

- `journal://` – Read logs from the systemd journal
- `docker://` – Read logs from the Docker daemon
- `file://var/log/*.log` – Read logs from files or file patterns

File sources can be specified multiple times.

You may also specify `tags` to identify the agent and enable metrics or
profiling collection.

Example request:

```bash
curl -u admin:admin -X POST \
  -H "Content-Type: application/json" \
  -d '{"hostname": "app.example.com", "log_sources": ["journal://"], "metrics": true, "profiling": true }' \
  https://finch.example.com/api/v1/agent

{"rid":"rid:finch:45190462017e8f71:agent:bf87bb48-3ef8-4baf-852c-7210ac48baa4"}
```

On success, the API returns a resource ID (`rid`) of the created agent.

## Fetching Agent Configuration

To fetch the configuration for a specific agent, use the resource ID returned earlier:

```bash
curl -u admin:admin \
  https://finch.example.com/api/v1/agent/rid:finch:45190462017e8f71:agent:bf87bb48-3ef8-4baf-852c-7210ac48baa4/config \
    -o agent.cfg
```

The downloaded configuration file can be used to
[enroll the agent](https://github.com/tschaefer/finchctl?tab=readme-ov-file#enrolling-an-observability-agent)
with finchctl.

## Further API Endpoints

Additional API endpoints are available for agent management:

- List all agents: `/api/v1/agent`
- Deregister an agent: `/api/v1/agent/{rid}`
- Service info: `/api/v1/info`

The OpenAPI specification is available at `/openapi.yaml`.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.
For major changes, open an issue first to discuss your proposal.

Ensure that your code adheres to the existing style and includes appropriate
tests.

## License

This project is licensed under the [MIT License](LICENSE).
