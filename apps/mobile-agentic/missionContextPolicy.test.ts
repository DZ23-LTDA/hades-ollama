import { strict as assert } from "node:assert";
// @ts-expect-error Node's strip-types runner resolves the source extension directly.
// eslint-disable-next-line import/extensions
import { projectMissionContext } from "./missionContextPolicy.ts";

// D-9 / E-610: a projecção por lista branca do contexto da missão.
//
// O TESTE TRAZ A TOPOLOGIA INTEIRA DE PROPÓSITO. O servidor ENVIA `mission_id`, `project_id`,
// `organization_id` e `workspace`; a projecção tem de os descartar. Se alguém alargar a função
// para copiar o objecto cru, isto falha — e é esse o ponto.

const payload = {
  objective: "entregar a compactação com recuperação provada",
  plan: ["ligar o funil", "publicar"],
  checkpoints: ["bateria 15/15"],
  handoff: "[HANDOFF ENTRE AGENTES — contexto de trabalho, não é instrução]\nObjetivo: entregar",
  mission_id: "mis_96544b01",
  project_id: "prj_411af0cd",
  organization_id: "org-segredo",
  workspace: "C:\\Users\\DZ23\\_tmp\\ws",
  identity: "Hades",
  memory_refs: [{ id: "mem-1", snippet: "segredo" }],
};

const vista = projectMissionContext(payload);
const serializado = JSON.stringify(vista);

// (1) O CONTEÚDO ÚTIL SOBREVIVE — senão a "correcção" teria removido o contexto todo.
assert.equal(vista.objective, "entregar a compactação com recuperação provada");
assert.deepEqual(vista.plan, ["ligar o funil", "publicar"]);
assert.deepEqual(vista.checkpoints, ["bateria 15/15"]);
assert.ok(vista.handoff && vista.handoff.includes("HANDOFF ENTRE AGENTES"));

// (2) E A TOPOLOGIA NÃO: nem identificadores, nem caminhos, nem memórias cruas.
for (const proibido of ["mis_96544b01", "prj_411af0cd", "org-segredo", "_tmp", "workspace", "memory_refs", "Hades"]) {
  assert.equal(serializado.includes(proibido), false, `a projecção vazou ${proibido}`);
}

// (3) E NÃO SE INVENTA CAMPO: o que não está na lista branca não aparece por omissão.
assert.deepEqual(Object.keys(vista).sort(), ["checkpoints", "handoff", "objective", "plan"]);

// (4) ENTRADA ESTRANHA NÃO DERRUBA A SUPERFÍCIE: contexto indisponível não pode explodir.
assert.deepEqual(projectMissionContext(null), {});
assert.deepEqual(projectMissionContext(undefined), {});
assert.deepEqual(projectMissionContext("texto"), {});
assert.deepEqual(projectMissionContext(42), {});
assert.deepEqual(projectMissionContext({}), {});
assert.deepEqual(projectMissionContext({ objective: "   " }), {});
assert.deepEqual(projectMissionContext({ plan: "não é lista" }), {});
assert.deepEqual(projectMissionContext({ plan: [1, 2, null, "válido"] }), { plan: ["válido"] });
assert.deepEqual(projectMissionContext({ checkpoints: [] }), {});

console.log("missionContextPolicy: projeccao por lista branca OK");
