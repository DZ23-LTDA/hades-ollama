//go:build windows

package secrets

import (
	"encoding/hex"
	"errors"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Matches the file written by scripts/dz23-configure.ps1 (DPAPI, user scope),
// which internal/multillm decodes.
const fileExtension = ".dpapi"

func protect(key string) ([]byte, error) {
	units := utf16.Encode([]rune(key))
	plain := make([]byte, len(units)*2)
	for i, unit := range units {
		plain[i*2] = byte(unit)
		plain[i*2+1] = byte(unit >> 8)
	}
	in := windows.DataBlob{Size: uint32(len(plain)), Data: &plain[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	if out.Data == nil || out.Size == 0 {
		return nil, errors.New("DPAPI returned no data")
	}
	cipher := unsafe.Slice(out.Data, out.Size)
	return []byte(hex.EncodeToString(cipher)), nil
}

func persistUserEnv(envName, path string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.DeleteValue(envName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	if path == "" {
		if err := key.DeleteValue(envName + "_FILE"); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	return key.SetStringValue(envName+"_FILE", path)
}
