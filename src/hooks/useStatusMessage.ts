import { useCallback, useEffect, useRef, useState } from "react";

export type StatusTone = "info" | "success" | "error";
export type StatusMessage = { text: string; tone: StatusTone };

export function useStatusMessage() {
  const [message, setMessage] = useState<StatusMessage | null>(null);
  const timerRef = useRef<number | null>(null);

  const clear = useCallback(() => {
    if (timerRef.current !== null) window.clearTimeout(timerRef.current);
    timerRef.current = null;
    setMessage(null);
  }, []);

  const show = useCallback((text: string, tone: StatusTone = "info", timeoutMs = tone === "error" ? 0 : 2400) => {
    if (timerRef.current !== null) window.clearTimeout(timerRef.current);
    timerRef.current = null;
    setMessage({ text, tone });
    if (timeoutMs > 0) timerRef.current = window.setTimeout(clear, timeoutMs);
  }, [clear]);

  useEffect(() => () => {
    if (timerRef.current !== null) window.clearTimeout(timerRef.current);
  }, []);

  return { message, show, clear };
}
