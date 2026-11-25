"use client";

import { useSafeReducedMotion } from "@/lib/use-reduced-motion";
import { AuthStage } from "@/components/auth/auth-stage";
import { DeviceCodePanel } from "@/components/auth/device-code-panel";
import { GitHubMark } from "@/components/auth/github-mark";
import { GitHubErrorScreen } from "@/components/blueprint/github-error-screen";
import { Button } from "@/components/ui/button";
import { StatusDot } from "@/components/ui/status-dot";
import {
  copyToClipboard,
  githubHealthFromError,
  rtkErrorMessage,
  useAuthorizeGitHubMutation,
  useCompleteGitHubCallbackMutation,
  useGetBlueprintStatusQuery,
  type GitHubHealth,
} from "@/lib/api";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { AnimatePresence, motion } from "framer-motion";
import {
  ChevronDown,
  Copy,
  Loader2,
  Server,
  Smartphone,
  TriangleAlert,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const EASE = [0.32, 0.72, 0, 1] as const;

type DeviceState = {
  userCode: string;
  verificationURI: string;
  deviceCode: string;
  interval: number;
  expiresAt: number;
  lifetimeSeconds: number;
};

export default function LoginPage() {
  const router = useRouter();
  const reduce = useSafeReducedMotion();
  // Polled so the screen below clears itself once GitHub recovers.
  const { data: status } = useGetBlueprintStatusQuery(undefined, {
    pollingInterval: 10000,
  });
  // A GitHub failure from this page's own requests (rate limit, 401/402/403, ...).
  const [apiError, setApiError] = useState<GitHubHealth | null>(null);
  const [authorize, { isLoading }] = useAuthorizeGitHubMutation();
  const [complete] = useCompleteGitHubCallbackMutation();
  const [redirecting, setRedirecting] = useState(false);
  const [device, setDevice] = useState<DeviceState | null>(null);

  useEffect(() => {
    if (status?.authenticated) {
      router.replace(status.repository ? "/overview" : "/onboard");
    }
  }, [status, router]);

  useEffect(() => {
    if (!device) return;
    let timer = 0;
    let stop = false;
    const tick = async () => {
      if (stop) return;
      try {
        const res = await complete({ device_code: device.deviceCode }).unwrap();
        if (res.status === "ok") {
          router.replace("/onboard");
          return;
        }
      } catch (e) {
        const gh = githubHealthFromError(e);
        if (gh?.blocking) {
          setApiError(gh);
          setDevice(null);
          return;
        }
        const msg = rtkErrorMessage(e);
        if (!msg.toLowerCase().includes("pending")) {
          toast.error(msg);
          return;
        }
      }
      if (!stop) timer = window.setTimeout(tick, device.interval * 1000);
    };
    timer = window.setTimeout(tick, device.interval * 1000);
    return () => {
      stop = true;
      window.clearTimeout(timer);
    };
  }, [device, complete, router]);

  async function browserLogin() {
    try {
      const start = await authorize({
        flow: "code",
        redirect_uri: `${window.location.origin}/login/callback`,
      }).unwrap();
      if (start.authorize_url) {
        setRedirecting(true);
        window.location.assign(start.authorize_url);
      }
    } catch (e) {
      setRedirecting(false);
      const gh = githubHealthFromError(e);
      if (gh?.blocking) {
        setApiError(gh);
        return;
      }
      const msg = rtkErrorMessage(e);
      const callback = `${window.location.origin}/login/callback`;
      if (msg.toLowerCase().includes("redirect")) {
        toast.error("GitHub redirect URL mismatch", callback);
      } else {
        toast.error(msg);
      }
    }
  }

  async function deviceLogin() {
    try {
      const start = await authorize({ flow: "device" }).unwrap();
      if (!start.device_code || !start.user_code || !start.verification_uri) {
        toast.error("GitHub did not return a device code");
        return;
      }
      const lifetime = start.expires_in && start.expires_in > 0 ? start.expires_in : 900;
      setDevice({
        userCode: start.user_code,
        verificationURI: start.verification_uri,
        deviceCode: start.device_code,
        interval: start.interval && start.interval > 0 ? start.interval : 5,
        expiresAt: Date.now() + lifetime * 1000,
        lifetimeSeconds: lifetime,
      });
    } catch (e) {
      const gh = githubHealthFromError(e);
      if (gh?.blocking) {
        setApiError(gh);
        return;
      }
      toast.error(rtkErrorMessage(e));
    }
  }

  // Sign-in cannot work while GitHub is refusing us: explain why instead of
  // offering buttons that would only fail again.
  const blocked = apiError ?? (status?.github_api?.blocking ? status.github_api : null);
  if (blocked) {
    return (
      <GitHubErrorScreen
        health={blocked}
        context="login"
        onRecovered={() => setApiError(null)}
      />
    );
  }

  const notConfigured = Boolean(status && !status.client_configured);
  // Fresh installs allow nobody until the operator runs `obelisk admin allow add` on the machine.
  const noUsersAllowed = Boolean(status?.login_allowlist_empty);
  const signInBlocked = notConfigured || noUsersAllowed;
  const busy = isLoading || redirecting;

  const item = (delay: number) =>
    reduce
      ? {}
      : {
          initial: { opacity: 0, y: 12 },
          animate: { opacity: 1, y: 0 },
          transition: { duration: 0.5, delay, ease: EASE },
        };

  return (
    // Keyed on the motion preference: the server cannot know it, so the first
    // client render assumes "no preference". Remounting once it is known makes
    // every entrance animation start from the right state instead of being
    // left half-played.
    <AuthStage key={reduce ? "reduced" : "motion"}>
      <motion.div
        initial={reduce ? false : { opacity: 0, y: 18, scale: 0.985 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.6, delay: 0.1, ease: EASE }}
        className="rounded-3xl border border-border bg-card p-6 shadow-rb-2 sm:p-8"
      >
        <motion.div {...item(0.2)} className="flex items-center justify-between gap-3">
          <span className="text-eyebrow text-muted-foreground">Sign in</span>
          {status?.machine_id && (
            <span
              className="inline-flex max-w-[230px] items-center gap-1.5 rounded-full bg-muted px-2.5 py-1 text-[11px] text-muted-foreground"
              title="The appliance you are signing in to"
            >
              <StatusDot tone="ok" className="size-1.5" />
              <Server className="size-3 shrink-0" strokeWidth={1.75} />
              <span className="truncate font-mono">{status.machine_id}</span>
            </span>
          )}
        </motion.div>

        <motion.h1
          {...item(0.28)}
          className="mt-4 text-[30px] font-light leading-9 tracking-[-0.03em]"
        >
          {device ? "Approve on GitHub" : "Welcome back"}
        </motion.h1>
        <motion.p {...item(0.34)} className="mt-1.5 text-body text-muted-foreground">
          {device
            ? "Enter the code below on github.com to finish signing in."
            : "Continue with GitHub to unlock this appliance\u2019s admin console."}
        </motion.p>

        <AnimatePresence mode="wait" initial={false}>
          {device ? (
            <DeviceCodePanel
              key="device"
              userCode={device.userCode}
              verificationURI={device.verificationURI}
              expiresAt={device.expiresAt}
              lifetimeSeconds={device.lifetimeSeconds}
              onCancel={() => setDevice(null)}
              onExpired={() => {
                setDevice(null);
                toast.warning("That code expired", "Request a new one to continue.");
              }}
            />
          ) : (
            <motion.div
              key="choices"
              initial={reduce ? false : { opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={reduce ? { opacity: 0 } : { opacity: 0, y: -8 }}
              transition={{ duration: 0.3, ease: EASE }}
            >
              {noUsersAllowed && <NoUsersAllowedNotice />}

              {notConfigured && (
                <div
                  role="alert"
                  className="mt-5 flex gap-3 rounded-2xl bg-destructive-tint p-3.5 text-[13px] leading-5 text-destructive"
                >
                  <TriangleAlert className="mt-0.5 size-4 shrink-0" strokeWidth={1.75} />
                  <p>
                    GitHub sign-in is not configured. Set{" "}
                    <code className="font-mono">OBELISK_GITHUB_CLIENT_ID</code> and the
                    client secret on the controller, or create Secret{" "}
                    <code className="font-mono">obelisk-github-oauth</code>.
                  </p>
                </div>
              )}

              <motion.div {...item(0.4)} className="mt-6">
                <Button
                  type="button"
                  variant="ink"
                  size="lg"
                  className="auth-cta w-full"
                  onClick={() => void browserLogin()}
                  disabled={busy || signInBlocked}
                >
                  {busy ? (
                    <Loader2 className="size-5 animate-spin" />
                  ) : (
                    <GitHubMark />
                  )}
                  {redirecting ? "Opening GitHub…" : "Continue with GitHub"}
                </Button>
              </motion.div>

              <motion.div
                {...item(0.46)}
                className="my-5 flex items-center gap-3 text-metadata text-muted-foreground"
                aria-hidden
              >
                <span className="h-px flex-1 bg-border" />
                or
                <span className="h-px flex-1 bg-border" />
              </motion.div>

              <motion.div {...item(0.52)}>
                <Button
                  type="button"
                  variant="outline"
                  size="lg"
                  className="w-full"
                  onClick={() => void deviceLogin()}
                  disabled={busy || signInBlocked}
                >
                  <Smartphone className="size-4" />
                  Use a device code
                </Button>
                <p className="mt-2.5 text-center text-metadata text-muted-foreground">
                  Best on a VPS or when the admin URL is an IP address. Approve from
                  any browser.
                </p>
              </motion.div>

              {status?.client_configured && (
                <motion.div {...item(0.58)}>
                  <CallbackUrl />
                </motion.div>
              )}
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>

      <motion.p
        {...item(0.7)}
        className="mx-auto mt-5 max-w-sm px-4 text-center text-metadata text-muted-foreground"
      >
        Obelisk asks for <code className="font-mono text-foreground">repo</code> to push the
        encrypted blueprint and <code className="font-mono text-foreground">read:user</code>{" "}
        to know who you are. Nothing else.
      </motion.p>
    </AuthStage>
  );
}

const ALLOW_COMMAND = "sudo obelisk admin allow add --login YOUR_GITHUB_USER";

/**
 * Shown on a fresh install: no GitHub user is allowed to sign in until the operator allows
 * one from the machine's terminal. Sign-in buttons are disabled while this is showing.
 */
function NoUsersAllowedNotice() {
  async function copy() {
    try {
      await copyToClipboard(ALLOW_COMMAND);
      toast.success("Command copied");
    } catch {
      toast.error("Could not copy the command");
    }
  }

  return (
    <div
      role="alert"
      className="mt-5 rounded-2xl bg-destructive-tint p-3.5 text-[13px] leading-5 text-destructive"
    >
      <div className="flex gap-3">
        <TriangleAlert className="mt-0.5 size-4 shrink-0" strokeWidth={1.75} />
        <p>
          <span className="font-medium">No GitHub user is allowed to sign in yet.</span> This
          appliance starts closed. Allow your GitHub account from a terminal on this machine,
          then reload this page.
        </p>
      </div>
      <div className="mt-2.5 flex items-start gap-2 rounded-xl bg-card p-3 text-foreground">
        <code className="min-w-0 flex-1 break-all font-mono text-[12px] leading-[18px]">
          {ALLOW_COMMAND}
        </code>
        <button
          type="button"
          onClick={() => void copy()}
          aria-label="Copy allow command"
          className="flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <Copy className="size-3.5" />
        </button>
      </div>
    </div>
  );
}

/** The callback URL GitHub must allow for browser sign-in, tucked away until needed. */
function CallbackUrl() {
  const reduce = useSafeReducedMotion();
  const [open, setOpen] = useState(false);
  // Only read once the user opens the panel, which is always on the client.
  const callback = open ? `${window.location.origin}/login/callback` : "";

  async function copy() {
    try {
      await copyToClipboard(callback);
      toast.success("Callback URL copied");
    } catch {
      toast.error("Could not copy the URL");
    }
  }

  return (
    <div className="mt-5 border-t border-border pt-4">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full cursor-pointer items-center justify-between gap-2 rounded-md text-left text-metadata text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        Browser sign-in not working?
        <ChevronDown className={cn("size-4 transition-transform", open && "rotate-180")} />
      </button>
      <AnimatePresence initial={false}>
        {open && (
          <motion.div
            initial={reduce ? false : { height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={reduce ? { opacity: 0 } : { height: 0, opacity: 0 }}
            transition={{ duration: reduce ? 0 : 0.26, ease: EASE }}
            className="overflow-hidden"
          >
            <p className="pt-3 text-body-small text-muted-foreground">
              GitHub only accepts exact redirect URLs. Add this as the{" "}
              <span className="font-medium text-foreground">Authorization callback URL</span>{" "}
              of the OAuth app. If the IP or hostname changes, add each variant (up to
              10), or use a device code, which needs no redirect.
            </p>
            <div className="mt-2.5 flex items-start gap-2 rounded-xl bg-muted p-3">
              <code className="min-w-0 flex-1 break-all font-mono text-[12px] leading-[18px] text-foreground">
                {callback}
              </code>
              <button
                type="button"
                onClick={() => void copy()}
                aria-label="Copy callback URL"
                className="flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-full bg-card text-muted-foreground shadow-rb-1 transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <Copy className="size-3.5" />
              </button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
