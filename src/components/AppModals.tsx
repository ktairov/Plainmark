import type { FormEvent } from "react";
import { Modal } from "../components";
import type { CryptoDialog, Pending } from "../appTypes";
import type { TrashInfo } from "../model";
import { t, tp } from "../i18n";

type CryptoModalProps = {
  dialog: CryptoDialog;
  password: string;
  confirm: string;
  error: string;
  busy: boolean;
  onPassword: (value: string) => void;
  onConfirm: (value: string) => void;
  onSubmit: (event: FormEvent) => void;
  onClose: () => void;
};

export function CryptoModal({ dialog, password, confirm, error, busy, onPassword, onConfirm, onSubmit, onClose }: CryptoModalProps) {
  const encrypting = dialog.kind === "encrypt";
  return (
    <Modal title={encrypting ? t("crypto.title.encrypt") : t("crypto.title.decrypt")} onClose={busy ? undefined : onClose}>
      <form onSubmit={onSubmit}>
        <p>
          {encrypting ? t("crypto.body.encrypt") : t("crypto.body.decrypt")}
        </p>
        <label className="dialog-field"><span>{t("crypto.password")}</span><input autoFocus type="password" autoComplete="off" value={password} onChange={(event) => onPassword(event.target.value)} /></label>
        {encrypting && <label className="dialog-field"><span>{t("crypto.confirmPassword")}</span><input type="password" autoComplete="off" value={confirm} onChange={(event) => onConfirm(event.target.value)} /></label>}
        {error && <p className="dialog-error" role="alert">{error}</p>}
        <div className="modal-actions">
          <button type="button" disabled={busy} onClick={onClose}>{t("common.cancel")}</button>
          <button className="primary" type="submit" disabled={busy || !password || (encrypting && !confirm)}>
            {busy ? (encrypting ? t("crypto.busy.encrypting") : t("crypto.busy.unlocking")) : encrypting ? t("crypto.encrypt") : t("crypto.unlock")}
          </button>
        </div>
      </form>
    </Modal>
  );
}

type PendingModalProps = {
  pending: Pending;
  dirtyPaths: string[];
  onStay: () => void;
  onDiscard: () => void;
  onSave: () => void;
};

export function PendingModal({ pending, dirtyPaths, onStay, onDiscard, onSave }: PendingModalProps) {
  const single = pending.kind === "close-tab" || dirtyPaths.length === 1;
  return (
    <Modal title={t("pending.title")} onClose={onStay}>
      <p>
        {pending.kind === "close-tab"
          ? t("pending.closeTab", { path: pending.path })
          : single
            ? t("pending.closeOne", { path: dirtyPaths[0] })
            : tp("pending.closeMany", dirtyPaths.length)}
      </p>
      <div className="modal-actions">
        <button onClick={onStay}>{t("common.stay")}</button>
        <button onClick={onDiscard}>{t("common.discard")}</button>
        <button className="primary" onClick={onSave}>{dirtyPaths.length > 1 && pending.kind !== "close-tab" ? t("common.saveAllAndContinue") : t("common.saveAndContinue")}</button>
      </div>
    </Modal>
  );
}

type ConflictModalProps = {
  onLoadFromDisk: () => void;
  onSaveCopy: () => void;
  onOverwrite: () => void;
};

export function ConflictModal({ onLoadFromDisk, onSaveCopy, onOverwrite }: ConflictModalProps) {
  return (
    <Modal title={t("conflict.title")}>
      <p>{t("conflict.body")}</p>
      <div className="modal-actions">
        <button onClick={onLoadFromDisk}>{t("common.loadFromDisk")}</button>
        <button onClick={onSaveCopy}>{t("common.saveCopy")}</button>
        <button className="primary" onClick={onOverwrite}>{t("common.overwrite")}</button>
      </div>
    </Modal>
  );
}

type TrashModalProps = {
  info: TrashInfo;
  onCancel: () => void;
  onConfirm: () => void;
};

export function TrashModal({ info, onCancel, onConfirm }: TrashModalProps) {
  return (
    <Modal title={t("trash.title", { name: info.name })} onClose={onCancel}>
      <p>{info.is_directory ? tp("trash.folder", info.nested_files) : t("trash.note")}</p>
      <div className="modal-actions">
        <button onClick={onCancel}>{t("common.cancel")}</button>
        <button className="danger-button" onClick={onConfirm}>{t("common.moveToTrash")}</button>
      </div>
    </Modal>
  );
}
