import { strict as assert } from "node:assert";
// @ts-expect-error Node's strip-types runner resolves the source extension directly.
import { inboxKey, loadInbox, loadInboxDetail } from "./inbox.ts";

const data = new Map<string, string>();
const storage = {
  getItem: async (key: string) => data.get(key) ?? null,
  setItem: async (key: string, value: string) => { data.set(key, value); },
};
const mission = { id: "m1", objective: "Revisar execução", state: "PAUSED" };
const events = [{ id: "e1", type: "mission.paused", created_at: "2026-09-29T12:00:00Z" }];
const key = inboxKey("http://localhost:11434", "org-a", true);
assert.ok(key);
assert.notEqual(key, inboxKey("http://localhost:11434", "org-b", true));
assert.equal(inboxKey("http://localhost:11434", null, true), null);

const loaded = await loadInbox(storage, key, async <T,>() => ({ missions: [mission] }) as T);
assert.equal(loaded.source, "network");
assert.deepEqual(loaded.missions, [mission]);

const cached = await loadInbox(storage, key, async () => { throw new Error("network down"); });
assert.equal(cached.source, "cache");
assert.deepEqual(cached.missions, [mission]);

const serverError = await loadInbox(storage, key, async () => { throw Object.assign(new Error("server failed"), { status: 503 }); });
assert.equal(serverError.source, "error");
assert.match(serverError.error, /server failed/);

const emptyError = await loadInbox(storage, `${key}.empty`, async () => { throw new Error("network down"); });
assert.equal(emptyError.source, "error");
assert.equal(emptyError.missions.length, 0);

const detailKey = "dz23.agent.cached.mission.local";
const detail = await loadInboxDetail(storage, detailKey, mission.id, async <T,>(path: string) => (path.endsWith("/events") ? { events } : mission) as T);
assert.equal(detail.source, "network");
assert.deepEqual(detail.events, events);

const offlineDetail = await loadInboxDetail(storage, detailKey, mission.id, async () => { throw new Error("offline"); });
assert.equal(offlineDetail.source, "cache");
assert.equal(offlineDetail.mission?.id, mission.id);

const wrongMission = await loadInboxDetail(storage, detailKey, "m2", async () => { throw new Error("offline"); });
assert.equal(wrongMission.source, "error");
assert.equal(wrongMission.mission, null);

console.log("inbox: PASS");
