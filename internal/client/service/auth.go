package service

import "context"

// Register регистрирует нового пользователя на сервере.
// Вычисляет auth_key и encryption_key из пароля через Argon2 на клиенте.
func (s *Service) Register(ctx context.Context, login, password string) error {
	return nil
}

// GetSalt возвращает argon2 salt пользователя с сервера.
func (s *Service) GetSalt(ctx context.Context, login string) ([]byte, error) {
	return nil, nil
}

// Login аутентифицирует пользователя.
// Вычисляет auth_key из пароля, получает мастер-ключ и сохраняет токен локально.
func (s *Service) Login(ctx context.Context, login, password string) error {
	return nil
}
