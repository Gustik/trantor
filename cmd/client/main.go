// Package main является точкой входа CLI-клиента Trantor.
package main

import (
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Gustik/trantor/cmd/client/commands"
)

func main() {
	if err := commands.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
