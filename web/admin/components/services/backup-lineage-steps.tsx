"use client";

import { BackupRowActionsMenu } from "@/components/backup/backup-row-actions-menu";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { BackupLineageNode } from "@/lib/types";
import { lineageSteps } from "@/lib/backup-lineage-layout";
import { formatBytes } from "@/lib/format-bytes";
import { relativeTime } from "@/lib/overview-series";
import { cn } from "@/lib/utils";
import * as React from "react";

type Props = {
  nodes: BackupLineageNode[];
  orphan?: boolean;
  clone?: boolean;
  confirmDelete: string | null;
  confirmText: string;
  deleteBusy: boolean;
  onConfirmTextChange: (v: string) => void;
  onStartDelete: (backupId: string) => void;
  onCancelDelete: () => void;
  onDelete: (backupId: string) => void;
  confirmPhrase: string;
  rowLabel?: string;
  onCloneNode?: (node: BackupLineageNode) => void;
};

export function BackupLineageSteps({
  nodes,
  orphan,
  clone,
  confirmDelete,
  confirmText,
  deleteBusy,
  onConfirmTextChange,
  onStartDelete,
  onCancelDelete,
  onDelete,
  confirmPhrase,
  rowLabel = "backup",
  onCloneNode,
}: Props) {
  const steps = React.useMemo(() => lineageSteps(nodes), [nodes]);

  if (nodes.length === 0) {
    return (
      <p className="text-metadata text-muted-foreground py-4 text-center">
        No backups in lineage — create one to start the timeline.
      </p>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h5 className="text-eyebrow text-muted-foreground">
          Backup lineage
          <span className="ml-1.5 font-normal normal-case tracking-normal">
            · {nodes.length}
          </span>
        </h5>
        <div className="flex flex-wrap gap-1.5">
          {clone ? (
            <Badge variant="outline" className="font-normal">
              Clone entry
            </Badge>
          ) : null}
          {orphan ? (
            <Badge variant="live" className="font-normal">
              Orphan
            </Badge>
          ) : null}
        </div>
      </div>

      <ol className="max-h-72 space-y-0 overflow-y-auto pr-1">
        {steps.map(({ node, isLast }) => (
          <li key={node.id} className="flex gap-2.5">
            <StepRail active={node.active} isLast={isLast} />
            <div className="min-w-0 flex-1 pb-1.5">
              <BackupStepBubble
                node={node}
                rowLabel={rowLabel}
                deleteBusy={deleteBusy}
                onCloneNode={onCloneNode}
                onDeleteSubtree={() => onStartDelete(node.id)}
              />
              {confirmDelete === node.id ? (
                <DeleteConfirm
                  confirmPhrase={confirmPhrase}
                  confirmText={confirmText}
                  deleteBusy={deleteBusy}
                  onConfirmTextChange={onConfirmTextChange}
                  onCancel={onCancelDelete}
                  onDelete={() => onDelete(node.id)}
                />
              ) : null}
            </div>
          </li>
        ))}
      </ol>
    </div>
  );
}

function StepRail({ active, isLast }: { active: boolean; isLast: boolean }) {
  return (
    <div className="flex w-4 shrink-0 flex-col items-center pt-3" aria-hidden>
      <div
        className={cn(
          "z-10 size-2.5 shrink-0 rounded-full border-[1.5px]",
          active
            ? "border-primary bg-primary shadow-[0_0_0_3px_hsl(var(--primary)/0.22)]"
            : "border-muted-foreground/50 bg-background",
        )}
      />
      {!isLast ? (
        <div className="mt-0.5 w-px flex-1 bg-border" />
      ) : null}
    </div>
  );
}

function BackupStepBubble({
  node,
  rowLabel,
  deleteBusy,
  onCloneNode,
  onDeleteSubtree,
}: {
  node: BackupLineageNode;
  rowLabel: string;
  deleteBusy: boolean;
  onCloneNode?: (node: BackupLineageNode) => void;
  onDeleteSubtree: () => void;
}) {
  const created = new Date(node.created_at);
  const absTime = Number.isNaN(created.getTime())
    ? node.created_at
    : created.toLocaleString();

  return (
    <article
      className={cn(
        "group rounded-lg border px-3 py-2 transition-colors",
        node.active
          ? "border-primary/40 bg-primary/[0.07]"
          : "border-border/60 bg-muted/25 hover:bg-muted/50",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
          <span className="text-sm font-medium" title={absTime}>
            {relativeTime(node.created_at)}
          </span>
          <Badge
            variant="outline"
            className="h-5 px-1.5 text-[10px] font-normal capitalize"
          >
            {node.trigger}
          </Badge>
          {node.active ? (
            <Badge
              variant="success"
              className="h-5 px-1.5 text-[10px] font-normal"
            >
              Active
            </Badge>
          ) : null}
        </div>
        <BackupRowActionsMenu
          label={`${rowLabel} ${node.id.slice(0, 8)}`}
          busy={deleteBusy}
          onCloneBackup={onCloneNode ? () => onCloneNode(node) : undefined}
          onDeleteSubtree={onDeleteSubtree}
        />
      </div>

      <p
        className="mt-0.5 flex flex-wrap items-center gap-x-1.5 text-xs text-muted-foreground"
        title={`${absTime} · ${node.id}`}
      >
        <span>{formatBytes(node.size)}</span>
        <span aria-hidden>·</span>
        <span>RF {node.witness.replication_factor}</span>
        <span aria-hidden>·</span>
        <span>{node.witness.data_retention_days}d retention</span>
        <span aria-hidden>·</span>
        <span className="font-mono text-[10px]">{node.id.slice(0, 8)}</span>
      </p>
    </article>
  );
}

function DeleteConfirm({
  confirmPhrase,
  confirmText,
  deleteBusy,
  onConfirmTextChange,
  onCancel,
  onDelete,
}: {
  confirmPhrase: string;
  confirmText: string;
  deleteBusy: boolean;
  onConfirmTextChange: (v: string) => void;
  onCancel: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="mt-2 space-y-2 rounded-xl border border-border bg-card px-3 py-3">
      <p className="text-metadata text-muted-foreground">
        Cascade-deletes this backup and all descendants. Type{" "}
        <span className="font-mono uppercase">{confirmPhrase}</span> to confirm.
      </p>
      <Input
        value={confirmText}
        onChange={(e) => onConfirmTextChange(e.target.value)}
        placeholder={confirmPhrase}
        aria-label="Confirm backup delete"
      />
      <div className="flex gap-2">
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          type="button"
          variant="ink"
          size="sm"
          disabled={deleteBusy}
          onClick={onDelete}
        >
          Delete subtree
        </Button>
      </div>
    </div>
  );
}