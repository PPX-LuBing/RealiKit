# RealiKit

A terminal UI tool for scanning and verifying Reality protocol target domains.

Combines [RealiTLScanner](https://github.com/XTLS/RealiTLScanner) (TLS scanner) and [RealityChecker](https://github.com/V2RaySSR/RealityChecker) (deep verification) into a single, easy-to-use TUI.

## Quick Start

### One-liner download & run

```bash
# Linux x86_64
wget -O realikit https://github.com/PPX-LuBing/RealiKit/releases/latest/download/realikit-linux-amd64 && chmod +x realikit && ./realikit

# macOS ARM (M1/M2)
curl -L -o realikit https://github.com/PPX-LuBing/RealiKit/releases/latest/download/realikit-darwin-arm64 && chmod +x realikit && ./realikit

# macOS Intel
curl -L -o realikit https://github.com/PPX-LuBing/RealiKit/releases/latest/download/realikit-darwin-amd64 && chmod +x realikit && ./realikit
```

### Or build from source

```bash
git clone https://github.com/PPX-LuBing/RealiKit.git
cd RealiKit
go build -o realikit ./cmd/reality-tui/
./realikit
```

## Features

### Scan Profiles (press `m` to cycle)

| Profile | Threads | Timeout | Verify | Use Case |
|---------|---------|---------|--------|----------|
| **Standard** | 10 | 5s | ✅ | Daily use (default) |
| **Quick** | 20 | 5s | ❌ | Bulk scanning |
| **Deep** | 5 | 10s | ✅ | Precision check |
| **Mass** | 50 | 3s | ❌ | Large CIDR sweep |

### Input Methods

- **IP/CIDR/domain** — `1.2.3.4` / `1.2.3.0/24` / `example.com`
- **From file** — one target per line, press `c` to verify CSV directly
- **From URL** — auto-extract domains from a webpage

### Detection Dimensions

| Check | Description |
|-------|-------------|
| TLS Protocol | TLS 1.3, h2 ALPN, X25519 |
| HTTP Reachability | Status code classification |
| CDN Detection | 3-level confidence (high/medium/low) via CNAME, HTTP headers, cert issuer |
| Certificate | Validity, SNI match, issuer |
| Blocked Check | GFW keyword matching |
| GeoIP | Country code |
| Hot Website | Built-in list to avoid high-risk targets |

### Key Bindings

| Key | Action |
|-----|--------|
| `s` | Start scan |
| `m` | Switch profile |
| `↑` `↓` | Navigate input/rows |
| `i` | Toggle IPv6 |
| `v` | Toggle verbose |
| `c` | Verify CSV directly |
| `t` | Manual verify mode |
| `S` | Save settings |
| `enter` | View details |
| `space` | Toggle favorite |
| `d` | Deduplicate by cert domain |
| `/` | Search/filter |
| `1` `2` | Sort by score / latency |
| `3` `4` `5` | Filter by min stars (`0` to clear) |
| `f` | Toggle show unfeasible |
| `F` | Favorites only |
| `g` | Generate Reality config with x25519 keys |
| `e` | Export CSV |
| `E` | Export favorites only |
| `w` | Export HTML report |
| `esc` | Go back |
| `q` | Quit (confirm if data exists) |

### Recommended Workflow

1. Select **Standard** profile
2. Enter your VPS IP or CIDR (e.g. `1.2.3.0/24`)
3. Press `s` to start
4. TLS scan → auto deep verify → 5-star rating
5. Filter by min stars (`4` or `5`)
6. Select a domain, press `enter` for details, `g` to generate Reality config

## Build

```bash
go build -o realikit ./cmd/reality-tui/
```

Requires Go 1.22+.

## License

MIT
