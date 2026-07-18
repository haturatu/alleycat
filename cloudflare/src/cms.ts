interface Env {
  ASSETS: Fetcher;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const asset = await env.ASSETS.fetch(request);
    if (asset.status >= 200 && asset.status < 300) return asset;
    return env.ASSETS.fetch(new Request(new URL("/index.html", request.url), request));
  },
} satisfies ExportedHandler<Env>;
