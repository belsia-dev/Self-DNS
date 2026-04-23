# SelfDNS

**A production-ready, privacy-first local DNS server with a native desktop Control Center.**

SelfDNS runs entirely on your machine (`127.0.0.1:53`) — it is never exposed to the network.
All upstream queries use **DNS-over-TLS** (Cloudflare, Google, Quad9, with automatic fallback
to your router). A built-in **ad/tracker blocker**, **custom hosts overrides**, **real-time
query log**, and a **security audit** panel are all accessible from the native desktop app.

---

## Screenshots

> _Control Center screenshots coming soon._
>
> ![Dashboard](docs/screenshots/dashboard.png)
> ![Query Log](docs/screenshots/querylog.png)
> ![Security](docs/screenshots/security.png)

---

## Install

### macOS / Linux (one-liner)

```bash
curl -fsSL https://github.com/belsia-dev/Self-DNS/releases/latest/download/install.sh | sudo bash
```

### Windows (PowerShell, run as Administrator)

```powershell
irm https://github.com/belsia-dev/Self-DNS/releases/latest/download/install.ps1 | iex
```

After installation, open **SelfDNS Control Center** from your Applications folder or Start Menu.

---

## What SelfDNS Does

| Feature | Detail |
|---|---|
| **DNS-over-TLS** | Upstream queries are encrypted; no plaintext DNS leaks |
| **Ad / tracker blocking** | NXDOMAIN for blocked domains; compatible with StevenBlack, OISD, etc. |
| **Custom hosts** | Override any domain → IP locally (e.g. `myapp.local → 127.0.0.1`) |
| **DNS rebinding protection** | Rejects private IPs returned for public domains |
| **DNSSEC** | DO bit set on all upstream queries |
| **Rate limiting** | Token-bucket per source IP (default 200 req/sec) |
| **In-memory cache** | TTL-respecting LRU cache (default 10 000 entries) |
| **Real-time stats** | QPS sparkline, top domains, cache hit rate, avg latency |
| **Security audit** | 9-point checklist with one-click auto-remediation |
| **Privacy mode** | Query logging off by default; domain names are anonymised |

---

## Upstream Fallback Chain

```
1.1.1.1:853 (Cloudflare DoT)
  └─ 8.8.8.8:853 (Google DoT)
       └─ 9.9.9.9:853 (Quad9 DoT)
            └─ System / router DNS (plain UDP, emergency fallback)
```

---

## Custom Hosts

Add `hosts` entries to `config.yaml` or use the **Hosts** tab in the Control Center:

```yaml
hosts:
  "myapp.local":    "127.0.0.1"
  "nas.home":       "192.168.1.50"
  "printer.local":  "192.168.1.100"
```

Overrides are answered authoritatively and bypass all upstream resolvers.

---

## Blocklist Usage

Enable blocking and add lists from the **Blocklist** tab, or in `config.yaml`:

```yaml
blocklist:
  enabled: true
  files:
    - /etc/selfdns/blocklists/stevenblack.hosts
  domains:
    - ads.example.com
    - tracker.example.net
```

Blocked domains receive an **NXDOMAIN** response instantly, with zero upstream traffic.

Tested compatible list formats:
- [StevenBlack/hosts](https://github.com/StevenBlack/hosts) (hosts format)
- [OISD](https://oisd.nl) (domain-per-line)
- [Hagezi DNS blocklists](https://github.com/hagezi/dns-blocklists)

---

## Security Model

- **Loopback-only**: both the DNS server (`127.0.0.1:53`) and the Control Center API
  (`127.0.0.1:5380`) bind exclusively to loopback — unreachable from any other machine.
- **No authentication**: not needed because only the local user can reach loopback.
- **DNS-over-TLS**: TLS 1.2 minimum, TLS 1.3 preferred on all upstream connections.
- **Packet validation**: malformed or empty DNS packets are dropped before processing.
- **Rate limiting**: token-bucket limiter (200 req/sec default) prevents local query floods.
- **DNS rebinding protection**: A/AAAA answers containing private IPs for public domains
  are replaced with NXDOMAIN.
- **Config permissions**: startup warns if `config.yaml` is world-readable (`chmod 600`).
- **Privacy mode**: query logging is off by default; domain names are anonymised.

---

## Building from Source

### Prerequisites

- Go 1.21+
- Node.js 18+ (for the Control Center frontend)
- [Wails v2](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Full Build (Recommended)

Use the included build script to compile everything and bundle the daemon into the app:

```bash
./build.sh
```

This will:
1. Build the `selfdns` DNS daemon binary
2. Build the Wails Control Center app
3. Copy the daemon into the app bundle (macOS `.app/Contents/Resources/`)

Then open the app — it will prompt for your password on first launch to start the DNS daemon on port 53.

### DNS Daemon Only

```bash
cd server
go build -o ../selfdns ./cmd/selfdns
sudo ./selfdns --config ~/.config/selfdns/config.yaml
```

### Control Center Only

```bash
cd ui
wails build
# Development mode with hot reload:
wails dev
```

---

## Uninstall

### macOS

```bash
sudo launchctl unload -w /Library/LaunchDaemons/io.selfdns.daemon.plist
sudo rm /Library/LaunchDaemons/io.selfdns.daemon.plist
sudo rm /usr/local/bin/selfdns /usr/local/bin/selfdns-app
sudo rm -rf /etc/selfdns /Applications/SelfDNS.app
# Restore DNS on each interface:
sudo networksetup -setdnsservers Wi-Fi "empty"
```

### Linux

```bash
sudo systemctl stop selfdns && sudo systemctl disable selfdns
sudo rm /etc/systemd/system/selfdns.service
sudo systemctl daemon-reload
sudo rm /usr/local/bin/selfdns /usr/local/bin/selfdns-app
sudo rm -rf /etc/selfdns
```

### Windows (run as Administrator)

```powershell
Stop-Service SelfDNS
sc.exe delete SelfDNS
Remove-Item -Recurse "C:\Program Files\SelfDNS"
Remove-Item -Recurse "$env:ProgramData\SelfDNS"
# Restore DNS: Set-DnsClientServerAddress -InterfaceIndex <idx> -ResetServerAddresses
```

---

## Configuration Reference

| Key | Default | Description |
|---|---|---|
| `listen` | `127.0.0.1:53` | DNS listen address (loopback only) |
| `api_listen` | `127.0.0.1:5380` | Control Center API address |
| `use_tls` | `true` | DNS-over-TLS for upstream queries |
| `log_queries` | `false` | Log full domain names (privacy risk) |
| `dns_rebinding_protection` | `true` | Reject private IPs for public domains |
| `dnssec` | `true` | Set DO bit on upstream queries |
| `upstream` | Cloudflare, Google, Quad9 | DoT upstream servers in fallback order |
| `cache.enabled` | `true` | Enable in-memory DNS cache |
| `cache.max_size` | `10000` | Maximum cached entries |
| `cache.min_ttl` | `60` | Minimum TTL floor (seconds) |
| `rate_limit.enabled` | `true` | Enable per-source rate limiting |
| `rate_limit.max_rps` | `200` | Max queries/sec per source IP |
| `blocklist.enabled` | `false` | Enable domain blocking |
| `blocklist.files` | `[]` | Paths to hosts-format blocklist files |
| `blocklist.domains` | `[]` | Manually blocked domains |
| `hosts` | `{}` | Custom domain → IP overrides |

---

## Control Center API

The HTTP API runs on `127.0.0.1:5380` and is consumed by the desktop app.
You can also call it directly from the terminal:

```bash
curl http://127.0.0.1:5380/api/status | jq
curl http://127.0.0.1:5380/api/stats  | jq
curl http://127.0.0.1:5380/api/queries | jq '.[0:5]'
curl -X POST http://127.0.0.1:5380/api/cache/flush
curl -X POST http://127.0.0.1:5380/api/blocklist/add \
     -H 'Content-Type: application/json' \
     -d '{"domain":"ads.example.com"}'
```

---

## Contributing

Contributions are welcome! Please read through the open issues before starting.

1. Fork the repository.
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Commit your changes with a clear message.
4. Open a Pull Request against `main`.

Please keep PRs focused — one feature or fix per PR. Add tests where possible.
Run `go vet ./...` and `go test ./...` before submitting.

---

## License

MIT — see [LICENSE](LICENSE).

Built with [miekg/dns](https://github.com/miekg/dns), [Wails](https://wails.io),
[React](https://react.dev), [Recharts](https://recharts.org), and [TailwindCSS](https://tailwindcss.com).
