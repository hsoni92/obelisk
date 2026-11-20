export function R2SetupHint() {
  return (
    <div className="rounded-md border border-border bg-muted/30 p-3 space-y-2 text-metadata text-muted-foreground">
      <p className="text-eyebrow text-foreground">Where to get R2 keys</p>
      <ol className="list-decimal pl-4 space-y-1">
        <li>
          Cloudflare dashboard → <span className="text-foreground">R2</span> →{" "}
          <span className="text-foreground">Manage R2 API tokens</span>.
        </li>
        <li>
          Create a token (
          <span className="italic">Admin Read &amp; Write</span> or{" "}
          <span className="italic">Object Read &amp; Write</span> on your bucket).
        </li>
        <li>
          Copy <span className="text-foreground">Access Key ID</span> and{" "}
          <span className="text-foreground">Secret Access Key</span> once — the
          secret is shown only at creation.
        </li>
      </ol>
      <p>
        Endpoint:{" "}
        <code className="font-mono text-[11px]">
          https://&lt;account-id&gt;.r2.cloudflarestorage.com
        </code>{" "}
        (account ID is on the R2 overview page). These are S3-style keys, not
        your Cloudflare login or global API token.
      </p>
      <p>
        <a
          href="https://developers.cloudflare.com/r2/api/s3/tokens/"
          target="_blank"
          rel="noopener noreferrer"
          className="underline underline-offset-2"
        >
          Cloudflare: R2 API tokens
        </a>
      </p>
    </div>
  );
}