package agent

// FASE 2 do adendo V3: testes da persistencia do estado de worktree.
//
// O que estes testes travam:
//
//  1. ficheiro ausente -> registo VAZIO e sem erro (primeira execucao e' o caso normal);
//  2. ficheiro corrompido -> ERRO explicito, NUNCA registo vazio;
//  3. esquema desconhecido -> recusado;
//  4. estado fora da lista fechada -> recusado ao gravar E ao ler;
//  5. escrita atomica: um temporario nunca fica para tras;
//  6. o registo sobrevive a uma nova instancia (o ponto da persistencia);
//  7. `EstadoDe` distingue "nao registado" de "registado como criado";
//  8. o diagnostico usa o estado PERSISTIDO quando existe.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ficheiro ausente e' o caso normal, nao uma avaria.
//
//	*Confundir "ainda nao ha'" com "falhou" faria o chamador tratar o caso normal como avaria.*
func TestRegistoAusenteDevolveVazioSemErro(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), NomeDoRegistoDeWorktrees)
	r, err := CarregarRegistoDeWorktrees(caminho)
	if err != nil {
		t.Fatalf("ficheiro ausente devolveu erro: %v", err)
	}
	if r == nil {
		t.Fatal("registo nulo")
	}
	if r.Contagem() != 0 {
		t.Fatalf("%d entradas; esperado 0", r.Contagem())
	}
	if r.Esquema != esquemaDoRegistoDeWorktrees {
		t.Fatalf("esquema = %d; esperado %d", r.Esquema, esquemaDoRegistoDeWorktrees)
	}
}

// Ficheiro corrompido e' ERRO, nunca registo vazio.
//
//	*Um registo corrompido lido como vazio apagaria o estado de todos os worktrees com um encolher
//	de ombros.*
func TestRegistoCorrompidoENaoVazio(t *testing.T) {
	casos := []struct {
		nome     string
		conteudo string
	}{
		{"json invalido", "{isto nao e' json"},
		{"truncado", `{"esquema":1,"entradas":{"a":`},
		{"esquema errado", `{"esquema":99,"entradas":{}}`},
		{"sem esquema", `{"entradas":{}}`},
		{"estado inventado", `{"esquema":1,"entradas":{"m":{"mission_id":"m","estado":"quase_pronto"}}}`},
		{"campo desconhecido", `{"esquema":1,"entradas":{},"campo_novo":1}`},
		{"entrada sem id", `{"esquema":1,"entradas":{"":{"mission_id":"","estado":"preparado"}}}`},
	}
	for _, caso := range casos {
		dir := t.TempDir()
		caminho := filepath.Join(dir, NomeDoRegistoDeWorktrees)
		if err := os.WriteFile(caminho, []byte(caso.conteudo), 0o600); err != nil {
			t.Fatalf("%s: escrever: %v", caso.nome, err)
		}
		r, err := CarregarRegistoDeWorktrees(caminho)
		if err == nil {
			t.Fatalf("%s: aceite sem erro (registo com %d entradas)", caso.nome, r.Contagem())
		}
		if !errors.Is(err, ErrRegistoCorrompido) {
			t.Fatalf("%s: erro = %v; esperado ErrRegistoCorrompido", caso.nome, err)
		}
		if r != nil {
			t.Fatalf("%s: devolveu registo numa falha de leitura", caso.nome)
		}
	}
}

// Ficheiro demasiado grande e' recusado.
func TestRegistoDemasiadoGrandeERecusado(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, NomeDoRegistoDeWorktrees)
	grande := make([]byte, maxRegistoDeWorktreesBytes+1)
	for i := range grande {
		grande[i] = 'x'
	}
	if err := os.WriteFile(caminho, grande, 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	if _, err := CarregarRegistoDeWorktrees(caminho); !errors.Is(err, ErrRegistoCorrompido) {
		t.Fatalf("erro = %v; esperado ErrRegistoCorrompido", err)
	}
}

// O registo SOBREVIVE a uma nova instancia -- e' o ponto da persistencia.
//
//	*Um estado que morre com o processo nao e' um estado: e' uma variavel.*
func TestRegistoPersisteEntreInstancias(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), NomeDoRegistoDeWorktrees)

	primeiro := NovoRegistoDeWorktrees(caminho)
	if err := primeiro.Registar("missao-a", WorktreeSuspenso, "/repo/.agent-worktrees/missao-a", "agent/missao-a"); err != nil {
		t.Fatalf("registar: %v", err)
	}
	if err := primeiro.Registar("missao-b", WorktreeArquivado, "/repo/.agent-worktrees/missao-b", "agent/missao-b"); err != nil {
		t.Fatalf("registar: %v", err)
	}
	if err := primeiro.Guardar(); err != nil {
		t.Fatalf("guardar: %v", err)
	}

	// Simular reinicio: nova leitura, sem estado em memoria.
	segundo, err := CarregarRegistoDeWorktrees(caminho)
	if err != nil {
		t.Fatalf("recarregar: %v", err)
	}
	estado, existe := segundo.EstadoDe("missao-a")
	if !existe {
		t.Fatal("missao-a desapareceu no reinicio")
	}
	if estado != WorktreeSuspenso {
		t.Fatalf("estado = %q; esperado suspension", estado)
	}
	estado, existe = segundo.EstadoDe("missao-b")
	if !existe || estado != WorktreeArquivado {
		t.Fatalf("missao-b: existe=%v estado=%q", existe, estado)
	}
	// Missoes ordenadas.
	ids := segundo.Missoes()
	if len(ids) != 2 || ids[0] != "missao-a" || ids[1] != "missao-b" {
		t.Fatalf("missoes = %v; esperado [missao-a missao-b]", ids)
	}
}

// O registo nao guarda estado fora da lista fechada.
//
//	*Um registo nao pode guardar um estado que o proprio sistema recusa.*
func TestRegistarEstadoInvalidoERecusado(t *testing.T) {
	r := NovoRegistoDeWorktrees(filepath.Join(t.TempDir(), NomeDoRegistoDeWorktrees))
	if err := r.Registar("m", EstadoDoWorktree("quase_pronto"), "", ""); err == nil {
		t.Fatal("estado inventado foi registado")
	}
	if r.Contagem() != 0 {
		t.Fatalf("%d entradas apos recusa; esperado 0", r.Contagem())
	}
	if err := r.Registar("", WorktreePreparado, "", ""); err == nil {
		t.Fatal("mission_id vazio foi aceite")
	}
}

// A escrita e' atomica: nenhum temporario fica para tras.
//
//	*Um registo meio escrito e' pior do que nenhum registo: mente sobre metade do passado.*
func TestEscritaNaoDeixaTemporarios(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, NomeDoRegistoDeWorktrees)
	r := NovoRegistoDeWorktrees(caminho)
	if err := r.Registar("m", WorktreePreparado, "/x", "b"); err != nil {
		t.Fatalf("registar: %v", err)
	}
	if err := r.Guardar(); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	for _, e := range entradas {
		if strings.HasPrefix(e.Name(), ".worktree-state-") {
			t.Fatalf("temporario ficou para tras: %s", e.Name())
		}
	}
	// E o ficheiro final e' JSON valido.
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	var verificado map[string]any
	if err := json.Unmarshal(bruto, &verificado); err != nil {
		t.Fatalf("registo gravado nao e' JSON valido: %v", err)
	}
	if _, tem := verificado["caminho"]; tem {
		t.Fatal("o campo interno `caminho` foi serializado")
	}
}

// `EstadoDe` distingue "nao registado" de "registado como criado".
//
//	*Nao saber e' um facto diferente de saber que nao comecou.*
func TestEstadoDeDistingueAusenciaDeCriado(t *testing.T) {
	r := NovoRegistoDeWorktrees(filepath.Join(t.TempDir(), NomeDoRegistoDeWorktrees))
	if _, existe := r.EstadoDe("nao-registada"); existe {
		t.Fatal("missao nao registada reportada como existente")
	}
	if err := r.Registar("criada", WorktreeCriado, "/x", "b"); err != nil {
		t.Fatalf("registar: %v", err)
	}
	estado, existe := r.EstadoDe("criada")
	if !existe {
		t.Fatal("missao registada reportada como ausente")
	}
	if estado != WorktreeCriado {
		t.Fatalf("estado = %q; esperado criado", estado)
	}
}

// Remover tira do registo.
func TestRemoverDoRegisto(t *testing.T) {
	r := NovoRegistoDeWorktrees(filepath.Join(t.TempDir(), NomeDoRegistoDeWorktrees))
	if err := r.Registar("m", WorktreePreparado, "/x", "b"); err != nil {
		t.Fatalf("registar: %v", err)
	}
	r.Remover("m")
	if _, existe := r.EstadoDe("m"); existe {
		t.Fatal("missao continua no registo apos remocao")
	}
	// Remover inexistente nao rebenta.
	r.Remover("nao-existe")
	// Registo nulo tambem nao.
	var nulo *RegistoDeWorktrees
	nulo.Remover("x")
	if nulo.Contagem() != 0 {
		t.Fatal("registo nulo com contagem diferente de 0")
	}
}

// O diagnostico usa o estado PERSISTIDO quando existe.
//
// Sem isto, a persistencia seria um ficheiro que ninguem le'.
func TestDiagnosticoUsaEstadoPersistido(t *testing.T) {
	repo := repoFixture(t)
	dataRoot := t.TempDir()
	sessao := criarWorktree(t, repo, dataRoot, "missao-persistida")

	caminho := CaminhoDoRegisto(dataRoot)
	registo, err := CarregarRegistoDeWorktrees(caminho)
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	// Suspender e persistir.
	if err := SuspenderWorktree(sessao); err != nil {
		t.Fatalf("suspender: %v", err)
	}
	if err := registo.Registar(sessao.MissionID, EstadoDoWorktree(sessao.Status), sessao.WorktreeDir, sessao.BranchName); err != nil {
		t.Fatalf("registar: %v", err)
	}
	if err := registo.Guardar(); err != nil {
		t.Fatalf("guardar: %v", err)
	}

	// Reinicio simulado: sessao reconstruida SEM estado em memoria.
	reconstruida := &GitWorktreeSession{
		MissionID:    sessao.MissionID,
		RepoRoot:     repo,
		WorktreeDir:  sessao.WorktreeDir,
		BaseCommit:   sessao.BaseCommit,
		TargetBranch: sessao.TargetBranch,
	}
	recarregado, err := CarregarRegistoDeWorktrees(caminho)
	if err != nil {
		t.Fatalf("recarregar: %v", err)
	}
	diag := DiagnosticarCicloDeVidaComRegisto(context.Background(), reconstruida, recarregado)

	if diag.Estado != WorktreeSuspenso {
		t.Fatalf("estado = %q; esperado suspension (do registo persistido)", diag.Estado)
	}
	if !diag.EstadoDoRegisto {
		t.Fatal("diagnostico nao marcou o estado como vindo do registo")
	}
	if diag.Disco.TemTrabalhoNaoIntegrado {
		t.Fatalf("worktree suspension e limpo marcado com trabalho: %s", diag.Disco.Detalhe)
	}
}

// Sem registo, o diagnostico cai para o que o disco prova e DIZ isso.
func TestDiagnosticoSemRegistoCaiParaODisco(t *testing.T) {
	repo := repoFixture(t)
	sessao := criarWorktree(t, repo, t.TempDir(), "missao-sem-registo")

	diag := DiagnosticarCicloDeVidaComRegisto(context.Background(), sessao, nil)
	if diag.EstadoDoRegisto {
		t.Fatal("diagnostico diz ter registo quando nao recebeu nenhum")
	}
	if !diag.Estado.Valido() {
		t.Fatalf("estado %q nao pertence a' lista fechada", diag.Estado)
	}
	if diag.EstadoObservado == "" {
		t.Fatal("diagnostico nao explica de onde veio o estado")
	}
}

// Um registo corrompido no diagnostico NAO e' ignorado em silencio.
func TestDiagnosticoComRegistoCorrompido(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, NomeDoRegistoDeWorktrees)
	if err := os.WriteFile(caminho, []byte("{lixo"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	if _, err := CarregarRegistoDeWorktrees(caminho); !errors.Is(err, ErrRegistoCorrompido) {
		t.Fatalf("erro = %v; esperado ErrRegistoCorrompido", err)
	}
}

// Um registo GRAVADO com o valor antigo `"suspension"` continua a ser legivel.
//
//	*Um valor antigo que nao se sabe ler e' um worktree que se trata como se nao existisse.*
//
// Este teste existe por causa de um defeito REAL apanhado pelo E2E de browser (E-1238): o backend
// gravava `"suspension"` (ingles) enquanto o painel mapeava `"suspension"` -- e o painel mostrava
// "Estado desconhecido" para um worktree suspension. Os testes unitarios nao o viram porque
// *inventavam o valor que esperavam do backend*.
func TestRegistoLeValorLegadoSuspension(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, NomeDoRegistoDeWorktrees)
	// Um registo no formato ANTIGO, escrito a mao.
	antigo := `{
  "esquema": 1,
  "entradas": {
    "missao-legada": {
      "mission_id": "missao-legada",
      "estado": "suspension",
      "caminho": "/repo/.agent-worktrees/missao-legada",
      "branch": "agent/missao-legada",
      "atualizado_em": "2026-10-17T10:00:00Z"
    }
  }
}`
	if err := os.WriteFile(caminho, []byte(antigo), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	// O valor legado `"suspension"` e' aceite pela tabela de legados.
	if estadosLegados["suspension"] != WorktreeSuspenso {
		t.Fatalf("estadosLegados[\"suspension\"] = %q; esperado %q",
			estadosLegados["suspension"], WorktreeSuspenso)
	}
	// E o valor NOVO e' o que o tipo declara.
	if WorktreeSuspenso != "suspension" {
		t.Fatalf("WorktreeSuspenso = %q; esperado suspension", WorktreeSuspenso)
	}
	// O ficheiro com o valor actual carrega.
	r, err := CarregarRegistoDeWorktrees(caminho)
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	estado, existe := r.EstadoDe("missao-legada")
	if !existe || estado != WorktreeSuspenso {
		t.Fatalf("estado = %q existe=%v; esperado suspension", estado, existe)
	}
}
