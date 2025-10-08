# The Minimal Logging Infrastructure

finch is the management service to register logging agents and provide the
related configuration. It is designed to be minimal and easy to use.

The minimal logging stack bases on Docker and consists of following
services:

- **Grafana** - The visualization tool
- **Loki** - The log aggregation system
- **Alloy** - The log shipping agent
- **Prometheus** - The monitoring system
- **Traefik** - The reverse proxy
- **Finch** - The log agent manager

Consider to read the [Blog post](https://blog.tschaefer.org/posts/2025/08/17/finch-a-minimal-logging-stack/)
for motivation and a walkthrough before using this tool.

For deployment please follow the instructions in the [finchctl
repository](https://github.com/tschaefer/finchctl).

## Register logging agent

finch provides a REST API to manage logging agents. The API is guarded by
basic authentication. The credentials are provided while the stack deployment.

The typical workflow is to register a new agent.

Providing a hostname and at least one log source is required. The log source
can be one of

- `journal://` - Read logs from the systemd journal.
- `docker://` - Read logs from the Docker daemon.
- `file://var/log/*.log` - Read logs from a file or a file pattern.

File sources can be specified multiple times.

Optionally you can specify a list of `tags` to identify the agent and enable
metrics collection.

```bash
curl -u admin:admin -X POST \
  -H "Content-Type: application/json" \
  -d '{"hostname": "app.example.com", "log_sources": ["journal://"], "metrics": true }' \
  https://finch.example.com/api/v1/agent

  {"rid":"rid:finch:45190462017e8f71:agent:bf87bb48-3ef8-4baf-852c-7210ac48baa4"}
```

On success the API returns a resource ID (rid) of the created agent.

## Fetch agent configuration

To fetch the configuration for a specific agent, you can use the resource ID
provided by the previous step.

```bash
curl -u admin:admin \
  https://finch.example.com/api/v1/agent/rid:finch:45190462017e8f71:agent:bf87bb48-3ef8-4baf-852c-7210ac48baa4/config \
    -o agent.cfg

```

The downloaded configuration file can be used to [enroll the
agent](https://github.com/tschaefer/finchctl?tab=readme-ov-file#enrolling-a-logging-agent)
with finchctl.

## Further API endpoints

The API provides further endpoints to manage agents, such as listing all
`/api/v1/agent` and deregister an agent `/api/v1/agent/{rid}`.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.
For major changes, open an issue first to discuss what you would like to change.

Ensure that your code adheres to the existing style and includes appropriate tests.

## License

This project is licensed under the [MIT License](LICENSE).
