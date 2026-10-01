//go:build linux

package agent

// strictSandboxLauncher is intentionally self-contained so it can run inside
// the chroot before the untrusted interpreter starts. It installs no_new_privs
// and a seccomp-BPF allowlist; every syscall not explicitly required by the
// small Python/Node runtime is killed, rather than allowed by default.
const strictSandboxLauncher = `import ctypes
import os
import platform
import resource
import sys

libc = ctypes.CDLL(None, use_errno=True)

class SockFilter(ctypes.Structure):
    _fields_ = [("code", ctypes.c_ushort), ("jt", ctypes.c_ubyte), ("jf", ctypes.c_ubyte), ("k", ctypes.c_uint)]

class SockFprog(ctypes.Structure):
    _fields_ = [("length", ctypes.c_ushort), ("filter", ctypes.POINTER(SockFilter))]

BPF_LD = 0x00
BPF_W = 0x00
BPF_ABS = 0x20
BPF_JMP = 0x05
BPF_JEQ = 0x10
BPF_K = 0x00
BPF_RET = 0x06
SECCOMP_SET_MODE_FILTER = 1
SECCOMP_RET_KILL_PROCESS = 0x80000000
PR_SET_NO_NEW_PRIVS = 38

# These are the syscalls needed by the restricted interpreter after the
# namespace/chroot setup has completed. The default action is KILL_PROCESS.
# Namespace, mount, ptrace, keyctl, bpf, module-loading, and socket syscalls
# are deliberately absent. The numbers are kernel ABI values, not libc ABI.
syscall_numbers = {
    "x86_64": {"arch": 0xc000003e, "seccomp": 317, "allow": [
        0, 1, 3, 5, 6, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
        21, 22, 23, 24, 25, 26, 27, 28, 32, 33, 35, 36, 37, 38, 39, 40,
        59, 60, 61, 62, 63, 72, 73, 74, 75, 77, 79, 80,
        83, 84, 85, 86, 87, 88, 89, 90, 91, 96, 97, 98, 99, 100, 102,
        104, 107, 108, 109, 110, 112, 113, 114, 115, 116, 117, 119, 121,
        122, 123, 124, 125, 126, 127, 128, 129, 130, 131, 132, 135, 140,
        141, 144, 145, 146, 147, 148, 157, 158, 186, 202, 203, 204, 206,
        207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219,
        220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232,
        233, 234, 257, 258, 259, 260, 261, 262, 263, 264, 265, 266, 267,
        268, 269, 270, 271, 273, 280, 281, 283, 284, 285, 286, 287, 288,
        289, 290, 291, 302, 318, 334,
    ]},
    "amd64": {"arch": 0xc000003e, "seccomp": 317, "allow": [
        0, 1, 3, 5, 6, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
        21, 22, 23, 24, 25, 26, 27, 28, 32, 33, 35, 36, 37, 38, 39, 40,
        59, 60, 61, 62, 63, 72, 73, 74, 75, 77, 79, 80,
        83, 84, 85, 86, 87, 88, 89, 90, 91, 96, 97, 98, 99, 100, 102,
        104, 107, 108, 109, 110, 112, 113, 114, 115, 116, 117, 119, 121,
        122, 123, 124, 125, 126, 127, 128, 129, 130, 131, 132, 135, 140,
        141, 144, 145, 146, 147, 148, 157, 158, 186, 202, 203, 204, 206,
        207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219,
        220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232,
        233, 234, 257, 258, 259, 260, 261, 262, 263, 264, 265, 266, 267,
        268, 269, 270, 271, 273, 280, 281, 283, 284, 285, 286, 287, 288,
        289, 290, 291, 302, 318, 334,
    ]},
    "aarch64": {"arch": 0xc00000b7, "seccomp": 277, "allow": [
        0, 1, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 20, 21,
        22, 23, 24, 25, 29, 32, 34, 35, 36, 39, 40, 43, 46, 48, 49, 56,
        57, 59, 62, 63, 64, 65, 66, 67, 68, 78, 79, 80, 82, 83, 93, 94,
        96, 97, 98, 101, 102, 103, 105, 113, 115, 124, 129, 134, 135, 139,
        147, 149, 154, 157, 158, 159, 160, 165, 167, 169, 172, 173, 174,
        175, 176, 177, 178, 179, 191, 214, 215, 216, 221, 222, 226, 227,
        232, 233, 260, 261, 278, 293,
    ]},
    "arm64": {"arch": 0xc00000b7, "seccomp": 277, "allow": [
        0, 1, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 20, 21,
        22, 23, 24, 25, 29, 32, 34, 35, 36, 39, 40, 43, 46, 48, 49, 56,
        57, 59, 62, 63, 64, 65, 66, 67, 68, 78, 79, 80, 82, 83, 93, 94,
        96, 97, 98, 101, 102, 103, 105, 113, 115, 124, 129, 134, 135, 139,
        147, 149, 154, 157, 158, 159, 160, 165, 167, 169, 172, 173, 174,
        175, 176, 177, 178, 179, 191, 214, 215, 216, 221, 222, 226, 227,
        232, 233, 260, 261, 278, 293,
    ]},
}
config = syscall_numbers.get(platform.machine())
if config is None:
    raise SystemExit("unsupported seccomp architecture")

# Apply process rlimits before installing seccomp and before exec'ing the
# untrusted interpreter. cgroup v2 remains the authoritative containment for
# fork count and memory; these limits provide an independent process-level
# ceiling and fail closed if the kernel refuses one.
for limit, value in [
    (resource.RLIMIT_CPU, (55, 55)),
    (resource.RLIMIT_AS, (524288 * 1024, 524288 * 1024)),
    (resource.RLIMIT_NPROC, (64, 64)),
    (resource.RLIMIT_NOFILE, (256, 256)),
    (resource.RLIMIT_FSIZE, (1048576, 1048576)),
]:
    resource.setrlimit(limit, value)

instructions = [
    SockFilter(BPF_LD | BPF_W | BPF_ABS, 0, 0, 4),
    SockFilter(BPF_JMP | BPF_JEQ | BPF_K, 1, 0, config["arch"]),
    SockFilter(BPF_RET | BPF_K, 0, 0, SECCOMP_RET_KILL_PROCESS),
    SockFilter(BPF_LD | BPF_W | BPF_ABS, 0, 0, 0),
]
for number in config["allow"]:
    instructions.append(SockFilter(BPF_JMP | BPF_JEQ | BPF_K, 0, 1, number))
    instructions.append(SockFilter(BPF_RET | BPF_K, 0, 0, 0x7fff0000))
instructions.append(SockFilter(BPF_RET | BPF_K, 0, 0, SECCOMP_RET_KILL_PROCESS))

array_type = SockFilter * len(instructions)
array = array_type(*instructions)
program = SockFprog(len(instructions), ctypes.cast(array, ctypes.POINTER(SockFilter)))
if libc.prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) != 0:
    raise OSError(ctypes.get_errno(), "PR_SET_NO_NEW_PRIVS failed")
if libc.syscall(config["seccomp"], SECCOMP_SET_MODE_FILTER, 0, ctypes.byref(program)) != 0:
    raise OSError(ctypes.get_errno(), "seccomp allowlist installation failed")

target = sys.argv[1]
arguments = sys.argv[2:]
if os.path.basename(target).startswith("python"):
    arguments = ["-I", "-S"] + arguments
os.execv(target, [target] + arguments)
`
