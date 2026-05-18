# 🤝 Contributing to Deck

We love contributions! Whether you want to add cloud providers (AWS EC2, DigitalOcean, Hetzner), improve TUI themes, or fix bugs, here is how you can help.

---

## 🛠️ Development Setup

### Prerequisites
* **Go 1.23+** installed
* `git` installed
* `make` (optional)

### Build & Run Locally
```bash
# Clone repository
git clone https://github.com/Dolyyyy/deck.git
cd deck

# Run directly without compilation
go run .

# Or build binary
make build
./bin/deck
```

### Run Tests
```bash
make test
# or
go test -v -race ./...
```

---

## 📐 Conventional Commits Standard

We adhere strictly to [Conventional Commits](https://www.conventionalcommits.org/):

* `feat(scope): ...` — New user-facing feature
* `fix(scope): ...` — Bug fix
* `refactor(scope): ...` — Code improvement without feature change
* `test(scope): ...` — Adding or updating tests
* `docs: ...` — Documentation updates
* `ci: ...` — CI/CD workflow changes
* `chore: ...` — Dependencies, build scripts

Examples:
* `feat(tui): add split-screen layout for cluster commands`
* `fix(config): handle spaces in identity file paths`
* `test(ping): add benchmark for concurrent worker pool`

---

## 🚀 Pull Request Workflow

1. Fork the repository.
2. Create your feature branch (`git checkout -b feat/my-provider`).
3. Ensure all tests pass (`go test -v -race ./...`).
4. Commit using conventional commits.
5. Push to your branch and open a Pull Request.
