import { describe, it, expect } from "vitest";
import { reorderById } from "./studioReorder";

const list = [{ id: "a" }, { id: "b" }, { id: "c" }, { id: "d" }];

describe("reorderById", () => {
  it("moves an item forward to the target position", () => {
    expect(reorderById(list, "a", "c").map((x) => x.id)).toEqual(["b", "c", "a", "d"]);
  });

  it("moves an item backward to the target position", () => {
    expect(reorderById(list, "d", "b").map((x) => x.id)).toEqual(["a", "d", "b", "c"]);
  });

  it("is a no-op when dragging onto itself", () => {
    expect(reorderById(list, "b", "b").map((x) => x.id)).toEqual(["a", "b", "c", "d"]);
  });

  it("returns a copy and never mutates the input", () => {
    const out = reorderById(list, "a", "b");
    expect(out).not.toBe(list);
    expect(list.map((x) => x.id)).toEqual(["a", "b", "c", "d"]);
  });

  it("ignores unknown ids", () => {
    expect(reorderById(list, "zzz", "b").map((x) => x.id)).toEqual(["a", "b", "c", "d"]);
  });
});
