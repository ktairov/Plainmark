import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import { bootThemeScript, type BootThemePalette } from "./src/bootTheme";
import { defaultAppSettings } from "./src/settings";
import { themes } from "./src/themes";

const bootPages: BootThemePalette = Object.fromEntries(
  Object.entries(themes).map(([id, theme]) => [id, { light: theme.light.page, dark: theme.dark.page }])
);
const bootDefaults = { light: defaultAppSettings.dayTheme, dark: defaultAppSettings.nightTheme };

/**
 * Injects a tiny inline script into index.html so the webview paints the
 * stored theme background before React (and even the CSS) has finished
 * loading. Without it the first frame is the light default and dark-mode
 * users see a white flash at startup.
 */
function themeBootstrap(): Plugin {
  const script = bootThemeScript(bootPages, bootDefaults);
  return {
    name: "plainmark-theme-bootstrap",
    transformIndexHtml(html) {
      return {
        html,
        tags: [{ tag: "script", children: script, injectTo: "head-prepend" }],
      };
    },
  };
}

export default defineConfig({
  plugins: [themeBootstrap(), react()],
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
    watch: {
      ignored: ["**/src-tauri/**"],
    },
  },
});
