// Package main является точкой входа CLI-клиента Trantor.
package main

import (
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Gustik/trantor/cmd/client/commands"
)

// Заполняются через ldflags при сборке:
//
//	-X main.version=v1.0.0 -X main.buildDate=2026-04-23
var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	if err := commands.New(version, buildDate).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
