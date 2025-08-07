package main

import (
	"os"

	"github.com/bbq191/dotfiles-go/cmd/dotfiles/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}