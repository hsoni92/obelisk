"use client";

import { useGetBlueprintStatusQuery } from "@/lib/api";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export function BlueprintGate({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { data, isLoading, isError } = useGetBlueprintStatusQuery(undefined, {
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
    return (
      <div className="min-h-screen flex items-center justify-center px-6">
        <p className="text-sm text-broadcast" role="alert">
          Admin API is unreachable. Start the controller and reload.
        </p>
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
