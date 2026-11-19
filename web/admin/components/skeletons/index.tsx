import { Card } from "@/components/ui/card";
import { Skeleton, SkeletonRegion } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

/** Mirrors <PageHeader />: 40px title + one-line subtitle. */
export function PageHeaderSkeleton({ action = false }: { action?: boolean }) {
  return (
    <div className="mb-8 flex items-end justify-between gap-4" aria-hidden>
      <div className="space-y-3">
        <Skeleton className="h-10 w-56 rounded-xl" />
        <Skeleton className="h-5 w-80 max-w-full" />
      </div>
      {action && <Skeleton className="h-10 w-36 rounded-full" />}
    </div>
  );
}

/** A run of `rows` list/table lines, each made of `cols` blocks. */
export function SkeletonRows({
  rows = 4,
  className,
}: {
  rows?: number;
  className?: string;
}) {
  return (
    <div className={cn("divide-y divide-border", className)} aria-hidden>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="flex items-center gap-3 px-4 py-3">
          <Skeleton className="size-8 shrink-0 rounded-full" />
          <Skeleton className="h-4 w-40 max-w-[40%]" />
          <Skeleton className="ml-auto h-4 w-24" />
          <Skeleton className="hidden h-5 w-16 rounded-full sm:block" />
        </div>
      ))}
    </div>
  );
}

function CardTitleSkeleton({ subtitle = true }: { subtitle?: boolean }) {
  return (
    <div className="space-y-2" aria-hidden>
      <Skeleton className="h-4 w-36" />
      {subtitle && <Skeleton className="h-3 w-52 max-w-full" />}
    </div>
  );
}

export function KpiCardSkeleton() {
  return (
    <Card className="p-4" aria-hidden>
      <div className="flex items-start justify-between">
        <Skeleton className="size-8 rounded-full" />
        <Skeleton className="h-5 w-14 rounded-full" />
      </div>
      <Skeleton className="mt-4 h-3 w-24" />
      <Skeleton className="mt-2 h-9 w-20 rounded-xl" />
      <Skeleton className="mt-2 h-3 w-32" />
    </Card>
  );
}

export function BackupOverviewSkeleton() {
  return (
    <Card className="overflow-hidden" aria-hidden>
      <div className="flex items-center justify-between gap-2 p-4 pb-3">
        <div className="flex items-center gap-3">
          <Skeleton className="size-8 rounded-full" />
          <CardTitleSkeleton />
        </div>
        <Skeleton className="h-9 w-20 rounded-full" />
      </div>
      <div className="grid grid-cols-2 gap-4 border-t border-border p-4 md:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <div key={i} className="space-y-2">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-5 w-24" />
            <Skeleton className="h-3 w-20" />
          </div>
        ))}
      </div>
      <SkeletonRows rows={2} className="border-t border-border" />
    </Card>
  );
}

/** Whole Overview page body (everything under the page header). */
export function OverviewSkeleton() {
  return (
    <SkeletonRegion label="Loading overview" className="space-y-4">
      <div className="flex items-center gap-2" aria-hidden>
        <Skeleton className="h-6 w-16 rounded-full" />
        <Skeleton className="h-6 w-20 rounded-full" />
        <Skeleton className="ml-auto h-10 w-36 rounded-full" />
      </div>
      <Skeleton className="h-16 w-full rounded-2xl" />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <KpiCardSkeleton key={i} />
        ))}
      </div>
      <div className="grid gap-3 lg:grid-cols-3" aria-hidden>
        <Card className="p-4 lg:col-span-2">
          <CardTitleSkeleton />
          <Skeleton className="mt-4 h-56 w-full rounded-xl" />
        </Card>
        <Card className="p-4">
          <CardTitleSkeleton />
          <Skeleton className="mx-auto mt-4 size-[150px] rounded-full" />
          <div className="mt-4 space-y-2">
            {Array.from({ length: 4 }, (_, i) => (
              <Skeleton key={i} className="h-3 w-full" />
            ))}
          </div>
        </Card>
      </div>
      <BackupOverviewSkeleton />
    </SkeletonRegion>
  );
}

/** One gateway (OpenRouter / Mail) section on Exposure. */
export function GatewayCardSkeleton() {
  return (
    <SkeletonRegion
      label="Loading gateway"
      className="rounded-xl border border-border/70 bg-card p-4"
    >
      <div className="flex items-start gap-3" aria-hidden>
        <Skeleton className="size-9 rounded-lg" />
        <div className="flex-1 space-y-2">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-56 max-w-full" />
        </div>
        <Skeleton className="h-5 w-9 rounded-full" />
      </div>
      <div className="mt-4 space-y-3" aria-hidden>
        <Skeleton className="h-8 w-full rounded-md" />
        <Skeleton className="h-8 w-full rounded-md" />
        <Skeleton className="h-16 w-full rounded-lg" />
      </div>
    </SkeletonRegion>
  );
}

/** Disaster Recovery: policy card + destinations card. */
export function DRSkeleton() {
  return (
    <SkeletonRegion label="Loading backup configuration" className="space-y-4">
      <Card className="space-y-4 p-4" aria-hidden>
        <CardTitleSkeleton />
        <div className="grid gap-3 sm:grid-cols-3">
          {Array.from({ length: 3 }, (_, i) => (
            <div key={i} className="space-y-2">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-10 w-full rounded-full" />
            </div>
          ))}
        </div>
        <Skeleton className="h-10 w-24 rounded-full" />
      </Card>
      <Card className="space-y-4 p-4" aria-hidden>
        <div className="flex items-center justify-between">
          <Skeleton className="h-5 w-48" />
          <div className="flex gap-2">
            <Skeleton className="h-8 w-20 rounded-full" />
            <Skeleton className="h-8 w-36 rounded-full" />
          </div>
        </div>
        <SkeletonRows rows={3} />
      </Card>
    </SkeletonRegion>
  );
}

/** Platform → Blueprint card. */
export function BlueprintSkeleton() {
  return (
    <SkeletonRegion label="Loading blueprint" className="mt-8 max-w-3xl">
      <div className="space-y-2" aria-hidden>
        <Skeleton className="h-6 w-28" />
        <Skeleton className="h-4 w-80 max-w-full" />
      </div>
      <Card className="mt-4 p-4" aria-hidden>
        <div className="grid gap-4 sm:grid-cols-2">
          {Array.from({ length: 4 }, (_, i) => (
            <div key={i} className="space-y-2">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-4 w-32" />
            </div>
          ))}
        </div>
        <div className="mt-5 flex gap-2">
          <Skeleton className="h-10 w-24 rounded-full" />
          <Skeleton className="h-10 w-24 rounded-full" />
        </div>
      </Card>
    </SkeletonRegion>
  );
}

/** Sidebar + header + generic content. Shown while the session is being verified. */
export function ConsoleShellSkeleton() {
  return (
    <SkeletonRegion label="Loading console" className="flex min-h-screen">
      <aside
        className="sticky top-0 hidden h-screen w-60 shrink-0 flex-col py-5 pl-5 md:flex"
        aria-hidden
      >
        <div className="flex items-center gap-2.5 px-1">
          <Skeleton className="size-8 rounded-full" />
          <div className="space-y-1.5">
            <Skeleton className="h-3.5 w-16" />
            <Skeleton className="h-2.5 w-24" />
          </div>
        </div>
        <div className="mt-6 flex flex-col gap-1 rounded-[28px] bg-card p-2 shadow-rb-2">
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} className="h-10 w-full rounded-full" />
          ))}
        </div>
        <Skeleton className="mt-auto h-24 w-full rounded-[28px]" />
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex h-16 items-center gap-3 px-6" aria-hidden>
          <Skeleton className="size-10 rounded-full" />
          <Skeleton className="h-10 w-full max-w-md rounded-full" />
        </div>
        <div className="mx-auto w-full max-w-[1400px] flex-1 px-6 pb-10 pt-4">
          <PageHeaderSkeleton />
          <div className="space-y-4" aria-hidden>
            <Skeleton className="h-10 w-28 rounded-full" />
            <Card className="overflow-hidden">
              <SkeletonRows rows={6} />
            </Card>
          </div>
        </div>
      </div>
    </SkeletonRegion>
  );
}

/** Auth pages (login callback): brand + card. */
export function AuthSkeleton({ caption }: { caption?: string }) {
  return (
    <SkeletonRegion
      label={caption ?? "Loading"}
      className="flex min-h-screen items-center justify-center px-4 py-10"
    >
      <div className="w-full max-w-md" aria-hidden>
        <div className="flex items-center gap-2.5">
          <Skeleton className="size-8 rounded-full" />
          <div className="space-y-1.5">
            <Skeleton className="h-3.5 w-16" />
            <Skeleton className="h-2.5 w-24" />
          </div>
        </div>
        <Skeleton className="mt-6 h-6 w-48" />
        <Skeleton className="mt-2 h-4 w-72 max-w-full" />
        <Card className="mt-6 space-y-3 p-5">
          <Skeleton className="h-10 w-44 rounded-full" />
          <Skeleton className="h-3 w-full" />
          <Skeleton className="h-3 w-2/3" />
        </Card>
        {caption && (
          <p className="mt-3 text-center text-metadata text-muted-foreground">
            {caption}
          </p>
        )}
      </div>
    </SkeletonRegion>
  );
}
