package multillm

// FASE 28.5 do adendo V3: Signing Identity Center.
//
// O que este ficheiro e': o registo de METADADOS das identidades de assinatura. Ele estende o
// broker de credenciais que ja' existia (`credentials.go`, com DPAPI no Windows) sem criar um
// segundo cofre: *dois cofres para o mesmo segredo sao duas verdades, e a que fica desatualizada
// e' sempre a que ninguem lembrou de rodar.*
//
// INVARIANTE CENTRAL: **nao existe campo para o valor privado.**
// A estrutura `IdentidadeDeAssinatura` descreve ONDE a chave vive (provedor, referencia opaca),
// QUEM pode usa-la (ambiente, runner) e COMO esta' (fingerprint publico, validade). O valor em si
// continua no keychain/DPAPI/vault, alcancavel so' pelo broker, e nunca atravessa esta camada.
//
// *Um painel de identidades que mostra a chave nao e' um painel: e' um vazamento com interface.*
//
// A prova e' estrutural, nao declarativa: o teste `TestIdentidadeDeAssinaturaNaoTemCampoSecreto`
// percorre os campos por reflexao e falha se algum se chamar Secret/Private/Password/Token/Value.
// Assim a regra sobrevive a quem acrescentar um campo no futuro.

import (
	"crypto/sha1"
	"crypto/x509"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// AmbienteDeAssinatura e' onde a identidade pode ser usada.
//
// A separacao e' o ponto: *autorizar um build de preview nunca pode autorizar um build de
// producao -- sao duas decisoes, e junta-las faz uma delas desaparecer.*
type AmbienteDeAssinatura string

const (
	AmbienteDesenvolvimento AmbienteDeAssinatura = "desenvolvimento"
	AmbientePreview         AmbienteDeAssinatura = "preview"
	AmbienteProducao        AmbienteDeAssinatura = "producao"
)

// Valido diz se o ambiente e' um dos tres declarados. Ambiente inventado falha fechado: nunca
// e' promovido a "generico" por omissao.
func (a AmbienteDeAssinatura) Valido() bool {
	switch a {
	case AmbienteDesenvolvimento, AmbientePreview, AmbienteProducao:
		return true
	}
	return false
}

// ExigeAprovacao diz se usar esta identidade exige confirmacao humana no momento da accao.
//
// Producao exige. Preview nao. *O limiar tem de estar escrito em codigo, nao na cabeca de quem
// carrega no botao.*
func (a AmbienteDeAssinatura) ExigeAprovacao() bool { return a == AmbienteProducao }

// PlataformaDeAssinatura e' o alvo do artefacto assinado.
//
// Uma plataforma por identidade: nao existe assinador universal. *A tentacao de ter "signing"
// generico e' exactamente o que produz a alegacao falsa de que EAS assina Windows.*
type PlataformaDeAssinatura string

const (
	PlataformaAndroid PlataformaDeAssinatura = "android"
	PlataformaIOS     PlataformaDeAssinatura = "ios"
	PlataformaMacOS   PlataformaDeAssinatura = "macos"
	PlataformaWindows PlataformaDeAssinatura = "windows"
	PlataformaLinux   PlataformaDeAssinatura = "linux"
	PlataformaWeb     PlataformaDeAssinatura = "web"
	PlataformaOCI     PlataformaDeAssinatura = "oci"
)

// Valida diz se a plataforma e' uma das declaradas.
func (p PlataformaDeAssinatura) Valida() bool {
	switch p {
	case PlataformaAndroid, PlataformaIOS, PlataformaMacOS, PlataformaWindows,
		PlataformaLinux, PlataformaWeb, PlataformaOCI:
		return true
	}
	return false
}

// EstadoDaIdentidade e' o que o registo CONSEGUE afirmar sobre a identidade AGORA.
//
// `EstadoDesconhecido` existe de proposito: *nao conseguir determinar nao e' o mesmo que estar
// tudo bem, e converter um no outro e' como um painel verde mente sem ninguem notar.*
type EstadoDaIdentidade string

const (
	EstadoAtiva        EstadoDaIdentidade = "ativa"
	EstadoExpiraBreve  EstadoDaIdentidade = "expira_breve"
	EstadoExpirada     EstadoDaIdentidade = "expirada"
	EstadoRevogada     EstadoDaIdentidade = "revogada"
	EstadoDesconhecido EstadoDaIdentidade = "desconhecida"
	janelaExpiraBreve                     = 30 * 24 * time.Hour
)

// IdentidadeDeAssinatura descreve UMA identidade sem revelar nada de privado.
//
// Cada campo e' metadado publico ou referencia opaca. Ver a invariante no topo do ficheiro.
type IdentidadeDeAssinatura struct {
	// Nome e' o identificador estavel usado em logs e approvals.
	Nome string `json:"nome"`
	// Provedor e' onde a chave vive (keychain, dpapi, eas, kms, vault, hsm).
	Provedor string `json:"provedor"`
	// Referencia e' o ponteiro OPACO para o broker -- nunca o valor.
	// Ex.: "dpapi:apple/hades-apple-distribution.p12". *Uma referencia nao e' um segredo.*
	Referencia            string                 `json:"referencia"`
	Plataforma            PlataformaDeAssinatura `json:"plataforma"`
	Ambiente              AmbienteDeAssinatura   `json:"ambiente"`
	FingerprintPublico    string                 `json:"fingerprint_publico"`
	CriadaEm              time.Time              `json:"criada_em"`
	ExpiraEm              time.Time              `json:"expira_em"`
	RevogadaEm            *time.Time             `json:"revogada_em,omitempty"`
	RunnersAutorizados    []string               `json:"runners_autorizados"`
	PacoteOuBundle        string                 `json:"pacote_ou_bundle"`
	UltimoUso             *time.Time             `json:"ultimo_uso,omitempty"`
	RotacaoPrevistaEmDias int                    `json:"rotacao_prevista_em_dias,omitempty"`
}

// Estado calcula o estado da identidade no instante `agora`.
//
// A ordem importa: revogada vence expirada, expirada vence expira_breve. *Uma chave revogada ha'
// um ano nao esta' "expira_breve": esta' revogada, e dizer o contrario sugeriria que ainda serve.*
func (i IdentidadeDeAssinatura) Estado(agora time.Time) EstadoDaIdentidade {
	if i.RevogadaEm != nil {
		return EstadoRevogada
	}
	if i.ExpiraEm.IsZero() {
		// Sem validade conhecida nao se afirma saude.
		return EstadoDesconhecido
	}
	if !agora.Before(i.ExpiraEm) {
		return EstadoExpirada
	}
	if i.ExpiraEm.Sub(agora) <= janelaExpiraBreve {
		return EstadoExpiraBreve
	}
	return EstadoAtiva
}

// Utilizavel diz se a identidade pode assinar AGORA.
//
// Exige estado ativa ou expira_breve. Desconhecida NAO e' utilizavel: *assinar com uma chave de
// validade desconhecida e' aceitar o risco em nome de quem nao foi perguntado.*
func (i IdentidadeDeAssinatura) Utilizavel(agora time.Time) bool {
	switch i.Estado(agora) {
	case EstadoAtiva, EstadoExpiraBreve:
		return true
	}
	return false
}

// NormalizarFingerprint reduz um fingerprint SHA-1 (com ou sem dois-pontos) a hex minusculo.
//
// Aceita as duas formas que aparecem na vida real: `AA:BB:...` (keytool) e `aabb...` (OpenSSL).
// Devolve erro quando nao sao exactamente 20 bytes de hex -- *um fingerprint truncado parece
// valido e compara mal.*
func NormalizarFingerprint(bruto string) (string, error) {
	limpo := strings.ToLower(strings.NewReplacer(":", "", " ", "", "-", "").Replace(strings.TrimSpace(bruto)))
	if len(limpo) != sha1.Size*2 {
		return "", fmt.Errorf("fingerprint com %d caracteres hex; SHA-1 exige %d", len(limpo), sha1.Size*2)
	}
	for _, c := range limpo {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", fmt.Errorf("fingerprint com caractere nao-hex: %q", c)
		}
	}
	return limpo, nil
}

// FingerprintDeCertificadoDER calcula o SHA-1 publico de um certificado em DER.
//
// Serve para LER metadados de um certificado PUBLICO (o `.cer`), nunca de uma chave. *O
// fingerprint de um certificado publico e' publico por desenho; o que nao e' publico e' o que
// ele certifica quando ha' chave associada.*
func FingerprintDeCertificadoDER(der []byte) (string, error) {
	if len(der) == 0 {
		return "", errors.New("certificado vazio")
	}
	if _, err := x509.ParseCertificate(der); err != nil {
		return "", fmt.Errorf("nao e' um certificado X.509 DER: %w", err)
	}
	soma := sha1.Sum(der)
	partes := make([]string, len(soma))
	for i, b := range soma {
		partes[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(partes, ":"), nil
}

// RegistoDeIdentidades e' o conjunto de identidades conhecidas.
//
// Nao e' um cofre: e' o INDICE dele. O cofre ja' existe (`credentials.go`).
type RegistoDeIdentidades struct {
	identidades []IdentidadeDeAssinatura
}

// NovoRegistoDeIdentidades cria um registo vazio.
func NovoRegistoDeIdentidades() *RegistoDeIdentidades { return &RegistoDeIdentidades{} }

// Adicionar valida e acrescenta. Duplicar o Nome e' erro: *duas identidades com o mesmo nome
// tornam ambiguo o approval, e um approval ambiguo nao aprova nada.*
func (r *RegistoDeIdentidades) Adicionar(i IdentidadeDeAssinatura) error {
	if strings.TrimSpace(i.Nome) == "" {
		return errors.New("identidade sem nome")
	}
	if !i.Plataforma.Valida() {
		return fmt.Errorf("identidade %q: plataforma invalida %q", i.Nome, i.Plataforma)
	}
	if !i.Ambiente.Valido() {
		return fmt.Errorf("identidade %q: ambiente invalido %q", i.Nome, i.Ambiente)
	}
	if strings.TrimSpace(i.Provedor) == "" {
		return fmt.Errorf("identidade %q: sem provedor (onde a chave vive)", i.Nome)
	}
	if strings.TrimSpace(i.Referencia) == "" {
		return fmt.Errorf("identidade %q: sem referencia opaca para o broker", i.Nome)
	}
	if _, err := NormalizarFingerprint(i.FingerprintPublico); err != nil {
		return fmt.Errorf("identidade %q: %w", i.Nome, err)
	}
	for _, existente := range r.identidades {
		if existente.Nome == i.Nome {
			return fmt.Errorf("identidade %q ja' registada", i.Nome)
		}
	}
	r.identidades = append(r.identidades, i)
	return nil
}

// Identidades devolve uma copia ordenada por nome.
//
// Copia, nao a fatia interna: *devolver a fatia interna deixaria o chamador mudar o registo sem
// passar por validacao.*
func (r *RegistoDeIdentidades) Identidades() []IdentidadeDeAssinatura {
	saida := make([]IdentidadeDeAssinatura, len(r.identidades))
	copy(saida, r.identidades)
	sort.Slice(saida, func(a, b int) bool { return saida[a].Nome < saida[b].Nome })
	return saida
}

// ParaAssinar devolve as identidades utilizaveis para (plataforma, ambiente) em `agora`.
//
// Lista vazia e' resposta legitima e significa "nao ha' identidade valida para esta combinacao".
// *Nunca devolve a mais proxima: assinar com a identidade errada e' pior do que nao assinar.*
func (r *RegistoDeIdentidades) ParaAssinar(plataforma PlataformaDeAssinatura, ambiente AmbienteDeAssinatura, agora time.Time) []IdentidadeDeAssinatura {
	var saida []IdentidadeDeAssinatura
	for _, i := range r.identidades {
		if i.Plataforma != plataforma || i.Ambiente != ambiente {
			continue
		}
		if !i.Utilizavel(agora) {
			continue
		}
		// Um runner so' pode usar a identidade se estiver na lista E a lista nao estiver vazia.
		// Lista vazia significa "nenhum runner autorizado", nao "todos".
		if len(i.RunnersAutorizados) == 0 {
			continue
		}
		saida = append(saida, i)
	}
	sort.Slice(saida, func(a, b int) bool { return saida[a].Nome < saida[b].Nome })
	return saida
}

// ProblemaDeIdentidade e' um achado da auditoria, ligado a uma identidade.
type ProblemaDeIdentidade struct {
	Nome       string `json:"nome"`
	Severidade string `json:"severidade"`
	Mensagem   string `json:"mensagem"`
}

// Auditar devolve os problemas das identidades no instante `agora`.
//
// Criterios: expirada, expira_breve, sem runner, producao sem rotacao prevista, referencia que
// parece conter valor. *A auditoria nao conserta nada -- aponta, para um humano decidir.*
func (r *RegistoDeIdentidades) Auditar(agora time.Time) []ProblemaDeIdentidade {
	var problemas []ProblemaDeIdentidade
	for _, i := range r.identidades {
		estado := i.Estado(agora)
		switch estado {
		case EstadoExpirada:
			problemas = append(problemas, ProblemaDeIdentidade{
				Nome: i.Nome, Severidade: "critico",
				Mensagem: "identidade expirada em " + i.ExpiraEm.Format(time.RFC3339),
			})
		case EstadoRevogada:
			problemas = append(problemas, ProblemaDeIdentidade{
				Nome: i.Nome, Severidade: "critico", Mensagem: "identidade revogada",
			})
		case EstadoExpiraBreve:
			problemas = append(problemas, ProblemaDeIdentidade{
				Nome: i.Nome, Severidade: "aviso",
				Mensagem: "identidade expira em " + i.ExpiraEm.Format(time.RFC3339),
			})
		case EstadoDesconhecido:
			problemas = append(problemas, ProblemaDeIdentidade{
				Nome: i.Nome, Severidade: "aviso", Mensagem: "validade desconhecida",
			})
		}
		if len(i.RunnersAutorizados) == 0 {
			problemas = append(problemas, ProblemaDeIdentidade{
				Nome: i.Nome, Severidade: "critico",
				Mensagem: "nenhum runner autorizado: a identidade e' inutilizavel por desenho",
			})
		}
		if i.Ambiente == AmbienteProducao && i.RotacaoPrevistaEmDias <= 0 {
			problemas = append(problemas, ProblemaDeIdentidade{
				Nome: i.Nome, Severidade: "aviso",
				Mensagem: "identidade de producao sem rotacao prevista",
			})
		}
		if ReferenciaPareceSegredo(i.Referencia) {
			problemas = append(problemas, ProblemaDeIdentidade{
				Nome: i.Nome, Severidade: "critico",
				Mensagem: "a referencia opaca parece conter um valor secreto: " +
					"ela deve apontar para o broker, nao transporter o segredo",
			})
		}
	}
	sort.Slice(problemas, func(a, b int) bool {
		if problemas[a].Severidade != problemas[b].Severidade {
			return problemas[a].Severidade < problemas[b].Severidade
		}
		return problemas[a].Nome < problemas[b].Nome
	})
	return problemas
}

// ReferenciaPareceSegredo reconhece formatos que NAO podem aparecer numa referencia opaca.
//
// A lista e' de PREFIXOS PUBLICAMENTE CONHECIDOS de credenciais -- nao de valores reais. *Um
// detector que precisa de exemplos reais para funcionar e' um detector que ja' vazou.*
func ReferenciaPareceSegredo(referencia string) bool {
	minusculo := strings.ToLower(referencia)
	for _, prefixo := range []string{
		"ghp_", "gho_", "ghu_", "ghs_", "ghr_", // GitHub
		"sk-", "sk_live_", "sk_test_", // APIs
		"-----begin",        // bloco PEM
		"bearer ", "basic ", // cabecalhos de auth
		"akia", "asia", // chaves AWS
	} {
		if strings.Contains(minusculo, prefixo) {
			return true
		}
	}
	// Um base64 muito longo numa "referencia" e' quase sempre o valor colado por engano.
	if len(referencia) > 200 && !strings.Contains(referencia, "/") && !strings.Contains(referencia, ":") {
		return true
	}
	return false
}

// ResumoDoRegisto e' a projecao segura para o Mission Control.
//
// Contagens e nomes de estado. Nenhum campo derivado de segredo.
type ResumoDoRegisto struct {
	Total       int            `json:"total"`
	PorEstado   map[string]int `json:"por_estado"`
	PorAmbiente map[string]int `json:"por_ambiente"`
	Problemas   int            `json:"problemas"`
}

// Resumir agrega o registo no instante `agora`.
func (r *RegistoDeIdentidades) Resumir(agora time.Time) ResumoDoRegisto {
	resumo := ResumoDoRegisto{
		Total:       len(r.identidades),
		PorEstado:   map[string]int{},
		PorAmbiente: map[string]int{},
	}
	resumo.Problemas = len(r.Auditar(agora))
	for _, i := range r.identidades {
		resumo.PorEstado[string(i.Estado(agora))]++
		resumo.PorAmbiente[string(i.Ambiente)]++
	}
	return resumo
}
