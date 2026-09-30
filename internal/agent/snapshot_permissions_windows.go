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
	acl, err := inheritableOwnerACL(sid, entry.AccessPermissions)
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

// inheritableOwnerACL builds a protected DACL holding exactly one allow ACE for
// the supplied SID, carrying both container- and object-inherit flags so the
// snapshot protection is passed on to child directories and files. The ACE is
// laid out explicitly because SetEntriesInAcl does not reliably preserve those
// inherit flags, and without them a snapshot directory would be protected but
// its children would not.
func inheritableOwnerACL(sid *windows.SID, mask windows.ACCESS_MASK) (*windows.ACL, error) {
	if sid == nil || !sid.IsValid() {
		return nil, errors.New("snapshot owner SID is invalid")
	}
	sidLength := sid.Len()
	// ACCESS_ALLOWED_ACE is ACE_HEADER (4 bytes), ACCESS_MASK (4 bytes), SID.
	aceSize := 4 + 4 + sidLength
	if aceSize%4 != 0 || aceSize > 0xFFFF {
		return nil, errors.New("snapshot owner SID has an unsupported length")
	}
	aclSize := 8 + aceSize
	raw := make([]byte, aclSize)
	// ACL header: revision, size, ACE count.
	raw[0] = 2 // ACL_REVISION
	raw[1] = 0
	raw[2] = byte(aclSize)
	raw[3] = byte(aclSize >> 8)
	raw[4] = 1
	// ACCESS_ALLOWED_ACE: type, size, inherit flags, access mask, SID.
	raw[8] = windows.ACCESS_ALLOWED_ACE_TYPE
	raw[9] = byte(windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE)
	raw[10] = byte(aceSize)
	raw[11] = byte(aceSize >> 8)
	raw[12] = byte(uint32(mask))
	raw[13] = byte(uint32(mask) >> 8)
	raw[14] = byte(uint32(mask) >> 16)
	raw[15] = byte(uint32(mask) >> 24)
	if err := windows.CopySid(uint32(sidLength), (*windows.SID)(unsafe.Pointer(&raw[16])), sid); err != nil {
		return nil, err
	}
	acl := (*windows.ACL)(unsafe.Pointer(&raw[0]))
	if acl.AceCount != 1 {
		return nil, errors.New("snapshot DACL was not built with exactly one ACE")
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(acl, 0, &ace); err != nil {
		return nil, err
	}
	if ace == nil || ace.Header.AceFlags&windows.OBJECT_INHERIT_ACE == 0 || ace.Header.AceFlags&windows.CONTAINER_INHERIT_ACE == 0 {
		return nil, errors.New("snapshot DACL entry lacks inherit flags")
	}
	if !(*windows.SID)(unsafe.Pointer(&ace.SidStart)).Equals(sid) {
		return nil, errors.New("snapshot DACL entry does not reference the owner SID")
	}
	return acl, nil
}
