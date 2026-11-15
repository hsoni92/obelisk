export type BackupRemoteProvider = "aws" | "r2" | "custom";

/** Common AWS regions for S3 regional endpoints (not exhaustive). */
export const AWS_S3_REGIONS: readonly { value: string; label: string }[] = [
  { value: "us-east-1", label: "US East (N. Virginia) — us-east-1" },
  { value: "us-east-2", label: "US East (Ohio) — us-east-2" },
  { value: "us-west-1", label: "US West (N. California) — us-west-1" },
  { value: "us-west-2", label: "US West (Oregon) — us-west-2" },
  { value: "ca-central-1", label: "Canada (Central) — ca-central-1" },
  { value: "ca-west-1", label: "Canada (West) — ca-west-1" },
  { value: "eu-west-1", label: "Europe (Ireland) — eu-west-1" },
  { value: "eu-west-2", label: "Europe (London) — eu-west-2" },
  { value: "eu-west-3", label: "Europe (Paris) — eu-west-3" },
  { value: "eu-central-1", label: "Europe (Frankfurt) — eu-central-1" },
  { value: "eu-central-2", label: "Europe (Zurich) — eu-central-2" },
  { value: "eu-north-1", label: "Europe (Stockholm) — eu-north-1" },
  { value: "eu-south-1", label: "Europe (Milan) — eu-south-1" },
  { value: "eu-south-2", label: "Europe (Spain) — eu-south-2" },
  { value: "ap-south-1", label: "Asia Pacific (Mumbai) — ap-south-1" },
  { value: "ap-south-2", label: "Asia Pacific (Hyderabad) — ap-south-2" },
  { value: "ap-southeast-1", label: "Asia Pacific (Singapore) — ap-southeast-1" },
  { value: "ap-southeast-2", label: "Asia Pacific (Sydney) — ap-southeast-2" },
  { value: "ap-southeast-3", label: "Asia Pacific (Jakarta) — ap-southeast-3" },
  { value: "ap-southeast-4", label: "Asia Pacific (Melbourne) — ap-southeast-4" },
  { value: "ap-northeast-1", label: "Asia Pacific (Tokyo) — ap-northeast-1" },
  { value: "ap-northeast-2", label: "Asia Pacific (Seoul) — ap-northeast-2" },
  { value: "ap-northeast-3", label: "Asia Pacific (Osaka) — ap-northeast-3" },
  { value: "ap-east-1", label: "Asia Pacific (Hong Kong) — ap-east-1" },
  { value: "me-south-1", label: "Middle East (Bahrain) — me-south-1" },
  { value: "me-central-1", label: "Middle East (UAE) — me-central-1" },
  { value: "af-south-1", label: "Africa (Cape Town) — af-south-1" },
  { value: "il-central-1", label: "Israel (Tel Aviv) — il-central-1" },
  { value: "sa-east-1", label: "South America (São Paulo) — sa-east-1" },
];

const PROVIDER_OPTIONS: { value: BackupRemoteProvider; label: string }[] = [
  { value: "aws", label: "Amazon S3" },
  { value: "r2", label: "Cloudflare R2" },
  { value: "custom", label: "Other S3-compatible" },
];

export { PROVIDER_OPTIONS as BACKUP_REMOTE_PROVIDER_OPTIONS };

export function awsEndpointForRegion(region: string): string {
  const r = region.trim().toLowerCase();
  if (!r) return "";
  return `https://s3.${r}.amazonaws.com`;
}

function hostnameFromEndpoint(raw: string): string | null {
  const trimmed = raw.trim();
  if (!trimmed) return null;
  try {
    const withScheme = trimmed.includes("://") ? trimmed : `https://${trimmed}`;
    return new URL(withScheme).hostname.toLowerCase();
  } catch {
    return null;
  }
}

function normalizeEndpoint(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return "";
  return trimmed.includes("://") ? trimmed : `https://${trimmed}`;
}

export function parseBackupRemoteEndpoint(raw: string): {
  provider: BackupRemoteProvider;
  awsRegion: string;
  customEndpoint: string;
} {
  const empty = {
    provider: "aws" as const,
    awsRegion: "us-east-1",
    customEndpoint: "",
  };
  const trimmed = raw.trim();
  if (!trimmed) return empty;

  const host = hostnameFromEndpoint(trimmed);
  if (!host) {
    return { provider: "custom", awsRegion: "us-east-1", customEndpoint: trimmed };
  }

  if (host === "s3.amazonaws.com") {
    return { provider: "aws", awsRegion: "us-east-1", customEndpoint: "" };
  }

  const awsMatch = host.match(/^s3\.([a-z0-9-]+)\.amazonaws\.com$/);
  if (awsMatch) {
    return { provider: "aws", awsRegion: awsMatch[1], customEndpoint: "" };
  }

  if (host.endsWith(".r2.cloudflarestorage.com")) {
    return {
      provider: "r2",
      awsRegion: "us-east-1",
      customEndpoint: normalizeEndpoint(trimmed),
    };
  }

  return {
    provider: "custom",
    awsRegion: "us-east-1",
    customEndpoint: normalizeEndpoint(trimmed),
  };
}

export function resolveBackupRemoteEndpoint(
  provider: BackupRemoteProvider,
  awsRegion: string,
  customEndpoint: string,
): string {
  if (provider === "aws") {
    return awsEndpointForRegion(awsRegion);
  }
  return normalizeEndpoint(customEndpoint);
}