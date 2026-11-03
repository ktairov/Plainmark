export type SystemTheme = "light" | "dark";

type MediaThemeSource = {
  matches: boolean;
  addEventListener: (type: "change", listener: (event: { matches: boolean }) => void) => void;
  removeEventListener: (type: "change", listener: (event: { matches: boolean }) => void) => void;
};

type NativeThemeSource = {
  current: () => Promise<SystemTheme | null>;
  subscribe: (listener: (theme: SystemTheme) => void) => Promise<() => void>;
};

export function observeSystemTheme(
  media: MediaThemeSource,
  native: NativeThemeSource,
  onChange: (dark: boolean) => void,
): () => void {
  let disposed = false;
  let unlistenNative: (() => void) | undefined;
  const update = (dark: boolean) => {
    if (!disposed) onChange(dark);
  };
  const onMediaChange = (event: { matches: boolean }) => update(event.matches);

  media.addEventListener("change", onMediaChange);
  update(media.matches);

  void (async () => {
    try {
      const unlisten = await native.subscribe(theme => update(theme === "dark"));
      if (disposed) unlisten();
      else unlistenNative = unlisten;
    } catch {
      // matchMedia remains the fallback outside Tauri and on older runtimes.
    }

    try {
      const theme = await native.current();
      if (theme) update(theme === "dark");
    } catch {
      // The initial matchMedia value has already been applied.
    }
  })();

  return () => {
    disposed = true;
    media.removeEventListener("change", onMediaChange);
    unlistenNative?.();
  };
}
