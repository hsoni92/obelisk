import { Suspense } from "react";
import { CallbackClient } from "./callback-client";

export default function LoginCallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-background flex items-center justify-center">
          <p className="text-body text-muted-foreground">Finishing GitHub sign-in…</p>
        </div>
      }
    >
      <CallbackClient />
    </Suspense>
  );
}
