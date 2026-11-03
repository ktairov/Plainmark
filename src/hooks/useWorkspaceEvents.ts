import { useEffect, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import { api, events } from "../api";
import { addedNotePaths, type FolderEntry, type NoteContent, type NoteEntry } from "../model";
import type { TabSessionAction } from "../tabSession";
import { t, tp } from "../i18n";
import type { StatusTone } from "./useStatusMessage";

type WorkspaceEventsParams = {
  refreshTree: (preferred: string) => Promise<string>;
  loadNote: (path: string, disposition?: "replace" | "open" | "switch", mode?: "preview" | "edit" | "split") => Promise<void>;
  isLocalTreeAddition: (path: string) => boolean;
  notesRef: MutableRefObject<NoteEntry[]>;
  foldersRef: MutableRefObject<FolderEntry[]>;
  activePathRef: MutableRefObject<string>;
  documentRef: MutableRefObject<NoteContent | null>;
  draftRef: MutableRefObject<string>;
  tabDispatch: (action: TabSessionAction) => void;
  setActiveFolder: Dispatch<SetStateAction<string>>;
  showStatus: (text: string, tone?: StatusTone, timeoutMs?: number) => void;
  showError: (error: unknown, prefix?: string) => void;
};

export function useWorkspaceEvents(params: WorkspaceEventsParams) {
  const {
    refreshTree,
    loadNote,
    isLocalTreeAddition,
    notesRef,
    foldersRef,
    activePathRef,
    documentRef,
    draftRef,
    tabDispatch,
    setActiveFolder,
    showStatus,
    showError,
  } = params;

  useEffect(() => {
    const changes = events.onFilesChanged(async (event) => {
      try {
        const previous = notesRef.current;
        const current = activePathRef.current;
        const next = await refreshTree(current);
        setActiveFolder((folder) => (folder && !foldersRef.current.some((candidate) => candidate.path === folder) ? "" : folder));
        const added = addedNotePaths(previous, notesRef.current).filter((path) => !isLocalTreeAddition(path));
        let announceAdded = true;
        const stillCurrent = () => activePathRef.current === current;
        if (stillCurrent() && current && event.paths.some((path) => path === current || path.startsWith(".notes-save-"))) {
          try {
            const disk = await api.readNote(current);
            if (stillCurrent()) {
              const doc = documentRef.current;
              if (doc?.path !== current) return;
              if (draftRef.current !== doc.markdown) {
                if (disk.modified_ms !== doc.modified_ms || disk.markdown !== doc.markdown) {
                  tabDispatch({ type: "external-conflict", path: current, note: disk });
                }
              } else if (disk.modified_ms !== doc.modified_ms || disk.markdown !== doc.markdown) {
                tabDispatch({ type: "replace-from-disk", path: current, note: disk });
              }
            }
          } catch {
            if (stillCurrent()) {
              const doc = documentRef.current;
              if (doc?.path === current && draftRef.current !== doc.markdown) {
                tabDispatch({ type: "external-conflict", path: current, note: null });
                showStatus(t("watch.noteRemoved"), "error", 0);
                announceAdded = false;
              } else if (next !== current) {
                await loadNote(next);
              }
            }
          }
        } else if (stillCurrent() && current && next !== current) {
          await loadNote(next);
        }
        if (announceAdded && added.length === 1) showStatus(t("watch.newNote", { path: added[0] }), "info", 3000);
        else if (announceAdded && added.length > 1) showStatus(tp("watch.newNotes", added.length), "info", 3000);
      } catch (error) {
        showError(error);
      }
    });
    const errors = events.onWatchError((message) => showStatus(message, "error", 0));
    return () => {
      changes.then((unlisten) => unlisten());
      errors.then((unlisten) => unlisten());
    };
  }, [activePathRef, documentRef, draftRef, foldersRef, isLocalTreeAddition, loadNote, notesRef, refreshTree, setActiveFolder, showError, showStatus, tabDispatch]);
}
