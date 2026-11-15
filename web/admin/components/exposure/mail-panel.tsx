"use client";

import {
  GatewayCard,
  type GatewayAccount,
} from "@/components/exposure/gateway-card";
import {
  rtkErrorMessage,
  useDeleteMailAccountMutation,
  useGetMailConfigQuery,
  usePatchMailAccountMutation,
  usePatchMailConfigMutation,
  usePostMailAccountMutation,
  useTestMailMutation,
} from "@/lib/api";
import { Mail } from "lucide-react";
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

  const [tokenReveal, setTokenReveal] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const baseURL = useMemo(() => {
    const path = config?.gateway_public_path ?? "/mail/api/v1";
    return typeof window === "undefined"
      ? path
      : `${window.location.origin}${path}`;
  }, [config?.gateway_public_path]);

  async function act(key: string, fn: () => Promise<void>) {
    onBusy(key);
    onError(null);
    try {
      await fn();
    } catch (e) {
      onError(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  const accounts: GatewayAccount[] = (config?.accounts ?? []).map((a) => ({
    id: a.id,
    label: a.label,
    subtitle: a.address,
    enabled: a.enabled,
    warning: a.has_secret ? undefined : "no secret",
  }));

  return (
    <GatewayCard
      icon={Mail}
      title="Mail gateway"
      description="HTTP send through pooled mailboxes — POST /send"
      baseURL={baseURL}
      noun="mailbox"
      enabled={config?.enabled ?? false}
      tokenConfigured={config?.gateway_token_configured ?? false}
      accounts={accounts}
      addFields={[
        { name: "label", label: "Label", placeholder: "e.g. notifications" },
        {
          name: "address",
          label: "Email address",
          type: "email",
          placeholder: "name@example.com",
        },
        { name: "secret", label: "App password", type: "password" },
      ]}
      message={message}
      tokenReveal={tokenReveal}
      onDismissToken={() => setTokenReveal(null)}
      testLabel="Test send"
      testDisabledReason={
        !config?.enabled
          ? "Turn the gateway on first"
          : !config?.gateway_token_configured
            ? "Generate a gateway token first"
            : "Add and enable a mailbox with a secret"
      }
      onToggle={(enabled) =>
        act("mail-enable", async () => {
          await patchConfig({ enabled }).unwrap();
          setMessage(
            enabled ? "Mail gateway enabled." : "Mail gateway disabled.",
          );
        })
      }
      onRotateToken={() =>
        act("mail-token", async () => {
          const res = await patchConfig({
            regenerate_gateway_token: true,
          }).unwrap();
          if ("gateway_token" in res) {
            setTokenReveal(res.gateway_token);
            setMessage("New gateway token generated.");
          }
          await refetch();
        })
      }
      onTest={() =>
        act("mail-test", async () => {
          setMessage(null);
          const res = await testMail().unwrap();
          if (res.ok) setMessage(`Test send OK to ${res.to ?? "mailbox"}.`);
          else onError(res.error ?? "Test failed");
        })
      }
      onAdd={async (v) => {
        try {
          await postAccount({
            label: v.label,
            address: v.address,
            secret: v.secret,
          }).unwrap();
          setMessage("Mailbox added.");
        } catch (e) {
          throw new Error(rtkErrorMessage(e));
        }
      }}
      onToggleAccount={(id, enabled) =>
        act(`mail-acc-${id}`, async () => {
          await patchAccount({ id, body: { enabled } }).unwrap();
          setMessage(`Mailbox ${enabled ? "enabled" : "disabled"}.`);
        })
      }
      onRemoveAccount={(id) =>
        act(`mail-del-${id}`, async () => {
          await deleteAccount({ id }).unwrap();
          setMessage("Mailbox removed.");
        })
      }
    />
  );
}
