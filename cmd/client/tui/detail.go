package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"

	"github.com/Gustik/trantor/internal/client/domain"
	secretsvc "github.com/Gustik/trantor/internal/client/secret"
)

type detailModel struct {
	id            uuid.UUID
	svc           *secretsvc.Service
	name          string
	secretType    string
	data          []byte
	metadata      map[string]string
	fp            filepicker.Model
	pickingDir    bool
	loading       bool
	confirmDelete bool
	copied        bool
	saved         bool
	savedPath     string
	loadErr       string
	opErr         string
}

func newDetailModel(id uuid.UUID, svc *secretsvc.Service) detailModel {
	fp := filepicker.New()
	if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = home
	}
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.AutoHeight = false
	fp.SetHeight(10)

	return detailModel{id: id, svc: svc, loading: true, fp: fp}
}

func (m detailModel) Init() tea.Cmd {
	return loadDetailCmd(m.id, m.svc)
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	if m.pickingDir {
		return m.updateDirPicker(msg)
	}

	switch msg := msg.(type) {
	case detailLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err.Error()
			return m, nil
		}
		m.name = msg.name
		m.secretType = msg.secretType
		m.data = msg.data
		m.metadata = msg.metadata

	case secretDeletedMsg:
		if msg.err != nil {
			m.confirmDelete = false
			m.opErr = msg.err.Error()
			return m, nil
		}
		return m, func() tea.Msg { return backMsg{} }

	case tea.KeyPressMsg:
		if m.loading {
			return m, nil
		}
		if m.confirmDelete {
			switch msg.String() {
			case "y", "д":
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
			if m.secretType != string(domain.SecretTypeBinary) {
				m.copied = true
				m.opErr = ""
				return m, tea.SetClipboard(string(m.data))
			}
		case "s", "S":
			if m.secretType == string(domain.SecretTypeBinary) {
				m.pickingDir = true
				m.saved = false
				m.opErr = ""
				return m, m.fp.Init()
			}
		case "d", "D":
			m.confirmDelete = true
			m.copied = false
			m.saved = false
			m.opErr = ""
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		}
	}
	return m, nil
}

func (m detailModel) updateDirPicker(msg tea.Msg) (detailModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.Code == tea.KeyEsc {
		m.pickingDir = false
		return m, nil
	}

	var cmd tea.Cmd
	m.fp, cmd = m.fp.Update(msg)
	if ok, dir := m.fp.DidSelectFile(msg); ok {
		filename := m.metadata["filename"]
		if filename == "" {
			filename = m.name
		}
		dest := filepath.Join(dir, filename)
		if err := os.WriteFile(dest, m.data, 0o600); err != nil {
			m.opErr = err.Error()
		} else {
			m.saved = true
			m.savedPath = dest
		}
		m.pickingDir = false
		return m, nil
	}
	return m, cmd
}

func (m detailModel) View() string {
	if m.pickingDir {
		var b strings.Builder
		b.WriteString(titleStyle.Render("Выбор папки для сохранения") + "\n\n")
		b.WriteString(m.fp.View() + "\n")
		b.WriteString(helpStyle.Render("enter/l/→ — открыть  •  h/←/backspace — назад  •  esc — отмена"))
		return b.String()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Секрет") + "\n\n")

	if m.loading {
		b.WriteString(subtleStyle.Render("загрузка...") + "\n")
		return b.String()
	}

	if m.loadErr != "" {
		b.WriteString(errorStyle.Render("Ошибка: "+m.loadErr) + "\n\n")
		b.WriteString(helpStyle.Render("esc — назад"))
		return b.String()
	}

	b.WriteString(labelStyle.Render("Имя:    ") + valueStyle.Render(m.name) + "\n")
	b.WriteString(labelStyle.Render("Тип:    ") + valueStyle.Render(m.secretType) + "\n")

	isBinary := m.secretType == string(domain.SecretTypeBinary)
	if isBinary {
		filename := m.metadata["filename"]
		if filename == "" {
			filename = m.name
		}
		b.WriteString(labelStyle.Render("Файл:   ") + valueStyle.Render(filename) + "\n")
		b.WriteString(labelStyle.Render("Размер: ") + valueStyle.Render(fmt.Sprintf("%d байт", len(m.data))) + "\n")
	} else {
		b.WriteString(labelStyle.Render("Данные: ") + valueStyle.Render(string(m.data)) + "\n")
	}

	if len(m.metadata) > 0 {
		b.WriteString(labelStyle.Render("Метаданные:") + "\n")
		for k, v := range m.metadata {
			if k == "filename" {
				continue
			}
			b.WriteString("  " + subtleStyle.Render(k+": ") + valueStyle.Render(v) + "\n")
		}
	}

	b.WriteString("\n")

	if m.opErr != "" {
		b.WriteString(errorStyle.Render("Ошибка: "+m.opErr) + "\n\n")
	}
	if m.copied {
		b.WriteString(successStyle.Render("✓ скопировано в буфер обмена") + "\n\n")
	}
	if m.saved {
		b.WriteString(successStyle.Render("✓ сохранено: "+m.savedPath) + "\n\n")
	}

	if m.confirmDelete {
		b.WriteString(confirmOverlayStyle.Render(
			"Удалить секрет?\n"+
				errorStyle.Render("y")+" — да  •  любая другая клавиша — отмена",
		) + "\n")
	} else if isBinary {
		b.WriteString(helpStyle.Render("s — сохранить файл  •  d — удалить  •  esc — назад"))
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
			data:       payload.Data,
			metadata:   payload.Metadata,
		}
	}
}
