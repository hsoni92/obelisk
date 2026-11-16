"use client";

import { AuthSkeleton } from "@/components/skeletons";
import { rtkErrorMessage, useCompleteGitHubCallbackMutation } from "@/lib/api";
import { toast } from "@/lib/toast";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";

const inFlight = new Map<string, Promise<void>>();

function finishGitHubOAuth(
  complete: ReturnType<typeof useCompleteGitHubCallbackMutation>[0],
  router: ReturnType<typeof useRouter>,
  code: string,
  state: string,
): Promise<void> {
  let run = inFlight.get(state);
  if (!run) {
    run = complete({ code, state })
      .unwrap()
      .then(() => {
        router.replace("/onboard");
      })
      .finally(() => {
        inFlight.delete(state);
      });
    inFlight.set(state, run);
  }
  return run;
}

export function CallbackClient() {
  const params = useSearchParams();
  const router = useRouter();
  const [complete] = useCompleteGitHubCallbackMutation();
  const [asyncFailed, setAsyncFailed] = useState(false);
  const toasted = useRef(false);

  const code = params.get("code");
  const state = params.get("state");
  const oauthError = params.get("error_description") || params.get("error");
  const syncFailed =
    Boolean(oauthError) || (!oauthError && (!code || !state));

  useEffect(() => {
    if (!syncFailed || toasted.current) return;
    toasted.current = true;
    toast.error(
      oauthError ?? "GitHub did not return an authorization code",
    );
  }, [syncFailed, oauthError]);

  useEffect(() => {
    if (syncFailed || !code || !state) return;
    let cancelled = false;
    void finishGitHubOAuth(complete, router, code, state).catch((e) => {
      if (!cancelled) {
        if (!toasted.current) {
          toasted.current = true;
          toast.error(rtkErrorMessage(e));
        }
        setAsyncFailed(true);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [syncFailed, code, state, complete, router]);

  const failed = syncFailed || asyncFailed;

  if (failed) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 px-4 text-center">
        <p className="text-body text-muted-foreground">
          GitHub sign-in could not be completed.
        </p>
        <Link href="/login" className="text-sm font-medium text-link">
          Back to sign in
        </Link>
      </div>
    );
  }

  return <AuthSkeleton caption="Finishing GitHub sign-in" />;
}