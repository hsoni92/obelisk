"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SelectField } from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import {
  activityHistoryPollOptions,
  obeliskApi,
  useGetActivityHistoryQuery,
} from "@/lib/api";
import { useAppDispatch } from "@/lib/store/hooks";
import { relativeTime } from "@/lib/overview-series";
import { stateBadgeVariant } from "@/lib/service-rows";
import type { OverviewHistoryEvent } from "@/lib/types";
import { cn } from "@/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Activity } from "lucide-react";
import {
  startTransition,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

const ROW_HEIGHT = 40;
const ACTIVITY_GRID =
  "grid grid-cols-[minmax(4.5rem,auto)_minmax(0,1fr)_minmax(5rem,auto)] md:grid-cols-[minmax(4.5rem,auto)_minmax(0,1fr)_minmax(5rem,auto)_minmax(0,1fr)] gap-x-2 text-[13px]";

const RANGE_OPTIONS = [
  { value: "1", label: "Last hour" },
  { value: "6", label: "Last 6 hours" },
  { value: "24", label: "Last 24 hours" },
];

export function RecentActivityDrawer() {
  const dispatch = useAppDispatch();
  const [open, setOpen] = useState(false);
  const [hours, setHours] = useState(6);
  const [showTable, setShowTable] = useState(false);

  const queryArgs = { hours };
  const { data, isLoading, isFetching } = useGetActivityHistoryQuery(queryArgs, {
    ...activityHistoryPollOptions,
    pollingInterval: open ? activityHistoryPollOptions.pollingInterval : 0,
    skip: !open,
  });

  const events = useMemo(
    () =>
      [...(data?.history ?? [])].sort(
        (a, b) => new Date(b.at).getTime() - new Date(a.at).getTime(),
      ),
    [data?.history],
  );

  useEffect(() => {
    if (!open) {
      setShowTable(false);
      return;
    }
    const id = requestAnimationFrame(() => {
      requestAnimationFrame(() => setShowTable(true));
    });
    return () => cancelAnimationFrame(id);
  }, [open, hours, data]);

  const prefetch = () => {
    void dispatch(
      obeliskApi.endpoints.getActivityHistory.initiate(queryArgs, {
        subscribe: false,
      }),
    );
  };

  return (
    <Sheet
      open={open}
      onOpenChange={(next) => {
        startTransition(() => setOpen(next));
      }}
    >
      <SheetTrigger asChild>
        <Button
          type="button"
          variant="icon"
          size="icon"
          aria-label="Recent activity"
          title="Recent activity"
          onPointerEnter={prefetch}
          onFocus={prefetch}
        >
          <Activity className="size-4" />
        </Button>
      </SheetTrigger>
      <SheetContent className="gap-0 p-0">
        <SheetHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <SheetTitle>Recent activity</SheetTitle>
              <SheetDescription>
                Service state changes and admin actions
                {data ? ` · ${events.length} in window` : ""}
                {isFetching && !isLoading ? " · refreshing…" : ""}
              </SheetDescription>
            </div>
            <SelectField
              value={String(hours)}
              onValueChange={(v) => setHours(Number(v))}
              options={RANGE_OPTIONS}
              aria-label="Activity time range"
              className="w-[10.5rem]"
            />
          </div>
        </SheetHeader>
        {!showTable || isLoading ? (
          <ActivityTableSkeleton />
        ) : (
          <ActivityHistoryVirtualList events={events} />
        )}
      </SheetContent>
    </Sheet>
  );
}

function ActivityHistoryVirtualList({
  events,
}: {
  events: OverviewHistoryEvent[];
}) {
  const parentRef = useRef<HTMLDivElement>(null);

  const rowVirtualizer = useVirtualizer({
    count: events.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 12,
  });

  if (events.length === 0) {
    return (
      <p className="px-6 py-12 text-center text-muted-foreground">
        No transitions in this window. Changes appear as component state shifts.
      </p>
    );
  }

  return (
    <div
      ref={parentRef}
      className="min-h-0 flex-1 overflow-y-auto overscroll-contain"
      role="region"
      aria-label="Recent activity list"
    >
      <div
        className={cn(
          ACTIVITY_GRID,
          "sticky top-0 z-10 border-b border-border bg-card px-4 py-2 text-left text-eyebrow text-muted-foreground",
        )}
        role="row"
      >
        <span className="pl-2 font-semibold" role="columnheader">
          When
        </span>
        <span className="font-semibold" role="columnheader">
          Component
        </span>
        <span className="font-semibold" role="columnheader">
          Status
        </span>
        <span className="hidden font-semibold md:block" role="columnheader">
          Detail
        </span>
      </div>

      <div
        className="relative w-full"
        style={{ height: `${rowVirtualizer.getTotalSize()}px` }}
      >
        {rowVirtualizer.getVirtualItems().map((virtualRow) => {
          const ev = events[virtualRow.index];
          if (!ev) return null;
          return (
            <div
              key={`${ev.at}-${ev.key}-${virtualRow.index}`}
              data-index={virtualRow.index}
              className={cn(
                ACTIVITY_GRID,
                "absolute left-0 w-full items-center border-t border-border px-4 hover:bg-muted/40",
              )}
              style={{
                height: ROW_HEIGHT,
                transform: `translateY(${virtualRow.start}px)`,
              }}
              role="row"
            >
              <span
                className="min-w-0 truncate pl-2 text-muted-foreground"
                title={new Date(ev.at).toLocaleString()}
                role="cell"
              >
                {relativeTime(ev.at)}
              </span>
              <span className="min-w-0 truncate font-medium" role="cell">
                {ev.key}
              </span>
              <span className="min-w-0" role="cell">
                      <Badge variant={activityBadgeVariant(ev.kind, ev.state)}>
                        {ev.state}
                      </Badge>
              </span>
              <span
                className="hidden min-w-0 truncate text-metadata text-muted-foreground md:block"
                title={ev.detail ?? undefined}
                role="cell"
              >
                {ev.detail ?? "—"}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function activityBadgeVariant(
  kind: string | undefined,
  state: string,
): ReturnType<typeof stateBadgeVariant> {
  if (kind === "audit") {
    if (
      state === "completed" ||
      state === "enabled" ||
      state === "created" ||
      state === "updated"
    ) {
      return "success";
    }
    if (state === "failed" || state === "degraded") return "live";
    if (state === "disabled" || state === "deleted") return "default";
  }
  return stateBadgeVariant(state);
}

function ActivityTableSkeleton() {
  return (
    <div className="space-y-2 px-6 py-4" aria-hidden>
      {Array.from({ length: 8 }, (_, i) => (
        <Skeleton key={i} className="h-9 w-full rounded-lg" />
      ))}
    </div>
  );
}