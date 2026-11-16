"use client";

import { BlueprintSkeleton } from "@/components/skeletons";
import { Button } from "@/components/ui/button";
import {
  rtkErrorMessage,
  useGetBlueprintStatusQuery,
  useLockBlueprintMutation,
  useLogoutGitHubMutation,
  usePushBlueprintMutation,
} from "@/lib/api";
import { forgetPassphrase } from "@/lib/blueprint-storage";
import { toast } from "@/lib/toast";
import { useRouter } from "next/navigation";

export function BlueprintPanel() {
  const router = useRouter();
  const { data, isLoading } = useGetBlueprintStatusQuery(undefined, {
    pollingInterval: 5000,
  });
  const [push, { isLoading: pushing }] = usePushBlueprintMutation();
  const [lock] = useLockBlueprintMutation();
  const [logout] = useLogoutGitHubMutation();
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
            <dd>{data.crypto_unlocked ? data.sync_status : "push_paused"}</dd>
          </div>
          <div>
            <dt className="text-metadata text-muted-foreground">Generation</dt>
            <dd>{data.generation}</dd>
          </div>
        </dl>
        {data.last_push_error && (
          <p className="mt-3 text-xs text-destructive">{data.last_push_error}</p>
        )}
        <div className="mt-4 flex flex-wrap gap-2">
          <Button type="button" onClick={() => void pushNow()} disabled={pushing || !data.crypto_unlocked}>
            Push now
          </Button>
          <Button type="button" variant="outline" onClick={() => void forget()}>
            Forget passphrase on this device
          </Button>
          <Button type="button" variant="outline" onClick={() => void signOut()}>
            Sign out
          </Button>
        </div>
      </div>
    </section>
  );
}
