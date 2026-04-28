package main

import "fmt"

// Version is set at build time via -ldflags "-X main.Version=v0.7.0".
var Version = "v0.7.0-4-7-dev"

func runVersion(_ []string) {
	fmt.Printf("vibeguard %s\n", Version)
}
