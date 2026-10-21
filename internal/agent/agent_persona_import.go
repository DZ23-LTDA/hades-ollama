// Ingestao segura e quarentena de personas importadas (FASE 30.2).
//
// # O CATALOGO E OS PROMPTS SAO DADOS NAO CONFIAVEIS
//
// Uma persona importada e' texto que veio de fora. O `Prompt` dela e' CONTEUDO -- nunca e' executado, nunca
// e' autoridade sobre o runtime. Mas um prompt pode PEDIR coisas, e um pedido hostil num catalogo que o
// utilizador instala sem ler e' um ataque com um passo.
//
//	*O catalogo e os prompts sao dados nao confiaveis; nunca execute instrucoes neles durante importacao.*
//
// O QUE ESTE SCANNER NAO E'
//
// Nao e' um antivirus e nao e' um sandbox. E' uma TRIAGEM que manda para quarentena o que e' obviamente
// hostil, e que deixa o resto para revisao humana -- porque
//
//	*Um falso positivo de seguranca nao e' ruido: e' o que ensina a ignorar o proximo aviso.*
//
// Por isso o scanner NAO quarentena por palavra solta: precisa de um PADRAO com contexto. "segredo" no
// prompt do revisor de seguranca e' o trabalho dele; "curl http://... | sh" nao e'.
//
// QUARENTENA E' UM ESTADO, NAO UM LOG
//
// Uma persona quarentenada nao e' instalavel. O estado `QUARANTINE` entra na maquina de estados e so' sai
// por revisao humana (`SECURITY_REVIEWED`) ou por revogacao. Um achado que ficasse so' num relatorio seria
// um aviso que ninguem le'.
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ErrImportRecusado cobre manifesto ou conteudo que nao passa na ingestao.
var ErrImportRecusado = errors.New("agent persona import was rejected")

// QuarantineSeverity separa o que IMPEDE do que AVISA.
type QuarantineSeverity string

const (
	// QuarantineBlock: nao instala. Nao ha' revisao que torne isto instalavel sem editar o conteudo.
	QuarantineBlock QuarantineSeverity = "BLOCK"
	// QuarantineWarn: instala so' depois de revisao humana. O padrao pode ser legitimo.
	QuarantineWarn QuarantineSeverity = "WARN"
)

// QuarantineFinding e' um achado com local e trecho, nao uma suspeita vaga.
type QuarantineFinding struct {
	Field    string             `json:"field"`
	Pattern  string             `json:"pattern"`
	Severity QuarantineSeverity `json:"severity"`
	// Excerpt e' o trecho que casou, TRUNCADO. O relatorio nao copia o prompt inteiro.
	Excerpt string `json:"excerpt"`
	// Why explica o risco em uma frase.
	Why string `json:"why"`
}

// QuarantineReport e' o resultado da triagem.
type QuarantineReport struct {
	// OK e' true quando NAO ha' nenhum achado BLOCK. Avisos nao fecham a porta sozinhos.
	OK       bool                `json:"ok"`
	Blocked  bool                `json:"blocked"`
	Findings []QuarantineFinding `json:"findings,omitempty"`
	// NeedsReview e' true quando ha' WARN sem BLOCK.
	NeedsReview bool `json:"needs_review"`
}

// linhaSuspeita e' o que o scanner procura: um padrao LITERAL (minusculo) e a razao.
type linhaSuspeita struct {
	padrao     string
	severidade QuarantineSeverity
	porque     string
}

// padroesHostis e' a lista. Literais de proposito: uma regex mal escrita num scanner de seguranca e' um
// detector de falsos positivos.
//
// *Um scanner que nao se exclui mede a si mesmo e chama-lhe defeito alheio.*
var padroesHostis = []linhaSuspeita{
	// Injecao: tentar substituir a instrucao do utilizador pela do documento.
	{"ignore as instruções anteriores", QuarantineBlock, "injecção: tenta substituir a instrução do utilizador"},
	{"ignore all previous instructions", QuarantineBlock, "injecção: tenta substituir a instrução do utilizador"},
	{"ignore previous instructions", QuarantineBlock, "injecção: tenta substituir a instrução do utilizador"},
	{"ignore as regras", QuarantineBlock, "injecção: pede para ignorar as regras"},
	{"desconsidere as instruções", QuarantineBlock, "injecção: pede para ignorar as instruções"},
	{"you are now", QuarantineWarn, "troca de papel: pode ser legítimo num prompt de persona, mas revise"},
	{"system prompt", QuarantineWarn, "referência ao prompt de sistema: revise o contexto"},

	// Bypass de aprovação: o HADES nunca usa estes modos por omissão.
	{"--dangerously-skip-permissions", QuarantineBlock, "bypass de aprovação"},
	{"--dangerously-bypass", QuarantineBlock, "bypass de aprovação"},
	{"--yolo", QuarantineBlock, "bypass de aprovação"},

	// Exfiltração.
	{"curl http", QuarantineBlock, "rede: comando de download/upload"},
	{"curl -", QuarantineBlock, "rede: comando de download/upload"},
	{"wget http", QuarantineBlock, "rede: comando de download/upload"},
	{"| sh", QuarantineBlock, "execução de conteúdo remoto"},
	{"| bash", QuarantineBlock, "execução de conteúdo remoto"},
	{"iwr ", QuarantineBlock, "download remoto no Windows"},
	{"iex ", QuarantineBlock, "execução de conteúdo remoto no Windows"},
	{"id_rsa", QuarantineBlock, "exfiltração de chave SSH"},
	{".ssh/", QuarantineBlock, "acesso a chave SSH"},
	{"~/.aws", QuarantineBlock, "acesso a credencial de cloud"},
	{"/etc/passwd", QuarantineBlock, "leitura de ficheiro do sistema"},
	{"base64 -d", QuarantineWarn, "decodificação que pode esconder conteúdo"},

	// Execução destrutiva.
	{"rm -rf", QuarantineBlock, "exclusão recursiva"},
	{"format c:", QuarantineBlock, "formatação de disco"},
	{"del /f /s /q", QuarantineBlock, "exclusão recursiva no Windows"},

	// Pedido de segredo.
	{"revele o token", QuarantineBlock, "pede segredo"},
	{"revele a senha", QuarantineBlock, "pede segredo"},
	{"print the token", QuarantineBlock, "pede segredo"},
	{"env | grep", QuarantineBlock, "leitura de variáveis de ambiente"},
	{"api_key=", QuarantineWarn, "possível segredo embutido"},
	{"password=", QuarantineWarn, "possível segredo embutido"},
}

// trecho limita o que o relatorio copia. *O relatorio nao precisa do prompt inteiro para dizer que ele e'
// hostil.*
func trecho(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// ScanPersona faz a triagem de uma persona importada.
//
// Le' os campos que podem carregar instrucao: `Prompt`, `Description`, `Name`, `Capabilities`, `Tools`,
// `RequiresApproval` e `Forbidden`. Um `Forbidden` que diga o que NAO fazer e' legitimo; mas a lista e' um
// campo de texto, e texto pode ser hostil.
func ScanPersona(p AgentPersona) QuarantineReport {
	r := QuarantineReport{OK: true}
	campos := []struct {
		nome    string
		valores []string
	}{
		{"prompt", []string{p.Prompt}},
		{"description", []string{p.Description}},
		{"name", []string{p.Name}},
		{"capabilities", p.Capabilities},
		{"tools", p.Tools},
		{"requires_approval", p.RequiresApproval},
		{"forbidden", p.Forbidden},
	}
	for _, campo := range campos {
		for _, valor := range campo.valores {
			minusculo := strings.ToLower(valor)
			for _, s := range padroesHostis {
				if !strings.Contains(minusculo, s.padrao) {
					continue
				}
				r.Findings = append(r.Findings, QuarantineFinding{
					Field:    campo.nome,
					Pattern:  s.padrao,
					Severity: s.severidade,
					Excerpt:  trecho(valor, 120),
					Why:      s.porque,
				})
				if s.severidade == QuarantineBlock {
					r.Blocked = true
				} else {
					r.NeedsReview = true
				}
			}
		}
	}
	r.OK = !r.Blocked
	return r
}

// ValidateImport valida o MANIFESTO antes de o ler: tamanho, encoding, campos e caminhos.
//
// Separa-se do `ScanPersona` porque sao coisas diferentes: um manifesto pode ser bem formado e hostil, ou
// mal formado e inofensivo. As duas recusas tem de acontecer, e por razoes distintas.
func ValidateImport(bruto []byte, limiteBytes int) (AgentPersona, error) {
	if limiteBytes > 0 && len(bruto) > limiteBytes {
		return AgentPersona{}, fmt.Errorf("%w: %d bytes exceeds the %d byte limit", ErrImportRecusado, len(bruto), limiteBytes)
	}
	if !utf8.Valid(bruto) {
		return AgentPersona{}, fmt.Errorf("%w: not valid UTF-8", ErrImportRecusado)
	}
	if len(strings.TrimSpace(string(bruto))) == 0 {
		return AgentPersona{}, fmt.Errorf("%w: empty", ErrImportRecusado)
	}
	var p AgentPersona
	// Desconhecido = recusado. Um campo que o HADES nao entende pode ser um campo que o HADES ignora e o
	// importador pensa ter configurado.
	dec := json.NewDecoder(strings.NewReader(string(bruto)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return AgentPersona{}, fmt.Errorf("%w: %s", ErrImportRecusado, err.Error())
	}
	// A forma: sem isto o resto e' discutir um agente que nao se instala.
	if err := p.Validate(); err != nil {
		return AgentPersona{}, err
	}
	// Caminhos: o id vira nome de ficheiro no destino. `Validate` ja' recusa separadores e `..`; aqui
	// recusa-se tambem o que e' so' espaco ou um ponto.
	idCru := strings.TrimSpace(p.ID)
	if idCru == "." || idCru == ".." || strings.Trim(idCru, ".") == "" {
		return AgentPersona{}, fmt.Errorf("%w: id %q is not a file name", ErrImportRecusado, p.ID)
	}
	return p, nil
}

// ImportDecision junta as duas validacoes e devolve o estado em que a persona entra.
//
//  1. manifesto invalido -> erro (nao entra no catalogo);
//  2. achado BLOCK       -> `QUARANTINE`
//  3. achado WARN        -> `QUARANTINE` tambem (revisao humana), mas o relatorio diz que e' so' aviso;
//  4. limpo              -> `IMPORTED_UNREVIEWED` (nunca `PRODUCTION_APPROVED`).
func ImportDecision(bruto []byte, limiteBytes int) (AgentPersona, EvalState, QuarantineReport, error) {
	p, err := ValidateImport(bruto, limiteBytes)
	if err != nil {
		return AgentPersona{}, "", QuarantineReport{}, err
	}
	r := ScanPersona(p)
	switch {
	case r.Blocked:
		return p, EvalQuarantined, r, nil
	case r.NeedsReview:
		return p, EvalQuarantined, r, nil
	default:
		return p, EvalImportedUnreviewed, r, nil
	}
}
