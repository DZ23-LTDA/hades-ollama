package agent

// FASE 28.1 item 8 e adendo 28.9 do adendo V3: o `ComplianceProvider`.
//
// O QUE ESTE PROVIDER PRODUZ, E O QUE RECUSA PRODUZIR
//
// O adendo 28.9 pede, para cada artefacto candidato: "4. produza SBOM e THIRD_PARTY_NOTICES" e
// "5. gere checksum e provenance/attestation".
//
// Este provider gera o **inventario de componentes** a partir dos ficheiros de dependencia que
// existem no repositorio: `go.mod` (modulos Go) e `package.json` (pacotes npm). Nao consulta a rede,
// nao resolve versoes, nao adivinha transitivas.
//
//	*Um SBOM que inclui o que o programador supoe e' pior do que nenhum: quem le' acredita.*
//
// AS TRES COISAS QUE ELE NAO FAZ, E QUE O CORPO DIZ
//
//  1. **Nao gera THIRD_PARTY_NOTICES.** Uma notice exige o TEXTO da licenca de cada componente, e o
//     texto nao esta' no `go.mod` nem no `package.json`. Escrever uma notice a partir de nomes de
//     licenca seria *documentacao a mentir*.
//
//  2. **Nao resolve a arvore transitiva.** O SBOM lista as dependencias DECLARADAS. As transitivas
//     exigiriam `go mod graph`/`npm ls` -- que este provider nao executa, porque executar
//     subprocessos a partir de uma varredura e' uma capacidade que nao deve vir de graca.
//
//  3. **Nao afirma a licenca quando ela nao esta' declarada.** `Licenca` vazio significa "nao
//     declarada", nunca "sem licenca": sao factos diferentes e o campo diz qual e'.
//
//	*Um relatorio que obriga a somar a mao e' um relatorio para quem ja' sabia a resposta.*

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TipoDeComponente classifica uma dependencia.
type TipoDeComponente string

const (
	// ComponenteModuloGo e' uma dependencia declarada num `go.mod`.
	ComponenteModuloGo TipoDeComponente = "MODULO_GO"
	// ComponentePacoteNPM e' uma dependencia declarada num `package.json`.
	ComponentePacoteNPM TipoDeComponente = "PACOTE_NPM"
	// ComponenteBinario e' um binario vendorizado encontrado no repositorio.
	ComponenteBinario TipoDeComponente = "BINARIO"
)

// Componente e' uma entrada do inventario de software.
//
// Nenhum campo e' preenchido por inferencia. `Licenca` vazio significa "nao declarada no ficheiro de
// origem", e `Direto` diz se veio da lista principal ou de `devDependencies`.
type Componente struct {
	Nome        string           `json:"nome"`
	Versao      string           `json:"versao"`
	Tipo        TipoDeComponente `json:"tipo"`
	Licenca     string           `json:"licenca,omitempty"`
	Direto      bool             `json:"direto"`
	Origem      string           `json:"origem"`
	DeclaradoEm string           `json:"declarado_em"`
}

// InventarioDeSoftware e' o SBOM minimo que este provider consegue produzir com honestidade.
type InventarioDeSoftware struct {
	Raiz        string         `json:"raiz"`
	MedidoEm    string         `json:"medido_em"`
	Componentes []Componente   `json:"componentes"`
	Total       int            `json:"total"`
	PorTipo     map[string]int `json:"por_tipo"`
	// SemLicenca conta os componentes cuja licenca NAO esta' declarada na origem.
	SemLicenca int `json:"sem_licenca"`
	// FicheirosLidos lista as fontes efectivamente consultadas.
	FicheirosLidos []string `json:"ficheiros_lidos"`
	// NaoProduzido lista o que o adendo pede e este provider NAO faz -- dito, nao omitido.
	NaoProduzido []string `json:"nao_produzido"`
	Aviso        string   `json:"aviso"`
}

// maxFicheirosDeDependencia limita quantos `package.json` se percorrem.
//
// Um monorepo pode ter dezenas; mais do que isso e' sinal de que a varredura deve ser dirigida.
const maxFicheirosDeDependencia = 64

// ComplianceProvider e' o contrato do item 8 do adendo 28.1.
type ComplianceProvider interface {
	// Inventariar le' os ficheiros de dependencia da raiz e devolve o SBOM minimo.
	Inventariar(ctx context.Context, raiz string) (InventarioDeSoftware, error)
}

// ProviderDeConformidade produz o inventario de software.
type ProviderDeConformidade struct {
	// MedidoEm permite fixar o instante nos testes.
	MedidoEm string
}

// Inventariar le' `go.mod` e os `package.json` da raiz.
func (p ProviderDeConformidade) Inventariar(
	ctx context.Context,
	raiz string,
) (InventarioDeSoftware, error) {
	if err := ctx.Err(); err != nil {
		return InventarioDeSoftware{}, err
	}
	absoluta, err := filepath.Abs(raiz)
	if err != nil {
		return InventarioDeSoftware{}, fmt.Errorf("%w: resolver %q: %v", ErrKitInexistente, raiz, err)
	}
	estado, err := os.Stat(absoluta)
	if err != nil {
		return InventarioDeSoftware{}, fmt.Errorf("%w: %q: %v", ErrKitInexistente, absoluta, err)
	}
	if !estado.IsDir() {
		return InventarioDeSoftware{}, fmt.Errorf("%w: %q nao e' um directorio", ErrKitInexistente, absoluta)
	}

	inventario := InventarioDeSoftware{
		Raiz:     absoluta,
		MedidoEm: p.MedidoEm,
		PorTipo:  map[string]int{},
		NaoProduzido: []string{
			"THIRD_PARTY_NOTICES: exige o TEXTO da licenca de cada componente, que nao esta' no " +
				"go.mod nem no package.json",
			"arvore transitiva: resolveria exige `go mod graph` ou `npm ls`, que este provider " +
				"nao executa",
			"provenance/attestation: exige um builder identificado e um artefacto final, que nao " +
				"existem nesta leitura",
		},
		Aviso: "este inventario lista as dependencias DECLARADAS nos ficheiros de origem; nao " +
			"resolve transitivas, nao consulta a rede e nao afirma licenca que nao esteja' " +
			"declarada. `licenca` vazio significa NAO DECLARADA, nunca \"sem licenca\".",
	}

	// `go.mod` na raiz.
	goMod := filepath.Join(absoluta, "go.mod")
	if _, err := os.Stat(goMod); err == nil {
		modulos, err := lerGoMod(goMod)
		if err != nil {
			return inventario, err
		}
		inventario.FicheirosLidos = append(inventario.FicheirosLidos, "go.mod")
		inventario.Componentes = append(inventario.Componentes, modulos...)
	}

	// `package.json` na raiz e nos subdirectorios imediatos (monorepo).
	pacotes, lidos, err := lerPacotesNPM(ctx, absoluta)
	if err != nil {
		return inventario, err
	}
	inventario.FicheirosLidos = append(inventario.FicheirosLidos, lidos...)
	inventario.Componentes = append(inventario.Componentes, pacotes...)

	sort.SliceStable(inventario.Componentes, func(i, j int) bool {
		if inventario.Componentes[i].Tipo != inventario.Componentes[j].Tipo {
			return inventario.Componentes[i].Tipo < inventario.Componentes[j].Tipo
		}
		return inventario.Componentes[i].Nome < inventario.Componentes[j].Nome
	})

	for _, componente := range inventario.Componentes {
		inventario.Total++
		inventario.PorTipo[string(componente.Tipo)]++
		if componente.Licenca == "" {
			inventario.SemLicenca++
		}
	}
	return inventario, nil
}

// lerGoMod extrai os `require` de um `go.mod`.
//
// O parser e' deliberadamente simples: `go.mod` tem uma gramatica pequena e as linhas que interessam
// comecam por `require` ou estao dentro de um bloco `require ( ... )`. Um parser completo traria
// dependencias que este pacote nao precisa.
func lerGoMod(caminho string) ([]Componente, error) {
	ficheiro, err := os.Open(caminho) //nolint:gosec // caminho derivado de uma raiz dada
	if err != nil {
		return nil, fmt.Errorf("abrir %q: %w", caminho, err)
	}
	defer ficheiro.Close()

	var componentes []Componente
	dentroDeRequire := false
	leitor := bufio.NewScanner(ficheiro)
	for leitor.Scan() {
		linha := strings.TrimSpace(leitor.Text())
		if linha == "" || strings.HasPrefix(linha, "//") {
			continue
		}
		if strings.HasPrefix(linha, "require (") {
			dentroDeRequire = true
			continue
		}
		if dentroDeRequire && linha == ")" {
			dentroDeRequire = false
			continue
		}
		var corpo string
		switch {
		case dentroDeRequire:
			corpo = linha
		case strings.HasPrefix(linha, "require "):
			corpo = strings.TrimPrefix(linha, "require ")
		default:
			continue
		}
		// `corpo` e' `modulo versao` ou `modulo versao // indirect`.
		indireto := strings.Contains(corpo, "// indirect")
		if posicao := strings.Index(corpo, "//"); posicao >= 0 {
			corpo = strings.TrimSpace(corpo[:posicao])
		}
		campos := strings.Fields(corpo)
		if len(campos) < 2 {
			continue
		}
		componentes = append(componentes, Componente{
			Nome:        campos[0],
			Versao:      campos[1],
			Tipo:        ComponenteModuloGo,
			Direto:      !indireto,
			Origem:      "go.mod",
			DeclaradoEm: "go.mod",
		})
	}
	if err := leitor.Err(); err != nil {
		return nil, fmt.Errorf("ler %q: %w", caminho, err)
	}
	return componentes, nil
}

// pacoteJSON e' o subconjunto de `package.json` que interessa.
type pacoteJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	License         string            `json:"license"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// lerPacotesNPM le' o `package.json` da raiz e dos subdirectorios imediatos.
func lerPacotesNPM(ctx context.Context, raiz string) ([]Componente, []string, error) {
	var componentes []Componente
	var lidos []string

	candidatos := []string{filepath.Join(raiz, "package.json")}
	entradas, err := os.ReadDir(raiz)
	if err == nil {
		for _, entrada := range entradas {
			if !entrada.IsDir() || strings.HasPrefix(entrada.Name(), ".") ||
				entrada.Name() == "node_modules" {
				continue
			}
			candidatos = append(candidatos, filepath.Join(raiz, entrada.Name(), "package.json"))
			if len(candidatos) > maxFicheirosDeDependencia {
				break
			}
		}
	}

	for _, caminho := range candidatos {
		if err := ctx.Err(); err != nil {
			return nil, lidos, err
		}
		bruto, err := os.ReadFile(caminho) //nolint:gosec // caminho derivado de uma raiz dada
		if err != nil {
			continue
		}
		var pacote pacoteJSON
		if err := json.Unmarshal(bruto, &pacote); err != nil {
			// Um `package.json` invalido nao aborta o inventario: registar e seguir.
			//
			//	*Um erro que nao aborta nao e' um erro que nao aconteceu* -- por isso o ficheiro
			//	entra em `FicheirosLidos` como tentado, e o componente nao entra como se existisse.
			relativo, relErr := filepath.Rel(raiz, caminho)
			if relErr != nil {
				relativo = caminho
			}
			lidos = append(lidos, relativo+" (INVALIDO)")
			continue
		}
		relativo, relErr := filepath.Rel(raiz, caminho)
		if relErr != nil {
			relativo = caminho
		}
		lidos = append(lidos, relativo)
		origem := relativo
		for nome, versao := range pacote.Dependencies {
			componentes = append(componentes, Componente{
				Nome: nome, Versao: versao, Tipo: ComponentePacoteNPM,
				Direto: true, Origem: origem, DeclaradoEm: origem,
			})
		}
		for nome, versao := range pacote.DevDependencies {
			componentes = append(componentes, Componente{
				Nome: nome, Versao: versao, Tipo: ComponentePacoteNPM,
				Direto: false, Origem: origem, DeclaradoEm: origem,
			})
		}
	}
	return componentes, lidos, nil
}

// ChecksumDoComponente devolve uma chave estavel para o componente, para o SBOM poder ser comparado
// entre duas leituras.
//
//	*Um SBOM que muda de ordem a cada leitura nao permite dizer se mudou.*
func ChecksumDoComponente(componente Componente) string {
	return string(componente.Tipo) + ":" + componente.Nome + "@" + componente.Versao
}

// AssinaturaDoInventario compoe a lista ordenada de chaves. Dois inventarios com a mesma assinatura
// tem os mesmos componentes -- e e' isso que permite diff.
func AssinaturaDoInventario(inventario InventarioDeSoftware) []string {
	chaves := make([]string, 0, len(inventario.Componentes))
	for _, componente := range inventario.Componentes {
		chaves = append(chaves, ChecksumDoComponente(componente))
	}
	sort.Strings(chaves)
	return chaves
}
