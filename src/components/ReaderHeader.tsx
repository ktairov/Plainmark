import { ChevronDown, Columns2, Eye, FilePlus2, FolderPlus, FolderOpen, Moon, PanelLeftClose, PanelLeftOpen, PencilLine, Plus, Sun } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Mode, SaveStatus } from "../appTypes";
import { t } from "../i18n";

type Props = {
  sidebarVisible: boolean;
  mode: Mode;
  colorMode: "light" | "dark";
  saveStatus: SaveStatus;
  saveError: string;
  rootLabel: string;
  children: ReactNode;
  chooseRoot: (path?: string) => void;
  create: (kind: "note" | "folder") => void;
  toggleSidebar: () => void;
  setMode: (mode: Mode) => void;
  toggleColorMode: () => void;
  recentRoots: string[];
};

export function ReaderHeader(props: Props) {
  const [createMenuOpen,setCreateMenuOpen]=useState(false);
  const [rootMenuOpen,setRootMenuOpen]=useState(false);
  const createMenuRef=useRef<HTMLDivElement>(null);
  const rootMenuRef=useRef<HTMLDivElement>(null);
  useEffect(()=>{
    if(!createMenuOpen)return;
    const closeOnPointerDown=(event:PointerEvent)=>{if(!createMenuRef.current?.contains(event.target as Node))setCreateMenuOpen(false)};
    const closeOnEscape=(event:KeyboardEvent)=>{if(event.key==="Escape")setCreateMenuOpen(false)};
    window.addEventListener("pointerdown",closeOnPointerDown);
    window.addEventListener("keydown",closeOnEscape);
    return()=>{window.removeEventListener("pointerdown",closeOnPointerDown);window.removeEventListener("keydown",closeOnEscape)};
  },[createMenuOpen]);
  useEffect(()=>{
    if(!rootMenuOpen)return;
    const closeOnPointerDown=(event:PointerEvent)=>{if(!rootMenuRef.current?.contains(event.target as Node))setRootMenuOpen(false)};
    const closeOnEscape=(event:KeyboardEvent)=>{if(event.key==="Escape")setRootMenuOpen(false)};
    window.addEventListener("pointerdown",closeOnPointerDown);
    window.addEventListener("keydown",closeOnEscape);
    return()=>{window.removeEventListener("pointerdown",closeOnPointerDown);window.removeEventListener("keydown",closeOnEscape)};
  },[rootMenuOpen]);
  const create=(kind:"note"|"folder")=>{setCreateMenuOpen(false);props.create(kind)};
  return <header className="reader-header" data-tauri-drag-region>
    <div className="header-leading">
      <button className="icon-button tooltip-button sidebar-toggle" data-tooltip={props.sidebarVisible?t("header.hideSidebar"):t("header.showSidebar")} aria-label={props.sidebarVisible?t("header.hideSidebar"):t("header.showSidebar")} aria-expanded={props.sidebarVisible} onClick={props.toggleSidebar}>{props.sidebarVisible?<PanelLeftClose size={18}/>:<PanelLeftOpen size={18}/>}</button>
      <div className="titlebar-root-menu" ref={rootMenuRef}>
        <button className="titlebar-root tooltip-button" type="button" data-tooltip={t("header.openFolder")} aria-label={t("header.currentFolder", { name: props.rootLabel })} onClick={()=>setRootMenuOpen(open=>!open)}><span>{props.rootLabel}</span><ChevronDown size={13}/></button>
        {rootMenuOpen&&<div className="titlebar-root-popover" role="menu">
          <div className="titlebar-root-recent">
            {props.recentRoots.length > 0 ? (
              props.recentRoots.map(path => (
                <button key={path} type="button" role="menuitem" onClick={()=>{setRootMenuOpen(false);props.chooseRoot(path)}}><FolderOpen size={15}/><span title={path}>{path}</span></button>
              ))
            ) : (
              <div className="titlebar-root-placeholder">{t("header.noRecentFolders")}</div>
            )}
          </div>
          <div className="titlebar-root-divider" />
          <button type="button" role="menuitem" onClick={()=>{setRootMenuOpen(false);props.chooseRoot()}}><FolderOpen size={15}/><span>{t("header.openFolderAction")}</span></button>
        </div>}
      </div>
      <div className="titlebar-create-menu" ref={createMenuRef}>
        <button className={`tooltip-button titlebar-create ${createMenuOpen?"active":""}`} type="button" data-tooltip={createMenuOpen?undefined:t("header.create")} aria-label={t("header.createAria")} aria-haspopup="menu" aria-expanded={createMenuOpen} onClick={()=>setCreateMenuOpen(open=>!open)}><Plus size={16}/></button>
        {createMenuOpen&&<div className="titlebar-create-popover" role="menu">
          <button type="button" role="menuitem" onClick={()=>create("note")}><FilePlus2 size={15}/><span>{t("header.newNote")}</span></button>
          <button type="button" role="menuitem" onClick={()=>create("folder")}><FolderPlus size={15}/><span>{t("header.newFolder")}</span></button>
        </div>}
      </div>
      <span className="titlebar-separator" aria-hidden="true"/>
    </div>
    <div className="titlebar-tabs" data-tauri-drag-region>{props.children}</div>
    <div className="mode-switch" role="group" aria-label={t("header.documentView")}>{([['preview','header.mode.preview',Eye],['edit','header.mode.edit',PencilLine],['split','header.mode.split',Columns2]] as const).map(([value,labelKey,Icon],index)=><button className={`tooltip-button ${props.mode===value?"selected":""}`} data-tooltip={t('header.modeShortcut', { label: t(labelKey), number: index + 1 })} aria-label={t(labelKey)} aria-pressed={props.mode===value} key={value} onClick={()=>props.setMode(value)}><Icon size={16}/></button>)}</div>
    <div className="toolbar"><button className="icon-button tooltip-button theme-toggle" type="button" data-tooltip={props.colorMode==="dark"?t("header.dayMode"):t("header.nightMode")} aria-label={props.colorMode==="dark"?t("header.switchToDay"):t("header.switchToNight")} onClick={props.toggleColorMode}>{props.colorMode==="dark"?<Sun size={16}/>:<Moon size={16}/>}</button>{(props.saveStatus==="conflict"||props.saveStatus==="error")&&<span className={`save-status ${props.saveStatus}`} title={props.saveError}>{props.saveStatus==="conflict"?t("header.externalConflict"):props.saveError||t("header.saveFailed")}</span>}</div>
  </header>;
}
