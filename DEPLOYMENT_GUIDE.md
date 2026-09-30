# PMail Production Deployment & Operator Guide (AI-Executable)

This document provides a deterministic, step-by-step Standard Operating Procedure (SOP) for any AI agent or systems engineer to compile, verify, and deploy PMail to any Linux VPS server without interactive prompts or configuration guesswork.

---

## 1. System Architecture Overview

```
 Internet (Port 80/443) ──► Caddy / Reverse Proxy ──► PMail Web/API (127.0.0.1:8080)
 Internet (Port 25)     ──────────────────────────► PMail SMTP Inbound
 Internet (Port 465)    ──────────────────────────► PMail SMTP SSL Outbound/Client
 Internet (Port 587)    ──────────────────────────► PMail SMTP STARTTLS
 Internet (Port 993)    ──────────────────────────► PMail IMAP TLS
 Internet (Port 995)    ──────────────────────────► PMail POP3 TLS
```

- **Target Operating System**: Linux (Ubuntu 22.04+, Debian 11+, RHEL/Rocky, Alpine)
- **Architecture**: `linux/amd64` (standard x86_64) or `linux/arm64` (AWS Graviton, Oracle ARM)
- **Database**: Pure Go embedded SQLite (`modernc.org/sqlite`), zero external DBMS needed.
- **Binary Model**: 100% self-contained static executable with embedded frontend (`//go:embed dist/*`).

---

## 2. Automated Lifecycle Manager (`deployer.py`)

The project includes `deployer.py` (and alias `lightmail_deployer.py`) that manages the complete server lifecycle with both an **Interactive Wizard** and a **CLI Argument Mode**.

### Mode 1: Interactive Wizard (Recommended for humans)
Simply run without arguments:
```bash
python deployer.py
# or
python lightmail_deployer.py
```
The wizard prompts for:
1. Server Host / IP
2. SSH Port (default 22)
3. SSH User (root / ashu / etc.) & Password / Key
4. Action:
   - `[1] Update`: Builds frontend (`bun run build`), compiles Go binary, gzips for fast SFTP transfer, atomic swap, and service restart.
   - `[2] Install`: Fresh server install (creates `/opt/pmail`, dependencies, systemd unit, capabilities, domain & admin setup).
   - `[3] Status`: Service health, listening ports, disk usage.
   - `[4] Restart`: Gracefully restarts `pmail.service`.
   - `[5] Logs`: Views recent journalctl output.
   - `[6] Uninstall`: Dismantles service, safely archives database and configs to `/tmp/pmail_backup_*.tar.gz`.

### Mode 2: Non-Interactive CLI Mode (Recommended for AI Agents & CI/CD)
```bash
# Update existing server
python deployer.py --host mail.yourdomain.com --user root --password "<PASS>" --action update

# Check status
python deployer.py --host mail.yourdomain.com --user root --password "<PASS>" --action status

# View live logs
python deployer.py --host mail.yourdomain.com --user root --password "<PASS>" --action logs

# Fresh install on a clean Linux VPS
python deployer.py --host <IP> --user root --password "<PASS>" --action install --domain mydomain.com --admin-user admin --admin-pass "<PASS>" --http-port 80
```

---

## 3. Webmail Accessibility & Gmail-Style Shortcuts

The Svelte frontend is optimized for screen reader users (NVDA, JAWS, VoiceOver, Orca):

- **Checkbox Screen Reader Navigation (`x` in NVDA/JAWS)**:
  - Formatted as: `[Read/Unread], [Sender], [Subject], [Snippet up to 250 chars], [Date]`.
  - NVDA automatically announces: `Check box not checked [or checked], Unread, Swarup BarAl, hi, body snippet..., 30 Sep`.
- **Keyboard Shortcuts**:
  - `j` / `↓`: Focus next email and announce details via polite live region.
  - `k` / `↑`: Focus previous email and announce details.
  - `x` / `X`: Toggle selection of the currently focused email with audible feedback.
  - `t` / `T`: Toggle Read/Unread status for focused or selected emails (instantly updates UI and backend database).
  - `Enter` / `o`: Open email reading view.
  - `#`: Delete / move to Trash.

---

## 2. Pre-Deployment Environment Inspection

Before performing any build or deployment, inspect the remote host over SSH:

```bash
# 1. Identify Target OS and Architecture
uname -m
cat /etc/os-release

# 2. Check if PMail is already running
systemctl status pmail || ps aux | grep -i pmail

# 3. Check port bindings (verify ports 25, 465, 587, 993, 8080)
ss -tulnp | grep -E ':(25|465|587|993|8080)\b'

# 4. Check Reverse Proxy (e.g. Caddy)
cat /etc/caddy/Caddyfile 2>/dev/null || cat /etc/nginx/sites-enabled/* 2>/dev/null
```

---

## 3. Local Compilation Protocol

### Step A: Build Frontend (Svelte)
```bash
cd fe
yarn install # or bun install
yarn build   # outputs to fe/dist/
```

### Step B: Sync Frontend Assets to Go Embed Path
```bash
# Ensure server/listen/http_server/dist contains fresh frontend build
rm -rf server/listen/http_server/dist
cp -r fe/dist server/listen/http_server/
```

### Step C: Compile Cross-Platform Static Binary
Run from the repository root:
```bash
cd server

# Target: Linux AMD64
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

GIT_HASH=$(git rev-parse HEAD 2>/dev/null || echo "release")
BUILD_TIME=$(date -u +"%Y-%m-%d %H:%M:%S")

go build -ldflags "-s -w \
  -X 'main.gitHash=${GIT_HASH}' \
  -X 'main.buildTime=${BUILD_TIME}'" \
  -o pmail_linux_amd64 main.go
```

Verify binary:
```bash
file pmail_linux_amd64
# Must output: ELF 64-bit LSB executable, x86-64, statically linked, stripped
```

---

## 4. Production Deployment Runbook

### Step 1: Transfer New Binary to Server
Upload the compiled binary to a temporary staging path (e.g. `/tmp/pmail.new` or `/home/$USER/pmail.new`):
```bash
# Example via scp/sftp
scp server/pmail_linux_amd64 root@mail.yourdomain.com:/tmp/pmail.new
```

### Step 2: Create a Full Timestamped Backup
**Never deploy without backing up the live database and certificates:**
```bash
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
sudo cp -a /opt/pmail /opt/pmail.backup-$TIMESTAMP
echo "Backup saved to: /opt/pmail.backup-$TIMESTAMP"
```

### Step 3: Replace Binary & Configure Privileges
PMail needs `CAP_NET_BIND_SERVICE` to listen on standard email ports (25, 465, 587, 993) under a non-root system user (`pmail`).

```bash
# 1. Stop existing service
sudo systemctl stop pmail

# 2. Swap binary
sudo mv /opt/pmail/pmail /opt/pmail/pmail.prev 2>/dev/null || true
sudo cp /tmp/pmail.new /opt/pmail/pmail
sudo rm -f /tmp/pmail.new

# 3. Set file ownership and permissions
sudo chown pmail:pmail /opt/pmail/pmail
sudo chmod 755 /opt/pmail/pmail

# 4. Set Linux Capabilities (allows low port binding without root)
sudo setcap 'cap_net_bind_service=+ep' /opt/pmail/pmail
getcap /opt/pmail/pmail
```

### Step 4: Restart & Verify Service
```bash
# 1. Start PMail
sudo systemctl start pmail

# 2. Verify systemd status
sudo systemctl status pmail --no-pager

# 3. Check live logs for migrations and errors
sudo journalctl -u pmail -n 30 --no-pager

# 4. Verify API health check
curl -s http://127.0.0.1:8080/api/ping
# Expected output: {"errorNo":0,"errorMsg":"","data":"pong"}
```

---

## 5. Systemd Service Unit Template

File location: `/etc/systemd/system/pmail.service`

```ini
[Unit]
Description=PMail Lightweight Mail Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=pmail
Group=pmail
WorkingDirectory=/opt/pmail
ExecStart=/opt/pmail/pmail -p 8080
Restart=on-failure
RestartSec=5
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

---

## 6. Reverse Proxy Configuration (Caddy & Nginx)

### Caddyfile Block (Recommended)
```caddy
wmail.yourdomain.com {
    reverse_proxy 127.0.0.1:8080
}
```

### Nginx Block (Alternative)
```nginx
server {
    server_name wmail.yourdomain.com;
    listen 80;
    listen 443 ssl http2;

    ssl_certificate /etc/letsencrypt/live/wmail.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/wmail.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 7. Required DNS Records for 100% Deliverability

| Record Type | Host / Name | Value / Destination | Purpose |
|---|---|---|---|
| **A** | `mail.yourdomain.com` | `<VPS_IPv4_Address>` | Mail server hostname |
| **AAAA** | `mail.yourdomain.com` | `<VPS_IPv6_Address>` | Mail server IPv6 |
| **MX** | `yourdomain.com` | `10 mail.yourdomain.com` | Mail routing |
| **TXT (SPF)**| `yourdomain.com` | `v=spf1 mx ip4:<VPS_IP> ip6:<VPS_IPv6> ~all` | Sender authorization |
| **TXT (DKIM)**| `default._domainkey.yourdomain.com` | `v=DKIM1; k=rsa; p=<PUBLIC_KEY>` | Cryptographic DKIM signature |
| **TXT (DMARC)**| `_dmarc.yourdomain.com` | `v=DMARC1; p=quarantine; rua=mailto:admin@yourdomain.com` | Alignment policy |
| **PTR (rDNS)** | `<VPS_IP>` (in VPS hosting panel) | `mail.yourdomain.com` | Reverse DNS lookup |

---

## 8. Rollback Procedure

If any issue arises during a new version deployment:
```bash
# 1. Stop service
sudo systemctl stop pmail

# 2. Restore previous binary
sudo cp /opt/pmail/pmail.prev /opt/pmail/pmail
sudo setcap 'cap_net_bind_service=+ep' /opt/pmail/pmail

# 3. (Optional) If database needs restoring:
# sudo cp /opt/pmail.backup-<TIMESTAMP>/config/pmail.db /opt/pmail/config/pmail.db

# 4. Restart service
sudo systemctl start pmail
sudo systemctl status pmail
```
