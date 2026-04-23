package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию клиента",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("trantor %s (built %s)\n", d.version, d.buildDate)
		},
	}
}
