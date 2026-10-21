package multillm

// FASE 28.5 (continuacao): carregar o registo de identidades de um ficheiro de METADADOS.
//
// O ficheiro vive FORA do repositorio, ao lado do broker de credenciais, e contem apenas o que
// `IdentidadeDeAssinatura` declara: nomes, provedores, referencias opacas, fingerprints PUBLICOS,
// validades e runners. Nunca contem o valor privado -- a invariante estrutural do tipo garante-o,
// e o carregador recusa um ficheiro que traga campos desconhecidos, para que um campo novo nao
// entre por engano.
//
// *Um ficheiro de configuracao que aceita campos que o tipo nao conhece e' um canal para
// contrabandear o segredo pela porta de servico.*

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// NomeDoFicheiroDeIdentidades e' o nome procurado dentro do diretorio do broker.
	NomeDoFicheiroDeIdentidades = "signing-identities.json"
	// maxIdentidadesBytes limita a leitura: um registo de metadados e' pequeno por natureza.
	maxIdentidadesBytes = 256 << 10
	// esquemaSuportado e' a versao do formato. Versao desconhecida falha fechado.
	esquemaSuportado = 1
)

// documentoDeIdentidades e' a forma do ficheiro em disco.
type documentoDeIdentidades struct {
	SchemaVersion int                      `json:"schema_version"`
	Identidades   []IdentidadeDeAssinatura `json:"identidades"`
}

// CaminhoDoFicheiroDeIdentidades devolve o caminho esperado dentro do diretorio do broker.
//
// Devolve vazio quando o diretorio nao pode ser determinado -- e nesse caso o chamador mostra
// "nao configurado" em vez de inventar um caminho. *Um caminho inventado produz um erro que
// aponta para o sitio errado.*
func CaminhoDoFicheiroDeIdentidades() string {
	dir := DefaultCredentialDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, NomeDoFicheiroDeIdentidades)
}

// ErroDeIdentidadesDesconhecidas e' devolvido quando o ficheiro tem campos que o tipo nao declara.
//
// Separado dos outros erros porque o significado e' diferente: nao e' "o ficheiro esta' mau", e'
// "o ficheiro esta' a tentar dizer-me algo que eu nao devia aceitar".
var ErroDeIdentidadesDesconhecidas = errors.New("ficheiro de identidades com campos desconhecidos")

// CarregarIdentidades le e valida o ficheiro de metadados.
//
// Comportamento por caso, todos deliberados:
//   - caminho vazio  -> registo vazio + erro de "nao configurado" (o chamador decide mostrar aviso)
//   - ficheiro ausente -> registo vazio + erro (ausencia NAO e' registo valido e vazio)
//   - ficheiro grande  -> erro (um registo de metadados nao tem razao para ser grande)
//   - JSON invalido    -> erro
//   - schema diferente -> erro (falha fechado)
//   - identidade invalida -> erro com o nome, e o registo NAO e' parcialmente carregado
//
// O ultimo ponto e' o mais importante: um registo meio-carregado faria a UI mostrar um subconjunto
// e o utilizador supor que e' o todo. *Um registo parcial e' mais perigoso do que um registo
// ausente, porque parece completo.*
func CarregarIdentidades(caminho string) (*RegistoDeIdentidades, error) {
	registo := NovoRegistoDeIdentidades()
	if strings.TrimSpace(caminho) == "" {
		return registo, errors.New("diretorio de credenciais indisponivel: nao ha' onde procurar o registo")
	}
	info, err := os.Stat(caminho)
	if err != nil {
		if os.IsNotExist(err) {
			return registo, fmt.Errorf("registo de identidades nao configurado em %s", caminho)
		}
		return registo, fmt.Errorf("nao foi possivel inspecionar o registo: %w", err)
	}
	if info.IsDir() {
		return registo, errors.New("o caminho do registo e' um diretorio")
	}
	if info.Size() > maxIdentidadesBytes {
		return registo, fmt.Errorf("registo com %d bytes; o limite e' %d", info.Size(), maxIdentidadesBytes)
	}
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		return registo, fmt.Errorf("nao foi possivel ler o registo: %w", err)
	}

	// Decodificacao ESTRITA: um campo desconhecido e' recusado, nao ignorado.
	//
	// Esta e' a defesa contra o cenario em que alguem escreve `"private_key": "..."` no ficheiro
	// e o carregador, por ser tolerante, o descarta em silencio. Descartar em silencio deixaria o
	// segredo no disco e daria a impressao de que o sistema o recusou.
	decodificador := json.NewDecoder(strings.NewReader(string(bruto)))
	decodificador.DisallowUnknownFields()
	var documento documentoDeIdentidades
	if err := decodificador.Decode(&documento); err != nil {
		if strings.Contains(err.Error(), "unknown field") {
			return registo, fmt.Errorf("%w: %v", ErroDeIdentidadesDesconhecidas, err)
		}
		return registo, fmt.Errorf("JSON invalido no registo: %w", err)
	}
	if documento.SchemaVersion != esquemaSuportado {
		return registo, fmt.Errorf("schema_version %d nao suportado (esperado %d)",
			documento.SchemaVersion, esquemaSuportado)
	}
	for _, identidade := range documento.Identidades {
		if err := registo.Adicionar(identidade); err != nil {
			return NovoRegistoDeIdentidades(), fmt.Errorf("registo invalido: %w", err)
		}
	}
	return registo, nil
}

// EstadoDoRegisto e' o que a UI mostra quando o ficheiro nao carrega.
//
// Existe para que "nao configurado", "invalido" e "carregado" nao sejam o mesmo ecra vazio.
// *Um ecra vazio que significa tres coisas diferentes ensina o utilizador a nao olhar.*
type EstadoDoRegisto struct {
	Configurado bool   `json:"configurado"`
	Carregado   bool   `json:"carregado"`
	Caminho     string `json:"caminho,omitempty"`
	Mensagem    string `json:"mensagem"`
	Total       int    `json:"total"`
}

// DescreverEstadoDoRegisto carrega e devolve o estado legivel, nunca o erro cru.
func DescreverEstadoDoRegisto(caminho string) (*RegistoDeIdentidades, EstadoDoRegisto) {
	estado := EstadoDoRegisto{Caminho: caminho}
	registo, err := CarregarIdentidades(caminho)
	if err != nil {
		estado.Mensagem = err.Error()
		// Configurado = o ficheiro existe; carregado = leu sem erro.
		if _, statErr := os.Stat(caminho); statErr == nil && caminho != "" {
			estado.Configurado = true
		}
		return registo, estado
	}
	estado.Configurado = true
	estado.Carregado = true
	estado.Total = len(registo.Identidades())
	if estado.Total == 0 {
		estado.Mensagem = "registo carregado e vazio: nenhuma identidade declarada"
	} else {
		estado.Mensagem = fmt.Sprintf("%d identidade(s) registada(s)", estado.Total)
	}
	return registo, estado
}

// IdentidadesParaODia devolve a projecao usada pela UI: metadados + estado calculado.
//
// O estado e' calculado AQUI, no momento da leitura, e nao guardado no ficheiro: *um estado
// gravado envelhece em silencio, e uma chave que expirou ontem continuaria a dizer "ativa".*
type IdentidadeParaODia struct {
	Nome            string                 `json:"nome"`
	Plataforma      PlataformaDeAssinatura `json:"plataforma"`
	Ambiente        AmbienteDeAssinatura   `json:"ambiente"`
	Provedor        string                 `json:"provedor"`
	FingerprintFim  string                 `json:"fingerprint_publico_fim"`
	ExpiraEm        time.Time              `json:"expira_em"`
	Estado          EstadoDaIdentidade     `json:"estado"`
	UtilizavelAgora bool                   `json:"utilizavel_agora"`
	ExigeAprovacao  bool                   `json:"exige_aprovacao"`
	Runners         int                    `json:"runners_autorizados"`
	PacoteOuBundle  string                 `json:"pacote_ou_bundle,omitempty"`
}

// ProjetarIdentidades converte o registo na lista para a UI.
//
// O fingerprint e' TRUNCADO ao fim (4 bytes). A UI precisa de distinguir duas identidades, nao de
// reconstruir a cadeia: *mostrar o fingerprint inteiro numa lista e' dar material de comparacao a
// quem so' queria saber qual e' qual.*
func ProjetarIdentidades(registo *RegistoDeIdentidades, agora time.Time) []IdentidadeParaODia {
	if registo == nil {
		return []IdentidadeParaODia{}
	}
	saida := make([]IdentidadeParaODia, 0, len(registo.Identidades()))
	for _, i := range registo.Identidades() {
		normalizado, err := NormalizarFingerprint(i.FingerprintPublico)
		truncado := ""
		if err == nil && len(normalizado) >= 8 {
			truncado = strings.ToUpper(normalizado[len(normalizado)-8:])
		}
		saida = append(saida, IdentidadeParaODia{
			Nome:            i.Nome,
			Plataforma:      i.Plataforma,
			Ambiente:        i.Ambiente,
			Provedor:        i.Provedor,
			FingerprintFim:  truncado,
			ExpiraEm:        i.ExpiraEm,
			Estado:          i.Estado(agora),
			UtilizavelAgora: i.Utilizavel(agora) && len(i.RunnersAutorizados) > 0,
			ExigeAprovacao:  i.Ambiente.ExigeAprovacao(),
			Runners:         len(i.RunnersAutorizados),
			PacoteOuBundle:  i.PacoteOuBundle,
		})
	}
	return saida
}
