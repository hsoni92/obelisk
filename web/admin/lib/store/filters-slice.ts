import {
  defaultServicesFilters,
  SERVICES_SURFACE,
} from "@/lib/filters/services-config";
import type { FilterSurfaceId, SurfaceFilterState, TriState } from "@/lib/filters/types";
import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

const DEFAULTS: Record<FilterSurfaceId, SurfaceFilterState> = {
  [SERVICES_SURFACE]: defaultServicesFilters(),
};

function cloneState(state: SurfaceFilterState): SurfaceFilterState {
  return {
    query: state.query,
    sets: Object.fromEntries(
      Object.entries(state.sets).map(([k, v]) => [k, [...v]]),
    ),
    tri: { ...state.tri },
  };
}

type FiltersState = {
  surfaces: Record<FilterSurfaceId, SurfaceFilterState>;
};

const initialState: FiltersState = {
  surfaces: {
    [SERVICES_SURFACE]: defaultServicesFilters(),
  },
};

const filtersSlice = createSlice({
  name: "filters",
  initialState,
  reducers: {
    setQuery(
      state,
      action: PayloadAction<{ surface: FilterSurfaceId; query: string }>,
    ) {
      const { surface, query } = action.payload;
      if (!state.surfaces[surface]) {
        state.surfaces[surface] = cloneState(
          DEFAULTS[surface] ?? { query: "", sets: {}, tri: {} },
        );
      }
      state.surfaces[surface].query = query;
    },
    setSetValues(
      state,
      action: PayloadAction<{
        surface: FilterSurfaceId;
        key: string;
        values: string[];
      }>,
    ) {
      const { surface, key, values } = action.payload;
      if (!state.surfaces[surface]) {
        state.surfaces[surface] = cloneState(DEFAULTS[surface]);
      }
      state.surfaces[surface].sets[key] = values;
    },
    toggleSetValue(
      state,
      action: PayloadAction<{
        surface: FilterSurfaceId;
        key: string;
        value: string;
      }>,
    ) {
      const { surface, key, value } = action.payload;
      if (!state.surfaces[surface]) {
        state.surfaces[surface] = cloneState(DEFAULTS[surface]);
      }
      const current = state.surfaces[surface].sets[key] ?? [];
      const next = current.includes(value)
        ? current.filter((v) => v !== value)
        : [...current, value];
      state.surfaces[surface].sets[key] = next;
    },
    setTri(
      state,
      action: PayloadAction<{
        surface: FilterSurfaceId;
        key: string;
        value: TriState;
      }>,
    ) {
      const { surface, key, value } = action.payload;
      if (!state.surfaces[surface]) {
        state.surfaces[surface] = cloneState(DEFAULTS[surface]);
      }
      state.surfaces[surface].tri[key] = value;
    },
    clearSetFacet(
      state,
      action: PayloadAction<{ surface: FilterSurfaceId; key: string }>,
    ) {
      const { surface, key } = action.payload;
      if (!state.surfaces[surface]) return;
      state.surfaces[surface].sets[key] = [];
    },
    resetSurface(state, action: PayloadAction<FilterSurfaceId>) {
      const surface = action.payload;
      state.surfaces[surface] = cloneState(
        DEFAULTS[surface] ?? { query: "", sets: {}, tri: {} },
      );
    },
  },
});

export const {
  setQuery,
  setSetValues,
  toggleSetValue,
  setTri,
  clearSetFacet,
  resetSurface,
} = filtersSlice.actions;

export const filtersReducer = filtersSlice.reducer;

export function selectSurfaceFilters(
  state: { filters: FiltersState },
  surface: FilterSurfaceId,
): SurfaceFilterState {
  return state.filters.surfaces[surface] ?? DEFAULTS[surface];
}

export function selectSurfaceFilterDefaults(surface: FilterSurfaceId) {
  return DEFAULTS[surface];
}