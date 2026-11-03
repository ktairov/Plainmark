import { describe, expect, it, vi } from "vitest";
import { observeSystemTheme, type SystemTheme } from "./systemTheme";

describe("system theme observer", () => {
  it("follows native macOS appearance changes and cleans up both listeners", async () => {
    let mediaListener: ((event: { matches: boolean }) => void) | undefined;
    let nativeListener: ((theme: SystemTheme) => void) | undefined;
    const removeMedia = vi.fn();
    const unlistenNative = vi.fn();
    const changes: boolean[] = [];
    const stop = observeSystemTheme(
      {
        matches: false,
        addEventListener: (_type, listener) => { mediaListener = listener; },
        removeEventListener: (_type, listener) => { removeMedia(listener); },
      },
      {
        current: async () => "light",
        subscribe: async listener => {
          nativeListener = listener;
          return unlistenNative;
        },
      },
      dark => changes.push(dark),
    );

    await vi.waitFor(() => expect(nativeListener).toBeTypeOf("function"));
    nativeListener?.("dark");
    expect(changes.at(-1)).toBe(true);

    mediaListener?.({ matches: false });
    expect(changes.at(-1)).toBe(false);

    stop();
    expect(removeMedia).toHaveBeenCalledOnce();
    expect(unlistenNative).toHaveBeenCalledOnce();
    nativeListener?.("dark");
    expect(changes.at(-1)).toBe(false);
  });
});
