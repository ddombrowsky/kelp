package main

import (
	"os"
	"testing"
)

// runs the binary's entry point end to end with a command that needs no config or network
func TestMainVersion(t *testing.T) {
	args := os.Args
	defer func() { os.Args = args }()

	os.Args = []string{"kelp", "version"}
	main()
}
