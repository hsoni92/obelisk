import { ConsoleLayout } from "@/components/layout/console-layout";
import { ConsoleShellSkeleton } from "@/components/skeletons";
import { Suspense } from "react";

export default function ConsoleRootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <Suspense fallback={<ConsoleShellSkeleton />}>
      <ConsoleLayout>{children}</ConsoleLayout>
    </Suspense>
  );
}