# Ptahcortex Tool Architecture

## Tool Categories

### Always Available (Basic OS)
- `read_file` — Read file contents
- `write_file` — Write to files
- `exec` — Execute shell commands
- `list_files` — List directory contents

### Optional (Lexa Code Intelligence)
- `search` — Search code patterns
- `outline` — Get file structure
- `audit` — Architecture audit

## Configuration

```yaml
# Enable Lexa
--lexa

# Disable Lexa (default)
(no flag)
```

## Tool Selection Strategy

### When Lexa is Enabled
- **PREFER** `search`, `outline`, `audit` for code analysis
- **USE** `exec`, `list_files` for OS operations
- **USE** `read_file`, `write_file` for file I/O

### When Lexa is Disabled
- **USE** `exec` with `grep`, `find` for code search
- **USE** `list_files` for directory listing
- **USE** `read_file`, `write_file` for file I/O

## Prompt Engineering

```
IMPORTANT: For code search and analysis, use Lexa tools 
(search, outline, audit) instead of basic tools (exec, list_files).
```

## Benefits

1. **Flexibility** — Works with or without Lexa
2. **Fallback** — Basic tools always available
3. **Choice** — LLM can choose best tool for task
4. **Configuration** — CLI flag controls tool availability
