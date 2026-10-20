package agent

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func metaDeTeste(escopo EscopoDeMemoria) MetaDaNota {
	return MetaDaNota{
		MemoryID:   "mem-0001",
		Escopo:     escopo,
		Project:    "hades",
		Author:     "dz23",
		CreatedAt:  "2026-10-19T00:00:00Z",
		UpdatedAt:  "2026-10-19T00:00:00Z",
		Confidence: "alta",
	}
}

func fatoDeTeste() Memoria {
	return Memoria{
		ID:       "mem-0001",
		Rotulo:   MemoriaFatoVerificado,
		Conteudo: "E-1433 mediu 129 PASS na familia de memoria.",
		Fonte:    "go test ./internal/agent -run Memor",
	}
}

// Sem escopo nao ha nota. E' a negacao por defeito: nao existe escopo "implicito" a inferir do caminho.
func TestNotaDeMemoriaRecusaSemEscopo(t *testing.T) {
	_, err := MontarNota(metaDeTeste(""), fatoDeTeste())
	if !errors.Is(err, ErrEscopoAusente) {
		t.Fatalf("esperava ErrEscopoAusente, veio %v", err)
	}
}

// Escopo fora da lista fechada nao e' aceito por ser "parecido".
func TestNotaDeMemoriaRecusaEscopoDesconhecido(t *testing.T) {
	_, err := MontarNota(metaDeTeste("PROJETO"), fatoDeTeste())
	if !errors.Is(err, ErrEscopoDesconhecido) {
		t.Fatalf("esperava ErrEscopoDesconhecido, veio %v", err)
	}
}

// O adaptador NAO tem regra de rotulo propria: quem recusa FACT_VERIFIED sem fonte e' `Memoria.Validar`.
// Se este teste passar a falhar, alguem duplicou a regra e as duas podem divergir.
func TestNotaDeMemoriaDefereValidacaoDoRotulo(t *testing.T) {
	m := fatoDeTeste()
	m.Fonte = ""
	_, err := MontarNota(metaDeTeste(EscopoProject), m)
	if !errors.Is(err, ErrPromocaoSemProva) {
		t.Fatalf("esperava ErrPromocaoSemProva (de Memoria.Validar), veio %v", err)
	}
}

func TestNotaDeMemoriaRoundTrip(t *testing.T) {
	original := fatoDeTeste()
	meta := metaDeTeste(EscopoProject)

	dados, err := MontarNota(meta, original)
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}

	meta2, volta, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	if volta.ID != original.ID || volta.Rotulo != original.Rotulo || volta.Fonte != original.Fonte {
		t.Fatalf("identidade perdida: %#v", volta)
	}
	if strings.TrimSpace(volta.Conteudo) != strings.TrimSpace(original.Conteudo) {
		t.Fatalf("conteudo perdido: %q", volta.Conteudo)
	}
	if meta2.Escopo != meta.Escopo {
		t.Fatalf("escopo perdido: %q", meta2.Escopo)
	}
	if meta2.ContentHash != HashDoConteudo(original.Conteudo) {
		t.Fatalf("content_hash nao corresponde ao corpo: %q", meta2.ContentHash)
	}
}

// Determinismo: mesma entrada, mesmos bytes. Se isto quebrar, o hash da nota deixa de significar alguma coisa.
func TestMontarNotaEhDeterminista(t *testing.T) {
	a, err := MontarNota(metaDeTeste(EscopoGoal), fatoDeTeste())
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	b, err := MontarNota(metaDeTeste(EscopoGoal), fatoDeTeste())
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("duas montagens da mesma entrada deram bytes diferentes")
	}
}

// A discriminacao que interessa: reanexar a MESMA memory_id nao acrescenta bytes nem os altera.
func TestAnexoIdempotenteNaoDuplicaNemReescreve(t *testing.T) {
	meta := metaDeTeste(EscopoProject)
	m := fatoDeTeste()

	primeiro, anexou, err := AnexarNotaIdempotente(nil, meta, m)
	if err != nil || !anexou {
		t.Fatalf("primeira anexacao devia ter anexado: anexou=%v err=%v", anexou, err)
	}

	// Mesma chave, corpo DIFERENTE: a idempotencia nao pode depender do corpo.
	m2 := m
	m2.Conteudo = "outro texto completamente diferente"

	segundo, anexou2, err := AnexarNotaIdempotente(primeiro, meta, m2)
	if err != nil {
		t.Fatalf("segunda anexacao: %v", err)
	}
	if anexou2 {
		t.Fatalf("reanexar a mesma memory_id nao podia anexar")
	}
	if !bytes.Equal(primeiro, segundo) {
		t.Fatalf("reanexar a mesma memory_id alterou os bytes")
	}
	if n := strings.Count(string(segundo), "memory_id: mem-0001"); n != 1 {
		t.Fatalf("esperava 1 ocorrencia da memory_id, encontrei %d", n)
	}
}

// memory_id diferente anexa de verdade -- senao o teste acima passaria por o append estar sempre quebrado.
func TestAnexoDeIdDiferenteAcrescenta(t *testing.T) {
	primeiro, _, err := AnexarNotaIdempotente(nil, metaDeTeste(EscopoProject), fatoDeTeste())
	if err != nil {
		t.Fatalf("preparacao: %v", err)
	}
	outra := fatoDeTeste()
	outra.ID = "mem-0002"
	meta := metaDeTeste(EscopoProject)
	meta.MemoryID = "mem-0002"

	segundo, anexou, err := AnexarNotaIdempotente(primeiro, meta, outra)
	if err != nil || !anexou {
		t.Fatalf("nova memory_id devia anexar: anexou=%v err=%v", anexou, err)
	}
	if len(segundo) <= len(primeiro) {
		t.Fatalf("os bytes nao cresceram")
	}
}

// Campo fora do schema e' recusado. Um campo ignorado em silencio e' um campo que o leitor julga ter lido.
func TestLerNotaRecusaCampoDesconhecido(t *testing.T) {
	dados := []byte("---\nmemory_id: mem-0001\nschema_version: 1\nscope: PROJECT\ncampo_inventado: x\n---\ncorpo\n")
	_, _, err := LerNota(dados)
	if !errors.Is(err, ErrCampoDesconhecido) {
		t.Fatalf("esperava ErrCampoDesconhecido, veio %v", err)
	}
}

func TestLerNotaRecusaFrontmatterAusente(t *testing.T) {
	_, _, err := LerNota([]byte("texto solto sem cabecalho\n"))
	if !errors.Is(err, ErrNotaSemFrontmatter) {
		t.Fatalf("esperava ErrNotaSemFrontmatter, veio %v", err)
	}
}

// author e' quem escreveu a nota; aprovador e' quem assinou a decisao. Se o autor virasse aprovador na leitura,
// uma decisao sem assinatura ganharia assinatura ao passar pelo disco.
func TestAutorNaoViraAprovador(t *testing.T) {
	meta := metaDeTeste(EscopoProject)
	meta.Author = "quem-escreveu"
	m := Memoria{ID: "mem-0002", Rotulo: MemoriaProposta, Conteudo: "proposta sem aprovador"}
	dados, err := MontarNota(meta, m)
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	_, volta, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	if volta.Aprovador != "" {
		t.Fatalf("o autor da nota virou aprovador da decisao: %q", volta.Aprovador)
	}
}

// O hash gravado tem de bater com o corpo lido. Se nao batesse (por causa do "\n" final), toda a leitura
// pareceria uma edicao humana -- um alarme que toca sempre nao avisa de nada.
func TestIntegridadeDaNotaAceitaLeituraLimpa(t *testing.T) {
	dados, err := MontarNota(metaDeTeste(EscopoProject), fatoDeTeste())
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	meta, m, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	if err := IntegridadeDaNota(meta, m); err != nil {
		t.Fatalf("leitura limpa acusada de adulteracao: %v", err)
	}
}

// Edicao humana no corpo tem de ser DETETADA -- e nao apagada.
func TestIntegridadeDaNotaDetectaEdicaoHumana(t *testing.T) {
	dados, err := MontarNota(metaDeTeste(EscopoProject), fatoDeTeste())
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	meta, m, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	m.Conteudo = m.Conteudo + "linha acrescentada a mao no Obsidian\n"
	if err := IntegridadeDaNota(meta, m); !errors.Is(err, ErrConteudoAlterado) {
		t.Fatalf("esperava ErrConteudoAlterado, veio %v", err)
	}
}

// Nota sem content_hash nao pode passar por integra: ausencia de prova nao e' prova.
func TestIntegridadeDaNotaRecusaSemHash(t *testing.T) {
	meta := metaDeTeste(EscopoProject)
	meta.ContentHash = ""
	if err := IntegridadeDaNota(meta, fatoDeTeste()); !errors.Is(err, ErrConteudoAlterado) {
		t.Fatalf("esperava ErrConteudoAlterado, veio %v", err)
	}
}

// A versao conta OCORRENCIAS, nao notas: e' isso que torna a corrida visivel.
func TestVersaoDaMemoriaContaOcorrencias(t *testing.T) {
	if v := VersaoDaMemoria(nil, "mem-0001"); v != 0 {
		t.Fatalf("ficheiro vazio devia dar versao 0, deu %d", v)
	}
	a, _, err := AnexarNotaIdempotente(nil, metaDeTeste(EscopoProject), fatoDeTeste())
	if err != nil {
		t.Fatalf("preparacao: %v", err)
	}
	if v := VersaoDaMemoria(a, "mem-0001"); v != 1 {
		t.Fatalf("esperava versao 1, deu %d", v)
	}
}

// O caso que a idempotencia NAO cobre: dois escritores leram "ausente" e ambos tentam escrever.
// O segundo tem de receber conflito -- nao pode ganhar em silencio.
func TestAnexoComVersaoRecusaConflito(t *testing.T) {
	meta := metaDeTeste(EscopoProject)
	m := fatoDeTeste()

	primeiro, err := AnexarComVersao(nil, meta, m, 0)
	if err != nil {
		t.Fatalf("primeira escrita devia passar na versao 0: %v", err)
	}

	// Segundo escritor com a MESMA leitura (versao 0): conflito explicito, bytes intactos.
	segundo, err := AnexarComVersao(primeiro, meta, m, 0)
	if !errors.Is(err, ErrVersaoDivergente) {
		t.Fatalf("esperava ErrVersaoDivergente, veio %v", err)
	}
	if !bytes.Equal(primeiro, segundo) {
		t.Fatalf("a recusa de conflito alterou os bytes")
	}
}

// Quem releu a versao corrente escreve -- a guarda nao pode bloquear tudo.
func TestAnexoComVersaoAceitaVersaoCorrente(t *testing.T) {
	meta := metaDeTeste(EscopoProject)
	m := fatoDeTeste()

	primeiro, err := AnexarComVersao(nil, meta, m, 0)
	if err != nil {
		t.Fatalf("primeira escrita: %v", err)
	}
	outra := fatoDeTeste()
	outra.ID = "mem-0002"
	meta2 := metaDeTeste(EscopoProject)
	meta2.MemoryID = "mem-0002"

	segundo, err := AnexarComVersao(primeiro, meta2, outra, 0)
	if err != nil {
		t.Fatalf("id novo na versao 0 devia passar: %v", err)
	}
	if len(segundo) <= len(primeiro) {
		t.Fatalf("os bytes nao cresceram")
	}
}

// O canario nao e' escolhido por mim: e' escolhido pelo DLP real. Se nenhum candidato for reconhecido, o teste
// FALHA em vez de passar por vacuidade -- e' essa a diferenca entre medir e decorar.
func TestMontarNotaRecusaConteudoComSegredo(t *testing.T) {
	candidatos := []string{
		"api" + "_key = " + "\"chave-de-teste-1234567890\"",
		"pass" + "word=" + "SuperSecreto123",
		"tok" + "en: " + "ghp_" + "0123456789abcdefghijklmnopqrstuvwx",
		"-----BEGIN " + "PRIVATE KEY-----\nMIIEvQIBADANBg\n-----END " + "PRIVATE KEY-----",
	}
	var canario string
	for _, c := range candidatos {
		if len(ScanDLP(c)) > 0 {
			canario = c
			break
		}
	}
	if canario == "" {
		t.Fatalf("nenhum canario foi reconhecido pelo ScanDLP: escolher outra forma antes de confiar no teste")
	}

	m := Memoria{ID: "mem-0001", Rotulo: MemoriaProposta, Conteudo: canario}
	if _, err := MontarNota(metaDeTeste(EscopoProject), m); !errors.Is(err, ErrSegredoNaNota) {
		t.Fatalf("esperava ErrSegredoNaNota, veio %v", err)
	}
}

// A recusa nao pode ser um bloqueio geral: conteudo normal continua a gravar.
func TestMontarNotaAceitaConteudoSemSegredo(t *testing.T) {
	if _, err := MontarNota(metaDeTeste(EscopoProject), fatoDeTeste()); err != nil {
		t.Fatalf("conteudo limpo recusado: %v", err)
	}
}

// --- I/O confinado a directorio temporario (nunca o vault operacional) ---

// Caminho que sai da raiz e' recusado ANTES de qualquer escrita. Inclui o caso que uma comparacao de strings
// ingenua deixaria passar depois de normalizar.
func TestCaminhoAutorizadoRecusaTravessia(t *testing.T) {
	raiz := t.TempDir()
	casos := []string{"../fora.md", "sub/../../fora.md", "/etc/passwd", "."}
	for _, c := range casos {
		if _, err := CaminhoAutorizado(raiz, c); !errors.Is(err, ErrForaDaRaiz) {
			t.Fatalf("caminho %q devia ser recusado, veio %v", c, err)
		}
	}
	ok, err := CaminhoAutorizado(raiz, "sub/nota.md")
	if err != nil {
		t.Fatalf("caminho legitimo recusado: %v", err)
	}
	if rel, rerr := filepath.Rel(raiz, ok); rerr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("caminho resolvido fora da raiz pedida: %q (raiz %q)", ok, raiz)
	}
}

// Grava, volta a ler, e prova que o segundo escritor com a mesma versao recebe conflito SEM alterar o ficheiro.
func TestAnexarEmArquivoGravaELe(t *testing.T) {
	raiz := t.TempDir()
	meta := metaDeTeste(EscopoProject)
	m := fatoDeTeste()

	anexou, err := AnexarEmArquivo(raiz, "notas/memoria.md", meta, m, 0)
	if err != nil || !anexou {
		t.Fatalf("primeira escrita: anexou=%v err=%v", anexou, err)
	}

	dados, err := os.ReadFile(filepath.Join(raiz, "notas", "memoria.md"))
	if err != nil {
		t.Fatalf("ler de volta: %v", err)
	}
	meta2, m2, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	if m2.ID != m.ID || meta2.Escopo != meta.Escopo {
		t.Fatalf("identidade perdida no disco: %#v", m2)
	}
	if err := IntegridadeDaNota(meta2, m2); err != nil {
		t.Fatalf("nota gravada nao passa a propria integridade: %v", err)
	}

	// Segundo escritor com a leitura antiga: conflito, e os bytes no disco NAO mudam.
	if _, err := AnexarEmArquivo(raiz, "notas/memoria.md", meta, m, 0); !errors.Is(err, ErrVersaoDivergente) {
		t.Fatalf("esperava ErrVersaoDivergente, veio %v", err)
	}
	depois, err := os.ReadFile(filepath.Join(raiz, "notas", "memoria.md"))
	if err != nil {
		t.Fatalf("reler: %v", err)
	}
	if !bytes.Equal(dados, depois) {
		t.Fatalf("a recusa de conflito alterou o ficheiro em disco")
	}
}

// Escrita atomica: depois do sucesso nao sobra temporario nenhum no directorio.
func TestAnexarEmArquivoNaoDeixaTemporarios(t *testing.T) {
	raiz := t.TempDir()
	if _, err := AnexarEmArquivo(raiz, "memoria.md", metaDeTeste(EscopoProject), fatoDeTeste(), 0); err != nil {
		t.Fatalf("escrita: %v", err)
	}
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entradas) != 1 || entradas[0].Name() != "memoria.md" {
		var nomes []string
		for _, e := range entradas {
			nomes = append(nomes, e.Name())
		}
		t.Fatalf("esperava apenas memoria.md, encontrei %v", nomes)
	}
}

// Dois escritores nao entram ao mesmo tempo: o segundo recebe ErrNotaBloqueada e NAO escreve.
func TestBloqueioImpedeSegundoEscritor(t *testing.T) {
	raiz := t.TempDir()
	alvo := filepath.Join(raiz, "memoria.md")
	if err := TravarNota(alvo); err != nil {
		t.Fatalf("primeiro bloqueio: %v", err)
	}
	defer DestravarNota(alvo)

	bloqueado, dono := Bloqueado(alvo)
	if !bloqueado || !strings.Contains(dono, "pid=") {
		t.Fatalf("bloqueio nao declarou dono: bloqueado=%v dono=%q", bloqueado, dono)
	}

	if _, err := AnexarEmArquivoComBloqueio(raiz, "memoria.md", metaDeTeste(EscopoProject), fatoDeTeste(), 0); !errors.Is(err, ErrNotaBloqueada) {
		t.Fatalf("esperava ErrNotaBloqueada, veio %v", err)
	}
	if _, err := os.Stat(alvo); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("o segundo escritor escreveu apesar do bloqueio")
	}
}

// Depois de libertar, o escritor seguinte entra e o ficheiro fica correto -- e sem sentinela para tras.
func TestBloqueioLibertadoPermiteEscrita(t *testing.T) {
	raiz := t.TempDir()
	alvo := filepath.Join(raiz, "memoria.md")
	if err := TravarNota(alvo); err != nil {
		t.Fatalf("bloquear: %v", err)
	}
	if err := DestravarNota(alvo); err != nil {
		t.Fatalf("destravar: %v", err)
	}
	if bloqueado, _ := Bloqueado(alvo); bloqueado {
		t.Fatalf("destravar deixou o sentinela")
	}

	anexou, err := AnexarEmArquivoComBloqueio(raiz, "memoria.md", metaDeTeste(EscopoProject), fatoDeTeste(), 0)
	if err != nil || !anexou {
		t.Fatalf("escrita apos libertar: anexou=%v err=%v", anexou, err)
	}
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entradas {
		if strings.HasSuffix(e.Name(), ".lock") {
			t.Fatalf("sentinela ficou para tras: %s", e.Name())
		}
	}
}

// Destravar o que nao esta' travado nao e' erro: um retry de limpeza nao pode falhar por ja' estar limpo.
func TestDestravarIdempotente(t *testing.T) {
	if err := DestravarNota(filepath.Join(t.TempDir(), "nunca-travado.md")); err != nil {
		t.Fatalf("destravar sem bloqueio devia ser no-op: %v", err)
	}
}

// --- ACL por identidade: negada por defeito ---

// Sem concessao nao ha acesso. Uma politica vazia e' negacao total, nao permissao total.
func TestACLSemConcessaoNega(t *testing.T) {
	var p PoliticaDeAcesso
	if err := p.Autorizar("agente-1", EscopoProject, PermissaoLer); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("politica vazia devia negar, veio %v", err)
	}
	// Identidade desconhecida numa politica que JA' tem concessoes continua negada.
	p = PoliticaDeAcesso{Concessoes: []Concessao{{Identidade: "agente-1", Escopo: EscopoProject, Permissoes: PermissaoLer}}}
	if err := p.Autorizar("estranho", EscopoProject, PermissaoLer); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("identidade sem concessao devia ser negada, veio %v", err)
	}
	if err := p.Autorizar("", EscopoProject, PermissaoLer); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("identidade vazia devia ser negada, veio %v", err)
	}
}

// Ler nao implica escrever, e o escopo e' parte da concessao: o mesmo agente fora do escopo e' negado.
func TestACLLeituraNaoImplicaEscrita(t *testing.T) {
	p := PoliticaDeAcesso{Concessoes: []Concessao{{Identidade: "revisor", Escopo: EscopoProject, Permissoes: PermissaoLer}}}

	if err := p.Autorizar("revisor", EscopoProject, PermissaoLer); err != nil {
		t.Fatalf("leitura concedida devia passar: %v", err)
	}
	err := p.Autorizar("revisor", EscopoProject, PermissaoEscrever)
	if !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("escrita sem concessao devia ser negada, veio %v", err)
	}
	if !strings.Contains(err.Error(), "ler") {
		t.Fatalf("a recusa devia dizer o que E' concedido, veio %q", err.Error())
	}
	if err := p.Autorizar("revisor", EscopoTenant, PermissaoLer); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("mesmo agente noutro escopo devia ser negado, veio %v", err)
	}
}

// Uma linha de ACL que nao concede nada e' recusada: quem a le acreditaria que concede.
func TestACLValidarRecusaConcessaoVazia(t *testing.T) {
	casos := []Concessao{
		{Identidade: "", Escopo: EscopoProject, Permissoes: PermissaoLer},
		{Identidade: "a", Escopo: "", Permissoes: PermissaoLer},
		{Identidade: "a", Escopo: EscopoProject, Permissoes: 0},
		{Identidade: "a", Escopo: EscopoProject, Permissoes: 1 << 5},
	}
	for i, c := range casos {
		if err := (PoliticaDeAcesso{Concessoes: []Concessao{c}}).Validar(); err == nil {
			t.Fatalf("caso %d devia ser recusado na validacao", i)
		}
	}
	boa := PoliticaDeAcesso{Concessoes: []Concessao{{Identidade: "a", Escopo: EscopoGoal, Permissoes: PermissaoLer | PermissaoEscrever}}}
	if err := boa.Validar(); err != nil {
		t.Fatalf("concessao legitima recusada: %v", err)
	}
	if err := boa.Autorizar("a", EscopoGoal, PermissaoLer|PermissaoEscrever); err != nil {
		t.Fatalf("concessao completa devia passar: %v", err)
	}
}

// A escrita consulta a ACL: sem concessao nao ha ficheiro, e o pedido de leitura nao autoriza escrita.
func TestAnexarAutorizadoNegaSemConcessao(t *testing.T) {
	raiz := t.TempDir()
	meta := metaDeTeste(EscopoProject)

	// Politica vazia: negacao total.
	if _, err := AnexarAutorizado(raiz, "memoria.md", PoliticaDeAcesso{}, "agente-1", meta, fatoDeTeste(), 0); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("politica vazia devia negar, veio %v", err)
	}
	// Concessao de LEITURA nao serve para escrever.
	soLeitura := PoliticaDeAcesso{Concessoes: []Concessao{{Identidade: "agente-1", Escopo: EscopoProject, Permissoes: PermissaoLer}}}
	if _, err := AnexarAutorizado(raiz, "memoria.md", soLeitura, "agente-1", meta, fatoDeTeste(), 0); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("leitura nao devia autorizar escrita, veio %v", err)
	}
	// Concessao no escopo errado tambem nao serve.
	outroEscopo := PoliticaDeAcesso{Concessoes: []Concessao{{Identidade: "agente-1", Escopo: EscopoTenant, Permissoes: PermissaoEscrever}}}
	if _, err := AnexarAutorizado(raiz, "memoria.md", outroEscopo, "agente-1", meta, fatoDeTeste(), 0); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("escopo errado devia negar, veio %v", err)
	}

	// O ponto do teste: em NENHUM dos casos acima ficou ficheiro nem sentinela no disco.
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entradas) != 0 {
		var nomes []string
		for _, e := range entradas {
			nomes = append(nomes, e.Name())
		}
		t.Fatalf("a recusa escreveu no disco: %v", nomes)
	}
}

// Com `PermissaoEscrever` no escopo certo, a escrita acontece e a nota passa a propria integridade.
func TestAnexarAutorizadoComConcessaoEscreve(t *testing.T) {
	raiz := t.TempDir()
	meta := metaDeTeste(EscopoProject)
	pol := PoliticaDeAcesso{Concessoes: []Concessao{{Identidade: "agente-1", Escopo: EscopoProject, Permissoes: PermissaoEscrever}}}

	anexou, err := AnexarAutorizado(raiz, "memoria.md", pol, "agente-1", meta, fatoDeTeste(), 0)
	if err != nil || !anexou {
		t.Fatalf("escrita autorizada devia passar: anexou=%v err=%v", anexou, err)
	}
	dados, err := os.ReadFile(filepath.Join(raiz, "memoria.md"))
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	meta2, m2, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	if err := IntegridadeDaNota(meta2, m2); err != nil {
		t.Fatalf("nota autorizada nao passa a integridade: %v", err)
	}
	// Outra identidade continua negada no MESMO ficheiro ja' existente.
	if _, err := AnexarAutorizado(raiz, "memoria.md", pol, "intruso", meta, fatoDeTeste(), 0); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("outra identidade devia ser negada, veio %v", err)
	}
}

// Ligacao de directorio: juncao no Windows (nao exige privilegio), symlink nos restantes.
func criarLigacaoDeDirectorio(t *testing.T, alvo, ligacao string) {
	t.Helper()
	var err error
	if runtime.GOOS == "windows" {
		err = exec.Command("cmd", "/c", "mklink", "/J", ligacao, alvo).Run()
	} else {
		err = os.Symlink(alvo, ligacao)
	}
	if err != nil {
		t.Fatalf("nao consegui criar a ligacao %q -> %q: %v", ligacao, alvo, err)
	}
}

// O caminho devolvido tem de estar sob a raiz PEDIDA, mesmo quando essa raiz resolve para outro caminho.
//
// Era assim que falhava em CI: a raiz vinha de `t.TempDir()` e, no Windows do runner, `EvalSymlinks` expandia
// `RUNNER~1` para `runneradmin`. Com o caminho montado sobre a raiz RESOLVIDA, o chamador recebia um caminho que
// nao estava sob a raiz que tinha dado -- e o teste via "fora da raiz" num caminho autorizado.
func TestCaminhoAutorizadoSobRaizQueResolveParaOutroCaminho(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(filepath.Join(real, "sub"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	alias := filepath.Join(base, "alias")
	criarLigacaoDeDirectorio(t, real, alias)

	ok, err := CaminhoAutorizado(alias, "sub/nota.md")
	if err != nil {
		t.Fatalf("raiz legitima recusada: %v", err)
	}

	// O discriminante: o devolvido e' o caminho pedido, nao o resolvido.
	if ok != filepath.Join(alias, "sub", "nota.md") {
		t.Fatalf("devolveu %q; o contrato e' %q (devolver o caminho resolvido %q quebra o chamador)",
			ok, filepath.Join(alias, "sub", "nota.md"), filepath.Join(real, "sub", "nota.md"))
	}
	if rel, rerr := filepath.Rel(alias, ok); rerr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("caminho devolvido %q nao esta' sob a raiz pedida %q (rel=%q)", ok, alias, rel)
	}

	// A proteccao contra travessia continua activa quando a raiz resolve para outro caminho.
	if _, err := CaminhoAutorizado(alias, "../fora.md"); !errors.Is(err, ErrForaDaRaiz) {
		t.Fatalf("travessia devia continuar recusada, veio %v", err)
	}
}
