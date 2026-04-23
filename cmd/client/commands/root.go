// Package commands содержит CLI-команды клиента Trantor.
package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/Gustik/trantor/internal/client/auth"
	"github.com/Gustik/trantor/internal/client/grpcclient"
	"github.com/Gustik/trantor/internal/client/storage"
	"github.com/Gustik/trantor/internal/common/config"
)

type deps struct {
	vault     *storage.Vault
	client    *grpcclient.Client
	authSvc   *auth.Service
	masterKey []byte
	version   string
	buildDate string
}

// New возвращает корневую cobra-команду со всеми подкомандами.
func New(version, buildDate string) *cobra.Command {
	d := &deps{version: version, buildDate: buildDate}

	root := &cobra.Command{
		Use:   "trantor",
		Short: "Менеджер паролей с E2E-шифрованием",
		// Нет подкоманды — входим в интерактивный режим.
		RunE: func(cmd *cobra.Command, args []string) error {
			return runREPL(d)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initDeps(d)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			closeDeps(d)
			return nil
		},
	}

	root.AddCommand(newRegisterCmd(d))
	root.AddCommand(newLoginCmd(d))
	root.AddCommand(newSecretCmd(d))
	root.AddCommand(newVersionCmd(d))

	return root
}

// runREPL запускает интерактивный режим: читает команды из stdin пока не получит EOF или exit.
func runREPL(d *deps) error {
	if err := initDeps(d); err != nil {
		return err
	}
	defer closeDeps(d)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Trantor — менеджер паролей. Введите help для справки, exit для выхода.")
	for {
		fmt.Print("trantor> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}

		args, err := splitArgs(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ошибка парсинга:", err)
			continue
		}

		cmd := newInteractiveRoot(d)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	return scanner.Err()
}

// newInteractiveRoot создаёт дерево команд без хуков инициализации (deps уже готовы).
func newInteractiveRoot(d *deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "trantor",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newRegisterCmd(d))
	root.AddCommand(newLoginCmd(d))
	root.AddCommand(newSecretCmd(d))
	root.AddCommand(newVersionCmd(d))
	return root
}

// initDeps инициализирует vault и gRPC-клиент. Идемпотентна.
func initDeps(d *deps) error {
	if d.vault != nil {
		return nil
	}

	cfg, err := config.LoadClient()
	if err != nil {
		return err
	}

	vaultPath := expandPath(cfg.VaultPath)
	if err := os.MkdirAll(filepath.Dir(vaultPath), 0700); err != nil {
		return err
	}

	d.vault, err = storage.New(vaultPath)
	if err != nil {
		return err
	}

	d.client, err = grpcclient.New(*cfg)
	if err != nil {
		return err
	}

	d.authSvc = auth.New(d.client, d.vault)
	return nil
}

func closeDeps(d *deps) {
	if d.client != nil {
		_ = d.client.Close()
		d.client = nil
	}
	if d.vault != nil {
		_ = d.vault.Close()
		d.vault = nil
	}
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// splitArgs разбивает строку на аргументы с поддержкой одинарных и двойных кавычек.
func splitArgs(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	var quote rune

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("незакрытая кавычка")
	}
	if cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args, nil
}
