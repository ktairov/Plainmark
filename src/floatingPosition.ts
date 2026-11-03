export type FloatingPosition = { x: number; y: number };

export function fitFloatingPosition(
  anchor: FloatingPosition,
  size: { width: number; height: number },
  viewport: { width: number; height: number },
  margin = 8,
): FloatingPosition {
  const maxX = Math.max(margin, viewport.width - size.width - margin);
  const maxY = Math.max(margin, viewport.height - size.height - margin);
  return {
    x: Math.min(Math.max(anchor.x, margin), maxX),
    y: Math.min(Math.max(anchor.y, margin), maxY),
  };
}
