"use client";

import { Badge } from "@/components/ui/badge";
import { useGetBlueprintStatusQuery } from "@/lib/api";

export function SyncBadge() {
  const { data } = useGetBlueprintStatusQuery(undefined, { pollingInterval: 5000 });
  if (!data?.repository) return null;
  const status = data.crypto_unlocked ? data.sync_status : "push_paused";
  const variant =
    status === "synced" ? "success" : status === "failed" ? "warning" : "info";
  const label =
    status === "push_paused"
      ? "push paused"
      : status === "synced"
        ? `synced g${data.generation}`
        : status.replaceAll("_", " ");
  return <Badge variant={variant}>{label}</Badge>;
}
