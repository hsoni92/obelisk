"use client";

import { CopyButton } from "@/components/services/copy-button";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
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
  useGetMailConfigQuery,
  useGetOpenRouterConfigQuery,
  useGetProjectsQuery,
  useGetStatusQuery,
  usePatchProjectMutation,
  useRotateProjectDiscoveryTokenMutation,
} from "@/lib/api";
import {
  EXPOSURE_MAIL,
  EXPOSURE_OPENROUTER,
  exposureMemberLabel,
  isExposureMemberRef,
} from "@/lib/project-members";
import type { ProjectView } from "@/lib/types";
import { toast } from "@/lib/toast";
import { Disclosure } from "@/components/ui/disclosure";
import { CardHeading } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Boxes,
  EllipsisVertical,
  FolderKanban,
  Globe,
  KeyRound,
  Pencil,
  Plus,
  RefreshCw,
  Sparkles,
  Terminal,
  Trash2,
  type LucideIcon,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

function instanceRef(type: string, id: string) {
  return `${type}/${id}`;
}

function discoveryAbsolute(path: string) {
  if (typeof window === "undefined") return path;
  return `${window.location.origin}${path}`;
}

function formatMemberRef(ref: string) {
  if (isExposureMemberRef(ref)) {
    return exposureMemberLabel(ref);
  }
  return ref;
}

type Props = {
  /** Increment from parent to open the new-project dialog. */
  createSignal?: number;
};

export function ProjectsPanel({ createSignal = 0 }: Props) {
  const { data: projects, isLoading } = useGetProjectsQuery();
  const { data: status } = useGetStatusQuery(undefined, statusPollOptions);
  const { data: openRouter } = useGetOpenRouterConfigQuery();
  const { data: mail } = useGetMailConfigQuery();
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
  const [accessId, setAccessId] = useState<string | null>(null);

  const prevCreateSignal = useRef(0);
  useEffect(() => {
    if (createSignal > prevCreateSignal.current) {
      prevCreateSignal.current = createSignal;
      resetForm();
      setOpen(true);
    }
  }, [createSignal]);

  const instances = useMemo(() => {
    const list = status?.instances ?? [];
    return list.map((i) => ({
      ref: instanceRef(i.type, i.id),
      label: i.display_name?.trim() || `${i.type}/${i.id}`,
      enabled: i.enabled,
    }));
  }, [status?.instances]);

  const exposureMembers = useMemo(
    () => [
      {
        ref: EXPOSURE_OPENROUTER,
        label: exposureMemberLabel(EXPOSURE_OPENROUTER),
        enabled: openRouter?.enabled ?? false,
        configured: openRouter?.gateway_token_configured ?? false,
      },
      {
        ref: EXPOSURE_MAIL,
        label: exposureMemberLabel(EXPOSURE_MAIL),
        enabled: mail?.enabled ?? false,
        configured: mail?.gateway_token_configured ?? false,
      },
    ],
    [mail, openRouter],
  );

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

  const accessProject =
    projects?.find((p) => p.id === accessId) ?? null;

  function openEdit(p: ProjectView) {
    resetForm(p);
    setOpen(true);
  }

  function confirmRemove(p: ProjectView) {
    if (window.confirm(`Delete project "${p.name}"? This cannot be undone.`)) {
      void remove(p.id);
    }
  }

  return (
    <div className="space-y-4">
      {isLoading ? (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-36 rounded-xl" />
          ))}
        </div>
      ) : (projects?.length ?? 0) === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border bg-muted/20 px-6 py-12 text-center">
          <span className="grid size-10 place-items-center rounded-xl bg-primary/10 text-primary">
            <FolderKanban className="size-5" aria-hidden />
          </span>
          <div>
            <p className="text-sm font-medium">No projects yet</p>
            <p className="text-metadata text-muted-foreground">
              Group services and gateways, then share one discovery endpoint.
            </p>
          </div>
          <Button
            type="button"
            size="sm"
            onClick={() => {
              resetForm();
              setOpen(true);
            }}
          >
            <Plus className="size-4" aria-hidden />
            New project
          </Button>
        </div>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {projects!.map((p) => (
            <ProjectCard
              key={p.id}
              project={p}
              busy={busy}
              onEdit={() => openEdit(p)}
              onAccess={() => setAccessId(p.id)}
              onDelete={() => confirmRemove(p)}
            />
          ))}
        </div>
      )}

      <Dialog
        open={!!accessProject}
        onOpenChange={(o) => !o && setAccessId(null)}
      >
        <DialogContent className="max-w-lg">
          {accessProject && (
            <>
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2">
                  <KeyRound className="size-4 text-primary" aria-hidden />
                  Client access · {accessProject.name}
                </DialogTitle>
                <DialogDescription>
                  Call the discovery endpoint with a Bearer token (header only).
                  The response contains connection URLs — treat it like a
                  password. For workloads on the same k3s cluster, append{" "}
                  <code className="text-xs">?audience=in_cluster</code> to the
                  discovery URL.
                </DialogDescription>
              </DialogHeader>
              <DialogBody className="space-y-4 text-sm">
                {accessProject.discovery_url ? (
                  <>
                    <AccessStep n={1} title="Discovery endpoint (GET)">
                      <div className="space-y-4">
                        <AccessUrlRow
                          icon={Globe}
                          title="Remote / edge"
                          hint="Vercel, Railway, LAN"
                          label="Discovery URL (edge)"
                          value={discoveryAbsolute(accessProject.discovery_url)}
                        />
                        {accessProject.discovery_kong_in_cluster_url ? (
                          <AccessUrlRow
                            icon={Boxes}
                            title="In-cluster"
                            hint="Pod on this k3s · returns Service DNS"
                            label="Discovery URL (in-cluster)"
                            value={accessProject.discovery_kong_in_cluster_url}
                          />
                        ) : accessProject.discovery_url_in_cluster ? (
                          <AccessUrlRow
                            icon={Boxes}
                            title="In-cluster"
                            hint="Append to the Kong or admin base URL"
                            label="Discovery path (in-cluster)"
                            value={accessProject.discovery_url_in_cluster}
                          />
                        ) : null}
                      </div>
                    </AccessStep>
                    <AccessStep n={2} title="Bearer token">
                      {tokenReveal?.projectId === accessProject.id ? (
                        <div className="space-y-2 rounded-lg border border-amber-500/40 bg-amber-500/5 p-3">
                          <p className="text-xs font-medium">
                            Copy now — this token is not stored in the UI
                          </p>
                          <div className="flex items-center gap-2">
                            <code className="min-w-0 flex-1 break-all text-xs">
                              {tokenReveal.token}
                            </code>
                            <CopyButton
                              label="Bearer token"
                              value={tokenReveal.token}
                            />
                          </div>
                        </div>
                      ) : (
                        <p className="text-xs text-muted-foreground">
                          {accessProject.discovery_configured
                            ? "A token exists in the cluster (not shown again). Rotate to issue a new one."
                            : "Not set yet — generate a token and copy it once."}
                        </p>
                      )}
                      <Button
                        type="button"
                        size="sm"
                        className="mt-2"
                        disabled={busy}
                        onClick={() => void rotate(accessProject.id)}
                      >
                        <RefreshCw className="size-3.5" aria-hidden />
                        {accessProject.discovery_configured
                          ? "Rotate token"
                          : "Generate token"}
                      </Button>
                    </AccessStep>
                    <Disclosure
                      title="Example curl"
                      icon={<Terminal className="size-3.5" aria-hidden />}
                    >
                      <pre className="overflow-x-auto whitespace-pre-wrap font-mono text-[11px] text-muted-foreground">
                        {`# Remote / edge (default)
curl -sS \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  "${discoveryAbsolute(accessProject.discovery_url)}"

# Same cluster (pod in another namespace)
curl -sS \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  "${discoveryAbsolute(accessProject.discovery_url)}?audience=in_cluster"`}
                      </pre>
                    </Disclosure>
                  </>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    Discovery is off for this project. Enable it from Edit.
                  </p>
                )}
              </DialogBody>
            </>
          )}
        </DialogContent>
      </Dialog>

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
              <legend className="px-0 text-sm font-medium">
                Data service instances
              </legend>
              <div className="max-h-40 space-y-2 overflow-y-auto rounded-md border border-border p-3">
                {instances.length === 0 ? (
                  <p className="text-xs text-muted-foreground">
                    No instances yet — create them under Services.
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
                      <span
                        className={inst.enabled ? "" : "text-muted-foreground"}
                      >
                        {inst.label}
                        {!inst.enabled ? " (disabled)" : ""}
                      </span>
                    </label>
                  ))
                )}
              </div>
            </fieldset>
            <fieldset className="space-y-2">
              <legend className="px-0 flex items-center gap-1.5 text-sm font-medium">
                <Sparkles className="size-3.5" aria-hidden />
                Exposure gateways
              </legend>
              <p className="text-xs text-muted-foreground">
                Include Kong HTTP gateways configured under Exposure. Discovery
                returns gateway URLs and bearer tokens when configured.
              </p>
              <div className="space-y-2 rounded-md border border-border p-3">
                {exposureMembers.map((ex) => (
                  <label
                    key={ex.ref}
                    className="flex cursor-pointer items-start gap-2 text-sm"
                  >
                    <Checkbox
                      className="mt-0.5"
                      checked={members.has(ex.ref)}
                      onCheckedChange={(on) => {
                        const next = new Set(members);
                        if (on) next.add(ex.ref);
                        else next.delete(ex.ref);
                        setMembers(next);
                      }}
                    />
                    <span>
                      <span className="font-medium">{ex.label}</span>
                      <span className="mt-0.5 block font-mono text-[11px] text-muted-foreground">
                        {ex.ref}
                      </span>
                      {!ex.enabled ? (
                        <span className="text-xs text-muted-foreground">
                          Gateway disabled on Exposure
                        </span>
                      ) : !ex.configured ? (
                        <span className="text-xs text-amber-700 dark:text-amber-400">
                          Enable gateway token on Exposure first
                        </span>
                      ) : null}
                    </span>
                  </label>
                ))}
              </div>
            </fieldset>
            {tokenReveal && edit?.id === tokenReveal.projectId && (
              <div className="space-y-2 rounded-md border border-amber-500/40 bg-amber-500/5 p-3">
                <p className="text-xs font-medium">
                  Copy token now — it will not be shown again
                </p>
                <code className="block break-all text-xs">
                  {tokenReveal.token}
                </code>
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

const MAX_CHIPS = 4;

function ProjectCard({
  project: p,
  busy,
  onEdit,
  onAccess,
  onDelete,
}: {
  project: ProjectView;
  busy: boolean;
  onEdit: () => void;
  onAccess: () => void;
  onDelete: () => void;
}) {
  const members = p.instance_ids ?? [];
  const shown = members.slice(0, MAX_CHIPS);
  const extra = members.length - shown.length;
  const hasDiscovery = p.discovery_enabled && !!p.discovery_url;

  return (
    <article className="group flex flex-col overflow-hidden rounded-xl border border-border/70 bg-card shadow-sm transition-colors hover:border-primary/40">
      <CardHeading
        icon={FolderKanban}
        tone={hasDiscovery ? "lime" : "neutral"}
        title={
          <button
            type="button"
            onClick={onEdit}
            className="truncate rounded-sm text-left outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
            title="Edit project"
          >
            {p.name}
          </button>
        }
        description={<span className="font-mono text-[11px]">{p.id}</span>}
        className="flex-nowrap"
        actions={
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="icon"
                size="icon"
                className="size-7 shrink-0 shadow-none"
                aria-label={`Actions for ${p.name}`}
              >
                <EllipsisVertical className="size-4" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              <DropdownMenuItem onSelect={onEdit}>
                <Pencil className="size-4" aria-hidden />
                Edit members
              </DropdownMenuItem>
              {hasDiscovery ? (
                <DropdownMenuItem onSelect={onAccess}>
                  <KeyRound className="size-4" aria-hidden />
                  Client access
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                disabled={busy}
                className="text-destructive focus:text-destructive"
                onSelect={onDelete}
              >
                <Trash2 className="size-4" aria-hidden />
                Delete…
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        }
      />
      <div className="flex flex-1 flex-col gap-3 p-4">
      <div className="flex min-h-6 flex-wrap items-center gap-1.5">
        {members.length === 0 ? (
          <span className="text-xs text-muted-foreground">No members yet</span>
        ) : (
          <>
            {shown.map((m) => (
              <span
                key={m}
                className="max-w-full truncate rounded-md bg-muted/60 px-1.5 py-0.5 text-[11px] text-foreground/80"
                title={formatMemberRef(m)}
              >
                {formatMemberRef(m)}
              </span>
            ))}
            {extra > 0 ? (
              <span className="rounded-md px-1.5 py-0.5 text-[11px] text-muted-foreground">
                +{extra} more
              </span>
            ) : null}
          </>
        )}
      </div>

      <footer className="mt-auto flex items-center justify-between gap-2 border-t border-border/60 pt-3">
        <span className="text-xs text-muted-foreground">
          {members.length} member{members.length === 1 ? "" : "s"}
        </span>
        {hasDiscovery ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-7 gap-1.5 px-2.5 text-xs"
            onClick={onAccess}
          >
            <KeyRound className="size-3.5" aria-hidden />
            Client access
            {!p.discovery_configured ? (
              <span
                className="size-1.5 rounded-full bg-amber-500"
                title="Token not generated yet"
              />
            ) : null}
          </Button>
        ) : (
          <Badge variant="outline" className="font-normal">
            Discovery off
          </Badge>
        )}
      </footer>
      </div>
    </article>
  );
}

function AccessUrlRow({
  icon: Icon,
  title,
  hint,
  label,
  value,
}: {
  icon: LucideIcon;
  title: string;
  hint: string;
  label: string;
  value: string;
}) {
  return (
    <div>
      <div className="flex items-center justify-between gap-2">
        <p className="flex items-center gap-1.5 text-[13px] font-medium text-foreground">
          <Icon className="size-3.5 text-muted-foreground" aria-hidden />
          {title}
        </p>
        <CopyButton label={label} value={value} />
      </div>
      <p className="mt-0.5 text-[11px] text-muted-foreground">{hint}</p>
      <code className="mt-1.5 block break-all rounded-xl bg-muted px-3 py-2 font-mono text-[11px] leading-relaxed text-foreground">
        {value}
      </code>
    </div>
  );
}

function AccessStep({
  n,
  title,
  children,
}: {
  n: number;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex gap-3">
      <span className="mt-0.5 grid size-5 shrink-0 place-items-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary">
        {n}
      </span>
      <div className="min-w-0 flex-1 space-y-1.5">
        <p className="text-xs font-medium text-muted-foreground">{title}</p>
        {children}
      </div>
    </div>
  );
}
