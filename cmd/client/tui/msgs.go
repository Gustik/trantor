package tui

import (
	"github.com/google/uuid"

	"github.com/Gustik/trantor/internal/client/domain"
)

// Переходы между экранами — обрабатываются rootModel.
type authSuccessMsg  struct{ masterKey []byte }
type backMsg         struct{}
type logoutMsg       struct{}
type secretSelectedMsg struct{ id uuid.UUID }
type createSecretMsg struct{}

// Результаты async-операций — обрабатываются sub-models.
type secretsLoadedMsg struct {
	secrets []*domain.Secret
	err     error
}
type secretCreatedMsg struct{ err error }
type secretDeletedMsg struct{ err error }
type syncDoneMsg      struct{ err error }
type detailLoadedMsg  struct {
	name       string
	secretType string
	data       string
	metadata   map[string]string
	err        error
}

// authErrMsg — ошибка аутентификации, обрабатывается внутри auth/password экранов.
type authErrMsg struct{ err error }
