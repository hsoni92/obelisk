import type { BackupLineageNode } from "@/lib/types";

function byCreated(a: BackupLineageNode, b: BackupLineageNode) {
  return (
    new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  );
}

export type LineageStepItem = {
  node: BackupLineageNode;
  /** Parent depth (0 = root snapshot). */
  depth: number;
  /** 1-based step index in display order. */
  step: number;
  isLast: boolean;
};

/** Depth-first walk: roots → children, siblings oldest first (vertical lineage). */
export function lineageSteps(nodes: BackupLineageNode[]): LineageStepItem[] {
  if (nodes.length === 0) {
    return [];
  }
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const children = new Map<string, BackupLineageNode[]>();
  for (const n of nodes) {
    const parent = n.parent_id?.trim() || "";
    const list = children.get(parent) ?? [];
    list.push(n);
    children.set(parent, list);
  }
  for (const list of children.values()) {
    list.sort(byCreated);
  }
  const roots = nodes
    .filter((n) => !n.parent_id?.trim() || !byId.has(n.parent_id))
    .sort(byCreated);

  const out: Omit<LineageStepItem, "step" | "isLast">[] = [];
  const walk = (n: BackupLineageNode, depth: number) => {
    out.push({ node: n, depth });
    for (const child of children.get(n.id) ?? []) {
      walk(child, depth + 1);
    }
  };
  for (const root of roots) {
    walk(root, 0);
  }
  const seen = new Set(out.map((x) => x.node.id));
  const detached = nodes.filter((n) => !seen.has(n.id)).sort(byCreated);
  for (const n of detached) {
    walk(n, 0);
  }
  const total = out.length;
  return out.map((item, i) => ({
    ...item,
    step: i + 1,
    isLast: i === total - 1,
  }));
}