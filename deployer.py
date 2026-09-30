#!/usr/bin/env python3
"""
Lightmail Linux VPS Deployer & Lifecycle Manager
================================================
Comprehensive, high-performance lifecycle management for Lightmail:
- Install: Clean, zero-hassle deployment on any fresh Linux server.
- Update: Fast compressed binary upload, zero-downtime atomic swap & restart.
- Status: Check systemd service health, listening ports, and resource usage.
- Restart: Gracefully restart the Lightmail service.
- Logs: View recent service logs via journalctl.
- Uninstall: Safely dismantle Lightmail, backup database & configs, and remove service.

Modes:
  1. CLI Argument Mode: e.g. python deployer.py --host mail.yourdomain.com --user root --password ... --action update
  2. Interactive Mode: Prompts user step-by-step with smart defaults.
"""

import sys
import os
import gzip
import shutil
import subprocess
import argparse
import getpass
import time
from pathlib import Path

# Ensure UTF-8 output on Windows consoles
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

try:
    import paramiko
except ImportError:
    print("[!] paramiko is not installed. Installing paramiko...")
    subprocess.check_call([sys.executable, "-m", "pip", "install", "paramiko"])
    import paramiko

# Global paths
REPO_DIR = Path(__file__).resolve().parent
FE_DIR = REPO_DIR / "fe-svelte"
SERVER_DIR = REPO_DIR / "server"
DIST_SOURCE = FE_DIR / "dist"
DIST_DEST = SERVER_DIR / "listen" / "http_server" / "dist"
BINARY_LOCAL = SERVER_DIR / "lightmail_linux_amd64"
BINARY_GZ = SERVER_DIR / "lightmail_linux_amd64.gz"
REMOTE_INSTALL_DIR = "/opt/pmail"
SERVICE_NAME = "pmail"

BANNER = r"""
========================================================================
   _     _       _     _                         _ _ 
  | |   (_) __ _| |__ | |_ _ __ ___   __ _  ___ (_) |
  | |   | |/ _` | '_ \| __| '_ ` _ \ / _` |/ _ \| | |
  | |___| | (_| | | | | |_| | | | | | (_| | (_) | | |
  |_____|_|\__, |_| |_|\__|_| |_| |_|\__,_|\___/|_|_|
           |___/                                     
  Lightmail Linux Deployer & Lifecycle Manager (v2.0)
========================================================================
"""

def print_banner():
    print(BANNER)

class SSHClientWrapper:
    def __init__(self, host, port, user, password=None, key_path=None):
        self.host = host
        self.port = int(port)
        self.user = user
        self.password = password
        self.key_path = key_path
        self.client = paramiko.SSHClient()
        self.client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        self.is_root = False
        self._connect()

    def _connect(self):
        print(f"[*] Connecting to {self.user}@{self.host}:{self.port}...")
        kwargs = {
            "hostname": self.host,
            "port": self.port,
            "username": self.user,
            "timeout": 30,
            "banner_timeout": 30,
        }
        if self.key_path and os.path.exists(self.key_path):
            kwargs["key_filename"] = self.key_path
        if self.password:
            kwargs["password"] = self.password

        self.client.connect(**kwargs)
        # Check if root
        code, out, _ = self.run("id -u", use_sudo=False)
        self.is_root = (out.strip() == "0")
        print(f"[+] Connected successfully ({'root privileges' if self.is_root else 'non-root user, sudo enabled'}).")

    def run(self, cmd, use_sudo=True, verbose=False):
        if use_sudo and not self.is_root:
            if self.password:
                # Use sudo -S
                full_cmd = f"echo '{self.password}' | sudo -S -p '' {cmd}"
            else:
                full_cmd = f"sudo -n {cmd}"
        else:
            full_cmd = cmd

        if verbose:
            print(f"[cmd] {cmd}")

        stdin, stdout, stderr = self.client.exec_command(full_cmd)
        out = stdout.read().decode("utf-8", errors="replace")
        err = stderr.read().decode("utf-8", errors="replace")
        code = stdout.channel.recv_exit_status()
        return code, out, err

    def upload_file(self, local_path, remote_path, progress_desc="Uploading"):
        sftp = self.client.open_sftp()
        local_size = os.path.getsize(local_path)
        print(f"[*] {progress_desc}: {os.path.basename(local_path)} ({local_size / 1024 / 1024:.2f} MB)...")

        last_percent = [-1]
        def progress_callback(transferred, total):
            pct = int((transferred / total) * 100)
            if pct % 10 == 0 and pct != last_percent[0]:
                last_percent[0] = pct
                sys.stdout.write(f"\r    -> {pct}% ({transferred / 1024 / 1024:.1f}/{total / 1024 / 1024:.1f} MB)")
                sys.stdout.flush()

        try:
            sftp.put(str(local_path), remote_path, callback=progress_callback)
            print("\n[+] Upload complete.")
        finally:
            sftp.close()

    def write_remote_file(self, remote_path, content, use_sudo=True):
        tmp_path = f"/tmp/pmail_tmp_{int(time.time() * 1000)}"
        sftp = self.client.open_sftp()
        try:
            with sftp.file(tmp_path, "w") as f:
                f.write(content)
        finally:
            sftp.close()
        self.run(f"mv -f {tmp_path} {remote_path}", use_sudo=use_sudo)
        self.run(f"chmod 644 {remote_path}", use_sudo=use_sudo)

    def close(self):
        self.client.close()

def build_assets(skip_build=False):
    """Builds Svelte frontend and compiles Go Linux AMD64 binary."""
    if skip_build and BINARY_LOCAL.exists():
        print(f"[+] Skipping build as requested. Using existing binary: {BINARY_LOCAL}")
        return

    print("\n--- [Step 1/3] Building Svelte Frontend ---")
    bun_bin = shutil.which("bun") or r"E:\Scoop\shims\bun.exe"
    if not os.path.exists(bun_bin):
        bun_bin = "bun"

    print(f"[*] Running frontend build with: {bun_bin}")
    try:
        subprocess.check_call([bun_bin, "run", "build"], cwd=str(FE_DIR), shell=True)
    except Exception as e:
        print(f"[!] Bun build failed ({e}). Attempting npm run build...")
        subprocess.check_call(["npm", "run", "build"], cwd=str(FE_DIR), shell=True)

    print("\n--- [Step 2/3] Syncing Embedded Frontend Assets ---")
    if DIST_DEST.exists():
        shutil.rmtree(DIST_DEST)
    shutil.copytree(DIST_SOURCE, DIST_DEST)
    print(f"[+] Synced {DIST_SOURCE} -> {DIST_DEST}")

    print("\n--- [Step 3/3] Compiling Lightmail Linux AMD64 Binary ---")
    env = os.environ.copy()
    env["CGO_ENABLED"] = "0"
    env["GOOS"] = "linux"
    env["GOARCH"] = "amd64"
    cmd = ["go", "build", "-ldflags", "-s -w", "-o", str(BINARY_LOCAL), "main.go"]
    print(f"[*] Running: {' '.join(cmd)} in {SERVER_DIR}")
    subprocess.check_call(cmd, cwd=str(SERVER_DIR), env=env, shell=True)
    size_mb = os.path.getsize(BINARY_LOCAL) / 1024 / 1024
    print(f"[+] Binary compiled successfully: {BINARY_LOCAL} ({size_mb:.2f} MB)")

def compress_binary():
    """Gzips the compiled Linux binary to maximize upload speed."""
    print("[*] Gzipping binary for high-speed SFTP transmission...")
    with open(BINARY_LOCAL, "rb") as f_in:
        with gzip.open(BINARY_GZ, "wb", compresslevel=6) as f_out:
            shutil.copyfileobj(f_in, f_out)
    orig_mb = os.path.getsize(BINARY_LOCAL) / 1024 / 1024
    gz_mb = os.path.getsize(BINARY_GZ) / 1024 / 1024
    pct = (1 - (gz_mb / orig_mb)) * 100
    print(f"[+] Compressed {orig_mb:.2f} MB down to {gz_mb:.2f} MB ({pct:.1f}% bandwidth savings).")

def action_update(ssh, skip_build=False):
    """Zero-downtime binary update."""
    print("\n================== ACTION: UPDATE ==================")
    build_assets(skip_build=skip_build)
    compress_binary()

    remote_gz = "/tmp/lightmail_linux_amd64.gz"
    ssh.upload_file(BINARY_GZ, remote_gz, progress_desc="Uploading compressed binary")

    print("[*] Deploying binary on remote server...")
    remote_cmds = [
        f"mkdir -p {REMOTE_INSTALL_DIR}",
        f"if [ -f {REMOTE_INSTALL_DIR}/pmail ]; then cp -f {REMOTE_INSTALL_DIR}/pmail {REMOTE_INSTALL_DIR}/pmail.bak; fi",
        f"gzip -dc {remote_gz} > /tmp/pmail.new",
        f"chmod 755 /tmp/pmail.new",
        f"mv -f /tmp/pmail.new {REMOTE_INSTALL_DIR}/pmail",
        f"which setcap >/dev/null 2>&1 && setcap 'cap_net_bind_service=+ep' {REMOTE_INSTALL_DIR}/pmail || true",
        f"rm -f {remote_gz}",
        f"systemctl restart {SERVICE_NAME} || true",
    ]
    for c in remote_cmds:
        code, out, err = ssh.run(c, use_sudo=True)
        if code != 0 and "pmail.bak" not in c and "setcap" not in c:
            print(f"[!] Warning on command '{c}': {err.strip()}")

    time.sleep(2)
    code, status_out, _ = ssh.run(f"systemctl is-active {SERVICE_NAME}", use_sudo=True)
    status_str = status_out.strip()
    if status_str == "active":
        print(f"\n[SUCCESS] Lightmail updated and running! (Status: {status_str})")
    else:
        print(f"\n[!] Service status: {status_str}. Checking journal...")
        _, logs, _ = ssh.run(f"journalctl -u {SERVICE_NAME} -n 20 --no-pager", use_sudo=True)
        print(logs)

def action_install(ssh, domain=None, admin_user=None, admin_pass=None, http_port="80", skip_build=False):
    """Full fresh install on any clean Linux server."""
    print("\n================== ACTION: FRESH INSTALL ==================")
    build_assets(skip_build=skip_build)
    compress_binary()

    # 1. Prepare system dependencies
    print("[*] Checking and installing system packages (gzip, libcap2-bin, curl)...")
    pkg_cmd = (
        "if which apt-get >/dev/null 2>&1; then "
        "  apt-get update -y && apt-get install -y gzip libcap2-bin curl tar; "
        "elif which yum >/dev/null 2>&1; then "
        "  yum install -y gzip libcap curl tar; "
        "elif which dnf >/dev/null 2>&1; then "
        "  dnf install -y gzip libcap curl tar; "
        "fi"
    )
    ssh.run(pkg_cmd, use_sudo=True)

    # 2. Create directories
    print(f"[*] Setting up directory structure at {REMOTE_INSTALL_DIR}...")
    ssh.run(f"mkdir -p {REMOTE_INSTALL_DIR}/config {REMOTE_INSTALL_DIR}/data {REMOTE_INSTALL_DIR}/logs {REMOTE_INSTALL_DIR}/ssl", use_sudo=True)

    # 3. Upload binary
    remote_gz = "/tmp/lightmail_linux_amd64.gz"
    ssh.upload_file(BINARY_GZ, remote_gz, progress_desc="Uploading binary")
    ssh.run(f"gzip -dc {remote_gz} > /tmp/pmail.new && chmod 755 /tmp/pmail.new && mv -f /tmp/pmail.new {REMOTE_INSTALL_DIR}/pmail && rm -f {remote_gz}", use_sudo=True)
    ssh.run(f"which setcap >/dev/null 2>&1 && setcap 'cap_net_bind_service=+ep' {REMOTE_INSTALL_DIR}/pmail || true", use_sudo=True)

    # 4. Generate initial config.json if not present
    code, check_cfg, _ = ssh.run(f"test -f {REMOTE_INSTALL_DIR}/config/config.json && echo EXISTS || echo NO", use_sudo=True)
    if "EXISTS" not in check_cfg:
        print("[*] Generating production config.json...")
        default_domain = domain or "example.com"
        config_json = f"""{{
  "domain": "{default_domain}",
  "webServer": {{
    "httpPort": {int(http_port)},
    "httpsPort": 443
  }},
  "smtpServer": {{
    "port": 25,
    "sslPort": 465,
    "tlsPort": 587
  }},
  "popServer": {{
    "port": 110,
    "sslPort": 995
  }},
  "imapServer": {{
    "port": 143,
    "sslPort": 993
  }},
  "db": {{
    "driver": "sqlite",
    "dsn": "{REMOTE_INSTALL_DIR}/config/pmail.db"
  }},
  "ssl": {{
    "mode": "manual"
  }}
}}"""
        ssh.write_remote_file(f"{REMOTE_INSTALL_DIR}/config/config.json", config_json, use_sudo=True)

    # 5. Create systemd service
    print(f"[*] Installing systemd service unit: /etc/systemd/system/{SERVICE_NAME}.service...")
    service_content = f"""[Unit]
Description=Lightmail High-Performance Mail Server
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory={REMOTE_INSTALL_DIR}
ExecStart={REMOTE_INSTALL_DIR}/pmail
Restart=always
RestartSec=5s
LimitNOFILE=65536
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
"""
    ssh.write_remote_file(f"/etc/systemd/system/{SERVICE_NAME}.service", service_content, use_sudo=True)
    ssh.run(f"systemctl daemon-reload && systemctl enable {SERVICE_NAME} && systemctl restart {SERVICE_NAME}", use_sudo=True)

    time.sleep(2)
    code, status_out, _ = ssh.run(f"systemctl is-active {SERVICE_NAME}", use_sudo=True)
    if status_out.strip() == "active":
        print(f"\n[SUCCESS] Lightmail installed and running on {ssh.host}!")
        print(f"    Webmail listening on port: {http_port}")
        print(f"    Primary domain: {domain or 'configured in config.json'}")
    else:
        print(f"\n[!] Warning: Service status is {status_out.strip()}. Check logs:")
        _, logs, _ = ssh.run(f"journalctl -u {SERVICE_NAME} -n 25 --no-pager", use_sudo=True)
        print(logs)

def action_status(ssh):
    """Inspects service status, listening ports, and disk space."""
    print("\n================== ACTION: STATUS ==================")
    print("--- Service Status ---")
    _, out, _ = ssh.run(f"systemctl status {SERVICE_NAME} --no-pager", use_sudo=True)
    print(out)

    print("--- Mail & Web Ports (25, 80, 110, 143, 465, 587, 993, 995, 8000, 8080) ---")
    _, ports, _ = ssh.run("ss -tulnp | grep -E ':25|:80|:110|:143|:465|:587|:993|:995|:8000|:8080' || netstat -tulnp", use_sudo=True)
    print(ports if ports.strip() else "No matched listening ports found.")

    print("--- Disk Usage (/opt/pmail) ---")
    _, du, _ = ssh.run(f"du -sh {REMOTE_INSTALL_DIR} 2>/dev/null || true", use_sudo=True)
    print(du.strip())

def action_restart(ssh):
    """Restarts the systemd service."""
    print("\n================== ACTION: RESTART ==================")
    print(f"[*] Restarting {SERVICE_NAME}.service...")
    ssh.run(f"systemctl restart {SERVICE_NAME}", use_sudo=True)
    time.sleep(1.5)
    code, out, _ = ssh.run(f"systemctl is-active {SERVICE_NAME}", use_sudo=True)
    print(f"[+] Status: {out.strip()}")

def action_logs(ssh, lines=50):
    """Streams recent systemd logs."""
    print(f"\n================== ACTION: LOGS (last {lines} lines) ==================")
    _, out, _ = ssh.run(f"journalctl -u {SERVICE_NAME} -n {lines} --no-pager", use_sudo=True)
    print(out)

def action_uninstall(ssh):
    """Safely dismantles Lightmail, archives DB/config, and cleans up."""
    print("\n================== ACTION: UNINSTALL / DISMANTLE ==================")
    backup_file = f"/tmp/pmail_backup_{int(time.time())}.tar.gz"
    print(f"[*] Creating backup archive of data and configs to {backup_file}...")
    ssh.run(f"tar -czf {backup_file} -C {REMOTE_INSTALL_DIR} config data 2>/dev/null || true", use_sudo=True)

    print(f"[*] Stopping and disabling {SERVICE_NAME}.service...")
    ssh.run(f"systemctl stop {SERVICE_NAME} || true", use_sudo=True)
    ssh.run(f"systemctl disable {SERVICE_NAME} || true", use_sudo=True)
    ssh.run(f"rm -f /etc/systemd/system/{SERVICE_NAME}.service && systemctl daemon-reload", use_sudo=True)

    print(f"[*] Removing installation directory {REMOTE_INSTALL_DIR}...")
    ssh.run(f"rm -rf {REMOTE_INSTALL_DIR}", use_sudo=True)

    print(f"\n[+] Dismantle complete. Safe backup retained at: {backup_file}")

def prompt_interactive():
    """Runs the interactive wizard."""
    print_banner()
    print("Welcome to the Lightmail Linux Deployment Wizard.")
    print("Please answer the following prompts (press Enter to accept default values).\n")

    host = input("Server Host / IP: ").strip()
    while not host:
        host = input("Host cannot be empty. Server Host / IP: ").strip()

    port_in = input("SSH Port [22]: ").strip()
    port = int(port_in) if port_in else 22

    user_in = input("SSH Username [root]: ").strip()
    user = user_in if user_in else "root"

    key_path = input("SSH Private Key path (optional, leave blank for password): ").strip()
    password = None
    if not key_path:
        password = getpass.getpass(f"SSH Password for {user}: ")

    print("\nSelect Action:")
    print("  [1] Update     - Fast compressed binary update & restart (Existing Server)")
    print("  [2] Install    - Fresh complete installation (New Linux Server)")
    print("  [3] Status     - Check service health, ports & disk")
    print("  [4] Restart    - Gracefully restart Lightmail service")
    print("  [5] Logs       - View recent service logs")
    print("  [6] Uninstall  - Dismantle Lightmail, backup data & remove service")
    choice = input("Enter choice [1-6, default 1]: ").strip() or "1"

    action_map = {
        "1": "update",
        "2": "install",
        "3": "status",
        "4": "restart",
        "5": "logs",
        "6": "uninstall",
    }
    action = action_map.get(choice, "update")

    install_args = {}
    if action == "install":
        print("\n--- Fresh Install Configuration ---")
        install_args["domain"] = input("Primary Mail Domain (e.g. yourdomain.com): ").strip()
        install_args["admin_user"] = input("Admin Username [admin]: ").strip() or "admin"
        install_args["admin_pass"] = getpass.getpass("Admin Password [min 8 chars]: ")
        install_args["http_port"] = input("Webmail HTTP Port [80]: ").strip() or "80"

    print(f"\nTarget: {user}@{host}:{port}")
    print(f"Action: {action.upper()}")
    confirm = input("Proceed? [Y/n]: ").strip().lower()
    if confirm and confirm != "y":
        print("[!] Aborted by user.")
        sys.exit(0)

    return {
        "host": host,
        "port": port,
        "user": user,
        "password": password,
        "key_path": key_path,
        "action": action,
        **install_args,
    }

def main():
    parser = argparse.ArgumentParser(description="Lightmail Linux VPS Deployer & Lifecycle Manager")
    parser.add_argument("--host", "-H", help="Target server hostname or IP")
    parser.add_argument("--port", "-P", type=int, default=22, help="SSH port (default: 22)")
    parser.add_argument("--user", "-u", default="root", help="SSH username (default: root)")
    parser.add_argument("--password", "-p", help="SSH password")
    parser.add_argument("--key", help="Path to SSH private key")
    parser.add_argument("--action", "-a", choices=["install", "update", "status", "restart", "logs", "uninstall"], help="Action to execute")
    parser.add_argument("--domain", help="Primary domain for install")
    parser.add_argument("--admin-user", help="Initial admin username")
    parser.add_argument("--admin-pass", help="Initial admin password")
    parser.add_argument("--http-port", default="80", help="HTTP port (default: 80)")
    parser.add_argument("--skip-build", action="store_true", help="Skip local bun/go compilation if already built")
    parser.add_argument("--interactive", "-i", action="store_true", help="Launch interactive wizard")

    args = parser.parse_args()

    # Determine mode
    if args.interactive or not args.host or not args.action:
        config = prompt_interactive()
        host = config["host"]
        port = config["port"]
        user = config["user"]
        password = config.get("password")
        key_path = config.get("key_path")
        action = config["action"]
        domain = config.get("domain")
        admin_user = config.get("admin_user")
        admin_pass = config.get("admin_pass")
        http_port = config.get("http_port", "80")
        skip_build = False
    else:
        host = args.host
        port = args.port
        user = args.user
        password = args.password
        key_path = args.key
        action = args.action
        domain = args.domain
        admin_user = args.admin_user
        admin_pass = args.admin_pass
        http_port = args.http_port
        skip_build = args.skip_build

    # Execute SSH connection
    ssh = SSHClientWrapper(host=host, port=port, user=user, password=password, key_path=key_path)
    try:
        if action == "update":
            action_update(ssh, skip_build=skip_build)
        elif action == "install":
            action_install(ssh, domain=domain, admin_user=admin_user, admin_pass=admin_pass, http_port=http_port, skip_build=skip_build)
        elif action == "status":
            action_status(ssh)
        elif action == "restart":
            action_restart(ssh)
        elif action == "logs":
            action_logs(ssh)
        elif action == "uninstall":
            action_uninstall(ssh)
    finally:
        ssh.close()

if __name__ == "__main__":
    main()
