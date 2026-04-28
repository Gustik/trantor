package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	secretsvc "github.com/Gustik/trantor/internal/client/secret"
	commondomain "github.com/Gustik/trantor/internal/common/domain"
)

var secretTypes = []commondomain.SecretType{
	commondomain.SecretTypeText,
	commondomain.SecretTypeLoginPassword,
	commondomain.SecretTypeBinary,
	commondomain.SecretTypeBankCard,
}

type createModel struct {
	nameInput textinput.Model
	dataInput textinput.Model
	typeIdx   int
	focused   int // 0=name, 1=type, 2=data
	svc       *secretsvc.Service
	loading   bool
	err       string
}

func newCreateModel(svc *secretsvc.Service) createModel {
	name := textinput.New()
	name.Placeholder = "название"
	name.Focus()

	data := textinput.New()
	data.Placeholder = "данные"

	return createModel{nameInput: name, dataInput: data, svc: svc}
}

func (m createModel) Init() tea.Cmd { return textinput.Blink }

func (m createModel) Update(msg tea.Msg) (createModel, tea.Cmd) {
	switch msg := msg.(type) {
	case secretCreatedMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err.Error()
			return m, nil
		}
		return m, func() tea.Msg { return backMsg{} }

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEsc:
			return m, func() tea.Msg { return backMsg{} }

		case tea.KeyTab, tea.KeyDown:
			m.focused = (m.focused + 1) % 3
			m.syncFocus()
			return m, textinput.Blink

		case tea.KeyShiftTab, tea.KeyUp:
			m.focused = (m.focused + 2) % 3
			m.syncFocus()
			return m, textinput.Blink

		case tea.KeyEnter:
			switch m.focused {
			case 0: // name → type
				m.focused = 1
				m.syncFocus()
				return m, textinput.Blink
			case 1: // type → data
				m.focused = 2
				m.syncFocus()
				return m, textinput.Blink
			case 2: // data → submit
				if m.nameInput.Value() == "" || m.dataInput.Value() == "" {
					m.err = "Имя и данные обязательны"
					return m, nil
				}
				m.loading = true
				m.err = ""
				payload := &commondomain.SecretPayload{
					Type: secretTypes[m.typeIdx],
					Name: m.nameInput.Value(),
					Data: []byte(m.dataInput.Value()),
				}
				return m, func() tea.Msg {
					return secretCreatedMsg{err: m.svc.Create(context.Background(), payload)}
				}
			}

		case tea.KeyLeft, tea.KeyRight:
			if m.focused == 1 {
				if msg.Type == tea.KeyLeft {
					m.typeIdx = (m.typeIdx + len(secretTypes) - 1) % len(secretTypes)
				} else {
					m.typeIdx = (m.typeIdx + 1) % len(secretTypes)
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	switch m.focused {
	case 0:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case 2:
		m.dataInput, cmd = m.dataInput.Update(msg)
	}
	return m, cmd
}

func (m *createModel) syncFocus() {
	m.nameInput.Blur()
	m.dataInput.Blur()
	switch m.focused {
	case 0:
		m.nameInput.Focus()
	case 2:
		m.dataInput.Focus()
	}
}

func (m createModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Новый секрет") + "\n\n")

	// Поле: имя
	nameLabel := labelStyle.Render("Имя:    ")
	if m.focused == 0 {
		nameLabel = selectedItemStyle.Render("Имя:    ")
	}
	b.WriteString(nameLabel + m.nameInput.View() + "\n\n")

	// Поле: тип (выбор стрелками)
	typeLabel := labelStyle.Render("Тип:    ")
	if m.focused == 1 {
		typeLabel = selectedItemStyle.Render("Тип:    ")
	}
	b.WriteString(typeLabel + badge(secretTypes[m.typeIdx]))
	if m.focused == 1 {
		b.WriteString("  " + helpStyle.Render("← →"))
	}
	b.WriteString("\n\n")

	// Поле: данные
	dataLabel := labelStyle.Render("Данные: ")
	if m.focused == 2 {
		dataLabel = selectedItemStyle.Render("Данные: ")
	}
	b.WriteString(dataLabel + m.dataInput.View() + "\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err) + "\n\n")
	}

	if m.loading {
		b.WriteString(subtleStyle.Render("Сохранение..."))
	} else {
		b.WriteString(helpStyle.Render("tab — следующее поле  •  enter — подтвердить  •  esc — отмена"))
	}

	return b.String()
}

