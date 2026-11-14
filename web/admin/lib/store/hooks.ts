import type { FilterSurfaceId, TriState } from "@/lib/filters/types";
import { countActiveFilters } from "@/lib/filters/engine";
import {
  clearSetFacet,
  resetSurface,
  selectSurfaceFilterDefaults,
  selectSurfaceFilters,
  setQuery,
  setSetValues,
  setTri,
  toggleSetValue,
} from "@/lib/store/filters-slice";
import { useDispatch, useSelector } from "react-redux";
import type { RootState, AppDispatch } from "./store";

export const useAppDispatch = useDispatch.withTypes<AppDispatch>();
export const useAppSelector = useSelector.withTypes<RootState>();

export function useSurfaceFilters(surface: FilterSurfaceId) {
  const dispatch = useAppDispatch();
  const filters = useAppSelector((s) => selectSurfaceFilters(s, surface));
  const defaults = selectSurfaceFilterDefaults(surface);
  const activeCount = countActiveFilters(filters, defaults);

  return {
    filters,
    defaults,
    activeCount,
    setQuery: (query: string) => dispatch(setQuery({ surface, query })),
    setSetValues: (key: string, values: string[]) =>
      dispatch(setSetValues({ surface, key, values })),
    toggleSetValue: (key: string, value: string) =>
      dispatch(toggleSetValue({ surface, key, value })),
    setTri: (key: string, value: TriState) =>
      dispatch(setTri({ surface, key, value })),
    clearSetFacet: (key: string) =>
      dispatch(clearSetFacet({ surface, key })),
    reset: () => dispatch(resetSurface(surface)),
  };
}