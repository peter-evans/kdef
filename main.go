package main

import (
	"github.com/peter-evans/kdef/cmd"
)

var version = "dev"

func main() {
	cmd.Execute(version)
}
