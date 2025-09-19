package main

import "github.com/alechenninger/orchard/internal/cli"

var version = "0.1.0-dev"

func main() {
	cli.Execute(version)
}
