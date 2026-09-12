package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The Cron standard module's REPL implementation. It calls the native
// runtime's ahdruntime/cron.go directly, so the evaluator and a compiled
// program share one parser, one occurrence search, and one scheduler loop.

const evaluatorCronSchedulerClass = ir.ClassID("builtin:Cron::class::Scheduler")

var evaluatorCronSchedulerField = ir.FieldID("builtin:Cron::class::Scheduler::field::handle")

// cronTaskFailure carries whatever a task raised through the runtime loop, so
// cronRecover can re-raise it unchanged instead of mistaking it for a
// scheduler failure.
type cronTaskFailure struct{ value any }

func (session *Session) cronBuiltin(name string, args []any) any {
	defer session.cronRecover()
	class := ahdruntime.AhdClassCronError
	switch name {
	case "scheduler":
		return &Instance{Class: evaluatorCronSchedulerClass, Fields: map[ir.FieldID]any{
			evaluatorCronSchedulerField: ahdruntime.AhdCronNewScheduler(),
		}}
	case "next":
		return session.dateTime(ahdruntime.AhdCronNextInstant(class, args[0].(string), session.instant(args[1])))
	}
	session.raise("Error", "unsupported Cron function "+name)
	return nil
}

func (session *Session) cronOperation(name string, receiver any, args []any) any {
	defer session.cronRecover()
	class := ahdruntime.AhdClassCronError
	instance := session.requireInstance(receiver)
	handle, ok := instance.Fields[evaluatorCronSchedulerField].(string)
	if !ok || instance.Class != evaluatorCronSchedulerClass {
		session.raise("CronError", "Scheduler storage is corrupted")
	}
	switch name {
	case "Scheduler.add":
		function, ok := args[1].(*FunctionValue)
		if !ok || function == nil {
			session.raise("CronError", "Scheduler.add: the task Function is missing")
		}
		ahdruntime.AhdCronSchedulerAdd(class, handle, args[0].(string), func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					panic(cronTaskFailure{value: recovered})
				}
			}()
			session.invoke(function, nil)
		})
		return Nothing
	case "Scheduler.run":
		// Output a task writes must be visible while the session waits.
		ahdruntime.AhdCronSchedulerRun(class, handle, func() {
			if flusher, ok := session.Output.(interface{ Flush() error }); ok {
				_ = flusher.Flush()
			}
		})
		return Nothing
	case "Scheduler.stop":
		ahdruntime.AhdCronSchedulerStop(class, handle)
		return Nothing
	}
	session.raise("Error", "unsupported Cron operation "+name)
	return nil
}

// cronRecover re-raises a task's own failure unchanged and turns a runtime
// scheduler signal into the evaluator's catchable CronError.
func (session *Session) cronRecover() {
	recovered := recover()
	if recovered == nil {
		return
	}
	if failure, ok := recovered.(cronTaskFailure); ok {
		panic(failure.value)
	}
	if signal, ok := recovered.(*ahdruntime.AhdSignal); ok {
		session.raise("CronError", signal.Message)
	}
	panic(recovered)
}
