// Package components provides reusable TUI components.
package components

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/tui/common"
)

var mountainArt = []struct {
	line  string
	color color.Color
}{
	{"          /\\          ", lipgloss.Color("#FFFFFF")},
	{"         /  \\  /\\     ", lipgloss.Color("#C8D4FF")},
	{"    /\\  /    \\/  \\    ", lipgloss.Color("#9BACD4")},
	{"   /  \\/            \\ ", lipgloss.Color("#6B7DB3")},
	{"  /   /              \\", lipgloss.Color("#3D4B6E")},
	{" /___/______________\\ ", lipgloss.Color("#24314A")},
}

var bannerDivStyle = lipgloss.NewStyle().Foreground(common.ColorBorder)

// RenderMountain renders the ASCII mountain with its gradient colors.
func RenderMountain() string {
	var lines []string
	for _, row := range mountainArt {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(row.color).
			Render(row.line))
	}
	return strings.Join(lines, "\n")
}

// RenderBanner joins the mountain decoration and the divider.
func RenderBanner() string {
	mountain := RenderMountain()
	divider := bannerDivStyle.Render(strings.Repeat("─", 44))

	return lipgloss.JoinVertical(
		lipgloss.Center,
		mountain,
		divider,
	)
}
