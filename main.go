package main

import (
	_ "embed"
	"os"

	"github.com/milhamdedi/notifuse-cli/internal/cmd"
)

//go:embed schemas/notifuse.openapi.json
var openAPISpec []byte

func main() {
	os.Exit(cmd.Execute(openAPISpec, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
