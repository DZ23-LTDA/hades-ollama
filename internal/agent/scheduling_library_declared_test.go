package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// E-441 (ADV-65, P0 PARA A ENTREGA) — **O MOTOR DE AGENDAMENTO E' BIBLIOTECA SEM CHAMADOR. DECLARADO, COM CATRACA.**
//
// **MEDIDO (rodada 150)**: busca ampla por `NewSchedulingStore` em **todo** o repositorio:
//
//	EM PRODUCAO (fora de _test): 2  <- e sao o COMENTARIO e a DEFINICAO, em `scheduling.go` L115-116
//	TESTE: 17 ocorrencias
//
// ⇒ *`SchedulingStore` — com `AddService`, `AddResource`, `Book`, `Cancel`, `Reschedule`, fila de espera e no-show —
// **nunca e' construido fora dos testes*** ✗ ⇒ ***o motor existe, e' testado, e **nao esta' ligado a nada**: e' a
// classe "capacidade sem superficie" que o proprio Estagio 26 do projeto ja' nomeou*** ✓.
//
// **A CATRACA DISPAROU (rodada 164), e este e' o registo da inversao.** Em E-441 o motor **nao** tinha chamador; a
// declaracao foi escrita **com esta catraca**, cuja mensagem dizia: *"Atualize a evidencia E-441 e o registro do ADV-65,
// dizendo ONDE o motor esta' ligado - em vez de apagar esta catraca."* Quando a rodada 162 ligou a superficie
// (`server/scheduling_routes.go` + `agent.NewSchedulingStore` em `agent_routes.go`), ela disparou:
//
//	DECLARACAO VENCIDA: o motor de agendamento passou a ter CHAMADOR em producao: server/scheduling_routes.go…
//
// ⇒ ***A catraca nao protegia contra o erro: protegia contra a **VERDADE MUDAR EM SILENCIO** - que foi o defeito do
// MEDIO-1 (E-439) e do numero "2 pendencias" (E-432).***
//
// **AGORA ELA GUARDA A VERDADE NOVA** (E-449): o motor **e' ligado em producao**, e a catraca **falha se o chamador
// DESAPARECER** — *desligar a superficie sem retirar esta declaracao repetiria o ADV-65, e ninguem notaria.*
//
// **O QUE ESTA CATRACA NAO FAZ**: nao diz que as rotas estao corretas nem que cobrem o motor inteiro ✗ - so' que a
// ligacao existe.
func TestMotorDeAgendamentoTemChamadorEmProducao(t *testing.T) {
	raiz := filepath.Join("..", "..")
	chamada := regexp.MustCompile(`NewSchedulingStore\s*\(`)
	definicao := regexp.MustCompile(`^func NewSchedulingStore\s*\(`)

	var emProducao []string
	err := filepath.WalkDir(raiz, func(caminho string, entrada os.DirEntry, erro error) error {
		if erro != nil {
			return nil // *caminho ilegivel nao e' achado: e' ruido de varredura.*
		}
		if entrada.IsDir() {
			nome := entrada.Name()
			// `.git` e `_tmp` ficam de fora: artefato e historico nao sao codigo de produto.
			if nome == ".git" || nome == "_tmp" || nome == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(caminho, ".go") || strings.HasSuffix(caminho, "_test.go") {
			return nil
		}
		dados, erroLeitura := os.ReadFile(caminho)
		if erroLeitura != nil {
			return nil
		}
		for numero, linha := range strings.Split(string(dados), "\n") {
			if !chamada.MatchString(linha) || definicao.MatchString(strings.TrimSpace(linha)) {
				continue
			}
			// Comentario tambem nao e' chamador: *a linha L115 e' o doc da funcao, e conta-la seria falso positivo.*
			aparado := strings.TrimSpace(linha)
			if strings.HasPrefix(aparado, "//") {
				continue
			}
			emProducao = append(emProducao, filepath.ToSlash(caminho)+":"+strconv.Itoa(numero+1)+" "+aparado)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varredura: %v", err)
	}

	if len(emProducao) == 0 {
		t.Fatal("LIGACAO DESAPARECEU: o motor de agendamento NAO tem mais chamador em producao — *a superficie de " +
			"`server/scheduling_routes.go` foi desligada sem ninguem notar, e o ADV-65 voltou a ficar aberto*. " +
			"Religue a superficie, ou atualize esta declaracao e a evidencia E-449 dizendo por que ela saiu.")
	}

	// CONTROLE: e a varredura realmente ALCANCA o arquivo que define o motor? *Senao o teste passaria por nao achar
	// nada - exatamente a classe do instrumento que nunca mede.*
	caminhoMotor := filepath.Join(raiz, "internal", "agent", "scheduling.go")
	if _, erroStat := os.Stat(caminhoMotor); erroStat != nil {
		t.Fatalf("a varredura nao alcancou %s (%v) - *sem isto, o teste passaria por nao medir nada*", caminhoMotor, erroStat)
	}
	dados, _ := os.ReadFile(caminhoMotor)
	if !strings.Contains(string(dados), "func NewSchedulingStore(") {
		t.Fatal("o arquivo do motor foi alcancado mas nao contem a definicao - *a sonda esta' medindo o arquivo errado*")
	}
}
