import { DndContext, DragOverlay, pointerWithin, type DragEndEvent, type DragStartEvent, type SensorDescriptor, type SensorOptions } from "@dnd-kit/core";
import { FileText, FolderOpen, Search, X } from "lucide-react";
import { useState, type RefObject } from "react";
import { NoteTree, PinnedNotes, RootDropZone } from "../components";
import type { DragItem } from "../appTypes";
import type { NoteEntry, TreeNode } from "../model";
import { t } from "../i18n";

type Props = {
  rootLabel: string;
  query: string;
  tree: TreeNode;
  pinnedNotes: NoteEntry[];
  activePath: string;
  activeFolder: string;
  expanded: Set<string>;
  activeDrag: DragItem | null;
  searchInputRef: RefObject<HTMLInputElement | null>;
  sensors: SensorDescriptor<SensorOptions>[];
  setQuery: (query: string) => void;
  selectFolder: (path: string) => void;
  selectNote: (path: string, newTab?: boolean, fragment?: string) => void;
  toggleFolder: (path: string) => void;
  showMenu: (event: React.MouseEvent, path: string, kind: "folder" | "note") => void;
  startDrag: (event: DragStartEvent) => void;
  endDrag: (event: DragEndEvent) => void;
  cancelDrag: () => void;
  showRootMenu: (event: React.MouseEvent<HTMLDivElement>) => void;
};

export function AppSidebar(props: Props) {
  const [notesOpen, setNotesOpen] = useState(true);
  return <aside className="sidebar">
    <div className="search"><Search size={16}/><input ref={props.searchInputRef} aria-label={t("sidebar.search")} value={props.query} onChange={event=>props.setQuery(event.target.value)} placeholder={t("sidebar.search")}/><button className={`search-clear tooltip-button ${props.query?"":"hidden"}`} type="button" data-tooltip={t("preview.clearSearch")} aria-label={t("preview.clearSearch")} aria-hidden={!props.query} tabIndex={props.query?0:-1} onClick={()=>{props.setQuery("");props.searchInputRef.current?.focus()}}><X size={15}/></button></div>
    <DndContext sensors={props.sensors} collisionDetection={pointerWithin} onDragStart={props.startDrag} onDragCancel={props.cancelDrag} onDragEnd={props.endDrag}>
      <div className="note-list" onContextMenu={props.showRootMenu}>
        <PinnedNotes notes={props.pinnedNotes} active={props.activePath} select={props.selectNote} menu={props.showMenu}/>
        <section className="notes-tree-section" aria-label={t("sidebar.notes")}>
          <RootDropZone label={props.rootLabel} active={!props.activeFolder} open={notesOpen} select={()=>{props.selectFolder("");setNotesOpen(value=>!value)}}/>
          {notesOpen&&<NoteTree node={props.tree} active={props.activePath} activeFolder={props.activeFolder} expanded={props.expanded} forced={Boolean(props.query.trim())} select={props.selectNote} selectFolder={props.selectFolder} toggle={props.toggleFolder} menu={props.showMenu}/>}
        </section>
      </div>
      <DragOverlay>{props.activeDrag&&<div className="drag-overlay">{props.activeDrag.kind==="folder"?<FolderOpen size={16}/>:<FileText size={15}/>}<span>{props.activeDrag.label}</span></div>}</DragOverlay>
    </DndContext>
  </aside>;
}
