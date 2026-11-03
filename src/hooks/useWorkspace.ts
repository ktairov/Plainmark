import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../api";
import type { FolderEntry, NoteEntry } from "../model";

export function useWorkspace() {
  const [root, setRoot] = useState("");
  const [notes, setNotes] = useState<NoteEntry[]>([]);
  const [folders, setFolders] = useState<FolderEntry[]>([]);
  const notesRef = useRef(notes);
  useEffect(() => {
    notesRef.current = notes;
  }, [notes]);
  const foldersRef = useRef(folders);
  useEffect(() => {
    foldersRef.current = folders;
  }, [folders]);
  const treeRefreshRequestRef = useRef(0);

  const cancelPendingRefresh = useCallback(() => {
    ++treeRefreshRequestRef.current;
  }, []);

  const clearWorkspace = useCallback(() => {
    notesRef.current = [];
    foldersRef.current = [];
    setNotes([]);
    setFolders([]);
  }, []);

  const refreshTree = useCallback(async (preferred: string) => {
    const request = ++treeRefreshRequestRef.current;
    const snapshot = await api.scanWorkspace();
    if (request !== treeRefreshRequestRef.current) return preferred;
    notesRef.current = snapshot.notes;
    foldersRef.current = snapshot.folders;
    setNotes(snapshot.notes);
    setFolders(snapshot.folders);
    const candidate = snapshot.notes.some((note) => note.path === preferred) ? preferred : snapshot.notes[0]?.path || "";
    return candidate;
  }, []);

  return { root, setRoot, notes, folders, notesRef, foldersRef, refreshTree, cancelPendingRefresh, clearWorkspace };
}
