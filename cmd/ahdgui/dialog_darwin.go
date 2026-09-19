//go:build darwin

package main

import (
	"runtime"
	"unsafe"

	"ahdidentity"

	"github.com/ebitengine/purego/objc"
)

// The macOS dialogs are AppKit's own NSOpenPanel, NSSavePanel, and NSAlert,
// called through the Objective-C runtime without cgo. AppKit must run on the
// main thread: the process locks its main goroutine there at start-up, and a
// dialog helper runs nothing else.

func init() { runtime.LockOSThread() }

var (
	selAlloc     = objc.RegisterName("alloc")
	selInit      = objc.RegisterName("init")
	selRelease   = objc.RegisterName("release")
	selRunModal  = objc.RegisterName("runModal")
	selUTF8      = objc.RegisterName("UTF8String")
	selCount     = objc.RegisterName("count")
	selObjectAt  = objc.RegisterName("objectAtIndex:")
	selPath      = objc.RegisterName("path")
	selArrayWith = objc.RegisterName("arrayWithObjects:count:")
)

// modalOK is NSModalResponseOK; alertFirst is NSAlertFirstButtonReturn.
const (
	modalOK    = 1
	alertFirst = 1000
)

func nsText(text string) objc.ID {
	bytes := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), unsafe.Pointer(&bytes[0]))
}

// goText copies an NSString into Go.
func goText(value objc.ID) string {
	if value == 0 {
		return ""
	}
	pointer := objc.Send[uintptr](value, selUTF8)
	if pointer == 0 {
		return ""
	}
	var bytes []byte
	for offset := uintptr(0); offset < maxPathBytes; offset++ {
		character := *(*byte)(unsafe.Add(unsafe.Pointer(nil), pointer+offset))
		if character == 0 {
			break
		}
		bytes = append(bytes, character)
	}
	return string(bytes)
}

// nativeDialog shows the AppKit dialog. It reports false when AppKit is not
// usable, so the helper draws its own dialog instead.
func nativeDialog(req dialogRequest) (dialogResult, bool, error) {
	appClass := objc.GetClass("NSApplication")
	if appClass == 0 || objc.GetClass("NSOpenPanel") == 0 || objc.GetClass("NSAlert") == 0 {
		return dialogResult{}, false, nil
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selAlloc).Send(selInit)
	defer pool.Send(objc.RegisterName("drain"))

	app := objc.ID(appClass).Send(objc.RegisterName("sharedApplication"))
	// A regular application, in front: the dialog belongs to AhdCode (or
	// to the packaged application) and takes the keyboard.
	app.Send(objc.RegisterName("setActivationPolicy:"), 0)
	app.Send(objc.RegisterName("finishLaunching"))
	ahdidentity.Prepare()
	ahdidentity.Apply()
	app.Send(objc.RegisterName("activateIgnoringOtherApps:"), true)

	switch req.kind {
	case dialogMessage, dialogConfirm:
		return alert(req), true, nil
	case dialogSaveFile:
		return savePanel(req), true, nil
	}
	return openPanel(req), true, nil
}

func alert(req dialogRequest) dialogResult {
	panel := objc.ID(objc.GetClass("NSAlert")).Send(selAlloc).Send(selInit)
	defer panel.Send(selRelease)
	panel.Send(objc.RegisterName("setMessageText:"), nsText(dialogTitle(req)))
	panel.Send(objc.RegisterName("setInformativeText:"), nsText(req.text))
	panel.Send(objc.RegisterName("addButtonWithTitle:"), nsText("OK"))
	if req.kind == dialogConfirm {
		panel.Send(objc.RegisterName("addButtonWithTitle:"), nsText("Cancel"))
	}
	answer := objc.Send[int](panel, selRunModal)
	return dialogResult{confirmed: answer == alertFirst}
}

// allowTypes limits a panel to the given file extensions.
func allowTypes(panel objc.ID, extensions []string) {
	if len(extensions) == 0 {
		return
	}
	values := make([]objc.ID, len(extensions))
	for index, extension := range extensions {
		values[index] = nsText(extension)
	}
	array := objc.ID(objc.GetClass("NSArray")).Send(selArrayWith, unsafe.Pointer(&values[0]), uint64(len(values)))
	panel.Send(objc.RegisterName("setAllowedFileTypes:"), array)
}

func openPanel(req dialogRequest) dialogResult {
	panel := objc.ID(objc.GetClass("NSOpenPanel")).Send(objc.RegisterName("openPanel"))
	folder := req.kind == dialogSelectFolder
	panel.Send(objc.RegisterName("setCanChooseFiles:"), !folder)
	panel.Send(objc.RegisterName("setCanChooseDirectories:"), folder)
	panel.Send(objc.RegisterName("setCanCreateDirectories:"), folder)
	panel.Send(objc.RegisterName("setAllowsMultipleSelection:"), req.kind == dialogOpenFiles)
	panel.Send(objc.RegisterName("setResolvesAliases:"), true)
	panel.Send(objc.RegisterName("setTitle:"), nsText(dialogTitle(req)))
	panel.Send(objc.RegisterName("setMessage:"), nsText(dialogTitle(req)))
	if !folder {
		allowTypes(panel, req.extensions)
	}
	if objc.Send[int](panel, selRunModal) != modalOK {
		return dialogResult{cancelled: true}
	}
	urls := panel.Send(objc.RegisterName("URLs"))
	count := objc.Send[uint64](urls, selCount)
	var paths []string
	for index := uint64(0); index < count && index < maxPaths; index++ {
		url := urls.Send(selObjectAt, index)
		if path := goText(url.Send(selPath)); path != "" {
			paths = append(paths, path)
		}
	}
	return dialogResult{paths: paths, cancelled: len(paths) == 0}
}

func savePanel(req dialogRequest) dialogResult {
	panel := objc.ID(objc.GetClass("NSSavePanel")).Send(objc.RegisterName("savePanel"))
	panel.Send(objc.RegisterName("setCanCreateDirectories:"), true)
	panel.Send(objc.RegisterName("setTitle:"), nsText(dialogTitle(req)))
	panel.Send(objc.RegisterName("setMessage:"), nsText(dialogTitle(req)))
	if req.name != "" {
		panel.Send(objc.RegisterName("setNameFieldStringValue:"), nsText(req.name))
	}
	allowTypes(panel, req.extensions)
	if objc.Send[int](panel, selRunModal) != modalOK {
		return dialogResult{cancelled: true}
	}
	path := goText(panel.Send(objc.RegisterName("URL")).Send(selPath))
	if path == "" {
		return dialogResult{cancelled: true}
	}
	return dialogResult{paths: []string{path}}
}
