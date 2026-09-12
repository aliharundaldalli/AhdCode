package semantic

import "testing"

const cronPreamble = "bring Cron\nbring Time\nfrom Cron bring (Scheduler, CronError)\nfrom Time bring DateTime\n\n"

// cronFixtures declares a Scheduler and task Functions of several shapes. AhdCode
// passes Functions by name; a block Function literal is not an argument
// expression.
const cronFixtures = `scheduler: Scheduler := Cron.scheduler()
tick: Function := () -> Nothing {
    write("tick")
}
withParameter: Function := (value: Int) -> Nothing {
    write(value)
}
withResult: Function := () -> Int {
    return 1
}
`

func TestCronModuleRegistered(t *testing.T) {
	module, ok := StandardModuleInterfaces()["Cron"]
	if !ok {
		t.Fatal("Cron module not found in StandardModuleInterfaces")
	}
	if module.ModuleID != "builtin:Cron" {
		t.Fatalf("Cron canonical identity = %q, want builtin:Cron", module.ModuleID)
	}
	exports := []string{"scheduler", "next", "Scheduler", "CronError"}
	for _, name := range exports {
		if module.Exports[name] == nil {
			t.Fatalf("Cron module missing export %q", name)
		}
	}
	if len(module.Exports) != len(exports) {
		t.Fatalf("Cron exports %v, want exactly %v", module.ExportNames, exports)
	}
}

func TestCronModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, cronPreamble+cronFixtures+`morning: Function := () -> Nothing {
    write("morning")
}
scheduler.add("*/5 * * * *", tick)
scheduler.add("0 9 * * 1-5", morning)
upcoming: DateTime := Cron.next("0 9 * * 1-5", Time.utc())
write(upcoming.toString())
scheduler.stop()
attempt {
    scheduler.run()
} except CronError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

func TestCronModuleAliasAndDirectImport(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring Cron as C
bring Time
from Cron bring (scheduler, next)
jobs := C.scheduler()
other := scheduler()
write(next("0 0 * * *", Time.utc()).toString())
`)
	requireSemanticClean(t, result)
}

// A task may stop its own Scheduler through an explicit Global dependency.
func TestCronTaskCanStopSchedulerThroughGlobal(t *testing.T) {
	result := analyzeWithStandardModules(t, cronPreamble+`scheduler: Scheduler := Cron.scheduler()
once: Function := () -> Nothing {
    scheduler: Global Scheduler
    scheduler.stop()
}
scheduler.add("* * * * *", once)
`)
	requireSemanticClean(t, result)
}

// A schedule String is validated when it is used, not at compile time: an
// invalid literal compiles and raises CronError at run time.
func TestCronScheduleTextIsRuntimeValidated(t *testing.T) {
	result := analyzeWithStandardModules(t, cronPreamble+cronFixtures+`scheduler.add("not a schedule", tick)
write(Cron.next("61 * * * *", Time.utc()).toString())
`)
	requireSemanticClean(t, result)
}

func TestCronRejectsWrongStaticShapes(t *testing.T) {
	tests := []struct {
		source string
		code   string
	}{
		{`scheduler.add(5, tick)`, codeTypeMismatch},
		{`scheduler.add("* * * * *", "tick")`, codeTypeMismatch},
		{`scheduler.add("* * * * *", 1)`, codeTypeMismatch},
		{`scheduler.add("* * * * *", withParameter)`, codeTypeMismatch},
		{`scheduler.add("* * * * *", withResult)`, codeTypeMismatch},
		{`scheduler.add("* * * * *")`, codeCallArguments},
		{`scheduler.add("* * * * *", tick, tick)`, codeCallArguments},
		{`scheduler.run(1)`, codeCallArguments},
		{`scheduler.stop(true)`, codeCallArguments},
		{`Cron.next("* * * * *", "2026-09-12")`, codeTypeMismatch},
		{`Cron.next(5, Time.now())`, codeTypeMismatch},
		{`other: Scheduler := Scheduler("cron-scheduler-1")`, codeCallArguments},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, cronPreamble+cronFixtures+test.source+"\n")
			requireSemanticFailure(t, result)
			if result.Diagnostics[0].Code != test.code {
				t.Fatalf("%s: first diagnostic %s %q, want %s", test.source,
					result.Diagnostics[0].Code, result.Diagnostics[0].Message, test.code)
			}
			if len(result.Diagnostics) != 1 {
				t.Fatalf("%s: expected one diagnostic without a cascade, got %+v", test.source, result.Diagnostics)
			}
		})
	}
}

func TestCronOperationsReturnNothing(t *testing.T) {
	for _, source := range []string{
		`value: Int := scheduler.run()`,
		`value: String := scheduler.stop()`,
		`value: Bool := scheduler.add("* * * * *", tick)`,
		`value: String := Cron.next("* * * * *", Time.now())`,
	} {
		t.Run(source, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, cronPreamble+cronFixtures+source+"\n"))
		})
	}
}
