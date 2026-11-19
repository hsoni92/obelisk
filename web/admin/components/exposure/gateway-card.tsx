"use client";

import { CopyButton } from "@/components/services/copy-button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { CardHeading } from "@/components/ui/card";
import { Field } from "@/components/ui/field";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  FlaskConical,
  KeyRound,
  Link2,
  Plus,
  Trash2,
  Users,
  X,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";

export type GatewayAccount = {
  id: string;
  label: string;
  subtitle?: string;
  enabled: boolean;
  /** Set when the account can't be used yet (e.g. "no key"). */
  warning?: string;
};

export type AddField = {
  name: string;
  label: string;
  type?: string;
  placeholder?: string;
};

export type GatewayCardProps = {
  icon: LucideIcon;
  title: string;
  description: string;
  baseURL: string;
  enabled: boolean;
  tokenConfigured: boolean;
  accounts: GatewayAccount[];
  /** Noun for accounts, e.g. "account" / "mailbox". */
  noun: string;
  addFields: AddField[];
  tokenReveal?: string | null;
  onDismissToken?: () => void;
  testLabel: string;
  testDisabledReason?: string;
  onToggle: (enabled: boolean) => Promise<void>;
  onRotateToken: () => Promise<void>;
  onTest: () => Promise<void>;
  onAdd: (values: Record<string, string>) => Promise<void>;
  onToggleAccount: (id: string, enabled: boolean) => Promise<void>;
  onRemoveAccount: (id: string) => Promise<void>;
};

export function GatewayCard(p: GatewayCardProps) {
  const [pending, setPending] = React.useState<string | null>(null);
  const [addOpen, setAddOpen] = React.useState(false);
  const [confirmRotate, setConfirmRotate] = React.useState(false);
  const [removeTarget, setRemoveTarget] = React.useState<GatewayAccount | null>(
    null,
  );

  const usable = p.accounts.filter((a) => a.enabled && !a.warning).length;
  const steps = [
    { id: "enable", label: "Enable", done: p.enabled },
    { id: "token", label: "Token", done: p.tokenConfigured },
    { id: "account", label: `Add ${p.noun}`, done: usable > 0 },
  ];
  const doneCount = steps.filter((s) => s.done).length;
  const ready = doneCount === steps.length;
  const nextStep = steps.find((s) => !s.done)?.id;

  async function run(key: string, fn: () => Promise<void>) {
    setPending(key);
    try {
      await fn();
    } finally {
      setPending(null);
    }
  }

  const status = !p.enabled
    ? { label: "Off", variant: "default" as const }
    : ready
      ? { label: "Live", variant: "success" as const }
      : { label: `Setup ${doneCount}/3`, variant: "warning" as const };

  const nounCap = `${p.noun[0].toUpperCase()}${p.noun.slice(1)}s`;

  return (
    <section
      aria-label={p.title}
      className="flex flex-col overflow-hidden rounded-xl border border-border/70 bg-card shadow-sm"
    >
      <CardHeading
        icon={p.icon}
        tone={!p.enabled ? "neutral" : ready ? "green" : "amber"}
        title={p.title}
        description={p.description}
        badge={<Badge variant={status.variant}>{status.label}</Badge>}
        actions={
          <Switch
            aria-label={`Enable ${p.title}`}
            checked={p.enabled}
            disabled={pending === "toggle"}
            onCheckedChange={(v) => void run("toggle", () => p.onToggle(v))}
          />
        }
        className="flex-nowrap"
      />
      <div className="flex flex-1 flex-col p-4">
      {/* Setup progress — disappears once live */}
      {p.enabled && !ready && (
        <div className="mt-3" role="group" aria-label="Setup progress">
          <div className="flex gap-1">
            {steps.map((s) => (
              <span
                key={s.id}
                className={cn(
                  "h-1 flex-1 rounded-full",
                  s.done
                    ? "bg-success"
                    : s.id === nextStep
                      ? "bg-broadcast"
                      : "bg-border",
                )}
              />
            ))}
          </div>
          <p className="mt-1.5 text-[11px] text-muted-foreground">
            Next:{" "}
            <span className="font-medium text-foreground">
              {steps.find((s) => !s.done)?.label}
            </span>
          </p>
        </div>
      )}

      <div className="mb-4 mt-4 space-y-4">
        <Block icon={Link2} title="Endpoint">
          <div className="flex items-center gap-1.5">
            <code className="min-w-0 flex-1 truncate rounded-md bg-muted px-2.5 py-1.5 font-mono text-[11px]">
              {p.baseURL}
            </code>
            <CopyButton value={p.baseURL} label="Base URL" />
          </div>
        </Block>

        <Block
          icon={KeyRound}
          title="Access token"
          action={
            <Button
              size="sm"
              className="h-6 px-2 text-[11px]"
              variant={nextStep === "token" ? "default" : "outline"}
              disabled={pending === "token"}
              onClick={() =>
                p.tokenConfigured
                  ? setConfirmRotate(true)
                  : void run("token", p.onRotateToken)
              }
            >
              {p.tokenConfigured ? "Rotate" : "Generate"}
            </Button>
          }
        >
          <span
            className={cn(
              "flex items-center gap-1.5 text-[13px]",
              p.tokenConfigured ? "text-foreground" : "text-muted-foreground",
            )}
          >
            <span
              className={cn(
                "size-1.5 rounded-full",
                p.tokenConfigured ? "bg-success" : "bg-border",
              )}
            />
            {p.tokenConfigured ? "Configured" : "Not generated yet"}
          </span>
          {p.tokenReveal && (
            <div
              role="status"
              className="mt-2 rounded-lg bg-broadcast-tint p-2.5 ring-1 ring-broadcast/40"
            >
              <div className="mb-1 flex items-center justify-between">
                <p className="text-[11px] font-semibold text-foreground">
                  Copy it now — it won’t be shown again
                </p>
                <button
                  type="button"
                  aria-label="Dismiss"
                  onClick={p.onDismissToken}
                  className="rounded-full p-0.5 text-foreground hover:bg-broadcast/30"
                >
                  <X className="size-3.5" />
                </button>
              </div>
              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 break-all font-mono text-[11px]">
                  {p.tokenReveal}
                </code>
                <CopyButton value={p.tokenReveal} label="Token" />
              </div>
            </div>
          )}
        </Block>

        <Block
          icon={Users}
          title={nounCap}
          meta={`${usable} of ${p.accounts.length} active`}
          action={
            p.accounts.length > 0 ? (
              <Button
                size="sm"
                className="h-6 px-2 text-[11px]"
                variant={nextStep === "account" ? "default" : "outline"}
                onClick={() => setAddOpen(true)}
              >
                <Plus className="size-3" /> Add
              </Button>
            ) : undefined
          }
        >
          {p.accounts.length === 0 ? (
            <button
              type="button"
              onClick={() => setAddOpen(true)}
              className={cn(
                "flex w-full items-center gap-2.5 rounded-lg border border-dashed px-3 py-2.5 text-left transition-colors hover:bg-muted",
                nextStep === "account"
                  ? "border-broadcast/50"
                  : "border-border",
              )}
            >
              <Plus className="size-4 text-muted-foreground" />
              <span>
                <span className="block text-[13px] font-medium">
                  Add your first {p.noun}
                </span>
                <span className="block text-[11px] text-muted-foreground">
                  Requests are balanced across enabled {p.noun}s
                </span>
              </span>
            </button>
          ) : (
            <ul className="max-h-44 divide-y divide-border/60 overflow-y-auto rounded-lg bg-muted/30 ring-1 ring-border/60">
              {p.accounts.map((a) => (
                <li key={a.id} className="flex items-center gap-2.5 px-2.5 py-1.5">
                  <span className="flex size-6 shrink-0 items-center justify-center rounded-md bg-background text-[10px] font-bold uppercase text-muted-foreground ring-1 ring-border/60">
                    {a.label.slice(0, 2)}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[13px] font-medium leading-4">
                      {a.label}
                    </p>
                    {a.subtitle && (
                      <p className="truncate font-mono text-[10px] text-muted-foreground">
                        {a.subtitle}
                      </p>
                    )}
                  </div>
                  {a.warning && <Badge variant="warning">{a.warning}</Badge>}
                  <Switch
                    aria-label={`${a.enabled ? "Disable" : "Enable"} ${a.label}`}
                    checked={a.enabled}
                    onCheckedChange={(v) =>
                      void run(`acc-${a.id}`, () => p.onToggleAccount(a.id, v))
                    }
                  />
                  <button
                    type="button"
                    aria-label={`Remove ${a.label}`}
                    title="Remove"
                    onClick={() => setRemoveTarget(a)}
                    className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </Block>
      </div>

      <footer className="mt-auto flex items-center gap-3 border-t border-border/60 pt-3">
        <FlaskConical
          className="size-3.5 shrink-0 text-muted-foreground"
          aria-hidden
        />
        <p className="min-w-0 flex-1 truncate text-[12px] text-muted-foreground">
          {ready ? "Ready for traffic." : p.testDisabledReason}
        </p>
        <Button
          size="sm"
          variant={ready ? "default" : "outline"}
          disabled={!ready || pending === "test"}
          onClick={() => void run("test", p.onTest)}
        >
          {pending === "test" ? "Testing…" : p.testLabel}
        </Button>
      </footer>
      </div>

      <AddAccountDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        noun={p.noun}
        title={p.title}
        fields={p.addFields}
        onAdd={p.onAdd}
      />

      <AlertDialog open={confirmRotate} onOpenChange={setConfirmRotate}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Rotate gateway token?</AlertDialogTitle>
            <AlertDialogDescription>
              The current token stops working immediately. Update every client
              that uses {p.title} with the new one.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => void run("token", p.onRotateToken)}
            >
              Rotate
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={!!removeTarget}
        onOpenChange={(o) => !o && setRemoveTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {removeTarget?.label}?</AlertDialogTitle>
            <AlertDialogDescription>
              This {p.noun} and its stored secret are deleted. This can’t be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                removeTarget &&
                void run(`acc-${removeTarget.id}`, () =>
                  p.onRemoveAccount(removeTarget.id),
                )
              }
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}

/** Compact labelled block: eyebrow title (+ meta / action) over content. */
function Block({
  icon: Icon,
  title,
  meta,
  action,
  children,
}: {
  icon: LucideIcon;
  title: string;
  meta?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex min-h-6 items-center gap-1.5">
        <Icon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        <h3 className="text-eyebrow text-muted-foreground">{title}</h3>
        {meta && (
          <span className="text-[11px] text-muted-foreground">· {meta}</span>
        )}
        {action && <span className="ml-auto">{action}</span>}
      </div>
      {children}
    </div>
  );
}

function AddAccountDialog({
  open,
  onOpenChange,
  noun,
  title,
  fields,
  onAdd,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  noun: string;
  title: string;
  fields: AddField[];
  onAdd: (values: Record<string, string>) => Promise<void>;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        {/* Unmounts on close, so the form resets each time. */}
        <AddAccountForm
          noun={noun}
          title={title}
          fields={fields}
          onAdd={onAdd}
          onClose={() => onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  );
}

function AddAccountForm({
  noun,
  title,
  fields,
  onAdd,
  onClose,
}: {
  noun: string;
  title: string;
  fields: AddField[];
  onAdd: (values: Record<string, string>) => Promise<void>;
  onClose: () => void;
}) {
  const [values, setValues] = React.useState<Record<string, string>>({});
  const [submitting, setSubmitting] = React.useState(false);
  const complete = fields.every((f) => values[f.name]?.trim());

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    try {
      await onAdd(
        Object.fromEntries(
          Object.entries(values).map(([k, v]) => [k, v.trim()]),
        ),
      );
      onClose();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not add");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={(e) => void submit(e)} className="contents">
      <DialogHeader>
        <DialogTitle>Add {noun}</DialogTitle>
        <DialogDescription>
          Added to {title}. Secrets are stored encrypted and never shown again.
        </DialogDescription>
      </DialogHeader>
      <DialogBody>
        {fields.map((f, i) => (
          <Field
            key={f.name}
            id={`add-${f.name}`}
            label={f.label}
            type={f.type}
            placeholder={f.placeholder}
            autoFocus={i === 0}
            autoComplete="off"
            value={values[f.name] ?? ""}
            onChange={(e) =>
              setValues((v) => ({ ...v, [f.name]: e.target.value }))
            }
          />
        ))}
      </DialogBody>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={!complete || submitting}>
          {submitting ? "Adding…" : `Add ${noun}`}
        </Button>
      </DialogFooter>
    </form>
  );
}
