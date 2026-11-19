"use client";

import { PageHeader } from "@/components/layout/page-header";
import { ProjectsPanel } from "@/components/projects/projects-panel";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { useState } from "react";

export default function ProjectsPage() {
  const [createSignal, setCreateSignal] = useState(0);

  return (
    <>
      <PageHeader
        section="projects"
        action={
          <Button type="button" onClick={() => setCreateSignal((n) => n + 1)}>
            <Plus className="size-4" aria-hidden />
            New project
          </Button>
        }
      />
      <ProjectsPanel createSignal={createSignal} />
    </>
  );
}