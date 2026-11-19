"use client";

import { CopyButton } from "@/components/services/copy-button";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  rtkErrorMessage,
  statusPollOptions,
  useCreateProjectMutation,
  useDeleteProjectMutation,
  useGetProjectsQuery,
  useGetStatusQuery,
  usePatchProjectMutation,
  useRotateProjectDiscoveryTokenMutation,
} from "@/lib/api";
import type { ProjectView } from "@/lib/types";
import { toast } from "@/lib/toast";
import { FolderKanban, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

function instanceRef(type: string, id: string) {
  return `${type}/${id}`;
}

function discoveryAbsolute(path: string) {
  if (typeof window === "undefined") return path;
  return `${window.location.origin}${path}`;
}

export function ProjectsPanel() {
  const { data: projects, isLoading } = useGetProjectsQuery();
  const { data: status } = useGetStatusQuery(undefined, statusPollOptions);
  const [createProject] = useCreateProjectMutation();
  const [patchProject] = usePatchProjectMutation();
  const [deleteProject] = useDeleteProjectMutation();
  const [rotateToken] = useRotateProjectDiscoveryTokenMutation();

  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<ProjectView | null>(null);
  const [name, setName] = useState("");
  const [members, setMembers] = useState<Set<string>>(() => new Set());
  const [discovery, setDiscovery] = useState(false);
  const [tokenReveal, setTokenReveal] = useState<{
    projectId: string;
    token: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);

  const instances = useMemo(() => {
    const list = status?.instances ?? [];
    return list.map((i) => ({
      ref: instanceRef(i.type, i.id),
      label: i.display_name?.trim() || `${i.type}/${i.id}`,
      enabled: i.enabled,
    }));
  }, [status?.instances]);

  function resetForm(p?: ProjectView) {
    setEdit(p ?? null);
    setName(p?.name ?? "");
    setMembers(new Set(p?.instance_ids ?? []));
    setDiscovery(p?.discovery_enabled ?? false);
    if (!p) setTokenReveal(null);
  }

  async function save() {
    const n = name.trim();
    if (!n) {
      toast.error("Name required");
      return;
    }
    setBusy(true);
    try {
      const instance_ids = [...members];
      if (edit) {
        await patchProject({
          id: edit.id,
          body: {
            name: n,
            instance_ids,
            discovery_enabled: discovery,
          },
        }).unwrap();
        toast.success("Project updated");
      } else {
        await createProject({
          name: n,
          instance_ids,
          discovery_enabled: discovery,
        }).unwrap();
        toast.success("Project created");
      }
      setOpen(false);
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    setBusy(true);
    try {
      await deleteProject(id).unwrap();
      toast.success("Project deleted");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  async function rotate(id: string) {
    setBusy(true);
    try {
      const res = await rotateToken(id).unwrap();
      setTokenReveal({ projectId: id, token: res.token });
      toast.success("Discovery token ready — copy it below");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-semibold">
            <FolderKanban className="size-5" aria-hidden />
            Projects
          </h2>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            A Project groups instances for connection discovery. It does not
            create databases — add those under each instance.
          </p>
        </div>
        <Button
          type="button"
          onClick={() => {
            resetForm();
            setOpen(true);
          }}
        >
          <Plus className="size-4" aria-hidden />
          New project
        </Button>
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading projects…</p>
      ) : (projects?.length ?? 0) === 0 ? (
        <p className="text-sm text-muted-foreground">No projects yet.</p>
      ) : (
        <div className="grid gap-3">
          {projects!.map((p) => (
            <Card key={p.id}>
              <CardHeader className="pb-2">
                <CardTitle className="text-base">{p.name}</CardTitle>
                <CardDescription className="font-mono text-xs">
                  {p.id}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                <p>
                  {(p.instance_ids?.length ?? 0) === 0
                    ? "No instances"
                    : `${p.instance_ids!.length} instance(s): ${p.instance_ids!.join(", ")}`}
                </p>
                {p.discovery_enabled && p.discovery_url && (
                  <div className="space-y-3 rounded-md border border-border p-3">
                    <p className="font-medium">Client access</p>
                    <p className="text-xs text-muted-foreground">
                      Call the discovery endpoint with a Bearer token (header
                      only). The JSON response contains connection URLs — treat
                      it like a password.
                    </p>
                    <div className="space-y-1">
                      <p className="text-xs font-medium text-muted-foreground">
                        1. Discovery endpoint (GET)
                      </p>
                      <div className="flex flex-wrap items-center gap-2">
                        <code className="break-all text-xs">
                          {discoveryAbsolute(p.discovery_url)}
                        </code>
                        <CopyButton
                          label="Discovery URL"
                          value={discoveryAbsolute(p.discovery_url)}
                        />
                      </div>
                    </div>
                    <div className="space-y-1">
                      <p className="text-xs font-medium text-muted-foreground">
                        2. Bearer token
                      </p>
                      {!p.discovery_configured &&
                        tokenReveal?.projectId !== p.id && (
                          <p className="text-xs text-amber-700 dark:text-amber-400">
                            Not set yet — click Generate token, then copy the
                            value shown once.
                          </p>
                        )}
                      {p.discovery_configured &&
                        tokenReveal?.projectId !== p.id && (
                          <p className="text-xs text-muted-foreground">
                            Token exists in the cluster (not shown again). Rotate
                            to issue a new token and copy it once.
                          </p>
                        )}
                      {tokenReveal?.projectId === p.id && (
                        <div className="space-y-2 rounded-md border border-amber-500/40 bg-amber-500/5 p-2">
                          <p className="text-xs font-medium">
                            Copy now — this token is not stored in the UI
                          </p>
                          <code className="block break-all text-xs">
                            {tokenReveal.token}
                          </code>
                          <CopyButton
                            label="Bearer token"
                            value={tokenReveal.token}
                          />
                        </div>
                      )}
                    </div>
                    <details className="text-xs text-muted-foreground">
                      <summary className="cursor-pointer font-medium text-foreground">
                        Example curl
                      </summary>
                      <pre className="mt-2 overflow-x-auto whitespace-pre-wrap rounded-md bg-muted p-2 font-mono text-[11px]">
                        {`curl -sS \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  "${discoveryAbsolute(p.discovery_url)}"`}
                      </pre>
                    </details>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        variant="default"
                        size="sm"
                        disabled={busy}
                        onClick={() => rotate(p.id)}
                      >
                        {p.discovery_configured ? "Rotate token" : "Generate token"}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          resetForm(p);
                          setOpen(true);
                        }}
                      >
                        Edit members
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={busy}
                        onClick={() => void remove(p.id)}
                      >
                        <Trash2 className="size-4" aria-hidden />
                        Delete
                      </Button>
                    </div>
                  </div>
                )}
                {!p.discovery_enabled && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      resetForm(p);
                      setOpen(true);
                    }}
                  >
                    Edit
                  </Button>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{edit ? "Edit project" : "New project"}</DialogTitle>
          </DialogHeader>
          <DialogBody className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="project-name">Name</Label>
              <Input
                id="project-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. vercel-webapp"
              />
            </div>
            <div className="flex items-center justify-between gap-3">
              <Label htmlFor="project-discovery">Enable discovery URL</Label>
              <Switch
                id="project-discovery"
                checked={discovery}
                onCheckedChange={setDiscovery}
              />
            </div>
            <fieldset className="space-y-2">
              <legend className="px-0 text-sm font-medium">Instances</legend>
              <div className="max-h-48 space-y-2 overflow-y-auto rounded-md border border-border p-3">
                {instances.length === 0 ? (
                  <p className="text-xs text-muted-foreground">
                    No instances yet.
                  </p>
                ) : (
                  instances.map((inst) => (
                    <label
                      key={inst.ref}
                      className="flex cursor-pointer items-center gap-2 text-sm"
                    >
                      <Checkbox
                        checked={members.has(inst.ref)}
                        onCheckedChange={(on) => {
                          const next = new Set(members);
                          if (on) next.add(inst.ref);
                          else next.delete(inst.ref);
                          setMembers(next);
                        }}
                      />
                      <span className={inst.enabled ? "" : "text-muted-foreground"}>
                        {inst.label}
                        {!inst.enabled ? " (disabled)" : ""}
                      </span>
                    </label>
                  ))
                )}
              </div>
            </fieldset>
            {tokenReveal && edit?.id === tokenReveal.projectId && (
              <div className="space-y-2 rounded-md border border-amber-500/40 bg-amber-500/5 p-3">
                <p className="text-xs font-medium">Copy token now — it will not be shown again</p>
                <code className="block break-all text-xs">{tokenReveal.token}</code>
                <CopyButton label="Bearer token" value={tokenReveal.token} />
              </div>
            )}
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="button" disabled={busy} onClick={() => void save()}>
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}