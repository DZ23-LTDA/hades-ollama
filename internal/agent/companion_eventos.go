package agent

import "sort"

// ConversorEventosDoCompanion liga os eventos REAIS de uma missao ao que o Companion
// pode dizer (complemento 48 / item 46). Antes deste ficheiro, `CompanionEventoVerificado`
// nao tinha produtor nenhum: o tipo e a derivacao existiam e estavam testados, mas nada
// no sistema os alimentava. *Era a classe de defeito que esta missao persegue: capacidade
// real, sem elo entre os dados e quem os usa.*
//
// TRES REGRAS, e nenhuma delas e' decorativa:
//
//  1. **Nao inventa detalhe.** `Event.Payload` e' `any` e NAO tem sanitizacao garantida
//     neste caminho. Despeja-lo para a UI seria levar dados nao filtrados para fora (DLP).
//     Por isso `Detalhe` sai VAZIO por omissao e so' e' preenchido a partir de
//     `detalhesSanitizados` - um mapa que o chamador tem de fornecer de forma EXPLICITA,
//     depois de sanitizar. *Melhor um detalhe ausente do que um detalhe vazado.*
//
//  2. **So' eventos que aconteceram.** Nao ha' sintese de estado de intencao: cada
//     `CompanionEventoVerificado` sai de um `Event` persistido, com o `Quando` real.
//     A ordenacao temporal existe para que a derivacao nao trate um evento futuro como
//     passado (o nucleo em `companion_estado.go` conta com isso).
//
//  3. **Determinismo.** Ordenacao estavel por `Quando`, com desempate por `PassoID` e
//     depois por `Tipo`: dois eventos no mesmo instante nao podem produzir estados
//     diferentes entre execucoes.
//
//  4. **Respeita o vocabulario fechado.** ***Defeito real que este filtro corrige***:
//     `Runtime.CreateMission` emite `mission.planned`, que **nao** pertence a
//     `EventosVerificadosDeMissao`. Sem filtrar, o conversor entregava ao nucleo um
//     evento que ele declara recusar - e a rota respondia **400 para TODA missao
//     recem-criada**. *Um adaptador que entrega a uma funcao o que ela declara rejeitar
//     nao e' um adaptador: e' um erro a espera de acontecer.* Os eventos fora do
//     vocabulario sao DESCARTADOS aqui - e o descarte e' contado por
//     `ConversorEventosDoCompanionComRelatorio`, para nao ser silencioso.
func ConversorEventosDoCompanion(eventos []Event, detalhesSanitizados map[string]string) []CompanionEventoVerificado {
	convertidos, _ := ConversorEventosDoCompanionComRelatorio(eventos, detalhesSanitizados)
	return convertidos
}

// CompanionEventoEhVerificado diz se um tipo pertence ao vocabulario fechado do nucleo.
// *Reutiliza a lista exportada por `companion_estado.go` - nao mantem uma copia propria,
// que envelheceria em silencio.*
func CompanionEventoEhVerificado(tipo string) bool {
	for _, conhecido := range EventosVerificadosDeMissao {
		if conhecido == tipo {
			return true
		}
	}
	return false
}

// ConversorEventosDoCompanionComRelatorio faz o mesmo que `ConversorEventosDoCompanion`
// e devolve tambem QUANTOS eventos foram descartados por estarem fora do vocabulario.
// *O numero existe porque um descarte invisivel e' indistinguivel de uma perda de dados.*
func ConversorEventosDoCompanionComRelatorio(eventos []Event, detalhesSanitizados map[string]string) ([]CompanionEventoVerificado, int) {
	convertidos := make([]CompanionEventoVerificado, 0, len(eventos))
	descartados := 0
	for _, evento := range eventos {
		if !CompanionEventoEhVerificado(evento.Type) {
			descartados++
			continue
		}
		// O detalhe so' entra se o chamador o tiver sanitizado e identificado pelo ID do evento.
		detalhe := ""
		if detalhesSanitizados != nil {
			detalhe = detalhesSanitizados[evento.ID]
		}
		convertidos = append(convertidos, CompanionEventoVerificado{
			Tipo:     evento.Type,
			MissaoID: evento.MissionID,
			PassoID:  evento.StepID,
			Quando:   evento.CreatedAt,
			Detalhe:  detalhe,
		})
	}

	sort.SliceStable(convertidos, func(i, j int) bool {
		return companionEventoAntes(convertidos[i], convertidos[j])
	})
	return convertidos, descartados
}

// companionEventoAntes define a ordem total usada acima. *Extraida para ser testavel
// isoladamente: a ordenacao e' parte do contrato, nao um detalhe de implementacao.*
func companionEventoAntes(a, b CompanionEventoVerificado) bool {
	if !a.Quando.Equal(b.Quando) {
		return a.Quando.Before(b.Quando)
	}
	if a.PassoID != b.PassoID {
		return a.PassoID < b.PassoID
	}
	return a.Tipo < b.Tipo
}

// EventosDoCompanionDaMissao e' o atalho usado pela rota: filtra os eventos de UMA missao
// e converte-os. *Filtrar por missao aqui nao substitui o controlo de acesso - a rota
// continua a ter de provar que a missao pertence ao chamador antes de chamar isto.*
func EventosDoCompanionDaMissao(missaoID string, eventos []Event, detalhesSanitizados map[string]string) []CompanionEventoVerificado {
	daMissao := make([]Event, 0, len(eventos))
	for _, evento := range eventos {
		if evento.MissionID == missaoID {
			daMissao = append(daMissao, evento)
		}
	}
	return ConversorEventosDoCompanion(daMissao, detalhesSanitizados)
}
