export { applyFilters, countActiveFilters } from "./engine";
export type { FilterConfig, FacetSetRule, FacetTriRule, TextSearchRule } from "./engine";
export {
  SERVICES_SURFACE,
  SERVICE_TYPES,
  SERVICE_STATES,
  SERVICE_TYPE_LABEL,
  defaultServicesFilters,
  servicesFilterConfig,
} from "./services-config";
export type { FilterSurfaceId, SurfaceFilterState, TriState } from "./types";