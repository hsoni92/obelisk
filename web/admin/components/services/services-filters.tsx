"use client";

import {
  FilterActiveSummary,
  type ActiveFilterChip,
} from "@/components/filters/filter-active-summary";
import { FacetRow } from "@/components/filters/facet-row";
import { Button } from "@/components/ui/button";
import { MultiSelect } from "@/components/ui/multi-select";
import { SelectField } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  SERVICE_STATES,
  SERVICE_TYPE_LABEL,
  SERVICE_TYPES,
  SERVICES_SURFACE,
} from "@/lib/filters";
import { useSurfaceFilters } from "@/lib/store/hooks";
import { SlidersHorizontal } from "lucide-react";
import { useMemo, useState } from "react";

type Tri = "all" | "yes" | "no";
type Draft = {
  types: string[];
  states: string[];
  enabled: Tri;
  endpoint: Tri;
};

const ENABLED_OPTIONS = [
  { value: "all", label: "All" },
  { value: "yes", label: "Enabled" },
  { value: "no", label: "Disabled" },
] as const;

const ENDPOINT_OPTIONS = [
  { value: "all", label: "All" },
  { value: "yes", label: "Exposed" },
  { value: "no", label: "Not exposed" },
] as const;

export function ServicesFilters() {
  const {
    filters,
    activeCount,
    setSetValues,
    setTri,
    reset,
    setQuery,
  } = useSurfaceFilters(SERVICES_SURFACE);
  const [open, setOpen] = useState(false);

  const enabledTri = filters.tri.enabled ?? "all";
  const endpointTri = filters.tri.endpoint ?? "all";

  const chips = useMemo((): ActiveFilterChip[] => {
    const typeValues = filters.sets.types ?? [];
    const stateValues = filters.sets.states ?? [];
    const out: ActiveFilterChip[] = [];
    if (filters.query.trim()) {
      out.push({
        id: "query",
        label: `Search: “${filters.query.trim()}”`,
        onRemove: () => setQuery(""),
      });
    }
    for (const t of typeValues) {
      out.push({
        id: `type-${t}`,
        label: SERVICE_TYPE_LABEL[t as keyof typeof SERVICE_TYPE_LABEL] ?? t,
        onRemove: () =>
          setSetValues(
            "types",
            typeValues.filter((v) => v !== t),
          ),
      });
    }
    for (const s of stateValues) {
      out.push({
        id: `state-${s}`,
        label: `Status: ${s}`,
        onRemove: () =>
          setSetValues(
            "states",
            stateValues.filter((v) => v !== s),
          ),
      });
    }
    if (enabledTri !== "all") {
      out.push({
        id: "enabled",
        label: enabledTri === "yes" ? "Desired: enabled" : "Desired: disabled",
        onRemove: () => setTri("enabled", "all"),
      });
    }
    if (endpointTri !== "all") {
      out.push({
        id: "endpoint",
        label:
          endpointTri === "yes" ? "Endpoint: exposed" : "Endpoint: not exposed",
        onRemove: () => setTri("endpoint", "all"),
      });
    }
    return out;
  }, [
    enabledTri,
    endpointTri,
    filters.query,
    filters.sets.states,
    filters.sets.types,
    setQuery,
    setSetValues,
    setTri,
  ]);

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setOpen(true)}
        >
          <SlidersHorizontal className="size-3.5" aria-hidden />
          Filters
          {activeCount > 0 && (
            <span className="ml-1 rounded-full bg-broadcast-tint px-1.5 text-[10px] font-semibold text-foreground">
              {activeCount}
            </span>
          )}
        </Button>
      </div>

      <FilterActiveSummary
        chips={chips}
        activeCount={activeCount}
        onClearAll={reset}
      />

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-md">
          {/* Unmounts on close, so the draft re-seeds from the store each open. */}
          <FilterForm
            initial={{
              types: filters.sets.types ?? [],
              states: filters.sets.states ?? [],
              enabled: enabledTri as Tri,
              endpoint: endpointTri as Tri,
            }}
            onCancel={() => setOpen(false)}
            onApply={(d) => {
              setSetValues("types", d.types);
              setSetValues("states", d.states);
              setTri("enabled", d.enabled);
              setTri("endpoint", d.endpoint);
              setOpen(false);
            }}
          />
        </DialogContent>
      </Dialog>
    </div>
  );
}

const EMPTY: Draft = { types: [], states: [], enabled: "all", endpoint: "all" };

function FilterForm({
  initial,
  onApply,
  onCancel,
}: {
  initial: Draft;
  onApply: (d: Draft) => void;
  onCancel: () => void;
}) {
  const [d, setD] = useState<Draft>(initial);
  const dirty = JSON.stringify(d) !== JSON.stringify(EMPTY);

  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        onApply(d);
      }}
    >
      <DialogHeader>
        <DialogTitle>Filter services</DialogTitle>
        <DialogDescription>
          Narrow the table. Leave a group empty to include everything.
        </DialogDescription>
      </DialogHeader>

      <FacetRow label="Type">
        <MultiSelect
          aria-label="Type"
          options={SERVICE_TYPES.map((t) => ({
            value: t,
            label: SERVICE_TYPE_LABEL[t],
          }))}
          value={d.types}
          onValueChange={(v) => setD((x) => ({ ...x, types: v }))}
          noun="types"
        />
      </FacetRow>

      <FacetRow label="Status">
        <MultiSelect
          aria-label="Status"
          options={SERVICE_STATES.map((s) => ({ value: s, label: s }))}
          value={d.states}
          onValueChange={(v) => setD((x) => ({ ...x, states: v }))}
          noun="statuses"
        />
      </FacetRow>

      <FacetRow label="Desired">
        <SelectField
          aria-label="Desired"
          options={ENABLED_OPTIONS}
          value={d.enabled}
          onValueChange={(v) => setD((x) => ({ ...x, enabled: v as Tri }))}
        />
      </FacetRow>

      <FacetRow label="Endpoint">
        <SelectField
          aria-label="Endpoint"
          options={ENDPOINT_OPTIONS}
          value={d.endpoint}
          onValueChange={(v) => setD((x) => ({ ...x, endpoint: v as Tri }))}
        />
        <p className="mt-1.5 text-metadata text-muted-foreground">
          “Exposed” means running with a connection string available to copy.
        </p>
      </FacetRow>

      <DialogFooter className="items-center">
        <Button
          type="button"
          variant="link"
          className="mr-auto"
          disabled={!dirty}
          onClick={() => setD(EMPTY)}
        >
          Reset all
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">Apply filters</Button>
      </DialogFooter>
    </form>
  );
}
