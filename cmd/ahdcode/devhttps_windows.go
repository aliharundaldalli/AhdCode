package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func mkdirPrivateDevHTTPS(path string) error {
	attributes, err := privateWindowsSecurityAttributes("OICI")
	if err != nil {
		return err
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.CreateDirectory(name, attributes)
}

func privateWindowsSecurityAttributes(inheritance string) (*windows.SecurityAttributes, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	rule := "(A;" + inheritance + ";FA;;;"
	sd, err := windows.SecurityDescriptorFromString("D:P" + rule + "SY)" + rule + "BA)" + rule + user.User.Sid.String() + ")")
	if err != nil {
		return nil, err
	}
	return &windows.SecurityAttributes{
		Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd,
	}, nil
}

func checkDevHTTPSPrivacy(path string) error {
	token := windows.GetCurrentProcessToken()
	user, err := token.GetTokenUser()
	if err != nil {
		return fmt.Errorf("cannot verify the Windows identity for local HTTPS key storage")
	}
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("cannot inspect Windows local HTTPS key permissions: %w", err)
	}
	owner, _, err := sd.Owner()
	ownerOK := owner != nil && (owner.Equals(user.User.Sid) || (token.IsElevated() && owner.IsWellKnown(windows.WinBuiltinAdministratorsSid)))
	if err != nil || !ownerOK {
		return fmt.Errorf("local HTTPS key storage must be owned by the current Windows user")
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		return fmt.Errorf("local HTTPS key storage needs a private Windows ACL")
	}
	for index := uint16(0); index < dacl.AceCount; index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, uint32(index), &ace); err != nil || ace == nil {
			return fmt.Errorf("could not inspect a Windows local HTTPS access rule")
		}
		switch ace.Header.AceType {
		case windows.ACCESS_DENIED_ACE_TYPE:
			continue
		case windows.ACCESS_ALLOWED_ACE_TYPE:
			sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
			if sid.Equals(user.User.Sid) || sid.IsWellKnown(windows.WinLocalSystemSid) || sid.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
				continue
			}
		}
		return fmt.Errorf("local HTTPS key ACL must grant access only to the current user, SYSTEM, and Administrators")
	}
	return nil
}
