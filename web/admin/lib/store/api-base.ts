/** Browser always uses Kong-routed /api/v1 (ignore stale build-time env). */
export const obeliskApiBaseUrl =
  typeof window !== "undefined"
    ? "/api/v1"
    : (process.env.NEXT_PUBLIC_API_BASE?.replace(/\/$/, "") ?? "/api/v1");