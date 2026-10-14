import { strict as assert } from "node:assert";
// @ts-expect-error Node's strip-types runner resolves the source extension directly.
// eslint-disable-next-line import/extensions
import { fetchMissionContext, missionContextPath } from "./missionContextClient.ts";

// D-9 / E-612: a busca do contexto — testada SEM rede e SEM React, com `fetch` injectado.

// (1) O CAMINHO: a rota canónica, com o id codificado (um id com barra não pode mudar de rota).
assert.equal(missionContextPath("m-1"), "/api/agent/v1/missions/m-1/context");
assert.equal(missionContextPath("a/b"), "/api/agent/v1/missions/a%2Fb/context");

const BASE = "https://exemplo.test/";

function fetchFalso(resposta: { ok: boolean; status?: number; corpo?: unknown; lanca?: boolean }): typeof fetch {
  return (async () => {
    if (resposta.lanca) throw new Error("sem rede");
    return {
      ok: resposta.ok,
      status: resposta.status ?? 200,
      json: async () => {
        if (resposta.corpo === "json-invalido") throw new Error("JSON inválido");
        return resposta.corpo;
      },
    };
  }) as unknown as typeof fetch;
}

// (2) CAMINHO FELIZ: projecta pela lista branca, e a topologia NÃO passa.
const feliz = await fetchMissionContext(
  "m-1",
  fetchFalso({
    ok: true,
    corpo: {
      objective: "entregar a compactação",
      plan: ["ligar o funil"],
      handoff: "[HANDOFF] Objetivo: entregar",
      mission_id: "mis_96544b01",
      project_id: "prj_411af0cd",
      workspace: "C:\\Users\\DZ23\\_tmp\\ws",
    },
  }),
  BASE,
  "token",
);
assert.equal(feliz.motivo, undefined);
assert.equal(feliz.vista.objective, "entregar a compactação");
assert.deepEqual(feliz.vista.plan, ["ligar o funil"]);
assert.ok(feliz.vista.handoff && feliz.vista.handoff.includes("HANDOFF"));
const serializado = JSON.stringify(feliz.vista);
for (const proibido of ["mis_96544b01", "prj_411af0cd", "_tmp", "workspace"]) {
  assert.equal(serializado.includes(proibido), false, `a busca vazou ${proibido}`);
}

// (3) ***CONTEXTO INDISPONÍVEL NUNCA DERRUBA A SUPERFÍCIE*** — os quatro modos de falha
// devolvem vista vazia COM motivo, e nenhum lança.
const httpErro = await fetchMissionContext("m-1", fetchFalso({ ok: false, status: 403 }), BASE);
assert.deepEqual(httpErro.vista, {});
assert.ok(httpErro.motivo && httpErro.motivo.includes("403"));

const semRede = await fetchMissionContext("m-1", fetchFalso({ ok: true, lanca: true }), BASE);
assert.deepEqual(semRede.vista, {});
assert.equal(semRede.motivo, "sem rede");

const jsonMau = await fetchMissionContext("m-1", fetchFalso({ ok: true, corpo: "json-invalido" }), BASE);
assert.deepEqual(jsonMau.vista, {});
assert.ok(jsonMau.motivo);

const semEscopo = await fetchMissionContext("", fetchFalso({ ok: true }), BASE);
assert.deepEqual(semEscopo.vista, {});
assert.ok(semEscopo.motivo);

const semBase = await fetchMissionContext("m-1", fetchFalso({ ok: true }), "");
assert.deepEqual(semBase.vista, {});
assert.ok(semBase.motivo);

console.log("missionContextClient: busca e projecao OK (sem rede, sem React)");
