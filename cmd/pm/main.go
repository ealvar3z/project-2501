package main

import (
	"fmt"
	"os"

	"project2501/internal/pmapp"
)

func main() {
	if err := pmapp.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "pm: %v\n", err)
		os.Exit(1)
	}
}
