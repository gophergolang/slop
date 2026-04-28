// Command vibeguard-operator-manager is the entrypoint for the Kubernetes
// operator that reconciles Application.vibeguard.dev/v1 resources.
//
// Status: scaffold on branch 4-7. The reconciler logic and controller-runtime
// wiring land in the follow-up branch (see docs/ROADMAP.md).
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "vibeguard-operator: scaffold only on branch 4-7. See docs/ROADMAP.md for the reconciler implementation timeline.")
	os.Exit(1)
}
