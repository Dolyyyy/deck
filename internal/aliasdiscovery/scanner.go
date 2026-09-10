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
	// Matches: alias foo="bar" or alias foo='bar' or alias foo=bar
	aliasRegex = regexp.MustCompile(`^\s*alias\s+([a-zA-Z0-9_\.\-]+)=['"]?(.*?)['"]?\s*$`)
	// Common standard shell aliases to ignore
	ignoredNames = map[string]bool{
		"ll": true, "la": true, "l": true, "ls": true,
		"grep": true, "fgrep": true, "egrep": true,
		"alert": true, "which": true, "vi": true, "vim": true,
		"cp": true, "mv": true, "rm": true,
	}
)

// DefaultCandidateFiles returns the list of candidate shell profile paths in the user's home directory.
func DefaultCandidateFiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	candidates := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_aliases"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".aliases"),
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".config", "fish", "config.fish"),
	}

	var existing []string
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			existing = append(existing, c)
		}
	}
	return existing
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

	for _, file := range s.files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		relPath := file
		if homeDir != "" && strings.HasPrefix(file, homeDir) {
			relPath = "~" + file[len(homeDir):]
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			matches := aliasRegex.FindStringSubmatch(line)
			if len(matches) != 3 {
				continue
			}

			aliasName := strings.TrimSpace(matches[1])
			cmdRaw := strings.Trim(strings.TrimSpace(matches[2]), `"'`)

			if ignoredNames[strings.ToLower(aliasName)] {
				continue
			}

			lowerCmd := strings.ToLower(cmdRaw)

			// 1. SSH Server Alias (e.g. alias hestia="ssh root@hestia.codoly.fr" or alias sandbox="ssh root@192.0.0.15")
			if strings.HasPrefix(lowerCmd, "ssh ") || strings.HasPrefix(lowerCmd, "ssh\t") {
				srv := parseSSHCommand(aliasName, cmdRaw)
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
				}
				continue
			}

			// 2. Directory Navigation Alias (e.g. alias p="cd /home/ubuntu/projects/..." or alias mydocs="cd ~/Documents")
			if strings.HasPrefix(lowerCmd, "cd ") || strings.HasPrefix(lowerCmd, "z ") {
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
				}
				continue
			}

			// 3. Command Snippet Alias (e.g. alias d="pnpm dev" or alias dps="docker ps")
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
		if p == "-p" && i+1 < len(parts) {
			if pt, err := strconv.Atoi(parts[i+1]); err == nil {
				port = pt
			}
			i++
			continue
		} else if strings.HasPrefix(p, "-p") && len(p) > 2 {
			if pt, err := strconv.Atoi(p[2:]); err == nil {
				port = pt
			}
			continue
		}

		if p == "-i" && i+1 < len(parts) {
			keyPath = parts[i+1]
			i++
			continue
		} else if strings.HasPrefix(p, "-i") && len(p) > 2 {
			keyPath = p[2:]
			continue
		}

		// Positional argument (user@host or host)
		if !strings.HasPrefix(p, "-") && hostname == "" {
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
	if len(parts) < 2 {
		return ""
	}
	pathArg := strings.Trim(parts[1], `"'`)
	if strings.HasPrefix(pathArg, "~/") && homeDir != "" {
		pathArg = filepath.Join(homeDir, pathArg[2:])
	} else if pathArg == "~" && homeDir != "" {
		pathArg = homeDir
	}
	return filepath.Clean(pathArg)
}
