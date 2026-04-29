package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/Gustik/trantor/internal/client/domain"
	secretsvc "github.com/Gustik/trantor/internal/client/secret"
	"github.com/Gustik/trantor/internal/client/storage"
)

type listItem struct {
	id         uuid.UUID
	secretType domain.SecretType
	name       string
}

type listModel struct {
	items         []listItem
	cursor        int
	loading       bool
	syncing       bool
	initialLoad   bool
	confirmLogout bool
	err           string
	spinner       spinner.Model
	vault         *storage.Vault
	svc           *secretsvc.Service
}

func newListModel(vault *storage.Vault, svc *secretsvc.Service) listModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return listModel{vault: vault, svc: svc, spinner: s, loading: true, initialLoad: true}
}

func (m listModel) Init() tea.Cmd {
	return tea.Batch(loadSecretsCmd(m.vault), m.spinner.Tick)
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case secretsLoadedMsg:
		m.loading = false
		firstLoad := m.initialLoad
		m.initialLoad = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.items = toListItems(msg.secrets)
		if m.cursor >= len(m.items) {
			m.cursor = max(0, len(m.items)-1)
		}
		if firstLoad && len(m.items) == 0 && !m.syncing {
			m.syncing = true
			return m, tea.Batch(syncCmd(m.svc), m.spinner.Tick)
		}

	case syncDoneMsg:
		m.syncing = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		return m, loadSecretsCmd(m.vault)

	case spinner.TickMsg:
		if m.loading || m.syncing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		if m.confirmLogout {
			switch msg.String() {
			case "y", "Y", "д", "Д":
				return m, func() tea.Msg { return logoutMsg{} }
			default:
				m.confirmLogout = false
			}
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.items) > 0 {
				id := m.items[m.cursor].id
				return m, func() tea.Msg { return secretSelectedMsg{id: id} }
			}
		case "n", "N":
			return m, func() tea.Msg { return createSecretMsg{} }
		case "s", "S":
			if !m.syncing {
				m.syncing = true
				m.err = ""
				return m, tea.Batch(syncCmd(m.svc), m.spinner.Tick)
			}
		case "L":
			m.confirmLogout = true
		case "q", "Q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m listModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Trantor") + "\n\n")

	if m.loading {
		b.WriteString(m.spinner.View() + " загрузка...\n")
		return b.String()
	}

	if m.err != "" {
		b.WriteString(errorStyle.Render("Ошибка: "+m.err) + "\n\n")
	}

	if len(m.items) == 0 {
		b.WriteString(subtleStyle.Render("Секретов нет. Нажмите n чтобы создать.") + "\n")
	} else {
		for i, item := range m.items {
			prefix := "  "
			style := normalItemStyle
			if i == m.cursor {
				prefix = "▶ "
				style = selectedItemStyle
			}
			fmt.Fprintf(&b, "%s%s %s\n", prefix, badge(item.secretType), style.Render(item.name))
		}
	}

	b.WriteString("\n")
	if m.syncing {
		b.WriteString(subtleStyle.Render(m.spinner.View() + " синхронизация..."))
	} else if m.confirmLogout {
		b.WriteString(confirmOverlayStyle.Render(
			"Выйти из аккаунта? Все локальные данные будут удалены.\n" +
				errorStyle.Render("y") + " — да  •  любая другая клавиша — отмена",
		))
	} else {
		b.WriteString(helpStyle.Render("↑↓ — навигация  •  enter — открыть  •  n — создать  •  s — sync  •  L — выйти  •  q — quit"))
	}

	return b.String()
}

func toListItems(secrets []*domain.Secret) []listItem {
	items := make([]listItem, len(secrets))
	for i, s := range secrets {
		items[i] = listItem{id: s.ID, secretType: s.Type, name: s.Name}
	}
	return items
}

func loadSecretsCmd(vault *storage.Vault) tea.Cmd {
	return func() tea.Msg {
		secrets, err := vault.ListSecrets(context.Background())
		return secretsLoadedMsg{secrets: secrets, err: err}
	}
}

func syncCmd(svc *secretsvc.Service) tea.Cmd {
	return func() tea.Msg {
		return syncDoneMsg{err: svc.Sync(context.Background())}
	}
}

