// Command publishversion publishes one documentation build into a checkout of
// the gh-pages branch, one directory per version, and keeps the files a
// version selector needs. It replaces the shell script the documentation
// workflows used to carry.
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
