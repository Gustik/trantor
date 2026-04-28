// Package commands содержит CLI-команды клиента Trantor.
package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/term"
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
			return runREPL(cmd.Context(), d)
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
func runREPL(ctx context.Context, d *deps) error {
	if err := initDeps(d); err != nil {
		return err
	}
	defer closeDeps(d)

	scanner := bufio.NewScanner(os.Stdin)

	if _, err := d.vault.GetAuthToken(ctx); err != nil {
		if err := promptFirstAuth(ctx, d, scanner); err != nil {
			return err
		}
	} else {
		if err := promptPassword(ctx, d, scanner); err != nil {
			return err
		}
	}

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

// promptPassword запрашивает только пароль когда токен уже есть.
// Восстанавливает мастер-ключ из локального кэша без обращения к серверу.
// Если кэш отсутствует (старый vault) — переходит к полному auth-flow.
func promptPassword(ctx context.Context, d *deps, scanner *bufio.Scanner) error {
	masterKey, err := func() ([]byte, error) {
		password, err := readPassword(scanner)
		if err != nil {
			return nil, err
		}
		return d.authSvc.DeriveFromCache(ctx, password)
	}()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			fmt.Println("Необходимо войти заново.")
			return promptFirstAuth(ctx, d, scanner)
		}
		return err
	}
	d.masterKey = masterKey
	return nil
}

// promptFirstAuth запрашивает логин/пароль при первом запуске (нет сохранённого токена).
func promptFirstAuth(ctx context.Context, d *deps, scanner *bufio.Scanner) error {
	fmt.Println("Добро пожаловать в Trantor!")
	fmt.Println("  1. Войти")
	fmt.Println("  2. Зарегистрироваться")
	fmt.Print("Выберите действие (1/2): ")
	if !scanner.Scan() {
		return scanner.Err()
	}
	choice := strings.TrimSpace(scanner.Text())

	fmt.Print("Логин: ")
	if !scanner.Scan() {
		return scanner.Err()
	}
	login := strings.TrimSpace(scanner.Text())

	password, err := readPassword(scanner)
	if err != nil {
		return err
	}

	switch choice {
	case "2":
		masterKey, err := d.authSvc.Register(ctx, login, password)
		if err != nil {
			return fmt.Errorf("регистрация: %w", err)
		}
		d.masterKey = masterKey
		fmt.Println("Регистрация успешна.")
	default:
		masterKey, err := d.authSvc.Login(ctx, login, password)
		if err != nil {
			return fmt.Errorf("вход: %w", err)
		}
		d.masterKey = masterKey
		fmt.Println("Успешный вход.")
	}
	return nil
}

// readPassword читает пароль без отображения символов если stdin — терминал.
func readPassword(scanner *bufio.Scanner) (string, error) {
	fmt.Print("Пароль: ")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("чтение пароля: %w", err)
		}
		return string(pw), nil
	}
	if !scanner.Scan() {
		return "", scanner.Err()
	}
	return scanner.Text(), nil
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
