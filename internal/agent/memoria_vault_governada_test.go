package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func politicaDeTeste(identidade string, escopo EscopoDeMemoria, permissoes PermissaoDeMemoria) PoliticaDeAcesso {
	return PoliticaDeAcesso{Concessoes: []Concessao{{Identidade: identidade, Escopo: escopo, Permissoes: permissoes}}}
}

func pedidoDeTeste(identidade, tenant, relativo string) PedidoDeAppendGovernado {
	return PedidoDeAppendGovernado{
		Identidade: identidade,
		Tenant:     tenant,
		Relativo:   relativo,
		Memoria: Memoria{
			Rotulo:    MemoriaFatoVerificado,
			Conteudo:  "medicao de teste",
			Fonte:     "go test",
			Aprovador: identidade,
		},
	}
}

func contarFicheiros(t *testing.T, raiz string) int {
	t.Helper()
	total := 0
	erro := filepath.Walk(raiz, func(caminho string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total++
		}
		return nil
	})
	if erro != nil {
		t.Fatalf("nao consegui listar %q: %v", raiz, erro)
	}
	return total
}

// NEGATIVO OBRIGATORIO: sem concessao, recusa segura E nenhum ficheiro criado.
func TestAnexarGovernadoSemConcessaoRecusaENaoCriaFicheiro(t *testing.T) {
	raiz := t.TempDir()
	servico := ServicoDeAppendGovernado{
		Raiz:     raiz,
		Politica: politicaDeTeste("outra-identidade", EscopoTenant, PermissaoEscrever),
	}

	escreveu, err := servico.Anexar(pedidoDeTeste("user-sem-concessao", "org-1", "notas/uma.md"))
	if !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("esperava ErrAcessoNegado, veio %v", err)
	}
	if escreveu {
		t.Fatal("nao podia ter escrito")
	}
	if n := contarFicheiros(t, raiz); n != 0 {
		t.Fatalf("a recusa deixou %d ficheiro(s) na raiz", n)
	}
}

// NEGATIVO: so' leitura nao implica escrita -- as permissoes sao bits separados.
func TestAnexarGovernadoSoLeituraNaoEscreve(t *testing.T) {
	raiz := t.TempDir()
	servico := ServicoDeAppendGovernado{
		Raiz:     raiz,
		Politica: politicaDeTeste("user-1", EscopoTenant, PermissaoLer),
	}
	if _, err := servico.Anexar(pedidoDeTeste("user-1", "org-1", "notas/uma.md")); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("leitura nao podia autorizar escrita, veio %v", err)
	}
	if n := contarFicheiros(t, raiz); n != 0 {
		t.Fatalf("a recusa deixou %d ficheiro(s)", n)
	}
}

// NEGATIVO: concessao noutro escopo nao autoriza o escopo do tenant.
func TestAnexarGovernadoConcessaoNoutroEscopoNaoServe(t *testing.T) {
	raiz := t.TempDir()
	servico := ServicoDeAppendGovernado{
		Raiz:     raiz,
		Politica: politicaDeTeste("user-1", EscopoSession, PermissaoEscrever),
	}
	if _, err := servico.Anexar(pedidoDeTeste("user-1", "org-1", "notas/uma.md")); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("escopo diferente nao podia autorizar, veio %v", err)
	}
	if n := contarFicheiros(t, raiz); n != 0 {
		t.Fatalf("a recusa deixou %d ficheiro(s)", n)
	}
}

// NEGATIVO: sem identidade ou sem tenant nao se escreve, mesmo com politica permissiva.
func TestAnexarGovernadoSemIdentidadeOuSemTenantRecusa(t *testing.T) {
	raiz := t.TempDir()
	servico := ServicoDeAppendGovernado{Raiz: raiz, Politica: politicaDeTeste("", EscopoTenant, PermissaoEscrever)}

	if _, err := servico.Anexar(pedidoDeTeste("", "org-1", "n.md")); !errors.Is(err, ErrIdentidadeDeSessaoAusente) {
		t.Fatalf("esperava ErrIdentidadeDeSessaoAusente, veio %v", err)
	}
	if _, err := servico.Anexar(pedidoDeTeste("user-1", "   ", "n.md")); !errors.Is(err, ErrTenantDaSessaoAusente) {
		t.Fatalf("esperava ErrTenantDaSessaoAusente, veio %v", err)
	}
	if n := contarFicheiros(t, raiz); n != 0 {
		t.Fatalf("a recusa deixou %d ficheiro(s)", n)
	}
}

// NEGATIVO: sem raiz configurada, recusa -- o adaptador nunca escolhe o vault operacional.
func TestAnexarGovernadoSemRaizRecusa(t *testing.T) {
	servico := ServicoDeAppendGovernado{Politica: politicaDeTeste("user-1", EscopoTenant, PermissaoEscrever)}
	if _, err := servico.Anexar(pedidoDeTeste("user-1", "org-1", "n.md")); !errors.Is(err, ErrRaizDoVaultAusente) {
		t.Fatalf("esperava ErrRaizDoVaultAusente, veio %v", err)
	}
}

// POSITIVO: com concessao de escrita no escopo do tenant, a nota e' criada DENTRO da raiz.
func TestAnexarGovernadoComConcessaoEscreveDentroDaRaiz(t *testing.T) {
	raiz := t.TempDir()
	servico := ServicoDeAppendGovernado{
		Raiz:     raiz,
		Politica: politicaDeTeste("user-1", EscopoTenant, PermissaoLer|PermissaoEscrever),
	}
	escreveu, err := servico.Anexar(pedidoDeTeste("user-1", "org-1", "notas/uma.md"))
	if err != nil {
		t.Fatalf("esperava escrita, veio erro %v", err)
	}
	if !escreveu {
		t.Fatal("esperava escreveu=true")
	}

	alvo := filepath.Join(raiz, "notas", "uma.md")
	conteudo, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatalf("nota nao ficou em %q: %v", alvo, err)
	}
	if !strings.Contains(string(conteudo), "medicao de teste") {
		t.Fatalf("nota sem o conteudo pedido: %q", string(conteudo))
	}
	// O caminho devolvido e' o criado, e continua a recusar travessia.
	if _, err := servico.Anexar(pedidoDeTeste("user-1", "org-1", "../fora.md")); !errors.Is(err, ErrForaDaRaiz) {
		t.Fatalf("travessia devia continuar recusada, veio %v", err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "..", "fora.md")); err == nil {
		t.Fatal("a travessia criou ficheiro fora da raiz")
	}
}

// NEGATIVO: anexar sobre uma nota editada FORA do adaptador tem de recusar (o hash diverge),
// em vez de escrever por cima de trabalho humano sem notar.
func TestAnexarGovernadoRecusaNotaEditadaAMao(t *testing.T) {
	raiz := t.TempDir()
	servico := ServicoDeAppendGovernado{
		Raiz:     raiz,
		Politica: politicaDeTeste("user-1", EscopoTenant, PermissaoLer|PermissaoEscrever),
	}
	if _, err := servico.Anexar(pedidoDeTeste("user-1", "org-1", "notas/humana.md")); err != nil {
		t.Fatalf("criacao devia passar: %v", err)
	}
	alvo := filepath.Join(raiz, "notas", "humana.md")
	dados, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	if err := os.WriteFile(alvo, append(dados, []byte("\nlinha humana\n")...), 0o600); err != nil {
		t.Fatalf("editar a mao: %v", err)
	}

	_, err = servico.Anexar(pedidoDeTeste("user-1", "org-1", "notas/humana.md"))
	if !errors.Is(err, ErrConteudoAlterado) {
		t.Fatalf("esperava ErrConteudoAlterado ao anexar sobre nota editada, veio %v", err)
	}
	final, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatalf("reler: %v", err)
	}
	if !strings.Contains(string(final), "linha humana") {
		t.Fatal("a edicao humana foi perdida")
	}
	if strings.Contains(string(final), "medicao de teste\\nmedicao de teste") {
		t.Fatal("o adaptador anexou por cima da edicao humana")
	}
}

// NEGATIVO: anexar num caminho que ja' e' de OUTRO tenant nao pode escrever nele.
func TestAnexarGovernadoRecusaCaminhoDeOutroTenant(t *testing.T) {
	raiz := t.TempDir()
	gravadorAlheio := ServicoDeAppendGovernado{
		Raiz:     raiz,
		Politica: politicaDeTeste("user-1", EscopoTenant, PermissaoLer|PermissaoEscrever),
	}
	if _, err := gravadorAlheio.Anexar(pedidoDeTeste("user-1", "org-DONA", "notas/comum.md")); err != nil {
		t.Fatalf("preparacao: %v", err)
	}
	_, err := gravadorAlheio.Anexar(pedidoDeTeste("user-1", "org-1", "notas/comum.md"))
	if !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("esperava ErrAcessoNegado ao escrever em nota de outro tenant, veio %v", err)
	}
}
