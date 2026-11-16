"use client";

import { AuthFrame } from "@/components/blueprint/auth-frame";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import {
  rtkErrorMessage,
  useGetBlueprintStatusQuery,
  useImportBlueprintMutation,
  useOnboardBlueprintMutation,
} from "@/lib/api";
import { writePassphrase } from "@/lib/blueprint-storage";
import { toast } from "@/lib/toast";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

export default function OnboardPage() {
  const router = useRouter();
  const { data: status } = useGetBlueprintStatusQuery(undefined, {
    pollingInterval: 5000,
  });
  const [onboard, { isLoading: creating }] = useOnboardBlueprintMutation();
  const [importBundle, { isLoading: importing }] = useImportBlueprintMutation();
  const [passphrase, setPassphrase] = useState("");
  const [confirm, setConfirm] = useState("");
  const [repo, setRepo] = useState("obelisk-appliance");
  const [consent, setConsent] = useState(false);
  const [choice, setChoice] = useState<"adopt" | "keep" | "">("");
  const seededRepo = useRef(false);

  useEffect(() => {
    if (!status) return;
    if (!status.authenticated) {
      router.replace("/login");
      return;
    }
    if (status.repository) {
      router.replace("/overview");
      return;
    }
    if (status.discovered_repository && !seededRepo.current) {
      seededRepo.current = true;
      setRepo(status.discovered_repository);
    }
  }, [status, router]);

  const busy = creating || importing;
  const restoring = Boolean(status?.bundle_present || status?.import_required);
  const mismatch = Boolean(status?.machine_id_mismatch);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (passphrase.length < 8) {
      toast.warning("Use at least 8 characters. This passphrase cannot be recovered.");
      return;
    }
    if (!restoring && passphrase !== confirm) {
      toast.warning("Passphrases do not match.");
      return;
    }
    if (!consent) {
      toast.warning("Confirm that this browser may remember the passphrase.");
      return;
    }
    if (mismatch && choice !== "adopt" && choice !== "keep") {
      toast.warning("Choose whether to adopt the blueprint machine id.");
      return;
    }
    try {
      if (restoring) {
        await importBundle({
          passphrase,
          repository: repo,
          machine_id_choice: mismatch ? (choice as "adopt" | "keep") : undefined,
        }).unwrap();
      } else {
        await onboard({
          repository: repo,
          branch: "main",
          passphrase,
        }).unwrap();
      }
      writePassphrase(passphrase);
      router.replace("/overview");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  return (
    <AuthFrame
      title={restoring ? "Restore blueprint" : "Create blueprint"}
      subtitle={
        restoring
          ? "This GitHub repo already has an encrypted blueprint. Unlock it to replace local settings."
          : "Choose a passphrase and a private repository. The first save is pushed as generation 1."
      }
    >
      <form onSubmit={(e) => void submit(e)} className="space-y-4">
        <Field
          id="blueprint-repo"
          label="GitHub repository"
          value={repo}
          onChange={(e) => setRepo(e.target.value)}
          autoComplete="off"
          placeholder="owner/obelisk-appliance"
        />
        <Field
          id="blueprint-pass"
          label="Passphrase"
          type="password"
          value={passphrase}
          onChange={(e) => setPassphrase(e.target.value)}
          autoComplete="new-password"
        />
        {!restoring && (
          <Field
            id="blueprint-pass-confirm"
            label="Confirm passphrase"
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
          />
        )}
        {mismatch && (
          <fieldset className="space-y-2 text-sm">
            <legend className="text-metadata text-muted-foreground">
              This host is {status?.machine_id}. The blueprint is{" "}
              {status?.remote_machine_id}.
            </legend>
            <label className="flex items-start gap-2">
              <input
                type="radio"
                name="machine-choice"
                checked={choice === "adopt"}
                onChange={() => setChoice("adopt")}
              />
              <span>Adopt blueprint id and restore that bucket’s data.</span>
            </label>
            <label className="flex items-start gap-2">
              <input
                type="radio"
                name="machine-choice"
                checked={choice === "keep"}
                onChange={() => setChoice("keep")}
              />
              <span>Keep this host’s id. Settings only — no automatic data restore.</span>
            </label>
          </fieldset>
        )}
        <label className="flex items-start gap-2 text-xs text-muted-foreground">
          <input
            type="checkbox"
            className="mt-0.5"
            checked={consent}
            onChange={(e) => setConsent(e.target.checked)}
          />
          <span>
            Remember this passphrase in this browser’s local storage so the
            console can unlock after a reboot. It is not written to the
            appliance disk. Anyone who can read this browser profile can use it.
            A lost passphrase cannot be recovered.
          </span>
        </label>
        <Button type="submit" disabled={busy || !consent}>
          {restoring ? "Unlock and import" : "Create and push"}
        </Button>
      </form>
    </AuthFrame>
  );
}
