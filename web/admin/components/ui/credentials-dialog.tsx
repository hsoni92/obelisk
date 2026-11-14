"use client";

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
import type { ServiceInstance } from "@/lib/api";

export function CredentialsDialog({
  instance,
  open,
  onOpenChange,
  onSave,
}: {
  instance: ServiceInstance;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (creds: ServiceInstance["credentials"]) => Promise<void>;
}) {
  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    const body: ServiceInstance["credentials"] = {};
    if (instance.type === "postgres" || instance.type === "mysql") {
      body.username = String(data.get("username") || "");
      body.password = String(data.get("password") || "");
      body.database = String(data.get("database") || "");
    } else if (instance.type === "redis") {
      body.password = String(data.get("password") || "");
    } else if (instance.type === "rustfs") {
      body.access_key = String(data.get("access_key") || "");
      body.secret_key = String(data.get("secret_key") || "");
    } else if (instance.type === "qdrant") {
      body.access_key = String(data.get("access_key") || "");
    } else if (instance.type === "searxng") {
      body.secret_key = String(data.get("secret_key") || "");
    }
    await onSave(body);
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={(e) => void submit(e)} className="space-y-3">
          <DialogHeader>
            <DialogTitle>
              Credentials — {instance.display_name || instance.id}
            </DialogTitle>
            <DialogDescription>
              Leave password fields blank to keep the current secret. New
              instances get random defaults until you change them here.
            </DialogDescription>
          </DialogHeader>
          {(instance.type === "postgres" || instance.type === "mysql") && (
            <>
              <Field
                label="Username"
                name="username"
                defaultValue={instance.credentials.username}
              />
              <Field
                label="Password"
                name="password"
                type="password"
                placeholder="••••••••"
              />
              <Field
                label="Database"
                name="database"
                defaultValue={instance.credentials.database}
              />
            </>
          )}
          {instance.type === "redis" && (
            <Field
              label="Password"
              name="password"
              type="password"
              placeholder="••••••••"
            />
          )}
          {instance.type === "rustfs" && (
            <>
              <Field
                label="Access key"
                name="access_key"
                defaultValue={instance.credentials.access_key}
              />
              <Field
                label="Secret key"
                name="secret_key"
                type="password"
                placeholder="••••••••"
              />
            </>
          )}
          {instance.type === "qdrant" && (
            <Field
              label="API key"
              name="access_key"
              type="password"
              placeholder="••••••••"
            />
          )}
          {instance.type === "searxng" && (
            <Field
              label="Secret key"
              name="secret_key"
              type="password"
              placeholder="••••••••"
            />
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit">Save</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
