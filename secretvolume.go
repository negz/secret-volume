package main

import "github.com/negz/secret-volume/cmd"

func main() {
	// This is a workaround to more easily use Go build tags in the 'real' main
	// in the cmd package.
	cmd.Run()
}
