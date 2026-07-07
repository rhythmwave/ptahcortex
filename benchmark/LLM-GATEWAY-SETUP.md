# LLM Gateway Setup Complete

## Gateway Configuration

- **Port:** 8082
- **Endpoint:** http://localhost:8082
- **Models:** mimo-v2.5 (OpenAI), mimo-v2.5-anthropic (Anthropic)
- **Logging:** Enabled (usage.db)

## Usage

### Claude Code
```bash
export ANTHROPIC_BASE_URL="http://localhost:8082"
export ANTHROPIC_API_KEY="tp-s7emr8e5k8enrm5jnz2gs82d3hvhoq22vvt0sw110sipuhfm"

claude --permission-mode bypassPermissions --model mimo-v2.5 --print "Your task"
```

### Ptahcortex
```yaml
llm:
  provider: openai
  model: mimo-v2.5
  base_url: http://localhost:8082
  api_key: tp-s7emr8e5k8enrm5jnz2gs82d3hvhoq22vvt0sw110sipuhfm
```

## Traffic Logged

The gateway logs all requests:
- **OpenAI endpoint:** /v1/chat/completions
- **Anthropic endpoint:** /v1/messages
- **Usage tracking:** usage.db (SQLite)

## Next Steps

1. **Check usage database** — Query usage.db for token usage
2. **Monitor traffic** — Watch gateway logs
3. **Compare usage** — Ptahcortex vs Claude Code

## Gateway Running

```
Gateway PID: 3990789
Port: 8082
Status: OK
```
