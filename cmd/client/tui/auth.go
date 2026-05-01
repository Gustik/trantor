package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/Gustik/trantor/internal/client/auth"
	"github.com/Gustik/trantor/internal/client/storage"
)

// passwordModel — экран ввода пароля для возвращающегося пользователя.
type passwordModel struct {
	input   textinput.Model
	authSvc *auth.Service
	err     string
	loading bool
}

func newPasswordModel(authSvc *auth.Service) passwordModel {
	ti := textinput.New()
	ti.Placeholder = "пароль"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	return passwordModel{input: ti, authSvc: authSvc}
}

func (m passwordModel) Init() tea.Cmd { return textinput.Blink }

func (m passwordModel) Update(msg tea.Msg) (passwordModel, tea.Cmd) {
	switch msg := msg.(type) {
	case authErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		m.input.SetValue("")
		return m, textinput.Blink

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.Code == tea.KeyEnter && !m.loading && m.input.Value() != "" {
			password := m.input.Value()
			m.loading = true
			m.err = ""
			svc := m.authSvc
			return m, func() tea.Msg {
				mk, err := svc.DeriveFromCache(context.Background(), password)
				if err != nil {
					return authErrMsg{err: err}
				}
				return authSuccessMsg{masterKey: mk}
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m passwordModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Trantor") + "\n\n")
	b.WriteString(labelStyle.Render("Пароль: ") + m.input.View() + "\n\n")
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err) + "\n\n")
	}
	if m.loading {
		b.WriteString(subtleStyle.Render("Проверка..."))
	} else {
		b.WriteString(helpStyle.Render("enter — войти  •  q — выйти"))
	}
	return b.String()
}

// authModel — экран первого входа: выбор login/register, затем форма.
type authStep int

const (
	authStepChoice authStep = iota
	authStepForm
)

type authModel struct {
	step    authStep
	choice  int // 0=login, 1=register
	inputs  [2]textinput.Model
	focused int
	authSvc *auth.Service
	vault   *storage.Vault
	err     string
	loading bool
}

func newAuthModel(authSvc *auth.Service, vault *storage.Vault) authModel {
	login := textinput.New()
	login.Placeholder = "логин"
	login.Focus()

	pass := textinput.New()
	pass.Placeholder = "пароль"
	pass.EchoMode = textinput.EchoPassword
	pass.EchoCharacter = '•'

	return authModel{
		authSvc: authSvc,
		vault:   vault,
		inputs:  [2]textinput.Model{login, pass},
	}
}

func (m authModel) Init() tea.Cmd { return textinput.Blink }

func (m authModel) Update(msg tea.Msg) (authModel, tea.Cmd) {
	switch msg := msg.(type) {
	case authErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		m.inputs[1].SetValue("")
		return m, textinput.Blink

	case tea.KeyPressMsg:
		switch m.step {
		case authStepChoice:
			switch msg.String() {
			case "1", "l":
				m.choice = 0
				m.step = authStepForm
				return m, textinput.Blink
			case "2", "r":
				m.choice = 1
				m.step = authStepForm
				return m, textinput.Blink
			}

		case authStepForm:
			switch msg.String() {
			case "esc":
				m.step = authStepChoice
				return m, nil

			case "tab", "down":
				m.inputs[m.focused].Blur()
				m.focused = (m.focused + 1) % 2
				m.inputs[m.focused].Focus()
				return m, textinput.Blink

			case "enter":
				if m.focused == 0 {
					m.inputs[0].Blur()
					m.focused = 1
					m.inputs[1].Focus()
					return m, textinput.Blink
				}
				if m.loading {
					return m, nil
				}
				login, password := m.inputs[0].Value(), m.inputs[1].Value()
				if login == "" || password == "" {
					m.err = "Логин и пароль не могут быть пустыми"
					return m, nil
				}
				m.loading = true
				m.err = ""
				svc := m.authSvc
				isRegister := m.choice == 1
				return m, func() tea.Msg {
					if isRegister {
						mk, err := svc.Register(context.Background(), login, password)
						if err != nil {
							return authErrMsg{err: err}
						}
						return authSuccessMsg{masterKey: mk}
					}
					mk, err := svc.Login(context.Background(), login, password)
					if err != nil {
						return authErrMsg{err: err}
					}
					return authSuccessMsg{masterKey: mk}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m authModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Trantor — менеджер паролей") + "\n\n")

	switch m.step {
	case authStepChoice:
		b.WriteString("  " + selectedItemStyle.Render("1.") + " Войти\n")
		b.WriteString("  " + selectedItemStyle.Render("2.") + " Зарегистрироваться\n\n")
		b.WriteString(helpStyle.Render("Нажмите 1 или 2"))

	case authStepForm:
		action := "Вход"
		if m.choice == 1 {
			action = "Регистрация"
		}
		b.WriteString(labelStyle.Render(action) + "\n\n")
		fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Логин:  "), m.inputs[0].View())
		fmt.Fprintf(&b, "%s %s\n\n", labelStyle.Render("Пароль: "), m.inputs[1].View())
		if m.err != "" {
			b.WriteString(errorStyle.Render(m.err) + "\n\n")
		}
		if m.loading {
			b.WriteString(subtleStyle.Render("Загрузка..."))
		} else {
			b.WriteString(helpStyle.Render("tab — следующее поле  •  enter — подтвердить  •  esc — назад"))
		}
	}
	return b.String()
}
