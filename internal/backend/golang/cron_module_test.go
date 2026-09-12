package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestCronRuntimeEmittedExactlyOnce(t *testing.T) {
	program := generate(t, "write(\"hi\")\n")
	seen := 0
	var content string
	for _, file := range program.Files {
		if file.Name == cronRuntimeFileName {
			seen++
			content = file.Content
		}
	}
	if seen != 1 {
		t.Fatalf("%s emitted %d times, want exactly 1", cronRuntimeFileName, seen)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), cronRuntimeFileName, content, 0)
	if err != nil {
		t.Fatalf("generated Cron runtime is not valid Go: %v", err)
	}
	if parsed.Name.Name != "main" {
		t.Fatalf("the Cron runtime package clause was not rewritten: package %s", parsed.Name.Name)
	}
	// Cron is in-process scheduling only. Its imports are exactly these four
	// standard packages: no process execution, network, signal handling, or
	// third-party dependency can hide behind a comment.
	var imports []string
	for _, spec := range parsed.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		imports = append(imports, path)
	}
	sort.Strings(imports)
	if strings.Join(imports, ",") != "strconv,strings,sync,time" {
		t.Fatalf("Cron runtime imports %v, want exactly strconv, strings, sync, time", imports)
	}
	// The scheduler loop starts no goroutine of its own.
	ast.Inspect(parsed, func(node ast.Node) bool {
		if _, ok := node.(*ast.GoStmt); ok {
			t.Fatal("Cron runtime must not start a goroutine")
		}
		return true
	})
}

func TestProgramWithoutCronDoesNotReferenceIt(t *testing.T) {
	source := programSource(t, generate(t, "write(\"hi\")\n"))
	for _, symbol := range []string{"AhdCronSchedulerRun", "AhdCronNext", "cd_CronError"} {
		if strings.Contains(source, symbol) {
			t.Fatalf("program that does not use Cron references %s", symbol)
		}
	}
}

func TestCronLoweringEmitsRuntimeCalls(t *testing.T) {
	source := programSource(t, generate(t, `bring Cron
bring Time
from Cron bring Scheduler

scheduler: Scheduler := Cron.scheduler()
once: Function := () -> Nothing {
    scheduler: Global Scheduler
    write("tick")
    scheduler.stop()
}
scheduler.add("* * * * *", once)
write(Cron.next("0 9 * * 1-5", Time.dateTimeUTC(2026, 9, 12)).toString())
scheduler.run()
`))
	for _, helper := range []string{"AhdCronNewScheduler()", "AhdCronSchedulerAdd(", "AhdCronSchedulerRun(", "AhdCronSchedulerStop(", "AhdCronNext("} {
		if !strings.Contains(source, helper) {
			t.Fatalf("generated program does not call %s", helper)
		}
	}
}
