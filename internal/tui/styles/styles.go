package styles

import (
	"fmt"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
	"github.com/charmbracelet/lipgloss"
)

// AvailableThemes lists all themes supported by Deck.
var AvailableThemes = []string{
	"Catppuccin Mocha",
	"Tokyo Night",
	"Nord",
	"Dracula",
	"Cyberpunk",
}

// CurrentTheme stores the currently active theme name.
var CurrentTheme = "Catppuccin Mocha"

var (
	// Palette Colors
	TextNormal lipgloss.Color
	TextMuted  lipgloss.Color
	TextDim    lipgloss.Color
	TextBold   lipgloss.Color

	AccentPrimary   lipgloss.Color
	AccentSecondary lipgloss.Color
	AccentWarning   lipgloss.Color
	AccentDanger    lipgloss.Color
	AccentSuccess   lipgloss.Color
	AccentOrange    lipgloss.Color

	BgHighlight lipgloss.Color
	BgSelected  lipgloss.Color
	BorderMuted lipgloss.Color

	// Lipgloss Styles
	CursorIndicator lipgloss.Style
	LogoStyle       lipgloss.Style
	VersionStyle    lipgloss.Style
	UpdateBadge     lipgloss.Style
	TabActive       lipgloss.Style
	TabInactive     lipgloss.Style
	TabBorder       lipgloss.Style
	SearchBox       lipgloss.Style

	StatusOnline   lipgloss.Style
	StatusOffline  lipgloss.Style
	StatusChecking lipgloss.Style
	StatusUnknown  lipgloss.Style

	LatencyFast   lipgloss.Style
	LatencyMedium lipgloss.Style
	LatencySlow   lipgloss.Style
	LatencyMuted  lipgloss.Style

	TagPill lipgloss.Style
	EnvPill lipgloss.Style

	ModalBox   lipgloss.Style
	FooterKey  lipgloss.Style
	FooterDesc lipgloss.Style

	KeyStyle    lipgloss.Style
	HeaderStyle lipgloss.Style
	BadgeTag    lipgloss.Style
	BadgeEnv    lipgloss.Style
)

func init() {
	ApplyTheme("Catppuccin Mocha")
}

// ApplyTheme switches all active colors and styles to the requested theme.
func ApplyTheme(name string) {
	CurrentTheme = name
	switch name {
	case "Tokyo Night":
		TextNormal = lipgloss.Color("#A9B1D6")
		TextMuted = lipgloss.Color("#565F89")
		TextDim = lipgloss.Color("#414868")
		TextBold = lipgloss.Color("#C0CAF5")

		AccentPrimary = lipgloss.Color("#7AA2F7")   // Tokyo Blue
		AccentSecondary = lipgloss.Color("#7DCFFF") // Cyan
		AccentWarning = lipgloss.Color("#E0AF68")   // Warm Gold
		AccentDanger = lipgloss.Color("#F7768E")    // Tokyo Pink/Red
		AccentSuccess = lipgloss.Color("#9ECE6A")   // Green
		AccentOrange = lipgloss.Color("#FF9E64")    // Orange

		BgHighlight = lipgloss.Color("#24283B")
		BgSelected = lipgloss.Color("#2E3440")
		BorderMuted = lipgloss.Color("#3B4261")

	case "Nord":
		TextNormal = lipgloss.Color("#D8DEE9")
		TextMuted = lipgloss.Color("#4C566A")
		TextDim = lipgloss.Color("#3B4252")
		TextBold = lipgloss.Color("#ECEFF4")

		AccentPrimary = lipgloss.Color("#88C0D0")   // Frost Cyan
		AccentSecondary = lipgloss.Color("#81A1C1") // Glacier Blue
		AccentWarning = lipgloss.Color("#EBCB8B")   // Nord Yellow
		AccentDanger = lipgloss.Color("#BF616A")    // Nord Red
		AccentSuccess = lipgloss.Color("#A3BE8C")   // Nord Green
		AccentOrange = lipgloss.Color("#D08770")    // Orange

		BgHighlight = lipgloss.Color("#2E3440")
		BgSelected = lipgloss.Color("#434C5E")
		BorderMuted = lipgloss.Color("#3B4252")

	case "Dracula":
		TextNormal = lipgloss.Color("#F8F8F2")
		TextMuted = lipgloss.Color("#6272A4")
		TextDim = lipgloss.Color("#44475A")
		TextBold = lipgloss.Color("#FFFFFF")

		AccentPrimary = lipgloss.Color("#BD93F9")   // Dracula Purple
		AccentSecondary = lipgloss.Color("#8BE9FD") // Cyan
		AccentWarning = lipgloss.Color("#F1FA8C")   // Yellow
		AccentDanger = lipgloss.Color("#FF5555")    // Red
		AccentSuccess = lipgloss.Color("#50FA7B")   // Green
		AccentOrange = lipgloss.Color("#FFB86C")    // Orange

		BgHighlight = lipgloss.Color("#282A36")
		BgSelected = lipgloss.Color("#44475A")
		BorderMuted = lipgloss.Color("#6272A4")

	case "Cyberpunk":
		TextNormal = lipgloss.Color("#E2E8F0")
		TextMuted = lipgloss.Color("#718096")
		TextDim = lipgloss.Color("#2D3748")
		TextBold = lipgloss.Color("#FFFFFF")

		AccentPrimary = lipgloss.Color("#00F0FF")   // Neon Cyan
		AccentSecondary = lipgloss.Color("#FF007F") // Neon Pink
		AccentWarning = lipgloss.Color("#FFE600")   // Electric Yellow
		AccentDanger = lipgloss.Color("#FF3366")    // Hot Red
		AccentSuccess = lipgloss.Color("#00FF66")   // Neon Green
		AccentOrange = lipgloss.Color("#FF7700")    // Neon Orange

		BgHighlight = lipgloss.Color("#1A1B26")
		BgSelected = lipgloss.Color("#2D234A")
		BorderMuted = lipgloss.Color("#3B2D54")

	default: // Catppuccin Mocha
		CurrentTheme = "Catppuccin Mocha"
		TextNormal = lipgloss.Color("#CDD6F4")
		TextMuted = lipgloss.Color("#6C7086")
		TextDim = lipgloss.Color("#45475A")
		TextBold = lipgloss.Color("#FFFFFF")

		AccentPrimary = lipgloss.Color("#89B4FA")   // Clean Blue
		AccentSecondary = lipgloss.Color("#94E2D5") // Soft Teal
		AccentWarning = lipgloss.Color("#F9E2AF")   // Warm Gold
		AccentDanger = lipgloss.Color("#F38BA8")    // Rose Red
		AccentSuccess = lipgloss.Color("#A6E3A1")   // Vibrant Green
		AccentOrange = lipgloss.Color("#FAB387")    // Peach

		BgHighlight = lipgloss.Color("#2A2B3D")
		BgSelected = lipgloss.Color("#313244")
		BorderMuted = lipgloss.Color("#313244")
	}

	// Rebuild Styles
	CursorIndicator = lipgloss.NewStyle().Bold(true).Foreground(AccentPrimary)

	LogoStyle = lipgloss.NewStyle().
		Bold(true).
		Background(AccentPrimary).
		Foreground(lipgloss.Color("#11111B")).
		Padding(0, 1)

	VersionStyle = lipgloss.NewStyle().
		Foreground(TextMuted).
		PaddingLeft(1)

	UpdateBadge = lipgloss.NewStyle().
		Bold(true).
		Background(AccentWarning).
		Foreground(lipgloss.Color("#11111B")).
		Padding(0, 1)

	TabActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(TextBold).
		Background(BgHighlight).
		Padding(0, 1)

	TabInactive = lipgloss.NewStyle().
		Foreground(TextMuted).
		Padding(0, 1)

	TabBorder = lipgloss.NewStyle().
		Foreground(BorderMuted)

	SearchBox = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(AccentPrimary).
		Padding(0, 0, 0, 0)

	StatusOnline = lipgloss.NewStyle().
		Foreground(AccentSuccess).
		Bold(true)

	StatusOffline = lipgloss.NewStyle().
		Foreground(AccentDanger).
		Bold(true)

	StatusChecking = lipgloss.NewStyle().
		Foreground(AccentWarning)

	StatusUnknown = lipgloss.NewStyle().
		Foreground(TextMuted)

	LatencyFast = lipgloss.NewStyle().
		Foreground(AccentSuccess).
		Bold(true)

	LatencyMedium = lipgloss.NewStyle().
		Foreground(AccentWarning)

	LatencySlow = lipgloss.NewStyle().
		Foreground(AccentDanger)

	LatencyMuted = lipgloss.NewStyle().
		Foreground(TextMuted)

	TagPill = lipgloss.NewStyle().
		Foreground(AccentSecondary)

	EnvPill = lipgloss.NewStyle().
		Foreground(AccentWarning)

	ModalBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(AccentPrimary).
		Padding(1, 2)

	FooterKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(AccentPrimary)

	FooterDesc = lipgloss.NewStyle().
		Foreground(TextMuted)

	KeyStyle = lipgloss.NewStyle().Bold(true).Foreground(AccentPrimary)
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(TextBold)
	BadgeTag = lipgloss.NewStyle().Bold(true).Foreground(AccentSecondary)
	BadgeEnv = lipgloss.NewStyle().Bold(true).Foreground(AccentWarning)
}

// FormatServerStatus formats server connectivity status with vibrant colored dot indicators.
func FormatServerStatus(status models.ServerStatus, prefix string) string {
	switch status {
	case models.StatusOnline:
		return prefix + StatusOnline.Render("● online")
	case models.StatusOffline:
		return prefix + StatusOffline.Render("○ offline")
	case models.StatusChecking:
		return prefix + StatusChecking.Render("◌ pinging...")
	default:
		return prefix + StatusUnknown.Render("○ unknown")
	}
}

// FormatLatency formats ping round-trip time with traffic light colors.
func FormatLatency(d time.Duration) string {
	if d <= 0 {
		return LatencyMuted.Render("-")
	}
	ms := d.Milliseconds()
	str := fmt.Sprintf("%dms", ms)
	if ms < 60 {
		return LatencyFast.Render(str)
	} else if ms < 150 {
		return LatencyMedium.Render(str)
	}
	return LatencySlow.Render(str)
}

// FormatTunnelStatus formats SSH tunnel state.
func FormatTunnelStatus(active bool, pid int) string {
	if active {
		if pid > 0 {
			return StatusOnline.Render(fmt.Sprintf("● UP (%d)", pid))
		}
		return StatusOnline.Render("● UP")
	}
	return StatusOffline.Render("○ DOWN")
}

// FormatEndpointStatus formats HTTP status code and availability state.
func FormatEndpointStatus(status models.EndpointStatus, code int) string {
	codeStr := ""
	if code > 0 {
		codeStr = fmt.Sprintf(" [%d]", code)
	}
	switch status {
	case models.EndpointStatusUp:
		return StatusOnline.Render("● UP" + codeStr)
	case models.EndpointStatusDegraded:
		return LatencyMedium.Render("▲ WARN" + codeStr)
	case models.EndpointStatusDown:
		return StatusOffline.Render("○ DOWN" + codeStr)
	case models.EndpointStatusChecking:
		return StatusChecking.Render("◌ probing...")
	default:
		return StatusUnknown.Render("○ UNKNOWN")
	}
}

// FormatSSLExpiry formats SSL certificate remaining days.
func FormatSSLExpiry(days int) string {
	if days <= 0 {
		return LatencyMuted.Render("n/a")
	}
	str := fmt.Sprintf("%dd", days)
	if days > 30 {
		return StatusOnline.Render(str + " left")
	} else if days > 7 {
		return LatencyMedium.Render(str + " left ⚠️")
	}
	return StatusOffline.Render(str + " left 🚨")
}
