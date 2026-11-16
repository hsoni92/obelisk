"use client";

import { rtkErrorMessage, useGetBlueprintStatusQuery } from "@/lib/api";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export function BlueprintGate({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { data, isLoading, isError, error } = useGetBlueprintStatusQuery(undefined, {
    pollingInterval: 5000,
  });

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

  if (isError) {
    const detail = rtkErrorMessage(error);
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-3 px-6 text-center">
        <p className="text-sm text-broadcast" role="alert">
          Admin API is unreachable through Kong.
        </p>
        <p className="text-body text-muted-foreground max-w-md">
          Open the appliance on port 80 (not :3000). On the host:{" "}
          <code className="font-mono text-xs">curl -fsS http://&lt;vm-ip&gt;/api/v1/health</code>
        </p>
        <p className="text-body text-muted-foreground text-xs font-mono">{detail}</p>
      </div>
    );
  }

  if (isLoading || !data?.authenticated || !data.repository) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-body text-muted-foreground">Checking GitHub session…</p>
      </div>
    );
  }

  return children;
}
