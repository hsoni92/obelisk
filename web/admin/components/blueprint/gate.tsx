"use client";

import { ConsoleShellSkeleton } from "@/components/skeletons";
import { rtkErrorMessage, useGetBlueprintStatusQuery } from "@/lib/api";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

export function BlueprintGate({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { data, isLoading, isError, error, isFetching } =
    useGetBlueprintStatusQuery(undefined, {
      pollingInterval: 5000,
    });
  const [bootGrace, setBootGrace] = useState(true);
  const [errorStreak, setErrorStreak] = useState(0);
  useEffect(() => {
    const t = window.setTimeout(() => setBootGrace(false), 25000);
    return () => window.clearTimeout(t);
  }, []);
  useEffect(() => {
    if (isError && !isFetching) {
      setErrorStreak((n) => n + 1);
    } else if (data) {
      setErrorStreak(0);
    }
  }, [isError, isFetching, data]);
  const wrongPort = useMemo(() => {
    if (typeof window === "undefined") return false;
    const port = window.location.port;
    return port === "3000" || port === "3080";
  }, []);

  useEffect(() => {
    if (!data) return;
    if (!data.authenticated) {
      router.replace("/login");
      return;
    }
    if (!data.repository) {
      router.replace("/onboard");
    }
  }, [data, router]);

  const showFatal =
    isError && !isFetching && errorStreak >= 3 && !bootGrace;
  if (showFatal) {
    const detail = rtkErrorMessage(error);
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-3 px-6 text-center">
        <p className="text-sm text-destructive" role="alert">
          Admin API is unreachable through Kong.
        </p>
        {wrongPort ? (
          <p className="text-body text-muted-foreground max-w-md">
            Use the Kong URL on port <strong>9080</strong>, not the admin UI container port (
            {typeof window !== "undefined" ? window.location.port : "3000"}).
          </p>
        ) : (
          <p className="text-body text-muted-foreground max-w-md">
            Open the appliance on port 9080 (not :3000). On the host:{" "}
            <code className="font-mono text-xs">curl -fsS http://&lt;vm-ip&gt;/api/v1/health</code>
          </p>
        )}
        <p className="text-body text-muted-foreground text-xs font-mono">{detail}</p>
      </div>
    );
  }

  if (
    isLoading ||
    isFetching ||
    bootGrace ||
    (isError && !showFatal) ||
    !data?.authenticated ||
    !data.repository
  ) {
    return <ConsoleShellSkeleton />;
  }

  return children;
}
