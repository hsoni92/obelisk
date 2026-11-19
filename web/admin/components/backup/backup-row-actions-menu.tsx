"use client";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { EllipsisVertical } from "lucide-react";

type Props = {
  label: string;
  disabled?: boolean;
  busy?: boolean;
  onViewLineage?: () => void;
  onCreateBackup?: () => void;
  onDeleteSubtree?: () => void;
  onDeleteAll?: () => void;
  onCloneBackup?: () => void;
  /** Orphan forests: delete is the primary recovery action */
  orphan?: boolean;
};

export function BackupRowActionsMenu({
  label,
  disabled,
  busy,
  onViewLineage,
  onCreateBackup,
  onDeleteSubtree,
  onDeleteAll,
  onCloneBackup,
  orphan,
}: Props) {
  const hasView = !!onViewLineage;
  const hasCreate = !!onCreateBackup;
  const hasClone = !!onCloneBackup;
  const hasSubtree = !!onDeleteSubtree;
  const hasDeleteAll = !!onDeleteAll;

  if (!hasView && !hasCreate && !hasClone && !hasSubtree && !hasDeleteAll) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="icon"
          size="icon"
          className={cn(
            "size-8 shrink-0 bg-card ring-1 ring-border shadow-none hover:bg-muted",
          )}
          disabled={disabled || busy}
          aria-label={`Actions for ${label}`}
        >
          <EllipsisVertical className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        {hasView ? (
          <DropdownMenuItem disabled={busy} onSelect={() => onViewLineage?.()}>
            View lineage
          </DropdownMenuItem>
        ) : null}
        {hasCreate ? (
          <DropdownMenuItem disabled={busy} onSelect={() => onCreateBackup?.()}>
            Create backup
          </DropdownMenuItem>
        ) : null}
        {hasClone ? (
          <DropdownMenuItem disabled={busy} onSelect={() => onCloneBackup?.()}>
            Clone backup entry…
          </DropdownMenuItem>
        ) : null}
        {(hasView || hasCreate || hasClone) && (hasSubtree || hasDeleteAll) ? (
          <DropdownMenuSeparator />
        ) : null}
        {hasSubtree ? (
          <DropdownMenuItem
            disabled={busy}
            className="text-destructive focus:text-destructive"
            onSelect={() => onDeleteSubtree?.()}
          >
            Delete subtree…
          </DropdownMenuItem>
        ) : null}
        {hasDeleteAll ? (
          <DropdownMenuItem
            disabled={busy}
            className="text-destructive focus:text-destructive"
            onSelect={() => onDeleteAll?.()}
          >
            {orphan ? "Delete orphan backups…" : "Delete all backups…"}
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}