package commands

import (
	"errors"
	"fmt"

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

func newSecretListCmd(d *deps) *cobra.Command {
	var login, password string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список секретов",
		RunE: func(cmd *cobra.Command, args []string) error {
			masterKey, err := resolveMasterKey(cmd, d, login, password)
			if err != nil {
				return err
			}

			svc := secretsvc.New(d.client, d.vault, masterKey)
			secrets, err := svc.List(cmd.Context())
			if err != nil {
				return err
			}

			if len(secrets) == 0 {
				fmt.Println("Секреты не найдены")
				return nil
			}
			for _, s := range secrets {
				fmt.Printf("[%s] %s: %s\n", s.Type, s.Name, s.Data)
			}
			return nil
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
