package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestDevHTTPSWindowsRefusesBroadKeyACL(t *testing.T) {
	home := newDevHTTPSTestHome(t)
	file := filepath.Join(home, "fake-key.pem")
	if err := os.WriteFile(file, []byte("test fixture without private material"), 0o600); err != nil {
		t.Fatal(err)
	}
	current, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FR;;;WD)(A;;FA;;;" + current.User.Sid.String() + ")")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(file, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	if err := checkDevHTTPSPrivacy(file); err == nil || !strings.Contains(err.Error(), "ACL") {
		t.Fatalf("broad ACL was accepted: %v", err)
	}
}

func TestDevHTTPSWindowsRefusesJunctionPath(t *testing.T) {
	home := newDevHTTPSTestHome(t)
	junction := filepath.Join(home, "redirect")
	target := t.TempDir()
	cmd := exec.Command("cmd", "/c", "mklink", "/J", junction, target)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("Windows junction unavailable: %v: %s", err, output)
	}
	t.Cleanup(func() { _ = os.Remove(junction) })
	if err := devHTTPSPathWithoutLinks(junction); err == nil {
		t.Fatal("junction accepted as HTTPS credential storage")
	}
}
