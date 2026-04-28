package tui

import (
	"github.com/charmbracelet/lipgloss"

	commondomain "github.com/Gustik/trantor/internal/common/domain"
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

	badgeStyles = map[commondomain.SecretType]lipgloss.Style{
		commondomain.SecretTypeLoginPassword: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		commondomain.SecretTypeText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("113")).
			Bold(true),
		commondomain.SecretTypeBinary: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
		commondomain.SecretTypeBankCard: lipgloss.NewStyle().
			Foreground(lipgloss.Color("204")).
			Bold(true),
	}

	confirmOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("9")).
				Padding(0, 2)
)

func badge(t commondomain.SecretType) string {
	labels := map[commondomain.SecretType]string{
		commondomain.SecretTypeLoginPassword: "login",
		commondomain.SecretTypeText:          "text",
		commondomain.SecretTypeBinary:        "bin",
		commondomain.SecretTypeBankCard:      "card",
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
