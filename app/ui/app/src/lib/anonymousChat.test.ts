import { afterEach, describe, expect, it, vi } from "vitest";
import {
  isAnonymousChat,
  isAnonymousModeEnabled,
  markAnonymousChat,
  setAnonymousMode,
  shouldSendTemporary,
  subscribeAnonymousChat,
} from "./anonymousChat";

describe("anonymous chat state", () => {
  afterEach(() => setAnonymousMode(false));

  it("only flags new chats while the mode is on", () => {
    expect(shouldSendTemporary("new")).toBe(false);
    setAnonymousMode(true);
    expect(isAnonymousModeEnabled()).toBe(true);
    expect(shouldSendTemporary("new")).toBe(true);
    expect(shouldSendTemporary("existing-id")).toBe(false);
  });

  it("remembers anonymous chat ids and notifies subscribers", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeAnonymousChat(listener);
    markAnonymousChat("abc");
    setAnonymousMode(true);
    unsubscribe();
    setAnonymousMode(false);
    expect(isAnonymousChat("abc")).toBe(true);
    expect(isAnonymousChat("other")).toBe(false);
    expect(isAnonymousChat(undefined)).toBe(false);
    expect(listener).toHaveBeenCalledTimes(2);
  });
});
