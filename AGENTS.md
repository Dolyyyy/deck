# 🤖 Guidelines for AI Coding Agents in Deck

Welcome AI assistant! When reading, maintaining, or modifying `deck`, adhere to these core design rules:

1. **Idiomatic Go**: Keep packages focused and cohesive. Do not create cyclic dependencies.
2. **Charm Stack Integrity**: Use `lipgloss` for styling, `bubbletea` for reactive TUI state, `bubbles` for widgets. Do not mix other terminal libraries.
3. **Security Standards**: Never log plaintext passwords or private keys. Cryptographic operations must use constant-time comparisons (`subtle.ConstantTimeCompare`) and cryptographically secure random sources (`crypto/rand`).
4. **Testing**: Any new functionality in `internal/*` or `pkg/*` must include unit tests with table-driven tests (`*_test.go`).
5. **Cross-Platform Compatibility**: Always account for Windows, Linux, and macOS differences (paths, line endings, syscall vs os/exec).
