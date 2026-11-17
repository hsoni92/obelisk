import { type NextRequest } from "next/server";

function controllerOrigin(): string {
  return (
    process.env.OBELISK_API_URL?.replace(/\/$/, "") ?? "http://127.0.0.1:8080"
  );
}

async function proxy(request: NextRequest, path: string[]) {
  const segment =
    path.length > 0
      ? path.map(encodeURIComponent).join("/")
      : request.nextUrl.pathname.replace(/^\/api\/v1\/?/, "");
  const search = request.nextUrl.search;
  const target = `${controllerOrigin()}/api/v1/${segment}${search}`;
  const headers = new Headers(request.headers);
  headers.delete("host");
  const init: RequestInit = {
    method: request.method,
    headers,
    redirect: "manual",
  };
  if (request.method !== "GET" && request.method !== "HEAD") {
    init.body = await request.arrayBuffer();
  }
  const upstream = await fetch(target, init);
  const outHeaders = new Headers(upstream.headers);
  outHeaders.delete("content-encoding");
  return new Response(upstream.body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers: outHeaders,
  });
}

export async function GET(
  request: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
) {
  const { path } = await ctx.params;
  return proxy(request, path);
}

export async function POST(
  request: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
) {
  const { path } = await ctx.params;
  return proxy(request, path);
}

export async function PATCH(
  request: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
) {
  const { path } = await ctx.params;
  return proxy(request, path);
}

export async function PUT(
  request: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
) {
  const { path } = await ctx.params;
  return proxy(request, path);
}

export async function DELETE(
  request: NextRequest,
  ctx: { params: Promise<{ path: string[] }> },
) {
  const { path } = await ctx.params;
  return proxy(request, path);
}