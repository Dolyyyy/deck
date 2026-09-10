package aliasdiscovery

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Dolyyyy/deck/pkg/models"
)

// ItemType defines the category of an auto-discovered shell item.
type ItemType string

const (
	TypeServer  ItemType = "SSH Server"
	TypeAlias   ItemType = "Directory Alias"
	TypeSnippet ItemType = "Command Snippet"
)

// DiscoveredItem represents an imported alias candidate.
type DiscoveredItem struct {
	Type       ItemType
	Name       string
	Target     string
	SourceFile string
	Selected   bool

	Server  *models.Server
	Alias   *models.DirectoryAlias
	Snippet *models.Snippet
}

var (
	// Common standard shell aliases to ignore
	ignoredNames = map[string]bool{
		"ll": true, "la": true, "l": true, "ls": true,
		"grep": true, "fgrep": true, "egrep": true,
		"alert": true, "which": true, "vi": true, "vim": true,
		"cp": true, "mv": true, "rm": true,
	}

	// Sourcing patterns: source <file> or . <file> or [ -f <file> ] or [[ -f <file> ]] or test -f <file>
	sourceRegex = regexp.MustCompile(`(?:^|[;&|]|\bthen\b|\bdo\b)\s*(?:source|\.)\s+["']?([^"'\s;&|#]+)["']?`)
	testRegex   = regexp.MustCompile(`(?:\[{1,2}|\btest)\s+-[fse]\s+["']?([^"'\s;&|#]+)["']?\s*\]{0,2}`)
	forRegex    = regexp.MustCompile(`\bfor\s+[a-zA-Z0-9_]+\s+in\s+["']?([^"'\s;]+)["']?`)

	// Function patterns: name() { command; } or function name() { command; }
	funcRegex = regexp.MustCompile(`^(?:function\s+)?([a-zA-Z0-9_\.\-]+)\s*\(\)\s*\{\s*(.*?)\s*;?\s*\}$`)
)

// DefaultCandidateFiles returns a comprehensive list of candidate shell profile paths across user and system files.
func DefaultCandidateFiles() []string {
	var candidates []string
	seen := make(map[string]bool)

	addCandidate := func(p string) {
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		seen[p] = true
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			candidates = append(candidates, p)
		}
	}

	addGlob := func(pattern string) {
		matches, err := filepath.Glob(pattern)
		if err == nil {
			for _, m := range matches {
				addCandidate(m)
			}
		}
	}

	var homeDirs []string
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		homeDirs = append(homeDirs, h)
	}
	// Check /root if accessible
	if fi, err := os.Stat("/root"); err == nil && fi.IsDir() {
		homeDirs = append(homeDirs, "/root")
	}
	// Check all users in /home/* (e.g. /home/ubuntu, /home/admin, /home/doly)
	if userHomes, err := filepath.Glob("/home/*"); err == nil {
		for _, uh := range userHomes {
			if fi, err := os.Stat(uh); err == nil && fi.IsDir() {
				homeDirs = append(homeDirs, uh)
			}
		}
	}

	// De-duplicate home directories
	seenHomes := make(map[string]bool)
	var uniqueHomes []string
	for _, h := range homeDirs {
		if !seenHomes[h] {
			seenHomes[h] = true
			uniqueHomes = append(uniqueHomes, h)
		}
	}

	for _, h := range uniqueHomes {
		// Specific profile and alias files
		addCandidate(filepath.Join(h, ".alias"))
		addCandidate(filepath.Join(h, ".aliases"))
		addCandidate(filepath.Join(h, ".bash_aliases"))
		addCandidate(filepath.Join(h, ".bashrc"))
		addCandidate(filepath.Join(h, ".zshrc"))
		addCandidate(filepath.Join(h, ".zsh_aliases"))
		addCandidate(filepath.Join(h, ".aliases.zsh"))
		addCandidate(filepath.Join(h, ".aliases.sh"))
		addCandidate(filepath.Join(h, ".profile"))
		addCandidate(filepath.Join(h, ".bash_profile"))
		addCandidate(filepath.Join(h, ".zprofile"))
		addCandidate(filepath.Join(h, ".zshenv"))
		addCandidate(filepath.Join(h, ".login"))
		addCandidate(filepath.Join(h, ".kshrc"))
		addCandidate(filepath.Join(h, ".cshrc"))
		addCandidate(filepath.Join(h, ".tcshrc"))
		addCandidate(filepath.Join(h, ".shrc"))
		addCandidate(filepath.Join(h, ".config", "fish", "config.fish"))
		addCandidate(filepath.Join(h, ".config", "alias"))
		addCandidate(filepath.Join(h, ".config", "aliases"))
		addCandidate(filepath.Join(h, ".config", "shell", "alias"))
		addCandidate(filepath.Join(h, ".config", "shell", "aliases"))
		addCandidate(filepath.Join(h, ".config", "deck", "aliases"))

		// Globs in user home
		addGlob(filepath.Join(h, ".alias*"))
		addGlob(filepath.Join(h, ".*aliases*"))
		addGlob(filepath.Join(h, ".bashrc.d", "*"))
		addGlob(filepath.Join(h, ".profile.d", "*"))
		addGlob(filepath.Join(h, ".config", "fish", "conf.d", "*.fish"))
		addGlob(filepath.Join(h, ".config", "fish", "functions", "*.fish"))
		addGlob(filepath.Join(h, ".config", "shell", "*"))
		addGlob(filepath.Join(h, ".oh-my-zsh", "custom", "*.zsh"))
		addGlob(filepath.Join(h, ".oh-my-zsh", "custom", "aliases.zsh"))
		addGlob(filepath.Join(h, ".zsh", "aliases*"))
		addGlob(filepath.Join(h, ".bash", "aliases*"))
	}

	// System-wide profiles
	addCandidate("/etc/bash.bashrc")
	addCandidate("/etc/zsh/zshrc")
	addCandidate("/etc/profile")
	addCandidate("/etc/environment")
	addCandidate("/etc/aliases")
	addGlob("/etc/profile.d/*.sh")

	return candidates
}

// Scanner discovers aliases and converts them to Deck models.
type Scanner struct {
	files []string
}

// NewScanner creates an alias scanner.
func NewScanner(customFiles ...string) *Scanner {
	files := customFiles
	if len(files) == 0 {
		files = DefaultCandidateFiles()
	}
	return &Scanner{files: files}
}

// Discover finds new SSH servers, directory aliases, and command snippets not yet in DeckConfig.
func (s *Scanner) Discover(
	existingServers []*models.Server,
	existingAliases []*models.DirectoryAlias,
	existingSnippets []*models.Snippet,
) ([]*DiscoveredItem, error) {
	var discovered []*DiscoveredItem
	seenKeys := make(map[string]bool)

	// Preload existing keys to prevent duplicates
	serverMap := make(map[string]bool)
	for _, srv := range existingServers {
		serverMap[strings.ToLower(srv.Name)] = true
		serverMap[strings.ToLower(srv.DisplayAddress())] = true
	}

	aliasMap := make(map[string]bool)
	for _, al := range existingAliases {
		aliasMap[strings.ToLower(al.Name)] = true
		aliasMap[strings.ToLower(al.Path)] = true
	}

	snippetMap := make(map[string]bool)
	for _, snip := range existingSnippets {
		snippetMap[strings.ToLower(snip.Name)] = true
	}

	homeDir, _ := os.UserHomeDir()

	queue := append([]string{}, s.files...)
	processedFiles := make(map[string]bool)

	for len(queue) > 0 {
		file := queue[0]
		queue = queue[1:]
		if processedFiles[file] {
			continue
		}
		processedFiles[file] = true

		f, err := os.Open(file)
		if err != nil {
			continue
		}

		relPath := file
		if homeDir != "" && strings.HasPrefix(file, homeDir) {
			relPath = "~" + file[len(homeDir):]
		}
		fileDir := filepath.Dir(file)

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// 1. Recursive Auto-Discovery of Sourced / Imported Files
			// e.g. source ~/.alias, . /root/.alias, [ -f ~/.alias ] && . ~/.alias
			sourcedFiles := extractSourcedFiles(line, fileDir, homeDir)
			for _, sf := range sourcedFiles {
				if !processedFiles[sf] {
					queue = append(queue, sf)
				}
			}

			// 2. Extract alias or function name and command
			aliasName, cmdRaw := parseAliasOrFunctionLine(line)
			if aliasName == "" || cmdRaw == "" {
				continue
			}

			if ignoredNames[strings.ToLower(aliasName)] {
				continue
			}

			lowerCmd := strings.ToLower(cmdRaw)

			// 3. SSH / Mosh Server Alias (e.g. ssh ..., echo '...' && ssh ..., ssh -A -J ...)
			sshIdx := -1
			if strings.HasPrefix(lowerCmd, "ssh ") || strings.HasPrefix(lowerCmd, "ssh\t") ||
				strings.HasPrefix(lowerCmd, "mosh ") || strings.HasPrefix(lowerCmd, "mosh\t") {
				sshIdx = 0
			} else if idx := strings.Index(lowerCmd, "&& ssh "); idx != -1 {
				sshIdx = idx + 3
			} else if idx := strings.Index(lowerCmd, "&& mosh "); idx != -1 {
				sshIdx = idx + 3
			} else if idx := strings.Index(lowerCmd, "; ssh "); idx != -1 {
				sshIdx = idx + 2
			} else if idx := strings.Index(lowerCmd, "; mosh "); idx != -1 {
				sshIdx = idx + 2
			} else if idx := strings.Index(lowerCmd, " ssh "); idx != -1 {
				sshIdx = idx + 1
			} else if idx := strings.Index(lowerCmd, " mosh "); idx != -1 {
				sshIdx = idx + 1
			}

			if sshIdx != -1 {
				sshPart := strings.TrimSpace(cmdRaw[sshIdx:])
				srv := parseSSHCommand(aliasName, sshPart)
				if srv != nil {
					key := "server:" + strings.ToLower(srv.Name)
					if !seenKeys[key] && !serverMap[strings.ToLower(srv.Name)] && !serverMap[strings.ToLower(srv.DisplayAddress())] {
						seenKeys[key] = true
						discovered = append(discovered, &DiscoveredItem{
							Type:       TypeServer,
							Name:       srv.Name,
							Target:     srv.DisplayAddress(),
							SourceFile: relPath,
							Selected:   true,
							Server:     srv,
						})
					}
					continue
				}
			}

			// 4. Directory Navigation Alias (e.g. alias 1="cd /home/ubuntu/doly" or alias web="cd /home/admin/web")
			dirPath := extractDirPath(cmdRaw, homeDir)
			if dirPath != "" {
				key := "alias:" + strings.ToLower(aliasName)
				if !seenKeys[key] && !aliasMap[strings.ToLower(aliasName)] && !aliasMap[strings.ToLower(dirPath)] {
					seenKeys[key] = true
					aliasObj := &models.DirectoryAlias{
						Name:        aliasName,
						Path:        dirPath,
						Description: fmt.Sprintf("Imported from %s", relPath),
					}
					discovered = append(discovered, &DiscoveredItem{
						Type:       TypeAlias,
						Name:       aliasName,
						Target:     dirPath,
						SourceFile: relPath,
						Selected:   true,
						Alias:      aliasObj,
					})
				}
				continue
			}

			// 5. Command Snippet Alias (e.g. alias c="clear" or alias pms="pm2 status" or alias p="chmod -R 777 *")
			key := "snippet:" + strings.ToLower(aliasName)
			if !seenKeys[key] && !snippetMap[strings.ToLower(aliasName)] {
				seenKeys[key] = true
				snip := &models.Snippet{
					Name:        aliasName,
					Command:     cmdRaw,
					TargetHost:  "",
					Description: fmt.Sprintf("Imported from %s", relPath),
					Tags:        []string{"shell-alias", "imported"},
				}
				discovered = append(discovered, &DiscoveredItem{
					Type:       TypeSnippet,
					Name:       aliasName,
					Target:     cmdRaw,
					SourceFile: relPath,
					Selected:   true,
					Snippet:    snip,
				})
			}
		}
		_ = f.Close()
	}

	return discovered, nil
}

// extractSourcedFiles finds paths sourced via `source`, `.`, or `[ -f ... ]` on the given line.
func extractSourcedFiles(line, currentFileDir, homeDir string) []string {
	var results []string
	seen := make(map[string]bool)

	cleanLine := line
	if idx := strings.Index(line, "#"); idx != -1 {
		cleanLine = strings.TrimSpace(line[:idx])
	}
	if cleanLine == "" {
		return nil
	}

	addPath := func(rawPath string) {
		rawPath = strings.TrimSpace(rawPath)
		rawPath = strings.Trim(rawPath, `"'`)
		if rawPath == "" || rawPath == "." || rawPath == ".." || strings.HasPrefix(rawPath, "-") {
			return
		}

		// Expand variables
		if homeDir != "" {
			rawPath = strings.ReplaceAll(rawPath, "$HOME", homeDir)
			rawPath = strings.ReplaceAll(rawPath, "${HOME}", homeDir)
			if strings.HasPrefix(rawPath, "~/") {
				expanded := filepath.Join(homeDir, rawPath[2:])
				if fi, err := os.Stat(expanded); err == nil && !fi.IsDir() {
					rawPath = expanded
				} else if currentFileDir != "" {
					relAttempt := filepath.Join(currentFileDir, rawPath[2:])
					if fi, err := os.Stat(relAttempt); err == nil && !fi.IsDir() {
						rawPath = relAttempt
					} else {
						rawPath = expanded
					}
				} else {
					rawPath = expanded
				}
			} else if rawPath == "~" {
				rawPath = homeDir
			}
		}

		if !filepath.IsAbs(rawPath) && currentFileDir != "" {
			rawPath = filepath.Join(currentFileDir, rawPath)
		}

		// Glob expansion
		if strings.ContainsAny(rawPath, "*?") {
			if matches, err := filepath.Glob(rawPath); err == nil {
				for _, m := range matches {
					m = filepath.Clean(m)
					if fi, err := os.Stat(m); err == nil && !fi.IsDir() && !seen[m] {
						seen[m] = true
						results = append(results, m)
					}
				}
			}
			return
		}

		clean := filepath.Clean(rawPath)
		if fi, err := os.Stat(clean); err == nil && !fi.IsDir() && !seen[clean] {
			seen[clean] = true
			results = append(results, clean)
		}
	}

	// Find `source <path>` or `. <path>`
	for _, match := range sourceRegex.FindAllStringSubmatch(cleanLine, -1) {
		if len(match) > 1 {
			addPath(match[1])
		}
	}

	// Find `[ -f <path> ]` or `[ -s <path> ]`
	for _, match := range testRegex.FindAllStringSubmatch(cleanLine, -1) {
		if len(match) > 1 {
			addPath(match[1])
		}
	}

	// Find `for f in <glob>`
	for _, match := range forRegex.FindAllStringSubmatch(cleanLine, -1) {
		if len(match) > 1 {
			addPath(match[1])
		}
	}

	return results
}

// parseAliasOrFunctionLine extracts name and target command from bash/zsh/fish alias or shell function.
func parseAliasOrFunctionLine(line string) (string, string) {
	// 1. Standard bash/zsh alias: alias name="cmd" or alias -g name='cmd'
	if strings.HasPrefix(line, "alias ") || strings.HasPrefix(line, "alias\t") {
		rest := strings.TrimSpace(line[5:])
		// Strip flags like -g, -s
		if strings.HasPrefix(rest, "-g ") || strings.HasPrefix(rest, "-s ") {
			rest = strings.TrimSpace(rest[3:])
		}
		eqIdx := strings.Index(rest, "=")
		if eqIdx != -1 {
			name := strings.TrimSpace(rest[:eqIdx])
			cmd := strings.TrimSpace(rest[eqIdx+1:])
			cmd = unquoteCmd(cmd)
			return name, cmd
		}
		// Fish style: alias name "cmd"
		fields := strings.Fields(rest)
		if len(fields) >= 2 {
			name := fields[0]
			cmd := strings.TrimSpace(rest[len(name):])
			cmd = unquoteCmd(cmd)
			return name, cmd
		}
	}

	// 2. Fish abbreviation: abbr -a name "cmd" or abbr name "cmd"
	if strings.HasPrefix(line, "abbr ") || strings.HasPrefix(line, "abbr\t") {
		rest := strings.TrimSpace(line[4:])
		for strings.HasPrefix(rest, "-") {
			parts := strings.Fields(rest)
			if len(parts) > 1 {
				rest = strings.TrimSpace(rest[len(parts[0]):])
			} else {
				break
			}
		}
		fields := strings.Fields(rest)
		if len(fields) >= 2 {
			name := fields[0]
			cmd := strings.TrimSpace(rest[len(name):])
			cmd = unquoteCmd(cmd)
			return name, cmd
		}
	}

	// 3. Single-line function: name() { ...; }
	if match := funcRegex.FindStringSubmatch(line); len(match) > 2 {
		name := strings.TrimSpace(match[1])
		cmd := strings.TrimSpace(match[2])
		cmd = strings.TrimSuffix(cmd, ";")
		cmd = strings.TrimSpace(cmd)
		return name, cmd
	}

	return "", ""
}

func unquoteCmd(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if (strings.HasPrefix(cmd, "\"") && strings.HasSuffix(cmd, "\"")) ||
		(strings.HasPrefix(cmd, "'") && strings.HasSuffix(cmd, "'")) {
		cmd = cmd[1 : len(cmd)-1]
	}
	return strings.TrimSpace(cmd)
}

func parseSSHCommand(name, cmdStr string) *models.Server {
	parts := strings.Fields(cmdStr)
	if len(parts) < 2 {
		return nil
	}

	var (
		user     string
		hostname string
		port     int = 22
		keyPath  string
	)

	for i := 1; i < len(parts); i++ {
		p := parts[i]
		if p == "&&" || p == ";" || p == "||" || p == "|" {
			break
		}

		if (p == "-p" || p == "-i" || p == "-J" || p == "-l" || p == "-o" || p == "-F" || p == "-c") && i+1 < len(parts) {
			if p == "-p" {
				if pt, err := strconv.Atoi(parts[i+1]); err == nil {
					port = pt
				}
			} else if p == "-i" {
				keyPath = parts[i+1]
			} else if p == "-l" {
				user = parts[i+1]
			}
			i++
			continue
		} else if strings.HasPrefix(p, "-p") && len(p) > 2 {
			if pt, err := strconv.Atoi(p[2:]); err == nil {
				port = pt
			}
			continue
		} else if strings.HasPrefix(p, "-i") && len(p) > 2 {
			keyPath = p[2:]
			continue
		}

		if strings.HasPrefix(p, "-") {
			continue
		}

		// Positional argument (user@host or host)
		if hostname == "" {
			if atIdx := strings.Index(p, "@"); atIdx != -1 {
				user = p[:atIdx]
				hostname = p[atIdx+1:]
			} else {
				hostname = p
			}
		}
	}

	if hostname == "" {
		return nil
	}

	if user == "" {
		user = "root"
	}

	env := "default"
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "prod") {
		env = "production"
	} else if strings.Contains(lowerName, "stag") {
		env = "staging"
	} else if strings.Contains(lowerName, "dev") || strings.Contains(lowerName, "local") {
		env = "development"
	}

	return &models.Server{
		Name:         name,
		Hostname:     hostname,
		User:         user,
		Port:         port,
		IdentityFile: keyPath,
		Environment:  env,
		Tags:         []string{"shell-alias", "imported"},
		Source:       "shell_alias",
		Status:       models.StatusUnknown,
	}
}

func extractDirPath(cmdStr, homeDir string) string {
	parts := strings.Fields(cmdStr)
	for i, p := range parts {
		if (p == "cd" || p == "z" || p == "pushd") && i+1 < len(parts) {
			pathArg := strings.Trim(parts[i+1], `"'`)
			if pathArg == ".." || pathArg == "." || pathArg == "-" || pathArg == "" {
				return ""
			}
			if strings.HasPrefix(pathArg, "~/") && homeDir != "" {
				pathArg = filepath.Join(homeDir, pathArg[2:])
			} else if pathArg == "~" && homeDir != "" {
				pathArg = homeDir
			}
			return filepath.Clean(pathArg)
		}
	}
	return ""
}
