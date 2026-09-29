//go:build windows

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const trustedInstallerSID = "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464"

func trustedSystemExecutableDirectory(path string) (string, bool) {
	resolved, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", false
	}
	canonical, err := filepath.EvalSymlinks(resolved)
	if err != nil || !strings.EqualFold(canonical, resolved) {
		return "", false
	}
	for current := resolved; ; current = filepath.Dir(current) {
		before, err := os.Lstat(current)
		if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
			return "", false
		}
		if !trustedWindowsSecurity(current) {
			return "", false
		}
		after, err := os.Lstat(current)
		if err != nil || !os.SameFile(before, after) {
			return "", false
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return resolved, true
}

func trustedSystemExecutableFile(path string, info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	before, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, before) || !trustedWindowsSecurity(path) {
		return false
	}
	after, err := os.Lstat(path)
	return err == nil && after.Mode().IsRegular() && after.Mode()&os.ModeSymlink == 0 && os.SameFile(before, after)
}

// trustedWindowsSecurity accepts write access only for system, local service,
// network service, Administrators and Windows TrustedInstaller. Ownership by
// any other principal is rejected because owners can rewrite an object's DACL.
func trustedWindowsSecurity(path string) bool {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil || sd == nil {
		return false
	}
	owner, _, err := sd.Owner()
	if err != nil || !trustedWindowsPrincipal(owner) {
		return false
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		return false
	}
	const writeRights = windows.ACCESS_MASK(
		windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_WRITE_EA |
			windows.FILE_WRITE_ATTRIBUTES | windows.DELETE | windows.WRITE_DAC |
			windows.WRITE_OWNER | windows.GENERIC_WRITE | windows.GENERIC_ALL | 0x40, // FILE_DELETE_CHILD
	)
	for index := uint32(0); index < uint32(dacl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, index, &ace); err != nil || ace == nil {
			return false
		}
		switch ace.Header.AceType {
		case windows.ACCESS_DENIED_ACE_TYPE:
			continue
		case windows.ACCESS_ALLOWED_ACE_TYPE:
		default:
			// Unknown/object/callback ACE layouts are ambiguous here. Fail closed
			// instead of guessing where their SID and access mask are located.
			return false
		}
		if ace.Mask&writeRights == 0 {
			continue
		}
		// An inherit-only ACE does not apply to this object itself, so it
		// cannot grant a caller write access to it. Windows places such
		// entries on trusted system directories (for example Git's folder);
		// skipping them keeps the audit accurate instead of failing closed
		// on an entry that grants nothing here.
		if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue
		}
		principal := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !trustedWindowsPrincipal(principal) {
			return false
		}
	}
	return true
}

func trustedWindowsPrincipal(sid *windows.SID) bool {
	if sid == nil {
		return false
	}
	if sid.IsWellKnown(windows.WinLocalSystemSid) || sid.IsWellKnown(windows.WinLocalServiceSid) || sid.IsWellKnown(windows.WinNetworkServiceSid) || sid.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
		return true
	}
	return sid.String() == trustedInstallerSID
}
