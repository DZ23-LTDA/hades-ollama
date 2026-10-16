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
// E-912 (finding 7, medido): a varredura procurava **`NewSchedulingStore\s*\(`** - *o **parentese** exigia que o
// chamador fosse uma **chamada direta***. Se o uso em producao passasse a ser **indireto**
// (`f := agent.NewSchedulingStore`, *method value*, entrada numa tabela de funcoes, comparacao com `nil`), a regex
// **nao o encontrava** ⇒ `emProducao == 0` ⇒ *a catraca **falhava com a mensagem errada** («LIGACAO DESAPARECEU»)
// quando a ligacao **existia** - so' tinha mudado de forma.*
//
// Um instrumento que da' **falso alarme** e' tao ruim quanto um que nao ve: *o primeiro treina a ignorar a
// catraca, o segundo esconde a regressao.*
//
// **Cura**: a varredura passa a procurar o **SIMBOLO** (`\bNewSchedulingStore\b`), e nao a chamada - *mas as
// exclusoes continuam a valer, e agora tambem excluem a **declaracao** (`func`), nao so' a definicao do construtor.*
// O comentario ja' era excluido (`:74`), e o controlo (`:91-100`) prova que a varredura **alcanca** o ficheiro.
func TestMotorDeAgendamentoTemChamadorEmProducao(t *testing.T) {
	raiz := filepath.Join("..", "..")
	// E-912: o SIMBOLO, nao a chamada. `\b` nos dois lados evita casar `NewSchedulingStoreX`.
	chamada := regexp.MustCompile(`\bNewSchedulingStore\b`)
	definicao := regexp.MustCompile(`^func NewSchedulingStore\s*\(`)

	var emProducao []string
	var saltados []string // E-503: *o que a varredura nao conseguiu ler nao some: fica aqui.*
	err := filepath.WalkDir(raiz, func(caminho string, entrada os.DirEntry, erro error) error {
		if erro != nil {
			saltados = append(saltados, caminho+": "+erro.Error())
			//nolint:nilerr // *caminho ilegivel nao invalida a varredura: o salto fica REGISTADO em `saltados`, e abortar aqui esconderia os restantes ficheiros.*
			return nil
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
			saltados = append(saltados, caminho+": "+erroLeitura.Error())
			//nolint:nilerr // *uma leitura que falhou nao invalida a varredura: o ficheiro entra em `saltados` com o motivo, e o laco continua.*
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
