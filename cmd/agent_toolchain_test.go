package cmd

// Testes do `agent toolchain` (FASE 28.7 + FASE 8).
//
// O que estes testes protegem, por ordem de importancia:
//   1. o contrato de exit code (0 pronto / 2 bloqueado / 1 erro) -- e' o que um script consome;
//   2. o `--json` ser UMA linha e ter as chaves declaradas;
//   3. um alvo desconhecido falhar em vez de inventar um plano;
//   4. `list` e `doctor` nunca fingirem sucesso.

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// comCapturaDeSaida troca `agentCLIOut` e o `--json` durante o teste, restaurando no fim.
//
// *Um teste que troca o stdout global corre contra os outros testes do pacote* -- por isso a
// variavel `agentCLIOut` existe em producao.
func comCapturaDeSaida(t *testing.T, jsonAtivo bool, corpo func()) string {
	t.Helper()
	var buf bytes.Buffer
	anteriorOut := agentCLIOut
	anteriorJSON := agentJSONOutput
	agentCLIOut = &buf
	agentJSONOutput = jsonAtivo
	t.Cleanup(func() {
		agentCLIOut = anteriorOut
		agentJSONOutput = anteriorJSON
	})
	corpo()
	return buf.String()
}

// `list` tem de listar os alvos que o catalogo declara, e o JSON tem de ser uma linha so'.
func TestToolchainListEmiteJSONDeUmaLinha(t *testing.T) {
	cmd := toolchainCommand()
	cmd.SetArgs([]string{"list"})
	var err error
	saida := comCapturaDeSaida(t, true, func() { err = cmd.Execute() })
	if err != nil {
		t.Fatalf("toolchain list: %v", err)
	}
	if strings.Count(strings.TrimRight(saida, "\n"), "\n") != 0 {
		t.Fatalf("--json emitiu mais de uma linha: %q", saida)
	}
	var listagem toolchainListagem
	if e := json.Unmarshal([]byte(saida), &listagem); e != nil {
		t.Fatalf("JSON invalido: %v (%q)", e, saida)
	}
	if len(listagem.Alvos) == 0 {
		t.Fatal("catalogo vazio: o resolvedor nao declara nenhum alvo")
	}
	nomes := map[string]bool{}
	for _, alvo := range listagem.Alvos {
		if strings.TrimSpace(alvo.Nome) == "" {
			t.Fatal("alvo sem nome na listagem")
		}
		if nomes[alvo.Nome] {
			t.Fatalf("alvo duplicado: %s", alvo.Nome)
		}
		nomes[alvo.Nome] = true
		if len(alvo.Exigencias) == 0 {
			t.Fatalf("alvo %s sem exigencias listadas", alvo.Nome)
		}
		for _, ex := range alvo.Exigencias {
			if !strings.Contains(ex, " -- ") {
				t.Fatalf("exigencia sem motivo legivel: %q", ex)
			}
		}
	}
}

// Um alvo desconhecido tem de sair 1 e nao produzir plano.
func TestToolchainAlvoDesconhecidoSaiComUm(t *testing.T) {
	cmd := toolchainCommand()
	cmd.SetArgs([]string{"plan", "blackberry-os"})
	var err error
	comCapturaDeSaida(t, true, func() { err = cmd.Execute() })
	if err == nil {
		t.Fatal("alvo desconhecido nao falhou")
	}
	if codigo := ExitCodeDoErro(err); codigo != exitToolchainErro {
		t.Fatalf("exit code = %d; esperado %d", codigo, exitToolchainErro)
	}
}

// O contrato de exit code: um alvo bloqueado sai 2, com relatorio valido.
//
// `ios` num host nao-macOS e' o caso deterministico: sempre bloqueado, e o bloqueio e' externo.
func TestToolchainAlvoBloqueadoSaiComDois(t *testing.T) {
	cmd := toolchainCommand()
	cmd.SetArgs([]string{"plan", "ios"})
	var err error
	var saida string = comCapturaDeSaida(t, true, func() { err = cmd.Execute() })
	if err == nil {
		t.Skip("este host e' macOS: ios nao esta bloqueado aqui")
	}
	if codigo := ExitCodeDoErro(err); codigo != exitToolchainBloqueado {
		t.Fatalf("exit code = %d; esperado %d (bloqueado). saida=%q", codigo, exitToolchainBloqueado, saida)
	}
	var relatorio toolchainRelatorio
	if e := json.Unmarshal([]byte(saida), &relatorio); e != nil {
		t.Fatalf("relatorio bloqueado nao e' JSON valido: %v (%q)", e, saida)
	}
	if relatorio.Pronto {
		t.Fatal("relatorio diz pronto mas o comando saiu bloqueado")
	}
	if relatorio.Veredicto != "bloqueado" {
		t.Fatalf("veredicto = %q; esperado bloqueado", relatorio.Veredicto)
	}
	if len(relatorio.Blockers) == 0 {
		t.Fatal("bloqueado sem blockers: o utilizador nao sabe o que falta")
	}
	if len(relatorio.Avanca) == 0 {
		t.Fatal("bloqueado nao diz o que ainda pode avancar")
	}
}

// `plan` de um alvo conhecido tem de emitir o schema completo, mesmo quando bloqueado.
func TestToolchainPlanTemSchemaCompleto(t *testing.T) {
	cmd := toolchainCommand()
	cmd.SetArgs([]string{"plan", "windows"})
	var err error
	saida := comCapturaDeSaida(t, true, func() { err = cmd.Execute() })
	// Pode sair 0 ou 2 conforme as ferramentas do host; o schema e' o mesmo.
	if err != nil {
		if codigo := ExitCodeDoErro(err); codigo != exitToolchainBloqueado {
			t.Fatalf("erro inesperado: %v (exit %d)", err, codigo)
		}
	}
	var relatorio toolchainRelatorio
	if e := json.Unmarshal([]byte(saida), &relatorio); e != nil {
		t.Fatalf("JSON invalido: %v (%q)", e, saida)
	}
	if relatorio.Alvo != "windows" {
		t.Fatalf("alvo = %q; esperado windows", relatorio.Alvo)
	}
	if len(relatorio.Probes) == 0 {
		t.Fatal("plano sem probes: nao sondou nada")
	}
	for _, probe := range relatorio.Probes {
		if probe.Nome == "" {
			t.Fatal("probe sem nome")
		}
		if probe.Estado == "" {
			t.Fatalf("probe %s sem estado", probe.Nome)
		}
		if probe.Motivo == "" {
			t.Fatalf("probe %s sem motivo", probe.Nome)
		}
		if probe.Detalhe == "" {
			t.Fatalf("probe %s sem detalhe: o Mission Control nao pode mostrar um estado mudo", probe.Nome)
		}
	}
	if relatorio.Blockers == nil {
		t.Fatal("blockers nil: um consumidor JSON espera lista, nao null")
	}
	if relatorio.Avanca == nil {
		t.Fatal("trabalho_independente nil: um consumidor JSON espera lista, nao null")
	}
}

// `doctor` tem de reportar todos os alvos e um exit code coerente.
func TestToolchainDoctorCobreTodosOsAlvos(t *testing.T) {
	cmd := toolchainCommand()
	cmd.SetArgs([]string{"doctor"})
	var err error
	saida := comCapturaDeSaida(t, true, func() { err = cmd.Execute() })
	if err != nil {
		if codigo := ExitCodeDoErro(err); codigo != exitToolchainBloqueado {
			t.Fatalf("erro inesperado em doctor: %v (exit %d)", err, codigo)
		}
	}
	var saida2 struct {
		Alvos []struct {
			Alvo   string `json:"alvo"`
			Pronto bool   `json:"pronto"`
			Falta  int    `json:"exigencias_bloqueantes"`
		} `json:"alvos"`
	}
	if e := json.Unmarshal([]byte(saida), &saida2); e != nil {
		t.Fatalf("JSON invalido: %v (%q)", e, saida)
	}
	if len(saida2.Alvos) == 0 {
		t.Fatal("doctor nao reportou nenhum alvo")
	}
	// Tres estados, nao dois: pronto, bloqueado (falta ferramenta obrigatoria) e incompleto
	// (ferramentas presentes mas sem health funcional).
	//
	// *Falta de prova nao e' prova de bloqueio*: um alvo com `Falta == 0` e `Pronto == false`
	// esta' INCOMPLETO, e chama-lo bloqueado faria o utilizador procurar o que ja' tem.
	prontos, bloqueados, incompletos := 0, 0, 0
	for _, alvo := range saida2.Alvos {
		switch {
		case alvo.Pronto && alvo.Falta == 0:
			prontos++
		case !alvo.Pronto && alvo.Falta > 0:
			bloqueados++
		case !alvo.Pronto && alvo.Falta == 0:
			incompletos++
		default:
			t.Fatalf("alvo %s: estado incoerente (pronto=%v falta=%d)", alvo.Alvo, alvo.Pronto, alvo.Falta)
		}
	}
	if prontos+bloqueados+incompletos != len(saida2.Alvos) {
		t.Fatal("contagem inconsistente")
	}
	t.Logf("alvos: %d prontos, %d bloqueados, %d incompletos", prontos, bloqueados, incompletos)
}

// Sem `--json`, a saida e' indentada (legivel por humano), mas continua JSON valido.
func TestToolchainSemJSONIndentaMasContinuaValido(t *testing.T) {
	cmd := toolchainCommand()
	cmd.SetArgs([]string{"list"})
	var err error
	saida := comCapturaDeSaida(t, false, func() { err = cmd.Execute() })
	if err != nil {
		t.Fatalf("toolchain list: %v", err)
	}
	if !strings.Contains(saida, "\n  ") {
		t.Fatalf("saida sem indentacao: %q", saida)
	}
	var listagem toolchainListagem
	if e := json.Unmarshal([]byte(saida), &listagem); e != nil {
		t.Fatalf("saida indentada nao e' JSON valido: %v", e)
	}
}

// O comando tem de estar montado na arvore de `agent`, nao solto.
func TestToolchainEstaMontadoNoAgent(t *testing.T) {
	raiz := agentCommand()
	encontrado := false
	for _, sub := range raiz.Commands() {
		if sub.Name() == "toolchain" {
			encontrado = true
			break
		}
	}
	if !encontrado {
		t.Fatal("toolchain nao esta montado em `agent`")
	}
}

// O contexto cancelado nao pode inventar resultado: `plan` continua a devolver plano ou erro,
// nunca um relatorio vazio dado como pronto.
func TestToolchainContextoCanceladoNaoMente(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	rel, err := resolverAlvo(ctx, "windows")
	if err != nil {
		if codigo := ExitCodeDoErro(err); codigo != exitToolchainErro {
			t.Fatalf("erro inesperado: %v (exit %d)", err, codigo)
		}
		return
	}
	// Se devolveu relatorio, ele tem de ser coerente consigo mesmo.
	if rel.Pronto && len(rel.Blockers) != 0 {
		t.Fatal("relatorio pronto com blockers")
	}
	if !rel.Pronto && rel.Veredicto != "bloqueado" {
		t.Fatalf("veredicto = %q para plano nao pronto", rel.Veredicto)
	}
}
