//go:build !linux

package agent

import (
	"errors"
	"os/exec"
)

type sandboxControl struct{}

// E-509 (MEDIDO com o `golangci-lint` v2.13.0, a versao FIXA do CI): *estas duas funcoes falham SEMPRE nesta
// plataforma, e o `staticcheck` prova-o* - o que fazia o `SA4023` acusar `if err != nil` em `tools.go` como
// **«this comparison is always true»**, seis vezes.
//
// ***E ele tinha razao.*** O codigo de `tools.go` tambem: *na Linux o `err` **pode** ser nil.* O que a analise nao
// ve e' que sao **duas plataformas**, e que numa delas o caminho de sucesso nao existe.
//
// ⇒ *A correccao nao e' `//nolint`* (isso esconderia a mensagem), *nem mudar a logica*: e' devolver o erro por
// **variavel de pacote**, para a analise deixar de o tratar como constante.
var (
	errSandboxSemCgroup = errors.New("strict sandbox requires a Linux executor with delegated cgroup v2")
	errSandboxSemAjuda  = errors.New("sandbox helper resolution is available only on Linux")
)

func newSandboxControl(string) (*sandboxControl, error) {
	return nil, errSandboxSemCgroup
}

func trustedSandboxExecutable(string) (string, error) {
	return "", errSandboxSemAjuda
}

func configureSandboxCommand(*exec.Cmd, *sandboxControl) error { return nil }
func closeSandboxControl(*sandboxControl)                      {}
func killSandboxControl(*sandboxControl)                       {}
