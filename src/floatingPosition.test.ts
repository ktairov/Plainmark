import { describe, expect, it } from "vitest";
import { fitFloatingPosition } from "./floatingPosition";

describe("floating menu positioning", () => {
  it("keeps a menu inside every viewport edge", () => {
    expect(fitFloatingPosition({ x: 100, y: 100 }, { width: 170, height: 240 }, { width: 800, height: 600 })).toEqual({ x: 100, y: 100 });
    expect(fitFloatingPosition({ x: 760, y: 570 }, { width: 170, height: 240 }, { width: 800, height: 600 })).toEqual({ x: 622, y: 352 });
    expect(fitFloatingPosition({ x: -20, y: -10 }, { width: 170, height: 240 }, { width: 800, height: 600 })).toEqual({ x: 8, y: 8 });
  });

  it("pins an oversized menu to the viewport margin", () => {
    expect(fitFloatingPosition({ x: 20, y: 20 }, { width: 500, height: 500 }, { width: 320, height: 240 })).toEqual({ x: 8, y: 8 });
  });
});
