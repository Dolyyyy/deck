# 🏛️ Deck Technical Architecture

`deck` is built with a clean, modular, port-and-adapter architecture designed for speed, safety, and extensibility.

```
┌─────────────────────────────────────────────────────────────┐
│                       Entrypoint                            │
│           main.go / cmd/deck (CLI Flags & Args)             │
└──────────────────────────────┬──────────────────────────────┘
                               │
       ┌───────────────────────┼──────────────────────┐
       ▼                       ▼                      ▼
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│  TUI Cockpit │       │ Direct Jump  │       │ Export/Import│
│ (Bubbletea)  │       │  (SSH / Dir) │       │   (Base64)   │
└──────┬───────┘       └───────┬──────┘       └───────┬──────┘
       │                       │                      │
       └───────────────────────┼──────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                    Core Service Layer                       │
│  ├── internal/config  : SSH Config & deck.yaml Manager      │
│  ├── internal/ping    : Async TCP Worker Pool Pinger        │
│  ├── internal/ssh     : Native syscall.Exec & Stats Runner  │
│  ├── internal/crypto  : AES-256-GCM + Argon2id Vault        │
│  └── internal/transfer: Base64 Serializer & Importer        │
└──────────────────────────────┬──────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                     Domain Models                           │
│  pkg/models (Server, DirectoryAlias, Tunnel, Stats, Vault)  │
└─────────────────────────────────────────────────────────────┘
```

---

## 1. Zero-Overhead Direct Terminal Handoff (`internal/ssh`)
When a user selects a server or runs `deck <target>`, Deck does not proxy raw PTY streams through an internal wrapper unless requested. On Linux and macOS, Deck invokes `syscall.Exec` directly on the native `/usr/bin/ssh` binary.

### Why this matters:
* Full fidelity with user's `~/.ssh/config`, `ssh-agent`, smartcards/YubiKeys, and `known_hosts`.
* Instant terminal signals (`Ctrl+C`, `Ctrl+Z`, terminal resize `SIGWINCH`) work natively.
* Zero memory footprint during the SSH session.

---

## 2. Non-Blocking Concurrent Ping Engine (`internal/ping`)
* Probes servers using raw non-blocking TCP handshakes (`net.DialTimeout`) to the target port.
* Dispatches requests across an elastic worker pool (default 10-15 workers).
* Emits live events through a Go channel `<-chan *PingResult` so Bubbletea renders status badges asynchronously without UI stutter.

---

## 3. Cryptographic Vault & Transfer Security (`internal/crypto`)
* **Key Derivation**: Argon2id (`memory: 64MB`, `iterations: 3`, `parallelism: 4`) deriving a 256-bit AES key from user passphrases.
* **Cipher**: AES-256 in Galois/Counter Mode (GCM) providing both confidentiality and integrity with random 96-bit nonces.
* **Zero-Knowledge Base64 Packaging**: Encrypted export payloads can be safely posted in team chats or slack channels without leaking IP addresses, keys, or server topologies.

---

## 4. Shell Directory Jumping (`internal/config`)
* Bookmarks are resolved in `O(1)` time.
* Seamless shell integration via `deck init bash|zsh|fish` wraps directory changes directly into the parent terminal session.
