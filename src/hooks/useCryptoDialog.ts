import { useCallback, useEffect, useState, type FormEvent, type MutableRefObject } from "react";
import type { EditorView } from "@codemirror/view";
import { api } from "../api";
import { encryptedMarkdownBlock, errorMessage, type NoteContent } from "../model";
import type { CryptoDialog, Mode } from "../appTypes";
import { t } from "../i18n";
import type { StatusTone } from "./useStatusMessage";

type CryptoDialogParams = {
  editorViewRef: MutableRefObject<EditorView | null>;
  mode: Mode;
  activeDocument: NoteContent | null;
  setDraft: (value: string) => void;
  showStatus: (text: string, tone?: StatusTone, timeoutMs?: number) => void;
};

export function useCryptoDialog(params: CryptoDialogParams) {
  const { editorViewRef, mode, activeDocument, setDraft, showStatus } = params;
  const [cryptoDialog, setCryptoDialog] = useState<CryptoDialog | null>(null);
  const [cryptoPassword, setCryptoPassword] = useState("");
  const [cryptoConfirm, setCryptoConfirm] = useState("");
  const [cryptoError, setCryptoError] = useState("");
  const [cryptoBusy, setCryptoBusy] = useState(false);
  const [unlocked, setUnlocked] = useState<Record<string, string>>({});

  const closeCrypto = useCallback(() => {
    setCryptoDialog(null);
    setCryptoPassword("");
    setCryptoConfirm("");
    setCryptoError("");
    setCryptoBusy(false);
  }, []);

  const requestEncryptSelection = useCallback(() => {
    const view = editorViewRef.current;
    if (mode === "preview" || !view) {
      showStatus(t("crypto.needEditorSelection"));
      return;
    }
    const { from, to } = view.state.selection.main;
    if (from === to) {
      showStatus(t("crypto.selectText"));
      return;
    }
    const plaintext = view.state.sliceDoc(from, to);
    setCryptoPassword("");
    setCryptoConfirm("");
    setCryptoError("");
    setCryptoDialog({ kind: "encrypt", from, to, plaintext });
  }, [editorViewRef, mode, showStatus]);

  const requestDecrypt = useCallback((payload: string) => {
    setCryptoPassword("");
    setCryptoConfirm("");
    setCryptoError("");
    setCryptoDialog({ kind: "decrypt", payload });
  }, []);

  const lockEncrypted = useCallback((payload: string) => {
    setUnlocked((current) => {
      const next = { ...current };
      delete next[payload];
      return next;
    });
  }, []);

  const submitCrypto = useCallback(
    async (event: FormEvent) => {
      event.preventDefault();
      if (!cryptoDialog || cryptoBusy) return;
      if (!cryptoPassword) {
        setCryptoError(t("crypto.passwordRequired"));
        return;
      }
      if (cryptoDialog.kind === "encrypt" && cryptoPassword !== cryptoConfirm) {
        setCryptoError(t("crypto.passwordMismatch"));
        return;
      }
      setCryptoBusy(true);
      setCryptoError("");
      try {
        if (cryptoDialog.kind === "encrypt") {
          const view = editorViewRef.current;
          if (!view || view.state.sliceDoc(cryptoDialog.from, cryptoDialog.to) !== cryptoDialog.plaintext) {
            throw new Error(t("crypto.selectionChanged"));
          }
          const payload = await api.encryptText(cryptoDialog.plaintext, cryptoPassword);
          const before = view.state.sliceDoc(Math.max(0, cryptoDialog.from - 1), cryptoDialog.from);
          const after = view.state.sliceDoc(cryptoDialog.to, Math.min(view.state.doc.length, cryptoDialog.to + 1));
          const block = encryptedMarkdownBlock(payload, before, after);
          view.dispatch({
            changes: { from: cryptoDialog.from, to: cryptoDialog.to, insert: block },
            selection: { anchor: cryptoDialog.from + block.length },
          });
          setDraft(view.state.doc.toString());
          view.focus();
          showStatus(t("crypto.encrypted"));
        } else {
          const plaintext = await api.decryptText(cryptoDialog.payload, cryptoPassword);
          setUnlocked((current) => ({ ...current, [cryptoDialog.payload]: plaintext }));
          showStatus(t("crypto.unlocked"), "success");
        }
        closeCrypto();
      } catch (error) {
        setCryptoError(errorMessage(error));
        setCryptoBusy(false);
      }
    },
    [closeCrypto, cryptoBusy, cryptoConfirm, cryptoDialog, cryptoPassword, editorViewRef, setDraft, showStatus]
  );

  useEffect(() => {
    setUnlocked({});
  }, [activeDocument?.path, activeDocument?.modified_ms]);

  return {
    cryptoDialog,
    cryptoPassword,
    setCryptoPassword,
    cryptoConfirm,
    setCryptoConfirm,
    cryptoError,
    cryptoBusy,
    unlocked,
    closeCrypto,
    requestEncryptSelection,
    requestDecrypt,
    lockEncrypted,
    submitCrypto,
  };
}
