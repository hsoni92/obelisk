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
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import {
  Check,
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
  message?: string | null;
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

  return (
    <section aria-label={p.title} className="px-5 py-5">
      {/* Section header: identity, status, master switch */}
      <header className="flex items-center gap-3">
        <span
          className={cn(
            "flex size-9 shrink-0 items-center justify-center rounded-full",
            ready
              ? "bg-success-tint text-success"
              : "bg-broadcast-tint text-broadcast",
          )}
        >
          <p.icon className="size-4" aria-hidden />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h2 className="text-section-title truncate">{p.title}</h2>
            <Badge variant={status.variant}>{status.label}</Badge>
          </div>
          <p className="truncate text-metadata text-muted-foreground">
            {p.description}
          </p>
        </div>
        <label className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
          {p.enabled ? "Enabled" : "Disabled"}
          <Switch
            aria-label={`Enable ${p.title}`}
            checked={p.enabled}
            disabled={pending === "toggle"}
            onCheckedChange={(v) => void run("toggle", () => p.onToggle(v))}
          />
        </label>
      </header>

      {/* Setup progress — disappears once live */}
      {p.enabled && !ready && (
        <ol
          aria-label="Setup progress"
          className="mt-4 flex items-center rounded-lg bg-muted px-3 py-2"
        >
          {steps.map((s, i) => (
            <li key={s.id} className="flex flex-1 items-center last:flex-none">
              <span
                className={cn(
                  "flex items-center gap-1.5 text-xs font-medium",
                  s.done
                    ? "text-success"
                    : s.id === nextStep
                      ? "text-foreground"
                      : "text-muted-foreground",
                )}
              >
                <span
                  className={cn(
                    "flex size-4 items-center justify-center rounded-full text-[10px] font-bold",
                    s.done
                      ? "bg-success text-white"
                      : s.id === nextStep
                        ? "bg-broadcast text-white"
                        : "bg-border text-muted-foreground",
                  )}
                >
                  {s.done ? <Check className="size-2.5" /> : i + 1}
                </span>
                {s.label}
              </span>
              {i < steps.length - 1 && (
                <span
                  className={cn(
                    "mx-2 h-px flex-1",
                    s.done ? "bg-success/40" : "bg-border",
                  )}
                />
              )}
            </li>
          ))}
        </ol>
      )}

      {/* Labelled rows share one grid: icon+label | content */}
      <div className="mt-4 divide-y divide-border">
        <Row icon={Link2} title="Endpoint" hint="Point clients here">
          <div className="flex items-center gap-2">
            <code className="min-w-0 flex-1 truncate rounded-md bg-muted px-3 py-1.5 font-mono text-xs">
              {p.baseURL}
            </code>
            <CopyButton value={p.baseURL} label="Base URL" />
          </div>
        </Row>

        <Row
          icon={KeyRound}
          title="Access token"
          hint="Bearer auth for clients"
        >
          <div className="flex items-center gap-3">
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
            <Button
              size="sm"
              className="ml-auto"
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
          </div>
          {p.tokenReveal && (
            <div
              role="status"
              className="mt-3 rounded-lg bg-broadcast-tint p-3 ring-1 ring-broadcast/20"
            >
              <div className="mb-1 flex items-center justify-between">
                <p className="text-xs font-semibold text-broadcast">
                  Copy it now — it won’t be shown again
                </p>
                <button
                  type="button"
                  aria-label="Dismiss"
                  onClick={p.onDismissToken}
                  className="rounded-full p-0.5 text-broadcast hover:bg-broadcast/10"
                >
                  <X className="size-3.5" />
                </button>
              </div>
              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 break-all font-mono text-xs">
                  {p.tokenReveal}
                </code>
                <CopyButton value={p.tokenReveal} label="Token" />
              </div>
            </div>
          )}
        </Row>

        <Row
          icon={Users}
          title={`${p.noun[0].toUpperCase()}${p.noun.slice(1)}s`}
          hint={`${usable} of ${p.accounts.length} active`}
        >
          {p.accounts.length === 0 ? (
            <button
              type="button"
              onClick={() => setAddOpen(true)}
              className={cn(
                "flex w-full items-center gap-3 rounded-lg border border-dashed px-3 py-3 text-left transition-colors hover:bg-muted",
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
                <span className="block text-metadata text-muted-foreground">
                  Requests are balanced across enabled {p.noun}s
                </span>
              </span>
            </button>
          ) : (
            <div className="space-y-2">
              <ul className="divide-y divide-border rounded-lg ring-1 ring-border">
                {p.accounts.map((a) => (
                  <li key={a.id} className="flex items-center gap-3 px-3 py-2">
                    <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-[11px] font-bold uppercase text-muted-foreground">
                      {a.label.slice(0, 2)}
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-[13px] font-medium leading-4">
                        {a.label}
                      </p>
                      {a.subtitle && (
                        <p className="truncate font-mono text-[11px] text-muted-foreground">
                          {a.subtitle}
                        </p>
                      )}
                    </div>
                    {a.warning && <Badge variant="warning">{a.warning}</Badge>}
                    <Switch
                      aria-label={`${a.enabled ? "Disable" : "Enable"} ${a.label}`}
                      checked={a.enabled}
                      onCheckedChange={(v) =>
                        void run(`acc-${a.id}`, () =>
                          p.onToggleAccount(a.id, v),
                        )
                      }
                    />
                    <button
                      type="button"
                      aria-label={`Remove ${a.label}`}
                      title="Remove"
                      onClick={() => setRemoveTarget(a)}
                      className="rounded-full p-1 text-muted-foreground hover:bg-broadcast-tint hover:text-broadcast"
                    >
                      <Trash2 className="size-3.5" />
                    </button>
                  </li>
                ))}
              </ul>
              <Button
                size="sm"
                variant={nextStep === "account" ? "default" : "outline"}
                onClick={() => setAddOpen(true)}
              >
                <Plus /> Add {p.noun}
              </Button>
            </div>
          )}
        </Row>

        <Row icon={FlaskConical} title="Verify" hint="Send a live test request">
          <div className="flex items-center gap-3">
            <p
              aria-live="polite"
              className="min-w-0 flex-1 truncate text-[13px] text-muted-foreground"
            >
              {p.message ??
                (ready ? "Ready for traffic." : p.testDisabledReason)}
            </p>
            <Button
              size="sm"
              variant={ready ? "default" : "outline"}
              disabled={!ready || pending === "test"}
              onClick={() => void run("test", p.onTest)}
            >
              {pending === "test" ? "Testing…" : p.testLabel}
            </Button>
          </div>
        </Row>
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

/** One labelled row: fixed icon+label column, flexible content column. */
function Row({
  icon: Icon,
  title,
  hint,
  children,
}: {
  icon: LucideIcon;
  title: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="grid gap-2 py-3.5 first:pt-0 last:pb-0 md:grid-cols-[11rem_1fr] md:gap-6">
      <div className="flex items-start gap-2.5">
        <Icon
          className="mt-0.5 size-4 shrink-0 text-muted-foreground"
          aria-hidden
        />
        <div>
          <h3 className="text-[13px] font-semibold leading-4">{title}</h3>
          {hint && (
            <p className="mt-0.5 text-metadata text-muted-foreground">{hint}</p>
          )}
        </div>
      </div>
      <div className="min-w-0">{children}</div>
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
  const [error, setError] = React.useState<string | null>(null);
  const complete = fields.every((f) => values[f.name]?.trim());

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await onAdd(
        Object.fromEntries(
          Object.entries(values).map(([k, v]) => [k, v.trim()]),
        ),
      );
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={(e) => void submit(e)} className="space-y-3">
      <DialogHeader>
        <DialogTitle>Add {noun}</DialogTitle>
        <DialogDescription>
          Added to {title}. Secrets are stored encrypted and never shown again.
        </DialogDescription>
      </DialogHeader>
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
      {error && (
        <p
          role="alert"
          className="rounded-md bg-broadcast-tint px-3 py-2 text-xs font-medium text-broadcast"
        >
          {error}
        </p>
      )}
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
