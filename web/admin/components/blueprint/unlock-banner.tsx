"use client";

import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import {
  rtkErrorMessage,
  useGetBlueprintStatusQuery,
  useUnlockBlueprintMutation,
} from "@/lib/api";
import { readPassphrase, writePassphrase } from "@/lib/blueprint-storage";
import { toast } from "@/lib/toast";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";

export function UnlockBanner() {
  const pathname = usePathname();
  const { data } = useGetBlueprintStatusQuery(undefined, { pollingInterval: 5000 });
  const [unlock, { isLoading }] = useUnlockBlueprintMutation();
  const [passphrase, setPassphrase] = useState("");
  const tried = useRef(false);

  useEffect(() => {
    if (!data || data.crypto_unlocked || !data.repository || tried.current) return;
    const stored = readPassphrase();
    if (!stored) return;
    tried.current = true;
    unlock({ passphrase: stored })
      .unwrap()
      .catch((e) => toast.error(rtkErrorMessage(e)));
  }, [data, unlock]);

  if (!data?.repository) return null;

  if (data.crypto_unlocked) {
    if (!data.import_required || pathname === "/platform") return null;
    return (
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-warning/30 bg-warning-tint p-4 shadow-rb-2">
        <div>
          <p className="text-section-title">Blueprint sync conflict</p>
          <p className="mt-1 text-metadata text-foreground/85">
            {data.remote_generation
              ? `GitHub (${data.repository}) is at generation ${data.remote_generation}, ahead of this appliance (generation ${data.generation}). `
              : `GitHub (${data.repository}) is ahead of this appliance (generation ${data.generation}). `}
            Choose whether to import from GitHub or overwrite GitHub with this appliance.
          </p>
        </div>
        <Button asChild size="sm">
          <Link href="/platform#blueprint">Resolve sync conflict</Link>
        </Button>
      </div>
    );
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    try {
      await unlock({ passphrase }).unwrap();
      writePassphrase(passphrase);
      setPassphrase("");
      toast.success("Blueprint unlocked");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  return (
    <form
      onSubmit={(e) => void submit(e)}
      className="mb-4 rounded-xl border border-border bg-card p-4 shadow-rb-2"
    >
      <p className="text-section-title">Blueprint push is paused</p>
      <p className="mt-1 text-metadata text-muted-foreground">
        The controller restarted or was locked. Local saves still work. Enter
        the passphrase to resume GitHub sync.
        {data.import_required
          ? data.remote_generation
            ? ` GitHub is at generation ${data.remote_generation} (ahead of local ${data.generation}).`
            : " GitHub is ahead of this appliance — resolve in Platform → Blueprint after unlocking."
          : ""}
      </p>
      {data.last_push_error && !data.import_required && (
        <p className="mt-2 text-xs text-muted-foreground">{data.last_push_error}</p>
      )}
      <div className="mt-3 flex flex-wrap items-end gap-3">
        <Field
          id="unlock-passphrase"
          label="Passphrase"
          type="password"
          value={passphrase}
          onChange={(e) => setPassphrase(e.target.value)}
          autoComplete="current-password"
          className="min-w-56"
        />
        <Button type="submit" disabled={isLoading || passphrase.length === 0}>
          Unlock
        </Button>
      </div>
    </form>
  );
}
