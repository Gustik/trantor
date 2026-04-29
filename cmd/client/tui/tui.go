package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gustik/trantor/internal/client/auth"
	grpcclient "github.com/Gustik/trantor/internal/client/grpcclient"
	"github.com/Gustik/trantor/internal/client/storage"
	"github.com/Gustik/trantor/internal/common/config"
)

// Start инициализирует зависимости и запускает TUI.
func Start(version, buildDate string) error {
	cfg, err := config.LoadClient()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	vaultPath, err := expandPath(cfg.VaultPath)
	if err != nil {
		return fmt.Errorf("expand vault path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(vaultPath), 0700); err != nil {
		return fmt.Errorf("create vault dir: %w", err)
	}

	vault, err := storage.New(vaultPath)
	if err != nil {
		return fmt.Errorf("open vault: %w", err)
	}
	defer func() { _ = vault.Close() }()

	client, err := grpcclient.New(*cfg)
	if err != nil {
		return fmt.Errorf("create grpc client: %w", err)
	}
	defer func() { _ = client.Close() }()

	authSvc := auth.New(client, vault)

	_, tokenErr := vault.GetAuthToken(context.Background())
	hasToken := tokenErr == nil

	root := newRoot(authSvc, vault, client, hasToken)
	_, err = tea.NewProgram(root, tea.WithAltScreen()).Run()
	return err
}

func expandPath(p string) (string, error) {
	if len(p) >= 2 && p[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}
