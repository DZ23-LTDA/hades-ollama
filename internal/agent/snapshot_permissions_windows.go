//go:build windows

package agent

import (
	"errors"
	"fmt"
	"os"
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
	ownerACL, err := inheritableOwnerACL(sid)
	if err != nil {
		return err
	}
	dacl, _, err := ownerACL.DACL()
	if err != nil {
		return err
	}
	if dacl == nil || dacl.AceCount != 1 {
		return errors.New("snapshot security descriptor does not contain exactly one owner ACE")
	}
	if err := windows.SetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil); err != nil {
		return err
	}
	return nil
}

// inheritableOwnerACL parses a Windows SDDL security descriptor with a protected
// DACL and one owner-only allow ACE. SDDL lets Windows construct the ACE layout
// and keeps OI/CI inheritance flags intact when the DACL is applied.
func inheritableOwnerACL(sid *windows.SID) (*windows.SECURITY_DESCRIPTOR, error) {
	if sid == nil || !sid.IsValid() {
		return nil, errors.New("snapshot owner SID is invalid")
	}
	sidText := sid.String()
	if sidText == "" {
		return nil, errors.New("snapshot owner SID could not be encoded")
	}
	sddl := fmt.Sprintf("D:P(A;OICI;GA;;;%s)", sidText)
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return nil, fmt.Errorf("build owner-only inheritable snapshot DACL: %w", err)
	}
	control, _, err := sd.Control()
	if err != nil {
		return nil, fmt.Errorf("inspect snapshot DACL protection: %w", err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return nil, errors.New("snapshot DACL is not protected from parent inheritance")
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return nil, err
	}
	if acl == nil || acl.AceCount != 1 {
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
	return sd, nil
}
