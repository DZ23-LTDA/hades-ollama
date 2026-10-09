// reorderById moves the item identified by draggedId to the position currently
// held by targetId, preserving the order of the rest. Pure + order-stable, so
// the Studio canvas drag-and-drop behaves deterministically and is unit-tested.
export function reorderById<T extends { id: string }>(
  list: readonly T[],
  draggedId: string,
  targetId: string,
): T[] {
  if (draggedId === targetId) return [...list];
  const from = list.findIndex((item) => item.id === draggedId);
  const to = list.findIndex((item) => item.id === targetId);
  if (from === -1 || to === -1) return [...list];
  const next = [...list];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next;
}
