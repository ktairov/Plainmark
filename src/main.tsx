import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import { cleanupLegacyStorage, loadSettings } from "./storage";
import { resolveLanguage, setActiveLanguage } from "./i18n";
import "katex/dist/katex.min.css";
import "./theme.css";
import "./styles.css";

cleanupLegacyStorage();
setActiveLanguage(resolveLanguage(loadSettings().language, [navigator.language, ...navigator.languages]));

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
