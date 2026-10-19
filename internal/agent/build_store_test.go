package agent

// FASE 28.7: testes do store de builds.
//
// Estes testes fixam as DUAS promessas que um store de estado pode quebrar sem falhar em nada:
//
//  1. cancelar um build que ja' terminou tem de FALHAR, nao devolver sucesso. Um `cancelar` que
//     devolve `nil` sobre um build concluido e' um botao que mente sobre o que fez.
//  2. um build cancelado NAO pode ser "des-cancelado" pelo `Terminar`. Se o provider acabar depois
//     do cancelamento chegar, o estado final tem de continuar CANCELADO -- senao o utilizador ve'
//     "cancelado" e o store diz "concluido", e um dos dois esta' a mentir.
//
//	*Um cancelamento que o resultado desfaz nao e' um cancelamento: e' uma sugestao.*

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStoreBuildRegistarEObter prova o caminho felizesimo: nasce, e' encontrado, e nasce NA_FILA.
func TestStoreBuildRegistarEObter(t *testing.T) {
	s := NovoBuildStore()
	id, err := s.Registar("windows", "/tmp/proj", "/tmp/dest")
	if err != nil {
		t.Fatalf("registar: %v", err)
	}
	if len(id) != 24 {
		t.Fatalf("ID devia ter 24 caracteres hex, tem %d: %q", len(id), id)
	}
	r, err := s.Obter(id)
	if err != nil {
		t.Fatalf("obter: %v", err)
	}
	if r.Estado != BuildNaFila {
		t.Fatalf("estado devia ser NA_FILA: %q", r.Estado)
	}
	if r.Alvo != "windows" || r.Raiz != "/tmp/proj" || r.Destino != "/tmp/dest" {
		t.Fatalf("campos mal guardados: %+v", r)
	}
	// As invariantes de fronteira vem do store, nao do chamador.
	if !r.NadaAssinado || !r.NadaPublicado {
		t.Fatalf("um build nunca assina nem publica: %+v", r)
	}
	if !r.Cancelavel {
		t.Fatal("um build NA_FILA tem de ser cancelavel")
	}
	if r.TerminadoEm != nil {
		t.Fatal("um build NA_FILA nao tem terminado_em")
	}
}

// TestStoreBuildDoisIDsSaoDiferentes prova que o ID nao e' previsivel nem reutilizado.
func TestStoreBuildDoisIDsSaoDiferentes(t *testing.T) {
	s := NovoBuildStore()
	vistos := map[string]bool{}
	for i := range 50 {
		id, err := s.Registar("linux", "/r", "/d")
		if err != nil {
			t.Fatalf("registar %d: %v", i, err)
		}
		if vistos[id] {
			t.Fatalf("ID repetido apos %d registos: %q", i, id)
		}
		vistos[id] = true
	}
}

// TestStoreBuildIniciarMudaParaAExecutar e' a transicao NA_FILA -> A_EXECUTAR.
func TestStoreBuildIniciarMudaParaAExecutar(t *testing.T) {
	s := NovoBuildStore()
	id, _ := s.Registar("windows", "/r", "/d")
	ctx, err := s.Iniciar(id)
	if err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	if ctx == nil {
		t.Fatal("iniciar devia dar contexto")
	}
	if ctx.Err() != nil {
		t.Fatal("o contexto nao devia estar cancelado a nascenca")
	}
	r, _ := s.Obter(id)
	if r.Estado != BuildAExecutar {
		t.Fatalf("estado devia ser A_EXECUTAR: %q", r.Estado)
	}
}

// TestStoreBuildIniciarDuasVezesRecusa impede dois executores no mesmo build.
func TestStoreBuildIniciarDuasVezesRecusa(t *testing.T) {
	s := NovoBuildStore()
	id, _ := s.Registar("windows", "/r", "/d")
	if _, err := s.Iniciar(id); err != nil {
		t.Fatalf("primeiro iniciar: %v", err)
	}
	if _, err := s.Iniciar(id); err == nil {
		t.Fatal("segundo iniciar devia ser recusado -- dois executores no mesmo build")
	}
}

// TestStoreBuildCancelarDevolveContextoCancelado prova que o cancelamento chega ao executor.
//
// Sem isto, `cancelar` mudaria o estado no store e deixaria o processo a correr -- que e' a pior
// combinacao possivel: o utilizador le' "cancelado" e o trabalho continua.
func TestStoreBuildCancelarDevolveContextoCancelado(t *testing.T) {
	s := NovoBuildStore()
	id, _ := s.Registar("windows", "/r", "/d")
	ctx, err := s.Iniciar(id)
	if err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	if err := s.Cancelar(id); err != nil {
		t.Fatalf("cancelar: %v", err)
	}
	select {
	case <-ctx.Done():
		// Bom: o contexto foi cancelado.
	case <-time.After(2 * time.Second):
		t.Fatal("o contexto do build nao foi cancelado -- o executor continuaria a correr")
	}
	r, _ := s.Obter(id)
	if r.Estado != BuildCancelado {
		t.Fatalf("estado devia ser CANCELADO: %q", r.Estado)
	}
	if r.Cancelavel {
		t.Fatal("um build cancelado nao e' cancelavel outra vez")
	}
}

// TestStoreBuildCancelarTerminadoRecusa separa o cancelamento honesto do falso.
//
//	*Um cancelamento que devolve sucesso sobre um build ja' terminado e' um botao que mente sobre o
//	que fez -- e o utilizador so' descobre quando pergunta pelo resultado.*
func TestStoreBuildCancelarTerminadoRecusa(t *testing.T) {
	casos := []struct {
		nome     string
		terminar func(s *BuildStore, id string)
	}{
		{"concluido", func(s *BuildStore, id string) {
			_ = s.Terminar(id, &ResultadoDeBuild{Artefatos: []Artefato{{Nome: "x"}}}, nil, "")
		}},
		{"falhou", func(s *BuildStore, id string) {
			_ = s.Terminar(id, nil, errors.New("go build falhou"), "")
		}},
		{"cancelado", func(s *BuildStore, id string) {
			_ = s.Cancelar(id)
		}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			s := NovoBuildStore()
			id, _ := s.Registar("windows", "/r", "/d")
			_, _ = s.Iniciar(id)
			c.terminar(s, id)
			err := s.Cancelar(id)
			if !errors.Is(err, ErrBuildJaTerminou) {
				t.Fatalf("cancelar build %s devia dar ErrBuildJaTerminou, deu %v", c.nome, err)
			}
		})
	}
}

// TestStoreBuildTerminarDepoisDeCanceladoNaoDesCancela e' o teste mais forte deste ficheiro.
//
// CENARIO REAL: o utilizador pede cancelamento; o processo `go build` demora a morrer; o provider
// acaba e chama `Terminar` com sucesso. Se o store aceitasse, o estado final seria CONCLUIDO.
//
//	*Um build que o utilizador cancelou e que depois se declara CONCLUIDO e' um cancelamento que
//	mentiu, e o historico passa a contar a versao de quem acabou por ultimo.*
func TestStoreBuildTerminarDepoisDeCanceladoNaoDesCancela(t *testing.T) {
	s := NovoBuildStore()
	id, _ := s.Registar("windows", "/r", "/d")
	_, _ = s.Iniciar(id)
	if err := s.Cancelar(id); err != nil {
		t.Fatalf("cancelar: %v", err)
	}
	// O provider acaba DEPOIS do cancelamento, com sucesso.
	if err := s.Terminar(id, &ResultadoDeBuild{Explicacao: "acabou apesar do cancelamento"}, nil, "saida"); err != nil {
		t.Fatalf("terminar: %v", err)
	}
	r, _ := s.Obter(id)
	if r.Estado != BuildCancelado {
		t.Fatalf("o estado TEM de continuar CANCELADO; ficou %q -- o Terminar des-cancelou", r.Estado)
	}
	// O resultado nao se perde: fica guardado para diagnostico.
	if r.Resultado == nil {
		t.Fatal("o resultado devia ficar guardado para diagnostico, mesmo cancelado")
	}
	if r.SaidaDoComando != "saida" {
		t.Fatalf("a saida devia ficar guardada: %q", r.SaidaDoComando)
	}
}

// TestStoreBuildTerminarComErroFicaFALHOU e' a transicao de falha.
func TestStoreBuildTerminarComErroFicaFALHOU(t *testing.T) {
	s := NovoBuildStore()
	id, _ := s.Registar("windows", "/r", "/d")
	_, _ = s.Iniciar(id)
	if err := s.Terminar(id, nil, errors.New("iscc nao encontrado"), "erro no iscc"); err != nil {
		t.Fatalf("terminar: %v", err)
	}
	r, _ := s.Obter(id)
	if r.Estado != BuildFalhou {
		t.Fatalf("estado devia ser FALHOU: %q", r.Estado)
	}
	if !strings.Contains(r.Erro, "iscc") {
		t.Fatalf("o erro devia ficar guardado: %q", r.Erro)
	}
	if r.TerminadoEm == nil {
		t.Fatal("um build terminado tem de ter terminado_em")
	}
	if r.Cancelavel {
		t.Fatal("um build terminado nao e' cancelavel")
	}
}

// TestStoreBuildSaidaTruncada impede que um build ruidoso consuma memoria sem tecto.
//
//	*Uma saida sem limite e' uma fuga de memoria com autorizacao do compilador.*
func TestStoreBuildSaidaTruncada(t *testing.T) {
	s := NovoBuildStore()
	id, _ := s.Registar("windows", "/r", "/d")
	_, _ = s.Iniciar(id)
	enorme := strings.Repeat("A", maxSaidaGuardada+5000)
	_ = s.Terminar(id, nil, nil, enorme)
	r, _ := s.Obter(id)
	if len(r.SaidaDoComando) >= len(enorme) {
		t.Fatalf("a saida devia ser truncada; guardou %d de %d", len(r.SaidaDoComando), len(enorme))
	}
	if !strings.Contains(r.SaidaDoComando, "truncada") {
		t.Fatal("a saida truncada tem de DIZER que foi truncada -- um corte silencioso esconde o resto")
	}
}

// TestStoreBuildDesconhecido devolve erro em vez de zero-value.
//
//	*Um store que devolve um registo vazio para um ID inexistente e' um store que inventa builds.*
func TestStoreBuildDesconhecido(t *testing.T) {
	s := NovoBuildStore()
	if _, err := s.Obter("nao-existe"); !errors.Is(err, ErrBuildDesconhecido) {
		t.Fatalf("Obter devia dar ErrBuildDesconhecido, deu %v", err)
	}
	if err := s.Cancelar("nao-existe"); !errors.Is(err, ErrBuildDesconhecido) {
		t.Fatalf("Cancelar devia dar ErrBuildDesconhecido, deu %v", err)
	}
	if _, err := s.Iniciar("nao-existe"); !errors.Is(err, ErrBuildDesconhecido) {
		t.Fatalf("Iniciar devia dar ErrBuildDesconhecido, deu %v", err)
	}
	if err := s.Terminar("nao-existe", nil, nil, ""); !errors.Is(err, ErrBuildDesconhecido) {
		t.Fatalf("Terminar devia dar ErrBuildDesconhecido, deu %v", err)
	}
}

// TestStoreBuildListarMaisRecentesPrimeiro prova a ordenacao.
func TestStoreBuildListarMaisRecentesPrimeiro(t *testing.T) {
	s := NovoBuildStore()
	agora := time.Unix(1000, 0)
	s.agora = func() time.Time { return agora }
	id1, _ := s.Registar("a", "/r", "/d")
	agora = time.Unix(2000, 0)
	id2, _ := s.Registar("b", "/r", "/d")
	agora = time.Unix(1500, 0)
	// Forca a ordem por IniciadoEm: o id2 e' o mais recente.
	l := s.Listar()
	if len(l) != 2 {
		t.Fatalf("esperados 2: %d", len(l))
	}
	if l[0].ID != id2 || l[1].ID != id1 {
		t.Fatalf("ordem errada: primeiro %q (quer %q), segundo %q", l[0].ID, id2, l[1].ID)
	}
}

// TestStoreBuildObterDevolveCopia impede que um chamador mute o estado interno.
//
//	*Um store que devolve o seu proprio ponteiro nao tem estado: tem um boato.*
func TestStoreBuildObterDevolveCopia(t *testing.T) {
	s := NovoBuildStore()
	id, _ := s.Registar("windows", "/r", "/d")
	r, _ := s.Obter(id)
	r.Estado = BuildConcluido
	r.Alvo = "MUTADO"
	r2, _ := s.Obter(id)
	if r2.Estado != BuildNaFila {
		t.Fatalf("mutar a copia mudou o store: estado=%q", r2.Estado)
	}
	if r2.Alvo != "windows" {
		t.Fatalf("mutar a copia mudou o alvo: %q", r2.Alvo)
	}
}

// TestStoreBuildPersisteEmDisco prova a escrita de diagnostico, quando ha' directorio.
func TestStoreBuildPersisteEmDisco(t *testing.T) {
	dir := t.TempDir()
	s := NovoBuildStoreComDirectorio(dir)
	id, _ := s.Registar("windows", "/r", "/d")
	_, _ = s.Iniciar(id)
	if err := s.Terminar(id, &ResultadoDeBuild{Explicacao: "ok"}, nil, "saida-limpa"); err != nil {
		t.Fatalf("terminar: %v", err)
	}
	corpo, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		t.Fatalf("o registo devia estar em disco: %v", err)
	}
	if !strings.Contains(string(corpo), "CONCLUIDO") {
		t.Fatalf("o JSON em disco devia ter o estado: %s", corpo)
	}
}

// TestStoreBuildPodarNaoDescartaEmCurso prova que o tecto de memoria nunca mata um build a correr.
//
//	*Um tecto que descarta o build que esta' a correr e' um limite que apaga o trabalho para o qual
//	foi pedido.*
func TestStoreBuildPodarNaoDescartaEmCurso(t *testing.T) {
	s := NovoBuildStore()
	// Enche o store com builds terminados.
	for range maxBuildsGuardados + 20 {
		id, _ := s.Registar("linux", "/r", "/d")
		_, _ = s.Iniciar(id)
		_ = s.Terminar(id, nil, nil, "")
	}
	// Um build a correr, registado por ultimo.
	emCurso, _ := s.Registar("windows", "/r", "/d")
	_, _ = s.Iniciar(emCurso)
	if _, err := s.Obter(emCurso); err != nil {
		t.Fatalf("o build em curso foi descartado pela poda: %v", err)
	}
	if n := len(s.Listar()); n > maxBuildsGuardados {
		t.Fatalf("o store devia ficar dentro do tecto %d, tem %d", maxBuildsGuardados, n)
	}
}

// TestStoreBuildCarregarRecuperaTerminados prova que um build CONCLUIDO volta do disco.
//
//	*Um store que existe e nao persiste e' um store que perde o trabalho quando o processo reinicia.*
func TestStoreBuildCarregarRecuperaTerminados(t *testing.T) {
	dir := t.TempDir()

	// Primeiro processo: cria, corre, conclui.
	s1 := NovoBuildStoreComDirectorio(dir)
	id, _ := s1.Registar("windows", "/r", "/d")
	_, _ = s1.Iniciar(id)
	if err := s1.Terminar(id, &ResultadoDeBuild{Explicacao: "ok"}, nil, "saida-limpa"); err != nil {
		t.Fatalf("terminar: %v", err)
	}

	// Segundo processo: mesmo directorio, store novo.
	s2 := NovoBuildStoreComDirectorio(dir)
	carregados, interrompidos := s2.CarregarDoDisco()
	if carregados != 1 {
		t.Fatalf("devia carregar 1 registo, carregou %d", carregados)
	}
	if interrompidos != 0 {
		t.Fatalf("nenhum devia estar interrompido: %d", interrompidos)
	}
	r, err := s2.Obter(id)
	if err != nil {
		t.Fatalf("o build nao foi recuperado: %v", err)
	}
	if r.Estado != BuildConcluido {
		t.Fatalf("estado devia ser CONCLUIDO: %q", r.Estado)
	}
	if r.SaidaDoComando != "saida-limpa" {
		t.Fatalf("a saida devia ser recuperada: %q", r.SaidaDoComando)
	}
}

// TestStoreBuildCarregarMarcaInterrompidoComoFALHOU e' o teste mais forte deste ficheiro.
//
// CENARIO REAL: o processo morre com um build A_EXECUTAR. No arranque seguinte, o registo em disco
// diz `A_EXECUTAR`. Se o store o carregasse assim, o painel mostraria "em curso" um build cujo
// processo ja' nao existe.
//
//	*Um build que o processo abandonou nao esta' a correr: esta' interrompido, e o estado tem de o
//	dizer.*
func TestStoreBuildCarregarMarcaInterrompidoComoFALHOU(t *testing.T) {
	dir := t.TempDir()

	s1 := NovoBuildStoreComDirectorio(dir)
	id, _ := s1.Registar("windows", "/r", "/d")
	if _, err := s1.Iniciar(id); err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	// Simula a morte do processo: o registo em disco fica em A_EXECUTAR (o `persistir` do Iniciar
	// nao e' chamado, por isso escrevemos o estado a mao, como ficaria).
	r0, _ := s1.Obter(id)
	r0.Estado = BuildAExecutar
	s1.mu.Lock()
	s1.persistir(&r0)
	s1.mu.Unlock()

	// Segundo processo.
	s2 := NovoBuildStoreComDirectorio(dir)
	carregados, interrompidos := s2.CarregarDoDisco()
	if carregados != 1 || interrompidos != 1 {
		t.Fatalf("esperado 1 carregado e 1 interrompido, veio %d/%d", carregados, interrompidos)
	}
	r, _ := s2.Obter(id)
	if r.Estado == BuildAExecutar || r.Estado == BuildNaFila {
		t.Fatalf("um build abandonado NAO pode ficar a correr: estado=%q", r.Estado)
	}
	if r.Estado != BuildFalhou {
		t.Fatalf("estado devia ser FALHOU: %q", r.Estado)
	}
	if !strings.Contains(r.Erro, "interrompido") {
		t.Fatalf("o motivo tem de dizer que foi interrompido: %q", r.Erro)
	}
	if r.Cancelavel {
		t.Fatal("um build interrompido nao e' cancelavel")
	}
	if r.TerminadoEm == nil {
		t.Fatal("um build interrompido tem de ter terminado_em")
	}
}

// TestStoreBuildCarregarIgnoraCorrompido prova que um ficheiro ilegivel nao impede o arranque.
//
//	*Um ficheiro corrompido nao pode derrubar o servidor, mas tambem nao pode ser contado como build.*
func TestStoreBuildCarregarIgnoraCorrompido(t *testing.T) {
	dir := t.TempDir()

	s1 := NovoBuildStoreComDirectorio(dir)
	id, _ := s1.Registar("windows", "/r", "/d")
	_, _ = s1.Iniciar(id)
	_ = s1.Terminar(id, nil, nil, "")

	// Um ficheiro que nao e' JSON.
	if err := os.WriteFile(filepath.Join(dir, "lixo.json"), []byte("{isto nao e json"), 0o644); err != nil {
		t.Fatalf("escrever lixo: %v", err)
	}
	// Um .json sem ID.
	if err := os.WriteFile(filepath.Join(dir, "semid.json"), []byte(`{"alvo":"x"}`), 0o644); err != nil {
		t.Fatalf("escrever semid: %v", err)
	}

	s2 := NovoBuildStoreComDirectorio(dir)
	carregados, _ := s2.CarregarDoDisco()
	if carregados != 1 {
		t.Fatalf("devia carregar so' o valido (1), carregou %d", carregados)
	}
	if _, err := s2.Obter(id); err != nil {
		t.Fatalf("o build valido devia estar la': %v", err)
	}
}

// TestStoreBuildCarregarSemDirectorioNaoFalha prova que um store em memoria diz que nao carrega nada.
func TestStoreBuildCarregarSemDirectorioNaoFalha(t *testing.T) {
	s := NovoBuildStore()
	c, i := s.CarregarDoDisco()
	if c != 0 || i != 0 {
		t.Fatalf("sem directorio devia dar 0/0, deu %d/%d", c, i)
	}
	if s.Directorio() != "" {
		t.Fatalf("Directorio devia ser vazio: %q", s.Directorio())
	}
}

// TestStoreBuildPodarApagaDoDisco prova que o tecto vale nos DOIS sitios.
//
// A primeira versao podava so' o mapa em memoria e deixava os ficheiros em disco a crescer.
//
//	*Uma politica de retencao que so' conta os registos que caberam em memoria nao e' uma politica de
//	retencao: e' uma politica de memoria.*
func TestStoreBuildPodarApagaDoDisco(t *testing.T) {
	dir := t.TempDir()
	s := NovoBuildStoreComDirectorio(dir)

	// Enche o store acima do tecto. Cada `Registar` chama `podar`, que agora tambem apaga do disco.
	for range maxBuildsGuardados + 15 {
		id, err := s.Registar("linux", "/r", "/d")
		if err != nil {
			t.Fatalf("registar: %v", err)
		}
		if _, err := s.Iniciar(id); err != nil {
			t.Fatalf("iniciar: %v", err)
		}
		if err := s.Terminar(id, nil, nil, ""); err != nil {
			t.Fatalf("terminar: %v", err)
		}
	}

	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ler dir: %v", err)
	}
	if len(entradas) > maxBuildsGuardados {
		t.Fatalf("o disco devia ficar dentro do tecto %d, tem %d ficheiros", maxBuildsGuardados, len(entradas))
	}
	if len(entradas) == 0 {
		t.Fatal("o disco ficou vazio -- a poda levou tudo")
	}
	if n := len(s.Listar()); n > maxBuildsGuardados {
		t.Fatalf("a memoria devia ficar dentro do tecto %d, tem %d", maxBuildsGuardados, n)
	}
}

// TestStoreBuildPodarDiscoFuncionaSemBuildsNovos prova a poda explicita.
//
// O `podar` so' corre quando um build NOVO entra: um processo que arranca, carrega 200 registos e fica
// parado nunca poda nada. `PodarDisco` existe para esse caso.
func TestStoreBuildPodarDiscoFuncionaSemBuildsNovos(t *testing.T) {
	dir := t.TempDir()

	// Escreve 12 ficheiros a mao, sem passar pelo store (simula um disco ja' cheio de outra sessao).
	for i := range 12 {
		nome := "a" + itoaDoStore(i) + ".json"
		corpo := `{"id":"a` + itoaDoStore(i) + `","alvo":"x","estado":"CONCLUIDO","iniciado_em":"2026-01-01T00:00:00Z"}`
		if err := os.WriteFile(filepath.Join(dir, nome), []byte(corpo), 0o644); err != nil {
			t.Fatalf("escrever %d: %v", i, err)
		}
	}

	s := NovoBuildStoreComDirectorio(dir)
	// Sem tecto excedido (12 < 200): nao remove nada.
	if n := s.PodarDisco(); n != 0 {
		t.Fatalf("com 12 ficheiros nao devia remover nada, removeu %d", n)
	}
}

// TestStoreBuildPodarDiscoNaoRemoveEmCurso e' o discriminante.
//
//	*Uma poda que remove o build que esta' a correr e' um limite que apaga o trabalho para o qual foi
//	pedido.*
func TestStoreBuildPodarDiscoNaoRemoveEmCurso(t *testing.T) {
	dir := t.TempDir()

	// Cria um build EM CURSO e persiste-o.
	s1 := NovoBuildStoreComDirectorio(dir)
	emCurso, _ := s1.Registar("windows", "/r", "/d")
	if _, err := s1.Iniciar(emCurso); err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	// Forca a persistencia do estado A_EXECUTAR.
	r, _ := s1.Obter(emCurso)
	s1.mu.Lock()
	s1.persistir(&r)
	s1.mu.Unlock()

	// Enche o disco com ficheiros antigos (mais recentes que o em-curso nao importa: a poda usa
	// ModTime e o em-curso e' o mais recente).
	for i := range maxBuildsGuardados + 10 {
		corpo := `{"id":"b` + itoaDoStore(i) + `","alvo":"x","estado":"CONCLUIDO","iniciado_em":"2026-01-01T00:00:00Z"}`
		if err := os.WriteFile(filepath.Join(dir, "b"+itoaDoStore(i)+".json"), []byte(corpo), 0o644); err != nil {
			t.Fatalf("escrever %d: %v", i, err)
		}
	}

	// Um store NOVO carrega tudo; o build em curso no processo ANTERIOR nao esta' em `registos` como
	// A_EXECUTAR (foi marcado interrompido). Para exercitar a poda contra um build vivo, usamos o s1.
	s1.mu.Lock()
	s1.persistir(&r)
	s1.mu.Unlock()

	removidos := s1.PodarDisco()
	if removidos == 0 {
		t.Fatal("a poda devia ter removido ficheiros antigos")
	}
	// O build em curso continua em memoria.
	if _, err := s1.Obter(emCurso); err != nil {
		t.Fatalf("o build em curso foi removido da memoria: %v", err)
	}
	// E o ficheiro dele continua em disco.
	if _, err := os.Stat(filepath.Join(dir, emCurso+".json")); err != nil {
		t.Fatalf("o ficheiro do build em curso foi apagado: %v", err)
	}
}

// TestStoreBuildCancelarDuasVezesNaoDisparaDoisCancelamentos prova idempotencia do pedido.
func TestStoreBuildCancelarDuasVezesNaoDisparaDoisCancelamentos(t *testing.T) {
	s := NovoBuildStore()
	id, _ := s.Registar("windows", "/r", "/d")
	ctx, _ := s.Iniciar(id)
	if err := s.Cancelar(id); err != nil {
		t.Fatalf("primeiro cancelar: %v", err)
	}
	if err := s.Cancelar(id); !errors.Is(err, ErrBuildJaTerminou) {
		t.Fatalf("segundo cancelar devia dar ErrBuildJaTerminou, deu %v", err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("contexto nao cancelado")
	}
	_ = context.Background()
}
