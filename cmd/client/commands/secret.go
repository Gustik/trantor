package commands

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	secretsvc "github.com/Gustik/trantor/internal/client/secret"
	commondomain "github.com/Gustik/trantor/internal/common/domain"
)

func newSecretCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Управление секретами",
	}
	cmd.AddCommand(newSecretCreateCmd(d))
	cmd.AddCommand(newSecretListCmd(d))
	cmd.AddCommand(newSecretGetCmd(d))
	cmd.AddCommand(newSecretDeleteCmd(d))
	cmd.AddCommand(newSecretSyncCmd(d))
	return cmd
}

func newSecretCreateCmd(d *deps) *cobra.Command {
	var login, password, name, secretType, data string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Создать новый секрет",
		RunE: func(cmd *cobra.Command, args []string) error {
			masterKey, err := resolveMasterKey(cmd, d, login, password)
			if err != nil {
				return err
			}

			svc := secretsvc.New(d.client, d.vault, masterKey)
			payload := &commondomain.SecretPayload{
				Type: commondomain.SecretType(secretType),
				Name: name,
				Data: []byte(data),
			}
			if err := svc.Create(cmd.Context(), payload); err != nil {
				return err
			}
			fmt.Println("Секрет создан")
			return nil
		},
	}

	cmd.Flags().StringVarP(&login, "login", "l", "", "логин (не нужен если уже выполнен login)")
	cmd.Flags().StringVarP(&password, "password", "p", "", "пароль (не нужен если уже выполнен login)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "имя секрета")
	cmd.Flags().StringVarP(&secretType, "type", "t", string(commondomain.SecretTypeText), "тип: text, login_password, binary, bank_card")
	cmd.Flags().StringVarP(&data, "data", "d", "", "данные секрета")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("data")

	return cmd
}

// newSecretListCmd выводит список секретов из локального vault.
// Мастер-ключ не нужен — type/name хранятся в открытом виде.
func newSecretListCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Показать список секретов",
		RunE: func(cmd *cobra.Command, args []string) error {
			secrets, err := d.vault.ListSecrets(cmd.Context())
			if err != nil {
				return err
			}
			if len(secrets) == 0 {
				fmt.Println("Секреты не найдены")
				return nil
			}
			for _, s := range secrets {
				fmt.Printf("%s  [%s]  %s\n", s.ID, s.Type, s.Name)
			}
			return nil
		},
	}
}

// newSecretGetCmd выводит полные данные секрета по UUID.
// Требует мастер-ключ для расшифровки поля Data.
func newSecretGetCmd(d *deps) *cobra.Command {
	var login, password string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Получить секрет по ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("неверный UUID: %w", err)
			}
			masterKey, err := resolveMasterKey(cmd, d, login, password)
			if err != nil {
				return err
			}
			svc := secretsvc.New(d.client, d.vault, masterKey)
			payload, err := svc.Get(cmd.Context(), id)
			if err != nil {
				return err
			}
			fmt.Printf("ID:     %s\n", args[0])
			fmt.Printf("Тип:    %s\n", payload.Type)
			fmt.Printf("Имя:    %s\n", payload.Name)
			fmt.Printf("Данные: %s\n", payload.Data)
			if len(payload.Metadata) > 0 {
				fmt.Printf("Метаданные: %v\n", payload.Metadata)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&login, "login", "l", "", "логин (не нужен если уже выполнен login)")
	cmd.Flags().StringVarP(&password, "password", "p", "", "пароль (не нужен если уже выполнен login)")

	return cmd
}

// newSecretDeleteCmd удаляет секрет на сервере и из локального vault.
// Мастер-ключ не нужен — Delete использует только токен авторизации.
func newSecretDeleteCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Удалить секрет",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("неверный UUID: %w", err)
			}
			svc := secretsvc.New(d.client, d.vault, nil)
			if err := svc.Delete(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Println("Секрет удалён")
			return nil
		},
	}
}

// newSecretSyncCmd запускает фоновую синхронизацию каждые 10 секунд.
// Первая синхронизация выполняется сразу при старте.
// Останавливается по Ctrl+C (SIGINT/SIGTERM).
// Ошибки одного цикла не прерывают следующие — временная недоступность сервера не роняет команду.
func newSecretSyncCmd(d *deps) *cobra.Command {
	var login, password string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Синхронизировать секреты с сервером (каждые 10 секунд)",
		RunE: func(cmd *cobra.Command, args []string) error {
			masterKey, err := resolveMasterKey(cmd, d, login, password)
			if err != nil {
				return err
			}

			svc := secretsvc.New(d.client, d.vault, masterKey)

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			fmt.Println("Синхронизация запущена. Нажмите Ctrl+C для остановки.")

			doSync := func() {
				if err := svc.Sync(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}
					fmt.Fprintf(os.Stderr, "ошибка синхронизации: %v\n", err)
				} else {
					fmt.Printf("[%s] синхронизация выполнена\n", time.Now().Format("15:04:05"))
				}
			}

			doSync()

			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					doSync()
				}
			}
		},
	}

	cmd.Flags().StringVarP(&login, "login", "l", "", "логин (не нужен если уже выполнен login)")
	cmd.Flags().StringVarP(&password, "password", "p", "", "пароль (не нужен если уже выполнен login)")

	return cmd
}

// resolveMasterKey возвращает masterKey из deps если он уже есть,
// иначе выполняет login с переданными флагами.
func resolveMasterKey(cmd *cobra.Command, d *deps, login, password string) ([]byte, error) {
	if d.masterKey != nil {
		return d.masterKey, nil
	}
	if login == "" || password == "" {
		return nil, errors.New("сначала выполните команду login, либо укажите --login и --password")
	}
	masterKey, err := d.authSvc.Login(cmd.Context(), login, password)
	if err != nil {
		return nil, err
	}
	d.masterKey = masterKey
	return masterKey, nil
}
