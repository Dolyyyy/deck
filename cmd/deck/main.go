package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Dolyyyy/deck/internal/aliasdiscovery"
	"github.com/Dolyyyy/deck/internal/config"
	"github.com/Dolyyyy/deck/internal/crypto"
	"github.com/Dolyyyy/deck/internal/endpoint"
	"github.com/Dolyyyy/deck/internal/logger"
	"github.com/Dolyyyy/deck/internal/ping"
	"github.com/Dolyyyy/deck/internal/snippet"
	"github.com/Dolyyyy/deck/internal/ssh"
	"github.com/Dolyyyy/deck/internal/transfer"
	"github.com/Dolyyyy/deck/internal/tui"
	"github.com/Dolyyyy/deck/internal/tui/styles"
	"github.com/Dolyyyy/deck/internal/updater"
	"github.com/Dolyyyy/deck/pkg/models"
)

var (
	Version   = "0.2.1"
	BuildDate = "2026-09-11"
)

func main() {
	logger.Init(false)
	cfgMgr := config.NewManager("", "")
	vault := crypto.NewVault("")
	transferSvc := transfer.NewService(vault)
	executor := ssh.NewExecutor()

	if len(os.Args) < 2 {
		runTUI(cfgMgr, executor, "")
		return
	}

	command := os.Args[1]

	switch command {
	case "version", "--version", "-v":
		fmt.Printf("deck version %s (built %s)\n", Version, BuildDate)

	case "help", "--help", "-h":
		printHelp()

	case "update":
		handleUpdate(cfgMgr)

	case "ls", "list":
		handleList(cfgMgr)

	case "path":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: deck path <alias>")
			os.Exit(1)
		}
		handlePath(cfgMgr, os.Args[2])

	case "init":
		shell := "bash"
		if len(os.Args) >= 3 {
			shell = os.Args[2]
		}
		fmt.Print(config.ShellIntegrationScript(shell))

	case "completion":
		shell := "bash"
		if len(os.Args) >= 3 {
			shell = os.Args[2]
		}
		printCompletion(shell)

	case "alias":
		handleAlias(cfgMgr, os.Args[2:])

	case "scan":
		handleScan(cfgMgr, os.Args[2:])

	case "tunnel":
		handleTunnel(cfgMgr, os.Args[2:])

	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: deck run <snippet_name>")
			os.Exit(1)
		}
		handleRun(cfgMgr, os.Args[2])

	case "endpoint":
		handleEndpoint(cfgMgr, os.Args[2:])

	case "ping":
		handlePing(cfgMgr)

	case "export":
		handleExport(cfgMgr, transferSvc, os.Args[2:])

	case "import":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: deck import <base64_string> [--password <pass>]")
			os.Exit(1)
		}
		handleImport(cfgMgr, transferSvc, os.Args[2:])

	case "lock":
		handleLock(vault)

	default:
		target := command
		handleQuickJumpOrConnect(cfgMgr, executor, target)
	}
}

func runTUI(cfgMgr *config.Manager, executor *ssh.Executor, prefilter string) {
	app, err := tui.NewApp(cfgMgr, Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize deck: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	p := tea.NewProgram(app, tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running deck TUI: %v\n", err)
		os.Exit(1)
	}

	finalModel := m.(*tui.AppModel)

	if finalModel.TargetServer != nil {
		fmt.Printf("🚀 Connecting to %s (%s)...\n", finalModel.TargetServer.Name, finalModel.TargetServer.DisplayAddress())
		if err := executor.Connect(finalModel.TargetServer); err != nil {
			fmt.Fprintf(os.Stderr, "SSH connection ended with error: %v\n", err)
		}
	}

	if finalModel.TargetDir != "" {
		jumpToDirectory(finalModel.TargetDir)
	}

	if finalModel.TargetSnippet != nil {
		snip := finalModel.TargetSnippet
		snipRunner := snippet.NewRunner()
		fmt.Printf("⚡ Executing snippet %q: %s\n", snip.Name, snip.Command)
		cmd, err := snipRunner.BuildCommand(snip)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to prepare command: %v\n", err)
		} else {
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Snippet execution ended: %v\n", err)
			}
		}
	}
}

func handleUpdate(cfgMgr *config.Manager) {
	up := updater.NewUpdater(cfgMgr.ConfigFilePath())
	fmt.Printf("🔍 Checking for updates (current version: %s)...\n", Version)

	rel, hasNew, err := up.CheckUpdate(Version, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check for updates: %v\n", err)
		os.Exit(1)
	}

	if !hasNew {
		fmt.Printf("✓ Deck is already up to date (%s)!\n", Version)
		return
	}

	fmt.Printf("⬇ New release found: %s! Downloading and installing...\n", rel.TagName)
	newVer, err := up.SelfUpdate(Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully updated to %s! Run 'deck' to launch.\n", newVer)
}

func handleQuickJumpOrConnect(cfgMgr *config.Manager, executor *ssh.Executor, query string) {
	// 1. Check Directory Aliases (exact match)
	if alias, err := cfgMgr.FindAlias(query); err == nil {
		jumpToDirectory(alias.ExpandedPath())
		return
	}

	// 2. Load Servers
	servers, err := cfgMgr.LoadAllServers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load servers: %v\n", err)
		os.Exit(1)
	}

	// 3. Exact Server Name match
	for _, s := range servers {
		if strings.EqualFold(s.Name, query) {
			fmt.Printf("⚡ Quick connect to %s (%s)...\n", s.Name, s.DisplayAddress())
			if err := executor.Connect(s); err != nil {
				fmt.Fprintf(os.Stderr, "SSH connection error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	// 4. Check if query is a typo of a known built-in command (e.g. "hell" -> "help")
	knownCommands := []string{"help", "version", "update", "ls", "list", "path", "init", "completion", "alias", "tunnel", "run", "endpoint", "ping", "export", "import", "lock"}
	closestCmd, cmdDist := findClosestMatch(query, knownCommands)
	if cmdDist <= 1 {
		fmt.Fprintf(os.Stderr, "deck: unknown command: %q\n\n", query)
		fmt.Fprintf(os.Stderr, "💡 Did you mean:  %s  ?\n", styles.KeyStyle.Render("deck "+closestCmd))
		fmt.Fprintf(os.Stderr, "   Run 'deck %s' to execute.\n\n", closestCmd)
		fmt.Fprintln(os.Stderr, "Run 'deck help' to view all available commands.")
		os.Exit(1)
	}

	// 5. Match Server by Name prefix or Hostname prefix (NEVER match tags or environments in CLI!)
	qLower := strings.ToLower(query)
	var matches []*models.Server
	for _, s := range servers {
		sName := strings.ToLower(s.Name)
		sHost := strings.ToLower(s.Hostname)
		if strings.HasPrefix(sName, qLower) || strings.HasPrefix(sHost, qLower) {
			matches = append(matches, s)
		}
	}

	if len(matches) == 1 {
		srv := matches[0]
		fmt.Printf("⚡ Quick connect to %s (%s)...\n", srv.Name, srv.DisplayAddress())
		if err := executor.Connect(srv); err != nil {
			fmt.Fprintf(os.Stderr, "SSH connection error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(matches) > 1 {
		runTUI(cfgMgr, executor, query)
		return
	}

	// 6. Not found -> Provide intelligent fuzzy correction suggestion
	deckCfg, _ := cfgMgr.LoadDeckConfig()
	var candidates []string
	candidates = append(candidates, knownCommands...)
	for _, s := range servers {
		candidates = append(candidates, s.Name)
	}
	for _, a := range deckCfg.Aliases {
		candidates = append(candidates, a.Name)
	}
	for _, sn := range deckCfg.Snippets {
		candidates = append(candidates, sn.Name)
	}

	bestMatch, dist := findClosestMatch(query, candidates)
	fmt.Fprintf(os.Stderr, "deck: unknown command, server or alias: %q\n\n", query)
	if dist <= 3 && bestMatch != "" {
		fmt.Fprintf(os.Stderr, "💡 Did you mean:  %s  ?\n", styles.KeyStyle.Render("deck "+bestMatch))
		fmt.Fprintf(os.Stderr, "   Run 'deck %s' to execute.\n\n", bestMatch)
	}
	fmt.Fprintln(os.Stderr, "Run 'deck help' to view all available commands.")
	os.Exit(1)
}

func handleList(cfgMgr *config.Manager) {
	servers, _ := cfgMgr.LoadAllServers()
	deckCfg, _ := cfgMgr.LoadDeckConfig()

	header := styles.HeaderStyle.Render("📦 DECK COCKPIT INVENTORY & LIVE HEALTH")
	fmt.Println(header)
	fmt.Println()

	// 1. Live probe all servers concurrently
	serverPingMap := make(map[string]*models.PingResult)
	if len(servers) > 0 {
		pinger := ping.NewTCPPinger(1500 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		resChan := pinger.ProbeAll(ctx, servers, 15)
		for res := range resChan {
			serverPingMap[res.ServerName] = res
		}
	}

	// Servers Section with vibrant status colors
	fmt.Println(styles.BadgeTag.Render("🖥️  SSH SERVERS (" + fmt.Sprintf("%d", len(servers)) + ")"))
	for _, s := range servers {
		badge := styles.StatusUnknown.Render("○ UNKNOWN")
		lat := "     -"
		if res, ok := serverPingMap[s.Name]; ok {
			if res.Status == models.StatusOnline {
				badge = styles.StatusOnline.Render("● ONLINE ")
				lat = fmt.Sprintf("%4dms", res.Latency.Milliseconds())
			} else {
				badge = styles.StatusOffline.Render("● OFFLINE")
				lat = "timeout"
			}
		}

		env := s.Environment
		if env == "" {
			env = "default"
		}
		tags := ""
		if len(s.Tags) > 0 {
			tags = fmt.Sprintf("[%s]", strings.Join(s.Tags, ", "))
		}

		fmt.Printf("  %s [%s] %-20s %-28s %-10s %s\n",
			badge,
			lat,
			styles.KeyStyle.Render(s.Name),
			s.DisplayAddress(),
			styles.BadgeEnv.Render(env),
			tags,
		)
	}
	fmt.Println()

	// Aliases Section with live directory existence check
	fmt.Println(styles.BadgeTag.Render("📁 DIRECTORY ALIASES (" + fmt.Sprintf("%d", len(deckCfg.Aliases)) + ")"))
	for _, a := range deckCfg.Aliases {
		status := styles.StatusOnline.Render("✓ EXISTS ")
		if !a.Exists() {
			status = styles.StatusOffline.Render("✗ MISSING")
		}
		fmt.Printf("  %s %-18s %-38s %s\n",
			status,
			styles.KeyStyle.Render(a.Name),
			a.Path,
			a.Description,
		)
	}
	fmt.Println()

	// Tunnels Section
	if len(deckCfg.Tunnels) > 0 {
		fmt.Println(styles.BadgeTag.Render("🔀 SSH TUNNELS (" + fmt.Sprintf("%d", len(deckCfg.Tunnels)) + ")"))
		for _, t := range deckCfg.Tunnels {
			status := styles.FormatTunnelStatus(t.Active, t.PID)
			fmt.Printf("  %s %-18s %-16s 127.0.0.1:%d -> %s:%d\n",
				status,
				styles.KeyStyle.Render(t.Name),
				t.ServerName,
				t.LocalPort,
				t.RemoteHost,
				t.RemotePort,
			)
		}
		fmt.Println()
	}

	// Snippets Section
	if len(deckCfg.Snippets) > 0 {
		fmt.Println(styles.BadgeTag.Render("⚡ SNIPPETS & RUNBOOKS (" + fmt.Sprintf("%d", len(deckCfg.Snippets)) + ")"))
		for _, sn := range deckCfg.Snippets {
			target := "local"
			if sn.TargetHost != "" {
				target = sn.TargetHost
			}
			fmt.Printf("  • %-18s [%-10s] %s\n",
				styles.KeyStyle.Render(sn.Name),
				target,
				sn.Command,
			)
		}
		fmt.Println()
	}
}

func handlePath(cfgMgr *config.Manager, aliasName string) {
	alias, err := cfgMgr.FindAlias(aliasName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(alias.ExpandedPath())
}

func jumpToDirectory(dirPath string) {
	if _, err := os.Stat(dirPath); err != nil {
		fmt.Fprintf(os.Stderr, "Directory %q does not exist\n", dirPath)
		os.Exit(1)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	fmt.Printf("📂 Dropping into interactive shell in %s...\n", dirPath)
	_ = os.Chdir(dirPath)
	cmd := exec.Command(shell)
	cmd.Dir = dirPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func handleAlias(cfgMgr *config.Manager, args []string) {
	if len(args) == 0 || args[0] == "ls" || args[0] == "list" {
		deckCfg, _ := cfgMgr.LoadDeckConfig()
		fmt.Printf("📁 Configured Directory Aliases (%d):\n", len(deckCfg.Aliases))
		for _, a := range deckCfg.Aliases {
			status := "✓"
			if !a.Exists() {
				status = "✗"
			}
			fmt.Printf("  • %-16s %s -> %s\n", a.Name, status, a.Path)
		}
		return
	}

	switch args[0] {
	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: deck alias add <name> <path> [description]")
			os.Exit(1)
		}
		targetPath := args[2]
		if abs, err := filepath.Abs(targetPath); err == nil {
			targetPath = abs
		}
		desc := ""
		if len(args) >= 4 {
			desc = strings.Join(args[3:], " ")
		}
		alias := &models.DirectoryAlias{
			Name:        args[1],
			Path:        targetPath,
			Description: desc,
		}
		if err := cfgMgr.AddOrUpdateAlias(alias); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save alias: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Added alias %q -> %s\n", alias.Name, alias.Path)

	case "rm", "remove", "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: deck alias rm <name>")
			os.Exit(1)
		}
		if err := cfgMgr.RemoveAlias(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove alias: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Removed alias %q\n", args[1])

	case "scan":
		handleScan(cfgMgr, args[1:])
	}
}

func handleScan(cfgMgr *config.Manager, args []string) {
	var customFiles []string
	autoImport := false

	for _, a := range args {
		if a == "-y" || a == "--yes" {
			autoImport = true
		} else if !strings.HasPrefix(a, "-") {
			customFiles = append(customFiles, a)
		}
	}

	scanner := aliasdiscovery.NewScanner(customFiles...)
	servers, _ := cfgMgr.LoadAllServers()
	deckCfg, _ := cfgMgr.LoadDeckConfig()
	if deckCfg == nil {
		deckCfg = &config.DeckConfig{}
	}

	fmt.Println("🔍 Scanning shell configuration files and profiles...")
	items, err := scanner.Discover(servers, deckCfg.Aliases, deckCfg.Snippets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan error: %v\n", err)
		return
	}

	if len(items) == 0 {
		fmt.Println("✓ No new unimported aliases or servers found.")
		return
	}

	fmt.Printf("⚡ Discovered %d new items:\n\n", len(items))
	for _, it := range items {
		fmt.Printf("  • [%-15s] %-14s %-32s (%s)\n", it.Type, it.Name, it.Target, it.SourceFile)
	}

	if autoImport {
		imported := 0
		for _, it := range items {
			if it.Type == aliasdiscovery.TypeServer && it.Server != nil {
				_ = cfgMgr.AddOrUpdateServer(it.Server)
				imported++
			} else if it.Type == aliasdiscovery.TypeAlias && it.Alias != nil {
				_ = cfgMgr.AddOrUpdateAlias(it.Alias)
				imported++
			} else if it.Type == aliasdiscovery.TypeSnippet && it.Snippet != nil {
				_ = cfgMgr.AddOrUpdateSnippet(it.Snippet)
				imported++
			}
		}
		fmt.Printf("\n✓ Successfully imported %d items into Deck!\n", imported)
	} else {
		fmt.Println("\n💡 Tip: Run 'deck scan -y' or open Deck cockpit ('deck') to import them interactively.")
	}
}

func handleTunnel(cfgMgr *config.Manager, args []string) {
	deckCfg, _ := cfgMgr.LoadDeckConfig()
	if len(args) == 0 || args[0] == "ls" || args[0] == "list" {
		fmt.Printf("🔀 SSH Tunnels (%d):\n", len(deckCfg.Tunnels))
		for _, t := range deckCfg.Tunnels {
			status := styles.FormatTunnelStatus(t.Active, t.PID)
			fmt.Printf("  %s %-16s %-14s 127.0.0.1:%d -> %s:%d\n",
				status, t.Name, t.ServerName, t.LocalPort, t.RemoteHost, t.RemotePort)
		}
		return
	}
}

func handleRun(cfgMgr *config.Manager, snippetName string) {
	deckCfg, _ := cfgMgr.LoadDeckConfig()
	var found *models.Snippet
	for _, sn := range deckCfg.Snippets {
		if strings.EqualFold(sn.Name, snippetName) {
			found = sn
			break
		}
	}
	if found == nil {
		fmt.Fprintf(os.Stderr, "Snippet %q not found\n", snippetName)
		os.Exit(1)
	}

	runner := snippet.NewRunner()
	fmt.Printf("⚡ Executing %s: %s\n", found.Name, found.Command)
	cmd, err := runner.BuildCommand(found)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
		os.Exit(1)
	}
}

func handleEndpoint(cfgMgr *config.Manager, args []string) {
	deckCfg, _ := cfgMgr.LoadDeckConfig()
	if len(deckCfg.Endpoints) == 0 {
		fmt.Println("No endpoints configured. Use 'deck' TUI [5] tab to add web endpoints.")
		return
	}

	prober := endpoint.NewProber(3 * time.Second)
	fmt.Printf("🌐 Probing %d endpoints...\n\n", len(deckCfg.Endpoints))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := prober.ProbeAll(ctx, deckCfg.Endpoints, 4)
	for ep := range results {
		statusStr := styles.FormatEndpointStatus(ep.Status, ep.StatusCode)
		latStr := styles.FormatLatency(ep.Latency)
		sslStr := styles.FormatSSLExpiry(ep.SSLDaysRemaining)
		fmt.Printf("  %s [%s] %-20s %-32s (%s)\n", statusStr, latStr, ep.Name, ep.URL, sslStr)
	}
}

func handlePing(cfgMgr *config.Manager) {
	servers, _ := cfgMgr.LoadAllServers()
	if len(servers) == 0 {
		fmt.Println("No servers found in inventory.")
		return
	}

	fmt.Printf("📡 Probing %d hosts asynchronously...\n\n", len(servers))
	pinger := ping.NewTCPPinger(2 * time.Second)
	ctx := context.Background()

	results := pinger.ProbeAll(ctx, servers, 15)
	for res := range results {
		statusBadge := styles.StatusOnline.Render("● ONLINE ")
		if res.Status == models.StatusOffline {
			statusBadge = styles.StatusOffline.Render("● OFFLINE")
		}
		latencyStr := styles.FormatLatency(res.Latency)
		if res.Status == models.StatusOffline {
			latencyStr = styles.LatencyMuted.Render("timeout")
		}
		fmt.Printf("  %s  [%s]  %-22s\n", statusBadge, latencyStr, res.ServerName)
	}
}

func handleExport(cfgMgr *config.Manager, svc *transfer.Service, args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	pwdFlag := fs.String("password", "", "Password to encrypt export payload with AES-256-GCM")
	jsonFlag := fs.Bool("json", false, "Output unencrypted raw JSON instead of Base64")
	_ = fs.Parse(args)

	servers, _ := cfgMgr.LoadAllServers()
	deckCfg, _ := cfgMgr.LoadDeckConfig()

	data := &models.ExportData{
		Servers:   make([]models.Server, 0, len(servers)),
		Aliases:   make([]models.DirectoryAlias, 0, len(deckCfg.Aliases)),
		Tunnels:   make([]models.Tunnel, 0, len(deckCfg.Tunnels)),
		Snippets:  make([]models.Snippet, 0, len(deckCfg.Snippets)),
		Endpoints: make([]models.Endpoint, 0, len(deckCfg.Endpoints)),
	}
	for _, s := range servers {
		data.Servers = append(data.Servers, *s)
	}
	for _, a := range deckCfg.Aliases {
		data.Aliases = append(data.Aliases, *a)
	}
	for _, t := range deckCfg.Tunnels {
		data.Tunnels = append(data.Tunnels, *t)
	}
	for _, sn := range deckCfg.Snippets {
		data.Snippets = append(data.Snippets, *sn)
	}
	for _, ep := range deckCfg.Endpoints {
		data.Endpoints = append(data.Endpoints, *ep)
	}

	if *jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(data)
		return
	}

	b64, err := svc.ExportBase64(data, *pwdFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(b64)
}

func handleImport(cfgMgr *config.Manager, svc *transfer.Service, args []string) {
	rawB64 := args[0]
	var pwd string
	for i, a := range args {
		if (a == "-p" || a == "--password") && i+1 < len(args) {
			pwd = args[i+1]
			break
		}
	}

	data, err := svc.ImportBase64(rawB64, pwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Import failed: %v\n", err)
		os.Exit(1)
	}

	deckCfg, _ := cfgMgr.LoadDeckConfig()
	for _, s := range data.Servers {
		srv := s
		deckCfg.Servers = append(deckCfg.Servers, &srv)
	}
	for _, a := range data.Aliases {
		alias := a
		deckCfg.Aliases = append(deckCfg.Aliases, &alias)
	}
	for _, t := range data.Tunnels {
		tun := t
		deckCfg.Tunnels = append(deckCfg.Tunnels, &tun)
	}
	for _, sn := range data.Snippets {
		snp := sn
		deckCfg.Snippets = append(deckCfg.Snippets, &snp)
	}
	for _, ep := range data.Endpoints {
		endp := ep
		deckCfg.Endpoints = append(deckCfg.Endpoints, &endp)
	}

	if err := cfgMgr.SaveDeckConfig(deckCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save imported config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully imported %d servers, %d aliases, %d tunnels, %d snippets!\n",
		len(data.Servers), len(data.Aliases), len(data.Tunnels), len(data.Snippets))
}

func handleLock(vault *crypto.Vault) {
	fmt.Print("Enter new master password to protect your Deck: ")
	var pass string
	_, _ = fmt.Scanln(&pass)
	if strings.TrimSpace(pass) == "" {
		fmt.Println("Password cannot be empty.")
		return
	}
	if err := vault.SetMasterPassword(pass); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set master password: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("🔒 Master password configured successfully!")
}

func printCompletion(shell string) {
	switch strings.ToLower(shell) {
	case "zsh":
		fmt.Print(`#compdef deck
_deck() {
    local -a commands
    commands=(
        'ls:List all servers and aliases with live health status'
        'ping:Concurrent latency test of all SSH servers'
        'alias:Manage directory bookmark shortcuts'
        'tunnel:Manage SSH port forwarding tunnels'
        'run:Execute a saved command snippet'
        'endpoint:Probe web services and SSL certificates'
        'export:Export inventory to Base64 (optional AES-256-GCM)'
        'import:Import inventory from Base64 string'
        'update:Self-update deck to latest GitHub release'
        'init:Output shell integration script'
        'lock:Configure master password'
        'version:Show version'
        'help:Show help'
    )
    _describe -t commands 'deck command' commands
}
_deck "$@"
`)
	default:
		fmt.Print(`# bash completion for deck
_deck_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local commands="ls ping alias tunnel run endpoint export import update lock init path version help"
    COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
}
complete -F _deck_completions deck
`)
	}
}

// Levenshtein distance for fuzzy suggestions
func levenshtein(s1, s2 string) int {
	s1, s2 = strings.ToLower(s1), strings.ToLower(s2)
	r1, r2 := []rune(s1), []rune(s2)
	m, n := len(r1), len(r2)
	d := make([][]int, m+1)
	for i := range d {
		d[i] = make([]int, n+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			d[i][j] = int(math.Min(
				float64(d[i-1][j]+1),
				math.Min(float64(d[i][j-1]+1), float64(d[i-1][j-1]+cost)),
			))
		}
	}
	return d[m][n]
}

func findClosestMatch(query string, candidates []string) (string, int) {
	best := ""
	minDist := 999
	for _, c := range candidates {
		dist := levenshtein(query, c)
		if dist < minDist {
			minDist = dist
			best = c
		}
	}
	return best, minDist
}

func printHelp() {
	fmt.Println(`deck — The high-performance terminal cockpit for SSH, bookmarks, tunnels, snippets & SSL.

Usage:
  deck                          Launch full interactive TUI cockpit (5 tabs)
  deck <name>                   Quick connect to SSH server or jump to directory alias
  deck ls                       List all servers and aliases with live health check
  deck update                   Check and self-update to latest GitHub release
  deck run <snippet>            Execute a saved command snippet
  deck endpoint                 Probe configured web endpoints and check SSL cert validity
  deck tunnel                   View configured SSH tunnels
  deck alias add <name> <path>  Bookmark a local directory shortcut
  deck alias ls                 List all directory aliases
  deck path <alias>             Output directory path for shell cd
  deck ping                     Test connectivity and latency to all servers
  deck export                   Generate Base64 export payload (supports --password and --json)
  deck import <base64>          Import servers, aliases, tunnels, snippets from Base64
  deck init <bash|zsh|fish>     Print shell integration function for instant directory cd
  deck completion <bash|zsh>    Output shell completion script
  deck lock                     Set a master password to protect your deck
  deck version                  Display version info

Cockpit Tabs in TUI:
  [1] SSH Servers               [2] Directory Aliases     [3] SSH Tunnels
  [4] Command Snippets          [5] Web Endpoints & SSL   [6] Settings

Keybindings in TUI:
  [1..5] Direct tab switch      [Tab] Cycle tabs          [Enter] Connect / Jump / Run
  [Space] Toggle tunnel         [c] Copy snippet          [a] Add entry
  [e] Edit focused entry        [x] Delete entry          [/] Filter tab
  [r/p] Re-probe hosts/URLs     [U] Update Deck           [?] Help
  [q] Quit Cockpit`)
}
