package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const cronModulePrefix = "builtin:Cron::"

var (
	cronSchedulerClass       = ir.ClassID("builtin:Cron::class::Scheduler")
	cronErrorClass           = ir.ClassID("builtin:Cron::class::CronError")
	cronSchedulerHandleField = ir.FieldID("builtin:Cron::class::Scheduler::field::handle")
)

// cronCall lowers the Cron module's functions. A Scheduler is a hidden String
// handle, the representation SMTPClient uses, and Cron.next reuses the Time
// module's DateTime interchange so the result is an ordinary DateTime value.
func (generator *generator) cronCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), cronModulePrefix)
	errorClass := generator.descriptorName(cronErrorClass)
	switch name {
	case "scheduler":
		return generator.smtpValueFrom(cronSchedulerClass, "AhdCronNewScheduler()", meta)
	case "next":
		if len(value.Arguments) != 2 || value.Arguments[0].Value == nil || value.Arguments[1].Value == nil {
			return generator.unsupported("Cron.next with a malformed argument list", meta.Span)
		}
		expression := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
		return generator.dateTimeFrom("AhdCronNext("+errorClass+", "+expression+", "+
			generator.civilOf(value.Arguments[1].Value)+")", meta)
	default:
		return generator.unsupported("Cron function "+name, meta.Span)
	}
}

// cronOperation lowers the Scheduler members. The task argument is already a
// Go func() because its static type is exactly () -> Nothing; it runs on the
// goroutine that called run, so anything it raises propagates normally.
func (generator *generator) cronOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(cronErrorClass)
	handle := generator.smtpDataOf(cronSchedulerClass, cronSchedulerHandleField, value.Callee)
	switch name {
	case "Scheduler.add":
		if len(value.Arguments) != 2 || value.Arguments[0].Value == nil || value.Arguments[1].Value == nil {
			return generator.unsupported("Scheduler.add with a malformed argument list", meta.Span)
		}
		return "AhdCronSchedulerAdd(" + errorClass + ", " + handle + ", " +
			generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false) + ", " +
			generator.expr(value.Arguments[1].Value) + ")"
	case "Scheduler.run":
		return "AhdCronSchedulerRun(" + errorClass + ", " + handle + ", AhdFlush)"
	case "Scheduler.stop":
		return "AhdCronSchedulerStop(" + errorClass + ", " + handle + ")"
	default:
		return generator.unsupported("Cron operation "+name, meta.Span)
	}
}
