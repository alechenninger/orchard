package main

import "github.com/alechenninger/orchard/cmd/orchard/cmd"

var version = "0.1.0-dev"

func main() {
	cmd.Execute(version)
}
