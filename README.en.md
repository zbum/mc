# mc - SSH Connection Manager

A CLI tool to quickly select and connect to SSH hosts from `~/.ssh/config` using a fuzzy finder.

## Demo

![demo](./demo.gif)

## Installation

```bash
go install
```

or

```bash
make build
```

## Usage

### Basic Usage

```bash
mc
```

When the host list appears:
- Type to search
- Use arrow keys to navigate
- Press Enter to select
- Press ESC or Ctrl+C to cancel

### With Initial Query

```bash
mc prod      # Start filtered with "prod"
mc web api   # Start filtered with "web api"
```

## SSH Config Example

`~/.ssh/config`:

```
# Production server
Host prod-web
    HostName web.prod.example.com
    User admin
    Port 22
    IdentityFile ~/.ssh/prod_key

# Development server
Host dev-web
    HostName web.dev.example.com
    User developer
```

## Debug Mode

For troubleshooting connection issues:

```bash
MC_DEBUG=1 mc
```

## Keyboard Shortcuts

| Key | Action |
|---|---|
| Enter | Connect to selected host |
| ESC / Ctrl+C | Cancel |
| Arrow Up/Down | Navigate hosts |
| Typing | Filter hosts |

## Authentication Order

1. `IdentityFile` specified in SSH config
2. SSH Agent
3. Default key files (`~/.ssh/id_ed25519`, `id_rsa`, etc.)
4. Password prompt

## License

MIT
