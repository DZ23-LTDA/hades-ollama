package agent

// Testes discriminantes da FASE 28.6.
//
// Cada teste aqui falha se o defeito correspondente for introduzido. Os que mais importam sao os
// que impedem o plano de MENTIR: `UNKNOWN` promovido a `ABSENT`, `--version` promovido a
// `HEALTHY`, e um plano bloqueado que nao diz o que ainda pode ser feito.

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func resolvedorFalso(respostas map[string]ToolProbe) *ResolverDeToolchain {
	return &ResolverDeToolchain{
		GOOS: "windows",
		Prober: func(_ context.Context, req ToolRequirement) ToolProbe {
			if p, ok := respostas[req.Nome]; ok {
				return p
			}
			return ToolProbe{Requisito: req, Estado: ToolAbsent, Detalhe: "ausente no simulador"}
		},
	}
}

func tudoSaudavel() map[string]ToolProbe {
	saudavel := func(nome string) ToolProbe {
		return ToolProbe{Estado: ToolHealthy, Caminho: "/usr/bin/" + nome, Versao: nome + " 1.0"}
	}
	saida := map[string]ToolProbe{}
	for _, nome := range []string{"go", "inno-setup", "powershell", "node", "npm", "eas", "java", "adb", "xcodebuild", "codesign", "dpkg-deb", "rpmbuild", "docker", "cosign"} {
		saida[nome] = saudavel(nome)
	}
	return saida
}

// Um alvo que nao existe no catalogo tem de falhar, e a mensagem tem de listar os que existem.
func TestAlvoDesconhecidoFalhaSemInventar(t *testing.T) {
	r := resolvedorFalso(tudoSaudavel())
	_, err := r.Plano(context.Background(), "blackberry-os")
	if err == nil {
		t.Fatal("alvo inexistente devolveu plano; um catalogo que aceita qualquer nome nao e' catalogo")
	}
	if !strings.Contains(err.Error(), "windows") || !strings.Contains(err.Error(), "android") {
		t.Fatalf("erro nao lista alvos conhecidos: %v", err)
	}
}

// A regra central: `UNKNOWN` NUNCA pode ser convertido em `ABSENT`.
//
// Se este teste falhar, o produto vai dizer "nao tens isto" sobre algo que so' nao foi possivel
// determinar -- e vai tentar instalar por cima do que ja' existe.
func TestUnknownNuncaViraAbsent(t *testing.T) {
	// Requisito sem comando verificavel: tem de sair `UNKNOWN`.
	r := resolvedorFalso(tudoSaudavel())
	plano, err := r.Plano(context.Background(), "android")
	if err != nil {
		t.Fatalf("plano android: %v", err)
	}
	probe, ok := plano.Exigencia("google-play-service-account")
	if !ok {
		t.Fatal("exigencia google-play-service-account ausente do plano")
	}
	if probe.Estado != ToolUnknown {
		t.Fatalf("estado = %s; sem comando verificavel tem de ser %s, nunca %s",
			probe.Estado, ToolUnknown, ToolAbsent)
	}
	if probe.Estado == ToolAbsent {
		t.Fatal("UNKNOWN foi promovido a ABSENT: o plano vai instalar por cima do que ja' existe")
	}
}

// `--version` respondeu nao e' o mesmo que a ferramenta FUNCIONA.
func TestVersionRespondeNaoEHealth(t *testing.T) {
	r := resolvedorFalso(map[string]ToolProbe{
		"go":         {Estado: ToolUnverified, Versao: "go version go1.26.9", Detalhe: "falta health funcional"},
		"inno-setup": {Estado: ToolUnverified, Versao: "ISCC 6.7.3"},
		"powershell": {Estado: ToolUnverified, Versao: "5.1"},
	})
	plano, err := r.Plano(context.Background(), "windows")
	if err != nil {
		t.Fatalf("plano windows: %v", err)
	}
	if plano.Pronto() {
		t.Fatal("plano declarou-se pronto apenas porque as ferramentas responderam `--version`")
	}
	for _, probe := range plano.Probes {
		if probe.Estado == ToolHealthy {
			t.Fatalf("%s marcada HEALTHY sem health funcional", probe.Requisito.Nome)
		}
	}
}

// Versao abaixo do minimo declarado pelo alvo tem de ser OUTDATED, nao UNVERIFIED.
func TestVersaoAbaixoDoMinimoEEOutdated(t *testing.T) {
	if got := compararVersao("go version go1.20.3", "1.22"); got != -1 {
		t.Fatalf("compararVersao(1.20.3, 1.22) = %d; esperado -1", got)
	}
	if got := compararVersao("v1.26.9", "1.22"); got != 1 {
		t.Fatalf("compararVersao(1.26.9, 1.22) = %d; esperado 1", got)
	}
	if got := compararVersao("1.22.0", "1.22"); got != 0 {
		t.Fatalf("compararVersao(1.22.0, 1.22) = %d; esperado 0", got)
	}
	// Pre-release nao pode inventar componentes.
	if got := compararVersao("1.22.0-rc1", "1.22"); got != 0 {
		t.Fatalf("compararVersao(1.22.0-rc1, 1.22) = %d; esperado 0", got)
	}
}

// Alvo que exige um host diferente tem de ser bloqueio EXTERNO explicito, sem sondar o impossivel.
func TestAlvoNativoDeOutroHostNaoSonda(t *testing.T) {
	chamadas := 0
	r := &ResolverDeToolchain{
		GOOS: "windows",
		Prober: func(_ context.Context, req ToolRequirement) ToolProbe {
			chamadas++
			return ToolProbe{Requisito: req, Estado: ToolAbsent}
		},
	}
	plano, err := r.Plano(context.Background(), "ios")
	if err != nil {
		t.Fatalf("plano ios em windows: %v", err)
	}
	if chamadas != 0 {
		t.Fatalf("sondou %d ferramentas de um alvo que exige macOS; devia classificar o host primeiro", chamadas)
	}
	if plano.Pronto() {
		t.Fatal("plano iOS em Windows declarou-se pronto")
	}
	if !strings.Contains(strings.Join(plano.Blockers, " "), "macos") {
		t.Fatalf("bloqueio nao menciona o host exigido: %v", plano.Blockers)
	}
}

// Com blockers, o trabalho independente NUNCA fica vazio.
//
// *"Depende de conta externa" nao e' razao para parar o que nao depende.*
func TestBloqueadoAindaDizOQuePodeAvancar(t *testing.T) {
	r := resolvedorFalso(map[string]ToolProbe{})
	plano, err := r.Plano(context.Background(), "windows")
	if err != nil {
		t.Fatalf("plano windows: %v", err)
	}
	if plano.Pronto() {
		t.Fatal("plano sem nenhuma ferramenta declarou-se pronto")
	}
	if len(plano.Blockers) == 0 {
		t.Fatal("plano sem ferramentas nao tem blockers")
	}
	if len(plano.TrabalhoIndependente) == 0 {
		t.Fatal("plano bloqueado nao diz o que ainda pode ser feito")
	}
}

// Ferramenta OPCIONAL ausente nao pode bloquear o alvo.
func TestFerramentaOpcionalAusenteNaoBloqueia(t *testing.T) {
	respostas := tudoSaudavel()
	delete(respostas, "adb")  // opcional em android
	delete(respostas, "eas")  // opcional em android
	delete(respostas, "java") // opcional em android
	r := resolvedorFalso(respostas)
	// O alvo android exige `node` obrigatoriamente; as restantes sao opcionais.
	plano, err := r.Plano(context.Background(), "android")
	if err != nil {
		t.Fatalf("plano android: %v", err)
	}
	if !plano.Pronto() {
		t.Fatalf("ferramenta opcional ausente bloqueou o alvo: %v", plano.Blockers)
	}
	if len(plano.Blockers) != 0 {
		t.Fatalf("blockers com so' opcionais ausentes: %v", plano.Blockers)
	}
}

// Ferramenta OBRIGATORIA quebrada bloqueia.
func TestFerramentaObrigatoriaQuebradaBloqueia(t *testing.T) {
	respostas := tudoSaudavel()
	respostas["go"] = ToolProbe{Estado: ToolBroken, Caminho: "/usr/bin/go", Detalhe: "responde mas falha"}
	r := resolvedorFalso(respostas)
	plano, err := r.Plano(context.Background(), "windows")
	if err != nil {
		t.Fatalf("plano windows: %v", err)
	}
	if plano.Pronto() {
		t.Fatal("ferramenta obrigatoria quebrada nao bloqueou")
	}
	if !strings.Contains(strings.Join(plano.Blockers, " "), "go") {
		t.Fatalf("blocker nao identifica go: %v", plano.Blockers)
	}
}

// Ausencia total tem de produzir ausencia nos tres obrigatorios de windows.
func TestAusenciaTotalListaOsTresObrigatorios(t *testing.T) {
	r := resolvedorFalso(map[string]ToolProbe{})
	// windows passou a ser o BINARIO (1 obrigatorio). Os tres (go, inno-setup, powershell) vivem
	// agora em windows-installer, que descreve o instalador. Ver E-1308.
	plano, err := r.Plano(context.Background(), "windows-installer")
	if err != nil {
		t.Fatalf("plano windows-installer: %v", err)
	}
	if len(plano.Blockers) != 3 {
		t.Fatalf("esperava 3 blockers (go, inno-setup, powershell), obtive %d: %v", len(plano.Blockers), plano.Blockers)
	}
}

// O catalogo nao pode ter nomes repetidos nem alvos vazios.
func TestCatalogoSemDuplicadosNemVazios(t *testing.T) {
	vistos := map[string]bool{}
	for _, alvo := range CatalogoDeAlvos() {
		if strings.TrimSpace(alvo.Nome) == "" {
			t.Fatal("alvo sem nome no catalogo")
		}
		chave := strings.ToLower(alvo.Nome)
		if vistos[chave] {
			t.Fatalf("alvo duplicado no catalogo: %s", alvo.Nome)
		}
		vistos[chave] = true
		if len(alvo.Exigencias) == 0 {
			t.Fatalf("alvo %s sem exigencias declaradas", alvo.Nome)
		}
		for _, req := range alvo.Exigencias {
			if strings.TrimSpace(req.Nome) == "" {
				t.Fatalf("alvo %s tem exigencia sem nome", alvo.Nome)
			}
			if strings.TrimSpace(req.Motivo) == "" {
				t.Fatalf("alvo %s, exigencia %s sem motivo: o plano nao pode exigir sem explicar", alvo.Nome, req.Nome)
			}
		}
	}
}

// O resolvedor real tem de ser honesto sobre o proprio host: `go` pode nao estar no PATH.
//
// Este teste NAO falha quando o go esta' ausente -- falha se o resolvedor disser `ABSENT` com
// detalhe vazio ou declarar `HEALTHY` sem health funcional.
func TestSondaRealNuncaDeclaraHealthy(t *testing.T) {
	probe := SondarFerramentaNoPATH(context.Background(), ToolRequirement{
		Nome: "go", Comando: "go", Obrigatorio: true, Motivo: "teste",
	})
	if probe.Estado == ToolHealthy {
		t.Fatal("SondarFerramentaNoPATH declarou HEALTHY: --version nao e' health funcional")
	}
	if probe.Estado != ToolAbsent && probe.Estado != ToolUnverified && probe.Estado != ToolOutdated && probe.Estado != ToolBroken {
		t.Fatalf("estado inesperado: %s", probe.Estado)
	}
	if probe.Detalhe == "" {
		t.Fatal("sonda sem detalhe: o Mission Control nao pode mostrar um estado mudo")
	}
}

// `exec.LookPath` falhar nao pode ser confundido com sucesso.
func TestProberAusenteUsaOSondadorReal(t *testing.T) {
	r := &ResolverDeToolchain{GOOS: "windows"}
	plano, err := r.Plano(context.Background(), "windows-installer")
	if err != nil {
		t.Fatalf("plano sem prober: %v", err)
	}
	if len(plano.Probes) != 3 {
		t.Fatalf("esperava 3 probes, obtive %d", len(plano.Probes))
	}
	for _, probe := range plano.Probes {
		if probe.Estado == ToolHealthy {
			t.Fatalf("%s: sem health funcional nao pode ser HEALTHY", probe.Requisito.Nome)
		}
	}
}

// O prober de producao tem de existir e nao pode ser nil.
func TestNovoResolverTemProber(t *testing.T) {
	r := NovoResolverDeToolchain()
	if r.Prober == nil {
		t.Fatal("NovoResolverDeToolchain devolveu resolvedor sem prober")
	}
	if _, err := r.Plano(context.Background(), "web"); err != nil {
		t.Fatalf("plano web: %v", err)
	}
}

// Erro de ferramenta sem comando existe para documentar a fronteira.
func TestErrFerramentaSemComandoExiste(t *testing.T) {
	if !errors.Is(ErrFerramentaSemComando, ErrFerramentaSemComando) {
		t.Fatal("ErrFerramentaSemComando nao e' um erro valido")
	}
}

// TestMotivosDeNaoProntoNaoDizZeroQuandoFaltaVerificacao fixa o defeito do E-1307.
//
// # O DEFEITO
//
// `Pronto()` devolve false quando uma exigencia obrigatoria esta' `PRESENT_COMPATIBLE_UNVERIFIED`,
// mas `Blockers` fica vazio -- porque `Bloqueia()` so' e' verdadeiro para ausencia/incompatibilidade.
// O erro do `Construir` contava `len(Blockers)` e dizia:
//
//	plano de capability incompleto: nao ha' build a executar: 0 blocker(s):
//
// Zero blockers e o plano nao estava pronto. Medido contra o servidor real.
//
//	*Um erro que conta os blockers e ignora as ferramentas nao verificadas reporta "0 blockers" e
//	nao diz o que falta.*
func TestMotivosDeNaoProntoNaoDizZeroQuandoFaltaVerificacao(t *testing.T) {
	plano := CapabilityPlan{
		Alvo: "linux",
		Probes: []ToolProbe{
			{
				Requisito: ToolRequirement{Nome: "go", Obrigatorio: true},
				Estado:    ToolUnverified,
				Detalhe:   "presente e respondeu; falta health funcional",
			},
		},
	}
	if len(plano.Blockers) != 0 {
		t.Fatalf("pre-condicao: Blockers devia estar vazio, tem %v", plano.Blockers)
	}
	if plano.Pronto() {
		t.Fatal("pre-condicao: Pronto() devia ser false com uma exigencia nao verificada")
	}
	motivos := plano.MotivosDeNaoPronto()
	if len(motivos) == 0 {
		t.Fatal("MotivosDeNaoPronto devolveu vazio -- e' exactamente o defeito que isto fixa")
	}
	juntos := strings.Join(motivos, " | ")
	if !strings.Contains(juntos, "go") {
		t.Fatalf("o motivo devia nomear a ferramenta: %q", juntos)
	}
	if !strings.Contains(juntos, "PRESENT_COMPATIBLE_UNVERIFIED") {
		t.Fatalf("o motivo devia nomear o estado real: %q", juntos)
	}
}

// TestMotivosDeNaoProntoNaoRepeteOBlocker garante que um blocker nao conta duas vezes.
func TestMotivosDeNaoProntoNaoRepeteOBlocker(t *testing.T) {
	plano := CapabilityPlan{
		Alvo:     "windows",
		Blockers: []string{"inno-setup (ABSENT): nao encontrado no PATH resolvido"},
		Probes: []ToolProbe{
			{
				Requisito: ToolRequirement{Nome: "inno-setup", Obrigatorio: true},
				Estado:    ToolAbsent,
				Detalhe:   "nao encontrado no PATH resolvido",
			},
		},
	}
	motivos := plano.MotivosDeNaoPronto()
	if len(motivos) != 1 {
		t.Fatalf("esperado 1 motivo (sem repetir), obtidos %d: %v", len(motivos), motivos)
	}
}

// TestWindowsNaoPrometeInstalador fixa o defeito do E-1308.
//
// # O DEFEITO
//
// O alvo windows dizia "Instalador .exe nao assinado (Inno Setup)" e exigia inno-setup e
// powershell. Medido com as duas ferramentas no PATH: o uild.start deu 200 e produziu
// uild-alvo-windows.exe -- e ZERO instaladores, porque o unico provider corre go build.
//
//	*Um alvo que exige ferramentas que nenhum provider invoca, e descreve um artefacto que nenhum
//	provider produz, nao descreve o que o HADES faz: descreve o que se esperava que fizesse.*
func TestWindowsNaoPrometeInstalador(t *testing.T) {
	var binario, instalador *AlvoDeclarado
	for i, alvo := range CatalogoDeAlvos() {
		switch alvo.Nome {
		case "windows":
			binario = &CatalogoDeAlvos()[i]
		case "windows-installer":
			instalador = &CatalogoDeAlvos()[i]
		}
	}
	if binario == nil {
		t.Fatal("o alvo windows desapareceu do catalogo")
	}
	if instalador == nil {
		t.Fatal("o instalador devia continuar declarado como windows-installer, nao desaparecer")
	}
	// O binario so' exige go -- e' o que o provider entrega.
	for _, req := range binario.Exigencias {
		if req.Nome != "go" {
			t.Fatalf("o alvo windows (binario) nao devia exigir %q: nenhum provider a invoca", req.Nome)
		}
	}
	// O instalador declara as tres, e o provider NAO o cobre.
	if len(instalador.Exigencias) != 3 {
		t.Fatalf("windows-installer devia exigir 3 ferramentas, tem %d", len(instalador.Exigencias))
	}
	for _, alvo := range (ProviderDeBuildGo{}).Alvos() {
		if strings.EqualFold(alvo, "windows-installer") {
			t.Fatal("o ProviderDeBuildGo nao pode declarar que cobre o instalador: so' corre go build")
		}
	}
}
