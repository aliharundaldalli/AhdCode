// Command ahdcode is the AhdCode compiler driver.
//
//	ahdcode build path/to/program.ahd [-o output]
//	ahdcode run   path/to/program.ahd [-- program arguments]
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ahdcode/internal/build"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/formatter"
	"ahdcode/internal/repl"
	"ahdcode/internal/source"
)

const usage = `AhdCode v0.1.20 toolchain

usage:
  ahdcode                                  start the interactive REPL
  ahdcode build <entry.ahd> [-o <output>]   compile to a native executable
  ahdcode run   <entry.ahd> [-- <args>...]  compile and run
  ahdcode format [--check] <file.ahd>        canonicalize source in place
  ahdcode --help                             show this help
  ahdcode --version                          print the compiler version
`

const version = "AhdCode v0.1.20"

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
	case "format":
		return runFormat(arguments[1:], output, errorOutput)
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
	code, result := build.RunProgramIO(entry, programArguments, input, output, errorOutput)
	reportTo(errorOutput, result)
	return code
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
