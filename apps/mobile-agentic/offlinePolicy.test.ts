import { strict as assert } from "node:assert";
// @ts-expect-error Node's strip-types runner resolves the source extension directly.
// eslint-disable-next-line import/extensions
import { buildApprovalDecisionPayload, missionStorageKey, pushStorageKey, queueStorageKey, shouldQueueOffline } from "./offlinePolicy.ts";

assert.equal(shouldQueueOffline(new TypeError("Network request failed")), true);
assert.equal(shouldQueueOffline(new TypeError("Failed to fetch")), true);
assert.equal(shouldQueueOffline(new Error("network down")), false);
assert.equal(shouldQueueOffline({ status: 409 }), false);
assert.equal(shouldQueueOffline({ status: 422 }), false);
assert.equal(shouldQueueOffline({ status: 503 }), false);
assert.equal(shouldQueueOffline({ message: "not an HTTP response" }), false);
assert.equal(shouldQueueOffline(new TypeError("Cannot read properties of undefined")), false);

assert.deepEqual(buildApprovalDecisionPayload(true, "nonce-1", "  revisar  "), {
  decision: "approve",
  nonce: "nonce-1",
  reason: "revisar",
});
assert.deepEqual(buildApprovalDecisionPayload(false, "nonce-2", "não seguro"), {
  decision: "reject",
  nonce: "nonce-2",
  reason: "não seguro",
});

const userAQueue = queueStorageKey("http://server", "org-1", "user-a", true);
const userBQueue = queueStorageKey("http://server", "org-1", "user-b", true);
assert.ok(userAQueue);
assert.ok(userBQueue);
assert.notEqual(userAQueue, userBQueue);
assert.equal(queueStorageKey("http://server", "org-1", null, true), null);
assert.equal(queueStorageKey("http://server", null, null, false), null);
assert.ok(queueStorageKey("http://server", "local", "local", false));
assert.notEqual(missionStorageKey("http://server", "org-1", "user-a", true), missionStorageKey("http://server", "org-1", "user-b", true));
assert.notEqual(pushStorageKey("http://server", "org-1", "user-a"), pushStorageKey("http://server", "org-1", "user-b"));

console.log("offlinePolicy: PASS");
