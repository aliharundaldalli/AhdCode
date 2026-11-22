package ahdruntime

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func unwritableDownloadDirectory(t *testing.T, parent string) string {
	t.Helper()
	path := filepath.Join(parent, "locked")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid := user.User.Sid.String()
	// Deny FILE_ADD_FILE (0x2) to the test identity on this directory.
	// Other rights remain available, including enumeration and cleanup.
	sd, err := windows.SecurityDescriptorFromString("D:P(D;;0x2;;;" + sid + ")(A;;FA;;;" + sid + ")(A;;FA;;;SY)")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	requireDownloadDirectoryDeniesCreation(t, path)
	return path
}
