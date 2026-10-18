package cmd

// FASE 28.7 e FASE 8 do adendo V3: expor o resolvedor de toolchain na CLI.
//
// O resolvedor da FASE 28.6 ja' existia em `internal/agent/toolchain_plan.go`, mas so' era
// alcancavel por codigo. *Uma capacidade que so' existe dentro do processo nao e' uma capacidade
// do produto: e' um detalhe de implementacao.* Estes comandos sao o caminho humano e o caminho
// automatizavel (`--json`) para a mesma decisao.
//
// IMPORTANTE: ao contrario dos outros subcomandos de `agent`, este NAO fala com o endpoint HTTP.
// Ele sonda o PATH da maquina onde corre. Isso e' deliberado: quem precisa de uma ferramenta para
// construir e' a maquina que constroi, nao o servidor de missoes.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ollama/ollama/internal/agent"
	"github.com/spf13/cobra"
)

// Exit codes do `toolchain` (documentados e estaveis, FASE 8).
//
//	0  pronto     -- todas as exigencias obrigatorias estao saudaveis (retorno nil)
//	2  bloqueado  -- ha' plano, mas falta ferramenta obrigatoria
//	1  erro       -- alvo desconhecido ou falha de escrita
const (
	// exitToolchainBloqueado: o alvo existe, mas falta ferramenta obrigatoria.
	//
	// Reutiliza o codigo 2 do `doctor` de proposito: para um script, "ha' um plano mas ele nao
	// pode correr" e' o mesmo tipo de resposta que "o endpoint responde mas esta' degradado".
	exitToolchainBloqueado = 2
	// exitToolchainErro: alvo desconhecido ou falha de escrita. Nao ha' plano nenhum.
	exitToolchainErro = 1
)

// toolchainErro carrega o codigo de saida para o `main` traduzir em `os.Exit`.
type toolchainErro struct {
	Codigo int
	Err    error
}

func (e toolchainErro) Error() string { return e.Err.Error() }
func (e toolchainErro) Unwrap() error { return e.Err }

// ExitCode permite ao chamador (e aos testes) distinguir "bloqueado" de "erro".
func (e toolchainErro) ExitCode() int { return e.Codigo }

// toolchainListagem e' a projecao estavel para `--json` no `list`.
//
// Projecao MINIMA de proposito: o consumidor precisa do nome e do porque', nao de ponteiros
// internos. *Um `--json` que despeja objetos internos amarra o consumidor a implementacao.*
type toolchainListagem struct {
	Alvos []toolchainAlvoResumo `json:"alvos"`
}

type toolchainAlvoResumo struct {
	Nome            string   `json:"nome"`
	Descricao       string   `json:"descricao"`
	HostNecessario  string   `json:"host_necessario,omitempty"`
	Exigencias      []string `json:"exigencias"`
	IndependeDeTudo []string `json:"trabalho_independente,omitempty"`
}

// toolchainRelatorio e' a projecao estavel de um `plan`.
type toolchainRelatorio struct {
	Alvo      string               `json:"alvo"`
	Pronto    bool                 `json:"pronto"`
	Probes    []toolchainProbeView `json:"probes"`
	Blockers  []string             `json:"blockers"`
	Avanca    []string             `json:"trabalho_independente"`
	Veredicto string               `json:"veredicto"`
}

type toolchainProbeView struct {
	Nome        string `json:"nome"`
	Estado      string `json:"estado"`
	Obrigatorio bool   `json:"obrigatorio"`
	Caminho     string `json:"caminho,omitempty"`
	Versao      string `json:"versao,omitempty"`
	Motivo      string `json:"motivo"`
	Detalhe     string `json:"detalhe"`
}

// toolchainMasculino devolve o comando `agent toolchain`.
func toolchainCommand() *cobra.Command {
	raiz := &cobra.Command{
		Use:   "toolchain",
		Short: "Preflight de ferramentas por alvo de build (nao fala com o servidor)",
		Long: "Sondar as ferramentas que um alvo de build exige, nesta maquina.\n\n" +
			"Exit codes (documentados, estaveis):\n" +
			"  0  pronto      -- todas as exigencias obrigatorias estao saudaveis\n" +
			"  2  bloqueado   -- ha' plano, mas falta ferramenta obrigatoria\n" +
			"  1  erro        -- alvo desconhecido ou falha de escrita\n\n" +
			"Um estado `PRESENT_COMPATIBLE_UNVERIFIED` NAO conta como pronto: significa que a\n" +
			"ferramenta respondeu `--version`, nao que ela executa o trabalho.\n\n" +
			"*Um plano que se declara pronto com ferramentas nao verificadas nao adianta o\n" +
			"trabalho: adia a descoberta do problema para o momento mais caro.*",
	}

	listar := &cobra.Command{
		Use:   "list",
		Short: "Listar os alvos de build declarados",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			listagem := toolchainListagem{Alvos: []toolchainAlvoResumo{}}
			for _, alvo := range agent.CatalogoDeAlvos() {
				resumo := toolchainAlvoResumo{
					Nome:            alvo.Nome,
					Descricao:       alvo.Descricao,
					HostNecessario:  alvo.ApenasNativo,
					Exigencias:      []string{},
					IndependeDeTudo: alvo.Independen,
				}
				for _, req := range alvo.Exigencias {
					marca := ""
					if req.Obrigatorio {
						marca = " (obrigatoria)"
					}
					resumo.Exigencias = append(resumo.Exigencias, req.Nome+marca+" -- "+req.Motivo)
				}
				listagem.Alvos = append(listagem.Alvos, resumo)
			}
			return escreverToolchain(listagem)
		},
	}

	plano := &cobra.Command{
		Use:   "plan ALVO",
		Short: "Resolver as ferramentas de um alvo e dizer o que falta",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			relatorio, err := resolverAlvo(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := escreverToolchain(relatorio); err != nil {
				return err
			}
			if !relatorio.Pronto {
				return toolchainErro{
					Codigo: exitToolchainBloqueado,
					Err:    fmt.Errorf("alvo %q bloqueado: %s", relatorio.Alvo, strings.Join(relatorio.Blockers, "; ")),
				}
			}
			return nil
		},
	}

	doctor := &cobra.Command{
		Use:   "doctor",
		Short: "Resolver todos os alvos e resumir o que esta bloqueado",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			nomes := make([]string, 0, len(agent.CatalogoDeAlvos()))
			for _, alvo := range agent.CatalogoDeAlvos() {
				nomes = append(nomes, alvo.Nome)
			}
			sort.Strings(nomes)

			type resumo struct {
				Alvo   string `json:"alvo"`
				Pronto bool   `json:"pronto"`
				Falta  int    `json:"exigencias_bloqueantes"`
			}
			saida := struct {
				Alvos []resumo `json:"alvos"`
			}{Alvos: []resumo{}}
			bloqueados := 0
			incompletos := 0
			var primeiroErro error
			for _, nome := range nomes {
				rel, err := resolverAlvo(cmd.Context(), nome)
				if err != nil {
					if primeiroErro == nil {
						primeiroErro = err
					}
					continue
				}
				// Bloqueado e' `len(Blockers) > 0`, NAO `!Pronto`: um alvo com ferramentas
				// presentes mas sem health funcional esta' INCOMPLETO. Conta-lo como bloqueado
				// fazia o resumo dizer "6 de 6 bloqueados" quando so' 2 tinham blocker -- e o
				// utilizador iria procurar ferramentas que ja' tem.
				//
				// *Falta de prova nao e' prova de bloqueio.*
				switch {
				case len(rel.Blockers) > 0:
					bloqueados++
				case !rel.Pronto:
					incompletos++
				}
				saida.Alvos = append(saida.Alvos, resumo{Alvo: rel.Alvo, Pronto: rel.Pronto, Falta: len(rel.Blockers)})
			}
			if err := escreverToolchain(saida); err != nil {
				return err
			}
			if primeiroErro != nil {
				return primeiroErro
			}
			if bloqueados > 0 {
				return toolchainErro{
					Codigo: exitToolchainBloqueado,
					Err: fmt.Errorf("%d de %d alvos bloqueados, %d incompletos (ferramentas presentes, falta health funcional)",
						bloqueados, len(saida.Alvos), incompletos),
				}
			}
			if incompletos > 0 {
				return toolchainErro{
					Codigo: exitToolchainBloqueado,
					Err: fmt.Errorf("nenhum alvo bloqueado, mas %d de %d incompletos (falta health funcional)",
						incompletos, len(saida.Alvos)),
				}
			}
			return nil
		},
	}

	raiz.AddCommand(listar, plano, doctor)
	return raiz
}

// resolverAlvo corre o resolvedor e projeta o resultado.
func resolverAlvo(ctx context.Context, alvo string) (toolchainRelatorio, error) {
	resolver := agent.NovoResolverDeToolchain()
	p, err := resolver.Plano(ctx, alvo)
	if err != nil {
		return toolchainRelatorio{}, toolchainErro{Codigo: exitToolchainErro, Err: err}
	}
	relatorio := toolchainRelatorio{
		Alvo:     p.Alvo,
		Pronto:   p.Pronto(),
		Blockers: p.Blockers,
		Avanca:   p.TrabalhoIndependente,
	}
	if relatorio.Blockers == nil {
		relatorio.Blockers = []string{}
	}
	if relatorio.Avanca == nil {
		relatorio.Avanca = []string{}
	}
	for _, probe := range p.Probes {
		relatorio.Probes = append(relatorio.Probes, toolchainProbeView{
			Nome:        probe.Requisito.Nome,
			Estado:      string(probe.Estado),
			Obrigatorio: probe.Requisito.Obrigatorio,
			Caminho:     probe.Caminho,
			Versao:      probe.Versao,
			Motivo:      probe.Requisito.Motivo,
			Detalhe:     probe.Detalhe,
		})
	}
	switch {
	case relatorio.Pronto:
		relatorio.Veredicto = "pronto"
	case len(relatorio.Blockers) > 0:
		// Ha' ferramenta obrigatoria ausente/quebrada/desactualizada: e' bloqueio.
		relatorio.Veredicto = "bloqueado"
	default:
		// Nao esta' pronto mas nada bloqueia: as ferramentas existem e responderam, so' nao
		// foram verificadas funcionalmente (health funcional ainda nao executado).
		//
		// *Falta de prova nao e' prova de bloqueio.* Chamar "bloqueado" a isto faria o
		// utilizador procurar uma ferramenta que ja' tem.
		relatorio.Veredicto = "incompleto"
	}
	return relatorio, nil
}

// escreverToolchain respeita o `--json` global de `agent`.
func escreverToolchain(valor any) error {
	encoder := json.NewEncoder(agentCLIOut)
	if !agentJSONOutput {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(valor)
}

// ExitCodeDoErro devolve o codigo de saida declarado por um erro, ou -1.
//
// Existe para o teste poder verificar o contrato de exit code sem chamar `os.Exit`.
func ExitCodeDoErro(err error) int {
	var comCodigo interface{ ExitCode() int }
	if errors.As(err, &comCodigo) {
		return comCodigo.ExitCode()
	}
	return -1
}
