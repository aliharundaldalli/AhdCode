//go:build darwin

package ahdidentity

import (
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// applyPlatform names the running helper AhdCode in the menu bar and gives it
// the AhdCode Dock icon. It runs after the application finished launching:
// LaunchServices only knows the process from then on, and AppKit replaces an
// icon set earlier.
//
// The menu-bar name of an executable without an application bundle comes
// from its file name; the only run-time way to change it is LaunchServices'
// _LSSetApplicationInformationItem with kLSDisplayNameKey, a private but
// long-stable function (browsers use it to name their helper processes). It
// is looked up with dlsym, so if a future macOS removes it the helper simply
// keeps its file name. No cgo, Objective-C source, or bundle is involved.
//
// Inside a packaged application the bundle's Info.plist already names the
// application and gives its Dock icon, so nothing is changed at run time and
// the private function is never called.
func applyPlatform() {
	if Packaged() {
		return
	}
	setDisplayName(DefaultName)
	setDockIcon(RoundedPNG(officialIcon(), 512))
}

func nsString(text string) objc.ID {
	bytes := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), unsafe.Pointer(&bytes[0]))
}

func setDisplayName(name string) {
	services, err := purego.Dlopen("/System/Library/Frameworks/CoreServices.framework/CoreServices", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	currentASN, err1 := purego.Dlsym(services, "_LSGetCurrentApplicationASN")
	setItem, err2 := purego.Dlsym(services, "_LSSetApplicationInformationItem")
	displayNameKey, err3 := purego.Dlsym(services, "_kLSDisplayNameKey")
	if err1 != nil || err2 != nil || err3 != nil || displayNameKey == 0 {
		return
	}
	asn, _, _ := purego.SyscallN(currentASN)
	if asn == 0 {
		return
	}
	// dlsym returns the address of the CFStringRef constant; read its value.
	key := *(*uintptr)(unsafe.Add(unsafe.Pointer(nil), displayNameKey))
	// kLSDefaultSessionID is -2.
	const defaultSession = uintptr(0xfffffffe)
	purego.SyscallN(setItem, defaultSession, asn, key, uintptr(nsString(name)), 0)
}

func setDockIcon(png []byte) {
	if len(png) == 0 {
		return
	}
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	data := objc.ID(objc.GetClass("NSData")).Send(objc.RegisterName("dataWithBytes:length:"), unsafe.Pointer(&png[0]), uint64(len(png)))
	image := objc.ID(objc.GetClass("NSImage")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("initWithData:"), data)
	if image == 0 {
		return
	}
	// Apply runs on Ebitengine's update thread; AppKit must be changed on the
	// main thread, and waiting for it here would deadlock the frame loop.
	app.Send(objc.RegisterName("performSelectorOnMainThread:withObject:waitUntilDone:"),
		objc.RegisterName("setApplicationIconImage:"), image, false)
}
