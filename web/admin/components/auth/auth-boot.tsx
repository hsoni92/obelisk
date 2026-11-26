"use client";

import { AuthSkeleton } from "@/components/skeletons";
import { useGetBlueprintStatusQuery } from "@/lib/api";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

/**
 * Routes "/" once the session state is known. Until then it shows a neutral
 * screen, so a signed-in user never sees the sign-in page first.
 */
export function AuthBoot() {
  const router = useRouter();
  const { data, error, isError } = useGetBlueprintStatusQuery(undefined, {
    pollingInterval: 5000,
  });

  useEffect(() => {
    if (!data) {
      if (isError) router.replace("/login");
      return;
    }
    if (!data.authenticated) {
      router.replace("/login");
      return;
    }
    const onboard =
      data.needs_onboarding ||
      (!data.repository && (data.bundle_present || data.import_required));
    router.replace(onboard ? "/onboard" : "/overview");
  }, [data, error, isError, router]);

  return <AuthSkeleton caption="Checking session" />;
}
