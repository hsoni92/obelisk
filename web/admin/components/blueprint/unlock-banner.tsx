"use client";

import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import {
  rtkErrorMessage,
  useGetBlueprintStatusQuery,
  useUnlockBlueprintMutation,
} from "@/lib/api";
import { readPassphrase, writePassphrase } from "@/lib/blueprint-storage";
import { useEffect, useRef, useState } from "react";

export function UnlockBanner() {
  const { data } = useGetBlueprintStatusQuery(undefined, { pollingInterval: 5000 });
  const [unlock, { isLoading }] = useUnlockBlueprintMutation();
  const [passphrase, setPassphrase] = useState("");
  const [error, setError] = useState<string | null>(null);
  const tried = useRef(false);

  useEffect(() => {
    if (!data || data.crypto_unlocked || !data.repository || tried.current) return;
    const stored = readPassphrase();
    if (!stored) return;
    tried.current = true;
    unlock({ passphrase: stored })
      .unwrap()
      .catch((e) => setError(rtkErrorMessage(e)));
  }, [data, unlock]);

  if (!data?.repository || data.crypto_unlocked) return null;

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    try {
      await unlock({ passphrase }).unwrap();
      writePassphrase(passphrase);
      setPassphrase("");
    } catch (e) {
      setError(rtkErrorMessage(e));
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
        {data.import_required ? " Import is required before the next push." : ""}
      </p>
      {error && (
        <p className="mt-2 text-xs text-broadcast" role="alert">
          {error}
        </p>
      )}
      {data.last_push_error && (
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
