package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Dolyyyy/deck/internal/config"
	"github.com/Dolyyyy/deck/internal/crypto"
	"github.com/Dolyyyy/deck/internal/endpoint"
	"github.com/Dolyyyy/deck/internal/ping"
	"github.com/Dolyyyy/deck/internal/snippet"
	"github.com/Dolyyyy/deck/internal/ssh"
	"github.com/Dolyyyy/deck/internal/transfer"
	"github.com/Dolyyyy/deck/internal/tunnel"
	"github.com/Dolyyyy/deck/internal/tui/styles"
	"github.com/Dolyyyy/deck/internal/updater"
	"github.com/Dolyyyy/deck/pkg/models"
)

type activeTab int

const (
	tabServers activeTab = iota
	tabAliases
	tabTunnels
	tabSnippets
	tabEndpoints
)

type activeModal int

const (
	modalNone activeModal = iota
	modalInspect
	modalExport
	modalImport
	modalHelp
	modalForm
)

type pingResultMsg struct {
	res *models.PingResult
}

type endpointResultMsg struct {
	ep *models.Endpoint
}

type statsResultMsg struct {
	serverName string
	stats      *models.ServerStats
	err        error
}

type updateCheckMsg struct {
	release *updater.ReleaseInfo
	hasNew  bool
	err     error
}

type selfUpdateResultMsg struct {
	newVersion string
	err        error
}

type formField struct {
	label string
	input textinput.Model
}

type modalFormState struct {
	title    string
	fields   []formField
	focused  int
	isEdit   bool
	origName string
}

// AppModel represents the state of the Deck terminal interface.
type AppModel struct {
	cfgManager     *config.Manager
	pinger         *ping.TCPPinger
	executor       *ssh.Executor
	transferSvc    *transfer.Service
	vault          *crypto.Vault
	tunnelMgr      *tunnel.Manager
	snippetRunner  *snippet.Runner
	endpointProber *endpoint.Prober
	updaterSvc     *updater.Updater

	servers         []*models.Server
	filteredServers []*models.Server

	aliases         []*models.DirectoryAlias
	filteredAliases []*models.DirectoryAlias

	tunnels         []*models.Tunnel
	filteredTunnels []*models.Tunnel

	snippets         []*models.Snippet
	filteredSnippets []*models.Snippet

	endpoints         []*models.Endpoint
	filteredEndpoints []*models.Endpoint

	currentTab   activeTab
	currentModal activeModal

	serverTable   table.Model
	aliasTable    table.Model
	tunnelTable   table.Model
	snippetTable  table.Model
	endpointTable table.Model

	searchInput   textinput.Model
	searching     bool

	// Modal states
	form            modalFormState
	passwordInput   textinput.Model
	genericInput    textinput.Model
	exportB64Result string
	modalMessage    string
	inspectStats    *models.ServerStats
	inspecting      bool

	updateAvailable bool
	latestRelease   *updater.ReleaseInfo
	statusToast     string
	width           int
	height          int
	keys            KeyMap

	// Action to execute upon TUI exit
	TargetServer  *models.Server
	TargetDir     string
	TargetSnippet *models.Snippet
}

// NewApp creates a new Bubbletea model for deck.
func NewApp(cfgManager *config.Manager) (*AppModel, error) {
	if cfgManager == nil {
		cfgManager = config.NewManager("", "")
	}

	servers, err := cfgManager.LoadAllServers()
	if err != nil {
		servers = []*models.Server{}
	}

	deckCfg, err := cfgManager.LoadDeckConfig()
	if err != nil {
		deckCfg = &config.DeckConfig{}
	}

	ti := textinput.New()
	ti.Placeholder = "Type to filter hosts, aliases, tags... (Esc to clear)"
	ti.Prompt = "  / "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.AccentPrimary).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.TextNormal)

	pwd := textinput.New()
	pwd.Placeholder = "Encryption passphrase (optional)..."
	pwd.EchoMode = textinput.EchoPassword
	pwd.EchoCharacter = '•'

	m := &AppModel{
		cfgManager:        cfgManager,
		pinger:            ping.NewTCPPinger(1500 * time.Millisecond),
		executor:          ssh.NewExecutor(),
		vault:             crypto.NewVault(""),
		tunnelMgr:         tunnel.NewManager(),
		snippetRunner:     snippet.NewRunner(),
		endpointProber:    endpoint.NewProber(3 * time.Second),
		updaterSvc:        updater.NewUpdater(cfgManager.ConfigFilePath()),
		servers:           servers,
		filteredServers:   servers,
		aliases:           deckCfg.Aliases,
		filteredAliases:   deckCfg.Aliases,
		tunnels:           deckCfg.Tunnels,
		filteredTunnels:   deckCfg.Tunnels,
		snippets:          deckCfg.Snippets,
		filteredSnippets:  deckCfg.Snippets,
		endpoints:         deckCfg.Endpoints,
		filteredEndpoints: deckCfg.Endpoints,
		currentTab:        tabServers,
		currentModal:      modalNone,
		searchInput:       ti,
		passwordInput:     pwd,
		width:             100,
		height:            28,
		keys:              DefaultKeyMap(),
	}
	m.transferSvc = transfer.NewService(m.vault)

	m.initTables()
	return m, nil
}

func (m *AppModel) initTables() {
	sStyle := table.DefaultStyles()
	sStyle.Header = sStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.BorderMuted).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.TextBold)

	// Sleek selection row background without overriding cell text foreground colors
	sStyle.Selected = sStyle.Selected.
		Background(styles.BgSelected).
		Bold(true)

	sStyle.Cell = sStyle.Cell.
		Foreground(styles.TextNormal)

	m.serverTable = table.New(table.WithFocused(true), table.WithStyles(sStyle))
	m.aliasTable = table.New(table.WithFocused(false), table.WithStyles(sStyle))
	m.tunnelTable = table.New(table.WithFocused(false), table.WithStyles(sStyle))
	m.snippetTable = table.New(table.WithFocused(false), table.WithStyles(sStyle))
	m.endpointTable = table.New(table.WithFocused(false), table.WithStyles(sStyle))

	m.resizeTables(m.width, m.height)
	m.updateServerRows()
	m.updateAliasRows()
	m.updateTunnelRows()
	m.updateSnippetRows()
	m.updateEndpointRows()
}

func (m *AppModel) resizeTables(width, height int) {
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 24
	}

	usableWidth := width - 4
	if usableWidth < 60 {
		usableWidth = 60
	}
	tableHeight := height - 9
	if tableHeight < 5 {
		tableHeight = 5
	}

	m.searchInput.Width = usableWidth - 8

	// Dynamic Server Columns
	serverCols := []table.Column{
		{Title: "STATUS", Width: 16},
		{Title: "NAME", Width: 18},
		{Title: "TARGET", Width: 28},
		{Title: "USER", Width: 10},
		{Title: "ENV", Width: 10},
		{Title: "LATENCY", Width: 10},
		{Title: "TAGS", Width: 16},
	}
	m.serverTable.SetColumns(serverCols)
	m.serverTable.SetWidth(usableWidth)
	m.serverTable.SetHeight(tableHeight)

	// Dynamic Alias Columns
	aliasW := 20
	existsW := 12
	pathW := (usableWidth - aliasW - existsW - 4) / 2
	if pathW < 20 {
		pathW = 20
	}
	descW := usableWidth - (aliasW + pathW + existsW + 4)
	if descW < 15 {
		descW = 15
	}
	aliasCols := []table.Column{
		{Title: "ALIAS", Width: aliasW},
		{Title: "DIRECTORY PATH", Width: pathW},
		{Title: "STATUS", Width: existsW},
		{Title: "DESCRIPTION", Width: descW},
	}
	m.aliasTable.SetColumns(aliasCols)
	m.aliasTable.SetWidth(usableWidth)
	m.aliasTable.SetHeight(tableHeight)

	// Dynamic Tunnel Columns
	tunnelCols := []table.Column{
		{Title: "STATUS", Width: 18},
		{Title: "NAME", Width: 18},
		{Title: "SERVER", Width: 16},
		{Title: "LOCAL PORT", Width: 12},
		{Title: "FORWARDING", Width: 32},
		{Title: "TYPE", Width: 10},
	}
	m.tunnelTable.SetColumns(tunnelCols)
	m.tunnelTable.SetWidth(usableWidth)
	m.tunnelTable.SetHeight(tableHeight)

	// Dynamic Snippet Columns
	snipW := 22
	targetW := 14
	tagsW := 18
	cmdW := usableWidth - (snipW + targetW + tagsW + 4)
	if cmdW < 20 {
		cmdW = 20
	}
	snippetCols := []table.Column{
		{Title: "NAME", Width: snipW},
		{Title: "COMMAND", Width: cmdW},
		{Title: "TARGET", Width: targetW},
		{Title: "TAGS", Width: tagsW},
	}
	m.snippetTable.SetColumns(snippetCols)
	m.snippetTable.SetWidth(usableWidth)
	m.snippetTable.SetHeight(tableHeight)

	// Dynamic Endpoint Columns
	epCols := []table.Column{
		{Title: "STATUS", Width: 18},
		{Title: "NAME", Width: 20},
		{Title: "URL", Width: 36},
		{Title: "LATENCY", Width: 10},
		{Title: "SSL EXPIRY", Width: 18},
	}
	m.endpointTable.SetColumns(epCols)
	m.endpointTable.SetWidth(usableWidth)
	m.endpointTable.SetHeight(tableHeight)
}

func (m *AppModel) updateServerRows() {
	var rows []table.Row
	cursorIdx := m.serverTable.Cursor()

	for i, s := range m.filteredServers {
		prefix := "  "
		if i == cursorIdx {
			prefix = styles.CursorIndicator.Render("❯ ")
		}

		statusStr := styles.FormatServerStatus(s.Status, prefix)
		latencyStr := styles.FormatLatency(s.Latency)

		user := s.User
		if user == "" {
			user = "-"
		}

		env := s.Environment
		if env == "" {
			env = "default"
		}

		tags := strings.Join(s.Tags, ", ")
		if tags == "" {
			tags = "-"
		}

		rows = append(rows, table.Row{
			statusStr,
			s.Name,
			s.DisplayAddress(),
			user,
			styles.EnvPill.Render(env),
			latencyStr,
			styles.TagPill.Render(tags),
		})
	}
	m.serverTable.SetRows(rows)
}

func (m *AppModel) updateAliasRows() {
	var rows []table.Row
	cursorIdx := m.aliasTable.Cursor()

	for i, a := range m.filteredAliases {
		prefix := "  "
		if i == cursorIdx {
			prefix = styles.CursorIndicator.Render("❯ ")
		}

		statusStr := styles.StatusOffline.Render("✖ missing")
		if a.Exists() {
			statusStr = styles.StatusOnline.Render("✓ found")
		}

		desc := a.Description
		if desc == "" {
			desc = "-"
		}

		rows = append(rows, table.Row{
			prefix + a.Name,
			a.Path,
			statusStr,
			desc,
		})
	}
	m.aliasTable.SetRows(rows)
}

func (m *AppModel) updateTunnelRows() {
	var rows []table.Row
	cursorIdx := m.tunnelTable.Cursor()

	for i, t := range m.filteredTunnels {
		prefix := "  "
		if i == cursorIdx {
			prefix = styles.CursorIndicator.Render("❯ ")
		}

		statusStr := prefix + styles.FormatTunnelStatus(t.Active, t.PID)
		fwdStr := fmt.Sprintf("127.0.0.1:%d -> %s:%d", t.LocalPort, t.RemoteHost, t.RemotePort)
		tunnelType := t.Type
		if tunnelType == "" {
			tunnelType = "local"
		}

		rows = append(rows, table.Row{
			statusStr,
			t.Name,
			t.ServerName,
			fmt.Sprintf("%d", t.LocalPort),
			fwdStr,
			tunnelType,
		})
	}
	m.tunnelTable.SetRows(rows)
}

func (m *AppModel) updateSnippetRows() {
	var rows []table.Row
	cursorIdx := m.snippetTable.Cursor()

	for i, s := range m.filteredSnippets {
		prefix := "  "
		if i == cursorIdx {
			prefix = styles.CursorIndicator.Render("❯ ")
		}

		target := styles.LatencyMuted.Render("local")
		if s.TargetHost != "" {
			target = styles.BadgeTag.Render(s.TargetHost)
		}

		tags := strings.Join(s.Tags, ", ")
		if tags == "" {
			tags = "-"
		}

		rows = append(rows, table.Row{
			prefix + s.Name,
			s.Command,
			target,
			styles.TagPill.Render(tags),
		})
	}
	m.snippetTable.SetRows(rows)
}

func (m *AppModel) updateEndpointRows() {
	var rows []table.Row
	cursorIdx := m.endpointTable.Cursor()

	for i, ep := range m.filteredEndpoints {
		prefix := "  "
		if i == cursorIdx {
			prefix = styles.CursorIndicator.Render("❯ ")
		}

		statusStr := prefix + styles.FormatEndpointStatus(ep.Status, ep.StatusCode)
		latencyStr := styles.FormatLatency(ep.Latency)
		sslStr := styles.FormatSSLExpiry(ep.SSLDaysRemaining)

		rows = append(rows, table.Row{
			statusStr,
			ep.Name,
			ep.URL,
			latencyStr,
			sslStr,
		})
	}
	m.endpointTable.SetRows(rows)
}

func (m *AppModel) switchTab(tab activeTab) {
	m.currentTab = tab
	m.serverTable.Blur()
	m.aliasTable.Blur()
	m.tunnelTable.Blur()
	m.snippetTable.Blur()
	m.endpointTable.Blur()

	switch tab {
	case tabServers:
		m.serverTable.Focus()
	case tabAliases:
		m.aliasTable.Focus()
	case tabTunnels:
		m.tunnelTable.Focus()
	case tabSnippets:
		m.snippetTable.Focus()
	case tabEndpoints:
		m.endpointTable.Focus()
	}
	m.applyFilter(m.searchInput.Value())
}

// Close cleans up background resources (tunnels).
func (m *AppModel) Close() {
	if m.tunnelMgr != nil {
		m.tunnelMgr.StopAll()
	}
}

// Init starts initial probes and updates check.
func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.pingAllServersCmd(),
		m.probeAllEndpointsCmd(),
		m.checkUpdateCmd(),
	)
}

func (m *AppModel) pingAllServersCmd() tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.servers {
		s.Status = models.StatusChecking
		srv := s
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			res, _ := m.pinger.Probe(ctx, srv)
			return pingResultMsg{res: res}
		})
	}
	return tea.Batch(cmds...)
}

func (m *AppModel) probeAllEndpointsCmd() tea.Cmd {
	var cmds []tea.Cmd
	for _, ep := range m.endpoints {
		ep.Status = models.EndpointStatusChecking
		e := ep
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			res := m.endpointProber.Probe(ctx, e)
			return endpointResultMsg{ep: res}
		})
	}
	return tea.Batch(cmds...)
}

func (m *AppModel) checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		rel, hasNew, err := m.updaterSvc.CheckUpdate("0.2.0", false)
		return updateCheckMsg{release: rel, hasNew: hasNew, err: err}
	}
}

func (m *AppModel) selfUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		newVer, err := m.updaterSvc.SelfUpdate("0.2.0")
		return selfUpdateResultMsg{newVersion: newVer, err: err}
	}
}

func (m *AppModel) fetchStatsCmd(server *models.Server) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		stats, err := m.executor.FetchStats(ctx, server)
		return statsResultMsg{
			serverName: server.Name,
			stats:      stats,
			err:        err,
		}
	}
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeTables(msg.Width, msg.Height)
		m.updateServerRows()
		m.updateAliasRows()
		m.updateTunnelRows()
		m.updateSnippetRows()
		m.updateEndpointRows()

	case pingResultMsg:
		if msg.res != nil {
			for _, s := range m.servers {
				if s.Name == msg.res.ServerName {
					s.Status = msg.res.Status
					s.Latency = msg.res.Latency
					break
				}
			}
			m.statusToast = ""
			m.updateServerRows()
		}

	case endpointResultMsg:
		if msg.ep != nil {
			for i, e := range m.endpoints {
				if strings.EqualFold(e.Name, msg.ep.Name) {
					m.endpoints[i] = msg.ep
					break
				}
			}
			m.updateEndpointRows()
		}

	case updateCheckMsg:
		if msg.hasNew && msg.release != nil {
			m.updateAvailable = true
			m.latestRelease = msg.release
		}

	case selfUpdateResultMsg:
		if msg.err != nil {
			m.statusToast = "Update error: " + msg.err.Error()
		} else {
			m.statusToast = "✓ Successfully updated to " + msg.newVersion + "! Please restart deck."
		}

	case statsResultMsg:
		m.inspecting = false
		if msg.err != nil {
			m.modalMessage = msg.err.Error()
		} else {
			m.inspectStats = msg.stats
		}

	case tea.KeyMsg:
		if m.currentModal != modalNone {
			return m.handleModalKey(msg)
		}

		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			case "enter":
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.applyFilter(m.searchInput.Value())
				return m, cmd
			}
		}

		return m.handleNormalKey(msg)
	}

	return m, cmd
}

func (m *AppModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "q", "ctrl+c":
		m.Close()
		return m, tea.Quit

	case "1":
		m.switchTab(tabServers)
		return m, nil
	case "2":
		m.switchTab(tabAliases)
		return m, nil
	case "3":
		m.switchTab(tabTunnels)
		return m, nil
	case "4":
		m.switchTab(tabSnippets)
		return m, nil
	case "5":
		m.switchTab(tabEndpoints)
		return m, nil

	case "tab":
		next := (m.currentTab + 1) % 5
		m.switchTab(next)
		return m, nil

	case "shift+tab":
		prev := (m.currentTab - 1 + 5) % 5
		m.switchTab(prev)
		return m, nil

	case "/":
		m.searching = true
		m.searchInput.Focus()
		return m, textinput.Blink

	case "r", "p":
		if m.currentTab == tabEndpoints {
			m.statusToast = "📡 Probing endpoints in background..."
			for _, ep := range m.endpoints {
				ep.Status = models.EndpointStatusChecking
			}
			m.updateEndpointRows()
			return m, m.probeAllEndpointsCmd()
		}
		m.statusToast = "📡 Probing hosts in background..."
		for _, s := range m.servers {
			s.Status = models.StatusChecking
			s.Latency = 0
		}
		m.updateServerRows()
		return m, m.pingAllServersCmd()

	case "U":
		if m.updateAvailable && m.latestRelease != nil {
			m.statusToast = "⬇ Downloading and installing " + m.latestRelease.TagName + "..."
			return m, m.selfUpdateCmd()
		}

	case "c":
		if m.currentTab == tabSnippets && len(m.filteredSnippets) > 0 {
			idx := m.snippetTable.Cursor()
			if idx >= 0 && idx < len(m.filteredSnippets) {
				snip := m.filteredSnippets[idx]
				if err := m.snippetRunner.CopyToClipboard(snip); err == nil {
					m.statusToast = "✓ Copied to clipboard: " + snip.Command
				} else {
					m.statusToast = "Error copying: " + err.Error()
				}
				return m, nil
			}
		}

	case " ":
		if m.currentTab == tabTunnels && len(m.filteredTunnels) > 0 {
			m.toggleSelectedTunnel()
			return m, nil
		}

	case "a":
		m.openAddModal()
		return m, textinput.Blink

	case "e":
		m.openEditModal()
		return m, textinput.Blink

	case "x", "delete":
		m.deleteSelectedItem()
		return m, nil

	case "s":
		if m.currentTab == tabServers && len(m.filteredServers) > 0 {
			idx := m.serverTable.Cursor()
			if idx >= 0 && idx < len(m.filteredServers) {
				srv := m.filteredServers[idx]
				m.currentModal = modalInspect
				m.inspecting = true
				m.inspectStats = nil
				m.modalMessage = fmt.Sprintf("Querying live metrics from %s...", srv.Name)
				return m, m.fetchStatsCmd(srv)
			}
		}

	case "E":
		m.currentModal = modalExport
		m.passwordInput.Reset()
		m.passwordInput.Focus()
		m.exportB64Result = ""
		m.modalMessage = "Press Enter to copy Base64 export payload."
		return m, textinput.Blink

	case "i":
		m.currentModal = modalImport
		m.genericInput = textinput.New()
		m.genericInput.Placeholder = "Paste Base64 export string..."
		m.genericInput.Focus()
		m.modalMessage = "Paste string and press Enter to import."
		return m, textinput.Blink

	case "?":
		m.currentModal = modalHelp
		return m, nil

	case "enter":
		switch m.currentTab {
		case tabServers:
			if len(m.filteredServers) > 0 {
				idx := m.serverTable.Cursor()
				if idx >= 0 && idx < len(m.filteredServers) {
					m.TargetServer = m.filteredServers[idx]
					return m, tea.Quit
				}
			}
		case tabAliases:
			if len(m.filteredAliases) > 0 {
				idx := m.aliasTable.Cursor()
				if idx >= 0 && idx < len(m.filteredAliases) {
					m.TargetDir = m.filteredAliases[idx].ExpandedPath()
					return m, tea.Quit
				}
			}
		case tabTunnels:
			if len(m.filteredTunnels) > 0 {
				m.toggleSelectedTunnel()
				return m, nil
			}
		case tabSnippets:
			if len(m.filteredSnippets) > 0 {
				idx := m.snippetTable.Cursor()
				if idx >= 0 && idx < len(m.filteredSnippets) {
					m.TargetSnippet = m.filteredSnippets[idx]
					return m, tea.Quit
				}
			}
		case tabEndpoints:
			if len(m.filteredEndpoints) > 0 {
				idx := m.endpointTable.Cursor()
				if idx >= 0 && idx < len(m.filteredEndpoints) {
					ep := m.filteredEndpoints[idx]
					openBrowser(ep.URL)
					m.statusToast = "Opening " + ep.URL
					return m, nil
				}
			}
		}
	}

	switch m.currentTab {
	case tabServers:
		m.serverTable, cmd = m.serverTable.Update(msg)
		m.updateServerRows()
	case tabAliases:
		m.aliasTable, cmd = m.aliasTable.Update(msg)
		m.updateAliasRows()
	case tabTunnels:
		m.tunnelTable, cmd = m.tunnelTable.Update(msg)
		m.updateTunnelRows()
	case tabSnippets:
		m.snippetTable, cmd = m.snippetTable.Update(msg)
		m.updateSnippetRows()
	case tabEndpoints:
		m.endpointTable, cmd = m.endpointTable.Update(msg)
		m.updateEndpointRows()
	}

	return m, cmd
}

func (m *AppModel) toggleSelectedTunnel() {
	idx := m.tunnelTable.Cursor()
	if idx < 0 || idx >= len(m.filteredTunnels) {
		return
	}
	t := m.filteredTunnels[idx]
	if t.Active {
		_ = m.tunnelMgr.Stop(t)
		m.statusToast = fmt.Sprintf("Tunnel %q stopped.", t.Name)
	} else {
		if err := m.tunnelMgr.Start(t); err != nil {
			m.statusToast = "Error starting tunnel: " + err.Error()
		} else {
			m.statusToast = fmt.Sprintf("Tunnel %q started (PID %d).", t.Name, t.PID)
		}
	}
	_ = m.cfgManager.AddOrUpdateTunnel(t)
	m.updateTunnelRows()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func (m *AppModel) openAddModal() {
	m.currentModal = modalForm
	switch m.currentTab {
	case tabServers:
		m.form = modalFormState{
			title:   "Add New SSH Connection",
			isEdit:  false,
			fields: []formField{
				{label: "Name", input: newFieldInput("e.g. prod-web")},
				{label: "Host / IP", input: newFieldInput("e.g. 192.168.1.50 or api.domain.com")},
				{label: "Port", input: newFieldInput("22")},
				{label: "User", input: newFieldInput("e.g. ubuntu, root, deploy")},
				{label: "Environment", input: newFieldInput("e.g. production, staging, dev")},
				{label: "Tags", input: newFieldInput("e.g. web, api, docker")},
			},
			focused: 0,
		}
	case tabAliases:
		m.form = modalFormState{
			title:   "Add Directory Alias",
			isEdit:  false,
			fields: []formField{
				{label: "Alias Name", input: newFieldInput("e.g. backend")},
				{label: "Directory Path", input: newFieldInput("e.g. ~/projects/backend")},
				{label: "Description", input: newFieldInput("e.g. Core API service")},
			},
			focused: 0,
		}
	case tabTunnels:
		m.form = modalFormState{
			title:   "Add SSH Tunnel",
			isEdit:  false,
			fields: []formField{
				{label: "Tunnel Name", input: newFieldInput("e.g. pg-tunnel")},
				{label: "SSH Host", input: newFieldInput("e.g. prod-web")},
				{label: "Local Port", input: newFieldInput("5432")},
				{label: "Remote Host", input: newFieldInput("127.0.0.1")},
				{label: "Remote Port", input: newFieldInput("5432")},
				{label: "Type", input: newFieldInput("local (or remote, dynamic)")},
			},
			focused: 0,
		}
	case tabSnippets:
		m.form = modalFormState{
			title:   "Add Command Snippet",
			isEdit:  false,
			fields: []formField{
				{label: "Name", input: newFieldInput("e.g. flush-cache")},
				{label: "Command", input: newFieldInput("e.g. redis-cli flushall")},
				{label: "Target Host", input: newFieldInput("Optional SSH host, or empty for local")},
				{label: "Description", input: newFieldInput("e.g. Clear Redis cache")},
				{label: "Tags", input: newFieldInput("e.g. redis, db, cache")},
			},
			focused: 0,
		}
	case tabEndpoints:
		m.form = modalFormState{
			title:   "Add Web Endpoint",
			isEdit:  false,
			fields: []formField{
				{label: "Endpoint Name", input: newFieldInput("e.g. prod-api")},
				{label: "URL", input: newFieldInput("https://api.example.com/health")},
			},
			focused: 0,
		}
	}
	m.form.fields[0].input.Focus()
}

func (m *AppModel) openEditModal() {
	switch m.currentTab {
	case tabServers:
		idx := m.serverTable.Cursor()
		if idx < 0 || idx >= len(m.filteredServers) {
			return
		}
		srv := m.filteredServers[idx]
		m.currentModal = modalForm
		m.form = modalFormState{
			title:    "Edit Connection: " + srv.Name,
			isEdit:   true,
			origName: srv.Name,
			fields: []formField{
				{label: "Name", input: newFieldInputWithPlaceholder(srv.Name, "e.g. prod-web")},
				{label: "Host / IP", input: newFieldInputWithPlaceholder(srv.Hostname, "e.g. 192.168.1.50")},
				{label: "Port", input: newFieldInputWithPlaceholder(fmt.Sprintf("%d", srv.EffectivePort()), "22")},
				{label: "User", input: newFieldInputWithPlaceholder(srv.User, "e.g. ubuntu, root")},
				{label: "Environment", input: newFieldInputWithPlaceholder(srv.Environment, "e.g. production, staging, dev")},
				{label: "Tags", input: newFieldInputWithPlaceholder(strings.Join(srv.Tags, ", "), "e.g. web, api, database")},
			},
			focused: 0,
		}
	case tabAliases:
		idx := m.aliasTable.Cursor()
		if idx < 0 || idx >= len(m.filteredAliases) {
			return
		}
		a := m.filteredAliases[idx]
		m.currentModal = modalForm
		m.form = modalFormState{
			title:    "Edit Directory Alias: " + a.Name,
			isEdit:   true,
			origName: a.Name,
			fields: []formField{
				{label: "Alias Name", input: newFieldInputWithPlaceholder(a.Name, "e.g. backend")},
				{label: "Directory Path", input: newFieldInputWithPlaceholder(a.Path, "e.g. ~/projects/backend")},
				{label: "Description", input: newFieldInputWithPlaceholder(a.Description, "e.g. Main project repo")},
			},
			focused: 0,
		}
	case tabTunnels:
		idx := m.tunnelTable.Cursor()
		if idx < 0 || idx >= len(m.filteredTunnels) {
			return
		}
		t := m.filteredTunnels[idx]
		m.currentModal = modalForm
		m.form = modalFormState{
			title:    "Edit SSH Tunnel: " + t.Name,
			isEdit:   true,
			origName: t.Name,
			fields: []formField{
				{label: "Tunnel Name", input: newFieldInputWithPlaceholder(t.Name, "e.g. pg-tunnel")},
				{label: "SSH Host", input: newFieldInputWithPlaceholder(t.ServerName, "e.g. prod-web")},
				{label: "Local Port", input: newFieldInputWithPlaceholder(fmt.Sprintf("%d", t.LocalPort), "5432")},
				{label: "Remote Host", input: newFieldInputWithPlaceholder(t.RemoteHost, "127.0.0.1")},
				{label: "Remote Port", input: newFieldInputWithPlaceholder(fmt.Sprintf("%d", t.RemotePort), "5432")},
				{label: "Type", input: newFieldInputWithPlaceholder(t.Type, "local")},
			},
			focused: 0,
		}
	case tabSnippets:
		idx := m.snippetTable.Cursor()
		if idx < 0 || idx >= len(m.filteredSnippets) {
			return
		}
		snip := m.filteredSnippets[idx]
		m.currentModal = modalForm
		m.form = modalFormState{
			title:    "Edit Snippet: " + snip.Name,
			isEdit:   true,
			origName: snip.Name,
			fields: []formField{
				{label: "Name", input: newFieldInputWithPlaceholder(snip.Name, "e.g. flush-cache")},
				{label: "Command", input: newFieldInputWithPlaceholder(snip.Command, "e.g. redis-cli flushall")},
				{label: "Target Host", input: newFieldInputWithPlaceholder(snip.TargetHost, "Local or SSH host")},
				{label: "Description", input: newFieldInputWithPlaceholder(snip.Description, "e.g. Clear cache")},
				{label: "Tags", input: newFieldInputWithPlaceholder(strings.Join(snip.Tags, ", "), "e.g. redis, db")},
			},
			focused: 0,
		}
	case tabEndpoints:
		idx := m.endpointTable.Cursor()
		if idx < 0 || idx >= len(m.filteredEndpoints) {
			return
		}
		ep := m.filteredEndpoints[idx]
		m.currentModal = modalForm
		m.form = modalFormState{
			title:    "Edit Endpoint: " + ep.Name,
			isEdit:   true,
			origName: ep.Name,
			fields: []formField{
				{label: "Endpoint Name", input: newFieldInputWithPlaceholder(ep.Name, "e.g. prod-api")},
				{label: "URL", input: newFieldInputWithPlaceholder(ep.URL, "https://...")},
			},
			focused: 0,
		}
	}
	m.form.fields[0].input.Focus()
}

func newFieldInputWithPlaceholder(val, placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = ""
	ti.Width = 40
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.TextNormal)
	ti.SetValue(val)
	return ti
}

func newFieldInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = ""
	ti.Width = 40
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.TextNormal)
	return ti
}

func (m *AppModel) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentModal = modalNone
		return m, nil

	case "tab", "down":
		if m.currentModal == modalForm {
			m.form.fields[m.form.focused].input.Blur()
			m.form.focused = (m.form.focused + 1) % len(m.form.fields)
			m.form.fields[m.form.focused].input.Focus()
			return m, textinput.Blink
		}

	case "shift+tab", "up":
		if m.currentModal == modalForm {
			m.form.fields[m.form.focused].input.Blur()
			m.form.focused = (m.form.focused - 1 + len(m.form.fields)) % len(m.form.fields)
			m.form.fields[m.form.focused].input.Focus()
			return m, textinput.Blink
		}

	case "enter":
		if m.currentModal == modalForm {
			m.saveForm()
			m.currentModal = modalNone
			if m.currentTab == tabServers {
				return m, m.pingAllServersCmd()
			} else if m.currentTab == tabEndpoints {
				return m, m.probeAllEndpointsCmd()
			}
			return m, nil
		}

		if m.currentModal == modalExport {
			pwd := m.passwordInput.Value()
			deckCfg, _ := m.cfgManager.LoadDeckConfig()
			data := &models.ExportData{
				Servers:   make([]models.Server, 0, len(m.servers)),
				Aliases:   make([]models.DirectoryAlias, 0, len(m.aliases)),
				Tunnels:   make([]models.Tunnel, 0, len(deckCfg.Tunnels)),
				Snippets:  make([]models.Snippet, 0, len(deckCfg.Snippets)),
				Endpoints: make([]models.Endpoint, 0, len(deckCfg.Endpoints)),
			}
			for _, s := range m.servers {
				data.Servers = append(data.Servers, *s)
			}
			for _, a := range m.aliases {
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

			b64, err := m.transferSvc.ExportBase64(data, pwd)
			if err != nil {
				m.modalMessage = fmt.Sprintf("Export error: %v", err)
			} else {
				m.exportB64Result = b64
				_ = clipboard.WriteAll(b64)
				m.modalMessage = "✓ Copied to clipboard successfully!"
			}
			return m, nil
		}

		if m.currentModal == modalImport {
			raw := m.genericInput.Value()
			pwd := m.passwordInput.Value()
			data, err := m.transferSvc.ImportBase64(raw, pwd)
			if err != nil {
				m.modalMessage = fmt.Sprintf("Import failed: %v", err)
			} else {
				deckCfg, _ := m.cfgManager.LoadDeckConfig()
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

				_ = m.cfgManager.SaveDeckConfig(deckCfg)
				m.servers, _ = m.cfgManager.LoadAllServers()
				m.aliases = deckCfg.Aliases
				m.tunnels = deckCfg.Tunnels
				m.snippets = deckCfg.Snippets
				m.endpoints = deckCfg.Endpoints
				m.applyFilter("")
				m.modalMessage = fmt.Sprintf("✓ Imported %d servers, %d aliases, %d tunnels, %d snippets.",
					len(data.Servers), len(data.Aliases), len(data.Tunnels), len(data.Snippets))
			}
			return m, nil
		}
	}

	if m.currentModal == modalForm {
		var cmd tea.Cmd
		m.form.fields[m.form.focused].input, cmd = m.form.fields[m.form.focused].input.Update(msg)
		return m, cmd
	}

	if m.currentModal == modalExport {
		var cmd tea.Cmd
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		return m, cmd
	}

	if m.currentModal == modalImport {
		var cmd tea.Cmd
		m.genericInput, cmd = m.genericInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *AppModel) saveForm() {
	switch m.currentTab {
	case tabServers:
		name := strings.TrimSpace(m.form.fields[0].input.Value())
		host := strings.TrimSpace(m.form.fields[1].input.Value())
		portStr := strings.TrimSpace(m.form.fields[2].input.Value())
		user := strings.TrimSpace(m.form.fields[3].input.Value())
		env := strings.TrimSpace(m.form.fields[4].input.Value())
		tagsStr := strings.TrimSpace(m.form.fields[5].input.Value())

		if name == "" {
			return
		}
		if host == "" {
			host = name
		}
		port, _ := strconv.Atoi(portStr)
		if port <= 0 {
			port = 22
		}

		var tags []string
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				if tr := strings.TrimSpace(t); tr != "" {
					tags = append(tags, tr)
				}
			}
		}

		srv := &models.Server{
			Name:        name,
			Hostname:    host,
			Port:        port,
			User:        user,
			Environment: env,
			Tags:        tags,
			Status:      models.StatusChecking,
		}

		_ = m.cfgManager.AddOrUpdateServer(srv)
		m.servers, _ = m.cfgManager.LoadAllServers()
		m.applyFilter(m.searchInput.Value())

	case tabAliases:
		name := strings.TrimSpace(m.form.fields[0].input.Value())
		path := strings.TrimSpace(m.form.fields[1].input.Value())
		desc := strings.TrimSpace(m.form.fields[2].input.Value())

		if name == "" || path == "" {
			return
		}

		alias := &models.DirectoryAlias{
			Name:        name,
			Path:        path,
			Description: desc,
		}

		_ = m.cfgManager.AddOrUpdateAlias(alias)
		deckCfg, _ := m.cfgManager.LoadDeckConfig()
		m.aliases = deckCfg.Aliases
		m.applyFilter(m.searchInput.Value())

	case tabTunnels:
		name := strings.TrimSpace(m.form.fields[0].input.Value())
		srvHost := strings.TrimSpace(m.form.fields[1].input.Value())
		lPortStr := strings.TrimSpace(m.form.fields[2].input.Value())
		rHost := strings.TrimSpace(m.form.fields[3].input.Value())
		rPortStr := strings.TrimSpace(m.form.fields[4].input.Value())
		tType := strings.TrimSpace(m.form.fields[5].input.Value())

		if name == "" || srvHost == "" {
			return
		}
		lPort, _ := strconv.Atoi(lPortStr)
		rPort, _ := strconv.Atoi(rPortStr)
		if lPort <= 0 {
			lPort = 8080
		}
		if rPort <= 0 {
			rPort = lPort
		}
		if rHost == "" {
			rHost = "127.0.0.1"
		}
		if tType == "" {
			tType = "local"
		}

		tun := &models.Tunnel{
			Name:       name,
			ServerName: srvHost,
			LocalPort:  lPort,
			RemoteHost: rHost,
			RemotePort: rPort,
			Type:       tType,
		}

		_ = m.cfgManager.AddOrUpdateTunnel(tun)
		deckCfg, _ := m.cfgManager.LoadDeckConfig()
		m.tunnels = deckCfg.Tunnels
		m.applyFilter(m.searchInput.Value())

	case tabSnippets:
		name := strings.TrimSpace(m.form.fields[0].input.Value())
		cmdStr := strings.TrimSpace(m.form.fields[1].input.Value())
		target := strings.TrimSpace(m.form.fields[2].input.Value())
		desc := strings.TrimSpace(m.form.fields[3].input.Value())
		tagsStr := strings.TrimSpace(m.form.fields[4].input.Value())

		if name == "" || cmdStr == "" {
			return
		}

		var tags []string
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				if tr := strings.TrimSpace(t); tr != "" {
					tags = append(tags, tr)
				}
			}
		}

		snip := &models.Snippet{
			Name:        name,
			Command:     cmdStr,
			TargetHost:  target,
			Description: desc,
			Tags:        tags,
		}

		_ = m.cfgManager.AddOrUpdateSnippet(snip)
		deckCfg, _ := m.cfgManager.LoadDeckConfig()
		m.snippets = deckCfg.Snippets
		m.applyFilter(m.searchInput.Value())

	case tabEndpoints:
		name := strings.TrimSpace(m.form.fields[0].input.Value())
		rawURL := strings.TrimSpace(m.form.fields[1].input.Value())

		if name == "" || rawURL == "" {
			return
		}
		if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
			rawURL = "https://" + rawURL
		}

		ep := &models.Endpoint{
			Name:   name,
			URL:    rawURL,
			Status: models.EndpointStatusChecking,
		}

		_ = m.cfgManager.AddOrUpdateEndpoint(ep)
		deckCfg, _ := m.cfgManager.LoadDeckConfig()
		m.endpoints = deckCfg.Endpoints
		m.applyFilter(m.searchInput.Value())
	}
}

func (m *AppModel) deleteSelectedItem() {
	switch m.currentTab {
	case tabServers:
		idx := m.serverTable.Cursor()
		if idx >= 0 && idx < len(m.filteredServers) {
			name := m.filteredServers[idx].Name
			deckCfg, _ := m.cfgManager.LoadDeckConfig()
			var remaining []*models.Server
			for _, s := range deckCfg.Servers {
				if !strings.EqualFold(s.Name, name) {
					remaining = append(remaining, s)
				}
			}
			deckCfg.Servers = remaining
			_ = m.cfgManager.SaveDeckConfig(deckCfg)
			m.servers, _ = m.cfgManager.LoadAllServers()
			m.applyFilter(m.searchInput.Value())
		}
	case tabAliases:
		idx := m.aliasTable.Cursor()
		if idx >= 0 && idx < len(m.filteredAliases) {
			_ = m.cfgManager.RemoveAlias(m.filteredAliases[idx].Name)
			deckCfg, _ := m.cfgManager.LoadDeckConfig()
			m.aliases = deckCfg.Aliases
			m.applyFilter(m.searchInput.Value())
		}
	case tabTunnels:
		idx := m.tunnelTable.Cursor()
		if idx >= 0 && idx < len(m.filteredTunnels) {
			t := m.filteredTunnels[idx]
			_ = m.tunnelMgr.Stop(t)
			_ = m.cfgManager.RemoveTunnel(t.Name)
			deckCfg, _ := m.cfgManager.LoadDeckConfig()
			m.tunnels = deckCfg.Tunnels
			m.applyFilter(m.searchInput.Value())
		}
	case tabSnippets:
		idx := m.snippetTable.Cursor()
		if idx >= 0 && idx < len(m.filteredSnippets) {
			_ = m.cfgManager.RemoveSnippet(m.filteredSnippets[idx].Name)
			deckCfg, _ := m.cfgManager.LoadDeckConfig()
			m.snippets = deckCfg.Snippets
			m.applyFilter(m.searchInput.Value())
		}
	case tabEndpoints:
		idx := m.endpointTable.Cursor()
		if idx >= 0 && idx < len(m.filteredEndpoints) {
			_ = m.cfgManager.RemoveEndpoint(m.filteredEndpoints[idx].Name)
			deckCfg, _ := m.cfgManager.LoadDeckConfig()
			m.endpoints = deckCfg.Endpoints
			m.applyFilter(m.searchInput.Value())
		}
	}
}

func (m *AppModel) applyFilter(q string) {
	q = strings.ToLower(strings.TrimSpace(q))

	var matchedServers []*models.Server
	for _, s := range m.servers {
		if s.MatchesQuery(q) {
			matchedServers = append(matchedServers, s)
		}
	}
	m.filteredServers = matchedServers
	m.updateServerRows()

	var matchedAliases []*models.DirectoryAlias
	for _, a := range m.aliases {
		if a.MatchesQuery(q) {
			matchedAliases = append(matchedAliases, a)
		}
	}
	m.filteredAliases = matchedAliases
	m.updateAliasRows()

	var matchedTunnels []*models.Tunnel
	for _, t := range m.tunnels {
		if t.MatchesQuery(q) {
			matchedTunnels = append(matchedTunnels, t)
		}
	}
	m.filteredTunnels = matchedTunnels
	m.updateTunnelRows()

	var matchedSnippets []*models.Snippet
	for _, sn := range m.snippets {
		if sn.MatchesQuery(q) {
			matchedSnippets = append(matchedSnippets, sn)
		}
	}
	m.filteredSnippets = matchedSnippets
	m.updateSnippetRows()

	var matchedEndpoints []*models.Endpoint
	for _, ep := range m.endpoints {
		if ep.MatchesQuery(q) {
			matchedEndpoints = append(matchedEndpoints, ep)
		}
	}
	m.filteredEndpoints = matchedEndpoints
	m.updateEndpointRows()
}

// View renders the complete Deck Cockpit TUI layout.
func (m *AppModel) View() string {
	var b strings.Builder

	// Top Bar & Logo
	logo := styles.LogoStyle.Render(" DECK ")
	version := styles.VersionStyle.Render("v0.2.0")
	leftHeader := lipgloss.JoinHorizontal(lipgloss.Center, logo, version)

	if m.updateAvailable && m.latestRelease != nil {
		badge := styles.UpdateBadge.Render("⬆ UPDATE " + m.latestRelease.TagName + " [U]")
		leftHeader = lipgloss.JoinHorizontal(lipgloss.Center, leftHeader, " ", badge)
	}

	// 5 Cockpit Tabs
	tabLabels := []string{
		fmt.Sprintf("[1] SSH Servers (%d)", len(m.filteredServers)),
		fmt.Sprintf("[2] Aliases (%d)", len(m.filteredAliases)),
		fmt.Sprintf("[3] Tunnels (%d)", len(m.filteredTunnels)),
		fmt.Sprintf("[4] Snippets (%d)", len(m.filteredSnippets)),
		fmt.Sprintf("[5] Endpoints (%d)", len(m.filteredEndpoints)),
	}

	var renderedTabs []string
	for i, lbl := range tabLabels {
		if activeTab(i) == m.currentTab {
			renderedTabs = append(renderedTabs, styles.TabActive.Render(lbl))
		} else {
			renderedTabs = append(renderedTabs, styles.TabInactive.Render(lbl))
		}
		if i < len(tabLabels)-1 {
			renderedTabs = append(renderedTabs, styles.TabBorder.Render("│"))
		}
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Center, renderedTabs...)
	topBar := lipgloss.JoinHorizontal(lipgloss.Center, leftHeader, "    ", tabs)
	b.WriteString("  " + topBar + "\n\n")

	// Search Box
	b.WriteString("  " + styles.SearchBox.Render(m.searchInput.View()) + "\n\n")

	// Main Tab Table (ANSI-aware custom table renderer)
	switch m.currentTab {
	case tabServers:
		b.WriteString("  " + m.renderCustomTable(m.serverTable) + "\n")
	case tabAliases:
		b.WriteString("  " + m.renderCustomTable(m.aliasTable) + "\n")
	case tabTunnels:
		b.WriteString("  " + m.renderCustomTable(m.tunnelTable) + "\n")
	case tabSnippets:
		b.WriteString("  " + m.renderCustomTable(m.snippetTable) + "\n")
	case tabEndpoints:
		b.WriteString("  " + m.renderCustomTable(m.endpointTable) + "\n")
	}

	// Footer / Status Toast
	footer := m.renderFooter()
	if m.statusToast != "" {
		footer = styles.FooterKey.Copy().Foreground(styles.AccentWarning).Render(m.statusToast) + "   " + footer
	}
	b.WriteString("\n  " + footer)

	// Modal Overlay
	if m.currentModal != modalNone {
		modalContent := m.renderModal()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalContent)
	}

	return b.String()
}

func (m *AppModel) renderFooter() string {
	var items []struct {
		key  string
		desc string
	}

	switch m.currentTab {
	case tabServers:
		items = []struct {
			key  string
			desc string
		}{
			{"enter", "connect"},
			{"a", "add"},
			{"e", "edit"},
			{"s", "stats"},
			{"x", "delete"},
			{"/", "filter"},
			{"tab", "switch"},
			{"r", "ping"},
			{"E", "export"},
			{"i", "import"},
			{"?", "help"},
			{"q", "quit"},
		}
	case tabAliases:
		items = []struct {
			key  string
			desc string
		}{
			{"enter", "jump to path"},
			{"a", "add alias"},
			{"e", "edit alias"},
			{"x", "delete"},
			{"/", "filter"},
			{"tab", "switch"},
			{"q", "quit"},
		}
	case tabTunnels:
		items = []struct {
			key  string
			desc string
		}{
			{"space/enter", "toggle tunnel"},
			{"a", "add tunnel"},
			{"e", "edit tunnel"},
			{"x", "delete"},
			{"/", "filter"},
			{"tab", "switch"},
			{"q", "quit"},
		}
	case tabSnippets:
		items = []struct {
			key  string
			desc string
		}{
			{"enter", "run snippet"},
			{"c", "copy command"},
			{"a", "add snippet"},
			{"e", "edit snippet"},
			{"x", "delete"},
			{"/", "filter"},
			{"tab", "switch"},
			{"q", "quit"},
		}
	case tabEndpoints:
		items = []struct {
			key  string
			desc string
		}{
			{"enter", "open in browser"},
			{"r", "re-probe all"},
			{"a", "add endpoint"},
			{"e", "edit endpoint"},
			{"x", "delete"},
			{"/", "filter"},
			{"tab", "switch"},
			{"q", "quit"},
		}
	}

	var parts []string
	for _, item := range items {
		k := styles.FooterKey.Render("[" + item.key + "]")
		d := styles.FooterDesc.Render(item.desc)
		parts = append(parts, k+" "+d)
	}
	return strings.Join(parts, "  •  ")
}

func (m *AppModel) renderModal() string {
	switch m.currentModal {
	case modalForm:
		var b strings.Builder
		b.WriteString(styles.HeaderStyle.Render(m.form.title) + "\n\n")

		for i, f := range m.form.fields {
			cursor := "  "
			labelStyle := styles.FooterDesc
			if i == m.form.focused {
				cursor = "❯ "
				labelStyle = styles.FooterKey
			}
			b.WriteString(fmt.Sprintf("%s%-16s [ %s ]\n", cursor, labelStyle.Render(f.label+":"), f.input.View()))
		}
		b.WriteString("\n" + styles.FooterDesc.Render("[Tab/↓] Next  •  [Shift+Tab/↑] Prev  •  [Enter] Save  •  [Esc] Cancel"))
		return styles.ModalBox.Render(b.String())

	case modalInspect:
		content := fmt.Sprintf("Remote Host Metrics\n\n%s\n", m.modalMessage)
		if m.inspectStats != nil && m.inspectStats.Error == "" {
			content = fmt.Sprintf("Remote Host Metrics\n\n"+
				"OS:      %s\n"+
				"Uptime:  %s\n"+
				"Load:    %s\n"+
				"Memory:  %s / %s (%d%%)\n"+
				"Disk:    %s / %s (%d%%)\n",
				m.inspectStats.OSInfo,
				m.inspectStats.Uptime,
				m.inspectStats.LoadAvg,
				m.inspectStats.MemoryUsed, m.inspectStats.MemoryTotal, m.inspectStats.MemoryPct,
				m.inspectStats.DiskUsed, m.inspectStats.DiskTotal, m.inspectStats.DiskPct,
			)
		} else {
			content = fmt.Sprintf("Remote Host Metrics\n\n⚠️ %s\n\nTip: Press [e] to edit the host or port.\n", m.modalMessage)
		}
		return styles.ModalBox.Render(content + "\n[Esc] Close")

	case modalExport:
		content := fmt.Sprintf("Export Inventory to Base64\n\n"+
			"Passphrase (optional for AES-256-GCM encryption):\n%s\n\n"+
			"%s\n",
			m.passwordInput.View(),
			m.modalMessage,
		)
		if m.exportB64Result != "" {
			preview := m.exportB64Result
			if len(preview) > 60 {
				preview = preview[:60] + "..."
			}
			content += fmt.Sprintf("\nPayload: %s\n", preview)
		}
		return styles.ModalBox.Render(content + "\n[Enter] Generate & Copy  •  [Esc] Close")

	case modalImport:
		return styles.ModalBox.Render(
			fmt.Sprintf("Import Inventory from Base64\n\n"+
				"%s\n\n%s\n\n[Enter] Import  •  [Esc] Close",
				m.genericInput.View(),
				m.modalMessage,
			),
		)

	case modalHelp:
		return styles.ModalBox.Render(
			"Deck Cockpit Keyboard Shortcuts\n\n" +
				"  1 .. 5     Switch cockpit tabs (Servers, Aliases, Tunnels, Snippets, Endpoints)\n" +
				"  Tab        Cycle through tabs\n" +
				"  Enter      Connect to SSH / Jump to directory / Run snippet / Open URL\n" +
				"  Space      Toggle SSH tunnel start / stop\n" +
				"  c          Copy snippet command to clipboard\n" +
				"  a          Add new entry to the active tab\n" +
				"  e          Edit focused entry\n" +
				"  x          Delete focused entry\n" +
				"  s          Query remote host metrics non-interactively\n" +
				"  /          Filter current tab by query\n" +
				"  r / p      Re-probe hosts or endpoints\n" +
				"  U          Update Deck to latest release from GitHub\n" +
				"  E / i      Export or Import encrypted Base64 vault inventory\n" +
				"  q          Quit Deck\n\n" +
				"[Esc] Close",
		)
	}
	return ""
}

func (m *AppModel) renderCustomTable(t table.Model) string {
	cols := t.Columns()
	rows := t.Rows()
	cursor := t.Cursor()
	height := t.Height()
	if height <= 0 {
		height = 10
	}

	if len(rows) == 0 {
		return "  " + styles.StatusUnknown.Render("(No items configured. Press 'a' to add one.)") + "\n"
	}

	var b strings.Builder

	// Header row
	var headerCells []string
	var totalWidth int
	for _, c := range cols {
		headerCells = append(headerCells, styles.HeaderStyle.Width(c.Width).Render(c.Title))
		totalWidth += c.Width
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerCells...) + "\n")
	b.WriteString(styles.TabBorder.Render(strings.Repeat("─", totalWidth)) + "\n")

	// Viewport windowing around cursor
	start := 0
	if cursor >= height {
		start = cursor - height + 1
	}
	end := start + height
	if end > len(rows) {
		end = len(rows)
	}

	// Render rows
	for i := start; i < end; i++ {
		row := rows[i]
		isSelected := (i == cursor)

		var cells []string
		for j, val := range row {
			w := cols[j].Width
			cellStyle := lipgloss.NewStyle().Width(w)
			if isSelected {
				cellStyle = cellStyle.Background(styles.BgSelected).Bold(true)
			}
			cells = append(cells, cellStyle.Render(val))
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...) + "\n")
	}

	return b.String()
}
