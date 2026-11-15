"use client";

import { BlueprintPanel } from "@/components/blueprint/blueprint-panel";
import { PageHeader } from "@/components/layout/page-header";
import { PlatformSettings } from "@/components/platform/platform-settings";

export default function PlatformPage() {
  return (
    <>
      <PageHeader section="platform" />
      <PlatformSettings />
      <BlueprintPanel />
    </>
  );
}