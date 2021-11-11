package main

import (
	"github.com/peter-evans/kdef/cli/cmd"
)

var version = "dev"

func main() {
	cmd.Execute(version)
}
