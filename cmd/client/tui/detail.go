package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/atotto/clipboard"
	"github.com/google/uuid"

	secretsvc "github.com/Gustik/trantor/internal/client/secret"
)

type detailModel struct {
	id            uuid.UUID
	svc           *secretsvc.Service
	name          string
	secretType    string
	data          string
	metadata      map[string]string
	loading       bool
	confirmDelete bool
	copied        bool
	err           string
}

func newDetailModel(id uuid.UUID, svc *secretsvc.Service) detailModel {
	return detailModel{id: id, svc: svc, loading: true}
}

func (m detailModel) Init() tea.Cmd {
	return loadDetailCmd(m.id, m.svc)
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case detailLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.name = msg.name
		m.secretType = msg.secretType
		m.data = msg.data
		m.metadata = msg.metadata

	case secretDeletedMsg:
		if msg.err != nil {
			m.confirmDelete = false
			m.err = msg.err.Error()
			return m, nil
		}
		return m, func() tea.Msg { return backMsg{} }

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y", "д", "Д":
				id := m.id
				return m, func() tea.Msg {
					return secretDeletedMsg{err: m.svc.Delete(context.Background(), id)}
				}
			default:
				m.confirmDelete = false
			}
			return m, nil
		}
		switch msg.String() {
		case "c", "C":
			if err := clipboard.WriteAll(m.data); err == nil {
				m.copied = true
			}
		case "d", "D":
			m.confirmDelete = true
			m.copied = false
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		}
	}
	return m, nil
}

func (m detailModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Секрет") + "\n\n")

	if m.loading {
		b.WriteString(subtleStyle.Render("загрузка...") + "\n")
		return b.String()
	}

	if m.err != "" {
		b.WriteString(errorStyle.Render("Ошибка: "+m.err) + "\n\n")
		b.WriteString(helpStyle.Render("esc — назад"))
		return b.String()
	}

	b.WriteString(labelStyle.Render("Имя:    ") + valueStyle.Render(m.name) + "\n")
	b.WriteString(labelStyle.Render("Тип:    ") + valueStyle.Render(m.secretType) + "\n")
	b.WriteString(labelStyle.Render("Данные: ") + valueStyle.Render(m.data) + "\n")

	if len(m.metadata) > 0 {
		b.WriteString(labelStyle.Render("Метаданные:") + "\n")
		for k, v := range m.metadata {
			b.WriteString("  " + subtleStyle.Render(k+": ") + valueStyle.Render(v) + "\n")
		}
	}

	b.WriteString("\n")

	if m.copied {
		b.WriteString(successStyle.Render("✓ скопировано в буфер обмена") + "\n\n")
	}

	if m.confirmDelete {
		b.WriteString(confirmOverlayStyle.Render(
			"Удалить секрет?\n"+
				errorStyle.Render("y")+" — да  •  любая другая клавиша — отмена",
		) + "\n")
	} else {
		b.WriteString(helpStyle.Render("c — копировать  •  d — удалить  •  esc — назад"))
	}

	return b.String()
}

func loadDetailCmd(id uuid.UUID, svc *secretsvc.Service) tea.Cmd {
	return func() tea.Msg {
		payload, err := svc.Get(context.Background(), id)
		if err != nil {
			return detailLoadedMsg{err: err}
		}
		return detailLoadedMsg{
			name:       payload.Name,
			secretType: string(payload.Type),
			data:       string(payload.Data),
			metadata:   payload.Metadata,
		}
	}
}
