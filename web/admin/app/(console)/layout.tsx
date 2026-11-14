import { ConsoleLayout } from "@/components/layout/console-layout";

export default function ConsoleRootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <ConsoleLayout>{children}</ConsoleLayout>;
}