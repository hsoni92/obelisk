"use client";

import { GatewayCardSkeleton } from "@/components/skeletons";
import {
  GatewayCard,
  type GatewayAccount,
} from "@/components/exposure/gateway-card";
import {
  rtkErrorMessage,
  useDeleteOpenRouterAccountMutation,
  useGetOpenRouterConfigQuery,
  usePatchOpenRouterAccountMutation,
  usePatchOpenRouterConfigMutation,
  usePostOpenRouterAccountMutation,
  useTestOpenRouterMutation,
} from "@/lib/api";
import { toast } from "@/lib/toast";
import { Sparkles } from "lucide-react";
import { useMemo, useState } from "react";

export function OpenRouterPanel({
  onBusy,
}: {
  onBusy: (key: string | null) => void;
}) {
  const { data: config, refetch, isLoading } = useGetOpenRouterConfigQuery();
  const [patchConfig] = usePatchOpenRouterConfigMutation();
  const [postAccount] = usePostOpenRouterAccountMutation();
  const [patchAccount] = usePatchOpenRouterAccountMutation();
  const [deleteAccount] = useDeleteOpenRouterAccountMutation();
  const [testOpenRouter] = useTestOpenRouterMutation();

  const [tokenReveal, setTokenReveal] = useState<string | null>(null);

  const baseURL = useMemo(() => {
    const path = config?.gateway_public_path ?? "/openrouter/api/v1";
    return typeof window === "undefined"
      ? path
      : `${window.location.origin}${path}`;
  }, [config?.gateway_public_path]);

  async function act(key: string, fn: () => Promise<void>) {
    onBusy(key);
    try {
      await fn();
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  const accounts: GatewayAccount[] = (config?.accounts ?? []).map((a) => ({
    id: a.id,
    label: a.label,
    subtitle: a.upstream_label
      ? `${a.upstream_label}${a.key_tail ? ` · …${a.key_tail}` : ""}`
      : a.key_tail
        ? `…${a.key_tail}`
        : undefined,
    enabled: a.enabled,
    warning: a.has_api_key ? undefined : "no key",
  }));

  if (isLoading) return <GatewayCardSkeleton />;

  return (
    <GatewayCard
      icon={Sparkles}
      title="OpenRouter gateway"
      description="OpenAI-compatible endpoint, round-robin across accounts"
      baseURL={baseURL}
      noun="account"
      enabled={config?.enabled ?? false}
      tokenConfigured={config?.gateway_token_configured ?? false}
      accounts={accounts}
      addFields={[
        { name: "label", label: "Label", placeholder: "e.g. team-a" },
        {
          name: "api_key",
          label: "OpenRouter API key",
          type: "password",
          placeholder: "sk-or-…",
        },
      ]}
      tokenReveal={tokenReveal}
      onDismissToken={() => setTokenReveal(null)}
      testLabel="Test /models"
      testDisabledReason={
        !config?.enabled
          ? "Turn the gateway on first"
          : !config?.gateway_token_configured
            ? "Generate a gateway token first"
            : "Add and enable an account with a key"
      }
      onToggle={(enabled) =>
        act("or-enable", async () => {
          await patchConfig({ enabled }).unwrap();
          toast.success(
            enabled ? "Gateway enabled" : "Gateway disabled",
          );
        })
      }
      onRotateToken={() =>
        act("or-token", async () => {
          const res = await patchConfig({
            regenerate_gateway_token: true,
          }).unwrap();
          if ("gateway_token" in res) {
            setTokenReveal(res.gateway_token);
            toast.info("New gateway token generated");
          }
          await refetch();
        })
      }
      onTest={() =>
        act("or-test", async () => {
          const res = await testOpenRouter({
            gateway_url:
              config?.gateway_local_base_url ?? "http://127.0.0.1:18081",
          }).unwrap();
          if (res.ok) {
            toast.success(`Test OK (HTTP ${res.status_code})`);
          } else {
            const detail =
              ("message" in res &&
                typeof res.message === "string" &&
                res.message) ||
              res.body_preview?.trim() ||
              res.error ||
              `HTTP ${res.status_code ?? "?"}`;
            toast.error("Gateway test failed", detail);
          }
        })
      }
      onAdd={async (v) => {
        try {
          await postAccount({ label: v.label, api_key: v.api_key }).unwrap();
          toast.success("Account added");
        } catch (e) {
          throw new Error(rtkErrorMessage(e));
        }
      }}
      onToggleAccount={(id, enabled) =>
        act(`or-acc-${id}`, async () => {
          await patchAccount({ id, body: { enabled } }).unwrap();
          toast.success(`Account ${enabled ? "enabled" : "disabled"}`);
        })
      }
      onRemoveAccount={(id) =>
        act(`or-del-${id}`, async () => {
          await deleteAccount({ id }).unwrap();
          toast.success("Account removed");
        })
      }
    />
  );
}
