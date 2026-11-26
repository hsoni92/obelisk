"use client";

import { ConnectivityPage } from "@/components/connectivity/connectivity-page";
import { PageHeader } from "@/components/layout/page-header";

export default function ConnectivityRoute() {
  return (
    <>
      <PageHeader section="connectivity" />
      <ConnectivityPage />
    </>
  );
}
