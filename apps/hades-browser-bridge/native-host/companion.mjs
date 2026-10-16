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
          const r = C.confirmarPairing(ultimoPedido, m.codigo, t);
          if (!r.ok) return erro(r.motivo);
          const c = C.consumirCodigo(ultimoPedido);
          if (!c.ok) return erro(c.motivo);
          // A chave vem da EXTENSAO: o host nao a inventa nem a deriva sozinho.
          if (!Array.isArray(m.chave) || m.chave.length !== 32) return erro("chave_ausente");
          const bytes = new Uint8Array(m.chave);
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
