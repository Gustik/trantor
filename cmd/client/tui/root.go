package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/Gustik/trantor/internal/client/auth"
	grpcclient "github.com/Gustik/trantor/internal/client/grpcclient"
	secretsvc "github.com/Gustik/trantor/internal/client/secret"
	"github.com/Gustik/trantor/internal/client/storage"
)

type screen int

const (
	screenPassword screen = iota
	screenAuth
	screenList
	screenDetail
	screenCreate
)

type rootModel struct {
	screen     screen
	password   passwordModel
	auth       authModel
	list       listModel
	detail     detailModel
	create     createModel
	authSvc    *auth.Service
	vault      *storage.Vault
	grpcClient *grpcclient.Client
	masterKey  []byte
	width      int
	height     int
	appTitle   string
}

func newRoot(authSvc *auth.Service, vault *storage.Vault, client *grpcclient.Client, hasToken bool, version, buildDate string) rootModel {
	title := formatAppTitle(version, buildDate)
	m := rootModel{authSvc: authSvc, vault: vault, grpcClient: client, appTitle: title}
	if hasToken {
		m.screen = screenPassword
		m.password = newPasswordModel(authSvc, title)
	} else {
		m.screen = screenAuth
		m.auth = newAuthModel(authSvc, vault, title)
	}
	return m
}

func (m rootModel) Init() tea.Cmd {
	switch m.screen {
	case screenPassword:
		return m.password.Init()
	case screenAuth:
		return m.auth.Init()
	}
	return nil
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case authSuccessMsg:
		m.masterKey = msg.masterKey
		m.screen = screenList
		m.list = newListModel(m.vault, m.newSecretSvc(), m.appTitle)
		return m, m.list.Init()

	case secretSelectedMsg:
		m.screen = screenDetail
		m.detail = newDetailModel(msg.id, m.newSecretSvc())
		return m, m.detail.Init()

	case createSecretMsg:
		m.screen = screenCreate
		m.create = newCreateModel(m.newSecretSvc())
		return m, m.create.Init()

	case backMsg:
		m.screen = screenList
		m.list = newListModel(m.vault, m.newSecretSvc(), m.appTitle)
		return m, m.list.Init()

	case logoutMsg:
		_ = m.vault.Clear(context.Background())
		m.masterKey = nil
		m.screen = screenAuth
		m.auth = newAuthModel(m.authSvc, m.vault, m.appTitle)
		return m, m.auth.Init()
	}

	return m.delegateUpdate(msg)
}

func (m rootModel) delegateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.screen {
	case screenPassword:
		m.password, cmd = m.password.Update(msg)
	case screenAuth:
		m.auth, cmd = m.auth.Update(msg)
	case screenList:
		m.list, cmd = m.list.Update(msg)
	case screenDetail:
		m.detail, cmd = m.detail.Update(msg)
	case screenCreate:
		m.create, cmd = m.create.Update(msg)
	}
	return m, cmd
}

func (m rootModel) View() tea.View {
	var content string
	switch m.screen {
	case screenPassword:
		content = m.password.View()
	case screenAuth:
		content = m.auth.View()
	case screenList:
		content = m.list.View()
	case screenDetail:
		content = m.detail.View()
	case screenCreate:
		content = m.create.View()
	}
	return tea.View{Content: content, AltScreen: true}
}

func (m rootModel) newSecretSvc() *secretsvc.Service {
	return secretsvc.New(m.grpcClient, m.vault, m.masterKey)
}
