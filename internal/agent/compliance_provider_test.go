package agent

// Testes do `ComplianceProvider` (E-1266).
//
// O que estes testes travam:
//
//  1. le' `go.mod` e separa diretos de indiretos;
//  2. le' `package.json` e separa `dependencies` de `devDependencies`;
//  3. um `go.mod` com bloco `require ( ... )` e' lido por inteiro;
//  4. um `package.json` invalido e' registado como INVALIDO e NAO inventa componentes;
//  5. `SemLicenca` conta o que nao tem licenca declarada -- e `Licenca` vazio nao vira "sem licenca";
//  6. `NaoProduzido` declara as tres coisas que o adendo pede e este provider nao faz;
//  7. a assinatura do inventario e' estavel e ordenada;
//  8. `node_modules` nao e' percorrido.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// criarProjecto monta um projecto com `go.mod` e `package.json`.
func criarProjecto(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()

	goMod := `module exemplo.com/projecto

go 1.26

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/stretchr/testify v1.9.0 // indirect
)

require golang.org/x/text v0.20.0 // indirect
`
	if err := os.WriteFile(filepath.Join(raiz, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}

	pacote := `{
  "name": "exemplo",
  "version": "1.0.0",
  "license": "MIT",
  "dependencies": {
    "react": "^19.3.0"
  },
  "devDependencies": {
    "typescript": "^5.9.0"
  }
}`
	if err := os.WriteFile(filepath.Join(raiz, "package.json"), []byte(pacote), 0o600); err != nil {
		t.Fatalf("escrever package.json: %v", err)
	}
	return raiz
}

func TestComplianceProviderLeGoMod(t *testing.T) {
	raiz := criarProjecto(t)
	provider := ProviderDeConformidade{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}

	porNome := map[string]Componente{}
	for _, componente := range inventario.Componentes {
		porNome[componente.Nome] = componente
	}

	// O bloco `require ( ... )` tem de ser lido por inteiro.
	gin, ok := porNome["github.com/gin-gonic/gin"]
	if !ok {
		t.Fatal("gin do bloco require nao apareceu")
	}
	if gin.Versao != "v1.10.0" {
		t.Fatalf("versao do gin = %q; esperado v1.10.0", gin.Versao)
	}
	if !gin.Direto {
		t.Fatal("gin devia ser direto")
	}

	// `// indirect` marca nao-direto.
	testify, ok := porNome["github.com/stretchr/testify"]
	if !ok {
		t.Fatal("testify nao apareceu")
	}
	if testify.Direto {
		t.Fatal("testify tem `// indirect` e foi marcado como direto")
	}

	// Um `require` de linha unica tambem e' lido.
	if _, ok := porNome["golang.org/x/text"]; !ok {
		t.Fatal("o require de linha unica nao foi lido")
	}
}

func TestComplianceProviderLePackageJSON(t *testing.T) {
	raiz := criarProjecto(t)
	provider := ProviderDeConformidade{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}

	porNome := map[string]Componente{}
	for _, componente := range inventario.Componentes {
		porNome[componente.Nome] = componente
	}

	react, ok := porNome["react"]
	if !ok {
		t.Fatal("react nao apareceu")
	}
	if react.Tipo != ComponentePacoteNPM {
		t.Fatalf("tipo do react = %q; esperado PACOTE_NPM", react.Tipo)
	}
	if !react.Direto {
		t.Fatal("react de `dependencies` devia ser direto")
	}

	typescript, ok := porNome["typescript"]
	if !ok {
		t.Fatal("typescript nao apareceu")
	}
	if typescript.Direto {
		t.Fatal("typescript de `devDependencies` foi marcado como direto")
	}
}

// Um `package.json` invalido e' REGISTADO, e nao inventa componentes.
func TestComplianceProviderRegistaPackageJSONInvalido(t *testing.T) {
	raiz := t.TempDir()
	if err := os.WriteFile(filepath.Join(raiz, "package.json"), []byte("{ isto nao e' json"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	provider := ProviderDeConformidade{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	if inventario.Total != 0 {
		t.Fatalf("total = %d; um package.json invalido nao pode produzir componentes", inventario.Total)
	}
	// Mas o ficheiro tem de aparecer como tentado.
	var encontrado bool
	for _, lido := range inventario.FicheirosLidos {
		if strings.Contains(lido, "package.json") && strings.Contains(lido, "INVALIDO") {
			encontrado = true
		}
	}
	if !encontrado {
		t.Fatalf("package.json invalido nao foi registado: %v", inventario.FicheirosLidos)
	}
}

// `Licenca` vazio significa NAO DECLARADA, nunca "sem licenca".
func TestComplianceProviderDistingueLicencaNaoDeclarada(t *testing.T) {
	raiz := criarProjecto(t)
	provider := ProviderDeConformidade{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	// Nenhum dos componentes de teste declara licenca -- nem o go.mod nem as deps npm.
	if inventario.SemLicenca != inventario.Total {
		t.Fatalf("sem_licenca = %d; total = %d", inventario.SemLicenca, inventario.Total)
	}
	for _, componente := range inventario.Componentes {
		if componente.Licenca != "" {
			t.Fatalf("%s tem licenca %q que nao estava declarada", componente.Nome, componente.Licenca)
		}
	}
	// E o aviso explica o que vazio significa.
	if !strings.Contains(inventario.Aviso, "NAO DECLARADA") {
		t.Fatalf("aviso nao explica o campo vazio: %q", inventario.Aviso)
	}
}

// O provider DECLARA o que o adendo pede e ele nao faz.
func TestComplianceProviderDeclaraONaoProduzido(t *testing.T) {
	raiz := criarProjecto(t)
	provider := ProviderDeConformidade{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	// O adendo 28.9 pede SBOM, notices, provenance. Este provider so' faz o primeiro.
	esperado := []string{"THIRD_PARTY_NOTICES", "arvore transitiva", "provenance"}
	for _, chave := range esperado {
		var encontrado bool
		for _, naoProduzido := range inventario.NaoProduzido {
			if strings.Contains(naoProduzido, chave) {
				encontrado = true
			}
		}
		if !encontrado {
			t.Fatalf("`%s` nao esta' declarado em NaoProduzido: %v", chave, inventario.NaoProduzido)
		}
	}
}

func TestComplianceProviderAssinaturaEstavel(t *testing.T) {
	raiz := criarProjecto(t)
	provider := ProviderDeConformidade{}
	primeiro, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	segundo, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar 2: %v", err)
	}
	assinaturaUm := AssinaturaDoInventario(primeiro)
	assinaturaDois := AssinaturaDoInventario(segundo)
	if len(assinaturaUm) != len(assinaturaDois) {
		t.Fatalf("assinaturas com tamanhos diferentes: %d e %d", len(assinaturaUm), len(assinaturaDois))
	}
	for i := range assinaturaUm {
		if assinaturaUm[i] != assinaturaDois[i] {
			t.Fatalf("assinatura mudou entre leituras: %q vs %q", assinaturaUm[i], assinaturaDois[i])
		}
	}
	// E a assinatura tem de estar ordenada.
	for i := 1; i < len(assinaturaUm); i++ {
		if assinaturaUm[i-1] > assinaturaUm[i] {
			t.Fatalf("assinatura desordenada em %d: %q > %q", i, assinaturaUm[i-1], assinaturaUm[i])
		}
	}
}

func TestComplianceProviderNaoPercorreNodeModules(t *testing.T) {
	raiz := criarProjecto(t)
	// Um `package.json` dentro de `node_modules` NAO pode entrar no SBOM.
	pasta := filepath.Join(raiz, "node_modules", "react")
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pasta, "package.json"),
		[]byte(`{"name":"react","dependencies":{"inventado":"1.0.0"}}`), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	provider := ProviderDeConformidade{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	for _, componente := range inventario.Componentes {
		if componente.Nome == "inventado" {
			t.Fatal("um package.json dentro de node_modules entrou no SBOM")
		}
	}
}

func TestComplianceProviderRecusaRaizInexistente(t *testing.T) {
	provider := ProviderDeConformidade{}
	_, err := provider.Inventariar(context.Background(), filepath.Join(t.TempDir(), "nao-existe"))
	if err == nil {
		t.Fatal("erro = nil para uma raiz inexistente")
	}
}

// Inventaria o repositorio REAL do HADES, quando `HADES_SBOM_RAIZ` esta' definido.
//
// Nao falha se a variavel nao existir: o teste e' uma medicao opcional, nao uma dependencia.
func TestComplianceProviderComRepositorioReal(t *testing.T) {
	raiz := os.Getenv("HADES_SBOM_RAIZ")
	if raiz == "" {
		t.Skip("HADES_SBOM_RAIZ nao definido")
	}
	if _, err := os.Stat(raiz); err != nil {
		t.Skipf("raiz ausente: %v", err)
	}
	provider := ProviderDeConformidade{}
	inventario, err := provider.Inventariar(context.Background(), raiz)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	t.Logf("SBOM real: %d componentes, %d sem licenca declarada, tipos=%v, ficheiros=%v",
		inventario.Total, inventario.SemLicenca, inventario.PorTipo, inventario.FicheirosLidos)
	if inventario.Total == 0 {
		t.Fatal("SBOM vazio num repositorio que tem go.mod")
	}
	// O repo tem `go.mod` e pelo menos um `package.json`.
	if inventario.PorTipo["MODULO_GO"] == 0 {
		t.Fatal("nenhum modulo Go encontrado no repositorio")
	}
	// A assinatura tem de ser estavel.
	chavesDoRepositorio := AssinaturaDoInventario(inventario)
	if len(chavesDoRepositorio) != inventario.Total {
		t.Fatalf("assinatura com %d entradas para %d componentes", len(chavesDoRepositorio), inventario.Total)
	}
}
