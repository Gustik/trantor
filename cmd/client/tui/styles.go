package tui

import (
	"charm.land/lipgloss/v2"

	domain "github.com/Gustik/trantor/internal/client/domain"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("246"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	badgeStyles = map[domain.SecretType]lipgloss.Style{
		domain.SecretTypeLoginPassword: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		domain.SecretTypeText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("113")).
			Bold(true),
		domain.SecretTypeBinary: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
		domain.SecretTypeBankCard: lipgloss.NewStyle().
			Foreground(lipgloss.Color("204")).
			Bold(true),
	}

	confirmOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("9")).
				Padding(0, 2)
)

func formatAppTitle(version, buildDate string) string {
	title := titleStyle.Render("Trantor")
	if version == "" && buildDate == "" {
		return title
	}
	meta := version
	if buildDate != "" {
		if meta != "" {
			meta += " • " + buildDate
		} else {
			meta = buildDate
		}
	}
	return title + " " + subtleStyle.Render(meta)
}

func badge(t domain.SecretType) string {
	labels := map[domain.SecretType]string{
		domain.SecretTypeLoginPassword: "login",
		domain.SecretTypeText:          "text",
		domain.SecretTypeBinary:        "bin",
		domain.SecretTypeBankCard:      "card",
	}
	label, ok := labels[t]
	if !ok {
		label = string(t)
	}
	if s, ok := badgeStyles[t]; ok {
		return s.Render("[" + label + "]")
	}
	return "[" + label + "]"
}
