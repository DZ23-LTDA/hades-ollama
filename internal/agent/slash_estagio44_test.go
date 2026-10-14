package agent

import (
	"errors"
	"strings"
	"testing"
)

// E-533 (Estagio 44) - os tres comandos que faltavam, e a regra que os governa.
//
// ***O QUE ESTE TESTE PROTEGE, E NAO E' A CONTAGEM.*** Nao basta haver 8 comandos: o Estagio 44 exige que o **Spark
// seja um modo**, e nao um segundo runtime. ⇒ o teste que importa e' o que compara o **backend** de `/spark` e
// `/cowork` com o de `/goal`: *se algum dia alguem lhes der um motor proprio, este teste falha.*

func TestSlashSparkECoworkPartilhamAMesmaEngineDoGoal(t *testing.T) {
	goal, ok := SlashCommandSpecByName("/goal")
	if !ok || !goal.Available {
		t.Fatalf("`/goal` tinha de existir e estar disponível: %+v", goal)
	}

	irmas := []string{"/spark", "/cowork"}
	for _, nome := range irmas {
		spec, ok := SlashCommandSpecByName(nome)
		if !ok {
			// ***A MENSAGEM DIZ O QUE FAZER, NAO SO' O QUE FALHOU.***
			t.Fatalf("ARMADILHA: %s nao esta' no registro - *o Estagio 44 exige os 8 comandos*", nome)
		}
		if !spec.Available {
			t.Fatalf("ARMADILHA: %s esta' indisponivel (%q) - *mas o backend dele e' o mesmo de /goal, que existe*",
				nome, spec.UnavailableReason)
		}
		if spec.Backend != goal.Backend {
			t.Fatalf("ARMADILHA (Estagio 44): %s aponta para %q e /goal para %q - *o Spark tem de ser um MODO sobre a "+
				"MESMA engine; um backend proprio seria o «segundo runtime em conflito» que o complemento proibe*",
				nome, spec.Backend, goal.Backend)
		}
		if spec.Stage != "44" {
			t.Fatalf("%s devia declarar o estagio 44, veio %q", nome, spec.Stage)
		}
	}
}

func TestSlashRejectEOParAuditadoDoApproveEDoDeny(t *testing.T) {
	approve, okA := SlashCommandSpecByName("/approve")
	deny, okD := SlashCommandSpecByName("/deny")
	reject, okR := SlashCommandSpecByName("/reject")
	if !okA || !okD || !okR {
		t.Fatalf("faltam comandos: approve=%v deny=%v reject=%v", okA, okD, okR)
	}
	// *Aprovar, recusar e rejeitar passam pelo MESMO caminho auditado.*
	if reject.Backend != approve.Backend || reject.Backend != deny.Backend {
		t.Fatalf("ARMADILHA: /reject aponta para %q, /approve para %q e /deny para %q - *um par a' solta seria um "+
			"comando que so' sabe dizer «nao», fora da trilha de auditoria*",
			reject.Backend, approve.Backend, deny.Backend)
	}
	if !reject.Available {
		t.Fatalf("/reject ficou indisponivel (%q) mas o backend existe", reject.UnavailableReason)
	}
	// E os dois argumentos obrigatorios, iguais aos do par.
	if len(reject.Args) != len(deny.Args) {
		t.Fatalf("/reject tem %d args e /deny tem %d - *o par tem de ter o mesmo contrato*", len(reject.Args), len(deny.Args))
	}
}

func TestSlashOsOitoDoEstagio44Existem(t *testing.T) {
	// O Estagio 44 nomeia 8 comandos. *Este teste e' a lista literal do complemento, para que a ausencia de um
	// deles falhe com o NOME e nao com uma contagem.*
	oito := []string{"/goal", "/spark", "/cowork", "/status", "/resume", "/cancel", "/approve", "/reject"}
	faltando := make([]string, 0, len(oito))
	for _, nome := range oito {
		if _, ok := SlashCommandSpecByName(nome); !ok {
			faltando = append(faltando, nome)
		}
	}
	if len(faltando) > 0 {
		t.Fatalf("ARMADILHA (Estagio 44): faltam %v dos 8 comandos nomeados pelo complemento", faltando)
	}
}

func TestSlashComandoRecusadoTrazOMotivoENuncaExecuta(t *testing.T) {
	// A doutrina do registro: comando indisponivel e' recusado COM o motivo - *nunca «executado»*.
	indisponivel := SlashCommandSpec{Name: "/exemplo", Available: false, UnavailableReason: "backend ainda nao existe"}
	err := checkSlashCommandAvailable(indisponivel)
	if !errors.Is(err, ErrSlashCommandUnavailable) {
		t.Fatalf("comando indisponivel devolveu %v", err)
	}
	if !strings.Contains(err.Error(), "backend ainda nao existe") {
		t.Fatalf("ARMADILHA: a recusa nao trouxe o motivo: %v - *um botao que nao faz nada e' pior do que um botao "+
			"que diz por que nao faz*", err)
	}
	disponivel := SlashCommandSpec{Name: "/exemplo", Available: true, Backend: "GET /x"}
	if err := checkSlashCommandAvailable(disponivel); err != nil {
		t.Fatalf("comando disponivel recusado: %v", err)
	}
}

func TestSlashSparkAceitaObjetivoEhRejeitadoSemEle(t *testing.T) {
	// ***E AQUI ESTA' UM ACHADO, E NAO UMA CORRECCAO DE CONVENIENCIA.***
	//
	// A primeira versao deste teste passava `"/spark idealizar tres variacoes de landing page"` - e FALHOU, com
	// «aceita no maximo 2 argumentos». A causa e' do PARSER, e e' **anterior a mim**: `ParseSlashInvocation` divide a
	// entrada por `strings.Fields`, logo **cada palavra conta como um argumento posicional**. ⇒ *um objetivo com
	// espaços NAO pode ser passado hoje* - nem para `/spark`, nem para `/goal`, nem para nenhum dos 36.
	//
	// ***NAO "CORRIGI" O PARSER AQUI.*** Mexer-lhe exigiria decidir aspas/escaping e re-medir 36 comandos e os seus
	// testes - e seria uma alteracao de contrato, nao um acrescento. *O que faco e' (a) usar uma palavra no teste, para
	// testar o que me propus, e (b) deixar o limite **escrito num teste proprio**, para que ele nao desapareca.*
	inv, err := ParseSlashInvocation("/spark landingpage")
	if err != nil {
		t.Fatalf("/spark com objetivo devia ser aceito: %v", err)
	}
	if inv.Args["objetivo"] == "" {
		t.Fatalf("o objetivo nao chegou aos args: %+v", inv.Args)
	}
	if _, err := ParseSlashInvocation("/spark"); !errors.Is(err, ErrSlashCommandArgsInvalid) {
		t.Fatalf("ARMADILHA: /spark SEM objetivo devia ser recusado com ErrSlashCommandArgsInvalid, veio %v", err)
	}
}

// TestSlashObjetivoComEspacosNaoEExpressavelHoje **documenta o limite**, em vez de o esconder.
//
// *Nao e' um teste que passa por gosto: e' um teste que falha se alguem corrigir o parser - e nesse dia alguem tem de
// vir aqui e actualizar a expectativa, que e' exactamente o que se quer.*
func TestSlashObjetivoComEspacosNaoEExpressavelHoje(t *testing.T) {
	_, err := ParseSlashInvocation("/spark idealizar tres variacoes")
	if !errors.Is(err, ErrSlashCommandArgsInvalid) {
		t.Fatalf("ARMADILHA: o parser passou a aceitar multiplas palavras (err=%v) - *se isto aconteceu, o limite "+
			"descrito aqui foi resolvido: actualize este teste e o E-533*", err)
	}
	// E o mesmo vale para `/goal`, que e' **anterior** a este acrescento: o limite nao e' do comando novo.
	if _, err := ParseSlashInvocation("/goal construir um relatorio"); !errors.Is(err, ErrSlashCommandArgsInvalid) {
		t.Fatalf("o limite devia valer tambem para /goal: %v", err)
	}
}
