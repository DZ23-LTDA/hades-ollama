// Testes do catalogo embutido.
//
// Discriminantes: toda entrada tem de passar `Validate()`, a ordem tem de ser determinística, a copia tem de
// ser copia, o lookup tem de ser exacto, e NENHUMA entrada pode estar avaliada sem ter sido medida.
package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// *Um catalogo com uma entrada que nao passa no proprio validador nao e' um catalogo: e' uma lista de
// intencoes.*
func TestCatalogoTodasAsEntradasPassamNoValidador(t *testing.T) {
	for _, p := range Catalog() {
		if err := p.Validate(); err != nil {
			t.Fatalf("%s: entrada do catalogo nao valida: %v", p.ID, err)
		}
	}
}

// *Nao avaliado e' um estado, e e' diferente de aprovado.*
func TestCatalogoNenhumaEntradaSeDizAvaliada(t *testing.T) {
	for _, p := range Catalog() {
		if strings.TrimSpace(p.Evaluation) != "" {
			t.Fatalf("%s: catalogo embutido nao pode declarar avaliacao (%q) -- ninguem a mediu",
				p.ID, p.Evaluation)
		}
	}
}

func TestCatalogoTemEntradas(t *testing.T) {
	if CatalogCount() < 4 {
		t.Fatalf("catalogo quase vazio: %d", CatalogCount())
	}
	if len(Catalog()) != CatalogCount() {
		t.Fatalf("CatalogCount()=%d != len(Catalog())=%d", CatalogCount(), len(Catalog()))
	}
}

// *Um catalogo que muda de ordem sozinho faz o painel parecer que mudou alguma coisa.*
func TestCatalogoOrdemDeterministica(t *testing.T) {
	a, err := json.Marshal(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("duas chamadas a Catalog() produziram bytes diferentes")
	}
	ids := make([]string, 0, CatalogCount())
	for _, p := range Catalog() {
		ids = append(ids, p.ID)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			t.Fatalf("ordem por ID violada: %q antes de %q", ids[i-1], ids[i])
		}
	}
}

// *Um chamador que altere a fatia nao altera o catalogo do processo seguinte.*
func TestCatalogoDevolveCopia(t *testing.T) {
	primeira := Catalog()
	original := primeira[0].Name
	primeira[0].Name = "ALTERADO"
	defer func() {
		// restaura para nao contaminar outros testes na mesma execucao
		primeira[0].Name = original
	}()
	segunda := Catalog()
	for _, p := range segunda {
		if p.Name == "ALTERADO" {
			t.Fatal("Catalog() devolveu a fatia interna: mutar uma chamada mudou a seguinte")
		}
	}
}

// *Um prefixo nao serve: `hades-executor` nao e' `hades-exec`.*
func TestCatalogoLookupEExacto(t *testing.T) {
	p, ok := CatalogLookup("hades-executor")
	if !ok {
		t.Fatal("hades-executor devia estar no catalogo")
	}
	if p.ID != "hades-executor" {
		t.Fatalf("lookup devolveu %q", p.ID)
	}
	if _, ok := CatalogLookup("hades-exec"); ok {
		t.Fatal("lookup aceitou um prefixo")
	}
	if _, ok := CatalogLookup("HADES-EXECUTOR"); ok {
		t.Fatal("lookup ignorou a caixa")
	}
	if _, ok := CatalogLookup(""); ok {
		t.Fatal("lookup aceitou id vazio")
	}
}

// *Uma persona que nao renderiza nao se instala, e uma que nao se instala nao pertence a um catalogo de
// instalaveis.*
func TestCatalogoRenderizaParaHades(t *testing.T) {
	for _, p := range Catalog() {
		b, err := Render(p, TargetHades)
		if err != nil {
			t.Fatalf("%s: Render(hades): %v", p.ID, err)
		}
		if len(b) == 0 {
			t.Fatalf("%s: render vazio", p.ID)
		}
		if _, err := p.ContentHash(); err != nil {
			t.Fatalf("%s: ContentHash: %v", p.ID, err)
		}
	}
}

// *Proveniencia: um agente sem fonte e sem licenca nao se audita nem se redistribui.*
func TestCatalogoTemProvenienciaPropria(t *testing.T) {
	for _, p := range Catalog() {
		if p.Source != "hades-catalogo" {
			t.Fatalf("%s: source=%q -- catalogo embutido tem de declarar a propria origem", p.ID, p.Source)
		}
		if p.License != "MIT" {
			t.Fatalf("%s: license=%q", p.ID, p.License)
		}
		if !strings.Contains(p.SourceRef, p.ID) {
			t.Fatalf("%s: source_ref nao cita o id: %q", p.ID, p.SourceRef)
		}
	}
}

// *Uma persona que pede uma capacidade que o catalogo nao declara e' uma promessa sem nome.*
func TestCatalogoCapacidadesNaoVazias(t *testing.T) {
	for _, p := range Catalog() {
		if len(p.Capabilities) == 0 {
			t.Fatalf("%s: nenhuma capacidade declarada", p.ID)
		}
		if len(p.Scopes) == 0 {
			t.Fatalf("%s: nenhum scope declarado", p.ID)
		}
	}
}

// *Um id que vira nome de ficheiro no destino e' um caminho, nao um id.*
func TestCatalogoIDsSaoNomesDeFicheiro(t *testing.T) {
	for _, p := range Catalog() {
		if strings.ContainsAny(p.ID, "/\\") || strings.Contains(p.ID, "..") {
			t.Fatalf("%s: id com separador de caminho", p.ID)
		}
		if strings.Contains(p.ID, " ") {
			t.Fatalf("%s: id com espaco", p.ID)
		}
	}
}

func TestCatalogoVersaoDeclarada(t *testing.T) {
	if CatalogVersion() == "" {
		t.Fatal("CatalogVersion vazia")
	}
}
