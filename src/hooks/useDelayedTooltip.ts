import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";

export type TooltipPopup = { text: string; x: number; y: number };

export function useDelayedTooltip(delayMs = 500) {
  const [tooltip, setTooltip] = useState<TooltipPopup | null>(null);
  const timerRef = useRef<number | null>(null);

  const hide = useCallback(() => {
    if (timerRef.current !== null) window.clearTimeout(timerRef.current);
    timerRef.current = null;
    setTooltip(null);
  }, []);

  const onPointerOver = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    const target = event.target instanceof Element ? event.target.closest<HTMLElement>(".tooltip-button[data-tooltip]") : null;
    if (!target || !event.currentTarget.contains(target)) return;
    const previous = event.relatedTarget instanceof Node ? event.relatedTarget : null;
    if (previous && target.contains(previous)) return;
    hide();
    timerRef.current = window.setTimeout(() => {
      timerRef.current = null;
      const rect = target.getBoundingClientRect();
      setTooltip({ text: target.dataset.tooltip || "", x: rect.left + rect.width / 2, y: rect.bottom + 7 });
    }, delayMs);
  }, [delayMs, hide]);

  const onPointerOut = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    const target = event.target instanceof Element ? event.target.closest<HTMLElement>(".tooltip-button[data-tooltip]") : null;
    if (!target) return;
    const next = event.relatedTarget instanceof Node ? event.relatedTarget : null;
    if (next && target.contains(next)) return;
    hide();
  }, [hide]);

  useEffect(() => {
    window.addEventListener("blur", hide);
    window.addEventListener("resize", hide);
    return () => {
      window.removeEventListener("blur", hide);
      window.removeEventListener("resize", hide);
      if (timerRef.current !== null) window.clearTimeout(timerRef.current);
    };
  }, [hide]);

  return { tooltip, hide, onPointerOver, onPointerOut };
}
