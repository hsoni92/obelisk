"use client";

import { PageHeader } from "@/components/layout/page-header";
import { OverviewPanel } from "@/components/overview/overview-panel";
import { useRouter } from "next/navigation";

export default function OverviewPage() {
  const router = useRouter();

  return (
    <>
      <PageHeader section="overview" />
      <OverviewPanel
        onOpenDR={() => router.push("/dr")}
        onOpenServices={(key) => {
          if (key) {
            const part = key.includes("/") ? key.split("/")[1] : key;
            router.push(`/services?q=${encodeURIComponent(part)}`);
            return;
          }
          router.push("/services");
        }}
      />
    </>
  );
}