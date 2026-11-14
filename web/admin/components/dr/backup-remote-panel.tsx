"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Label } from "@/components/ui/label";
import {
  rtkErrorMessage,
  useGetBackupTargetQuery,
  usePatchBackupTargetMutation,
  useTestBackupRemoteMutation,
} from "@/lib/api";
import { useState } from "react";

export function BackupRemotePanel({
  onBusy,
  onError,
}: {
  onBusy: (key: string | null) => void;
  onError: (message: string | null) => void;
}) {
  const { data: target } = useGetBackupTargetQuery();
  const [patchBackupTarget] = usePatchBackupTargetMutation();
  const [testBackupRemote] = useTestBackupRemoteMutation();

  const serverEndpoint = target?.remote.endpoint ?? "";
  const [endpointDraft, setEndpointDraft] = useState<string | null>(null);
  const endpoint = endpointDraft ?? serverEndpoint;
  const [accessKey, setAccessKey] = useState("");
  const [secretKey, setSecretKey] = useState("");
  const [message, setMessage] = useState<string | null>(null);

  async function save() {
    onBusy("dr-save");
    onError(null);
    setMessage(null);
    try {
      const body: Parameters<typeof patchBackupTarget>[0] = {
        remote: { endpoint },
      };
      if (accessKey && secretKey) {
        body.credentials = {
          access_key_id: accessKey,
          secret_access_key: secretKey,
        };
      }
      await patchBackupTarget(body).unwrap();
      setEndpointDraft(null);
      setAccessKey("");
      setSecretKey("");
      setMessage("Remote backup settings saved.");
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  async function testConnection() {
    onBusy("dr-test");
    onError(null);
    setMessage(null);
    try {
      if (accessKey && secretKey) {
        await patchBackupTarget({
          remote: { endpoint },
          credentials: {
            access_key_id: accessKey,
            secret_access_key: secretKey,
          },
        }).unwrap();
        setAccessKey("");
        setSecretKey("");
      }
      const res = await testBackupRemote().unwrap();
      setMessage(res.last_test_message ?? "Test complete.");
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  const layout = target?.layout;

  return (
    <div className="space-y-4 max-w-xl rounded-xl border border-border bg-card p-4 shadow-rb-2">
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-section-title">Remote backup</h2>
        {target?.backup_ready ? (
          <Badge variant="success">ready</Badge>
        ) : (
          <Badge variant="warning">setup required</Badge>
        )}
        {target?.implementation_status === "stub" && (
          <Badge variant="outline">upload worker stub</Badge>
        )}
      </div>

      <div className="rounded-lg bg-muted/60 p-3 space-y-2">
        <p className="text-eyebrow text-muted-foreground">Machine identity</p>
        <p className="font-mono text-sm break-all">
          {target?.machine_id ?? "—"}
        </p>
        <p className="text-metadata text-muted-foreground">
          Assigned at first boot and stored in obelisk.yaml. Use this exact name
          when you create your R2/S3 bucket — one bucket per appliance.
        </p>
      </div>

      {layout && (
        <div className="text-metadata text-muted-foreground space-y-1">
          <p className="text-eyebrow text-foreground">Object layout (fixed)</p>
          <ul className="font-mono text-[11px] list-disc pl-4 space-y-0.5">
            <li>
              s3://{layout.bucket}/{layout.config_prefix}
              … control plane (obelisk.yaml, Kong)
            </li>
            <li>
              s3://{layout.bucket}/{layout.data_prefix}
              {"{type}/{instance}/"}… pg_dump, RDB, RustFS
            </li>
          </ul>
        </div>
      )}

      <p className="text-metadata text-muted-foreground">
        You only configure the R2/S3 endpoint and API keys. Keys live in{" "}
        <code className="font-mono text-[11px]">
          {target?.credentials_secret ?? "obelisk-backup-remote"}
        </code>
        , not in git.
      </p>

      <Field
        label="Endpoint URL"
        value={endpoint}
        onChange={(e) => setEndpointDraft(e.target.value)}
        placeholder="https://<account>.r2.cloudflarestorage.com"
      />

      <div className="space-y-2 border-t border-border pt-3">
        <Label className="text-eyebrow">API credentials</Label>
        <p className="text-metadata text-muted-foreground">
          {target?.credentials_configured
            ? "Keys on file. Enter both fields below to rotate."
            : "Enter R2/S3 access key and secret."}
        </p>
        <Field
          label="Access key ID"
          value={accessKey}
          onChange={(e) => setAccessKey(e.target.value)}
          autoComplete="off"
        />
        <Field
          label="Secret access key"
          type="password"
          value={secretKey}
          onChange={(e) => setSecretKey(e.target.value)}
          autoComplete="new-password"
        />
      </div>

      {message && <p className="text-xs font-medium text-link">{message}</p>}
      {target?.last_test_at && (
        <p className="text-metadata text-muted-foreground">
          Last test {target.last_test_at}:{" "}
          {target.last_test_ok ? "OK" : "failed"} — {target.last_test_message}
        </p>
      )}

      <div className="flex flex-wrap gap-2">
        <Button type="button" onClick={() => void save()}>
          Save
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={() => void testConnection()}
        >
          Test connection
        </Button>
      </div>
    </div>
  );
}