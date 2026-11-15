"use client";

import { rtkErrorMessage, useCompleteGitHubCallbackMutation } from "@/lib/api";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

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
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const code = params.get("code");
    const state = params.get("state");
    const oauthError = params.get("error_description") || params.get("error");
    if (oauthError) {
      setError(oauthError);
      return;
    }
    if (!code || !state) {
      setError("GitHub did not return an authorization code");
      return;
    }
    let cancelled = false;
    void finishGitHubOAuth(complete, router, code, state).catch((e) => {
      if (!cancelled) setError(rtkErrorMessage(e));
    });
    return () => {
      cancelled = true;
    };
  }, [params, complete, router]);

  return (
    <div className="min-h-screen bg-background flex items-center justify-center px-4">
      <p className="text-body text-muted-foreground" role={error ? "alert" : undefined}>
        {error ?? "Finishing GitHub sign-in…"}
      </p>
    </div>
  );
}