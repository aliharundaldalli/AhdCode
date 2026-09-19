//go:build !darwin && !windows

package main

// Without a system dialog of its own, the helper always draws the fallback
// dialog (dialog_fallback.go).
func nativeDialog(dialogRequest) (dialogResult, bool, error) {
	return dialogResult{}, false, nil
}
