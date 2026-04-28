// Package main является точкой входа CLI-клиента Trantor.
package main

import (
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Gustik/trantor/cmd/client/tui"
)

// Заполняются через ldflags при сборке:
//
//	-X main.version=v1.0.0 -X main.buildDate=2026-04-23
var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("trantor %s (built %s)\n", version, buildDate)
		return
	}
	if err := tui.Start(version, buildDate); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
