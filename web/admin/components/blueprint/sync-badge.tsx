"use client";

import { GITHUB_SHORT_LABEL } from "@/components/blueprint/github-health-shared";
import { Badge } from "@/components/ui/badge";
import { useGetBlueprintStatusQuery } from "@/lib/api";

export function SyncBadge() {
  const { data } = useGetBlueprintStatusQuery(undefined, { pollingInterval: 5000 });
  if (!data?.repository) return null;
  if (data.github_api?.blocking) {
    return (
      <Badge variant="warning">
        {GITHUB_SHORT_LABEL[data.github_api.state] ?? "GitHub unavailable"}
      </Badge>
    );
  }
  if (data.import_required) {
    const label = data.remote_generation
      ? `conflict g${data.generation} vs g${data.remote_generation}`
      : "sync conflict";
    return <Badge variant="warning">{label}</Badge>;
  }
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
