package multillm

// FASE 28.4 do adendo V3: matriz de assinatura por plataforma e formato.
//
// O QUE ESTE FICHEIRO RESOLVE
//
// O `signing_identity.go` sabe QUE identidades existem. O `toolchain_plan.go` sabe QUE alvos de
// build ha'. Falta a peca que liga as duas: dado um formato de artefacto, QUAL e' o assinador,
// QUAL e' o verificador, e o que continua NAO suportado.
//
// A CONCLUSAO ARQUITETURAL QUE O ADENDO EXIGE, ESCRITA EM CODIGO
//
// **Nao existe um assinador universal.** Cada formato tem o seu caminho nativo, e um formato que
// nao tem caminho tem de o DIZER -- em vez de ser omitido da tabela. *Um formato que desaparece da
// matriz parece inexistente; um formato que declara "nao suportado" impede alguem de o prometer.*
//
// O erro que este ficheiro existe para impedir esta' escrito nos testes:
//   - EAS **nao** assina Windows, Linux nem macOS desktop;
//   - um certificado Apple Distribution **nao** substitui Developer ID Application para
//     distribuicao macOS notarizada;
//   - a Microsoft Store pode assinar um MSIX, mas isso **nao** assina um `.exe` distribuido fora
//     dela;
//   - "Web assinado" nao e' assinatura de binario: e' provenance e checksum, e tem de o dizer;
//   - "Linux assinado" nao existe sem dizer o artefacto, a chave, o verificador e o canal.
//
// Cada formato declara: ferramenta de assinatura, ferramenta de verificacao, e se a assinatura e'
// de BINARIO ou de METADADOS. Sem os tres, nao entra na tabela como suportado.

import (
	"fmt"
	"sort"
	"strings"
)

// NaturezaDaAssinatura distingue o que esta' a ser assinado.
//
// A distincao e' o ponto: *"Web assinado" e "binario assinado" sao promessas diferentes, e
// junta-las numa so' palavra faz uma delas mentir.*
type NaturezaDaAssinatura string

const (
	// NaturezaBinario assina o proprio ficheiro executavel (Authenticode, codesign, apksigner).
	NaturezaBinario NaturezaDaAssinatura = "binario"
	// NaturezaMetadados assina metadados/indices, nao o payload (APT, RPM, provenance OCI).
	NaturezaMetadados NaturezaDaAssinatura = "metadados"
	// NaturezaProvenance assina uma atestacao sobre o artefacto (Sigstore, SLSA, checksums).
	NaturezaProvenance NaturezaDaAssinatura = "provenance"
	// NaturezaNaoAssinavel e' um formato que nao tem assinatura no modelo adoptado.
	NaturezaNaoAssinavel NaturezaDaAssinatura = "nao_assinavel"
)

// FormatoDeArtefacto e' o que sai do build.
type FormatoDeArtefacto string

const (
	FormatoAPK      FormatoDeArtefacto = "apk"
	FormatoAAB      FormatoDeArtefacto = "aab"
	FormatoIPA      FormatoDeArtefacto = "ipa"
	FormatoApp      FormatoDeArtefacto = "app"
	FormatoPKG      FormatoDeArtefacto = "pkg"
	FormatoDMG      FormatoDeArtefacto = "dmg"
	FormatoEXE      FormatoDeArtefacto = "exe"
	FormatoMSI      FormatoDeArtefacto = "msi"
	FormatoMSIX     FormatoDeArtefacto = "msix"
	FormatoDEB      FormatoDeArtefacto = "deb"
	FormatoRPM      FormatoDeArtefacto = "rpm"
	FormatoAppImage FormatoDeArtefacto = "appimage"
	FormatoFlatpak  FormatoDeArtefacto = "flatpak"
	FormatoSnap     FormatoDeArtefacto = "snap"
	FormatoWeb      FormatoDeArtefacto = "web"
	FormatoOCI      FormatoDeArtefacto = "oci"
)

// CaminhoDeAssinatura descreve COMO um formato se assina e COMO se verifica.
//
// Todos os campos sao obrigatorios para um formato suportado. Um campo vazio torna o formato
// `nao_suportado` por construcao: *nao ha' como verificar o que nao se sabe como verificar.*
type CaminhoDeAssinatura struct {
	Formato    FormatoDeArtefacto     `json:"formato"`
	Plataforma PlataformaDeAssinatura `json:"plataforma"`
	Natureza   NaturezaDaAssinatura   `json:"natureza"`
	// FerramentaDeAssinatura e' o CLI/SDK que assina (apksigner, signtool, codesign, rpm --addsign).
	FerramentaDeAssinatura string `json:"ferramenta_de_assinatura"`
	// FerramentaDeVerificacao e' o CLI/SDK INDEPENDENTE que confirma. Distinto do assinador.
	FerramentaDeVerificacao string `json:"ferramenta_de_verificacao"`
	// RequerRunnerMacOS marca os formatos que so' se assinam em macOS. *Nao e' uma limitacao de
	// configuracao: e' uma limitacao de plataforma, e o plano tem de a ver antes de prometer.*
	RequerRunnerMacOS bool `json:"requer_runner_macos"`
	// RequerIdentidadeDeProducao marca o que nao se prova com certificado descartavel.
	RequerIdentidadeDeProducao bool `json:"requer_identidade_de_producao"`
	// Nota explica o limite em linguagem que o utilizador le'.
	Nota string `json:"nota,omitempty"`
}

// suportado diz se o caminho tem o minimo para ser prometido.
//
// Sem assinador, sem verificador, ou sem natureza declarada, o formato nao entra como suportado.
// *Um formato que se diz suportado sem verificador e' uma promessa sem prova.*
func (c CaminhoDeAssinatura) suportado() bool {
	if strings.TrimSpace(c.FerramentaDeAssinatura) == "" {
		return false
	}
	if strings.TrimSpace(c.FerramentaDeVerificacao) == "" {
		return false
	}
	switch c.Natureza {
	case NaturezaBinario, NaturezaMetadados, NaturezaProvenance:
		return true
	case NaturezaNaoAssinavel:
		// Nao-assinavel e' um estado declarado, nao um caminho de assinatura.
		return false
	}
	return false
}

// MatrizDeAssinatura e' o catalogo canonico dos caminhos conhecidos.
//
// Declarativo e ordenado. *Um catalogo que se constroi por descoberta em runtime muda de forma
// quando a maquina muda, e deixa de ser comparavel entre dois hosts.*
func MatrizDeAssinatura() []CaminhoDeAssinatura {
	return []CaminhoDeAssinatura{
		{
			Formato: FormatoAPK, Plataforma: PlataformaAndroid, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "apksigner", FerramentaDeVerificacao: "apksigner verify",
			Nota: "APK instalavel em preview; a upload key do Play nao e a app-signing key.",
		},
		{
			Formato: FormatoAAB, Plataforma: PlataformaAndroid, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "jarsigner", FerramentaDeVerificacao: "bundletool validate",
			RequerIdentidadeDeProducao: true,
			Nota:                       "AAB e' assinado com a upload key; a Play App Signing re-assina na distribuicao.",
		},
		{
			Formato: FormatoIPA, Plataforma: PlataformaIOS, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "codesign", FerramentaDeVerificacao: "codesign --verify",
			RequerRunnerMacOS: true, RequerIdentidadeDeProducao: true,
			Nota: "Exige Apple Distribution + provisioning profile; simulator build nao prova isto.",
		},
		{
			Formato: FormatoApp, Plataforma: PlataformaMacOS, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "codesign", FerramentaDeVerificacao: "codesign --verify --deep",
			RequerRunnerMacOS: true, RequerIdentidadeDeProducao: true,
			Nota: "Developer ID Application; Apple Distribution NAO substitui para notarizacao.",
		},
		{
			Formato: FormatoPKG, Plataforma: PlataformaMacOS, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "productsign", FerramentaDeVerificacao: "pkgutil --check-signature",
			RequerRunnerMacOS: true, RequerIdentidadeDeProducao: true,
			Nota: "Developer ID Installer, que e' um certificado diferente do Application.",
		},
		{
			Formato: FormatoDMG, Plataforma: PlataformaMacOS, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "codesign", FerramentaDeVerificacao: "codesign --verify",
			RequerRunnerMacOS: true, RequerIdentidadeDeProducao: true,
			Nota: "Assinatura do DMG + notarizacao e stapling para Gatekeeper.",
		},
		{
			Formato: FormatoEXE, Plataforma: PlataformaWindows, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "signtool sign", FerramentaDeVerificacao: "signtool verify",
			Nota: "Authenticode. A Microsoft Store NAO assina um .exe distribuido fora dela.",
		},
		{
			Formato: FormatoMSI, Plataforma: PlataformaWindows, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "signtool sign", FerramentaDeVerificacao: "signtool verify",
			Nota: "Authenticode sobre o MSI; o mesmo limite do .exe.",
		},
		{
			Formato: FormatoMSIX, Plataforma: PlataformaWindows, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "signtool sign", FerramentaDeVerificacao: "signtool verify",
			Nota: "A Store pode assinar o MSIX que distribui; isso nao cobre o .exe fora da Store.",
		},
		{
			Formato: FormatoDEB, Plataforma: PlataformaLinux, Natureza: NaturezaMetadados,
			FerramentaDeAssinatura: "gpg --detach-sign", FerramentaDeVerificacao: "gpg --verify",
			Nota: "Assina o Release/InRelease do repositorio APT, nao o .deb individual.",
		},
		{
			Formato: FormatoRPM, Plataforma: PlataformaLinux, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "rpmsign --addsign", FerramentaDeVerificacao: "rpmkeys --checksig",
			Nota: "Assinatura do pacote e, quando ha' repositorio, do repomd.",
		},
		{
			Formato: FormatoAppImage, Plataforma: PlataformaLinux, Natureza: NaturezaBinario,
			FerramentaDeAssinatura: "appimagetool --sign", FerramentaDeVerificacao: "appimagetool --validate",
			Nota: "Assinatura AppImage quando o pipeline adoptado a suportar.",
		},
		{
			Formato: FormatoFlatpak, Plataforma: PlataformaLinux, Natureza: NaturezaProvenance,
			FerramentaDeAssinatura: "flatpak build-sign", FerramentaDeVerificacao: "flatpak build-verify",
			RequerIdentidadeDeProducao: true,
			Nota:                       "Flathub tem o seu proprio modelo de identidade e publicacao.",
		},
		{
			Formato: FormatoSnap, Plataforma: PlataformaLinux, Natureza: NaturezaProvenance,
			FerramentaDeAssinatura: "snap sign", FerramentaDeVerificacao: "snap ack",
			RequerIdentidadeDeProducao: true,
			Nota:                       "Snap Store gere a identidade; nao ha' assinatura local equivalente.",
		},
		{
			Formato: FormatoWeb, Plataforma: PlataformaWeb, Natureza: NaturezaProvenance,
			FerramentaDeAssinatura: "sha256sum + manifest", FerramentaDeVerificacao: "sha256sum --check",
			Nota: "NAO e' assinatura de binario: e' checksum + provenance + headers. " +
				"Dizer \"web assinado\" sem isto e' outra promessa.",
		},
		{
			Formato: FormatoOCI, Plataforma: PlataformaOCI, Natureza: NaturezaProvenance,
			FerramentaDeAssinatura: "cosign sign", FerramentaDeVerificacao: "cosign verify",
			Nota: "Sigstore com OIDC keyless ou chave em KMS/vault; SBOM e attestation a parte.",
		},
	}
}

// EstadosDeAceite sao os veredictos permitidos pela FASE 28.11 do adendo.
//
// A lista e' fechada. *Um estado novo tem de ser uma decisao, nao um erro de ortografia numa
// string.*
type EstadoDeAceite string

const (
	AceiteCompletoVerificado           EstadoDeAceite = "COMPLETO_VERIFICADO"
	AceiteBuildVerificadoNaoAssinado   EstadoDeAceite = "BUILD_VERIFICADO_NAO_ASSINADO"
	AceiteAssinadoComIdentidadeDeTeste EstadoDeAceite = "ASSINADO_COM_IDENTIDADE_DE_TESTE"
	AceiteProntoParaAssinatura         EstadoDeAceite = "PRONTO_PARA_ASSINATURA"
	AceiteConfigRequerido              EstadoDeAceite = "CONFIG_REQUIRED"
	AceitePlanoRequerido               EstadoDeAceite = "PLAN_REQUIRED"
	AceiteBloqueadoExterno             EstadoDeAceite = "BLOCKED_EXTERNAL"
	AceiteExperimental                 EstadoDeAceite = "EXPERIMENTAL"
	AceiteNaoSuportado                 EstadoDeAceite = "UNSUPPORTED"
	AceiteFalhado                      EstadoDeAceite = "FAILED"
)

// Valido diz se o estado pertence a lista fechada.
func (e EstadoDeAceite) Valido() bool {
	switch e {
	case AceiteCompletoVerificado, AceiteBuildVerificadoNaoAssinado,
		AceiteAssinadoComIdentidadeDeTeste, AceiteProntoParaAssinatura,
		AceiteConfigRequerido, AceitePlanoRequerido, AceiteBloqueadoExterno,
		AceiteExperimental, AceiteNaoSuportado, AceiteFalhado:
		return true
	}
	return false
}

// EvidenciaDeAssinatura e' o que se SABE sobre um artefacto concreto.
//
// Zero values significam "nao provado", nunca "provado". *Um booleano que comeca em false e' a
// unica forma segura de representar uma prova que ainda nao existe.*
type EvidenciaDeAssinatura struct {
	Formato              FormatoDeArtefacto `json:"formato"`
	BuildExecutado       bool               `json:"build_executado"`
	AssinaturaAplicada   bool               `json:"assinatura_aplicada"`
	AssinaturaVerificada bool               `json:"assinatura_verificada"`
	InstalacaoTestada    bool               `json:"instalacao_testada"`
	IdentidadeDeTeste    bool               `json:"identidade_de_teste"`
	RunnerDisponivel     bool               `json:"runner_disponivel"`
	MotivoDoBloqueio     string             `json:"motivo_do_bloqueio,omitempty"`
}

// AvaliarAssinatura decide o estado de aceite de um formato.
//
// A ordem das verificacoes e' a ordem das dependencias: sem caminho nao ha' build; sem build nao
// ha' assinatura; sem verificacao nao ha' prova. *Colapsar passos faz um estado parecer melhor do
// que e'.*
//
// Regra dura do adendo: **`COMPLETO_VERIFICADO` exige instalacao testada.** Um artefacto assinado
// e verificado que nunca foi instalado e' uma promessa sobre um ficheiro, nao sobre um produto.
func AvaliarAssinatura(formato FormatoDeArtefacto, ev EvidenciaDeAssinatura) EstadoDeAceite {
	caminho, existe := caminhoPara(formato)
	if !existe || !caminho.suportado() {
		return AceiteNaoSuportado
	}
	if !ev.RunnerDisponivel {
		// O runner e' requisito de PLATAFORMA, nao de configuracao do utilizador.
		return AceiteBloqueadoExterno
	}
	if !ev.BuildExecutado {
		return AceiteProntoParaAssinatura
	}
	if !ev.AssinaturaAplicada {
		return AceiteBuildVerificadoNaoAssinado
	}
	if !ev.AssinaturaVerificada {
		// Assinado sem verificacao independente nao e' prova de assinatura.
		return AceiteFalhado
	}
	if !ev.InstalacaoTestada {
		if ev.IdentidadeDeTeste {
			return AceiteAssinadoComIdentidadeDeTeste
		}
		return AceiteProntoParaAssinatura
	}
	if ev.IdentidadeDeTeste {
		// Instalado com identidade descartavel: prova o pipeline, nao a confianca publica.
		return AceiteAssinadoComIdentidadeDeTeste
	}
	return AceiteCompletoVerificado
}

// caminhoPara procura o caminho de um formato.
func caminhoPara(formato FormatoDeArtefacto) (CaminhoDeAssinatura, bool) {
	for _, c := range MatrizDeAssinatura() {
		if c.Formato == formato {
			return c, true
		}
	}
	return CaminhoDeAssinatura{}, false
}

// FormatosPorPlataforma devolve os formatos conhecidos de uma plataforma, ordenados.
//
// Inclui os NAO suportados de proposito: *omitir o que nao se suporta faz o utilizador supor que
// nao existe; declara-lo permite-lhe decidir.*
func FormatosPorPlataforma(plataforma PlataformaDeAssinatura) []FormatoDeArtefacto {
	var saida []FormatoDeArtefacto
	for _, c := range MatrizDeAssinatura() {
		if c.Plataforma == plataforma {
			saida = append(saida, c.Formato)
		}
	}
	sort.Slice(saida, func(a, b int) bool { return saida[a] < saida[b] })
	return saida
}

// ProblemaDaMatriz e' um achado sobre a propria matriz (auto-auditoria).
type ProblemaDaMatriz struct {
	Formato  string `json:"formato,omitempty"`
	Mensagem string `json:"mensagem"`
}

// AuditarMatriz verifica a coerencia interna do catalogo.
//
// Serve para o catalogo nao apodrecer: *uma matriz que ninguem audita ganha linhas que parecem
// suportadas e nao sao.*
func AuditarMatriz() []ProblemaDaMatriz {
	var problemas []ProblemaDaMatriz
	vistos := map[FormatoDeArtefacto]bool{}
	for _, c := range MatrizDeAssinatura() {
		if vistos[c.Formato] {
			problemas = append(problemas, ProblemaDaMatriz{
				Formato: string(c.Formato), Mensagem: "formato duplicado na matriz",
			})
		}
		vistos[c.Formato] = true
		if !c.Plataforma.Valida() {
			problemas = append(problemas, ProblemaDaMatriz{
				Formato: string(c.Formato), Mensagem: "plataforma invalida: " + string(c.Plataforma),
			})
		}
		// Formatos macOS sem a marca de runner macOS seriam prometidos em Windows.
		if (c.Formato == FormatoIPA || c.Formato == FormatoApp || c.Formato == FormatoPKG ||
			c.Formato == FormatoDMG) && !c.RequerRunnerMacOS {
			problemas = append(problemas, ProblemaDaMatriz{
				Formato:  string(c.Formato),
				Mensagem: "formato Apple sem `RequerRunnerMacOS`: seria prometido fora de macOS",
			})
		}
		if !c.suportado() && c.Natureza != NaturezaNaoAssinavel {
			problemas = append(problemas, ProblemaDaMatriz{
				Formato: string(c.Formato),
				Mensagem: "declarado na matriz mas sem assinador/verificador: " +
					"devia ser `nao_assinavel` ou sair da matriz",
			})
		}
	}
	return problemas
}

// FormatoDaPlataforma valida que a plataforma serve o formato.
//
// E' o erro que o adendo nomeia: *alegar que o EAS assina Windows.*
func FormatoDaPlataforma(formato FormatoDeArtefacto, plataforma PlataformaDeAssinatura) error {
	c, existe := caminhoPara(formato)
	if !existe {
		return fmt.Errorf("formato %q nao esta' na matriz", formato)
	}
	if c.Plataforma != plataforma {
		return fmt.Errorf("formato %q pertence a plataforma %q, nao a %q",
			formato, c.Plataforma, plataforma)
	}
	return nil
}
