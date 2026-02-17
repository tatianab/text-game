package main

import (
	"fmt"
	"os"

	"github.com/tatianab/text-game/internal/tui"
)

func main() {
	if err := tui.Start(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
