package multillm

// FASE 28.4: testes da matriz de assinatura por plataforma.
//
// Os testes mais importantes deste ficheiro sao os NEGATIVOS: os que provam que a matriz recusa
// as alegacoes que o adendo nomeia --
//
//   - EAS nao assina Windows, Linux nem macOS desktop;
//   - Apple Distribution nao substitui Developer ID Application para macOS notarizado;
//   - a Microsoft Store nao assina um .exe distribuido fora dela;
//   - "Web assinado" nao e' assinatura de binario;
//   - `COMPLETO_VERIFICADO` nunca sem instalacao testada.
//
// *Uma matriz que so' sabe dizer "sim" e' uma lista de desejos.*

import (
	"strings"
	"testing"
)

// INVARIANTE CENTRAL: todo o formato na matriz tem assinador E verificador.
//
// Sem os dois, o formato devia ser `nao_assinavel` ou sair. Este teste impede o meio-termo.
func TestTodoFormatoTemAssinadorEVerificador(t *testing.T) {
	for _, c := range MatrizDeAssinatura() {
		if strings.TrimSpace(c.FerramentaDeAssinatura) == "" {
			t.Fatalf("formato %q sem ferramenta de assinatura", c.Formato)
		}
		if strings.TrimSpace(c.FerramentaDeVerificacao) == "" {
			t.Fatalf("formato %q sem ferramenta de verificacao: *uma promessa sem prova*", c.Formato)
		}
		if !c.suportado() {
			t.Fatalf("formato %q declarado mas nao suportado", c.Formato)
		}
	}
}

// O assinador e o verificador tem de ser DIFERENTES.
//
// *Quem assina nao pode ser a unica testemunha de que assinou.*
func TestVerificadorEDistintoDoAssinador(t *testing.T) {
	for _, c := range MatrizDeAssinatura() {
		assinador := strings.ToLower(strings.TrimSpace(c.FerramentaDeAssinatura))
		verificador := strings.ToLower(strings.TrimSpace(c.FerramentaDeVerificacao))
		if assinador == verificador {
			t.Fatalf("formato %q usa a mesma ferramenta para assinar e verificar (%q)",
				c.Formato, c.FerramentaDeAssinatura)
		}
		// A verificacao tem de ser uma invocacao propria, nao o mesmo comando.
		if !strings.HasPrefix(verificador, strings.Fields(assinador)[0]) &&
			!strings.Contains(verificador, "verify") &&
			!strings.Contains(verificador, "check") &&
			!strings.Contains(verificador, "validate") &&
			!strings.Contains(verificador, "check-signature") {
			t.Fatalf("formato %q: verificacao %q nao parece uma verificacao",
				c.Formato, c.FerramentaDeVerificacao)
		}
	}
}

// Os formatos Apple exigem runner macOS, e o catalogo tem de o declarar.
func TestFormatosAppleExigemRunnerMacOS(t *testing.T) {
	for _, formato := range []FormatoDeArtefacto{FormatoIPA, FormatoApp, FormatoPKG, FormatoDMG} {
		c, existe := caminhoPara(formato)
		if !existe {
			t.Fatalf("formato Apple %q ausente do catalogo", formato)
		}
		if !c.RequerRunnerMacOS {
			t.Fatalf("formato %q nao declara `RequerRunnerMacOS`: seria prometido em Windows", formato)
		}
	}
	// E o contrario: um formato Windows/Linux NAO pode exigir macOS.
	for _, formato := range []FormatoDeArtefacto{FormatoEXE, FormatoMSI, FormatoMSIX, FormatoDEB, FormatoRPM} {
		c, _ := caminhoPara(formato)
		if c.RequerRunnerMacOS {
			t.Fatalf("formato %q exige macOS sem razao", formato)
		}
	}
}

// O erro que o adendo nomeia: alegar que o EAS assina Windows.
func TestFormatoDaPlataformaRecusaCruzamento(t *testing.T) {
	// O caso certo passa.
	if err := FormatoDaPlataforma(FormatoEXE, PlataformaWindows); err != nil {
		t.Fatalf("EXE em Windows recusado: %v", err)
	}
	// O caso do adendo: EXE "em iOS" (o que equivaleria a dizer que o EAS assina Windows).
	if err := FormatoDaPlataforma(FormatoEXE, PlataformaIOS); err == nil {
		t.Fatal("EXE aceite como formato de iOS: *alegar que o EAS assina Windows*")
	}
	if err := FormatoDaPlataforma(FormatoIPA, PlataformaWindows); err == nil {
		t.Fatal("IPA aceite como formato de Windows")
	}
	if err := FormatoDaPlataforma(FormatoDEB, PlataformaAndroid); err == nil {
		t.Fatal("DEB aceite como formato de Android")
	}
	if err := FormatoDaPlataforma("formato-inventado", PlataformaLinux); err == nil {
		t.Fatal("formato inventado aceite")
	}
}

// "Web assinado" nao e' assinatura de binario.
func TestWebEProvenanceNaoBinario(t *testing.T) {
	c, existe := caminhoPara(FormatoWeb)
	if !existe {
		t.Fatal("formato web ausente")
	}
	if c.Natureza == NaturezaBinario {
		t.Fatal("web declarado como assinatura de binario: *e' checksum e provenance, e tem de o dizer*")
	}
	if c.Natureza != NaturezaProvenance {
		t.Fatalf("natureza do web = %q; esperado provenance", c.Natureza)
	}
	// E a nota tem de explicar o limite, nao apenas existir.
	if !strings.Contains(c.Nota, "NAO e' assinatura de binario") {
		t.Fatalf("nota do web nao explica o limite: %q", c.Nota)
	}
}

// O mesmo para o .deb: assina-se o repositorio, nao o pacote individual.
func TestDEBAssinaMetadadosNaoOBinario(t *testing.T) {
	c, _ := caminhoPara(FormatoDEB)
	if c.Natureza != NaturezaMetadados {
		t.Fatalf("natureza do deb = %q; esperado metadados (Release/InRelease)", c.Natureza)
	}
	if !strings.Contains(c.Nota, "nao o .deb individual") {
		t.Fatalf("nota do deb nao distingue repositorio de pacote: %q", c.Nota)
	}
}

// A Microsoft Store pode assinar um MSIX, mas nao um .exe fora dela.
func TestNotaDoEXEDistingueDaStore(t *testing.T) {
	c, _ := caminhoPara(FormatoEXE)
	if !strings.Contains(c.Nota, "Microsoft Store") || !strings.Contains(c.Nota, "fora dela") {
		t.Fatalf("nota do EXE nao distingue a Store: %q", c.Nota)
	}
}

// Apple Distribution nao substitui Developer ID Application para macOS.
func TestNotaDoAppDistingueDistributionDeDeveloperID(t *testing.T) {
	c, _ := caminhoPara(FormatoApp)
	if !strings.Contains(c.Nota, "Developer ID Application") {
		t.Fatalf("nota do .app nao nomeia a identidade certa: %q", c.Nota)
	}
	if !strings.Contains(c.Nota, "NAO substitui") {
		t.Fatalf("nota do .app nao avisa que Distribution nao serve: %q", c.Nota)
	}
}

// A REGRA DURA: `COMPLETO_VERIFICADO` nunca sem instalacao testada.
func TestCompletoVerificadoExigeInstalacao(t *testing.T) {
	completo := EvidenciaDeAssinatura{
		BuildExecutado: true, AssinaturaAplicada: true,
		AssinaturaVerificada: true, InstalacaoTestada: true,
		RunnerDisponivel: true,
	}
	if estado := AvaliarAssinatura(FormatoEXE, completo); estado != AceiteCompletoVerificado {
		t.Fatalf("com tudo provado o estado = %q; esperado COMPLETO_VERIFICADO", estado)
	}

	semInstalar := completo
	semInstalar.InstalacaoTestada = false
	estado := AvaliarAssinatura(FormatoEXE, semInstalar)
	if estado == AceiteCompletoVerificado {
		t.Fatal("*artefacto assinado e verificado mas nunca instalado declarado COMPLETO_VERIFICADO*")
	}
	if estado != AceiteProntoParaAssinatura {
		t.Fatalf("estado sem instalacao = %q; esperado PRONTO_PARA_ASSINATURA "+
			"(build pronto, prova final pendente)", estado)
	}
}

// Assinar sem verificar independentemente e' uma falha, nao um sucesso parcial.
func TestAssinadoSemVerificacaoEFalhado(t *testing.T) {
	ev := EvidenciaDeAssinatura{
		BuildExecutado: true, AssinaturaAplicada: true,
		AssinaturaVerificada: false, RunnerDisponivel: true,
	}
	if estado := AvaliarAssinatura(FormatoEXE, ev); estado != AceiteFalhado {
		t.Fatalf("estado = %q; esperado FAILED (*assinado sem verificacao nao e' prova*)", estado)
	}
}

// Sem build, o maximo e' "pronto para assinar".
func TestSemBuildEProntoParaAssinatura(t *testing.T) {
	ev := EvidenciaDeAssinatura{RunnerDisponivel: true}
	if estado := AvaliarAssinatura(FormatoIPA, ev); estado != AceiteProntoParaAssinatura {
		t.Fatalf("estado = %q; esperado PRONTO_PARA_ASSINATURA", estado)
	}
}

// Um formato fora da matriz e' NAO SUPORTADO, nao um erro nem um vazio.
func TestFormatoForaDaMatrizENaoSuportado(t *testing.T) {
	estado := AvaliarAssinatura("formato-que-nao-existe", EvidenciaDeAssinatura{RunnerDisponivel: true})
	if estado != AceiteNaoSuportado {
		t.Fatalf("estado = %q; esperado UNSUPPORTED", estado)
	}
}

// Um runner ausente e' BLOCKED_EXTERNAL, nao falha do produto.
func TestSemRunnerEBloqueadoExterno(t *testing.T) {
	ev := EvidenciaDeAssinatura{BuildExecutado: true, AssinaturaAplicada: true, AssinaturaVerificada: true}
	if estado := AvaliarAssinatura(FormatoIPA, ev); estado != AceiteBloqueadoExterno {
		t.Fatalf("estado = %q; esperado BLOCKED_EXTERNAL (*o runner e' requisito de plataforma*)",
			estado)
	}
}

// Uma identidade descartavel prova o pipeline, nao a confianca publica.
func TestIdentidadeDeTesteNuncaDaCompletoVerificado(t *testing.T) {
	ev := EvidenciaDeAssinatura{
		BuildExecutado: true, AssinaturaAplicada: true, AssinaturaVerificada: true,
		InstalacaoTestada: true, IdentidadeDeTeste: true, RunnerDisponivel: true,
	}
	if estado := AvaliarAssinatura(FormatoAPK, ev); estado != AceiteAssinadoComIdentidadeDeTeste {
		t.Fatalf("estado = %q; esperado ASSINADO_COM_IDENTIDADE_DE_TESTE "+
			"(*certificado de teste prova o pipeline, nao a confianca publica*)", estado)
	}
}

// A lista de estados e' fechada.
func TestEstadosDeAceiteSaoFechados(t *testing.T) {
	for _, e := range []EstadoDeAceite{
		AceiteCompletoVerificado, AceiteBuildVerificadoNaoAssinado,
		AceiteAssinadoComIdentidadeDeTeste, AceiteProntoParaAssinatura,
		AceiteConfigRequerido, AceitePlanoRequerido, AceiteBloqueadoExterno,
		AceiteExperimental, AceiteNaoSuportado, AceiteFalhado,
	} {
		if !e.Valido() {
			t.Fatalf("estado declarado rejeitado: %q", e)
		}
	}
	if EstadoDeAceite("quase_pronto").Valido() {
		t.Fatal("estado inventado aceite: *um estado novo tem de ser uma decisao*")
	}
}

// A auto-auditoria do catalogo nao encontra problemas.
func TestAuditarMatrizNaoEncontraProblemas(t *testing.T) {
	problemas := AuditarMatriz()
	if len(problemas) != 0 {
		for _, p := range problemas {
			t.Logf("  %s: %s", p.Formato, p.Mensagem)
		}
		t.Fatalf("%d problema(s) na matriz", len(problemas))
	}
}

// Toda a plataforma declarada tem pelo menos um formato, e nenhum formato fica orfao.
func TestTodaPlataformaTemFormato(t *testing.T) {
	for _, plataforma := range []PlataformaDeAssinatura{
		PlataformaAndroid, PlataformaIOS, PlataformaMacOS,
		PlataformaWindows, PlataformaLinux, PlataformaWeb, PlataformaOCI,
	} {
		if len(FormatosPorPlataforma(plataforma)) == 0 {
			t.Fatalf("plataforma %q sem nenhum formato na matriz", plataforma)
		}
	}
	// E o inverso: todo o formato tem plataforma valida (AuditarMatriz ja cobre, mas explicito).
	for _, c := range MatrizDeAssinatura() {
		if !c.Plataforma.Valida() {
			t.Fatalf("formato %q com plataforma invalida %q", c.Formato, c.Plataforma)
		}
	}
}

// A matriz e' determinista: duas chamadas devolvem a mesma coisa.
//
// *Um catalogo que muda de forma entre chamadas deixa de ser comparavel entre dois hosts.*
func TestMatrizDeterminista(t *testing.T) {
	a := MatrizDeAssinatura()
	b := MatrizDeAssinatura()
	if len(a) != len(b) {
		t.Fatalf("tamanhos diferentes: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("entrada %d difere entre chamadas", i)
		}
	}
}
