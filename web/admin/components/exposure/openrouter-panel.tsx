"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  rtkErrorMessage,
  useDeleteOpenRouterAccountMutation,
  useGetOpenRouterConfigQuery,
  usePatchOpenRouterAccountMutation,
  usePatchOpenRouterConfigMutation,
  usePostOpenRouterAccountMutation,
  useTestOpenRouterMutation,
} from "@/lib/api";
import { useMemo, useState } from "react";

export function OpenRouterPanel({
  onBusy,
  onError,
}: {
  onBusy: (key: string | null) => void;
  onError: (message: string | null) => void;
}) {
  const { data: config, refetch } = useGetOpenRouterConfigQuery();
  const [patchConfig] = usePatchOpenRouterConfigMutation();
  const [postAccount] = usePostOpenRouterAccountMutation();
  const [patchAccount] = usePatchOpenRouterAccountMutation();
  const [deleteAccount] = useDeleteOpenRouterAccountMutation();
  const [testOpenRouter] = useTestOpenRouterMutation();

  const [label, setLabel] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [tokenReveal, setTokenReveal] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const baseURL = useMemo(() => {
    if (typeof window === "undefined") {
      return config?.gateway_public_path ?? "/openrouter/api/v1";
    }
    return `${window.location.origin}${config?.gateway_public_path ?? "/openrouter/api/v1"}`;
  }, [config?.gateway_public_path]);

  async function toggleEnabled(enabled: boolean) {
    onBusy("or-enable");
    onError(null);
    try {
      await patchConfig({ enabled }).unwrap();
      setMessage(enabled ? "Gateway enabled." : "Gateway disabled.");
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  async function regenerateToken() {
    onBusy("or-token");
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
    if (!label.trim() || !apiKey.trim()) {
      onError("Label and OpenRouter API key required.");
      return;
    }
    onBusy("or-add");
    onError(null);
    try {
      await postAccount({ label: label.trim(), api_key: apiKey.trim() }).unwrap();
      setLabel("");
      setApiKey("");
      setMessage("Account added.");
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  async function runTest() {
    onBusy("or-test");
    onError(null);
    setMessage(null);
    try {
      const res = await testOpenRouter({
        gateway_url: config?.gateway_local_base_url ?? "http://127.0.0.1:18081",
      }).unwrap();
      if (res.ok) {
        setMessage(`Test OK (HTTP ${res.status_code})`);
      } else {
        const detail =
          ("message" in res && typeof res.message === "string" && res.message) ||
          res.body_preview?.trim() ||
          res.error ||
          `HTTP ${res.status_code ?? "?"}`;
        setMessage(`Test failed: ${detail}`);
      }
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  const hasAccounts =
    (config?.accounts?.filter((a) => a.enabled && a.has_api_key).length ?? 0) > 0;
  const ready =
    !!config?.enabled &&
    config?.gateway_token_configured &&
    hasAccounts;

  const accounts = config?.accounts ?? [];

  return (
    <div className="space-y-4 max-w-2xl rounded-xl border border-border bg-card p-4 shadow-rb-2">
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-section-title">OpenRouter gateway</h2>
        {config?.enabled && ready ? (
          <Badge variant="success">live</Badge>
        ) : (
          <Badge variant="outline">setup</Badge>
        )}
        {config?.runtime?.enabled && (
          <Badge variant="info">runtime up</Badge>
        )}
      </div>

      <p className="text-body text-muted-foreground">
        OpenRouter-compatible base URL with round-robin across accounts. Client
        SDK: set <code className="text-metadata">baseURL</code> to{" "}
        <code className="text-metadata break-all">{baseURL}</code> and use the gateway
        bearer token.
      </p>

      <div className="flex flex-row items-center justify-between gap-4">
        <div>
          <Label htmlFor="or-enabled">Enable gateway</Label>
          <p className="text-metadata text-muted-foreground">
            When off, proxy returns 503.
          </p>
        </div>
        <Switch
          id="or-enabled"
          checked={config?.enabled ?? false}
          onCheckedChange={(v) => void toggleEnabled(v)}
        />
      </div>

      <div className="flex flex-wrap gap-2">
        <Button type="button" variant="outline" onClick={() => void regenerateToken()}>
          {config?.gateway_token_configured ? "Rotate gateway token" : "Generate gateway token"}
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={() => void runTest()}
          disabled={!ready}
          title={
            !config?.enabled
              ? "Turn on Enable gateway first"
              : !config?.gateway_token_configured
                ? "Generate a gateway token first"
                : !hasAccounts
                  ? "Add at least one enabled account with a key"
                  : undefined
          }
        >
          Test GET /models
        </Button>
      </div>

      {tokenReveal && (
        <p className="text-metadata font-mono break-all bg-muted p-2 rounded-md">
          {tokenReveal}
        </p>
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
                <p className="text-metadata text-muted-foreground">
                  {acc.upstream_label ? (
                    <>
                      OpenRouter: <span className="font-mono">{acc.upstream_label}</span>
                      {acc.key_tail ? (
                        <> · tail …{acc.key_tail}</>
                      ) : null}
                    </>
                  ) : acc.key_tail ? (
                    <>Key tail …{acc.key_tail}</>
                  ) : (
                    acc.id
                  )}
                </p>
                {acc.upstream_user_id && (
                  <p className="text-[10px] text-muted-foreground font-mono truncate max-w-md">
                    {acc.upstream_user_id}
                  </p>
                )}
              </div>
              <div className="flex items-center gap-2">
                {acc.has_api_key ? (
                  <Badge variant="outline">key set</Badge>
                ) : (
                  <Badge variant="warning">no key</Badge>
                )}
                <Switch
                  checked={acc.enabled}
                  onCheckedChange={(v) =>
                    void patchAccount({ id: acc.id, body: { enabled: v } }).then(
                      () => setMessage(`${acc.label} ${v ? "enabled" : "disabled"}.`),
                      (e) => onError(rtkErrorMessage(e)),
                    )
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
        <h3 className="text-body font-semibold">Add account</h3>
        <Field
          label="Label"
          id="or-label"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
        />
        <Field
          label="OpenRouter API key"
          id="or-key"
          type="password"
          autoComplete="off"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
        />
        <Button type="button" onClick={() => void addAccount()}>
          Add account
        </Button>
      </div>

      {message && <p className="text-metadata text-muted-foreground">{message}</p>}
    </div>
  );
}