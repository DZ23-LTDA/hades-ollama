package agent

// FASE 28.1 item 4 do adendo V3: o `SigningProvider`.
//
// O QUE ESTA FATIA E' E O QUE ELA NAO E'
//
// O adendo pede dez providers. O quinto e' `SigningProvider`. Mas a missao **proibe assinar**:
//
//	"a autorizacao vigente continua proibindo release, tag, loja, marketplace e publicacao externa"
//
//	"O DeepSeek NAO esta autorizado nesta missao a usar a chave privada, solicitar senha, exportar o
//	certificado, assinar manifesto ou iniciar interacao criptografica com o A1"
//
// Escrever um `Assinar()` que devolve `ErrNaoAutorizado` seria a coisa mais facil e a mais inutil:
//
//	*Um provider que existe no catalogo e nunca correu e' uma promessa com nome de modulo.*
//
// O que **pode** ser feito — e o que o adendo exige antes de qualquer assinatura — e' decidir se uma
// identidade **serve** para assinar o formato pretendido. Isso e' uma verificacao real, medivel
// agora, e tem um caso concreto neste host:
//
//	O certificado A1 da DZ23 LTDA NAO tem o EKU de Code Signing (1.3.6.1.5.5.7.3.3) e falha na
//	politica Authenticode com `NotValidForUsage`.
//
// Um pipeline que descobrisse isso **depois** de tentar assinar teria gasto tempo e produzido um erro
// opaco do Windows. Este provider descobre-o antes, com o motivo exacto.
//
//	*Um erro que diz "nao sei o que isto e'" nao e' o mesmo que "isto nao existe".*
//
// A FRONTEIRA DITA NO TIPO
//
// `AptidaoDaIdentidade` tem um valor `APTA` que significa "os gates que sei aplicar passaram". Nao
// significa "o artefacto vai ser assinado": a assinatura continua a exigir a chave privada, que este
// processo nunca toca.
//
//	*Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates existiram".*

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// TipoDeAssinatura identifica o formato de assinatura, que determina o EKU exigido.
//
//	*Nao existe um unico assinador universal.* (adendo 28.1)
type TipoDeAssinatura string

const (
	// AssinaturaAuthenticode e' o formato do Windows (`.exe`, `.dll`, `.msi`, `.msix`).
	AssinaturaAuthenticode TipoDeAssinatura = "AUTHENTICODE"
	// AssinaturaAppleDeveloperID e' o formato do macOS fora da Mac App Store.
	AssinaturaAppleDeveloperID TipoDeAssinatura = "APPLE_DEVELOPER_ID"
	// AssinaturaAndroidAPK e' o formato do Android (APK/AAB), via JKS, nao via X.509 direto.
	AssinaturaAndroidAPK TipoDeAssinatura = "ANDROID_APK"
	// AssinaturaOpenPGP e' o formato de metadados Debian/APT e RPM assinados.
	AssinaturaOpenPGP TipoDeAssinatura = "OPENPGP"
)

// OIDs dos EKUs que interessam. Escritos por extenso porque o nome do campo nao os traz.
const (
	oidEKUCodeSigning    = "1.3.6.1.5.5.7.3.3"
	oidEKUAppleDeveloper = "1.2.840.113635.100.6.1.13" // Apple Developer ID Application
)

// AptidaoDaIdentidade e' o veredicto sobre uma identidade de assinatura.
type AptidaoDaIdentidade string

const (
	// IdentidadeApta: os gates aplicaveis passaram. NAO significa que a assinatura foi feita.
	IdentidadeApta AptidaoDaIdentidade = "APTA"
	// IdentidadeSemEKU: o certificado existe mas nao tem o EKU exigido pelo formato.
	IdentidadeSemEKU AptidaoDaIdentidade = "SEM_EKU_DE_ASSINATURA"
	// IdentidadeExpirada: fora do periodo de validade.
	IdentidadeExpirada AptidaoDaIdentidade = "EXPIRADA"
	// IdentidadeAindaNaoValida: `NotBefore` no futuro (relogio errado ou certificado novo demais).
	IdentidadeAindaNaoValida AptidaoDaIdentidade = "AINDA_NAO_VALIDA"
	// IdentidadeDeOutraCA: o emissor nao e' o esperado para o formato.
	IdentidadeDeOutraCA AptidaoDaIdentidade = "EMISSOR_INESPERADO"
	// IdentidadeFormatoDesconhecido: o formato nao tem regra de EKU conhecida por este provider.
	IdentidadeFormatoDesconhecido AptidaoDaIdentidade = "FORMATO_DESCONHECIDO"
	// IdentidadeIlegivel: nao foi possivel ler o certificado.
	IdentidadeIlegivel AptidaoDaIdentidade = "ILEGIVEL"
)

// AvaliacaoDaIdentidade e' o resultado completo, com o motivo escrito.
type AvaliacaoDaIdentidade struct {
	Tipo             TipoDeAssinatura    `json:"tipo"`
	Aptidao          AptidaoDaIdentidade `json:"aptidao"`
	Apta             bool                `json:"apta"`
	Sujeito          string              `json:"sujeito"`
	Emissor          string              `json:"emissor"`
	ValidoDe         time.Time           `json:"valido_de"`
	ValidoAte        time.Time           `json:"valido_ate"`
	EKUs             []string            `json:"ekus"`
	TemEKUNecessario bool                `json:"tem_eku_necessario"`
	Motivo           string              `json:"motivo"`
}

// ErrCertificadoIlegivel e' devolvido quando o ficheiro nao contem um certificado X.509.
var ErrCertificadoIlegivel = errors.New("ficheiro nao contem um certificado X.509 legivel")

// ekuNecessario devolve o OID exigido pelo formato, e se este provider sabe qual e'.
func ekuNecessario(tipo TipoDeAssinatura) (string, bool) {
	switch tipo {
	case AssinaturaAuthenticode:
		return oidEKUCodeSigning, true
	case AssinaturaAppleDeveloperID:
		return oidEKUAppleDeveloper, true
	case AssinaturaAndroidAPK, AssinaturaOpenPGP:
		// Android assina com JKS (nao X.509 direto); OpenPGP tem o seu proprio modelo de chave. Nao
		// ha' EKU a verificar aqui, e inventar um seria *um comentario que declara uma intencao que
		// o codigo nao cumpre*.
		return "", false
	default:
		return "", false
	}
}

// AvaliarCertificado decide se um certificado serve para o tipo de assinatura pedido.
//
// A ordem das verificacoes importa: validade antes de EKU, porque um certificado expirado e' recusado
// por essa razao qualquer que seja o EKU -- e dizer "falta o EKU" a um certificado expirado manda
// quem le' corrigir a coisa errada.
func AvaliarCertificado(
	certificado *x509.Certificate,
	tipo TipoDeAssinatura,
	agora time.Time,
) AvaliacaoDaIdentidade {
	avaliacao := AvaliacaoDaIdentidade{
		Tipo:      tipo,
		ValidoDe:  certificado.NotBefore,
		ValidoAte: certificado.NotAfter,
		EKUs:      ekuDoCertificado(certificado),
	}
	if certificado.Subject.CommonName != "" {
		avaliacao.Sujeito = certificado.Subject.CommonName
	} else {
		avaliacao.Sujeito = certificado.Subject.String()
	}
	avaliacao.Emissor = certificado.Issuer.CommonName

	if agora.Before(certificado.NotBefore) {
		avaliacao.Aptidao = IdentidadeAindaNaoValida
		avaliacao.Motivo = fmt.Sprintf(
			"o certificado so' passa a ser valido em %s; o relogio deste host marca %s",
			certificado.NotBefore.Format(time.RFC3339), agora.Format(time.RFC3339),
		)
		return avaliacao
	}
	if agora.After(certificado.NotAfter) {
		avaliacao.Aptidao = IdentidadeExpirada
		avaliacao.Motivo = fmt.Sprintf(
			"o certificado expirou em %s", certificado.NotAfter.Format(time.RFC3339),
		)
		return avaliacao
	}

	necessario, conhecido := ekuNecessario(tipo)
	if !conhecido {
		avaliacao.Aptidao = IdentidadeFormatoDesconhecido
		avaliacao.Motivo = fmt.Sprintf(
			"este provider nao conhece o EKU exigido por %q; nao afirma que serve nem que nao serve",
			tipo,
		)
		return avaliacao
	}

	// O EMISSOR ANTES DO EKU, e por uma razao MEDIDA neste host: o certificado Apple Distribution do
	// kit da DZ23 LTDA TEM o EKU de Code Signing (`1.3.6.1.5.5.7.3.3`), exactamente como um
	// certificado Authenticode. Sem esta verificacao o provider declararia um certificado da APPLE
	// apto a assinar um `.exe` do WINDOWS.
	//
	//	*Um estado que so' o programador pode nomear invalido nao e' um estado: e' um campo de texto.*
	//
	// `IdentidadeDeOutraCA` estava declarado e nunca era produzido -- codigo morto a fingir de
	// verificacao. Foi um certificado real que o expos.
	if !emissorCompativel(certificado, tipo) {
		avaliacao.Aptidao = IdentidadeDeOutraCA
		avaliacao.Motivo = fmt.Sprintf(
			"o certificado foi emitido por %q, que nao e' uma CA de %q; um certificado de outra "+
				"plataforma pode ter o mesmo EKU de Code Signing e nao assina este formato",
			avaliacao.Emissor, tipo,
		)
		return avaliacao
	}

	avaliacao.TemEKUNecessario = temEKU(certificado, necessario)
	if !avaliacao.TemEKUNecessario {
		avaliacao.Aptidao = IdentidadeSemEKU
		avaliacao.Motivo = fmt.Sprintf(
			"o certificado nao declara o EKU %s exigido por %q; apresenta estes: %s. "+
				"Assinar com ele falharia na politica do formato com um erro opaco -- o A1 da DZ23 "+
				"LTDA esta' neste caso (NotValidForUsage)",
			necessario, tipo, descreverEKUs(avaliacao.EKUs),
		)
		return avaliacao
	}

	avaliacao.Aptidao = IdentidadeApta
	avaliacao.Apta = true
	avaliacao.Motivo = fmt.Sprintf(
		"o certificado tem o EKU %s exigido por %q e esta' dentro da validade; a assinatura em si "+
			"continua a exigir a chave privada, que este processo nao toca",
		necessario, tipo,
	)
	return avaliacao
}

// temEKU diz se o certificado declara o OID pedido, em qualquer EKU.
func temEKU(certificado *x509.Certificate, oid string) bool {
	for _, eku := range certificado.ExtKeyUsage {
		if ekuOID(eku) == oid {
			return true
		}
	}
	for _, eku := range certificado.UnknownExtKeyUsage {
		if eku.String() == oid {
			return true
		}
	}
	return false
}

// ekuOID traduz um `x509.ExtKeyUsage` para o OID correspondente.
//
// Go nao exporta esta tabela e nao ha' `String()` util para ela -- mas os valores sao estaveis e
// documentados no RFC 5280. Os que nao estao aqui aparecem como `UNKNOWN_<n>` em vez de serem
// silenciosamente omitidos: *um relatorio que obriga a somar a mao e' um relatorio para quem ja'
// sabia a resposta*, e um EKU omitido faria um certificado parecer mais vazio do que e'.
func ekuOID(eku x509.ExtKeyUsage) string {
	switch eku {
	case x509.ExtKeyUsageAny:
		return "2.5.29.37.0"
	case x509.ExtKeyUsageServerAuth:
		return "1.3.6.1.5.5.7.3.1"
	case x509.ExtKeyUsageClientAuth:
		return "1.3.6.1.5.5.7.3.2"
	case x509.ExtKeyUsageCodeSigning:
		return oidEKUCodeSigning
	case x509.ExtKeyUsageEmailProtection:
		return "1.3.6.1.5.5.7.3.4"
	case x509.ExtKeyUsageIPSECEndSystem:
		return "1.3.6.1.5.5.7.3.5"
	case x509.ExtKeyUsageIPSECTunnel:
		return "1.3.6.1.5.5.7.3.6"
	case x509.ExtKeyUsageIPSECUser:
		return "1.3.6.1.5.5.7.3.7"
	case x509.ExtKeyUsageTimeStamping:
		return "1.3.6.1.5.5.7.3.8"
	case x509.ExtKeyUsageOCSPSigning:
		return "1.3.6.1.5.5.7.3.9"
	default:
		return fmt.Sprintf("UNKNOWN_%d", int(eku))
	}
}

// ekuDoCertificado lista os OIDs declarados, para o relatorio poder mostrar o que EXISTE.
func ekuDoCertificado(certificado *x509.Certificate) []string {
	oids := make([]string, 0, len(certificado.ExtKeyUsage)+len(certificado.UnknownExtKeyUsage))
	for _, eku := range certificado.ExtKeyUsage {
		oids = append(oids, ekuOID(eku))
	}
	for _, eku := range certificado.UnknownExtKeyUsage {
		oids = append(oids, eku.String())
	}
	return oids
}

// descreverEKUs compoe a lista para o motivo. Uma lista vazia e' dita por extenso.
func descreverEKUs(oids []string) string {
	if len(oids) == 0 {
		return "nenhum"
	}
	return strings.Join(oids, ", ")
}

// CarregarCertificadoDeFicheiro le' um certificado de um ficheiro PEM ou DER.
//
// Aceita os dois porque um `.cer` do Windows costuma ser DER e um `.pem` do OpenSSL e' PEM, e exigir
// que quem chama saiba a diferenca seria *um campo que precisa de documentacao externa para nao ser
// mal interpretado*.
func CarregarCertificadoDeFicheiro(caminho string) (*x509.Certificate, error) {
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("ler %q: %w", caminho, err)
	}
	// PEM primeiro: e' o formato com marcador explicito.
	if bloco, _ := pem.Decode(bruto); bloco != nil && bloco.Type == "CERTIFICATE" {
		certificado, err := x509.ParseCertificate(bloco.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: %q (PEM): %v", ErrCertificadoIlegivel, caminho, err)
		}
		return certificado, nil
	}
	// DER: o corpo inteiro e' o certificado.
	certificado, err := x509.ParseCertificate(bruto)
	if err != nil {
		return nil, fmt.Errorf("%w: %q (nem PEM nem DER): %v", ErrCertificadoIlegivel, caminho, err)
	}
	return certificado, nil
}

// ---------------------------------------------------------------------------
// O CONTRATO DO PROVIDER
// ---------------------------------------------------------------------------

// SigningProvider e' o contrato do item 4 do adendo 28.1.
//
// Tem DOIS metodos e nao tres: `Planear` e `Avaliar`. Nao ha' `Assinar` porque esta missao nao
// autoriza assinar, e um metodo que existisse so' para devolver `ErrNaoAutorizado` seria:
//
//	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
type SigningProvider interface {
	// Alvos devolve os tipos de assinatura que este provider sabe avaliar.
	Alvos() []TipoDeAssinatura
	// Avaliar diz se a identidade indicada serve para o tipo pedido.
	Avaliar(ctx context.Context, tipo TipoDeAssinatura, identidade IdentidadeDeAssinatura) (AvaliacaoDaIdentidade, error)
}

// IdentidadeDeAssinatura referencia uma credencial SEM revelar o valor.
//
//	*O MCP e' uma interface de ferramentas, nao e' a autoridade de assinatura nem um cofre.*
type IdentidadeDeAssinatura struct {
	// CaminhoDoCertificado e' o certificado PUBLICO. Nunca uma chave privada.
	CaminhoDoCertificado string `json:"caminho_do_certificado"`
	// Rotulo e' o nome humano da identidade, para o relatorio.
	Rotulo string `json:"rotulo"`
	// Ambiente permitido: `desenvolvimento`, `preview` ou `producao`.
	Ambiente string `json:"ambiente"`
}

// ProviderDeAssinaturaX509 avalia certificados X.509 para Authenticode e Apple Developer ID.
type ProviderDeAssinaturaX509 struct {
	// Agora permite fixar o relogio nos testes. Zero significa `time.Now()`.
	Agora time.Time
}

// Alvos devolve os formatos que este provider sabe avaliar.
func (ProviderDeAssinaturaX509) Alvos() []TipoDeAssinatura {
	return []TipoDeAssinatura{AssinaturaAuthenticode, AssinaturaAppleDeveloperID}
}

// Avaliar carrega o certificado publico e decide a aptidao.
//
// Recusa um caminho vazio antes de tocar no disco: uma identidade sem certificado nao e' uma
// identidade, e' um rotulo.
func (p ProviderDeAssinaturaX509) Avaliar(
	ctx context.Context,
	tipo TipoDeAssinatura,
	identidade IdentidadeDeAssinatura,
) (AvaliacaoDaIdentidade, error) {
	if err := ctx.Err(); err != nil {
		return AvaliacaoDaIdentidade{}, err
	}
	if strings.TrimSpace(identidade.CaminhoDoCertificado) == "" {
		return AvaliacaoDaIdentidade{
			Tipo:    tipo,
			Aptidao: IdentidadeIlegivel,
			Motivo:  "a identidade nao aponta para nenhum certificado",
		}, ErrSemReferencia
	}
	certificado, err := CarregarCertificadoDeFicheiro(identidade.CaminhoDoCertificado)
	if err != nil {
		if errors.Is(err, ErrCertificadoIlegivel) {
			return AvaliacaoDaIdentidade{
				Tipo:    tipo,
				Aptidao: IdentidadeIlegivel,
				Motivo:  err.Error(),
			}, err
		}
		return AvaliacaoDaIdentidade{}, err
	}
	agora := p.Agora
	if agora.IsZero() {
		agora = time.Now()
	}
	return AvaliarCertificado(certificado, tipo, agora), nil
}

// emissorCompativel diz se o emissor do certificado pertence a familia de CAs que emite para aquele
// formato de assinatura.
//
// LIMITE DECLARADO: isto verifica o NOME da CA emissora, nao a cadeia criptografica. Validar a cadeia
// exigiria as raizes de confianca do sistema e a verificacao de revogacao -- que este provider nao
// faz. E diz isso no motivo quando recusa, em vez de deixar quem le' supor que houve validacao de
// cadeia:
//
//	*Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates existiram".*
func emissorCompativel(certificado *x509.Certificate, tipo TipoDeAssinatura) bool {
	emissor := strings.ToLower(certificado.Issuer.String())
	daApple := strings.Contains(emissor, "apple")
	switch tipo {
	case AssinaturaAppleDeveloperID:
		// Developer ID e' emitido pela Apple Worldwide Developer Relations. Exigir "apple" no
		// emissor recusa uma CA publica qualquer que tenha o mesmo EKU.
		return daApple
	case AssinaturaAuthenticode:
		// Nenhuma CA da Apple emite para Authenticode. Recusar a Apple nao e' suficiente para
		// garantir que a CA e' aceite pela Microsoft, mas evita o erro grosseiro de tratar um
		// certificado Apple como assinatura de Windows.
		return !daApple
	default:
		return false
	}
}
