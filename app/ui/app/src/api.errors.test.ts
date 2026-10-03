import { afterEach, describe, expect, it, vi } from "vitest";

import { getChat, getChats, sendMessage } from "./api";
import { ErrorEvent, Model } from "@/gotypes";
import type { ChatEventUnion } from "./api";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("api error handling", () => {
  it("getChat throws a descriptive error on a non-ok response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 500 }),
    );
    await expect(getChat("abc")).rejects.toThrow(/500/);
  });

  it("getChats throws a descriptive error on a non-ok response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 502 }),
    );
    await expect(getChats()).rejects.toThrow(/502/);
  });

  it("sendMessage yields an error event instead of silently completing", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 500 }),
    );
    const events: ChatEventUnion[] = [];
    for await (const event of sendMessage(
      "chat1",
      "oi",
      new Model({ model: "qwen2.5:0.5b" }),
    )) {
      events.push(event);
    }
    expect(events).toHaveLength(1);
    expect(events[0]).toBeInstanceOf(ErrorEvent);
    expect((events[0] as ErrorEvent).error).toContain("500");
  });
});
