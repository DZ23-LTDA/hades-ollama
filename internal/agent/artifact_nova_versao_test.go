package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func manifestoAtual() ArtifactManifest {
	return ArtifactManifest{
		ID: "art1", MissionID: "m1", Name: "index.html", Path: "site/index.html",
		MediaType: "text/html", Size: 12, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Version: 3, CreatedAt: time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC),
	}
}

// POSITIVO: cria a versao N+1 com o hash novo e mantem a anterior intacta.
func TestArtefactoPrepararNovaVersaoCriaProxima(t *testing.T) {
	atual := manifestoAtual()
	agora := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

	novo, err := ArtefactoPrepararNovaVersao(atual, []byte("<h1>editado</h1>"), 3, agora)
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if novo.Version != 4 {
		t.Fatalf("versao = %d, queria 4", novo.Version)
	}
	if novo.SHA256 == atual.SHA256 {
		t.Fatal("hash devia mudar")
	}
	if novo.Size != int64(len("<h1>editado</h1>")) {
		t.Fatalf("size = %d", novo.Size)
	}
	if !novo.CreatedAt.Equal(agora.UTC()) {
		t.Fatalf("created_at nao actualizado: %v", novo.CreatedAt)
	}
	// A versao anterior tem de continuar como estava (nao mutamos o manifesto recebido).
	if atual.Version != 3 || atual.Size != 12 {
		t.Fatalf("manifesto original foi mutado: %+v", atual)
	}
}

// NEGATIVO: conteudo vazio e' recusado.
func TestArtefactoPrepararNovaVersaoRecusaVazio(t *testing.T) {
	_, err := ArtefactoPrepararNovaVersao(manifestoAtual(), nil, 3, time.Now())
	if !errors.Is(err, ErrConteudoArtefactoVazio) {
		t.Fatalf("esperava ErrConteudoArtefactoVazio, veio %v", err)
	}
}

// NEGATIVO: acima do limite e' recusado (1 MiB).
func TestArtefactoPrepararNovaVersaoRecusaGrande(t *testing.T) {
	enorme := make([]byte, LimiteConteudoArtefacto+1)
	for i := range enorme {
		enorme[i] = 'a'
	}
	_, err := ArtefactoPrepararNovaVersao(manifestoAtual(), enorme, 3, time.Now())
	if !errors.Is(err, ErrConteudoArtefactoGrande) {
		t.Fatalf("esperava ErrConteudoArtefactoGrande, veio %v", err)
	}
}

// NEGATIVO: CAS — versao esperada diferente da corrente nao sobrescreve.
func TestArtefactoPrepararNovaVersaoRecusaCAS(t *testing.T) {
	_, err := ArtefactoPrepararNovaVersao(manifestoAtual(), []byte("x"), 2, time.Now())
	if !errors.Is(err, ErrRestauroConflitante) {
		t.Fatalf("esperava ErrRestauroConflitante, veio %v", err)
	}
	// Sem expected_version (0) tambem e' recusado: nao ha CAS sem expectativa.
	if _, err := ArtefactoPrepararNovaVersao(manifestoAtual(), []byte("x"), 0, time.Now()); !errors.Is(err, ErrRestauroConflitante) {
		t.Fatalf("sem expected_version devia recusar, veio %v", err)
	}
}

// NEGATIVO: conteudo identico nao inventa versao nova.
func TestArtefactoPrepararNovaVersaoRecusaIdentico(t *testing.T) {
	conteudo := []byte("<h1>igual</h1>")
	atual := manifestoAtual()
	soma := sha256Hex(conteudo)
	atual.SHA256 = soma
	_, err := ArtefactoPrepararNovaVersao(atual, conteudo, 3, time.Now())
	if !errors.Is(err, ErrConteudoArtefactoInalterado) {
		t.Fatalf("esperava ErrConteudoArtefactoInalterado, veio %v", err)
	}
	if !strings.EqualFold(soma, atual.SHA256) {
		t.Fatal("helper de teste incoerente")
	}
}
