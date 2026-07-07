# Contributing to Ptahcortex

Thank you for your interest in contributing to Ptahcortex! This document provides guidelines and information for contributors.

## Acknowledgments

Ptahcortex builds on top of excellent open-source projects:

- **[Lexa](https://github.com/anvia-hq/lexa)** — Fast local code intelligence for AI agents
  - Graph indexing, pattern search, dependency tracing
  - 80% token reduction vs baseline approaches
  - MCP server for seamless integration

- **[MCP Protocol](https://modelcontextprotocol.io/)** — Model Context Protocol for tool interoperability

- **[OTel](https://opentelemetry.io/)** — OpenTelemetry for observability

We are committed to contributing back to these projects and the open-source community.

## How to Contribute

### 1. Report Issues

Found a bug? Open an issue on [GitHub Issues](https://github.com/rhythmwave/ptahcortex/issues).

### 2. Suggest Features

Have an idea? Open a discussion on [GitHub Discussions](https://github.com/rhythmwave/ptahcortex/discussions).

### 3. Submit Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### 4. Contribute to Dependencies

We encourage contributions to the projects we depend on:

- **Lexa**: [github.com/anvia-hq/lexa](https://github.com/anvia-hq/lexa)
- **MCP Protocol**: [modelcontextprotocol.io](https://modelcontextprotocol.io/)
- **OTel**: [opentelemetry.io](https://opentelemetry.io/)

## Development Setup

### Prerequisites

- Go 1.26.2+
- Lexa MCP server (optional, for code intelligence)
- Access to LLM API (OpenAI-compatible)

### Local Development

```bash
# Clone the repository
git clone https://github.com/rhythmwave/ptahcortex.git
cd ptahcortex

# Build
go build -o ptahcortex ./cmd/agentkit

# Run tests
go test ./...

# Run with Lexa
./ptahcortex --config configs/code-reviewer.yaml --task "Review code"
```

### Project Structure

```
ptahcortex/
├── cmd/agentkit/          # CLI entrypoint
├── internal/
│   ├── agent/             # Agent loop (plan/execute/reflect)
│   ├── context/           # Context manager (call-aware assembly)
│   ├── mcp/               # MCP client (stdio JSON-RPC)
│   ├── llm/               # LLM provider interface
│   ├── otel/              # Observability (traces, metrics)
│   └── tools/             # Tool execution engine
├── configs/               # Agent configurations
├── docs/                  # Documentation
└── benchmark/             # Benchmark comparisons
```

## Coding Standards

### Go Style

- Follow [Effective Go](https://go.dev/doc/effective-go) guidelines
- Use `gofmt` for formatting
- Use `go vet` for static analysis
- Write tests for new functionality

### Commit Messages

- Use conventional commits format
- Examples: `feat:`, `fix:`, `docs:`, `test:`, `chore:`

### Documentation

- Update README.md for new features
- Add godoc comments for public APIs
- Update examples if applicable

## Community Guidelines

### Code of Conduct

We follow the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).

### Getting Help

- **GitHub Discussions**: Ask questions, share ideas
- **GitHub Issues**: Report bugs, request features
- **Discord**: Join our community (if available)

## Recognition

Contributors will be recognized in:

- README.md contributors section
- Release notes
- Annual contributor appreciation

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

**Thank you for contributing to Ptahcortex!**
