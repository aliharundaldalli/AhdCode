package main

// The setup program is linked for the Windows GUI subsystem so double-clicking
// it in Explorer never opens a console and never waits on stdin. This file
// holds the small amount of Win32 needed for that: a confirmation dialog, a
// progress window, a result dialog, and the environment-change broadcast that
// makes a freshly opened terminal see the new PATH. Only user32, kernel32,
// gdi32, and comctl32 are used; no installer framework is involved.

import (
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	comctl32 = windows.NewLazySystemDLL("comctl32.dll")

	procMessageBox           = user32.NewProc("MessageBoxW")
	procRegisterClassEx      = user32.NewProc("RegisterClassExW")
	procCreateWindowEx       = user32.NewProc("CreateWindowExW")
	procDefWindowProc        = user32.NewProc("DefWindowProcW")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessage           = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessage      = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procPostMessage          = user32.NewProc("PostMessageW")
	procSendMessage          = user32.NewProc("SendMessageW")
	procSendMessageTimeout   = user32.NewProc("SendMessageTimeoutW")
	procSetWindowText        = user32.NewProc("SetWindowTextW")
	procLoadCursor           = user32.NewProc("LoadCursorW")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procGetStockObject       = gdi32.NewProc("GetStockObject")
	procGetModuleHandle      = kernel32.NewProc("GetModuleHandleW")
	procAttachConsole        = kernel32.NewProc("AttachConsole")
	procInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")
)

const (
	mbOK        = 0x0000
	mbOKCancel  = 0x0001
	mbIconError = 0x0010
	mbIconInfo  = 0x0040
	mbIconWarn  = 0x0030
	mbTopMost   = 0x00040000
	mbSetFore   = 0x00010000
	idOK        = 1

	wsOverlapped = 0x00000000
	wsCaption    = 0x00C00000
	wsSysMenu    = 0x00080000
	wsVisible    = 0x10000000
	wsChild      = 0x40000000
	ssLeftNoWrap = 0x0000000C

	wmDestroy     = 0x0002
	wmAppDone     = 0x8001
	pbmSetRange32 = 0x0406
	pbmSetPos     = 0x0402

	iccProgressClass    = 0x00000020
	attachParentProcess = ^uintptr(0)
	swShow              = 5
	smCxScreen          = 0
	smCyScreen          = 1
	idcArrow            = 32512
	defaultGUIFont      = 17
	wmSetFont           = 0x0030
	colorBtnFace        = 15
)

func utf16(text string) *uint16 {
	pointer, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		pointer, _ = syscall.UTF16PtrFromString("")
	}
	return pointer
}

// messageBox shows a modal dialog and reports the button the user chose.
func messageBox(text, caption string, flags uintptr) uintptr {
	result, _, _ := procMessageBox.Call(0,
		uintptr(unsafe.Pointer(utf16(text))),
		uintptr(unsafe.Pointer(utf16(caption))),
		flags|mbTopMost|mbSetFore)
	return result
}

func confirmDialog(text, caption string) bool {
	return messageBox(text, caption, mbOKCancel|mbIconInfo) == idOK
}

func errorDialog(text, caption string) {
	messageBox(text, caption, mbOK|mbIconError)
}

func infoDialog(text, caption string) {
	messageBox(text, caption, mbOK|mbIconInfo)
}

// attachParentConsole connects a GUI-subsystem process to the console of the
// shell that started it, so `AhdCode-Setup.exe --silent` from PowerShell can
// still report what it did. Double-clicking from Explorer has no parent
// console, so this fails and the program stays purely graphical.
func attachParentConsole() *os.File {
	if handle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && handle != 0 && handle != windows.InvalidHandle {
		return os.Stdout
	}
	if result, _, _ := procAttachConsole.Call(attachParentProcess); result == 0 {
		return nil
	}
	output, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return nil
	}
	return output
}

// broadcastEnvironmentChange tells running programs that the environment was
// edited. Explorer caches its environment block and hands it to every terminal
// it launches, so without this a newly opened PowerShell keeps the old PATH
// and `ahdcode` stays unresolvable until the user signs out.
func broadcastEnvironmentChange() {
	const wmSettingChange = 0x001A
	const hwndBroadcast = 0xFFFF
	const smtoAbortIfHung = 0x0002
	var result uintptr
	procSendMessageTimeout.Call(hwndBroadcast, wmSettingChange, 0,
		uintptr(unsafe.Pointer(utf16("Environment"))),
		smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
}

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   windows.Handle
	icon       windows.Handle
	cursor     windows.Handle
	background windows.Handle
	menuName   *uint16
	className  *uint16
	iconSmall  windows.Handle
}

type point struct{ x, y int32 }

type msg struct {
	hwnd    windows.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

// progressWindow is a small always-visible window with a line of status text
// and a progress bar. It exists so the user is never left staring at a screen
// where "nothing happened" while several hundred megabytes are unpacked.
type progressWindow struct {
	hwnd    windows.Handle
	label   windows.Handle
	bar     windows.Handle
	class   *uint16
	mutex   sync.Mutex
	lastSet time.Time
	ok      bool
}

var progressClassOnce sync.Once
var progressCallback uintptr

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	case wmAppDone:
		procPostQuitMessage.Call(0)
		return 0
	}
	result, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)
	return result
}

// newProgressWindow returns a window, or a disabled one if any Win32 step
// fails. A cosmetic problem must never stop an installation, so every failure
// here degrades to "no progress display" rather than an error.
func newProgressWindow(caption string) *progressWindow {
	window := &progressWindow{}
	instance, _, _ := procGetModuleHandle.Call(0)
	className := utf16("AhdCodeSetupProgress")

	progressClassOnce.Do(func() {
		progressCallback = windows.NewCallback(wndProc)
		cursor, _, _ := procLoadCursor.Call(0, idcArrow)
		class := wndClassEx{
			size:       uint32(unsafe.Sizeof(wndClassEx{})),
			wndProc:    progressCallback,
			instance:   windows.Handle(instance),
			cursor:     windows.Handle(cursor),
			background: windows.Handle(colorBtnFace + 1),
			className:  className,
		}
		procRegisterClassEx.Call(uintptr(unsafe.Pointer(&class)))
	})
	if progressCallback == 0 {
		return window
	}

	controls := struct{ size, icc uint32 }{8, iccProgressClass}
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&controls)))

	const width, height = 460, 150
	screenWidth, _, _ := procGetSystemMetrics.Call(smCxScreen)
	screenHeight, _, _ := procGetSystemMetrics.Call(smCyScreen)
	x := (int32(screenWidth) - width) / 2
	y := (int32(screenHeight) - height) / 2

	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16(caption))),
		wsOverlapped|wsCaption|wsSysMenu|wsVisible,
		uintptr(x), uintptr(y), width, height,
		0, 0, instance, 0)
	if hwnd == 0 {
		return window
	}
	window.hwnd = windows.Handle(hwnd)

	label, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(utf16("STATIC"))),
		uintptr(unsafe.Pointer(utf16("Preparing..."))),
		wsChild|wsVisible|ssLeftNoWrap,
		20, 20, width-50, 40, hwnd, 0, instance, 0)
	window.label = windows.Handle(label)

	bar, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(utf16("msctls_progress32"))),
		uintptr(unsafe.Pointer(utf16(""))),
		wsChild|wsVisible,
		20, 60, width-60, 24, hwnd, 0, instance, 0)
	window.bar = windows.Handle(bar)
	if window.bar != 0 {
		procSendMessage.Call(uintptr(window.bar), pbmSetRange32, 0, 100)
	}
	if font, _, _ := procGetStockObject.Call(defaultGUIFont); font != 0 && window.label != 0 {
		procSendMessage.Call(uintptr(window.label), wmSetFont, font, 1)
	}

	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	window.ok = true
	return window
}

// Update is safe to call from the installing goroutine; Windows marshals the
// messages to the thread that owns the window.
func (w *progressWindow) Update(status string, percent int) {
	if w == nil || !w.ok {
		return
	}
	w.mutex.Lock()
	recent := time.Since(w.lastSet) < 40*time.Millisecond
	if !recent {
		w.lastSet = time.Now()
	}
	w.mutex.Unlock()
	if recent {
		return
	}
	if w.label != 0 {
		procSetWindowText.Call(uintptr(w.label), uintptr(unsafe.Pointer(utf16(status))))
	}
	if w.bar != 0 {
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		procSendMessage.Call(uintptr(w.bar), pbmSetPos, uintptr(percent), 0)
	}
}

// Done releases the window and ends the message loop started by Pump.
func (w *progressWindow) Done() {
	if w == nil || !w.ok || w.hwnd == 0 {
		return
	}
	procPostMessage.Call(uintptr(w.hwnd), wmAppDone, 0, 0)
}

// Pump runs the message loop until Done is called. Windows delivers messages to
// the thread that created the window, so the caller must have locked its
// goroutine to that thread before creating it. The installation itself runs on
// another goroutine, so the window keeps repainting and Windows never marks the
// program as not responding.
func (w *progressWindow) Pump() {
	if w == nil || !w.ok {
		return
	}
	var message msg
	for {
		result, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if result == 0 || result == ^uintptr(0) {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
	if w.hwnd != 0 {
		procDestroyWindow.Call(uintptr(w.hwnd))
		w.hwnd = 0
	}
}
