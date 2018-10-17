package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tzneal/cpdocs2impl"
	"golang.org/x/tools/go/packages"
)

func main() {
	replace := flag.Bool("replace", false, "replace any existing documentation")
	imports := flag.Bool("imports", false, "consider interfaces from imports as well (slower)")

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTION]... [PACKAGE]...\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Copy docs from interfaces to implementations")
		flag.PrintDefaults()
		os.Exit(1)
	}

	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading packagses: %v\n", err)
		os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	c := cpdocs2impl.NewCommentCollector(*imports, *replace)
	c.Process(pkgs)
}
