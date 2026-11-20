"use client";

import { CopyableId } from "@/components/ui/copyable-id";
import { Badge } from "@/components/ui/badge";
import { CardHeading } from "@/components/ui/card";
import { ServerCog } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select";
import {
  AWS_S3_REGIONS,
  BACKUP_REMOTE_PROVIDER_OPTIONS,
  type BackupRemoteProvider,
  parseBackupRemoteEndpoint,
  resolveBackupRemoteEndpoint,
} from "@/lib/backup-remote-endpoint";
import {
  rtkErrorMessage,
  useGetBackupTargetQuery,
  usePatchBackupTargetMutation,
  useTestBackupRemoteMutation,
} from "@/lib/api";
import { toast } from "@/lib/toast";
import { useMemo, useState } from "react";

export function BackupRemotePanel({
  onBusy,
}: {
  onBusy: (key: string | null) => void;
}) {
  const { data: target } = useGetBackupTargetQuery();
  const [patchBackupTarget] = usePatchBackupTargetMutation();
  const [testBackupRemote] = useTestBackupRemoteMutation();

  const serverEndpoint = target?.remote.endpoint ?? "";
  const parsedFromServer = useMemo(
    () => parseBackupRemoteEndpoint(serverEndpoint),
    [serverEndpoint],
  );

  const [providerDraft, setProviderDraft] = useState<BackupRemoteProvider | null>(
    null,
  );
  const [awsRegionDraft, setAwsRegionDraft] = useState<string | null>(null);
  const [customEndpointDraft, setCustomEndpointDraft] = useState<string | null>(
    null,
  );

  const provider = providerDraft ?? parsedFromServer.provider;
  const awsRegion = awsRegionDraft ?? parsedFromServer.awsRegion;
  const customEndpoint =
    customEndpointDraft ?? parsedFromServer.customEndpoint;

  const endpoint = resolveBackupRemoteEndpoint(
    provider,
    awsRegion,
    customEndpoint,
  );

  const awsRegionOptions = useMemo(() => {
    if (AWS_S3_REGIONS.some((r) => r.value === awsRegion)) {
      return AWS_S3_REGIONS;
    }
    return [
      { value: awsRegion, label: `${awsRegion} (current)` },
      ...AWS_S3_REGIONS,
    ];
  }, [awsRegion]);

  const [accessKey, setAccessKey] = useState("");
  const [secretKey, setSecretKey] = useState("");

  function clearEndpointDrafts() {
    setProviderDraft(null);
    setAwsRegionDraft(null);
    setCustomEndpointDraft(null);
  }

  async function save() {
    onBusy("dr-save");
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
      clearEndpointDrafts();
      setAccessKey("");
      setSecretKey("");
      toast.success("Remote backup settings saved");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  async function testConnection() {
    onBusy("dr-test");
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
      toast.success(res.last_test_message ?? "Connection test complete");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  const layout = target?.layout;

  return (
    <div className="space-y-4 max-w-xl overflow-hidden rounded-xl border border-border bg-card p-4 shadow-rb-2">
      <CardHeading
        icon={ServerCog}
        tone={target?.backup_ready ? "green" : "amber"}
        title="Remote backup"
        className="-mx-4 -mt-4"
        badge={
          <>
            {target?.backup_ready ? (
              <Badge variant="success">ready</Badge>
            ) : (
              <Badge variant="warning">setup required</Badge>
            )}
            {target?.implementation_status === "stub" && (
              <Badge variant="outline">upload worker stub</Badge>
            )}
          </>
        }
      />

      <div className="space-y-2">
        <p className="text-eyebrow text-muted-foreground">Machine identity</p>
        {target?.machine_id ? (
          <CopyableId value={target.machine_id} label="machine ID" variant="block" />
        ) : (
          <p className="font-mono text-sm">—</p>
        )}
        <p className="text-metadata text-muted-foreground">
          Assigned at first boot and stored in obelisk.yaml. Test connection
          creates an S3/R2 bucket with this name when it does not exist yet
          (IAM needs <code className="font-mono text-[11px]">CreateBucket</code>
          ).
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

      <div className="space-y-1">
        <Label className="text-eyebrow">Remote storage</Label>
        <SelectField
          options={BACKUP_REMOTE_PROVIDER_OPTIONS}
          value={provider}
          onValueChange={(v) => {
            setProviderDraft(v as BackupRemoteProvider);
          }}
          aria-label="Remote storage provider"
        />
      </div>

      {provider === "aws" ? (
        <>
          <div className="space-y-1">
            <Label className="text-eyebrow">AWS region</Label>
            <SelectField
              options={awsRegionOptions}
              value={awsRegion}
              onValueChange={(v) => setAwsRegionDraft(v)}
              placeholder="Select region"
              aria-label="AWS region for S3"
            />
            <p className="text-metadata text-muted-foreground">
              Create the bucket in this same region. Stored endpoint:{" "}
              <code className="font-mono text-[11px] break-all">{endpoint}</code>
            </p>
          </div>
        </>
      ) : (
        <Field
          label={provider === "r2" ? "R2 endpoint URL" : "Endpoint URL"}
          value={customEndpoint}
          onChange={(e) => setCustomEndpointDraft(e.target.value)}
          placeholder="https://<account>.r2.cloudflarestorage.com"
        />
      )}

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