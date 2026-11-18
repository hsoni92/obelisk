"use client";

import { BlueprintSkeleton } from "@/components/skeletons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FieldPassword } from "@/components/ui/field";
import {
  rtkErrorMessage,
  useGetBlueprintStatusQuery,
  useImportBlueprintMutation,
  useLockBlueprintMutation,
  useLogoutGitHubMutation,
  usePushBlueprintMutation,
} from "@/lib/api";
import {
  forgetPassphrase,
  readPassphrase,
  writePassphrase,
} from "@/lib/blueprint-storage";
import { toast } from "@/lib/toast";
import { AlertTriangle, Download, Upload } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

export function BlueprintPanel() {
  const router = useRouter();
  const { data, isLoading } = useGetBlueprintStatusQuery(undefined, {
    pollingInterval: 5000,
  });
  const [push, { isLoading: pushing }] = usePushBlueprintMutation();
  const [importBundle, { isLoading: importing }] = useImportBlueprintMutation();
  const [lock] = useLockBlueprintMutation();
  const [logout] = useLogoutGitHubMutation();

  const [importOpen, setImportOpen] = useState(false);
  const [overwriteOpen, setOverwriteOpen] = useState(false);
  const [passphrase, setPassphrase] = useState("");
  const [needsPassphrase, setNeedsPassphrase] = useState(false);
  const [choice, setChoice] = useState<"adopt" | "keep" | "">("");
  const [dialogError, setDialogError] = useState("");

  async function pushNow() {
    try {
      const next = await push().unwrap();
      if (next.last_commit_sha) {
        toast.success(
          "Blueprint pushed",
          `${next.last_commit_sha.slice(0, 7)} (generation ${next.generation})`,
        );
      } else {
        toast.info("Blueprint is already current");
      }
    } catch (e) {
      toast.error("Push failed", rtkErrorMessage(e));
    }
  }

  function openImportDialog() {
    const stored = readPassphrase() ?? "";
    setPassphrase(stored);
    setNeedsPassphrase(!data?.crypto_unlocked && !stored);
    setChoice("");
    setDialogError("");
    setImportOpen(true);
  }

  function openOverwriteDialog() {
    setDialogError("");
    setOverwriteOpen(true);
  }

  async function confirmImport(event: React.FormEvent) {
    event.preventDefault();
    if (!data) return;
    setDialogError("");

    const mismatch = Boolean(data.machine_id_mismatch);
    if (mismatch && choice !== "adopt" && choice !== "keep") {
      setDialogError("Choose whether to adopt the GitHub machine id or keep this host's id.");
      return;
    }

    const effectivePassphrase = passphrase.trim();
    if (!data.crypto_unlocked && !effectivePassphrase) {
      setNeedsPassphrase(true);
      setDialogError("Enter your blueprint passphrase to decrypt the GitHub bundle.");
      return;
    }

    try {
      const next = await importBundle({
        passphrase: effectivePassphrase || undefined,
        repository: data.repository,
        machine_id_choice: mismatch ? (choice as "adopt" | "keep") : undefined,
      }).unwrap();
      if (effectivePassphrase) {
        writePassphrase(effectivePassphrase);
      }
      setImportOpen(false);
      toast.success(
        "Blueprint imported",
        `Appliance updated to generation ${next.generation}; services are reconciling.`,
      );
    } catch (e) {
      const msg = rtkErrorMessage(e);
      if (
        msg.toLowerCase().includes("passphrase") ||
        msg.toLowerCase().includes("locked")
      ) {
        setNeedsPassphrase(true);
      }
      setDialogError(msg);
    }
  }

  async function confirmOverwrite() {
    setDialogError("");
    try {
      const next = await push({ force: true }).unwrap();
      setOverwriteOpen(false);
      toast.success(
        "GitHub updated",
        `Pushed this appliance's settings as generation ${next.generation}.`,
      );
    } catch (e) {
      setDialogError(rtkErrorMessage(e));
    }
  }

  async function forget() {
    forgetPassphrase();
    await lock().unwrap().catch(() => undefined);
    toast.info(
      "Passphrase forgotten",
      "Push stays paused until you unlock on this browser.",
    );
  }

  async function signOut() {
    forgetPassphrase();
    await logout().unwrap().catch(() => undefined);
    router.replace("/login");
  }

  if (isLoading) return <BlueprintSkeleton />;
  if (!data) return null;

  const conflict = Boolean(data.import_required);
  const remoteGen = data.remote_generation;
  const nextOverwriteGen = Math.max(data.generation, remoteGen || 0) + 1;
  const isImportRequiredError =
    typeof data.last_push_error === "string" &&
    data.last_push_error.toLowerCase().includes("import required");
  const showRawError =
    Boolean(data.last_push_error) && !(conflict && isImportRequiredError);

  return (
    <section id="blueprint" aria-labelledby="blueprint-heading" className="mt-8 max-w-3xl">
      <h2 id="blueprint-heading" className="text-page-title">
        Blueprint
      </h2>
      <p className="mt-0.5 text-body text-muted-foreground">
        GitHub holds the encrypted settings mirror. S3 holds data only.
      </p>
      <div className="mt-4 rounded-xl border border-border bg-card p-4 shadow-rb-2 text-sm">
        <dl className="grid gap-2 sm:grid-cols-2">
          <div>
            <dt className="text-metadata text-muted-foreground">GitHub</dt>
            <dd>{data.login || "—"}</dd>
          </div>
          <div>
            <dt className="text-metadata text-muted-foreground">Repository</dt>
            <dd className="font-mono text-[12px]">{data.repository || "—"}</dd>
          </div>
          <div>
            <dt className="text-metadata text-muted-foreground">Sync</dt>
            <dd>
              {conflict ? (
                <Badge variant="warning">
                  {data.crypto_unlocked ? "conflict — choose copy" : "push paused (conflict)"}
                </Badge>
              ) : data.crypto_unlocked ? (
                data.sync_status
              ) : (
                "push_paused"
              )}
            </dd>
          </div>
          <div>
            <dt className="text-metadata text-muted-foreground">Generation</dt>
            <dd>
              {remoteGen && remoteGen !== data.generation
                ? `local ${data.generation} · GitHub ${remoteGen}`
                : data.generation}
            </dd>
          </div>
        </dl>

        {conflict && (
          <div className="mt-4 rounded-xl border border-warning/30 bg-warning-tint p-4">
            <div className="flex items-start gap-2.5">
              <AlertTriangle className="mt-0.5 size-4 shrink-0 text-warning" />
              <div className="min-w-0 flex-1">
                <p className="text-section-title text-foreground">
                  GitHub is ahead of this appliance
                </p>
                <p className="mt-1 text-metadata text-foreground/85">
                  {remoteGen
                    ? `This appliance is on generation ${data.generation}, while ${data.repository} on GitHub is on generation ${remoteGen}. `
                    : `GitHub (${data.repository}) has a newer generation than this appliance (generation ${data.generation}). `}
                  Obelisk cannot merge snapshots automatically. Choose which copy to keep:
                </p>
                <div className="mt-3 flex flex-wrap gap-2">
                  <Button
                    type="button"
                    size="sm"
                    onClick={openImportDialog}
                    disabled={importing || pushing}
                  >
                    <Download className="size-3.5" />
                    {remoteGen ? `Use GitHub (g${remoteGen})` : "Use GitHub (import)"}
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={openOverwriteDialog}
                    disabled={pushing || importing || !data.crypto_unlocked}
                  >
                    <Upload className="size-3.5" />
                    Use this appliance (overwrite GitHub)
                  </Button>
                </div>
                {!data.crypto_unlocked && (
                  <p className="mt-2 text-xs text-muted-foreground">
                    Unlock with your passphrase above to overwrite GitHub with this appliance&apos;s settings.
                  </p>
                )}
              </div>
            </div>
          </div>
        )}

        {showRawError && (
          <p className="mt-3 text-xs text-destructive">{data.last_push_error}</p>
        )}

        <div className="mt-4 flex flex-wrap gap-2">
          {!conflict && (
            <Button
              type="button"
              onClick={() => void pushNow()}
              disabled={pushing || !data.crypto_unlocked}
            >
              Push now
            </Button>
          )}
          <Button type="button" variant="outline" onClick={() => void forget()}>
            Forget passphrase on this device
          </Button>
          <Button type="button" variant="outline" onClick={() => void signOut()}>
            Sign out
          </Button>
        </div>
      </div>

      <Dialog open={importOpen} onOpenChange={setImportOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              {remoteGen ? `Use GitHub blueprint (g${remoteGen})?` : "Import from GitHub?"}
            </DialogTitle>
            <DialogDescription>
              Replaces this appliance&apos;s local settings with the snapshot from{" "}
              <span className="font-mono">{data.repository}</span>.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={(e) => void confirmImport(e)}>
            <DialogBody>
              <div className="rounded-lg border border-border bg-muted/50 p-3 text-xs space-y-1">
                <p>
                  <span className="font-medium text-foreground">Keep:</span>{" "}
                  GitHub snapshot ({remoteGen ? `generation ${remoteGen}` : "latest"})
                </p>
                <p>
                  <span className="font-medium text-foreground">Discard:</span>{" "}
                  Unpushed local changes on this appliance since generation {data.generation}
                </p>
                <p className="text-muted-foreground pt-1">
                  Services will reconcile to the imported configuration automatically.
                </p>
              </div>

              {data.machine_id_mismatch && (
                <fieldset className="space-y-2 text-sm">
                  <legend className="text-metadata text-muted-foreground">
                    This host is <span className="font-mono">{data.machine_id}</span>. The GitHub
                    blueprint is <span className="font-mono">{data.remote_machine_id}</span>.
                  </legend>
                  <label className="flex items-start gap-2">
                    <input
                      type="radio"
                      name="panel-machine-choice"
                      checked={choice === "adopt"}
                      onChange={() => setChoice("adopt")}
                    />
                    <span>Adopt blueprint id and use that backup bucket.</span>
                  </label>
                  <label className="flex items-start gap-2">
                    <input
                      type="radio"
                      name="panel-machine-choice"
                      checked={choice === "keep"}
                      onChange={() => setChoice("keep")}
                    />
                    <span>Keep this host&apos;s id (settings only).</span>
                  </label>
                </fieldset>
              )}

              {needsPassphrase && (
                <FieldPassword
                  id="import-passphrase"
                  label="Blueprint passphrase"
                  value={passphrase}
                  onChange={(e) => setPassphrase(e.target.value)}
                  autoComplete="current-password"
                  placeholder="Enter passphrase to decrypt GitHub bundle"
                />
              )}

              {dialogError && (
                <p className="text-xs text-destructive" role="alert">
                  {dialogError}
                </p>
              )}
            </DialogBody>
            <DialogFooter>
              <DialogClose asChild>
                <Button type="button" variant="outline" disabled={importing}>
                  Cancel
                </Button>
              </DialogClose>
              <Button type="submit" disabled={importing}>
                {importing
                  ? "Importing…"
                  : remoteGen
                    ? `Import generation ${remoteGen}`
                    : "Import from GitHub"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={overwriteOpen} onOpenChange={setOverwriteOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Overwrite GitHub with this appliance?</DialogTitle>
            <DialogDescription>
              Replaces the newer snapshot in{" "}
              <span className="font-mono">{data.repository}</span> with this appliance&apos;s
              current settings.
            </DialogDescription>
          </DialogHeader>
          <DialogBody>
            <div className="rounded-lg border border-border bg-muted/50 p-3 text-xs space-y-1">
              <p>
                <span className="font-medium text-foreground">Keep:</span>{" "}
                This appliance&apos;s current settings (pushed as generation {nextOverwriteGen})
              </p>
              <p>
                <span className="font-medium text-foreground">Overwrite:</span>{" "}
                Newer GitHub snapshot ({remoteGen ? `generation ${remoteGen}` : "remote"})
              </p>
            </div>
            {dialogError && (
              <p className="text-xs text-destructive" role="alert">
                {dialogError}
              </p>
            )}
          </DialogBody>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline" disabled={pushing}>
                Cancel
              </Button>
            </DialogClose>
            <Button
              type="button"
              variant="ink"
              disabled={pushing}
              onClick={() => void confirmOverwrite()}
            >
              {pushing ? "Pushing…" : `Overwrite GitHub (push g${nextOverwriteGen})`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
