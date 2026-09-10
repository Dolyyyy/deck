package styles

import (
	"fmt"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Professional Terminal Color Palette (Catppuccin Mocha / Tokyo Night inspiration)
	// Respects terminal transparency and avoids garish AI neon clichés.
	TextNormal = lipgloss.Color("#CDD6F4")
	TextMuted  = lipgloss.Color("#6C7086")
	TextDim    = lipgloss.Color("#45475A")
	TextBold   = lipgloss.Color("#FFFFFF")

	AccentPrimary   = lipgloss.Color("#89B4FA") // Clean Blue / Indigo
	AccentSecondary = lipgloss.Color("#94E2D5") // Soft Mint / Cyan
	AccentWarning   = lipgloss.Color("#F9E2AF") // Warm Gold
	AccentDanger    = lipgloss.Color("#F38BA8") // Soft Rose / Coral Red
	AccentSuccess   = lipgloss.Color("#A6E3A1") // Vibrant Green
	AccentOrange    = lipgloss.Color("#FAB387") // Peach / Orange

	BgHighlight = lipgloss.Color("#2A2B3D") // Distinct row background
	BgSelected  = lipgloss.Color("#313244") // High-contrast active row selection
	BorderMuted = lipgloss.Color("#313244") // Clean subtle border

	// Cursor & Selection indicator
	CursorIndicator = lipgloss.NewStyle().
			Bold(true).
			Foreground(AccentPrimary)

	// Top Bar & Branding
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

	// Clean Tabs (Consistent height & alignment)
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

	// Search Input
	SearchBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(AccentPrimary).
			Padding(0, 0, 0, 0)

	// Status Badges (Minimal, pro unix dot indicators)
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

	// Latency Badges
	LatencyFast = lipgloss.NewStyle().
			Foreground(AccentSuccess).
			Bold(true)

	LatencyMedium = lipgloss.NewStyle().
			Foreground(AccentWarning)

	LatencySlow = lipgloss.NewStyle().
			Foreground(AccentDanger)

	LatencyMuted = lipgloss.NewStyle().
			Foreground(TextMuted)

	// Tags & Environments
	TagPill = lipgloss.NewStyle().
		Foreground(AccentSecondary)

	EnvPill = lipgloss.NewStyle().
		Foreground(AccentWarning)

	// Modal Overlays
	ModalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentPrimary).
			Padding(1, 2)

	// Footer Keybindings Bar (Compact, pro unix layout)
	FooterKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(AccentPrimary)

	FooterDesc = lipgloss.NewStyle().
			Foreground(TextMuted)

	// CLI Helpers
	KeyStyle    = lipgloss.NewStyle().Bold(true).Foreground(AccentPrimary)
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(TextBold)
	BadgeTag    = lipgloss.NewStyle().Bold(true).Foreground(AccentSecondary)
	BadgeEnv    = lipgloss.NewStyle().Bold(true).Foreground(AccentWarning)
)

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

// FormatLatency formats ping round-trip time with traffic light colors (Green < 60ms, Yellow < 150ms, Red >= 150ms).
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

// FormatTunnelStatus formats SSH tunnel state (Green for UP, Red for DOWN).
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

// FormatSSLExpiry formats SSL certificate remaining days with warning colors.
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
