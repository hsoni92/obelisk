"use client";

import { CopyButton } from "@/components/services/copy-button";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ServiceInstance } from "@/lib/api";
import {
  useAddInstanceDatabaseMutation,
  useGetInstanceDatabasesQuery,
  useRemoveInstanceDatabaseMutation,
} from "@/lib/store/obelisk-api";
import { toast } from "@/lib/toast";
import * as React from "react";

const RELATIONAL = new Set(["postgres", "mysql", "mongodb"]);

export function InstanceDatabasesPanel({
  inst,
  enabled,
  running,
}: {
  inst: ServiceInstance;
  enabled: boolean;
  running: boolean;
}) {
  const [name, setName] = React.useState("");
  if (!RELATIONAL.has(inst.type)) {
    return null;
  }
  const { data, isLoading, refetch } = useGetInstanceDatabasesQuery(
    { id: inst.id, type: inst.type },
    { skip: !enabled },
  );
  const [addDb, { isLoading: adding }] = useAddInstanceDatabaseMutation();
  const [removeDb, { isLoading: removing }] = useRemoveInstanceDatabaseMutation();

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = name.trim().toLowerCase();
    if (!trimmed) return;
    try {
      await addDb({ id: inst.id, type: inst.type, name: trimmed }).unwrap();
      setName("");
      void refetch();
      toast.success(`Database "${trimmed}" queued — reconcile will create it`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not add database");
    }
  }

  async function remove(db: string) {
    try {
      await removeDb({ id: inst.id, type: inst.type, name: db }).unwrap();
      void refetch();
      toast.success(`Removed "${db}" from this instance`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not remove");
    }
  }

  if (!enabled) {
    return (
      <p className="text-metadata text-muted-foreground">
        Enable this instance to manage logical databases.
      </p>
    );
  }

  return (
    <div className="space-y-3 py-2">
      <p className="text-metadata text-muted-foreground">
        Bootstrap database <code className="text-xs">default</code> is created
        when the instance starts. Add more databases for separate apps (
        <code className="text-xs">data</code>, <code className="text-xs">analytics</code>
        , …).
      </p>
      {isLoading ? (
        <p className="text-metadata text-muted-foreground">Loading…</p>
      ) : (
        <ul className="space-y-2">
          {(data?.connections ?? []).map((c) => (
            <li
              key={c.name}
              className="rounded-lg border border-border bg-card px-3 py-2"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <div className="text-eyebrow text-muted-foreground">
                    {c.name}
                    {c.name === (data?.bootstrap_database ?? "default")
                      ? " (bootstrap)"
                      : ""}
                  </div>
                  <code className="block break-all font-mono text-[11px]">
                    {c.url}
                  </code>
                </div>
                <div className="flex shrink-0 gap-1">
                  <CopyButton value={c.url} label={`Copy ${c.name}`} />
                  {c.name !== (data?.bootstrap_database ?? "default") ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={removing || !running}
                      onClick={() => void remove(c.name)}
                    >
                      Remove
                    </Button>
                  ) : null}
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={(e) => void submit(e)} className="flex gap-2 pt-1">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. data"
          pattern="[a-z][a-z0-9_]{0,62}"
          title="Lowercase letters, digits, underscore"
          disabled={!running || adding}
          autoComplete="off"
        />
        <Button type="submit" disabled={!running || adding || !name.trim()}>
          {adding ? "Adding…" : "Add database"}
        </Button>
      </form>
      {!running ? (
        <p className="text-metadata text-warning">
          Start the instance before adding databases.
        </p>
      ) : null}
    </div>
  );
}