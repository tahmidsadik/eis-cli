package main

import (
	"os"

	"bitbucket.org/cover42/eiscli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
