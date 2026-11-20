"use client";

import { PageHeader } from "@/components/layout/page-header";
import { PlatformSettings } from "@/components/platform/platform-settings";

export default function PlatformPage() {
  return (
    <>
      <PageHeader section="platform" />
      <PlatformSettings />
    </>
  );
}