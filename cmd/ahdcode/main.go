// Command ahdcode is the AhdCode compiler driver.
//
//	ahdcode build path/to/program.ahd [-o output]
//	ahdcode run   path/to/program.ahd [-- program arguments]
package main

import (
	"fmt"
	"os"

	"ahdcode/internal/build"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
)

const usage = `AhdCode compiler

usage:
  ahdcode build <entry.ahd> [-o <output>]   compile to a native executable
  ahdcode run   <entry.ahd> [-- <args>...]  compile and run
  ahdcode version                           print the compiler version
`

const version = "AhdCode v0.1 (milestone G)"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(arguments []string) int {
	if len(arguments) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	switch arguments[0] {
	case "build":
		return runBuild(arguments[1:])
	case "run":
		return runRun(arguments[1:])
	case "version", "--version", "-v":
		fmt.Fprintln(os.Stdout, version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(os.Stdout, usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", arguments[0], usage)
		return 2
	}
}

func runBuild(arguments []string) int {
	entry, output := "", ""
	for index := 0; index < len(arguments); index++ {
		switch arguments[index] {
		case "-o", "--output":
			if index+1 >= len(arguments) {
				fmt.Fprintln(os.Stderr, "ahdcode build: -o requires an output path")
				return 2
			}
			index++
			output = arguments[index]
		default:
			if entry != "" {
				fmt.Fprintln(os.Stderr, "ahdcode build: exactly one entry module is expected")
				return 2
			}
			entry = arguments[index]
		}
	}
	if entry == "" {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	path, result := build.BuildProgram(entry, output)
	report(result)
	if result.HasErrors() {
		return 1
	}
	fmt.Fprintln(os.Stdout, path)
	return 0
}

func runRun(arguments []string) int {
	if len(arguments) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	entry := arguments[0]
	programArguments := arguments[1:]
	if len(programArguments) > 0 && programArguments[0] == "--" {
		programArguments = programArguments[1:]
	}
	code, result := build.RunProgram(entry, programArguments, os.Stdin, os.Stdout, os.Stderr)
	report(result)
	return code
}

func report(result build.Result) {
	for _, item := range result.Diagnostics {
		fmt.Fprintln(os.Stderr, format(item, result.Files))
	}
}

func format(item diagnostics.Diagnostic, files map[source.FileID]source.File) string {
	severity := "error"
	if item.Severity == diagnostics.SeverityWarning {
		severity = "warning"
	}
	location := ""
	if file, known := files[item.Span.FileID]; known && item.Span.FileID != 0 {
		location = fmt.Sprintf("%s:%d:%d: ", file.Path, item.Span.Start.Line, item.Span.Start.Column)
	}
	message := fmt.Sprintf("%s%s [%s] %s", location, severity, item.Code, item.Message)
	if item.Hint != "" {
		message += "\n  hint: " + item.Hint
	}
	return message
}
