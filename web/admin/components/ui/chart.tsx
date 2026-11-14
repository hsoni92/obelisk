"use client";

import { cn } from "@/lib/utils";
import * as React from "react";
import * as RechartsPrimitive from "recharts";

/** shadcn/ui chart primitives (Recharts v3). */

export type ChartConfig = {
  [k: string]: {
    label?: React.ReactNode;
    icon?: React.ComponentType;
    color?: string;
  };
};

const ChartContext = React.createContext<{ config: ChartConfig } | null>(null);

function useChart() {
  const ctx = React.useContext(ChartContext);
  if (!ctx) throw new Error("useChart must be used within <ChartContainer />");
  return ctx;
}

function ChartContainer({
  id,
  className,
  children,
  config,
  ...props
}: React.ComponentProps<"div"> & {
  config: ChartConfig;
  children: React.ComponentProps<
    typeof RechartsPrimitive.ResponsiveContainer
  >["children"];
}) {
  const uid = React.useId();
  const chartId = `chart-${id || uid.replace(/:/g, "")}`;
  return (
    <ChartContext.Provider value={{ config }}>
      <div
        data-slot="chart"
        data-chart={chartId}
        className={cn(
          "flex aspect-video justify-center text-xs",
          "[&_.recharts-cartesian-axis-tick_text]:fill-muted-foreground",
          "[&_.recharts-cartesian-grid_line[stroke='#ccc']]:stroke-border/60",
          "[&_.recharts-curve.recharts-tooltip-cursor]:stroke-border",
          "[&_.recharts-dot[stroke='#fff']]:stroke-transparent",
          "[&_.recharts-layer]:outline-hidden [&_.recharts-sector]:outline-hidden [&_.recharts-surface]:outline-hidden",
          className,
        )}
        {...props}
      >
        <ChartStyle id={chartId} config={config} />
        <RechartsPrimitive.ResponsiveContainer>
          {children}
        </RechartsPrimitive.ResponsiveContainer>
      </div>
    </ChartContext.Provider>
  );
}

function ChartStyle({ id, config }: { id: string; config: ChartConfig }) {
  const colors = Object.entries(config).filter(([, c]) => c.color);
  if (!colors.length) return null;
  return (
    <style
      dangerouslySetInnerHTML={{
        __html: `[data-chart=${id}] {\n${colors
          .map(([key, c]) => `  --color-${key}: ${c.color};`)
          .join("\n")}\n}`,
      }}
    />
  );
}

const ChartTooltip = RechartsPrimitive.Tooltip;

type TooltipPayloadItem = {
  dataKey?: string | number;
  name?: string | number;
  value?: string | number;
  color?: string;
  payload?: Record<string, unknown> & { fill?: string };
};

function ChartTooltipContent({
  active,
  payload,
  label,
  className,
  labelFormatter,
  hideLabel = false,
  nameKey,
}: {
  active?: boolean;
  payload?: TooltipPayloadItem[];
  label?: React.ReactNode;
  className?: string;
  labelFormatter?: (label: React.ReactNode) => React.ReactNode;
  hideLabel?: boolean;
  nameKey?: string;
}) {
  const { config } = useChart();
  if (!active || !payload?.length) return null;

  return (
    <div
      className={cn(
        "grid min-w-[8rem] gap-1.5 rounded-lg border border-border bg-popover px-2.5 py-1.5 text-xs shadow-rb-2",
        className,
      )}
    >
      {!hideLabel && label != null && (
        <div className="font-medium">
          {labelFormatter ? labelFormatter(label) : label}
        </div>
      )}
      <div className="grid gap-1">
        {payload.map((item, i) => {
          const key = String(
            (nameKey && (item.payload?.[nameKey] as string)) ||
              item.name ||
              item.dataKey ||
              "value",
          );
          const cfg = config[key];
          const color = item.payload?.fill || item.color;
          return (
            <div key={`${key}-${i}`} className="flex items-center gap-2">
              <span
                className="size-2 shrink-0 rounded-full"
                style={{ background: color }}
              />
              <span className="text-muted-foreground">{cfg?.label ?? key}</span>
              <span className="ml-auto font-mono font-medium tabular-nums">
                {item.value}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export {
  ChartContainer,
  ChartStyle,
  ChartTooltip,
  ChartTooltipContent,
  useChart,
};
