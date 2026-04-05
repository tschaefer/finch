<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".brand/finch-logo-horizontal-dark.svg">
  <img src=".brand/finch-logo-horizontal.svg" alt="Finch - The Minimal Observability Infrastructure" width="300">
</picture>

# The Minimal Observability Infrastructure

[![Tag](https://img.shields.io/github/tag/tschaefer/finchctl.svg)](https://github.com/tschaefer/finchctl/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/tschaefer/finchctl)](https://goreportcard.com/report/github.com/tschaefer/finchctl)
[![Contributors](https://img.shields.io/github/contributors/tschaefer/finchctl)](https://github.com/tschaefer/finchctl/graphs/contributors)
[![License](https://img.shields.io/github/license/tschaefer/finchctl)](./LICENSE)

Finch brings production-grade observability to your infrastructure — no
Kubernetes, no cloud vendor, no expertise required. Deploy a full logs,
metrics, and profiling stack in one command. Enroll agents on any Linux or
macOS machine in one more. Everything else — TLS, authentication, agent
configuration — is handled for you.

> Background, motivation, and a walkthrough: [Blog post](https://blog.tschaefer.org/posts/2025/08/17/finch-a-minimal-logging-stack/)

## Getting Started

Install the Finch CLI:

```bash
curl -sSfL https://finch.coresec.zone | sudo sh -
```

Alternatively, download a binary from the
[releases page](https://github.com/tschaefer/finchctl/releases) or build from
source.

## Deploy the Stack

You need a Linux machine with SSH access and superuser privileges.

```bash
finchctl service deploy root@10.19.80.100
```

That's it. The full observability stack is up at `https://10.19.80.100`.
Open `/grafana` in your browser — user `admin`, password `admin`.
Your local mTLS credentials are saved automatically to `~/.config/finch.json`.

> Need Let's Encrypt or a custom certificate? See
[TLS options](https://tschaefer.github.io/finch-docs/deployment/tls-options/).

## Enroll an Agent

Register a new agent with the Finch service and deploy it to a target machine:

```bash
finchctl agent register \
    --agent.hostname sparrow \
    --agent.logs.journal \
    10.19.80.100
```

The agent config is saved as `finch-agent.cfg` and contains all endpoints and
credentials.

```bash
finchctl agent deploy --agent.config finch-agent.cfg root@172.17.0.4
```

Alloy is installed and started on the target machine automatically.

> Want to collect Docker logs, log files, metrics, or profiles? See
[Agent options](https://tschaefer.github.io/finch-docs/agent/options/).

## Open the Dashboard

```bash
finchctl service dashboard --web --permission.session-timeout 1800 10.19.80.100
```

The dashboard opens in your browser with a fresh session token.

## What's Next

- [TLS options](https://tschaefer.github.io/finch-docs/deployment/tls-options/) - Let's Encrypt, custom certificates
- [Agent options](https://tschaefer.github.io/finch-docs/agent/options/) - Docker logs, file logs, metrics, profiles, labels
- [Managing agents](https://tschaefer.github.io/finch-docs/agent/manage/) - list, describe, edit, deregister
- [Token renewal](https://tschaefer.github.io/finch-docs/agent/token-renewal/) - refreshing agent credentials before expiry
- [Security model](https://tschaefer.github.io/finch-docs/security/) - how Finch handles auth, rotation, and recovery
- [Windows agents](https://tschaefer.github.io/finch-docs/agent/windows/) - enrolling agents on Windows

## Contributing

Fork the repository and submit a pull request. For major changes, open an issue
first to discuss your proposal.

## License

This project is licensed under the [MIT License](LICENSE).
