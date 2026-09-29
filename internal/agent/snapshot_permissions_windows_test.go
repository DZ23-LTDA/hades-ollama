//go:build windows

package agent

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func assertWorkspaceSnapshotOwnerDACL(t *testing.T, path string, requireInheritance bool) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("snapshot directory DACL is not protected from inheritance")
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if dacl == nil || dacl.AceCount == 0 {
		t.Fatal("snapshot directory has no explicit owner DACL")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	owner, _, err := sd.Owner()
	if err != nil {
		t.Fatal(err)
	}
	if owner == nil || !owner.Equals(user.User.Sid) {
		t.Fatal("snapshot directory is not owned by the current service SID")
	}
	ownerAllowed := false
	for index := uint32(0); index < uint32(dacl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, index, &ace); err != nil {
			t.Fatal(err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			t.Fatalf("unexpected non-allow ACE type %d", ace.Header.AceType)
		}
		aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !aceSID.Equals(user.User.Sid) {
			t.Fatalf("snapshot ACL grants access to unexpected SID %s", aceSID.String())
		}
		// Windows expands a GENERIC_ALL grant into its specific access bits
		// when the ACE is stored, so assert the resolved full-access bits
		// rather than the generic alias that never survives the round trip.
		const fullAccessBits = windows.ACCESS_MASK(
			windows.FILE_READ_DATA | windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA |
				windows.FILE_READ_EA | windows.FILE_WRITE_EA | windows.FILE_READ_ATTRIBUTES |
				windows.FILE_WRITE_ATTRIBUTES | windows.DELETE | windows.READ_CONTROL |
				windows.WRITE_DAC | windows.WRITE_OWNER | windows.SYNCHRONIZE,
		)
		if ace.Mask&fullAccessBits != fullAccessBits {
			t.Fatalf("service SID access mask %#x does not grant full access to the owner", ace.Mask)
		}
		if requireInheritance && ace.Header.AceFlags&(windows.CONTAINER_INHERIT_ACE|windows.OBJECT_INHERIT_ACE) != windows.CONTAINER_INHERIT_ACE|windows.OBJECT_INHERIT_ACE {
			t.Fatalf("snapshot ACL does not inherit to child directories and files: flags=%#x", ace.Header.AceFlags)
		}
		ownerAllowed = true
	}
	if !ownerAllowed {
		t.Fatal("snapshot DACL does not grant access to the current service SID")
	}
}

func TestWorkspaceSnapshotWindowsDACLProtectsOwnerAndChildren(t *testing.T) {
	parent := t.TempDir()
	parentRoot, err := os.OpenRoot(parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := secureWorkspaceSnapshotDirectory(parent, parentRoot); err != nil {
		_ = parentRoot.Close()
		t.Fatalf("secure parent snapshot directory: %v", err)
	}
	assertWorkspaceSnapshotOwnerDACL(t, parent, true)

	child := filepath.Join(parent, "tree")
	if err := os.Mkdir(child, 0o700); err != nil {
		_ = parentRoot.Close()
		t.Fatal(err)
	}
	childRoot, err := os.OpenRoot(child)
	if err != nil {
		_ = parentRoot.Close()
		t.Fatal(err)
	}
	if err := secureWorkspaceSnapshotDirectory(child, childRoot); err != nil {
		_ = childRoot.Close()
		_ = parentRoot.Close()
		t.Fatalf("secure child snapshot directory: %v", err)
	}
	assertWorkspaceSnapshotOwnerDACL(t, child, true)
	if err := childRoot.Close(); err != nil {
		t.Errorf("close child root while parent remains open: %v", err)
	}
	if err := parentRoot.Close(); err != nil {
		t.Errorf("close parent root: %v", err)
	}
}
