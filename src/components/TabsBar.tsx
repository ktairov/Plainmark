import { closestCenter, DndContext, DragOverlay, PointerSensor, useSensor, useSensors, type DragEndEvent, type Modifier } from "@dnd-kit/core";
import { useEffect, useRef, useState } from "react";
import { DraggableTab } from "../components";
import type { OpenNoteTab } from "../model";
import { t } from "../i18n";

const horizontalTabDrag: Modifier = ({ transform }) => ({ ...transform, y: 0 });

type Props = {
  tabs: OpenNoteTab[];
  activePath: string;
  isDirty: (path: string) => boolean;
  navigate: (path: string) => void;
  close: (path: string) => void;
  reorder: (from: string, to: string) => void;
};

export function TabsBar({ tabs, activePath, isDirty, navigate, close, reorder }: Props) {
  const [draggingTabPath, setDraggingTabPath] = useState("");
  const containerRef = useRef<HTMLDivElement | null>(null);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));

  const resetTabScroll = () => {
    const container = containerRef.current;
    if (!container) return;
    container.scrollTop = 0;
    requestAnimationFrame(() => {
      if (containerRef.current) containerRef.current.scrollTop = 0;
    });
  };

  useEffect(() => {
    const activeTab = containerRef.current?.querySelector<HTMLElement>(".document-tab.active");
    activeTab?.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [activePath, tabs.length]);

  const handleDragEnd = (event: DragEndEvent) => {
    setDraggingTabPath("");
    resetTabScroll();
    const from = String(event.active.data.current?.path || "");
    const to = String(event.over?.data.current?.path || "");
    if (from && to) reorder(from, to);
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      modifiers={[horizontalTabDrag]}
      autoScroll={false}
      onDragStart={(event) => {
        resetTabScroll();
        setDraggingTabPath(String(event.active.data.current?.path || ""));
      }}
      onDragCancel={() => {
        setDraggingTabPath("");
        resetTabScroll();
      }}
      onDragEnd={handleDragEnd}
    >
      <div
        className="document-tabs"
        ref={containerRef}
        role="tablist"
        aria-label={t("header.openNotes")}
        data-tauri-drag-region
        onWheel={(event) => {
          const container = event.currentTarget;
          if (container.scrollWidth <= container.clientWidth || Math.abs(event.deltaX) > Math.abs(event.deltaY)) return;
          container.scrollLeft += event.deltaY;
        }}
      >
        {tabs.map((tab) => (
          <DraggableTab
            key={tab.path}
            tab={tab}
            active={tab.path === activePath}
            dirty={isDirty(tab.path)}
            navigate={(path) => navigate(path)}
            close={close}
          />
        ))}
      </div>
      <DragOverlay modifiers={[horizontalTabDrag]} dropAnimation={null}>
        {draggingTabPath && <div className="tab-drag-overlay">{tabs.find((tab) => tab.path === draggingTabPath)?.title}</div>}
      </DragOverlay>
    </DndContext>
  );
}
