import { STRIP_COLOR } from "@/components/connectivity/connectivity-format";
import type { ConnectivityState } from "@/lib/types";

export function HeatStrip({ cells }: { cells: ConnectivityState[] }) {
  return (
    <div role="img" aria-label="Combined availability, last 24 hours" className="flex h-4 w-full gap-px overflow-hidden rounded">
      {cells.map((s, i) => (
        <span key={i} className="h-full min-w-px flex-1" style={{ backgroundColor: STRIP_COLOR[s], opacity: s === "no_data" ? 0.7 : 1 }} />
      ))}
    </div>
  );
}

/**
 * Overview card for the gateway peer mesh. Renders nothing until the role is
 * known, and nothing on hosts that are not a gateway.
 */
