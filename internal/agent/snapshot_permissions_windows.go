//go:build windows

package agent

import (
	"errors"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

func secureWorkspaceSnapshotDirectory(path string, root *os.Root) error {
	if root == nil || path == "" {
		return errors.New("snapshot directory path and handle are required")
	}
	openedInfo, err := root.Stat(".")
	if err != nil {
		return err
	}
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(pathPtr,
		windows.FILE_READ_ATTRIBUTES|windows.READ_CONTROL|windows.WRITE_DAC,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return errors.New("could not wrap Windows snapshot directory handle")
	}
	defer file.Close()
	pathInfo, err := file.Stat()
	if err != nil {
		return err
	}
	if !pathInfo.IsDir() || !os.SameFile(openedInfo, pathInfo) {
		return ErrWorkspaceSnapshotChanged
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sid := user.User.Sid
	if sid == nil || !sid.IsValid() {
		return errors.New("current Windows user SID is invalid")
	}
	descriptor, err := windows.GetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		return err
	}
	if owner == nil || !owner.Equals(sid) {
		return errors.New("snapshot directory is not owned by the current service SID")
	}
	var pinner runtime.Pinner
	pinner.Pin(sid)
	defer pinner.Unpin()
	entry := windows.EXPLICIT_ACCESS{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{entry}, nil)
	if err != nil {
		return err
	}
	acl, err = aclWithInheritance(acl)
	if err != nil {
		return err
	}
	if err := windows.SetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil); err != nil {
		return err
	}
	return nil
}

// aclWithInheritance rebuilds a single-ACE DACL so its ACE carries the
// container/object inherit flags explicitly. SetEntriesInAcl does not always
// preserve them, and without the flags a snapshot directory would not pass
// its protection on to child files and directories.
func aclWithInheritance(source *windows.ACL) (*windows.ACL, error) {
	if source == nil || source.AceCount != 1 {
		return nil, errors.New("snapshot DACL must contain exactly one ACE")
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(source, 0, &ace); err != nil {
		return nil, err
	}
	if ace == nil || ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
		return nil, errors.New("snapshot DACL entry must be an allow ACE")
	}
	aceSize := int(ace.Header.AceSize)
	if aceSize < 8 || aceSize > 1024 {
		return nil, errors.New("snapshot DACL entry has an invalid size")
	}
	raw := make([]byte, aceSize)
	copy(raw, (*[(1 << 31) - 1]byte)(unsafe.Pointer(ace))[:aceSize:aceSize])
	rebuilt, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: ace.Mask,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID((*windows.SID)(unsafe.Pointer(&raw[8]))),
		},
	}}, nil)
	if err != nil {
		return nil, err
	}
	var rebuiltACE *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(rebuilt, 0, &rebuiltACE); err != nil {
		return nil, err
	}
	if rebuiltACE == nil || rebuiltACE.Header.AceFlags&windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT != windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT {
		return nil, errors.New("could not apply inheritable access to the snapshot directory")
	}
	return rebuilt, nil
}
