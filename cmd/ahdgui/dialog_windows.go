//go:build windows

package main

import (
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// The Windows dialogs are the system's common dialogs -- GetOpenFileNameW,
// GetSaveFileNameW, SHBrowseForFolderW, and MessageBoxW -- called through
// the standard library's syscall package, without cgo.

var (
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procGetOpenFileName   = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileName   = comdlg32.NewProc("GetSaveFileNameW")
	procDialogError       = comdlg32.NewProc("CommDlgExtendedError")
	procBrowseForFolder   = shell32.NewProc("SHBrowseForFolderW")
	procPathFromIDList    = shell32.NewProc("SHGetPathFromIDListW")
	procMessageBox        = user32.NewProc("MessageBoxW")
	procCoInitialize      = ole32.NewProc("CoInitializeEx")
	procCoTaskMemFree     = ole32.NewProc("CoTaskMemFree")
	procGetForegroundWind = user32.NewProc("GetForegroundWindow")
)

// openFileName is OPENFILENAMEW.
type openFileName struct {
	structSize    uint32
	owner         uintptr
	instance      uintptr
	filter        *uint16
	customFilter  *uint16
	maxCustom     uint32
	filterIndex   uint32
	file          *uint16
	maxFile       uint32
	fileTitle     *uint16
	maxFileTitle  uint32
	initialDir    *uint16
	title         *uint16
	flags         uint32
	fileOffset    uint16
	fileExtension uint16
	defExt        *uint16
	custData      uintptr
	hook          uintptr
	templateName  *uint16
	reserved      uintptr
	reserved2     uint32
	flagsEx       uint32
}

// browseInfo is BROWSEINFOW.
type browseInfo struct {
	owner       uintptr
	root        uintptr
	displayName *uint16
	title       *uint16
	flags       uint32
	callback    uintptr
	parameter   uintptr
	image       int32
}

const (
	ofnAllowMultiSelect = 0x00000200
	ofnPathMustExist    = 0x00000800
	ofnFileMustExist    = 0x00001000
	ofnOverwritePrompt  = 0x00000002
	ofnExplorer         = 0x00080000
	ofnNoChangeDir      = 0x00000008
	bifReturnOnlyFS     = 0x00000001
	bifNewDialogStyle   = 0x00000040
	mbOK                = 0x00000000
	mbOKCancel          = 0x00000001
	mbIconInformation   = 0x00000040
	mbIconQuestion      = 0x00000020
	mbTopmost           = 0x00040000
	idOK                = 1
	fileBufferChars     = 1 << 16
)

func wide(text string) *uint16 {
	pointer, _ := syscall.UTF16PtrFromString(text)
	return pointer
}

// filterOf builds the filter string: one entry for the allowed extensions
// and one for every file.
func filterOf(extensions []string) *uint16 {
	var parts []string
	if len(extensions) > 0 {
		patterns := make([]string, len(extensions))
		for index, extension := range extensions {
			patterns[index] = "*." + extension
		}
		joined := strings.Join(patterns, ";")
		parts = append(parts, joined, joined)
	}
	parts = append(parts, "All files", "*.*")
	encoded := utf16.Encode([]rune(strings.Join(parts, "\x00") + "\x00\x00"))
	return &encoded[0]
}

func nativeDialog(req dialogRequest) (dialogResult, bool, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if procGetOpenFileName.Find() != nil || procMessageBox.Find() != nil {
		return dialogResult{}, false, nil
	}
	owner, _, _ := procGetForegroundWind.Call()
	switch req.kind {
	case dialogMessage, dialogConfirm:
		flags := uintptr(mbOK | mbIconInformation | mbTopmost)
		if req.kind == dialogConfirm {
			flags = mbOKCancel | mbIconQuestion | mbTopmost
		}
		result, _, _ := procMessageBox.Call(owner, uintptr(unsafe.Pointer(wide(req.text))), uintptr(unsafe.Pointer(wide(dialogTitle(req)))), flags)
		return dialogResult{confirmed: result == idOK}, true, nil
	case dialogSelectFolder:
		return selectFolder(req, owner), true, nil
	}
	buffer := make([]uint16, fileBufferChars)
	if req.kind == dialogSaveFile && req.name != "" {
		copy(buffer, utf16.Encode([]rune(req.name)))
	}
	dialog := openFileName{owner: owner, filter: filterOf(req.extensions), filterIndex: 1, file: &buffer[0],
		maxFile: fileBufferChars, title: wide(dialogTitle(req)), flags: ofnExplorer | ofnNoChangeDir | ofnPathMustExist}
	dialog.structSize = uint32(unsafe.Sizeof(dialog))
	procedure := procGetOpenFileName
	switch req.kind {
	case dialogSaveFile:
		procedure = procGetSaveFileName
		dialog.flags |= ofnOverwritePrompt
		if len(req.extensions) > 0 {
			dialog.defExt = wide(req.extensions[0])
		}
	case dialogOpenFiles:
		dialog.flags |= ofnFileMustExist | ofnAllowMultiSelect
	default:
		dialog.flags |= ofnFileMustExist
	}
	ok, _, _ := procedure.Call(uintptr(unsafe.Pointer(&dialog)))
	if ok == 0 {
		if code, _, _ := procDialogError.Call(); code != 0 {
			return dialogResult{}, true, errDialog
		}
		return dialogResult{cancelled: true}, true, nil
	}
	return dialogResult{paths: splitFiles(buffer, req.kind == dialogOpenFiles)}, true, nil
}

// splitFiles reads the chosen paths: one full path, or, for several files,
// the folder followed by the file names, each ended by a NUL.
func splitFiles(buffer []uint16, several bool) []string {
	var parts []string
	start := 0
	for index, unit := range buffer {
		if unit != 0 {
			continue
		}
		if index == start {
			break
		}
		parts = append(parts, string(utf16.Decode(buffer[start:index])))
		start = index + 1
		if !several {
			break
		}
	}
	if len(parts) <= 1 {
		return parts
	}
	folder := parts[0]
	paths := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		paths = append(paths, strings.TrimRight(folder, `\`)+`\`+name)
	}
	return paths
}

func selectFolder(req dialogRequest, owner uintptr) dialogResult {
	_, _, _ = procCoInitialize.Call(0, 2) // COINIT_APARTMENTTHREADED
	name := make([]uint16, 260)
	info := browseInfo{owner: owner, displayName: &name[0], title: wide(dialogTitle(req)), flags: bifReturnOnlyFS | bifNewDialogStyle}
	list, _, _ := procBrowseForFolder.Call(uintptr(unsafe.Pointer(&info)))
	if list == 0 {
		return dialogResult{cancelled: true}
	}
	defer procCoTaskMemFree.Call(list)
	path := make([]uint16, fileBufferChars)
	if ok, _, _ := procPathFromIDList.Call(list, uintptr(unsafe.Pointer(&path[0]))); ok == 0 {
		return dialogResult{cancelled: true}
	}
	return dialogResult{paths: []string{syscall.UTF16ToString(path)}}
}
