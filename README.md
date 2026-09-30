# Lightmail ⚡

> **The World's Lightest, Most Streamlined Production Email Server & Webmail.**  
> Single binary • Pure SQLite • Modern Svelte Webmail • Multi-Domain • 1-Click Cloudflare DNS • IPv6-Native

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Frontend](https://img.shields.io/badge/Frontend-Svelte%205%20%2B%20Vite-FF3E00?logo=svelte)](https://svelte.dev/)
[![Memory Footprint](https://img.shields.io/badge/RAM-~5MB-success)](https://github.com/ashu-choudhury/PMail)

---

## 🌟 Acknowledgements & Roots

**Lightmail** is an advanced, ultra-streamlined fork of [**Jinnrry/PMail**](https://github.com/Jinnrry/PMail). We express our deepest gratitude and credit to [@Jinnrry](https://github.com/Jinnrry) and all original contributors for laying down the foundation of a minimalistic email server in Go.

This repository represents an extensive evolution of that vision: completely trimmed of bloat, re-engineered for **pure SQLite**, equipped with a modern **Svelte 5** webmail interface, full **multi-domain DKIM management**, **1-Click Cloudflare DNS synchronization**, resilient **IPv6/DNS64 networking**, and an automated **zero-downtime Python VPS lifecycle deployer**.

---

## 💡 Why Lightmail?

Most self-hosted email servers (Mailcow, Mail-in-a-Box, iRedMail, Docker-mailserver) require complex multi-container stacks running Postfix, Dovecot, SpamAssassin, ClamAV, Redis, MySQL, and Nginx—consuming **2GB to 4GB of RAM** just to idle.

**Lightmail replaces that entire stack with a single ~16MB binary that runs smoothly in just 5MB of RAM.**

```
Traditional Mail Stacks (~2.5GB RAM)         Lightmail (~5MB RAM)
┌─────────────────────────────────┐         ┌──────────────────────────────┐
│  Postfix + Dovecot + Rspamd     │         │                              │
│  Redis + MySQL/PostgreSQL       │  ====>  │  Single Native Go Binary     │
│  Nginx + PHP-FPM + Roundcube    │         │  Embedded Svelte SPA         │
│  ClamAV + Unbound + Cron        │         │  Pure SQLite Database        │
└─────────────────────────────────┘         └──────────────────────────────┘
```

---

## ✨ Features at a Glance

### 🚀 Ultra-Light & Self-Contained
- **Single Static Binary**: Built with Go; embeds all frontend assets into the binary.
- **Pure SQLite Architecture**: No database server to install or maintain. All emails, users, rules, and settings are stored in a single, robust `/opt/pmail/config/pmail.db` SQLite database with automatic schema migrations.
- **Minimal RAM Footprint**: Idles at **~4.8MB RAM**, making it perfect for budget VPS instances ($1/month or free-tier instances).

### 🌐 Multi-Domain & Complete DNS Automation
- **Unlimited Mail Domains**: Host multiple apex domains (e.g., `yourdomain.com`) and subdomains (e.g., `mail.yourdomain.com`) simultaneously on one server.
- **Automatic 2048-bit RSA DKIM**: Generates isolated cryptographic DKIM keypairs per domain upon creation.
- **1-Click Cloudflare DNS Sync**:
  - Automatically provisions **MX**, **SPF**, **DKIM** (`default._domainkey`), **DMARC**, and **AAAA/CNAME** records directly to Cloudflare via official API v4.
  - Supports modern **Cloudflare API Tokens** (`Authorization: Bearer`) and legacy **Global API Keys** (`X-Auth-Key` / `X-Auth-Email`).
  - **Permanent SQLite Credential Persistence**: Enter credentials once; they are safely stored in the database indefinitely.
- **Live DNS Health Verification**:
  - Real-time DNS status engine with direct queries to global public resolvers (`1.1.1.1` and `8.8.8.8`).
  - Instant live badges for **Host IP**, **MX Routing**, **SPF**, **DKIM**, and **DMARC**. Zero negative-caching lag.

### 🔒 Modern Email Protocol Support
- **Inbound & Outbound SMTP**: Full support on Ports **25**, **465** (Implicit TLS), and **587** (STARTTLS).
- **IMAP4rev1**: Port **993** (SSL/TLS) for syncing with mobile and desktop mail clients.
- **POP3**: Ports **110** (Plain/STARTTLS) and **995** (SSL/TLS).
- **10/10 Mail-Tester Deliverability**: Dynamic IPv6/IPv4 SPF alignment and RFC-compliant DKIM signatures ensure emails land straight in Gmail/Outlook inboxes without spam penalties.

### 🌍 IPv6-Native & NAT64/DNS64 Ready
- Strictly tested on IPv6-only environments (e.g. Proxmox LXC containers).
- Automatically detects global IPv6 addresses and dynamically injects `ip6:<address>` into SPF records.
- Built-in fallback DNS resolvers prevent failures during outbound MX resolution on IPv6-only networks.

### 🎨 Clean Svelte 5 Webmail Interface
- Blazing-fast responsive user interface built with Svelte 5 and Vite (< 150KB gzip).
- Clean Inbox, Sent, Drafts, Trash, and Spam management.
- Rich HTML email editor with attachment support.
- User Management: Create accounts, reset passwords, and assign storage quotas.
- Domain Management: Live health checks, 1-click Cloudflare provisioning, and copyable DNS records.

---

## 🏗️ Architecture & How It Works

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

---

## 🚀 Quick Start & Deployment

### Method 1: Automated Python VPS Deployer (Recommended)

Lightmail includes an automated lifecycle manager [`deployer.py`](deployer.py) that builds the frontend, compiles the Linux binary, uploads with gzip compression, configures permissions, creates systemd units, and verifies health.

```bash
# Clone the repository
git clone https://github.com/ashu-choudhury/PMail.git
cd PMail

# Install & launch on a fresh Linux server
python deployer.py --host mail.yourdomain.com --user root --password "your_password" --action install

# Zero-downtime binary update
python deployer.py --host mail.yourdomain.com --user root --password "your_password" --action update

# Check service health, ports, and resource usage
python deployer.py --host mail.yourdomain.com --user root --password "your_password" --action status
```

### Method 2: Manual Single-Binary Run

1. **Build Frontend & Binary**:
   ```bash
   cd fe-svelte && bun run build && cd ..
   cp -r fe-svelte/dist/* server/listen/http_server/dist/
   cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o pmail main.go
   ```

2. **Run Initial Setup**:
   ```bash
   ./pmail -p 8080
   ```
   Open `http://your-server-ip:8080` in your web browser and follow the on-screen setup wizard to configure your admin credentials and primary domain.

---

## ⚙️ Configuration Reference

Lightmail stores its persistent configuration in `/opt/pmail/config/config.json`:

```jsonc
{
  "logLevel": "info",
  "domain": "yourdomain.com",                     // Primary mail domain
  "domains": [                                   // Configured multi-domains
    {
      "name": "yourdomain.com",
      "dkimSelector": "default",
      "dkimPrivateKeyPath": "config/dkim/dkim.priv"
    },
    {
      "name": "seconddomain.org",
      "dkimSelector": "default",
      "dkimPrivateKeyPath": "config/dkim/seconddomain.org.priv"
    }
  ],
  "webDomain": "mail.yourdomain.com",            // Webmail interface domain
  "dkimPrivateKeyPath": "config/dkim/dkim.priv",
  "sslType": "0",                                // 0: Auto Let's Encrypt, 1: Manual
  "SSLPrivateKeyPath": "config/ssl/private.key",
  "SSLPublicKeyPath": "config/ssl/public.crt",
  "dbDSN": "./config/pmail.db",                  // SQLite database path
  "dbType": "sqlite",                            // Pure SQLite engine
  "httpsEnabled": 2,                             // 0: HTTPS, 1: Redirect, 2: Reverse Proxy mode
  "httpPort": 8080,                              // Internal HTTP port
  "httpsPort": 443,                              // HTTPS port
  "spamFilterLevel": 0,                          // 0: Off, 1: SPF+DKIM fail, 2: SPF fail, 3: DKIM fail
  "isInit": true
}
```

---

## 📱 Mail Client Configuration

Lightmail works out-of-the-box with any standard email client (Thunderbird, Apple Mail, Outlook, iOS, Android, K-9 Mail):

| Protocol | Server Address | Port | Security / Encryption |
| :--- | :--- | :--- | :--- |
| **IMAP (Incoming)** | `imap.yourdomain.com` (or `smtp.yourdomain.com`) | **993** | SSL / TLS |
| **POP3 (Incoming)** | `pop.yourdomain.com` (or `smtp.yourdomain.com`) | **995** | SSL / TLS |
| **SMTP (Outgoing)** | `smtp.yourdomain.com` | **465** | SSL / TLS |
| **SMTP (Submission)** | `smtp.yourdomain.com` | **587** | STARTTLS |

- **Username**: Your full email address (e.g. `ashu@yourdomain.com`)
- **Password**: Your Lightmail mailbox password

---

## 🛠️ Developer Guide

### Development Prerequisites
- **Go**: 1.22 or newer
- **Node.js / Bun**: For compiling the Svelte webmail interface

### Project Structure
```
PMail/
├── fe-svelte/              # Modern Svelte 5 + Vite Webmail frontend
│   ├── src/components/     # Webmail & Admin UI components
│   └── src/lib/            # REST API client & reactive stores
├── server/                 # Go backend core
│   ├── controllers/        # HTTP API routes (domains, users, settings)
│   ├── db/                 # Pure SQLite storage layer (xorm)
│   ├── listen/             # Protocol servers (SMTP, IMAP, POP3, HTTP)
│   ├── models/             # SQLite database models (User, Email, Setting)
│   └── services/domain/    # Multi-domain DKIM & Cloudflare API v4 integration
├── deployer.py             # Lifecycle VPS deployment automation script
└── README.md
```

### Running Tests
```bash
cd server
go test ./...
```

---

## 📄 License

Lightmail is distributed under the [MIT License](LICENSE).  
Original PMail implementation Copyright © Jinnrry. Enhancements and Lightmail refactoring Copyright © Ashu Choudhury.
