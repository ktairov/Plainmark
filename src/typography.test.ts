import { describe, expect, it } from "vitest";
import { typographicDashes } from "./typography";

describe("typographicDashes", () => {
  it("converts a triple dash at the beginning of a quote text node", () => {
    expect(typographicDashes("--- Мы пришли к тебе потому, что были свободными.")).toBe(
      "— Мы пришли к тебе потому, что были свободными.",
    );
  });

  it("converts dashes after whitespace and across wrapped quote text", () => {
    expect(typographicDashes("Старик ответил:\n--- Пока нас было сорок\nтысяч, свобода была порядком.")).toBe(
      "Старик ответил:\n— Пока нас было сорок\nтысяч, свобода была порядком.",
    );
  });

  it("uses an en dash for a double dash", () => {
    expect(typographicDashes("Версия 1 -- черновик")).toBe("Версия 1 – черновик");
  });
});
