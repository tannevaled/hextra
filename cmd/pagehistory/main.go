// Command pagehistory writes, as JSON, when and by whom every content page of
// a Hugo site was created, from git. It replaces scripts/page-history.sh.
package main

import (
	"os"
	"runtime/debug"
)

// osExit is replaceable by tests.
var osExit = os.Exit

func main() {
	root := RootCmd()
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		root.Version = bi.Main.Version
	}
	if err := root.Execute(); err != nil {
		osExit(1)
	}
}
