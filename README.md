# Lightmail ⚡

> **The World's Lightest, Most Streamlined Production Email Server & Webmail.**  
> Single Binary • ~8MB Compressed (~16MB Static) • ~4.8MB RAM • Pure SQLite • Svelte 5 Webmail • Multi-Domain DKIM • 1-Click Cloudflare DNS • IPv6-Native • Zero External Daemons

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Frontend](https://img.shields.io/badge/Frontend-Svelte%205%20%2B%20Vite-FF3E00?logo=svelte)](https://svelte.dev/)
[![RAM Footprint](https://img.shields.io/badge/Memory-~4.8MB%20RAM-success)](https://github.com/ashu-choudhury/lightmail)
[![Binary Size](https://img.shields.io/badge/Binary-~8MB%20(compressed)-informational)](https://github.com/ashu-choudhury/lightmail/releases)
[![Build Status](https://github.com/ashu-choudhury/lightmail/actions/workflows/release.yml/badge.svg)](https://github.com/ashu-choudhury/lightmail/actions/workflows/release.yml)

---

## 🌟 Roots & Honest Attribution

**Lightmail** began as an extensive, independent refactor and evolution of [**Jinnrry/PMail**](https://github.com/Jinnrry/PMail). We give our heartfelt thanks and 100% credit to [@Jinnrry](https://github.com/Jinnrry) and the original contributors for building the initial Go mail foundation.

### What Was Re-Engineered & Stripped Down?
While PMail was already lighter than traditional setups, it still contained unnecessary bulk. In **Lightmail**, we stripped down the codebase as much as technically possible to create the world's most streamlined mail server:
- **Stripped Heavy Dependencies**: Removed legacy Chinese push notifications (WeChat Push), deprecated debug hooks, and redundant daemons.
- **Rewrote Frontend with Svelte 5**: Replaced the entire heavy Vue 3 + Element Plus frontend stack with an ultra-responsive, zero-dependency **Svelte 5** webmail interface (< 150KB gzip).
- **100% Pure SQLite**: Eliminated external database servers (MySQL/PostgreSQL/Redis). All accounts, emails, settings, and routing rules live in a single SQLite database (`/opt/pmail/config/pmail.db`) with automatic schema migrations.
- **1-Click Cloudflare DNS Automation**: Complete RFC & Cloudflare API v4 integration supporting both API Tokens and Global API Keys with permanent SQLite credential persistence.
- **Live Direct DNS Health Verification**: Integrated custom recursive DNS resolvers querying Cloudflare (`1.1.1.1`) and Google (`8.8.8.8`) directly, eliminating 30-minute negative DNS caching delays.
- **IPv6-Native & NAT64/DNS64 Compatibility**: Tested and verified on strictly IPv6-only VPS environments (e.g. Proxmox LXC containers) with dynamic IPv6 SPF generation and fallback DNS resolvers.
- **Continuous Automated Releases**: Pre-compiled multi-platform binaries are built and published automatically on every push via GitHub Actions.

---

## ⚡ 1-Command Linux Quick Install

You don't need to compile anything or install complex dependencies. Run this single command on any clean Linux VPS (Ubuntu, Debian, CentOS, AlmaLinux, Alpine, Arch):

```bash
curl -fsSL https://raw.githubusercontent.com/ashu-choudhury/lightmail/master/install.sh | sudo bash
```

### What This Command Does Automatically:
1. Detects your CPU architecture (`x86_64` or `arm64`).
2. Installs required system packages (`curl`, `tar`, `gzip`, `libcap`).
3. Downloads the latest pre-compiled Lightmail binary (~8MB compressed).
4. Installs the binary into `/opt/pmail/pmail`.
5. Grants low-port network binding capabilities (`cap_net_bind_service=+ep`).
6. Creates and starts the `systemd` service (`pmail.service`).
7. Prints your server's web setup URL: `http://<YOUR-SERVER-IP>:8080`.

---

## 🛡️ Recommended Reverse Proxy: Caddy

To access your Webmail interface with clean, automated HTTPS (`https://mail.yourdomain.com`), **we strongly recommend [Caddy](https://caddyserver.com)**.

### Why Caddy?
- **Zero-Touch Automatic SSL**: Automatically provisions and renews Let's Encrypt / ZeroSSL certificates without needing certbot or cron jobs.
- **Dedicated Port Separation**: Caddy handles web traffic (ports 80 & 443), while Lightmail handles email protocols (SMTP 25, 465, 587; IMAP 993; POP3 110, 995).
- **HTTP/2 & HTTP/3 Out-of-the-Box**: Blazing fast webmail loading speeds.

### How to Configure Caddy (Takes 30 Seconds):
1. Install Caddy:
   ```bash
   sudo apt install -y caddy   # Debian/Ubuntu
   # or dnf install -y caddy  # RHEL/AlmaLinux
   ```
2. Open `/etc/caddy/Caddyfile` and add:
   ```caddyfile
   mail.yourdomain.com {
       reverse_proxy 127.0.0.1:8080
   }
   ```
3. Restart Caddy:
   ```bash
   sudo systemctl restart caddy
   ```
Your webmail is now live at `https://mail.yourdomain.com` with valid SSL!

---

## 📊 Footprint: Lightmail vs Traditional Stacks

| Metric | Traditional Stacks (Mailcow, iRedMail, etc.) | Lightmail ⚡ |
| :--- | :--- | :--- |
| **Idle Memory (RAM)** | 2,000 MB – 4,000 MB | **~4.8 MB** |
| **Storage / Binary Size** | Multiple GBs of Docker images | **~8 MB** (compressed) / ~16MB (static) |
| **Components** | Postfix, Dovecot, SpamAssassin, Redis, MySQL, Nginx, PHP | **Single Go Binary** |
| **Database** | MySQL / MariaDB / PostgreSQL Server | **Pure SQLite (`pmail.db`)** |
| **DNS Configuration** | Manual copy-pasting of DNS strings | **1-Click Cloudflare DNS Sync** |
| **IPv6 Support** | Often broken or requires complex dual-stack NAT | **IPv6-Native with Dynamic SPF** |
| **Setup Time** | 30 – 60 minutes | **1 minute** |

---

## ✨ Full Feature Overview

```
                                  INTERNET
                                      │
            ┌─────────────────────────┼─────────────────────────┐
            │                         │                         │
      Inbound SMTP              Webmail HTTP               Client IMAP/POP3
      (Port 25/465)             (Port 80/8080)               (Port 993/995)
            │                         │                         │
            ▼                         ▼                         ▼
   ┌─────────────────────────────────────────────────────────────────┐
   │                      LIGHTMAIL ENGINE                           │
   │                                                                 │
   │  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐  │
   │  │   SMTP Server    │  │   HTTP Server    │  │  IMAP / POP3  │  │
   │  │  (SPF/DKIM Verify│  │  (Svelte Webmail │  │    Server     │  │
   │  │   & Deliver)     │  │   & Admin API)   │  │ (Client Sync) │  │
   │  └─────────┬────────┘  └─────────┬────────┘  └───────┬───────┘  │
   │            │                     │                   │          │
   │            └──────────────┐      │      ┌────────────┘          │
   │                           ▼      ▼      ▼                       │
   │                   ┌────────────────────────────┐                │
   │                   │    Pure SQLite Database    │                │
   │                   │     (/opt/pmail/config)    │                │
   │                   │  Users • Mails • Settings  │                │
   │                   └────────────────────────────┘                │
   │                                  │                              │
   │  ┌───────────────────────────────┴───────────────────────────┐  │
   │  │                  Outbound Mail Dispatcher                 │  │
   │  │   • Resolves Recipient MX via Direct DNS                  │  │
   │  │   • Signs with Domain's 2048-bit Private RSA DKIM Key     │  │
   │  │   • Transmits over Outbound Port 25 with STARTTLS         │  │
   │  └───────────────────────────────────────────────────────────┘  │
   └─────────────────────────────────────────────────────────────────┘
```

### 1. Multi-Domain & 1-Click Cloudflare Provisioning
- Add unlimited primary and secondary domains.
- Automatically generates isolated 2048-bit RSA DKIM keypairs per domain.
- Synchronizes **MX**, **SPF** (`v=spf1 ip6:<ip> a mx ~all`), **DKIM** (`default._domainkey`), **DMARC**, and **AAAA/CNAME** records in one click via Cloudflare API v4.
- Automatically saves API credentials to SQLite so you never have to re-enter them.

### 2. Live Real-Time DNS Health Checker
- Bypasses local recursive resolver negative caching.
- Queries Cloudflare and Google public authoritative DNS over IPv6 and IPv4.
- Displays immediate status: **Host IP Resolves**, **MX Routing Active**, **SPF Configured**, **DKIM Published**, and **DMARC Active**.

### 3. Full Protocol Mail Stack
- **SMTP**: Ports 25, 465 (Implicit TLS), and 587 (STARTTLS).
- **IMAP4rev1**: Port 993 (SSL/TLS).
- **POP3**: Ports 110 and 995 (SSL/TLS).
- **10/10 Deliverability**: Meets all Gmail, Yahoo, and Outlook deliverability requirements out-of-the-box.

---

## 📱 Email Client Configuration

Lightmail works seamlessly with all desktop and mobile mail clients (Thunderbird, Apple Mail, Outlook, iOS Mail, K-9 Mail, Fairemail):

| Protocol | Server Hostname | Port | Encryption |
| :--- | :--- | :--- | :--- |
| **IMAP (Incoming)** | `smtp.yourdomain.com` (or `imap.yourdomain.com`) | **993** | SSL / TLS |
| **POP3 (Incoming)** | `smtp.yourdomain.com` (or `pop.yourdomain.com`) | **995** | SSL / TLS |
| **SMTP (Outgoing)** | `smtp.yourdomain.com` | **465** | SSL / TLS |
| **SMTP (Submission)** | `smtp.yourdomain.com` | **587** | STARTTLS |

- **Username**: Your full email address (e.g. `user@yourdomain.com`)
- **Password**: Your mailbox password

---

## 🛠️ How to Build from Source (Optional)

If you prefer building the project yourself rather than downloading the pre-compiled GitHub releases:

### 1. Prerequisites
- **Go**: 1.22+ installed
- **Bun** (or Node.js / npm): For compiling the Svelte webmail

### 2. Build Steps
```bash
# Clone the repository
git clone https://github.com/ashu-choudhury/lightmail.git
cd lightmail

# Step 1: Build the modern Svelte frontend
cd fe-svelte
bun install
bun run build
cd ..

# Step 2: Copy built assets to embedded server location
cp -r fe-svelte/dist/* server/listen/http_server/dist/

# Step 3: Compile the stripped static binary (~16MB static, ~8MB compressed)
cd server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../lightmail_linux_amd64 main.go
cd ..

# Step 4: Run initial setup
./lightmail_linux_amd64 -p 8080
```

---

## 🚀 Lifecycle Management via `deployer.py`

Lightmail includes a standalone Python deployment script [`deployer.py`](deployer.py) for remote VPS management:

```bash
# Deploy / Update on remote VPS with zero downtime
python deployer.py --host mail.yourdomain.com --user root --password "your_password" --action update

# Check remote service health, ports, and RAM
python deployer.py --host mail.yourdomain.com --user root --password "your_password" --action status

# View recent service logs
python deployer.py --host mail.yourdomain.com --user root --password "your_password" --action logs
```

---

## 📄 License & Legal Notice

Lightmail is distributed under the [MIT License](LICENSE).  
Original PMail implementation Copyright © Jinnrry.  
Enhancements, Svelte webmail, Cloudflare automation, SQLite migration, and Lightmail distribution Copyright © Ashu Choudhury.
