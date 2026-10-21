//go:build darwin && !cgo

package discover

// Nota medida: `unix.SysctlUint64` NAO existe no package syscall do darwin
// nesta versao de Go (a primeira tentativa falhou com
// `undefined: unix.SysctlUint64`), por isso usa-se `golang.org/x/sys/unix`,
// que ja e' dependencia do modulo (v0.49.0).
import "golang.org/x/sys/unix"

// GetCPUMem devolve memoria total e livre sem cgo.
//
// PORQUE ESTE FICHEIRO EXISTE: o unico `GetCPUMem` para darwin estava em
// `gpu_darwin.go`, que usa cgo (`#cgo CFLAGS: -x objective-c`). Com
// `CGO_ENABLED=0` esse ficheiro e' excluido pelo build, e o resultado era:
//
//	GOOS=darwin CGO_ENABLED=0 -> undefined: GetCPUMem
//
// Medido no HADES_PLATFORM_MATRIX.md (macOS = BLOCKED_EXTERNAL por esta causa),
// ou seja: a build macOS estava bloqueada por um DEFEITO NOSSO de portabilidade,
// nao por um recurso externo. Este ficheiro fecha isso lendo o mesmo dado por
// sysctl, sem cgo.
func GetCPUMem() (memInfo, error) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return memInfo{}, err
	}
	mem := memInfo{TotalMemory: total}
	// FreeMemory e' opcional: se o sysctl nao responder, devolve-se so' o total em
	// vez de falhar a descoberta inteira. O `FreeSwap` fica a zero porque o Darwin
	// usa paginacao dinamica (mesmo criterio do caminho com cgo).
	if pages, err := unix.SysctlUint64("vm.page_free_count"); err == nil {
		if pageSize, err := unix.SysctlUint64("vm.pagesize"); err == nil {
			mem.FreeMemory = pages * pageSize
		}
	}
	return mem, nil
}

// IsNUMA existe tambem no ficheiro com cgo; sem cgo o suporte a NUMA nao e'
// aplicavel em darwin (o upstream limita-o a Linux).
func IsNUMA() bool {
	return false
}
