package agent

// FASE 28.4 secao WINDOWS: o provider do INSTALADOR.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// O E-1308 separou `windows` (binario) de `windows-installer` (instalador) porque o
// `ProviderDeBuildGo` so' corre `go build` e nunca invocaria o Inno Setup. Esta fatia da' ao
// `windows-installer` o provider que ele declarava nao ter.
//
// O QUE ESTE PROVIDER FAZ
//
// Corre o `iscc` (Inno Setup Compiler) sobre o script `.iss` do projecto e mede o `.exe` que ele
// produz. Nada mais: nao instala, nao assina, nao publica.
//
// O QUE ELE **NAO** FAZ
//
//  1. Nao assina. Um instalador Inno Setup nao assinado e' o requisito desta RC; assinar exige
//     Authenticode com chave propria e vive no `SigningProvider`.
//  2. Nao inventa o nome do instalador. O nome vem do `OutputBaseFilename` do `.iss`, que este
//     provider nao interpreta -- por isso mede o `.exe` que APARECEU no destino, e nao um nome
//     que ele supoe.
//     *Um artefacto cujo nome o provider adivinha e' um artefacto que o provider pode medir a si
//     mesmo.*
//  3. Nao apaga o destino. Um `dist` pode conter artefactos de builds anteriores e apaga-los seria
//     destrutivo; o que este provider garante e' que o artefacto devolvido e' NOVO (comparando a
//     listagem antes e depois).
//
// A DEPENDENCIA EXTERNA E' DECLARADA, NAO ESCONDIDA
//
// O `iscc` tem de estar no PATH do servidor. Se nao estiver, o `CapabilityPlan` ja' o reporta como
// `ABSENT` e o `Construir` recusa com `ErrPlanoIncompleto` -- nao ha' um erro obscuro de
// `exec: "iscc": executable file not found`.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AlvoDoInstaladorWindows e' o alvo do catalogo que este provider serve.
const AlvoDoInstaladorWindows = "windows-installer"

// TimeoutPorOmissaoDeInstalador limita a compilacao do instalador.
//
// O Inno Setup comprime o payload; e' mais lento do que um `go build`, e um limite curto
// transformaria um instalador grande num falso timeout.
const TimeoutPorOmissaoDeInstalador = 15 * time.Minute

// ProviderDeInstaladorWindows compila um instalador Inno Setup.
type ProviderDeInstaladorWindows struct{}

// Alvos declara o que sabe construir: so' o instalador Windows.
func (ProviderDeInstaladorWindows) Alvos() []string {
	return []string{AlvoDoInstaladorWindows}
}

// Construir corre o `iscc` sobre o `.iss` do projecto e devolve o `.exe` produzido.
func (p ProviderDeInstaladorWindows) Construir(
	ctx context.Context,
	plano CapabilityPlan,
	opcoes OpcoesDeBuild,
) (ResultadoDeBuild, error) {
	// Um plano com ferramentas obrigatorias ausentes nao compila um instalador. Mesma regra do
	// provider de binarios: recusa por BLOCKERS reais, e nomeia-os.
	if len(plano.Blockers) != 0 {
		return ResultadoDeBuild{}, fmt.Errorf("%w: %s",
			ErrPlanoIncompleto, strings.Join(plano.MotivosDeNaoPronto(), "; "))
	}
	if !strings.EqualFold(strings.TrimSpace(plano.Alvo), AlvoDoInstaladorWindows) {
		return ResultadoDeBuild{}, fmt.Errorf("%w: %q (provider cobre %s)",
			ErrSemProviderParaAlvo, plano.Alvo, AlvoDoInstaladorWindows)
	}

	raiz := strings.TrimSpace(opcoes.Raiz)
	if raiz == "" {
		return ResultadoDeBuild{}, errors.New("raiz do projection vazia: nao ha' instalador a compilar")
	}
	raizAbs, err := filepath.Abs(raiz)
	if err != nil {
		return ResultadoDeBuild{}, fmt.Errorf("resolver raiz %q: %w", raiz, err)
	}
	if info, errStat := os.Stat(raizAbs); errStat != nil {
		return ResultadoDeBuild{}, fmt.Errorf("raiz %q: %w", raizAbs, errStat)
	} else if !info.IsDir() {
		return ResultadoDeBuild{}, fmt.Errorf("raiz %q nao e' um directorio", raizAbs)
	}

	// O script `.iss`. Se houver mais de um, o chamador tem de dizer qual: escolher o primeiro
	// por ordem alfabetica seria uma decisao do provider sobre o conteudo do utilizador.
	scripts, errScripts := scriptsInno(raizAbs)
	if errScripts != nil {
		return ResultadoDeBuild{}, errScripts
	}
	if len(scripts) == 0 {
		return ResultadoDeBuild{}, fmt.Errorf(
			"%s nao tem nenhum ficheiro .iss: nao e' um projection de instalador Inno Setup", raizAbs)
	}
	if len(scripts) > 1 {
		return ResultadoDeBuild{}, fmt.Errorf(
			"%s tem %d ficheiros .iss (%s): escolha um com `destino` ou reduza a um",
			raizAbs, len(scripts), strings.Join(scripts, ", "))
	}
	script := scripts[0]

	destino := strings.TrimSpace(opcoes.Destino)
	if destino == "" {
		destino = filepath.Join(raizAbs, "dist")
	}
	destinoAbs, errDestino := filepath.Abs(destino)
	if errDestino != nil {
		return ResultadoDeBuild{}, fmt.Errorf("resolver destino %q: %w", destino, errDestino)
	}

	args := []string{"/Q", "/O" + destinoAbs, script}

	resultado := ResultadoDeBuild{}
	if opcoes.DryRun {
		resultado.Explicacao = fmt.Sprintf("dry-run: iscc %s (destino %s)", strings.Join(args, " "), destinoAbs)
		return resultado, nil
	}

	if errMkdir := os.MkdirAll(destinoAbs, 0o700); errMkdir != nil {
		return ResultadoDeBuild{}, fmt.Errorf("criar destino %q: %w", destinoAbs, errMkdir)
	}

	antes, _ := executaveisNoDirectorio(destinoAbs)

	timeout := opcoes.Timeout
	if timeout <= 0 {
		timeout = TimeoutPorOmissaoDeInstalador
	}
	ctxBuild, cancelar := context.WithTimeout(ctx, timeout)
	defer cancelar()

	cmd := exec.CommandContext(ctxBuild, "iscc", args...)
	cmd.Dir = raizAbs
	inicio := time.Now()
	saida, errExec := cmd.CombinedOutput()
	duracao := time.Since(inicio)
	if errExec != nil {
		// A saida REAL do compilador, truncada: um erro de instalador sem a saida obriga quem le'
		// a correr o comando a mao.
		return ResultadoDeBuild{}, fmt.Errorf("iscc falhou (%v): %s", errExec, truncarSaida(string(saida)))
	}

	// O artefacto e' o `.exe` que APARECEU. Nao se adivinha o `OutputBaseFilename`.
	depois, errLista := executaveisNoDirectorio(destinoAbs)
	if errLista != nil {
		return ResultadoDeBuild{}, fmt.Errorf("listar destino %q: %w", destinoAbs, errLista)
	}
	novos := diferencaDeListas(depois, antes)
	if len(novos) == 0 {
		return ResultadoDeBuild{}, fmt.Errorf(
			"iscc reportou sucesso mas nenhum .exe novo apareceu em %q (saida: %s)",
			destinoAbs, truncarSaida(string(saida)))
	}
	sort.Strings(novos)
	caminhoFinal := filepath.Join(destinoAbs, novos[0])

	info, errStat := os.Stat(caminhoFinal)
	if errStat != nil {
		return ResultadoDeBuild{}, fmt.Errorf("o instalador %q desapareceu depois de medido: %w",
			caminhoFinal, errStat)
	}
	checksum, errHash := sha256DeFicheiro(caminhoFinal)
	if errHash != nil {
		return ResultadoDeBuild{}, fmt.Errorf("checksum de %q: %w", caminhoFinal, errHash)
	}

	resultado.Artefatos = append(resultado.Artefatos, Artefato{
		Caminho:        caminhoFinal,
		Nome:           novos[0],
		TamanhoBytes:   info.Size(),
		SHA256:         checksum,
		Alvo:           AlvoDoInstaladorWindows,
		PidDoProcesso:  cmd.ProcessState.Pid(),
		DuracaoMS:      duracao.Milliseconds(),
		SaidaDoComando: truncarSaida(string(saida)),
	})
	resultado.Explicacao = fmt.Sprintf(
		"1 instalador produzido (alvo %s, %d bytes, sha256 %s) -- NAO assinado",
		AlvoDoInstaladorWindows, info.Size(), checksum[:12])
	return resultado, nil
}

// scriptsInno devolve os ficheiros `.iss` do directorio, ordenados.
func scriptsInno(raiz string) ([]string, error) {
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		return nil, fmt.Errorf("ler %q: %w", raiz, err)
	}
	nomes := make([]string, 0, 2)
	for _, entrada := range entradas {
		if entrada.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entrada.Name()), ".iss") {
			nomes = append(nomes, entrada.Name())
		}
	}
	sort.Strings(nomes)
	return nomes, nil
}

// executaveisNoDirectorio lista os `.exe` do directorio. Directorio inexistente devolve vazio.
func executaveisNoDirectorio(dir string) ([]string, error) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	nomes := make([]string, 0, 4)
	for _, entrada := range entradas {
		if entrada.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entrada.Name()), ".exe") {
			nomes = append(nomes, entrada.Name())
		}
	}
	return nomes, nil
}

// diferencaDeListas devolve o que esta' em `depois` e nao em `antes`.
func diferencaDeListas(depois, antes []string) []string {
	vistos := make(map[string]bool, len(antes))
	for _, nome := range antes {
		vistos[nome] = true
	}
	novos := make([]string, 0, len(depois))
	for _, nome := range depois {
		if !vistos[nome] {
			novos = append(novos, nome)
		}
	}
	return novos
}
