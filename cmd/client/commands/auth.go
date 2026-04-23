package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRegisterCmd(d *deps) *cobra.Command {
	var login, password string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Зарегистрировать нового пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			masterKey, err := d.authSvc.Register(cmd.Context(), login, password)
			if err != nil {
				return err
			}
			d.masterKey = masterKey
			fmt.Println("Регистрация успешна")
			return nil
		},
	}

	cmd.Flags().StringVarP(&login, "login", "l", "", "логин")
	cmd.Flags().StringVarP(&password, "password", "p", "", "пароль")
	_ = cmd.MarkFlagRequired("login")
	_ = cmd.MarkFlagRequired("password")

	return cmd
}

func newLoginCmd(d *deps) *cobra.Command {
	var login, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Войти в аккаунт",
		RunE: func(cmd *cobra.Command, args []string) error {
			masterKey, err := d.authSvc.Login(cmd.Context(), login, password)
			if err != nil {
				return err
			}
			d.masterKey = masterKey
			fmt.Println("Успешный вход")
			return nil
		},
	}

	cmd.Flags().StringVarP(&login, "login", "l", "", "логин")
	cmd.Flags().StringVarP(&password, "password", "p", "", "пароль")
	_ = cmd.MarkFlagRequired("login")
	_ = cmd.MarkFlagRequired("password")

	return cmd
}
