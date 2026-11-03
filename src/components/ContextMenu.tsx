import { useLayoutEffect, useRef, useState } from "react";
import { ClipboardCopy, FilePlus2, FolderOpen, FolderPlus, MoreVertical, Pin, Trash2 } from "lucide-react";
import type { MenuTarget } from "../appTypes";
import { fitFloatingPosition } from "../floatingPosition";
import { t } from "../i18n";

type Props = {
  menu: MenuTarget;
  pinned: boolean;
  onNewNote: () => void;
  onNewFolder: () => void;
  onCopyPath: () => void;
  onOpenInNewTab: () => void;
  onTogglePin: () => void;
  onRename: () => void;
  onMove: () => void;
  onDelete: () => void;
};

export function ContextMenu({ menu, pinned, onNewNote, onNewFolder, onCopyPath, onOpenInNewTab, onTogglePin, onRename, onMove, onDelete }: Props) {
  const elementRef = useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = useState({ x: menu.x, y: menu.y });

  useLayoutEffect(() => {
    const element = elementRef.current;
    if (!element) return;
    const place = () => {
      const bounds = element.getBoundingClientRect();
      setPosition(fitFloatingPosition(
        { x: menu.x, y: menu.y },
        { width: bounds.width, height: bounds.height },
        { width: window.innerWidth, height: window.innerHeight },
      ));
    };
    place();
    window.addEventListener("resize", place);
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(place);
    observer?.observe(element);
    return () => {
      window.removeEventListener("resize", place);
      observer?.disconnect();
    };
  }, [menu.x, menu.y]);

  return (
    <div ref={elementRef} className="context-menu" style={{ left: position.x, top: position.y }} onClick={(event) => event.stopPropagation()}>
      {menu.kind !== "note" && (
        <>
          <button onClick={onNewNote}><FilePlus2 size={15} />{t("context.newNote")}</button>
          <button onClick={onNewFolder}><FolderPlus size={15} />{t("context.newFolder")}</button>
        </>
      )}
      {menu.kind === "note" && (
        <>
          <button onClick={onCopyPath}><ClipboardCopy size={15} />{t("context.copyPath")}</button>
          <button onClick={onOpenInNewTab}><FilePlus2 size={15} />{t("context.openInNewTab")}</button>
          <button onClick={onTogglePin}><Pin size={15} />{pinned ? t("context.unpin") : t("context.pin")}</button>
        </>
      )}
      {menu.kind !== "root" && (
        <>
          <button onClick={onRename}><MoreVertical size={15} />{t("context.rename")}</button>
          <button onClick={onMove}><FolderOpen size={15} />{t("context.move")}</button>
          <button className="danger" onClick={onDelete}><Trash2 size={15} />{t("context.trash")}</button>
        </>
      )}
    </div>
  );
}
