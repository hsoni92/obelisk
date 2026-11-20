import { ConsoleLayout } from "@/components/layout/console-layout";
import { Suspense } from "react";

export default function ConsoleRootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <Suspense fallback={null}>
      <ConsoleLayout>{children}</ConsoleLayout>
    </Suspense>
  );
}