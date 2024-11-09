package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Dolyyyy/deck/pkg/models"
)

// ParseSSHConfig reads and parses an OpenSSH config file into a slice of Server models.
func ParseSSHConfig(filePath string) ([]*models.Server, error) {
	if filePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot find home dir: %w", err)
		}
		filePath = filepath.Join(home, ".ssh", "config")
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*models.Server{}, nil
		}
		return nil, fmt.Errorf("failed to open ssh config: %w", err)
	}
	defer file.Close()

	var servers []*models.Server
	var currentServer *models.Server

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split line by whitespace or '='
		var key, value string
		if idx := strings.IndexAny(line, " \t="); idx != -1 {
			key = strings.ToLower(strings.TrimSpace(line[:idx]))
			value = strings.TrimSpace(strings.TrimPrefix(line[idx:], "="))
			value = strings.TrimSpace(value)
		} else {
			key = strings.ToLower(line)
		}

		if key == "host" {
			// If we had a previous valid server, add it
			if currentServer != nil && isValidHost(currentServer.Name) {
				servers = append(servers, currentServer)
			}

			// Some host lines have multiple patterns: "Host srv1 srv2"
			hosts := strings.Fields(value)
			if len(hosts) > 0 {
				currentServer = &models.Server{
					Name:         hosts[0],
					Hostname:     hosts[0],
					Port:         22,
					Source:       "ssh_config",
					Status:       models.StatusUnknown,
					ExtraOptions: make(map[string]string),
				}
			} else {
				currentServer = nil
			}
			continue
		}

		if currentServer == nil {
			continue
		}

		switch key {
		case "hostname":
			currentServer.Hostname = value
		case "user":
			currentServer.User = value
		case "port":
			if p, err := strconv.Atoi(value); err == nil {
				currentServer.Port = p
			}
		case "identityfile":
			currentServer.IdentityFile = expandTilde(value)
		case "proxyjump":
			currentServer.ProxyJump = value
		default:
			currentServer.ExtraOptions[key] = value
		}
	}

	if currentServer != nil && isValidHost(currentServer.Name) {
		servers = append(servers, currentServer)
	}

	return servers, scanner.Err()
}

func isValidHost(name string) bool {
	if name == "" || strings.Contains(name, "*") || strings.Contains(name, "?") {
		return false
	}
	return true
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
