"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  rtkErrorMessage,
  useDeleteMailAccountMutation,
  useGetMailConfigQuery,
  usePatchMailAccountMutation,
  usePatchMailConfigMutation,
  usePostMailAccountMutation,
  useTestMailMutation,
} from "@/lib/api";
import { useMemo, useState } from "react";

export function MailPanel({
  onBusy,
  onError,
}: {
  onBusy: (key: string | null) => void;
  onError: (message: string | null) => void;
}) {
  const { data: config, refetch } = useGetMailConfigQuery();
  const [patchConfig] = usePatchMailConfigMutation();
  const [postAccount] = usePostMailAccountMutation();
  const [patchAccount] = usePatchMailAccountMutation();
  const [deleteAccount] = useDeleteMailAccountMutation();
  const [testMail] = useTestMailMutation();

  const [label, setLabel] = useState("");
  const [address, setAddress] = useState("");
  const [secret, setSecret] = useState("");
  const [tokenReveal, setTokenReveal] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const baseURL = useMemo(() => {
    if (typeof window === "undefined") {
      return config?.gateway_public_path ?? "/mail/api/v1";
    }
    return `${window.location.origin}${config?.gateway_public_path ?? "/mail/api/v1"}`;
  }, [config?.gateway_public_path]);

  async function toggleEnabled(enabled: boolean) {
    onBusy("mail-enable");
    onError(null);
    try {
      await patchConfig({ enabled }).unwrap();
      setMessage(enabled ? "Mail gateway enabled." : "Mail gateway disabled.");
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  async function regenerateToken() {
    onBusy("mail-token");
    onError(null);
    try {
      const res = await patchConfig({ regenerate_gateway_token: true }).unwrap();
      if ("gateway_token" in res) {
        setTokenReveal(res.gateway_token);
        setMessage("New gateway token generated — copy now.");
      }
      await refetch();
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  async function addAccount() {
    if (!label.trim() || !address.trim() || !secret.trim()) {
      onError("Label, email address, and app password required.");
      return;
    }
    onBusy("mail-add");
    onError(null);
    try {
      await postAccount({
        label: label.trim(),
        address: address.trim(),
        secret: secret.trim(),
      }).unwrap();
      setLabel("");
      setAddress("");
      setSecret("");
      setMessage("Mailbox added.");
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  async function runTest() {
    onBusy("mail-test");
    onError(null);
    setMessage(null);
    try {
      const res = await testMail().unwrap();
      if (res.ok) {
        setMessage(`Test send OK to ${res.to ?? "mailbox"}.`);
      } else {
        onError(res.error ?? "Test failed");
      }
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  const hasAccounts =
    (config?.accounts?.filter((a) => a.enabled && a.has_secret).length ?? 0) > 0;
  const ready =
    !!config?.enabled && config?.gateway_token_configured && hasAccounts;

  const accounts = config?.accounts ?? [];

  return (
    <div className="space-y-4 max-w-2xl rounded-xl border border-border bg-card p-4 shadow-rb-2">
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-section-title">Mail gateway</h2>
        {config?.enabled && ready ? (
          <Badge variant="success">live</Badge>
        ) : (
          <Badge variant="outline">setup</Badge>
        )}
        {config?.runtime?.enabled && <Badge variant="info">runtime up</Badge>}
      </div>

      <p className="text-body text-muted-foreground">
        HTTP send via pooled mailboxes. Use{" "}
        <code className="text-metadata break-all">{baseURL}</code> with{" "}
        <code className="text-metadata">POST /send</code> and the gateway bearer token.
      </p>

      <div className="flex flex-row items-center justify-between gap-4">
        <div>
          <Label htmlFor="mail-enabled">Enable gateway</Label>
          <p className="text-metadata text-muted-foreground">When off, send returns 503.</p>
        </div>
        <Switch
          id="mail-enabled"
          checked={config?.enabled ?? false}
          onCheckedChange={(v) => void toggleEnabled(v)}
        />
      </div>

      <div className="flex flex-wrap gap-2">
        <Button type="button" variant="outline" onClick={() => void regenerateToken()}>
          {config?.gateway_token_configured ? "Rotate gateway token" : "Generate gateway token"}
        </Button>
        <Button type="button" variant="outline" onClick={() => void runTest()} disabled={!ready}>
          Test send
        </Button>
      </div>

      {tokenReveal && (
        <p className="text-metadata font-mono break-all bg-muted p-2 rounded-md">{tokenReveal}</p>
      )}

      {accounts.length > 0 && (
        <ul className="space-y-2">
          {accounts.map((acc) => (
            <li
              key={acc.id}
              className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border p-3"
            >
              <div>
                <p className="font-medium">{acc.label}</p>
                <p className="text-metadata text-muted-foreground font-mono">{acc.address}</p>
              </div>
              <div className="flex items-center gap-2">
                {acc.has_secret ? (
                  <Badge variant="outline">secret set</Badge>
                ) : (
                  <Badge variant="warning">no secret</Badge>
                )}
                <Switch
                  checked={acc.enabled}
                  onCheckedChange={(v) =>
                    void patchAccount({ id: acc.id, body: { enabled: v } })
                      .unwrap()
                      .then(() =>
                        setMessage(`${acc.label} ${v ? "enabled" : "disabled"}.`),
                      )
                      .catch((e) => onError(rtkErrorMessage(e)))
                  }
                />
                <Button
                  type="button"
                  variant="link"
                  onClick={() =>
                    void deleteAccount({ id: acc.id })
                      .unwrap()
                      .then(() => setMessage(`Removed ${acc.label}.`))
                      .catch((e) => onError(rtkErrorMessage(e)))
                  }
                >
                  Remove
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}

      <div className="space-y-3 border-t border-border pt-4">
        <h3 className="text-body font-semibold">Add mailbox</h3>
        <Field label="Label" id="mail-label" value={label} onChange={(e) => setLabel(e.target.value)} />
        <Field
          label="Email address"
          id="mail-address"
          type="email"
          autoComplete="off"
          value={address}
          onChange={(e) => setAddress(e.target.value)}
        />
        <Field
          label="App password"
          id="mail-secret"
          type="password"
          autoComplete="off"
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
        />
        <Button type="button" onClick={() => void addAccount()}>
          Add mailbox
        </Button>
      </div>

      {message && <p className="text-metadata text-muted-foreground">{message}</p>}
    </div>
  );
}