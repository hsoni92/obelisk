export type FilterSurfaceId = "services";

export type TriState = "all" | "yes" | "no";

/** Shared shape for any list/table surface in the admin UI. */
export type SurfaceFilterState = {
  query: string;
  /** Facet key → selected values. Empty array means “no restriction”. */
  sets: Record<string, string[]>;
  /** Facet key → yes/no/all boolean-style filters. */
  tri: Record<string, TriState>;
};

export const emptySurfaceFilter = (): SurfaceFilterState => ({
  query: "",
  sets: {},
  tri: {},
});