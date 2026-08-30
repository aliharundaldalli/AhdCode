package evaluator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"
)

func (session *Session) sessionPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(session.CWD, path)
}

func (session *Session) fileError(operation, path string, err error) {
	message := operation + " " + fmt.Sprintf("%q", path) + " failed"
	if err != nil {
		message += ": " + err.Error()
	}
	session.raise("FileError", message)
}

func (session *Session) fileBuiltin(name string, arguments []any) any {
	path := arguments[0].(string)
	resolved := session.sessionPath(path)
	switch name {
	case "exists":
		_, err := os.Stat(resolved)
		if err == nil {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		session.fileError("stat", path, err)
	case "readText":
		content, err := os.ReadFile(resolved)
		if err != nil {
			session.fileError("read", path, err)
		}
		if !utf8.Valid(content) {
			session.fileError("read", path, fmt.Errorf("content is not valid UTF-8"))
		}
		return string(content)
	case "writeText":
		content := arguments[1].(string)
		if !utf8.ValidString(content) {
			session.fileError("write", path, fmt.Errorf("content is not valid UTF-8"))
		}
		if err := os.WriteFile(resolved, []byte(content), 0o666); err != nil {
			session.fileError("write", path, err)
		}
		return Nothing
	case "append":
		content := arguments[1].(string)
		if !utf8.ValidString(content) {
			session.fileError("append", path, fmt.Errorf("content is not valid UTF-8"))
		}
		file, err := os.OpenFile(resolved, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			session.fileError("append", path, err)
		}
		if _, err = io.WriteString(file, content); err != nil {
			_ = file.Close()
			session.fileError("append", path, err)
		}
		if err = file.Close(); err != nil {
			session.fileError("close", path, err)
		}
		return Nothing
	case "delete":
		if err := os.Remove(resolved); err != nil {
			session.fileError("delete", path, err)
		}
		return Nothing
	case "createDir":
		if err := os.MkdirAll(resolved, 0o777); err != nil {
			session.fileError("create directory", path, err)
		}
		return Nothing
	case "list":
		entries, err := os.ReadDir(resolved)
		if err != nil {
			session.fileError("list", path, err)
		}
		names := make([]string, len(entries))
		for index, entry := range entries {
			names[index] = entry.Name()
		}
		sort.Strings(names)
		items := make([]any, len(names))
		for index := range names {
			items[index] = names[index]
		}
		return &List{Items: items}
	}
	session.raise("Error", "unsupported File operation "+name)
	return nil
}

func (session *Session) pathBuiltin(name string, arguments []any) any {
	switch name {
	case "join":
		list := session.requireList(arguments[0])
		parts := make([]string, len(list.Items))
		for index, item := range list.Items {
			parts[index] = item.(string)
		}
		return filepath.Join(parts...)
	case "ext":
		return filepath.Ext(arguments[0].(string))
	case "base":
		return filepath.Base(arguments[0].(string))
	case "dir":
		return filepath.Dir(arguments[0].(string))
	}
	session.raise("Error", "unsupported Path operation "+name)
	return nil
}
