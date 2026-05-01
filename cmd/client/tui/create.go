package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/Gustik/trantor/internal/client/domain"
	secretsvc "github.com/Gustik/trantor/internal/client/secret"
)

var secretTypes = []domain.SecretType{
	domain.SecretTypeText,
	domain.SecretTypeLoginPassword,
	domain.SecretTypeBinary,
	domain.SecretTypeBankCard,
}

type createModel struct {
	nameInput   textinput.Model
	dataInput   textinput.Model
	fp          filepicker.Model
	pickingFile bool
	typeIdx     int
	focused     int // 0=name, 1=type, 2=data
	svc         *secretsvc.Service
	loading     bool
	err         string
}

func newCreateModel(svc *secretsvc.Service) createModel {
	name := textinput.New()
	name.Placeholder = "название"
	name.SetWidth(30)
	name.Focus()

	data := textinput.New()
	data.Placeholder = "данные"
	data.SetWidth(30)

	fp := filepicker.New()
	if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = home
	}
	fp.AutoHeight = false
	fp.SetHeight(10)

	return createModel{nameInput: name, dataInput: data, fp: fp, svc: svc}
}

func (m createModel) Init() tea.Cmd { return textinput.Blink }

func (m createModel) Update(msg tea.Msg) (createModel, tea.Cmd) {
	if m.pickingFile {
		return m.updateFilePicker(msg)
	}

	switch msg := msg.(type) {
	case secretCreatedMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err.Error()
			return m, nil
		}
		return m, func() tea.Msg { return backMsg{} }

	case tea.KeyPressMsg:
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return backMsg{} }

		case "tab", "down":
			m.focused = (m.focused + 1) % 3
			m.syncFocus()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.focused = (m.focused + 2) % 3
			m.syncFocus()
			return m, textinput.Blink

		case "enter":
			switch m.focused {
			case 0:
				m.focused = 1
				m.syncFocus()
				return m, textinput.Blink
			case 1:
				m.focused = 2
				m.syncFocus()
				return m, textinput.Blink
			case 2:
				if m.nameInput.Value() == "" || m.dataInput.Value() == "" {
					m.err = "Имя и данные обязательны"
					return m, nil
				}
				m.loading = true
				m.err = ""
				secretType := secretTypes[m.typeIdx]
				name := m.nameInput.Value()
				inputVal := m.dataInput.Value()
				return m, func() tea.Msg {
					payload := &domain.SecretPayload{
						Type: secretType,
						Name: name,
					}
					if secretType == domain.SecretTypeBinary {
						content, err := os.ReadFile(inputVal)
						if err != nil {
							return secretCreatedMsg{err: err}
						}
						payload.Data = content
						payload.Metadata = map[string]string{"filename": filepath.Base(inputVal)}
					} else {
						payload.Data = []byte(inputVal)
					}
					return secretCreatedMsg{err: m.svc.Create(context.Background(), payload)}
				}
			}

		case "left":
			if m.focused == 1 {
				m.typeIdx = (m.typeIdx + len(secretTypes) - 1) % len(secretTypes)
				m.updateDataPlaceholder()
				return m, nil
			}

		case "right":
			if m.focused == 1 {
				m.typeIdx = (m.typeIdx + 1) % len(secretTypes)
				m.updateDataPlaceholder()
				return m, nil
			}

		case "f":
			if m.focused == 2 && secretTypes[m.typeIdx] == domain.SecretTypeBinary {
				m.pickingFile = true
				return m, m.fp.Init()
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

func (m createModel) updateFilePicker(msg tea.Msg) (createModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.Code == tea.KeyEsc {
		m.pickingFile = false
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.fp, cmd = m.fp.Update(msg)
	if ok, path := m.fp.DidSelectFile(msg); ok {
		m.dataInput.SetValue(path)
		m.pickingFile = false
		return m, textinput.Blink
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

func (m *createModel) updateDataPlaceholder() {
	if secretTypes[m.typeIdx] == domain.SecretTypeBinary {
		m.dataInput.Placeholder = "путь к файлу"
	} else {
		m.dataInput.Placeholder = "данные"
	}
}

func (m createModel) View() string {
	if m.pickingFile {
		var b strings.Builder
		b.WriteString(titleStyle.Render("Выбор файла") + "\n\n")
		b.WriteString(m.fp.View() + "\n")
		b.WriteString(helpStyle.Render("enter/l/→ — открыть  •  h/←/backspace — назад  •  esc — отмена"))
		return b.String()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Новый секрет") + "\n\n")

	nameLabel := labelStyle.Render("Имя:    ")
	if m.focused == 0 {
		nameLabel = selectedItemStyle.Render("Имя:    ")
	}
	b.WriteString(nameLabel + m.nameInput.View() + "\n\n")

	typeLabel := labelStyle.Render("Тип:    ")
	if m.focused == 1 {
		typeLabel = selectedItemStyle.Render("Тип:    ")
	}
	b.WriteString(typeLabel + badge(secretTypes[m.typeIdx]))
	if m.focused == 1 {
		b.WriteString("  " + helpStyle.Render("← →"))
	}
	b.WriteString("\n\n")

	isBinary := secretTypes[m.typeIdx] == domain.SecretTypeBinary
	dataLabelText := "Данные: "
	if isBinary {
		dataLabelText = "Файл:   "
	}
	dataLabel := labelStyle.Render(dataLabelText)
	if m.focused == 2 {
		dataLabel = selectedItemStyle.Render(dataLabelText)
	}
	b.WriteString(dataLabel + m.dataInput.View() + "\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err) + "\n\n")
	}

	if m.loading {
		b.WriteString(subtleStyle.Render("Сохранение..."))
	} else if isBinary && m.focused == 2 {
		b.WriteString(helpStyle.Render("f — выбрать файл  •  enter — подтвердить  •  esc — отмена"))
	} else {
		b.WriteString(helpStyle.Render("tab — следующее поле  •  enter — подтвердить  •  esc — отмена"))
	}

	return b.String()
}
