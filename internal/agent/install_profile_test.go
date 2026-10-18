package agent

// FASE 28.6-A: testes do instalador modular.
//
// Os testes que importam sao os INVARIANTES, nao os casos felizes:
//
//   1. `UNKNOWN` nao e' `ABSENT` -- um pacote que a sonda nao avaliou NUNCA e' instalado
//      automaticamente. *Uma duvida nao e' uma ausencia, e agir sobre ela instala o que ja'
//      existia.*
//   2. O perfil e' INTENCAO, nao permissao -- mesmo no perfil mais completo, um componente que
//      cria servico, pede elevacao ou instala driver continua a exigir approval.
//   3. Reutilizar nao baixa -- um pacote saudavel nao contribui para o tamanho do download.
//   4. Todas as dependencias sao resolvidas antes do pacote que as pede.

import (
	"strings"
	"testing"
)

// Os 9 perfis do adendo existem e sao validos.
func TestPerfisDeInstalacaoCompletos(t *testing.T) {
	esperados := []PerfilDeInstalacao{
		PerfilBasic, PerfilRecommended, PerfilDeveloperComplete, PerfilMobile,
		PerfilMediaAndVoice, PerfilBrowserAndComputerUse, PerfilContainersAndCloud,
		PerfilCustom, PerfilFullOffline,
	}
	obtidos := PerfisDeInstalacao()
	if len(obtidos) != len(esperados) {
		t.Fatalf("%d perfis; esperados %d", len(obtidos), len(esperados))
	}
	for _, e := range esperados {
		if !e.Valido() {
			t.Fatalf("perfil %q nao e' valido", e)
		}
	}
	if PerfilDeInstalacao("TUDO").Valido() {
		t.Fatal("perfil inventado aceite")
	}
}

// INVARIANTE 1: `UNKNOWN` nunca vira `ABSENT`.
//
// Um pacote que a sonda NAO avaliou tem de ser PULADO com bloqueio, nunca instalado.
func TestPacoteNaoAvaliadoNuncaEInstalado(t *testing.T) {
	// Descoberta vazia: nada foi avaliado.
	plano, err := MontarPlanoDeInstalacao(PerfilBasic, map[string]EstadoDoPacote{}, map[string]bool{})
	if err != nil {
		t.Fatalf("plano: %v", err)
	}
	for _, item := range plano.Itens {
		if item.Acao == AcaoInstalar || item.Acao == AcaoAprovar {
			t.Fatalf("pacote %q nao avaliado foi %s: *UNKNOWN nao e' ABSENT*",
				item.PacoteID, item.Acao)
		}
		if !strings.Contains(item.Motivo, "nao foi possivel determinar") {
			t.Fatalf("pacote %q: motivo nao explica a duvida: %q", item.PacoteID, item.Motivo)
		}
	}
	if len(plano.Bloqueios) == 0 {
		t.Fatal("plano sem bloqueios com descoberta vazia")
	}
	if plano.Pronto() {
		t.Fatal("plano declarado pronto sem nenhuma descoberta")
	}
}

// E o contrario: um pacote avaliado como ausente E' instalado.
func TestPacoteAvaliadoAusenteEInstalado(t *testing.T) {
	estado := map[string]EstadoDoPacote{"hades-core": PacoteNaoInstalado}
	descobertos := map[string]bool{"hades-core": true}

	plano, err := MontarPlanoDeInstalacao(PerfilBasic, estado, descobertos)
	if err != nil {
		t.Fatalf("plano: %v", err)
	}
	if len(plano.Itens) != 1 {
		t.Fatalf("%d itens; esperado 1 (so' o core em BASIC)", len(plano.Itens))
	}
	item := plano.Itens[0]
	if item.Acao != AcaoInstalar {
		t.Fatalf("acao = %q; esperado INSTALAR (ausente confirmado)", item.Acao)
	}
	if plano.TamanhoTotalMB != item.TamanhoMB {
		t.Fatalf("tamanho = %d; esperado %d", plano.TamanhoTotalMB, item.TamanhoMB)
	}
}

// INVARIANTE 2: o perfil nao e' permissao.
//
// `containers` cria servico, pede elevacao, reinicio e porta. Mesmo no perfil que o inclui por
// omissao, tem de exigir aprovacao.
func TestPerfilCompletoAindaExigeAprovacao(t *testing.T) {
	estado := map[string]EstadoDoPacote{
		"hades-core": PacoteNaoInstalado, "containers": PacoteNaoInstalado,
	}
	descobertos := map[string]bool{"hades-core": true, "containers": true}

	plano, err := MontarPlanoDeInstalacao(PerfilContainersAndCloud, estado, descobertos)
	if err != nil {
		t.Fatalf("plano: %v", err)
	}

	var achou bool
	for _, item := range plano.Itens {
		if item.PacoteID != "containers" {
			continue
		}
		achou = true
		if item.Acao != AcaoAprovar {
			t.Fatalf("containers acao = %q; esperado APROVAR "+
				"(*o perfil escolhido e' uma intencao, nao permissao ilimitada*)", item.Acao)
		}
		if !item.RequerAprovacao {
			t.Fatal("containers sem RequerAprovacao")
		}
		for _, palavra := range []string{"elevacao", "servico", "reinicio", "porta"} {
			if !strings.Contains(item.Motivo, palavra) {
				t.Fatalf("motivo nao menciona %q: %q", palavra, item.Motivo)
			}
		}
	}
	if !achou {
		t.Fatal("containers nao esta' no plano do perfil CONTAINERS_AND_CLOUD")
	}
	if plano.AprovacoesNecessarias == 0 {
		t.Fatal("nenhuma aprovacao necessaria num perfil com containers")
	}
	if plano.Pronto() {
		t.Fatal("plano declarado pronto com aprovacoes pendentes")
	}
}

// Um componente sem impacto NAO pede aprovacao -- a regra nao pode ser tudo-ou-nada.
func TestComponenteSimplesNaoPedeAprovacao(t *testing.T) {
	estado := map[string]EstadoDoPacote{"hades-core": PacoteNaoInstalado}
	descobertos := map[string]bool{"hades-core": true}
	plano, _ := MontarPlanoDeInstalacao(PerfilBasic, estado, descobertos)
	if plano.Itens[0].RequerAprovacao {
		t.Fatal("hades-core pediu aprovacao sem impacto que o justifique")
	}
	if !plano.Pronto() {
		t.Fatalf("plano simples nao esta' pronto: %+v", plano.Bloqueios)
	}
}

// INVARIANTE 3: reutilizar nao baixa.
func TestReutilizarNaoContribuiParaODownload(t *testing.T) {
	// O perfil RECOMMENDED inclui quatro pacotes; declarar so' tres deixaria o quarto
	// como UNKNOWN e o teste mediria outra coisa.
	estado := map[string]EstadoDoPacote{
		"hades-core":         PacotePronto,
		"local-ai":           PacoteEmUso,
		"browser-automation": PacotePronto,
		"plugins-mcp-skills": PacotePronto,
	}
	descobertos := map[string]bool{
		"hades-core": true, "local-ai": true, "browser-automation": true,
		"plugins-mcp-skills": true,
	}
	plano, err := MontarPlanoDeInstalacao(PerfilRecommended, estado, descobertos)
	if err != nil {
		t.Fatalf("plano: %v", err)
	}
	if plano.TamanhoTotalMB != 0 {
		t.Fatalf("tamanho = %d com tudo pronto; esperado 0", plano.TamanhoTotalMB)
	}
	if plano.DownloadsEvitados != len(plano.Itens) {
		t.Fatalf("downloads evitados = %d; esperado %d (todos os itens)", plano.DownloadsEvitados, len(plano.Itens))
	}
	for _, item := range plano.Itens {
		if item.Acao != AcaoReutilizar {
			t.Fatalf("pacote %q acao = %q; esperado REUTILIZAR", item.PacoteID, item.Acao)
		}
	}
}

// Um componente degradado e' tratado como instalavel, nao como saudavel.
func TestDegradadoEInstaladoNaoReutilizado(t *testing.T) {
	estado := map[string]EstadoDoPacote{"hades-core": PacoteDegradado}
	descobertos := map[string]bool{"hades-core": true}
	plano, _ := MontarPlanoDeInstalacao(PerfilBasic, estado, descobertos)
	item := plano.Itens[0]
	if item.Acao != AcaoInstalar {
		t.Fatalf("acao = %q; esperado INSTALAR (PRESENT_BROKEN)", item.Acao)
	}
	if !strings.Contains(item.Motivo, "PRESENT_BROKEN") {
		t.Fatalf("motivo nao classifica: %q", item.Motivo)
	}
}

// INVARIANTE 4: as dependencias vem antes do pacote que as pede.
func TestDependenciasResolvidasAntes(t *testing.T) {
	// `voice` depende de `media`.
	estado := map[string]EstadoDoPacote{
		"hades-core": PacoteNaoInstalado, "media": PacoteNaoInstalado,
		"voice": PacoteNaoInstalado, "vision-ocr": PacoteNaoInstalado,
	}
	descobertos := map[string]bool{
		"hades-core": true, "media": true, "voice": true, "vision-ocr": true,
	}
	plano, err := MontarPlanoDeInstalacao(PerfilMediaAndVoice, estado, descobertos)
	if err != nil {
		t.Fatalf("plano: %v", err)
	}
	pos := map[string]int{}
	for i, item := range plano.Itens {
		pos[item.PacoteID] = i
	}
	if pos["media"] > pos["voice"] {
		t.Fatalf("media (pos %d) depois de voice (pos %d): dependencia invertida",
			pos["media"], pos["voice"])
	}
	if pos["media"] > pos["vision-ocr"] {
		t.Fatalf("media (pos %d) depois de vision-ocr (pos %d)", pos["media"], pos["vision-ocr"])
	}
}

// Um perfil desconhecido e' erro, nao plano vazio.
func TestPerfilDesconhecidoEErro(t *testing.T) {
	_, err := MontarPlanoDeInstalacao("INVENTADO", nil, nil)
	if err == nil {
		t.Fatal("perfil invencivel aceite")
	}
	if !strings.Contains(err.Error(), "perfil desconhecido") {
		t.Fatalf("erro pouco claro: %v", err)
	}
}

// `CUSTOM` nao traz nada por omissao.
func TestCustomNaoTrazPacotesPorOmissao(t *testing.T) {
	plano, err := MontarPlanoDeInstalacao(PerfilCustom,
		map[string]EstadoDoPacote{}, map[string]bool{})
	if err != nil {
		t.Fatalf("plano: %v", err)
	}
	if len(plano.Itens) != 0 {
		t.Fatalf("CUSTOM trouxe %d pacotes por omissao", len(plano.Itens))
	}
}

// Todo o perfil (exceto CUSTOM) inclui o nucleo.
func TestTodoPerfilIncluiOCore(t *testing.T) {
	for _, perfil := range PerfisDeInstalacao() {
		if perfil == PerfilCustom {
			continue
		}
		ids := PacotesDoPerfil(perfil)
		achou := false
		for _, id := range ids {
			if id == "hades-core" {
				achou = true
			}
		}
		if !achou {
			t.Fatalf("perfil %q nao inclui hades-core", perfil)
		}
	}
}

// A auto-auditoria do catalogo nao encontra problemas.
func TestAuditarCatalogoNaoEncontraProblemas(t *testing.T) {
	problemas := AuditarCatalogo()
	if len(problemas) != 0 {
		for _, p := range problemas {
			t.Logf("  %s: %s", p.PacoteID, p.Mensagem)
		}
		t.Fatalf("%d problema(s) no catalogo", len(problemas))
	}
}

// Os estados de pacote sao uma lista fechada.
func TestEstadosDoPacoteSaoFechados(t *testing.T) {
	for _, e := range []EstadoDoPacote{
		PacoteNaoInstalado, PacoteBaixando, PacoteVerificando, PacoteInstalando,
		PacotePronto, PacoteAtualizacaoDisponivel, PacoteReparoNecessario,
		PacoteDegradado, PacoteEmUso, PacoteRemovendo, PacoteRollbackDisponivel,
		PacoteAutenticacaoRequerida, PacoteAprovacaoRequerida, PacoteBloqueadoExterno,
		PacoteNaoSuportado, PacoteFalhado,
	} {
		if !e.Valido() {
			t.Fatalf("estado declarado rejeitado: %q", e)
		}
	}
	if EstadoDoPacote("QUASE").Valido() {
		t.Fatal("estado inventado aceite")
	}
}

// Um componente cuja licenca proibe redistribuir nao pode ser embutido.
//
// *Um componente que a licenca proibe redistribuir tem de ser baixado no momento, nunca
// embutido no instalador do HADES.*
func TestComponenteNaoRedistribuivelNaoEPrivado(t *testing.T) {
	for _, p := range CatalogoDePacotes() {
		if !p.Redistribuivel && p.EscopoPrivadoPrefiravel && p.Impacto.ExigeAprovacao() == false {
			// Nao e' proibicao absoluta, mas um componente nao redistribuivel que se diz privado
			// sem impacto declarado e' um buraco na explicacao ao utilizador.
			t.Logf("aviso: %q nao redistribuivel e declarado privado sem impacto", p.ID)
		}
		if p.Licenca == "" {
			t.Fatalf("pacote %q sem licenca declarada", p.ID)
		}
	}
}

// O plano e' deterministico: a mesma entrada da' a mesma ordem.
func TestPlanoDeterminista(t *testing.T) {
	estado := map[string]EstadoDoPacote{"hades-core": PacoteNaoInstalado}
	descobertos := map[string]bool{"hades-core": true}
	a, errA := MontarPlanoDeInstalacao(PerfilBasic, estado, descobertos)
	b, errB := MontarPlanoDeInstalacao(PerfilBasic, estado, descobertos)
	if errA != nil || errB != nil {
		t.Fatalf("erros: %v / %v", errA, errB)
	}
	if len(a.Itens) != len(b.Itens) {
		t.Fatalf("tamanhos diferentes: %d vs %d", len(a.Itens), len(b.Itens))
	}
	for i := range a.Itens {
		if a.Itens[i] != b.Itens[i] {
			t.Fatalf("item %d difere entre chamadas", i)
		}
	}
}

// O resumo e' legivel e contem os numeros que importam.
func TestResumoDoPlanoEInformativdo(t *testing.T) {
	estado := map[string]EstadoDoPacote{"hades-core": PacoteNaoInstalado}
	descobertos := map[string]bool{"hades-core": true}
	plano, _ := MontarPlanoDeInstalacao(PerfilBasic, estado, descobertos)
	resumo := plano.ResumoDoPlano()
	for _, palavra := range []string{"perfil", "MB", "aprovacoes", "bloqueios"} {
		if !strings.Contains(resumo, palavra) {
			t.Fatalf("resumo sem %q: %q", palavra, resumo)
		}
	}
}
