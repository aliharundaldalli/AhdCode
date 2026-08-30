package evaluator

import (
	"fmt"
	"time"

	"ahdcode/internal/ir"
)

var evaluatorMonotonicOrigin = time.Now()

func (session *Session) evalTime(name string, arguments []any) any {
	switch name {
	case "now":
		return session.dateTime(time.Now())
	case "monotonic":
		return time.Since(evaluatorMonotonicOrigin).Seconds()
	case "sleep":
		milliseconds := arguments[0].(int64)
		if milliseconds < 0 {
			session.raise("DomainError", "sleep requires non-negative milliseconds")
		}
		time.Sleep(time.Duration(milliseconds) * time.Millisecond)
		return Nothing
	case "duration":
		return session.duration(arguments[0].(int64))
	case "between":
		first, second := session.instant(arguments[0]), session.instant(arguments[1])
		return session.duration(int64(second.Sub(first) / time.Millisecond))
	case "dateTime":
		parts := make([]int64, 7)
		for index := range parts {
			if index < len(arguments) && arguments[index] != nil {
				parts[index] = arguments[index].(int64)
			}
		}
		if parts[0] < 1 || parts[0] > 9999 || parts[1] < 1 || parts[1] > 12 ||
			parts[3] < 0 || parts[3] > 23 || parts[4] < 0 || parts[4] > 59 ||
			parts[5] < 0 || parts[5] > 59 || parts[6] < 0 || parts[6] > 999 {
			session.raise("ValueError", "dateTime received invalid civil components")
		}
		instant := time.Date(int(parts[0]), time.Month(parts[1]), int(parts[2]), int(parts[3]), int(parts[4]), int(parts[5]), int(parts[6])*1e6, time.Local)
		if int64(instant.Year()) != parts[0] || int64(instant.Month()) != parts[1] || int64(instant.Day()) != parts[2] {
			session.raise("ValueError", "dateTime received invalid civil components")
		}
		return session.dateTime(instant)
	}
	session.raise("Error", "unsupported Time operation "+name)
	return nil
}

func (session *Session) dateTime(value time.Time) *Instance {
	class := ir.ClassID("builtin:Time::class::DateTime")
	instance := &Instance{Class: class, Fields: make(map[ir.FieldID]any)}
	weekday := int64((int(value.Weekday())+6)%7 + 1)
	values := []int64{int64(value.Year()), int64(value.Month()), int64(value.Day()), int64(value.Hour()), int64(value.Minute()), int64(value.Second()), int64(value.Nanosecond() / 1e6), weekday}
	names := []string{"year", "month", "day", "hour", "minute", "second", "millisecond", "weekday"}
	for index, name := range names {
		instance.Fields[ir.FieldID(string(class)+"::field::"+name)] = values[index]
	}
	return instance
}

func (session *Session) duration(milliseconds int64) *Instance {
	class := ir.ClassID("builtin:Time::class::Duration")
	return &Instance{Class: class, Fields: map[ir.FieldID]any{
		ir.FieldID(string(class) + "::field::milliseconds"): milliseconds,
		ir.FieldID(string(class) + "::field::seconds"):      float64(milliseconds) / 1000,
	}}
}

func (session *Session) instant(value any) time.Time {
	instance := session.requireInstance(value)
	get := func(name string) int64 {
		return instance.Fields[ir.FieldID("builtin:Time::class::DateTime::field::"+name)].(int64)
	}
	return time.Date(int(get("year")), time.Month(get("month")), int(get("day")), int(get("hour")), int(get("minute")), int(get("second")), int(get("millisecond"))*1e6, time.Local)
}

func (session *Session) timeOperation(name string, receiver any, arguments []any) any {
	switch name {
	case "DateTime.before":
		return session.instant(receiver).Before(session.instant(arguments[0]))
	case "DateTime.after":
		return session.instant(receiver).After(session.instant(arguments[0]))
	case "DateTime.sameMoment":
		return session.instant(receiver).Equal(session.instant(arguments[0]))
	case "DateTime.toString":
		return session.instant(receiver).Format("2006-01-02 15:04:05")
	case "Calendar.isLeapYear":
		year := arguments[0].(int64)
		return year%400 == 0 || year%4 == 0 && year%100 != 0
	case "Calendar.daysInMonth":
		year, month := arguments[0].(int64), arguments[1].(int64)
		if month < 1 || month > 12 {
			session.raise("ValueError", "month must be between 1 and 12")
		}
		return int64(time.Date(int(year), time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day())
	case "Calendar.weekday":
		year, month, day := arguments[0].(int64), arguments[1].(int64), arguments[2].(int64)
		value := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.Local)
		if int64(value.Year()) != year || int64(value.Month()) != month || int64(value.Day()) != day {
			session.raise("ValueError", "weekday received an invalid date")
		}
		return int64((int(value.Weekday())+6)%7 + 1)
	}
	session.raise("Error", fmt.Sprintf("unsupported Time operation %s", name))
	return nil
}
