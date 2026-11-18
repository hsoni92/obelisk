"use client";

import { Button } from "@/components/ui/button";
import { FieldPassword } from "@/components/ui/field";
import {
  rtkErrorMessage,
  useGetBlueprintStatusQuery,
  useGetRestoreChainQuery,
  useStartRestoreChainMutation,
} from "@/lib/api";
import { readPassphrase } from "@/lib/blueprint-storage";
import { toast } from "@/lib/toast";
import Link from "next/link";
import { useEffect, useState } from "react";

export function RestoreWizard() {
  const { data: blueprint } = useGetBlueprintStatusQuery(undefined, {
    pollingInterval: 5000,
  });
  const [start, { isLoading }] = useStartRestoreChainMutation();
  const [watching, setWatching] = useState(false);
  const { data: chain } = useGetRestoreChainQuery(undefined, {
    pollingInterval: watching ? 2000 : 0,
    skip: !watching,
  });
  const [passphrase, setPassphrase] = useState("");
  const [choice, setChoice] = useState<"adopt" | "keep" | "">("");

  useEffect(() => {
    const stored = readPassphrase();
    if (stored) setPassphrase(stored);
  }, []);

  useEffect(() => {
    if (chain && !chain.running && chain.phase !== "idle") {
      setWatching(false);
    }
  }, [chain]);

  const mismatch = Boolean(blueprint?.machine_id_mismatch);

  async function run(event: React.FormEvent) {
    event.preventDefault();
    if (mismatch && choice !== "adopt" && choice !== "keep") {
      toast.warning("Choose adopt or keep before restoring.");
      return;
    }
    try {
      await start({
        passphrase,
        machine_id_choice: mismatch ? (choice as "adopt" | "keep") : undefined,
        repository: blueprint?.repository,
      }).unwrap();
      setWatching(true);
      toast.info("Restore chain started");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  return (
    <form
      onSubmit={(e) => void run(e)}
      className="rounded-xl border border-border bg-card p-4 shadow-rb-2"
    >
      <p className="text-section-title">Chained restore</p>
      <p className="mt-1 text-metadata text-muted-foreground">
        Pull the GitHub blueprint, unlock it, apply settings, then restore S3{" "}
        <code className="font-mono text-[11px]">data/</code>. Settings and data
        are reported separately. Funnel is forced off at the end.{" "}
        <Link href="/platform" className="text-link">
          Blueprint sync
        </Link>
      </p>
      {blueprint?.sync_status && (
        <p className="mt-2 text-xs text-muted-foreground">
          Sync {blueprint.sync_status.replaceAll("_", " ")}
          {blueprint.generation ? ` · generation ${blueprint.generation}` : ""}
          {blueprint.last_commit_sha ? ` · ${blueprint.last_commit_sha.slice(0, 7)}` : ""}
        </p>
      )}
      {chain && chain.phase !== "idle" && (
        <p className="mt-2 text-xs" role="status">
          Phase {chain.phase.replaceAll("_", " ")}
          {chain.settings_restored ? " · settings restored" : ""}
          {chain.data_restored
            ? ` · data restored (${chain.data_objects})`
            : chain.phase === "complete"
              ? " · no data objects"
              : ""}
          {chain.error ? ` · ${chain.error}` : ""}
        </p>
      )}
      {mismatch && (
        <fieldset className="mt-3 space-y-2 text-sm">
          <legend className="text-metadata text-muted-foreground">
            Local {blueprint?.machine_id} differs from blueprint{" "}
            {blueprint?.remote_machine_id}.
          </legend>
          <label className="flex gap-2">
            <input
              type="radio"
              name="restore-machine"
              checked={choice === "adopt"}
              onChange={() => setChoice("adopt")}
            />
            Adopt blueprint id and restore that bucket.
          </label>
          <label className="flex gap-2">
            <input
              type="radio"
              name="restore-machine"
              checked={choice === "keep"}
              onChange={() => setChoice("keep")}
            />
            Keep this id. Skip data restore.
          </label>
        </fieldset>
      )}
      <div className="mt-3 flex flex-wrap items-end gap-3">
        <FieldPassword
          id="restore-passphrase"
          label="Passphrase"
          value={passphrase}
          onChange={(e) => setPassphrase(e.target.value)}
          autoComplete="current-password"
          className="min-w-56"
        />
        <Button type="submit" disabled={isLoading || passphrase.length === 0}>
          Run restore chain
        </Button>
      </div>
    </form>
  );
}
