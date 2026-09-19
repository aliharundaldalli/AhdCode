//go:build !darwin

package ahdidentity

// applyPlatform has nothing left to do: Prepare already set the window icon,
// and the window title already names the window.
func applyPlatform() {}
