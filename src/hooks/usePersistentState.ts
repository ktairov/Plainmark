import { useEffect, useState } from "react";

export function usePersistentState<T>(key: string, read: () => T, serialize: (value: T) => string = JSON.stringify) {
  const [value, setValue] = useState<T>(read);
  useEffect(() => { localStorage.setItem(key, serialize(value)); }, [key, value]);
  return [value, setValue] as const;
}
