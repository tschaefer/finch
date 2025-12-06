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

## gRPC API

Finch provides a gRPC API with two services:

- **AgentService** – Manages observability agents
- **InfoService** – Provides service information

The API is protected by basic authentication; credentials are provided during stack deployment.

### Using grpcurl

[grpcurl](https://github.com/fullstorydev/grpcurl) is a command-line tool for
interacting with gRPC servers.

**Register a new agent:**

To register a new agent, supply a hostname and at least one log source.
Supported log sources include:

- `journal://` – Read logs from the systemd journal
- `docker://` – Read logs from the Docker daemon
- `file://var/log/*.log` – Read logs from files or file patterns

File sources can be specified multiple times. You may also specify `tags` to
identify the agent and enable metrics or profiling collection.

```bash
grpcurl \
  -H "Authorization: Basic $(echo -n 'admin:admin' | base64)" \
  -d '{
    "hostname": "app.example.com",
    "log_sources": ["journal://"],
    "metrics": true,
    "profiles": true
  }' \
  finch.example.com:443 \
  finch.AgentService/RegisterAgent
```

Response:
```json
{
  "rid": "rid:finch:45190462017e8f71:agent:bf87bb48-3ef8-4baf-852c-7210ac48baa4"
}
```

**List all agents:**

```bash
grpcurl \
  -H "Authorization: Basic $(echo -n 'admin:admin' | base64)" \
  finch.example.com:443 \
  finch.AgentService/ListAgents
```

**Get agent details:**

```bash
grpcurl \
  -H "Authorization: Basic $(echo -n 'admin:admin' | base64)" \
  -d '{"rid": "rid:finch:45190462017e8f71:agent:bf87bb48-3ef8-4baf-852c-7210ac48baa4"}' \
  finch.example.com:443 \
  finch.AgentService/GetAgent
```

**Get agent configuration:**

```bash
grpcurl \
  -H "Authorization: Basic $(echo -n 'admin:admin' | base64)" \
  -d '{"rid": "rid:finch:45190462017e8f71:agent:bf87bb48-3ef8-4baf-852c-7210ac48baa4"}' \
  finch.example.com:443 \
  finch.AgentService/GetAgentConfig | jq -r .config | base64 -d > agent.cfg
```

The downloaded configuration file can be used to
[enroll the agent](https://github.com/tschaefer/finchctl?tab=readme-ov-file#enrolling-an-observability-agent)
with finchctl.

**Deregister an agent:**

```bash
grpcurl \
  -H "Authorization: Basic $(echo -n 'admin:admin' | base64)" \
  -d '{"rid": "rid:finch:45190462017e8f71:agent:bf87bb48-3ef8-4baf-852c-7210ac48baa4"}' \
  finch.example.com:443 \
  finch.AgentService/DeregisterAgent
```

**Get service info:**

```bash
grpcurl \
  -H "Authorization: Basic $(echo -n 'admin:admin' | base64)" \
  finch.example.com:443 \
  finch.InfoService/GetServiceInfo
```

### Testing locally

For local testing without TLS, use the `-plaintext` flag:

```bash
grpcurl -plaintext \
  -H "authorization: Basic $(echo -n 'admin:admin' | base64)" \
  127.0.0.1:9090 \
  finch.InfoService/GetServiceInfo
```

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.
For major changes, open an issue first to discuss your proposal.

Ensure that your code adheres to the existing style and includes appropriate
tests.

## License

This project is licensed under the [MIT License](LICENSE).
