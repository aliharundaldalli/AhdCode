package main

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Dialog mode. GUI.openFile, openFiles, selectFolder, saveFile, message,
// and confirm each start one ahdgui process whose first and only request is
//
//	{"op":"dialog","version":3,"kind":...,"title":...,"text":...,"name":...,"extensions":[...]}
//
// The helper shows one dialog and answers with one response line, then
// exits:
//
//	{"ok":true,"paths":[...]}          files or a folder were chosen
//	{"ok":true,"cancelled":true}       the user cancelled
//	{"ok":true,"confirmed":true|false} a message was acknowledged or a
//	                                   confirmation answered
//	{"ok":false,"error":...}           the dialog could not be shown
//
// macOS and Windows show the system's own dialogs (dialog_darwin.go,
// dialog_windows.go); elsewhere, and if a system dialog cannot be shown at
// all, the helper draws its own dialog window with the GUI widgets
// (dialog_fallback.go). The helper never reads, creates, or changes a chosen
// file: choosing a path and using it are separate steps of the program.

// Dialog kinds.
const (
	dialogOpenFile     = "openFile"
	dialogOpenFiles    = "openFiles"
	dialogSelectFolder = "selectFolder"
	dialogSaveFile     = "saveFile"
	dialogMessage      = "message"
	dialogConfirm      = "confirm"
)

// Dialog bounds.
const (
	maxNameRunes    = 255
	maxExtensions   = 32
	maxExtensionLen = 16
	maxPaths        = 10_000
	maxPathBytes    = 32 << 10
)

// answer is a headless test dialog's scripted answer.
type answer struct {
	Cancel  bool     `json:"cancel,omitempty"`
	Paths   []string `json:"paths,omitempty"`
	Confirm bool     `json:"confirm,omitempty"`
	Fail    string   `json:"fail,omitempty"`
}

// dialogRequest is a validated dialog request.
type dialogRequest struct {
	kind, title, text, name string
	extensions              []string
}

// dialogResult is what the user chose.
type dialogResult struct {
	cancelled bool
	paths     []string
	confirmed bool
}

// validExtension accepts a plain file extension: letters and digits only.
func validExtension(extension string) bool {
	if extension == "" || len(extension) > maxExtensionLen {
		return false
	}
	for _, character := range extension {
		if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9') {
			return false
		}
	}
	return true
}

// validFileName accepts a suggested file name: no folder part.
func validFileName(name string) bool {
	return validText(name, maxNameRunes) && !strings.ContainsAny(name, `/\:`) && name != "." && name != ".."
}

func validateDialog(first request) (dialogRequest, error) {
	if first.Version != protocolVersion {
		return dialogRequest{}, errors.New("unsupported dialog protocol version")
	}
	switch first.Kind {
	case dialogOpenFile, dialogOpenFiles, dialogSelectFolder, dialogSaveFile, dialogMessage, dialogConfirm:
	default:
		return dialogRequest{}, errors.New("unknown dialog kind " + first.Kind)
	}
	if !validText(first.Title, maxTitleRunes) || !validText(first.Text, maxTextRunes) {
		return dialogRequest{}, errors.New("invalid dialog text")
	}
	if first.Name != "" && !validFileName(first.Name) {
		return dialogRequest{}, errors.New("invalid suggested file name")
	}
	if len(first.Extensions) > maxExtensions {
		return dialogRequest{}, errors.New("too many file extensions")
	}
	for _, extension := range first.Extensions {
		if !validExtension(extension) {
			return dialogRequest{}, errors.New("invalid file extension")
		}
	}
	if first.Answer != nil && !first.Headless {
		return dialogRequest{}, errors.New("only a headless dialog has a scripted answer")
	}
	return dialogRequest{kind: first.Kind, title: first.Title, text: first.Text, name: first.Name,
		extensions: first.Extensions}, nil
}

// runDialog shows one dialog and returns the response to send.
func runDialog(first request) response {
	req, err := validateDialog(first)
	if err != nil {
		return response{Error: err.Error()}
	}
	var result dialogResult
	switch {
	case first.Headless:
		result, err = scriptedDialog(req, first.Answer)
	default:
		var handled bool
		result, handled, err = nativeDialog(req)
		if !handled {
			result, err = fallbackDialog(req)
		}
	}
	if err != nil {
		return response{Error: err.Error()}
	}
	return dialogResponse(req, result)
}

// scriptedDialog answers a headless test dialog from its script.
func scriptedDialog(req dialogRequest, scripted *answer) (dialogResult, error) {
	if scripted == nil {
		return dialogResult{}, errors.New("the headless dialog has no scripted answer")
	}
	if scripted.Fail != "" {
		return dialogResult{}, errors.New(scripted.Fail)
	}
	switch req.kind {
	case dialogMessage:
		return dialogResult{confirmed: true}, nil
	case dialogConfirm:
		return dialogResult{confirmed: scripted.Confirm}, nil
	}
	if scripted.Cancel {
		return dialogResult{cancelled: true}, nil
	}
	return dialogResult{paths: scripted.Paths}, nil
}

// dialogResponse checks what the dialog returned: every path is absolute,
// valid text, and within bounds; one-path dialogs return exactly one. A
// saved name without an allowed extension gets the first one.
func dialogResponse(req dialogRequest, result dialogResult) response {
	switch req.kind {
	case dialogMessage, dialogConfirm:
		confirmed := result.confirmed
		return response{OK: true, Confirmed: &confirmed}
	}
	if result.cancelled || len(result.paths) == 0 {
		return response{OK: true, Cancelled: true}
	}
	if len(result.paths) > maxPaths || (req.kind != dialogOpenFiles && len(result.paths) != 1) {
		return response{Error: "the dialog returned an unexpected number of paths"}
	}
	paths := make([]string, len(result.paths))
	for index, path := range result.paths {
		if path == "" || len(path) > maxPathBytes || !utf8.ValidString(path) || strings.ContainsRune(path, 0) || !filepath.IsAbs(path) {
			return response{Error: "the dialog returned an invalid path"}
		}
		paths[index] = filepath.Clean(path)
	}
	if req.kind == dialogSaveFile && len(req.extensions) > 0 && !hasExtension(paths[0], req.extensions) {
		paths[0] += "." + req.extensions[0]
	}
	return response{OK: true, Paths: paths}
}

func hasExtension(path string, extensions []string) bool {
	extension := strings.TrimPrefix(filepath.Ext(path), ".")
	for _, allowed := range extensions {
		if strings.EqualFold(extension, allowed) {
			return true
		}
	}
	return false
}

// dialogTitle is the dialog's title, or a default one.
func dialogTitle(req dialogRequest) string {
	if req.title != "" {
		return req.title
	}
	return map[string]string{
		dialogOpenFile: "Open File", dialogOpenFiles: "Open Files", dialogSelectFolder: "Select Folder",
		dialogSaveFile: "Save File", dialogMessage: "Message", dialogConfirm: "Confirm",
	}[req.kind]
}
