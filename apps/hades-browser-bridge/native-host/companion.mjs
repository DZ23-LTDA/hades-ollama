#!/usr/bin/env node
/**
 * HADES Local Companion -- PROCESSO REAL (`E-1029`).
 *
 * O QUE ISTO PROVA E O QUE NAO PROVA:
 *   Prova: o transporte do `E-1027` corre contra `process.stdin`/`process.stdout`
 *   de um PROCESSO REAL, arrancado como subprocesso, e o handshake do `E-1025`
 *   acontece pelo fio. Nao e' memoria nem mock.
 *   NAO prova: instalacao, `allowed_origins` em uso, permissao do SO, nem
 *   assinatura. E' um host **nao instalado e nao assinado**.
 *
 * REGRA DE OURO: o `stdout` e' o FIO. Nada pode ser escrito nele que nao seja uma
 * mensagem enquadrada -- um `console.log` perdido dentro do protocolo corrompe o
 * fluxo do outro lado. Por isso todo diagnostico vai para o `stderr`.
 *
 * Este host implementa SO' o pairing e o handshake. Nao executa ferramenta, nao
 * toca no SO e nao tem shell: entregar o handshake e' o criterio desta fatia.
 */
import { enquadrar, desenquadrar, ErroDeTransporte, MAX_CORPO_BYTES } from "../src/transporte.js";
import * as C from "../src/canal_local.js";

/** Versao do protocolo do host, separada da versao da extensao. */
export const HOST_PROTOCOLO = 1;

/** Capacidades declaradas. Vazio e' honesto: este host nao controla o PC. */
export const CAPACIDADES = [];

/**
 * PORQUE A CONSULTA DE APROVACAO NAO VERIFICA NADA (`E-1043`).
 *
 * Nao e' uma limitacao esquecida: e' uma limitacao MEDIDA. A rota de approvals do
 * runtime e' autenticada por **sessao**; este processo nao tem sessao e nao deve ter.
 * *Um processo local com a sessao do utilizador e' exactamente o que nao queremos.*
 *
 * O motivo viaja na RESPOSTA, nao num documento: quem pergunta recebe a razao.
 */
export const MOTIVO_SEM_ROTA = "sem_rota_autenticavel_no_runtime";

/** Diagnostico vai para stderr, NUNCA para o fio. */
function diag(...partes) {
  process.stderr.write("[companion] " + partes.join(" ") + "\n");
}

/**
 * O estado do host. `pairing` e `dispositivos` vivem AQUI, em memoria: reiniciar o
 * processo apaga tudo. E' uma limitacao declarada, nao um detalhe.
 */
export function criarHost(agora = () => Date.now()) {
  const estado = C.estadoInicial();
  let ultimoPedido = null;
  return {
    estado,
    /** Trata uma mensagem ja' desenquadrada e devolve a resposta (ou null). */
    async tratar(m) {
      if (!m || typeof m !== "object") return erro("forma_invalida");
      const t = agora();
      switch (m.cmd) {
        case "host:ola":
          return {
            ok: true, hostProtocolo: HOST_PROTOCOLO, protocolo: C.PROTOCOLO,
            capacidades: CAPACIDADES, instalado: false, assinado: false,
          };

        case "pairing:iniciar": {
          const p = C.abrirPairing(t);
          ultimoPedido = p;
          return { ok: true, pedido: p.pedido, codigo: p.codigo, expiraEm: p.expiraEm };
        }

        case "pairing:confirmar": {
          if (!ultimoPedido) return erro("sem_pedido");
          // ***`E-1118`: VALIDAR OS INPUTS ANTES DE CONSUMIR O CODIGO.*** *A ordem antiga era
          // `confirmarPedido` -> `consumirCodigo` -> validar a `chave`. Logo uma chamada **sem chave**
          // consumia o codigo e so' depois era recusada:* *o utilizador legitimo ficava com o codigo
          // queimado por uma tentativa que nunca podia ter sucesso -- um **DoS trivial**, porque
          // qualquer processo que fale com o host consome o codigo de um pairing em curso.*
          // => *o codigo de 6 digitos e' **o recurso escasso**: so' se gasta quando a troca esta'
          // completa.*
          const bytesCru = Array.isArray(m.chave) ? m.chave : null;
          if (!bytesCru || bytesCru.length !== 32) return erro("chave_ausente");
          const r = C.confirmarPairing(ultimoPedido, m.codigo, t);
          if (!r.ok) return erro(r.motivo);
          const c = C.consumirCodigo(ultimoPedido);
          if (!c.ok) return erro(c.motivo);
          // A chave vem da EXTENSAO: o host nao a inventa nem a deriva sozinho.
          const bytes = new Uint8Array(bytesCru);
          const inst = m.instalacao ?? r.pedido;
          const reg = C.registrarDispositivo(estado, inst, bytes, t);
          if (!reg.ok) return erro(reg.motivo);
          return { ok: true, instalacao: inst };
        }

        case "handshake:desafio": {
          const d = C.abrirDesafio(m.instalacao, t);
          if (!d.ok) return erro(d.motivo);
          estado.desafios[d.desafio.nonce] = d.desafio;
          return { ok: true, nonce: d.desafio.nonce, expiraEm: d.desafio.expiraEm };
        }

        case "handshake:aceitar": {
          const r = await C.aceitarHandshake(
            { instalacao: m.instalacao, nonce: m.nonce, prova: m.prova }, estado, t,
          );
          return r.ok ? { ok: true, instalacao: r.instalacao, missao: r.missao } : erro(r.motivo);
        }

        case "envelope:aceitar": {
          const r = await C.aceitar(m.envelope, estado, t);
          return r.ok ? { ok: true, instalacao: r.instalacao, corpo: r.corpo } : erro(r.motivo);
        }

        case "revogar": {
          const r = C.revogarDispositivo(estado, m.instalacao);
          return r.ok ? { ok: true } : erro(r.motivo);
        }

        // `E-1043`: A ROTA DE APROVACAO -- o lado que E' executavel.
        //
        // No `E-1042` medi que a rota HTTP de approvals (`serviceAgentApprovals`) e'
        // autenticada por **sessao**, e o Companion e' um processo local **sem sessao**.
        // *Dar-lhe a sessao do utilizador seria pior: um processo local com a sessao do
        // utilizador e' exactamente o que nao queremos.*
        //
        // Entao o que existe AQUI e' o contrato e a recusa honesta. O Companion responde
        // sempre -- e responde `verificada: false` com o motivo **exacto**. *Uma rota que
        // responde «nao posso» e' melhor do que uma rota que nao existe: quem pergunta fica a
        // saber que perguntou ao sitio certo.*
        //
        // O que falta para `verificada: true` e' **uma linha no runtime**: expor
        // `FindApproved` num canal que o Companion possa autenticar com o par emparelhado.
        // Naquele dia, muda-se `MOTIVO_SEM_ROTA` para uma chamada -- e nada mais.
        case "aprovacao:consultar": {
          // Sem par emparelhado nao ha' sequer direito de perguntar.
          const pareados = Object.keys(estado.dispositivos ?? {});
          if (pareados.length === 0) return erro("sem_dispositivo_pareado");
          const fingerprint = String(m.fingerprint ?? "");
          if (!/^[a-f0-9]{64}$/.test(fingerprint)) return erro("fingerprint_invalido");
          if (typeof m.instalacao !== "string" || !pareados.includes(m.instalacao)) {
            return erro("instalacao_desconhecida");
          }
          // ***`E-1118`: UM DISPOSITIVO REVOGADO NAO TEM DIREITO A PERGUNTAR.*** *O `revogarDispositivo`
          // **marca** `revogada: true` mas **nao remove** de `estado.dispositivos` -- logo este
          // `includes` passava, e o dispositivo revogado continuava a receber uma resposta.* => *e a
          // inconsistencia era visivel: o `aceitarHandshake` e o `rodarChave` **verificam** `revogada`, e
          // este caminho **nao** -- *«curar dois de tres sitios deixa o terceiro a aceitar o que os outros
          // recusam»* (`E-1035`).*
          if (estado.dispositivos[m.instalacao].revogada === true) {
            return erro("dispositivo_revogado");
          }
          return {
            ok: true,
            cmd: "aprovacao:consultar",
            fingerprint,
            // *A verdade: o Companion NAO consultou o store. Nunca diz que sim.*
            verificada: false,
            motivo: MOTIVO_SEM_ROTA,
            // O passo de desbloqueio, dito a quem pergunta.
            desbloqueio: "expor FindApproved num canal autenticavel pelo par emparelhado",
          };
        }

        default:
          return erro("cmd_desconhecido");
      }
    },
  };
}

function erro(motivo) {
  return { ok: false, motivo };
}

/**
 * O laco do fio. Le' bytes do `entrada`, remonta mensagens com `desenquadrar` e
 * escreve respostas enquadradas no `saida`.
 *
 * Um erro de enquadramento e' FATAL para o fluxo: depois dele nao ha' como saber
 * onde comeca a proxima mensagem. O host responde `erro_de_fluxo` e ENCERRA, em
 * vez de continuar a adivinhar.
 */
export function ligarFio(entrada, saida, host = criarHost()) {
  let pendente = new Uint8Array(0);
  const escrever = (obj) => saida.write(Buffer.from(enquadrar(obj)));
  return {
    host,
    alimentar(bytes) {
      const junto = new Uint8Array(pendente.length + bytes.length);
      junto.set(pendente, 0);
      junto.set(bytes, pendente.length);
      let r;
      try {
        r = desenquadrar(junto);
      } catch (e) {
        const codigo = e instanceof ErroDeTransporte ? e.codigo : "erro_desconhecido";
        diag("erro de fluxo:", codigo, e.message);
        escrever({ ok: false, motivo: "erro_de_fluxo", codigo });
        return false; // o fluxo morreu
      }
      pendente = r.resto;
      for (const m of r.mensagens) {
        // O `id` TEM de ser ecoado (`E-1030`): o cliente correlaciona respostas por
        // ele. Um host que responde sem `id` deixa o pedido pendente para sempre --
        // e o defeito nao aparece como erro, aparece como SILENCIO.
        const ecoar = (resp) => {
          if (m && typeof m === "object" && "id" in m) resp.id = m.id;
          escrever(resp);
        };
        host.tratar(m).then(ecoar).catch((e) => {
          diag("erro ao tratar:", e.message);
          ecoar({ ok: false, motivo: "erro_interno" });
        });
      }
      return true;
    },
    pendentes: () => pendente.length,
  };
}

/* Somente quando executado como script -- nunca ao ser importado por um teste. */
const ehPrincipal = process.argv[1] && import.meta.url.endsWith(
  process.argv[1].replace(/\\/g, "/").split("/").pop(),
);
if (ehPrincipal) {
  diag("a arrancar (protocolo " + HOST_PROTOCOLO + ", capacidades: " +
    (CAPACIDADES.length === 0 ? "NENHUMA" : CAPACIDADES.join(",")) + ")");
  const fio = ligarFio(process.stdin, process.stdout);
  process.stdin.on("data", (b) => {
    if (!fio.alimentar(new Uint8Array(b))) process.exit(2);
  });
  process.stdin.on("end", () => {
    // Fim de fluxo com bytes a meio e' um cliente que morreu a meio de uma mensagem.
    if (fio.pendentes() > 0) {
      diag("fim de fluxo com", fio.pendentes(), "bytes pendentes");
      process.exitCode = 3;
    }
  });
}
