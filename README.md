# ⚡ DECK — The Terminal Cockpit & Fleet Manager

<div align="center">

```text
  ██████╗ ███████╗ ██████╗██╗  ██╗
  ██╔══██╗██╔════╝██╔════╝██║ ██╔╝
  ██║  ██║█████╗  ██║     █████═╝ 
  ██║  ██║██╔══╝  ██║     ██╔═██╗ 
  ██████╔╝███████╗╚██████╗██║  ██╗
  ╚═════╝ ╚══════╝ ╚═════╝╚═╝  ╚═╝
```

**The high-performance terminal cockpit for SSH servers, directory bookmarks, port-forwarding tunnels, command runbooks & SSL monitoring.**  
*Crafted in pure Go with [Charm.sh](https://charm.sh) (`bubbletea`, `lipgloss`, `bubbles`).*

[![Release](https://img.shields.io/github/v/release/Dolyyyy/deck?style=flat-square&color=blue)](https://github.com/Dolyyyy/deck/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![CI Status](https://img.shields.io/badge/build-passing-brightgreen.svg?style=flat-square)](.github/workflows/ci.yml)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](CONTRIBUTING.md)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-blueviolet.svg?style=flat-square)](#installation)

</div>

---

## 🌟 5 Cockpit Tabs in One Binary

* 🖥️ **`[1] SSH Servers`**: Instant terminal handoff (`syscall.Exec`), concurrent live TCP latency probes with vibrant traffic-light badges (`● ONLINE` in Green, `○ OFFLINE` in Red, ms latency), non-interactive remote metrics inspection (`s`).
* 📁 **`[2] Directory Aliases`**: Bookmark project paths with `deck alias add backend ~/projects/backend` and jump in seconds with `deck backend` or `deck cd <alias>`.
* 🔀 **`[3] SSH Tunnels`**: Toggle background port-forwarding tunnels (`localhost:5432 -> remote:5432`) on and off with a single tap of `Space`. Graceful process group cleanup on app exit.
* ⚡ **`[4] Command Snippets`**: Save recurrent terminal commands and runbooks. Execute them locally or remotely over SSH with `Enter`, or copy to clipboard with `c`.
* 🌐 **`[5] Web Endpoints & SSL`**: Monitor HTTP/HTTPS services for latency, status codes (`200 OK`), and automatically check SSL certificate expiration days remaining (`SSL: 82d left ✅` / `SSL: 4d left 🚨`).
* 🔄 **Built-in Auto-Updater**: Detects new GitHub releases in the top bar (`[U] UPDATE AVAILABLE`) and self-updates the binary with `deck update`.
* 🔒 **AES-256-GCM + Argon2id Vault**: Export/import your complete cockpit setup across machines using single encrypted Base64 strings.

---

## 📥 Installation

### 🐧 Linux & 🍎 macOS (1-Line Quick Install)
```bash
curl -fsSL https://raw.githubusercontent.com/Dolyyyy/deck/main/scripts/install.sh | bash
```

### 🪟 Windows (PowerShell 1-Line)
```powershell
irm https://raw.githubusercontent.com/Dolyyyy/deck/main/scripts/install.ps1 | iex
```

### 🐹 Via Go Install
```bash
go install github.com/Dolyyyy/deck/cmd/deck@latest
```

### 📦 Self-Update
Once installed, update to the latest release anytime with:
```bash
deck update
```

---

## 🚀 Quickstart & CLI Commands

### 1. Launch the Interactive Cockpit
Simply run `deck` to open the full-screen terminal cockpit:
```bash
deck
```

### 2. Instant SSH Shortcut
```bash
deck prod-web        # Connects directly to server 'prod-web'
deck rc              # Fuzzy matches 'release-candidate-01' and jumps immediately!
```

### 3. Directory Bookmarks & Instant Shell `cd`
```bash
# Add an alias
deck alias add backend ~/projects/backend "Main backend API service"

# Jump into it directly
deck backend

# Shell integration: add to ~/.bashrc or ~/.zshrc:
eval "$(deck init bash)"
deck cd backend      # Changes directory inside current shell
```

### 4. SSH Tunnels
```bash
deck tunnel          # View active/configured tunnels
```

### 5. Run Command Snippets
```bash
deck run docker-clean
```

### 6. Web Endpoints & SSL Check
```bash
deck endpoint        # Probe URLs and check certificate expiration
```

### 7. Overview & Live Health Check (`deck ls`)
```bash
deck ls
```

### 8. Encrypted Vault Export & Import
```bash
# Export encrypted
deck export --password "MySecretPassphrase"

# Import on another machine:
deck import "<base64_string>" --password "MySecretPassphrase"
```

---

## ⌨️ Cockpit Keybindings

| Key | Action |
| :--- | :--- |
| `1` .. `5` | Direct jump to cockpit tab (Servers, Aliases, Tunnels, Snippets, Endpoints) |
| `Tab` / `Shift+Tab` | Cycle through cockpit tabs |
| `Enter` | Connect to SSH / Jump to folder / Run snippet / Open URL in browser |
| `Space` | Toggle SSH tunnel Start / Stop |
| `c` | Copy snippet command to clipboard |
| `a` | Add new entry to the active tab (modal form) |
| `e` | Edit focused entry (pre-filled modal) |
| `x` | Delete focused entry |
| `s` | Query remote host metrics non-interactively (CPU, RAM, Disk, Uptime) |
| `/` | Live filter / search bar |
| `r` / `p` | Re-run health ping / probe all endpoints |
| `U` | Trigger auto-updater when a new release is available |
| `E` / `i` | Encrypted Base64 inventory export / import |
| `?` | Keyboard shortcut help overlay |
| `q` / `Ctrl+C` | Exit Deck (gracefully stops active tunnels) |

---

## 🔐 Security Architecture

* **Native SSH Terminal Handoff**: Sessions run through `syscall.Exec`, retaining full hardware key (YubiKey), `ssh-agent`, and OpenSSH config compatibility.
* **Argon2id Key Derivation**: High-memory derivation (`64 MB`, `3 iterations`, `4 threads`) protects against GPU/ASIC password cracking.
* **AES-256-GCM**: Authenticated encryption ensures export tamper-resistance.
* **Master Password Protection**: Run `deck lock` to enforce a master passphrase on launch.

---

## 🏗️ Development & Tests

```bash
# Run directly without compilation
go run .

# Run all unit tests with race detector
go test -v -race ./...

# Cross-compile for all architectures
make cross-build
```

---

## 📄 License

Distributed under the **MIT License**. See [LICENSE](LICENSE) for more information.
