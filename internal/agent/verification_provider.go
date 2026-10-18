package agent

// FASE 28.1 item 5 do adendo V3: `VerificationProvider` -- o adaptador que VERIFICA o artefacto.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1251 produziu um artefacto e mediu o seu SHA-256. A E-1252 gerou um instalador e mediu o seu.
// Mas o unico verificador era o proprio produtor: o `BuildProvider` calcula o hash e devolve-o.
//
//	*Um artefacto cuja unica verificacao e' o hash calculado por quem o produziu nao e' verificado:
//	e' assinado por si mesmo.*
//
// Este ficheiro e' o item 5 de dez. Verifica o que o item 3 produziu, com uma referencia EXTERNA ao
// produtor.
//
// O QUE ELE VERIFICA, E O QUE ELE RECUSA VERIFICAR
//
//   1. o ficheiro existe, e' um ficheiro (nao um directorio) e tem tamanho > 0;
//   2. o SHA-256 medido AGORA corresponde ao SHA-256 esperado, que o chamador fornece;
//   3. o estado de assinatura vem de um adaptador INJECTADO -- nunca se assume assinado;
//   4. adulteracao: um byte diferente produz `Adulterado = true` e reprova.
//
// Um artefacto SEM checksum esperado nao pode ser "verificado": pode ser medido.
//
//	*Uma verificacao sem referencia e' uma medicao; chamar-lhe verificacao e' chamar prova ao que
//	ainda nao tem com o que se comparar.*
//
// Por isso `Verificar` devolve `ErrSemReferencia` quando o checksum esperado esta' vazio, em vez de
// devolver um resultado verde que ninguem pode contestar.
//
// A ASSINATURA E' UM ADAPTADOR, E O ESTADO POR OMISSAO E' "NAO VERIFICADO"
//
// Este pacote corre em Windows, Linux e macOS. A assinatura Authenticode so' existe no Windows; a
// `codesign` so' no macOS. Em vez de esconder a diferenca atras de um `bool Assinado`, o estado e'
// um tipo explicito com tres valores, e o adaptador e' uma funcao injectada:
//
//	*Um `false` que significa "nao esta' assinado" e "nao consegui verificar" ao mesmo tempo e' pior
//	do que nenhum campo: quem le' decide o contrario do que aconteceu.*
//
// O adaptador de Windows usa `Get-AuthenticodeSignature`. Se a ferramenta nao existir, o estado e'
// `AssinaturaNaoVerificada` -- **nunca** `AssinaturaAusente`, porque "nao consegui olhar" e "olhei e
// nao ha'" sao factos diferentes.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// EstadoDeAssinatura diz o que se sabe sobre a assinatura de um artefacto.
type EstadoDeAssinatura string

const (
	// AssinaturaAusente: olhou-se, e nao ha' assinatura. E' o estado esperado da RC Windows desta
	// missao, que e' intencionalmente `NotSigned`.
	AssinaturaAusente EstadoDeAssinatura = "AUSENTE"
	// AssinaturaPresente: olhou-se, e ha' uma assinatura cujo estado o adaptador conseguiu ler.
	AssinaturaPresente EstadoDeAssinatura = "PRESENTE"
	// AssinaturaInvalida: ha' uma assinatura e ela nao valida.
	AssinaturaInvalida EstadoDeAssinatura = "INVALIDA"
	// AssinaturaNaoVerificada: NAO se conseguiu olhar. Distinto de `AUSENTE`.
	//
	//	*"Nao consegui olhar" e "olhei e nao ha'" sao factos diferentes, e um campo que os junta
	//	mente sobre um dos dois.*
	AssinaturaNaoVerificada EstadoDeAssinatura = "NAO_VERIFICADA"
)

// ResultadoDeVerificacao e' o que o verificador devolve.
type ResultadoDeVerificacao struct {
	// Caminho e' o caminho absoluto verificado.
	Caminho string `json:"caminho"`
	// TamanhoBytes e' o tamanho medido agora.
	TamanhoBytes int64 `json:"tamanho_bytes"`
	// SHA256Medido e' o checksum calculado agora, do ficheiro no disco.
	SHA256Medido string `json:"sha256_medido"`
	// SHA256Esperado e' a referencia contra a qual se comparou.
	SHA256Esperado string `json:"sha256_esperado"`
	// Integro diz se os dois checksums coincidem. E' o unico campo que autoriza "passou".
	Integro bool `json:"integro"`
	// Adulterado e' verdadeiro quando o checksum NAO coincide: o ficheiro mudou desde a referencia.
	Adulterado bool `json:"adulterado"`
	// Assinatura e' o estado declarado, nunca inferido.
	Assinatura EstadoDeAssinatura `json:"assinatura"`
	// DetalheDaAssinatura explica o estado sem expor segredo nenhum.
	DetalheDaAssinatura string `json:"detalhe_da_assinatura,omitempty"`
	// Aprovado e' verdadeiro apenas quando `Integro` e a assinatura NAO e' invalida.
	Aprovado bool `json:"aprovado"`
	// DuracaoMS e' o tempo real gasto.
	DuracaoMS int64 `json:"duracao_ms"`
	// Explicacao diz numa frase o que aconteceu.
	Explicacao string `json:"explicacao"`
}

// ErrSemReferencia e' devolvido quando se tenta verificar sem checksum esperado.
var ErrSemReferencia = errors.New("verificacao sem checksum de referencia: isto seria uma medicao")

// ErrFicheiroInexistente e' devolvido quando o caminho nao existe.
var ErrFicheiroInexistente = errors.New("artefacto nao encontrado")

// VerificadorDeAssinatura le o estado de assinatura de um ficheiro.
//
// E' injectado para que o verificador nao dependa do sistema onde corre, e para que o teste possa
// provar os TRES estados sem precisar de um binario assinado.
type VerificadorDeAssinatura func(ctx context.Context, caminho string) (EstadoDeAssinatura, string, error)

// VerificationProvider verifica um artefacto contra uma referencia externa.
type VerificationProvider interface {
	Verificar(ctx context.Context, caminho, sha256Esperado string) (ResultadoDeVerificacao, error)
}

// VerificadorDeArtefacto e' a implementacao canonica.
type VerificadorDeArtefacto struct {
	// Assinatura e' o adaptador de assinatura. Nil significa "nao sei verificar assinatura", e o
	// resultado declara `AssinaturaNaoVerificada` -- nunca `AssinaturaAusente`.
	Assinatura VerificadorDeAssinatura
}

// Verificar mede o ficheiro, compara com o checksum esperado e le' o estado de assinatura.
func (v VerificadorDeArtefacto) Verificar(
	ctx context.Context,
	caminho, sha256Esperado string,
) (ResultadoDeVerificacao, error) {
	inicio := time.Now()
	esperado := strings.ToLower(strings.TrimSpace(sha256Esperado))
	if esperado == "" {
		return ResultadoDeVerificacao{}, fmt.Errorf("%w: %s", ErrSemReferencia, caminho)
	}
	// O checksum esperado tem de PARECER um SHA-256. Um valor de 8 caracteres nao e' uma referencia:
	// e' um prefixo, e comparar contra ele "passaria" com um ficheiro diferente.
	//
	//	*Uma referencia curta nao e' uma referencia: e' uma esperanca com hexadecimal.*
	if len(esperado) != 64 || !apenasHexadecimal(esperado) {
		return ResultadoDeVerificacao{}, fmt.Errorf(
			"%w: o checksum esperado tem %d caracteres (um SHA-256 tem 64) e %q nao serve de referencia",
			ErrSemReferencia, len(esperado), sha256Esperado)
	}

	absoluto, err := filepath.Abs(caminho)
	if err != nil {
		return ResultadoDeVerificacao{}, fmt.Errorf("resolver %q: %w", caminho, err)
	}
	info, err := os.Stat(absoluto)
	if err != nil {
		if os.IsNotExist(err) {
			return ResultadoDeVerificacao{}, fmt.Errorf("%w: %s", ErrFicheiroInexistente, absoluto)
		}
		return ResultadoDeVerificacao{}, fmt.Errorf("stat %q: %w", absoluto, err)
	}
	if info.IsDir() {
		return ResultadoDeVerificacao{}, fmt.Errorf("%q e' um directorio; um artefacto e' um ficheiro", absoluto)
	}
	// *Um ficheiro de zero bytes tem um SHA-256 valido e nao serve para nada.* O tamanho faz parte da
	// verificacao, e um vazio reprova antes de se falar de checksum.
	if info.Size() == 0 {
		return ResultadoDeVerificacao{}, fmt.Errorf("%w: %q tem 0 bytes", ErrFicheiroInexistente, absoluto)
	}

	medido, err := sha256DeFicheiro(absoluto)
	if err != nil {
		return ResultadoDeVerificacao{}, fmt.Errorf("checksum de %q: %w", absoluto, err)
	}

	resultado := ResultadoDeVerificacao{
		Caminho:        absoluto,
		TamanhoBytes:   info.Size(),
		SHA256Medido:   medido,
		SHA256Esperado: esperado,
		Integro:        medido == esperado,
		Adulterado:     medido != esperado,
	}

	// A assinatura vem do adaptador. Sem adaptador, o estado e' NAO_VERIFICADA e a explicacao di-lo.
	if v.Assinatura == nil {
		resultado.Assinatura = AssinaturaNaoVerificada
		resultado.DetalheDaAssinatura = "nenhum adaptador de assinatura configurado"
	} else {
		estado, detalhe, errAssinatura := v.Assinatura(ctx, absoluto)
		if errAssinatura != nil {
			// *Uma falha do adaptador nao se transforma em `AUSENTE`: isso diria ao auditor que se
			// olhou e nao havia, quando o que houve foi um erro a olhar.*
			resultado.Assinatura = AssinaturaNaoVerificada
			resultado.DetalheDaAssinatura = "adaptador falhou: " + errAssinatura.Error()
		} else {
			resultado.Assinatura = estado
			resultado.DetalheDaAssinatura = detalhe
		}
	}

	// *Aprovado exige integridade. A assinatura INVALIDA reprova; a AUSENTE e a NAO_VERIFICADA nao.*
	//
	//	*Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates existiram".*
	//
	// A RC Windows desta missao e' intencionalmente `NotSigned`: reprovar por `AUSENTE` tornaria o
	// gate inutil. E reprovar por `NAO_VERIFICADA` puniria o auditor pelo instrumento que falta no
	// host -- Authenticode so' existe no Windows. **Quem exige assinatura le' o campo `Assinatura` e
	// decide**; o que o verificador nao faz e' esconder a diferenca entre "olhei e nao ha'" e "nao
	// consegui olhar".
	//
	// (A primeira versao do teste esperava `Aprovado=false` para o adaptador que falha. Estava
	// errado: seria reprovar tudo fora do Windows.)
	resultado.Aprovado = resultado.Integro && resultado.Assinatura != AssinaturaInvalida

	switch {
	case resultado.Adulterado:
		resultado.Explicacao = "ADULTERADO: o checksum medido nao corresponde ao esperado"
	case resultado.Assinatura == AssinaturaInvalida:
		resultado.Explicacao = "integro, mas a assinatura presente NAO valida"
	case !resultado.Aprovado:
		resultado.Explicacao = "nao aprovado"
	case resultado.Assinatura == AssinaturaNaoVerificada:
		resultado.Explicacao = "integro; assinatura NAO verificada neste host"
	default:
		resultado.Explicacao = fmt.Sprintf("integro; assinatura %s", resultado.Assinatura)
	}
	resultado.DuracaoMS = time.Since(inicio).Milliseconds()
	return resultado, nil
}

// AdaptadorAuthenticode devolve um `VerificadorDeAssinatura` que usa `Get-AuthenticodeSignature` no
// Windows. Fora do Windows devolve um adaptador que declara sempre `AssinaturaNaoVerificada`.
//
//	*A ferramenta de verificacao de uma plataforma nao existe noutra; fingir que existe produziria
//	um veredito sobre um sistema que nao foi olhado.*
func AdaptadorAuthenticode() VerificadorDeAssinatura {
	return func(ctx context.Context, caminho string) (EstadoDeAssinatura, string, error) {
		if !strings.EqualFold(filepath.Ext(caminho), ".exe") &&
			!strings.EqualFold(filepath.Ext(caminho), ".msi") &&
			!strings.EqualFold(filepath.Ext(caminho), ".dll") {
			return AssinaturaNaoVerificada, "Authenticode so' se aplica a .exe/.dll/.msi", nil
		}
		// `Get-AuthenticodeSignature` e' o verificador OFICIAL do Windows para Authenticode. Ler o PE
		// a mao seria reimplementar a validacao da cadeia -- e uma reimplementacao parcial da' um
		// veredito parcial que ninguem distingue de um completo.
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
			`(Get-AuthenticodeSignature -LiteralPath $env:HADES_ARTEFACTO).Status.ToString()`)
		cmd.Env = append(os.Environ(), "HADES_ARTEFACTO="+caminho)
		saida, err := cmd.CombinedOutput()
		if err != nil {
			return AssinaturaNaoVerificada, "", fmt.Errorf("Get-AuthenticodeSignature: %w (%s)", err,
				truncarSaida(string(saida)))
		}
		estado := strings.TrimSpace(string(saida))
		switch estado {
		case "NotSigned":
			return AssinaturaAusente, "NotSigned (o esperado numa RC nao assinada)", nil
		case "Valid":
			return AssinaturaPresente, "Valid", nil
		case "HashMismatch", "NotTrusted", "UnknownError", "Incompatible":
			return AssinaturaInvalida, estado, nil
		default:
			// Qualquer estado que nao se reconheca fica NAO_VERIFICADO: *um estado novo nao e' um
			// estado bom.*
			return AssinaturaNaoVerificada, "estado desconhecido: " + estado, nil
		}
	}
}

// apenasHexadecimal diz se a string so' tem digitos hexadecimais.
func apenasHexadecimal(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
