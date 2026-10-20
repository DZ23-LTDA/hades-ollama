package server

import "github.com/ollama/ollama/internal/agent"

// pareDoServidor e' o travao consultado pelos executores HTTP.
//
// # PORQUE UMA INSTANCIA DO SERVIDOR, E NAO UMA POR PEDIDO
//
// Um pare que nasce em cada pedido nunca esta' acionado -- nao e' um travao, e' uma variavel local com
// nome de travao. Para parar execucoes, o estado tem de sobreviver ao pedido.
//
// # O QUE AINDA FALTA, DITO AQUI
//
// Nao ha' rota para O ACIONAR. Ou seja: o executor HTTP ja' **consulta** o pare (deixou de ser inerte
// no codigo), mas ninguem o consegue accionar em producao. Expor o accionamento e' uma decisao de
// superficie (quem pode parar tudo? com que autorizacao?) que esta fatia NAO toma.
var pareDoServidor = agent.NovoPareDeEmergencia()
