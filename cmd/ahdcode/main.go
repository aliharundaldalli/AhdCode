// Command ahdcode is the AhdCode compiler driver.
//
//	ahdcode build path/to/program.ahd [-o output]
//	ahdcode run   path/to/program.ahd [-- program arguments]
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ahdcode/internal/build"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/formatter"
	"ahdcode/internal/lsp"
	"ahdcode/internal/repl"
	"ahdcode/internal/source"
)

const usage = `AhdCode v0.10.0 toolchain

usage:
  ahdcode                                  start the interactive REPL
  ahdcode build <entry.ahd> [-o <output>]   compile to a native executable
  ahdcode run   <entry.ahd> [-- <args>...]  compile and run
  ahdcode kill  [--force] <app.run>          stop an application started by run
  ahdcode format [--check] <file.ahd>        canonicalize source in place
  ahdcode lsp                                start the language server (stdio)
  ahdcode --help                             show this help
  ahdcode --version                          print the compiler version
`

const version = "AhdCode v0.10.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(arguments []string) int {
	return runWithIO(arguments, os.Stdin, os.Stdout, os.Stderr)
}

func runWithIO(arguments []string, input io.Reader, output, errorOutput io.Writer) int {
	if len(arguments) == 0 {
		return repl.Run(input, output, errorOutput, version)
	}
	switch arguments[0] {
	case "build":
		return runBuild(arguments[1:], output, errorOutput)
	case "run":
		return runRun(arguments[1:], input, output, errorOutput)
	case "kill":
		return runKill(arguments[1:], output, errorOutput)
	case "format":
		return runFormat(arguments[1:], output, errorOutput)
	case "lsp":
		return runLSP(arguments[1:], input, output, errorOutput)
	case "version", "--version", "-v":
		if len(arguments) != 1 {
			fmt.Fprintln(errorOutput, "ahdcode --version: no arguments are accepted")
			return 2
		}
		fmt.Fprintln(output, version)
		return 0
	case "help", "--help", "-h":
		if len(arguments) != 1 {
			fmt.Fprintln(errorOutput, "ahdcode --help: no arguments are accepted")
			return 2
		}
		fmt.Fprint(output, usage)
		return 0
	default:
		fmt.Fprintf(errorOutput, "unknown command %q\n\n%s", arguments[0], usage)
		return 2
	}
}

func runBuild(arguments []string, outputWriter, errorOutput io.Writer) int {
	entry, output := "", ""
	for index := 0; index < len(arguments); index++ {
		switch arguments[index] {
		case "-o", "--output":
			if index+1 >= len(arguments) {
				fmt.Fprintln(errorOutput, "ahdcode build: -o requires an output path")
				return 2
			}
			index++
			output = arguments[index]
		default:
			if strings.HasPrefix(arguments[index], "-") {
				fmt.Fprintf(errorOutput, "ahdcode build: unknown flag %q\n", arguments[index])
				return 2
			}
			if entry != "" {
				fmt.Fprintln(errorOutput, "ahdcode build: exactly one entry module is expected")
				return 2
			}
			entry = arguments[index]
		}
	}
	if entry == "" {
		fmt.Fprint(errorOutput, usage)
		return 2
	}
	path, result := build.BuildProgram(entry, output)
	reportTo(errorOutput, result)
	if result.HasErrors() {
		return 1
	}
	fmt.Fprintln(outputWriter, path)
	return 0
}

func runRun(arguments []string, input io.Reader, output, errorOutput io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprint(errorOutput, usage)
		return 2
	}
	if strings.HasPrefix(arguments[0], "-") {
		fmt.Fprintf(errorOutput, "ahdcode run: unknown flag %q\n", arguments[0])
		return 2
	}
	entry := arguments[0]
	programArguments := arguments[1:]
	if len(programArguments) > 0 && programArguments[0] == "--" {
		programArguments = programArguments[1:]
	}

	// While this application runs, a sibling app.run descriptor names it so
	// `ahdcode kill app.run` can stop it later. The descriptor points at this
	// process's loopback control channel rather than granting authority over a
	// process id; see runcontrol.go.
	runFile := runFileFor(entry)
	if err := claimRunFile(runFile); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode run: %v\n", err)
		return 1
	}
	control, err := startRunControlServer()
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode run: could not start the run control channel: %v\n", err)
		return 1
	}
	defer control.close()
	defer removeOwnRunDescriptor(runFile, control.port())

	stopSignals := make(chan os.Signal, 1)
	signal.Notify(stopSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopSignals)
	go func() {
		for range stopSignals {
			// Ctrl-C would otherwise end this process without cleanup; the
			// child receives the same terminal signal and exits on its own.
			removeOwnRunDescriptor(runFile, control.port())
		}
	}()

	code, result := build.RunProgramObserved(entry, programArguments, input, output, errorOutput, func(process *os.Process) {
		// The supervisor owns this process; it is the only thing that will
		// ever terminate it.
		control.attach(process)
		// The descriptor is written only once the control channel is ready,
		// so a descriptor never names an endpoint that cannot answer.
		if err := startRunDescriptor(runFile, entry, process.Pid, control.port(), control.token); err != nil {
			fmt.Fprintf(errorOutput, "ahdcode run: could not write the run file: %v\n", err)
			return
		}
		control.ownDescriptor(runFile)
	})
	reportTo(errorOutput, result)
	return code
}

// runKill stops the application named by one AhdCode run descriptor.
//
// It never signals the pid recorded in the file. A descriptor only locates the
// live supervisor: kill authenticates to that supervisor's loopback control
// channel, and the supervisor terminates the child it actually owns. A forged
// descriptor naming an unrelated process therefore stops nothing, and a
// recycled pid is harmless.
func runKill(arguments []string, output, errorOutput io.Writer) int {
	force := false
	target := ""
	for _, argument := range arguments {
		switch {
		case argument == "--force":
			force = true
		case strings.HasPrefix(argument, "-"):
			fmt.Fprintf(errorOutput, "ahdcode kill: unknown flag %q\n", argument)
			return 2
		case target == "":
			target = argument
		default:
			fmt.Fprintln(errorOutput, "ahdcode kill: exactly one run file is expected")
			return 2
		}
	}
	if target == "" {
		fmt.Fprintln(errorOutput, "ahdcode kill: a run file is required, as in: ahdcode kill app.run")
		return 2
	}

	descriptor, err := readRunDescriptor(target)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(errorOutput, "ahdcode kill: no run file at %s\n", target)
			return 1
		}
		fmt.Fprintf(errorOutput, "ahdcode kill: %s is not a usable AhdCode run file (%v); no process was stopped\n", target, err)
		return 1
	}

	action := runControlStop
	if force {
		action = runControlForceStop
	}
	if err := requestRunControl(descriptor.ControlPort, descriptor.ControlToken, action); err != nil {
		if errors.Is(err, errRunControlUnreachable) {
			// Nothing answered for this descriptor, so there is no AhdCode
			// application here to stop. The recorded pid is deliberately left
			// alone: it may belong to something else entirely.
			if removeErr := os.Remove(target); removeErr != nil {
				fmt.Fprintf(errorOutput, "ahdcode kill: the application is not running, but the stale run file could not be removed: %v\n", removeErr)
				return 1
			}
			fmt.Fprintf(output, "AhdCode application is not running; removed the stale run file %s (no process was signalled)\n", target)
			return 0
		}
		fmt.Fprintf(errorOutput, "ahdcode kill: %v; no process was stopped\n", err)
		return 1
	}

	// The supervisor stopped its child and removes its own descriptor as it
	// exits; wait briefly for that, then clean up the file we authenticated
	// against if the supervisor did not get to it.
	for attempt := 0; attempt < 50; attempt++ {
		if _, statErr := os.Stat(target); os.IsNotExist(statErr) {
			fmt.Fprintf(output, "Stopped the AhdCode application (pid %d) and removed %s\n", descriptor.PID, target)
			return 0
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(errorOutput, "ahdcode kill: stopped the application but could not remove %s: %v\n", target, err)
		return 1
	}
	fmt.Fprintf(output, "Stopped the AhdCode application (pid %d) and removed %s\n", descriptor.PID, target)
	return 0
}

// runLSP starts the AhdCode language server over stdio. It contains no
// language-server semantic implementation of its own -- internal/lsp owns
// the protocol, and internal/analysis owns the compiler-backed diagnostics
// and hover facts it serves; this function only wires them to the process's
// standard streams. output must never receive anything except LSP
// Content-Length-framed protocol bytes, so unlike every other subcommand
// here, nothing is ever written to it directly from this function -- not a
// version banner, not a usage message, nothing.
//
// "--stdio" is accepted and ignored: it is not an AhdCode-invented flag but
// the argument real LSP client libraries (e.g. vscode-languageclient's Node
// transport) unconditionally append when they launch a server configured
// for stdio transport, to select that transport among several a server
// might support. ahdcode lsp only ever speaks stdio, so the flag is a no-op
// here, but rejecting it -- as an arbitrary unrecognized argument would be
// -- breaks every real editor integration outright. Anything else is still
// rejected clearly.
func runLSP(arguments []string, input io.Reader, output, errorOutput io.Writer) int {
	for _, argument := range arguments {
		if argument != "--stdio" {
			fmt.Fprintf(errorOutput, "ahdcode lsp: unexpected argument %q\n", argument)
			return 2
		}
	}
	server := lsp.NewServer(errorOutput)
	if err := server.Run(input, output); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode lsp: %v\n", err)
		return 1
	}
	return 0
}

func report(result build.Result) {
	reportTo(os.Stderr, result)
}

func reportTo(writer io.Writer, result build.Result) {
	for _, item := range result.Diagnostics {
		fmt.Fprintln(writer, format(item, result.Files))
	}
}

func format(item diagnostics.Diagnostic, files map[source.FileID]source.File) string {
	return diagnostics.Render(item, files)
}

func runFormat(arguments []string, output, errorOutput io.Writer) int {
	check := false
	path := ""
	for _, argument := range arguments {
		switch argument {
		case "--check":
			check = true
		default:
			if strings.HasPrefix(argument, "-") {
				fmt.Fprintf(errorOutput, "ahdcode format: unknown flag %q\n", argument)
				return 2
			}
			if path != "" {
				fmt.Fprintln(errorOutput, "ahdcode format: exactly one source file is expected")
				return 2
			}
			path = argument
		}
	}
	if path == "" {
		fmt.Fprintln(errorOutput, "ahdcode format: one source file is required")
		return 2
	}
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode format: could not read %s: %v\n", path, err)
		return 1
	}
	file := source.NewFile(1, path, string(content))
	formatted := formatter.Format(file)
	if formatted.HasErrors() {
		for _, item := range formatted.Diagnostics {
			fmt.Fprintln(errorOutput, diagnostics.Render(item, map[source.FileID]source.File{1: file}))
		}
		return 1
	}
	if formatted.Text == string(content) {
		return 0
	}
	if check {
		fmt.Fprintf(errorOutput, "%s is not canonically formatted\n", path)
		return 1
	}
	if err := writeAtomic(path, []byte(formatted.Text)); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode format: could not update %s: %v\n", path, err)
		return 1
	}
	return 0
}

func writeAtomic(path string, content []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".ahdcode-format-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := func() { _ = os.Remove(temporaryPath) }
	defer cleanup()
	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
