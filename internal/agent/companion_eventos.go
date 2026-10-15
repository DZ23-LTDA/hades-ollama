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
func ConversorEventosDoCompanion(eventos []Event, detalhesSanitizados map[string]string) []CompanionEventoVerificado {
	convertidos := make([]CompanionEventoVerificado, 0, len(eventos))
	for _, evento := range eventos {
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
	return convertidos
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
