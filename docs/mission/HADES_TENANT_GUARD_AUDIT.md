# Auditoria de guarda de tenant nos handlers HTTP

## 1. O método aplicado a outro eixo

A varredura anterior (E-090) fechou a família de **caminhos irmãos** no escopo por
**ator**. Esta rodada aplica o mesmo raciocínio ao escopo por **organização**: todo
handler HTTP que toca o store precisa de guarda de tenant **antes** de servir dado.

Ferramenta reexecutável: `scripts/audit_tenant_guards.py`.

## 2. A calibração (e por que ela importa)

A primeira versão do detector, com seis guardas, acusou **49 handlers**. Quase tudo
era **falso positivo**: o código usa helpers que o detector não conhecia.

| Versão | Guardas reconhecidas | Candidatos |
| --- | --- | --- |
| 1 | 6 nomes literais | **49** |
| 2 (calibrada) | + `require*`, `*ForRequest`, `*OrganizationID(c)`, `missionByID`, `scopedRuntime(` | **14** |
| 3 (explicados) | + as 14 exceções documentadas | **0 a revisar** |

Ler os casos antes de acusar **não** é zelo excessivo: um relatório com 49 achados
falsos custa a confiança que os achados verdadeiros precisam. E, ao contrário, um
detector que **omite** guarda por não conhecê-la produz o pior resultado possível —
silêncio parecendo segurança.

## 3. Resultado medido

`python scripts/audit_tenant_guards.py` → **136** handlers tocam o store com guarda
reconhecida; **14** não têm guarda de helper, e **todos os 14 estão explicados no
próprio código**:

| Handler | Por que não é achado |
| --- | --- |
| `health`, `metrics`, `prometheus` | diagnóstico/observabilidade, sem dado de tenant na resposta |
| `tools` | lista estática de ferramentas do runtime |
| `startDevicePairing`, `completeDevicePairing` | pareamento de dispositivo: fluxo público por desenho |
| `mediaImage`, `mediaVideo`, `mediaSpeech`, `mediaTranscribe`, `mediaVision` | superfícies de compatibilidade de mídia no runtime único; não servem dado de outro tenant |
| `downloadBackup` | **já recusado** com `if a.authRequired`: o comentário no código declara que o backup contém todas as organizações e que não há super-admin cruzado, então em modo autenticado devolve **403** |
| `whatsappWebhook`, `whatsappWebhookVerify` | públicos por desenho: assinatura **HMAC** verificada (401 em assinatura inválida) e verificação de desafio por token |

## 4. Veredito

**Nenhum defeito novo de tenant nesta dimensão.** É um resultado negativo — e
medido, não presumido. O valor está em três coisas:

1. a ferramenta fica no repositório para rodar de novo a cada mudança de rota;
2. as exceções estão **nomeadas**, então um handler novo que apareça na lista
   acende revisão em vez de se perder no ruído;
3. a calibração mostra o custo real de um detector ingênuo: **49 falsos positivos**
   na primeira passada.
