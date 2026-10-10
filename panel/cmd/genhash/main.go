// Command genhash prints a werkzeugpw hash for a plaintext password (dev tooling).
package main

import (
	"fmt"
	"os"

	"vietpanel/panel/internal/core/werkzeugpw"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: genhash <password>")
		os.Exit(1)
	}
	h, err := werkzeugpw.GeneratePasswordHash(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(h)
}
