package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

const (
	evaluatorSMTPClientClass  = ir.ClassID("builtin:SMTP::class::SMTPClient")
	evaluatorSMTPMessageClass = ir.ClassID("builtin:SMTP::class::SMTPMessage")
)

var (
	evaluatorSMTPClientField  = ir.FieldID("builtin:SMTP::class::SMTPClient::field::handle")
	evaluatorSMTPMessageField = ir.FieldID("builtin:SMTP::class::SMTPMessage::field::data")
)

func (session *Session) smtpClient(handle string) *Instance {
	return &Instance{Class: evaluatorSMTPClientClass, Fields: map[ir.FieldID]any{evaluatorSMTPClientField: handle}}
}

func (session *Session) smtpMessage(data string) *Instance {
	return &Instance{Class: evaluatorSMTPMessageClass, Fields: map[ir.FieldID]any{evaluatorSMTPMessageField: data}}
}

func (session *Session) smtpDataOf(value any, class ir.ClassID, field ir.FieldID, name string) string {
	instance := session.requireInstance(value)
	data, ok := instance.Fields[field].(string)
	if !ok || instance.Class != class {
		session.raise("SMTPError", name+" storage is corrupted")
	}
	return data
}

func (session *Session) smtpRecover() {
	recovered := recover()
	if recovered == nil {
		return
	}
	if signal, ok := recovered.(*ahdruntime.AhdSignal); ok {
		session.raise("SMTPError", signal.Message)
	}
	panic(recovered)
}

func (session *Session) smtpStringArg(args []any, index int, fallback string) string {
	if index >= len(args) || args[index] == nil {
		return fallback
	}
	return args[index].(string)
}

func (session *Session) smtpIntArg(args []any, index int, fallback int64) int64 {
	if index >= len(args) || args[index] == nil {
		return fallback
	}
	return args[index].(int64)
}

func (session *Session) smtpStringList(value any) []string {
	if value == nil {
		return nil
	}
	list := session.requireList(value)
	result := make([]string, len(list.Items))
	for index, item := range list.Items {
		result[index] = item.(string)
	}
	return result
}

func (session *Session) smtpBuiltin(name string, args []any) any {
	defer session.smtpRecover()
	class := ahdruntime.AhdClassSMTPError
	switch name {
	case "client":
		return session.smtpClient(ahdruntime.AhdSMTPClient(class,
			args[0].(string), args[1].(int64),
			session.smtpStringArg(args, 2, "starttls"),
			session.smtpIntArg(args, 3, 30)))
	case "message":
		return session.smtpMessage(ahdruntime.AhdSMTPMessage(class,
			args[0].(string), session.smtpStringList(args[1]), args[2].(string)))
	}
	session.raise("Error", "unsupported SMTP function "+name)
	return nil
}

func (session *Session) smtpOperation(name string, receiver any, args []any) any {
	defer session.smtpRecover()
	class := ahdruntime.AhdClassSMTPError
	switch name {
	case "SMTPClient.withPlainAuth":
		return session.smtpClient(ahdruntime.AhdSMTPClientWithPlainAuth(class,
			session.smtpDataOf(receiver, evaluatorSMTPClientClass, evaluatorSMTPClientField, "SMTPClient"),
			args[0].(string), args[1].(string)))
	case "SMTPClient.send":
		ahdruntime.AhdSMTPClientSend(class,
			session.smtpDataOf(receiver, evaluatorSMTPClientClass, evaluatorSMTPClientField, "SMTPClient"),
			session.smtpDataOf(args[0], evaluatorSMTPMessageClass, evaluatorSMTPMessageField, "SMTPMessage"))
		return Nothing
	case "SMTPMessage.withCc":
		return session.smtpMessage(ahdruntime.AhdSMTPMessageWithCc(class,
			session.smtpDataOf(receiver, evaluatorSMTPMessageClass, evaluatorSMTPMessageField, "SMTPMessage"),
			session.smtpStringList(args[0])))
	case "SMTPMessage.withBcc":
		return session.smtpMessage(ahdruntime.AhdSMTPMessageWithBcc(class,
			session.smtpDataOf(receiver, evaluatorSMTPMessageClass, evaluatorSMTPMessageField, "SMTPMessage"),
			session.smtpStringList(args[0])))
	case "SMTPMessage.withReplyTo":
		return session.smtpMessage(ahdruntime.AhdSMTPMessageWithReplyTo(class,
			session.smtpDataOf(receiver, evaluatorSMTPMessageClass, evaluatorSMTPMessageField, "SMTPMessage"),
			args[0].(string)))
	case "SMTPMessage.withText":
		return session.smtpMessage(ahdruntime.AhdSMTPMessageWithText(class,
			session.smtpDataOf(receiver, evaluatorSMTPMessageClass, evaluatorSMTPMessageField, "SMTPMessage"),
			args[0].(string)))
	case "SMTPMessage.withHtml":
		return session.smtpMessage(ahdruntime.AhdSMTPMessageWithHtml(class,
			session.smtpDataOf(receiver, evaluatorSMTPMessageClass, evaluatorSMTPMessageField, "SMTPMessage"),
			args[0].(string)))
	}
	session.raise("Error", "unsupported SMTP operation "+name)
	return nil
}
