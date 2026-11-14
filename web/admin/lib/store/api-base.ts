const raw = process.env.NEXT_PUBLIC_API_BASE?.replace(/\/$/, "") ?? "/api/v1";

export const obeliskApiBaseUrl = raw;