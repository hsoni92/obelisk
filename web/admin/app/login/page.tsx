"use client";

import { AuthFrame } from "@/components/blueprint/auth-frame";
import { Button } from "@/components/ui/button";
import {
  rtkErrorMessage,
  useAuthorizeGitHubMutation,
  useCompleteGitHubCallbackMutation,
  useGetBlueprintStatusQuery,
} from "@/lib/api";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function LoginPage() {
  const router = useRouter();
  const { data: status } = useGetBlueprintStatusQuery();
  const [authorize, { isLoading }] = useAuthorizeGitHubMutation();
  const [complete] = useCompleteGitHubCallbackMutation();
  const [error, setError] = useState<string | null>(null);
  const [device, setDevice] = useState<{
    userCode: string;
    verificationURI: string;
    deviceCode: string;
    interval: number;
  } | null>(null);

  useEffect(() => {
    if (status?.authenticated) {
      router.replace(status.repository ? "/overview" : "/onboard");
    }
  }, [status, router]);

  useEffect(() => {
    if (!device) return;
    let timer = 0;
    let stop = false;
    const tick = async () => {
      if (stop) return;
      try {
        const res = await complete({ device_code: device.deviceCode }).unwrap();
        if (res.status === "ok") {
          router.replace("/onboard");
          return;
        }
      } catch (e) {
        const msg = rtkErrorMessage(e);
        if (!msg.toLowerCase().includes("pending")) {
          setError(msg);
          return;
        }
      }
      if (!stop) timer = window.setTimeout(tick, device.interval * 1000);
    };
    timer = window.setTimeout(tick, device.interval * 1000);
    return () => {
      stop = true;
      window.clearTimeout(timer);
    };
  }, [device, complete, router]);

  async function browserLogin() {
    setError(null);
    try {
      const start = await authorize({
        flow: "code",
        redirect_uri: `${window.location.origin}/login/callback`,
      }).unwrap();
      if (start.authorize_url) {
        window.location.assign(start.authorize_url);
      }
    } catch (e) {
      setError(rtkErrorMessage(e));
    }
  }

  async function deviceLogin() {
    setError(null);
    try {
      const start = await authorize({ flow: "device" }).unwrap();
      if (!start.device_code || !start.user_code || !start.verification_uri) {
        setError("GitHub did not return a device code");
        return;
      }
      setDevice({
        userCode: start.user_code,
        verificationURI: start.verification_uri,
        deviceCode: start.device_code,
        interval: start.interval && start.interval > 0 ? start.interval : 5,
      });
    } catch (e) {
      setError(rtkErrorMessage(e));
    }
  }

  return (
    <AuthFrame
      title="Sign in with GitHub"
      subtitle="The admin console stays locked until this appliance is linked to the GitHub account that holds its encrypted blueprint."
    >
      {status && !status.client_configured && (
        <p className="mb-4 text-xs text-broadcast" role="alert">
          Set <code className="font-mono">OBELISK_GITHUB_CLIENT_ID</code> (and
          secret) on the controller, or create Secret{" "}
          <code className="font-mono">obelisk-github-oauth</code>. The app needs
          the <code className="font-mono">repo</code> scope.
        </p>
      )}
      {error && (
        <p className="mb-4 text-xs text-broadcast" role="alert">
          {error}
        </p>
      )}
      <Button type="button" onClick={() => void browserLogin()} disabled={isLoading}>
        Sign in with GitHub
      </Button>
      <div className="mt-6 border-t border-border pt-4">
        <p className="text-metadata text-muted-foreground">
          Headless VPS: approve a device code from a phone or laptop.
        </p>
        <Button
          type="button"
          variant="outline"
          className="mt-3"
          onClick={() => void deviceLogin()}
          disabled={isLoading || !!device}
        >
          Use a device code
        </Button>
        {device && (
          <div className="mt-4 rounded-lg bg-muted p-3">
            <p className="text-xs text-muted-foreground">Enter this code at</p>
            <a
              className="text-sm font-medium text-link"
              href={device.verificationURI}
              target="_blank"
              rel="noreferrer"
            >
              {device.verificationURI}
            </a>
            <p className="mt-2 font-mono text-lg tracking-widest">{device.userCode}</p>
            <p className="mt-2 text-xs text-muted-foreground">Waiting for approval…</p>
          </div>
        )}
      </div>
    </AuthFrame>
  );
}
