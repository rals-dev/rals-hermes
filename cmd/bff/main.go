// Command bff is the backend-for-frontend of the Hermes monitoring dashboard.
//
// It fans out to the API server of every configured Hermes profile, aggregates
// the results, serves a read-only JSON API plus the embedded web UI, and never
// starts an agent run. See docs/prd.md for the full contract.
package main

import (
	"fmt"
	"os"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// M1 (T-101) replaces this stub with the real server bootstrap.
	fmt.Fprintf(os.Stderr, "hermes-bff %s: not implemented yet (see docs/prd.md § 7, M1)\n", version)
	os.Exit(2)
}
