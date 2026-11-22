package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func prepareStudioEnvPermissionFixture(t *testing.T, dir string) {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	// A child that simply inherits these permissions would be readable by
	// Everyone. The new .env must have its own protected ACL.
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FR;;;WD)(A;OICI;FA;;;SY)(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
}

func studioEnvPermissionFingerprint(t *testing.T, path string) string {
	t.Helper()
	if err := checkDevHTTPSPrivacy(path); err != nil {
		t.Fatalf("Studio env must have a private ACL: %v", err)
	}
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	return sd.String()
}
