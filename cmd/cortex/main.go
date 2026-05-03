package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("cortex", version)
		os.Exit(0)
	}
	fmt.Println("cortex — coming soon")
}