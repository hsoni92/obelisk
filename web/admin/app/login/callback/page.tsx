import { Suspense } from "react";
import { AuthSkeleton } from "@/components/skeletons";
import { CallbackClient } from "./callback-client";

export default function LoginCallbackPage() {
  return (
    <Suspense
      fallback={<AuthSkeleton caption="Finishing GitHub sign-in" />}
    >
      <CallbackClient />
    </Suspense>
  );
}
