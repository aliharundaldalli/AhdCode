package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"ahdcode/internal/initweb"
)

func runPWA(args []string, output, diagnostics io.Writer) int {
	const pwaUsage = "Usage: ahdcode pwa init [--name NAME] [--short-name NAME]"
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(output, pwaUsage)
		return 0
	}
	if len(args) == 0 || args[0] != "init" {
		fmt.Fprintln(diagnostics, pwaUsage)
		return 2
	}
	flags := flag.NewFlagSet("pwa init", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	flags.Usage = func() { fmt.Fprintln(diagnostics, pwaUsage) }
	name := flags.String("name", "", "manifest name (defaults to directory name)")
	short := flags.String("short-name", "", "manifest short name (defaults to name)")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return 0
	} else if err != nil || flags.NArg() != 0 {
		flags.Usage()
		return 2
	}
	root, err := os.Getwd()
	if err == nil {
		err = initweb.PWA(root, *name, *short)
	}
	if err != nil {
		fmt.Fprintf(diagnostics, "ahdcode pwa init: %v\n", err)
		return 1
	}
	fmt.Fprintln(output, `Created PWA/ with a manifest, worker, and public assets.
Add require("PWA/Integration.ahd") to the entry and shared layout.
Call ahdPwaMount(site) before site.start().
Add ahdPwaManifest() and ahdPwaRegistration() as nodes in the layout head.`)
	return 0
}
