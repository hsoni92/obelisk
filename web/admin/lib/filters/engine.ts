import type { SurfaceFilterState, TriState } from "./types";

export type TextSearchRule<T> = {
  getHaystack: (item: T) => string;
};

export type FacetSetRule<T> = {
  key: string;
  getValues: (item: T) => string | readonly string[];
};

export type FacetTriRule<T> = {
  key: string;
  test: (item: T) => boolean;
};

export type FilterConfig<T> = {
  text?: TextSearchRule<T>;
  sets?: FacetSetRule<T>[];
  tris?: FacetTriRule<T>[];
};

function matchesSet(
  itemValues: string | readonly string[],
  selected: string[],
): boolean {
  if (selected.length === 0) return true;
  const values = typeof itemValues === "string" ? [itemValues] : [...itemValues];
  return values.some((v) => selected.includes(v));
}

function matchesTri(test: boolean, mode: TriState): boolean {
  if (mode === "all") return true;
  return mode === "yes" ? test : !test;
}

export function applyFilters<T>(
  items: T[],
  state: SurfaceFilterState,
  config: FilterConfig<T>,
): T[] {
  let result = items;

  const q = state.query.trim().toLowerCase();
  if (q && config.text) {
    result = result.filter((item) =>
      config.text!.getHaystack(item).toLowerCase().includes(q),
    );
  }

  for (const rule of config.sets ?? []) {
    const selected = state.sets[rule.key] ?? [];
    if (selected.length === 0) continue;
    result = result.filter((item) =>
      matchesSet(rule.getValues(item), selected),
    );
  }

  for (const rule of config.tris ?? []) {
    const mode = state.tri[rule.key] ?? "all";
    if (mode === "all") continue;
    result = result.filter((item) => matchesTri(rule.test(item), mode));
  }

  return result;
}

export function countActiveFilters(
  state: SurfaceFilterState,
  baseline: SurfaceFilterState,
): number {
  let n = 0;
  if (state.query.trim() !== baseline.query.trim()) n++;

  const setKeys = new Set([
    ...Object.keys(baseline.sets),
    ...Object.keys(state.sets),
  ]);
  for (const key of setKeys) {
    const cur = state.sets[key] ?? [];
    if (cur.length > 0) n += cur.length;
  }

  const triKeys = new Set([
    ...Object.keys(baseline.tri),
    ...Object.keys(state.tri),
  ]);
  for (const key of triKeys) {
    const cur = state.tri[key] ?? "all";
    const base = baseline.tri[key] ?? "all";
    if (cur !== base) n++;
  }

  return n;
}