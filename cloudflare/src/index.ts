import bcrypt from "bcryptjs";

interface Env {
  DB: D1Database;
  MEDIA: R2Bucket;
  ASSETS: Fetcher;
  AUTH_SECRET: string;
  ENVIRONMENT: string;
  CMS_ORIGIN: string;
}

type Data = Record<string, unknown>;

type StoredRow = {
  collection: string;
  id: string;
  data: string;
  created: string;
  updated: string;
};

type Actor = {
  id: string;
  email: string;
  name?: string;
  role: string;
  exp: number;
};

const JSON_HEADERS = { "content-type": "application/json; charset=utf-8" };
const PUBLIC_COLLECTIONS = new Set(["posts", "pages", "post_translations", "settings", "media"]);
const EDITOR_COLLECTIONS = new Set(["posts", "pages", "post_translations", "media"]);
const KNOWN_COLLECTIONS = new Set([
  "cms_users",
  "posts",
  "pages",
  "post_translations",
  "translation_jobs",
  "settings",
  "app_secrets",
  "media",
]);

const DEFAULT_SETTINGS: Data = {
  site_name: "Example Blog",
  description: "A calm place to write.",
  welcome_text: "Welcome to your blog",
  home_top_image: "/default-hero.svg",
  home_top_image_alt: "Default hero image",
  footer_html: "",
  theme: "ember",
  site_url: "",
  site_language: "ja",
  enable_post_translation: false,
  translation_source_locale: "ja",
  translation_locales: "en",
  translation_model: "gemini-1.5-flash",
  translation_requests_per_minute: 5,
  feed_items_limit: 20,
  excerpt_length: 200,
  enable_feed_xml: true,
  enable_feed_json: true,
  enable_ogp_image_generation: false,
  enable_code_highlight: true,
  highlight_theme: "github-dark",
  home_page_size: 3,
  archive_page_size: 10,
  show_toc: true,
  show_archive_tags: true,
  show_archive_search: true,
  show_tags: true,
  show_categories: true,
  show_related_posts: false,
  enable_analytics: false,
  analytics_url: "",
  analytics_site_id: "",
  enable_ads: false,
  ads_client: "",
  enable_comments: false,
  comments_script_tag: "",
};

function json(value: unknown, status = 200): Response {
  return new Response(JSON.stringify(value), { status, headers: JSON_HEADERS });
}

function apiError(status: number, message: string, data: Data = {}): Response {
  return json({ status, code: status, message, data }, status);
}

function randomId(length = 15): string {
  const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
  const bytes = crypto.getRandomValues(new Uint8Array(length));
  return Array.from(bytes, (byte) => chars[byte % chars.length]).join("");
}

function base64Url(input: ArrayBuffer | string): string {
  const bytes = typeof input === "string" ? new TextEncoder().encode(input) : new Uint8Array(input);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function decodeBase64Url(input: string): ArrayBuffer {
  const normalized = input.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(input.length / 4) * 4, "=");
  const bytes = Uint8Array.from(atob(normalized), (char) => char.charCodeAt(0));
  return bytes.buffer as ArrayBuffer;
}

async function authKey(secret: string): Promise<CryptoKey> {
  return crypto.subtle.importKey("raw", new TextEncoder().encode(secret), { name: "HMAC", hash: "SHA-256" }, false, ["sign", "verify"]);
}

async function createToken(actor: Omit<Actor, "exp">, secret: string): Promise<string> {
  const header = base64Url(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const payload = base64Url(JSON.stringify({ ...actor, exp: Math.floor(Date.now() / 1000) + 7 * 86400 }));
  const unsigned = `${header}.${payload}`;
  const signature = await crypto.subtle.sign("HMAC", await authKey(secret), new TextEncoder().encode(unsigned));
  return `${unsigned}.${base64Url(signature)}`;
}

async function actorFromRequest(request: Request, env: Env): Promise<Actor | null> {
  const authorization = request.headers.get("authorization") || "";
  const token = authorization.startsWith("Bearer ") ? authorization.slice(7) : authorization.trim();
  const parts = token.split(".");
  if (parts.length !== 3) return null;
  try {
    const valid = await crypto.subtle.verify(
      "HMAC",
      await authKey(env.AUTH_SECRET),
      decodeBase64Url(parts[2]),
      new TextEncoder().encode(`${parts[0]}.${parts[1]}`),
    );
    if (!valid) return null;
    const actor = JSON.parse(new TextDecoder().decode(decodeBase64Url(parts[1]))) as Actor;
    if (!actor.id || actor.exp <= Math.floor(Date.now() / 1000)) return null;
    return actor;
  } catch {
    return null;
  }
}

function parseRow(row: StoredRow): Data {
  return {
    ...JSON.parse(row.data),
    id: row.id,
    collectionId: row.collection,
    collectionName: row.collection,
    created: row.created,
    updated: row.updated,
  };
}

async function rowById(env: Env, collection: string, id: string): Promise<StoredRow | null> {
  return env.DB.prepare("SELECT collection, id, data, created, updated FROM records WHERE collection = ? AND id = ?")
    .bind(collection, id)
    .first<StoredRow>();
}

async function collectionRows(env: Env, collection: string): Promise<StoredRow[]> {
  const result = await env.DB.prepare("SELECT collection, id, data, created, updated FROM records WHERE collection = ?")
    .bind(collection)
    .all<StoredRow>();
  return result.results || [];
}

function isPublished(record: Data): boolean {
  if (record.published !== true) return false;
  const publishedAt = String(record.published_at || "").trim();
  return !publishedAt || Date.parse(publishedAt) <= Date.now();
}

function canRead(collection: string, record: Data, actor: Actor | null): boolean {
  if (actor) return collection !== "app_secrets" || actor.role === "admin";
  if (!PUBLIC_COLLECTIONS.has(collection)) return false;
  if (collection === "settings") return true;
  if (collection === "media") return record.public === true;
  return isPublished(record);
}

function canWrite(collection: string, actor: Actor | null): boolean {
  if (!actor) return false;
  if (actor.role === "admin") return true;
  if (actor.role !== "editor") return false;
  return EDITOR_COLLECTIONS.has(collection) || collection === "settings";
}

function unescapeFilterValue(value: string): string {
  return value.replace(/\\"/g, '"').replace(/\\\\/g, "\\");
}

function matchesClause(record: Data, clause: string): boolean {
  const match = clause.trim().match(/^([a-zA-Z0-9_]+)\s*(=|!=|<=|>=|<|>|~)\s*(.+)$/);
  if (!match) return true;
  const [, field, operator, rawExpected] = match;
  let expected: unknown = rawExpected.trim();
  if (expected === "true") expected = true;
  else if (expected === "false") expected = false;
  else if (expected === "@now") expected = new Date().toISOString();
  else if (/^".*"$/.test(String(expected))) expected = unescapeFilterValue(String(expected).slice(1, -1));
  const actual = record[field];
  switch (operator) {
    case "=": return String(actual ?? "") === String(expected);
    case "!=": return String(actual ?? "") !== String(expected);
    case "~": return String(actual ?? "").toLocaleLowerCase().includes(String(expected).toLocaleLowerCase());
    case "<=": return String(actual ?? "") <= String(expected);
    case ">=": return String(actual ?? "") >= String(expected);
    case "<": return String(actual ?? "") < String(expected);
    case ">": return String(actual ?? "") > String(expected);
  }
  return true;
}

function matchesFilter(record: Data, filter: string): boolean {
  if (!filter.trim()) return true;
  return filter.split(/\s+\|\|\s+/).some((orPart) =>
    orPart.split(/\s+&&\s+/).every((clause) => matchesClause(record, clause.replace(/^\(|\)$/g, ""))),
  );
}

function sortRecords(records: Data[], sort: string): Data[] {
  const fields = (sort || "-created").split(",").map((field) => field.trim()).filter(Boolean);
  return records.sort((left, right) => {
    for (const input of fields) {
      const desc = input.startsWith("-");
      const field = input.replace(/^[+-]/, "");
      const a = String(left[field] ?? "");
      const b = String(right[field] ?? "");
      if (a === b) continue;
      return (a < b ? -1 : 1) * (desc ? -1 : 1);
    }
    return 0;
  });
}

function selectFields(record: Data, fields: string): Data {
  if (!fields) return record;
  const selected: Data = {};
  for (const field of fields.split(",")) {
    const key = field.trim();
    if (key && key in record) selected[key] = record[key];
  }
  return selected;
}

async function bodyData(request: Request): Promise<{ data: Data; file?: File }> {
  const contentType = request.headers.get("content-type") || "";
  if (contentType.includes("multipart/form-data")) {
    const form = await request.formData();
    const data: Data = {};
    let file: File | undefined;
    form.forEach((value, key) => {
      if (value instanceof File) {
        if (value.size > 10 * 1024 * 1024) throw new Error("Files must be 10 MB or smaller.");
        file = value;
        data[key] = value.name;
      } else if (value === "true" || value === "false") {
        data[key] = value === "true";
      } else {
        data[key] = value;
      }
    });
    return { data, file };
  }
  return { data: await request.json<Data>() };
}

function safeFilename(value: string): string {
  const cleaned = value.normalize("NFKC").replace(/[^a-zA-Z0-9._-]/g, "_").replace(/^\.+/, "");
  return cleaned.slice(0, 180) || "upload.bin";
}

async function upsertRecord(env: Env, collection: string, id: string, data: Data, created?: string): Promise<Data> {
  const now = new Date().toISOString();
  await env.DB.prepare(
    "INSERT INTO records (collection, id, data, created, updated) VALUES (?, ?, ?, ?, ?) " +
    "ON CONFLICT(collection, id) DO UPDATE SET data = excluded.data, updated = excluded.updated",
  ).bind(collection, id, JSON.stringify(data), created || now, now).run();
  return { ...data, id, collectionId: collection, collectionName: collection, created: created || now, updated: now };
}

async function ensureUnique(env: Env, collection: string, data: Data, exceptId = ""): Promise<Response | null> {
  const uniqueFields = collection === "media" ? ["checksum"] : ["slug"];
  if (collection === "pages") uniqueFields.push("url");
  const records = (await collectionRows(env, collection)).map(parseRow);
  for (const field of uniqueFields) {
    const value = String(data[field] || "").trim();
    if (value && records.some((record) => record.id !== exceptId && String(record[field] || "") === value)) {
      return apiError(400, "Failed to create record.", { [field]: { code: `validation_not_unique`, message: "Value must be unique." } });
    }
  }
  return null;
}

async function listRecords(request: Request, env: Env, collection: string, actor: Actor | null): Promise<Response> {
  const url = new URL(request.url);
  const page = Math.max(1, Number(url.searchParams.get("page") || 1));
  const perPage = Math.min(500, Math.max(1, Number(url.searchParams.get("perPage") || 30)));
  const filter = url.searchParams.get("filter") || "";
  const fields = url.searchParams.get("fields") || "";
  let records = (await collectionRows(env, collection)).map(parseRow).filter((record) => canRead(collection, record, actor));
  records = sortRecords(records.filter((record) => matchesFilter(record, filter)), url.searchParams.get("sort") || "-created");
  const totalItems = records.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / perPage));
  const items = records.slice((page - 1) * perPage, page * perPage).map((record) => selectFields(record, fields));
  return json({ page, perPage, totalItems, totalPages, items });
}

async function recordsApi(request: Request, env: Env, collection: string, id: string | undefined, actor: Actor | null): Promise<Response> {
  if (!KNOWN_COLLECTIONS.has(collection)) return apiError(404, "The requested resource wasn't found.");
  if (request.method === "GET" && !id) return listRecords(request, env, collection, actor);

  if (request.method === "GET" && id) {
    const row = await rowById(env, collection, id);
    if (!row) return apiError(404, "The requested resource wasn't found.");
    const record = parseRow(row);
    if (!canRead(collection, record, actor)) return apiError(404, "The requested resource wasn't found.");
    return json(selectFields(record, new URL(request.url).searchParams.get("fields") || ""));
  }

  if (!actor) return apiError(401, "Authentication required.");
  if (!canWrite(collection, actor)) return apiError(403, "You are not allowed to perform this request.");

  if (request.method === "DELETE" && id) {
    const row = await rowById(env, collection, id);
    if (!row) return apiError(404, "The requested resource wasn't found.");
    if (collection === "media") {
      const record = parseRow(row);
      const filename = String(record.file || "");
      const path = String(record.path || "").replace(/^\/uploads\//, "");
      if (filename) await env.MEDIA.delete(`media/${id}/${filename}`);
      if (path) await env.MEDIA.delete(`uploads/${path}`);
    }
    await env.DB.prepare("DELETE FROM records WHERE collection = ? AND id = ?").bind(collection, id).run();
    return new Response(null, { status: 204 });
  }

  if ((request.method === "POST" && !id) || (request.method === "PATCH" && id)) {
    let parsed: { data: Data; file?: File };
    try {
      parsed = await bodyData(request);
    } catch (error) {
      return apiError(400, error instanceof Error ? error.message : "Invalid request body.");
    }
    const existing = id ? await rowById(env, collection, id) : null;
    if (id && !existing) return apiError(404, "The requested resource wasn't found.");
    const recordId = id || randomId();
    const existingData = existing ? JSON.parse(existing.data) as Data : {};
    const next = { ...existingData, ...parsed.data };
    const uniqueError = await ensureUnique(env, collection, next, id);
    if (uniqueError) return uniqueError;

    if (collection === "media" && parsed.file) {
      const mediaCount = await env.DB.prepare("SELECT COUNT(*) AS count FROM records WHERE collection = 'media'").first<{ count: number }>();
      if ((mediaCount?.count || 0) >= 5000) return apiError(400, "The free-tier media limit of 5,000 files has been reached.");
      const filename = safeFilename(parsed.file.name);
      next.file = filename;
      next.file_size = parsed.file.size;
      const checksum = String(next.checksum || "").toLowerCase();
      const extension = filename.includes(".") ? `.${filename.split(".").pop()!.toLowerCase()}` : "";
      const alias = checksum ? `${checksum}${extension}` : filename;
      next.path = next.path || `/uploads/${alias}`;
      const httpMetadata = { contentType: parsed.file.type || "application/octet-stream" };
      await env.MEDIA.put(`uploads/${alias}`, parsed.file.stream(), { httpMetadata });
    }

    const record = await upsertRecord(env, collection, recordId, next, existing?.created);
    return json(record, existing ? 200 : 200);
  }

  return apiError(405, "Method not allowed.");
}

async function authWithPassword(request: Request, env: Env): Promise<Response> {
  const input = await request.json<{ identity?: string; password?: string }>();
  const identity = String(input.identity || "").trim().toLowerCase();
  const rows = await collectionRows(env, "cms_users");
  const row = rows.find((item) => String((JSON.parse(item.data) as Data).email || "").toLowerCase() === identity);
  if (!row) return apiError(400, "Failed to authenticate.", { identity: { message: "Invalid login credentials." } });
  const credential = await env.DB.prepare("SELECT password_hash FROM auth_credentials WHERE collection = 'cms_users' AND record_id = ?")
    .bind(row.id).first<{ password_hash: string }>();
  if (!credential || !(await bcrypt.compare(String(input.password || ""), credential.password_hash))) {
    return apiError(400, "Failed to authenticate.", { identity: { message: "Invalid login credentials." } });
  }
  const record = parseRow(row);
  const actor = { id: row.id, email: String(record.email), name: String(record.name || ""), role: String(record.role || "viewer") };
  return json({ token: await createToken(actor, env.AUTH_SECRET), record });
}

async function refreshAuthentication(request: Request, env: Env): Promise<Response> {
  const actor = await actorFromRequest(request, env);
  if (!actor) return apiError(401, "Authentication required.");
  const row = await rowById(env, "cms_users", actor.id);
  if (!row) return apiError(401, "Authentication required.");
  const record = parseRow(row);
  const refreshedActor = {
    id: row.id,
    email: String(record.email || ""),
    name: String(record.name || ""),
    role: String(record.role || "viewer"),
  };
  return json({ token: await createToken(refreshedActor, env.AUTH_SECRET), record });
}

async function bootstrap(request: Request, env: Env): Promise<Response> {
  if (request.headers.get("x-bootstrap-secret") !== env.AUTH_SECRET) return apiError(403, "Bootstrap authorization failed.");
  const count = await env.DB.prepare("SELECT COUNT(*) AS count FROM records WHERE collection = 'cms_users'").first<{ count: number }>();
  if ((count?.count || 0) > 0) return json({ created: false, message: "Administrator already exists." });
  const input = await request.json<{ email?: string; password?: string }>();
  const email = String(input.email || "").trim().toLowerCase();
  const password = String(input.password || "");
  if (!email.includes("@") || password.length < 12) return apiError(400, "A valid email and a password of at least 12 characters are required.");
  const id = randomId();
  const record = await upsertRecord(env, "cms_users", id, { email, emailVisibility: true, name: "Administrator", role: "admin", verified: true });
  const hash = await bcrypt.hash(password, 10);
  await env.DB.prepare("INSERT INTO auth_credentials (collection, record_id, password_hash) VALUES ('cms_users', ?, ?)").bind(id, hash).run();
  await upsertRecord(env, "settings", randomId(), DEFAULT_SETTINGS);
  return json({ created: true, id: record.id });
}

async function serveR2(request: Request, env: Env, key: string): Promise<Response> {
  const object = await env.MEDIA.get(key);
  if (!object) return new Response("Not found", { status: 404 });
  const headers = new Headers();
  object.writeHttpMetadata(headers);
  headers.set("etag", object.httpEtag);
  headers.set("cache-control", "public, max-age=31536000, immutable");
  return new Response(object.body, { headers });
}

function escapeHtml(value: unknown): string {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[char]!);
}

function asNumber(value: unknown, fallback: number): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

async function settings(env: Env): Promise<Data> {
  const rows = await collectionRows(env, "settings");
  return rows.length ? { ...DEFAULT_SETTINGS, ...parseRow(rows[0]) } : DEFAULT_SETTINGS;
}

function layout(config: Data, title: string, body: string, pages: Data[], request: Request): Response {
  const siteName = String(config.site_name || DEFAULT_SETTINGS.site_name);
  const theme = String(config.theme || "ember").replace(/[^a-z0-9_-]/gi, "");
  const canonical = new URL(request.url).origin + new URL(request.url).pathname;
  const menu = pages.filter(isPublished).sort((a, b) => Number(a.menuOrder || 0) - Number(b.menuOrder || 0))
    .filter((page) => page.menuVisible === true)
    .map((page) => `<a href="${escapeHtml(page.url)}">${escapeHtml(page.menuTitle || page.title)}</a>`).join("");
  const html = `<!doctype html><html lang="${escapeHtml(config.site_language || "ja")}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${escapeHtml(title === siteName ? title : `${title} | ${siteName}`)}</title><meta name="description" content="${escapeHtml(config.description)}"><link rel="canonical" href="${escapeHtml(canonical)}"><link rel="stylesheet" href="/themes/${theme}/styles.css"><link rel="stylesheet" href="/styles.css"></head><body><header><nav><a href="/">${escapeHtml(siteName)}</a><a href="/archive/">Archive</a>${menu}</nav></header><main>${body}</main><footer>${String(config.footer_html || "")}</footer></body></html>`;
  return new Response(html, { headers: { "content-type": "text/html; charset=utf-8", "cache-control": "public, max-age=60" } });
}

async function publicSite(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  const config = await settings(env);
  const pages = (await collectionRows(env, "pages")).map(parseRow);
  const posts = sortRecords((await collectionRows(env, "posts")).map(parseRow).filter(isPublished), "-published_at");
  const translations = (await collectionRows(env, "post_translations")).map(parseRow).filter(isPublished);

  if (url.pathname === "/robots.txt") {
    return new Response(`User-agent: *\nAllow: /\nSitemap: ${url.origin}/sitemap.xml\n`, { headers: { "content-type": "text/plain; charset=utf-8" } });
  }
  if (url.pathname === "/sitemap.xml") {
    const locations = ["/", "/archive/", ...pages.filter(isPublished).map((page) => String(page.url)), ...posts.map((post) => `/posts/${post.slug}/`)];
    return new Response(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${locations.map((path) => `<url><loc>${escapeHtml(url.origin + path)}</loc></url>`).join("")}</urlset>`, { headers: { "content-type": "application/xml; charset=utf-8" } });
  }
  if (url.pathname === "/feed.json" && config.enable_feed_json === true) {
    return json({ version: "https://jsonfeed.org/version/1.1", title: config.site_name, home_page_url: `${url.origin}/`, feed_url: `${url.origin}/feed.json`, items: posts.slice(0, asNumber(config.feed_items_limit, 20)).map((post) => ({ id: `${url.origin}/posts/${post.slug}/`, url: `${url.origin}/posts/${post.slug}/`, title: post.title, content_html: post.body, date_published: post.published_at })) });
  }
  if (url.pathname === "/feed.xml" && config.enable_feed_xml === true) {
    const items = posts.slice(0, asNumber(config.feed_items_limit, 20)).map((post) => `<item><title>${escapeHtml(post.title)}</title><link>${escapeHtml(`${url.origin}/posts/${post.slug}/`)}</link><description>${escapeHtml(post.excerpt || "")}</description><pubDate>${escapeHtml(post.published_at)}</pubDate></item>`).join("");
    return new Response(`<?xml version="1.0"?><rss version="2.0"><channel><title>${escapeHtml(config.site_name)}</title><link>${url.origin}/</link>${items}</channel></rss>`, { headers: { "content-type": "application/rss+xml; charset=utf-8" } });
  }
  if (url.pathname === "/") {
    const cards = posts.slice(0, asNumber(config.home_page_size, 3)).map((post) => `<article><h2><a href="/posts/${escapeHtml(post.slug)}/">${escapeHtml(post.title)}</a></h2><p>${escapeHtml(post.excerpt || "")}</p></article>`).join("");
    return layout(config, String(config.site_name), `<section><h1>${escapeHtml(config.welcome_text)}</h1><img src="${escapeHtml(config.home_top_image)}" alt="${escapeHtml(config.home_top_image_alt)}"></section><section>${cards || "<p>No published posts yet.</p>"}</section>`, pages, request);
  }
  if (url.pathname === "/archive" || url.pathname.startsWith("/archive/")) {
    const query = (url.searchParams.get("q") || "").toLowerCase();
    const filtered = query ? posts.filter((post) => [post.title, post.tags, post.category].some((value) => String(value || "").toLowerCase().includes(query))) : posts;
    const items = filtered.map((post) => `<li><a href="/posts/${escapeHtml(post.slug)}/">${escapeHtml(post.title)}</a> <time>${escapeHtml(String(post.published_at || "").slice(0, 10))}</time></li>`).join("");
    return layout(config, "Archive", `<h1>Archive</h1><ul>${items}</ul>`, pages, request);
  }
  const localized = url.pathname.match(/^\/([a-zA-Z0-9-]+)\/posts\/([^/]+)\/?$/);
  if (localized) {
    const post = translations.find((item) => String(item.locale).toLowerCase() === localized[1].toLowerCase() && item.slug === decodeURIComponent(localized[2]));
    if (post) return layout(config, String(post.title), `<article><h1>${escapeHtml(post.title)}</h1><div>${String(post.body || "")}</div></article>`, pages, request);
  }
  const postMatch = url.pathname.match(/^\/posts\/([^/]+)\/?$/);
  if (postMatch) {
    const post = posts.find((item) => item.slug === decodeURIComponent(postMatch[1]));
    if (post) return layout(config, String(post.title), `<article><h1>${escapeHtml(post.title)}</h1><div>${String(post.body || "")}</div></article>`, pages, request);
  }
  const page = pages.find((item) => isPublished(item) && item.url === url.pathname);
  if (page) return layout(config, String(page.title), `<article><h1>${escapeHtml(page.title)}</h1><div>${String(page.body || "")}</div></article>`, pages, request);
  return layout(config, "Not Found", "<h1>Not Found</h1>", pages, request);
}

async function handle(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  if (url.pathname === "/healthz") return json({ ok: true, service: "alleycat", environment: env.ENVIRONMENT });
  if (url.pathname === "/admin") return Response.redirect(`${url.origin}/admin/`, 308);
  if (url.pathname.startsWith("/admin/")) {
    const asset = await env.ASSETS.fetch(request);
    if (asset.status >= 200 && asset.status < 300) return asset;
    return env.ASSETS.fetch(new Request(new URL("/admin/index.html", request.url), request));
  }
  if (url.pathname === "/api/bootstrap" && request.method === "POST") return bootstrap(request, env);
  if (url.pathname === "/api/collections/cms_users/auth-with-password" && request.method === "POST") return authWithPassword(request, env);
  if (url.pathname === "/api/collections/cms_users/auth-refresh" && request.method === "POST") return refreshAuthentication(request, env);

  const actor = await actorFromRequest(request, env);
  if (url.pathname === "/api/ai/slug/status") return json({ enabled: Boolean(actor) });
  if (url.pathname === "/api/ai/slug" && request.method === "POST") {
    if (!actor) return apiError(401, "Authentication required.");
    const input = await request.json<{ title?: string }>();
    const slug = String(input.title || "").normalize("NFKD").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 100) || `post-${Date.now()}`;
    return json({ slug });
  }

  const recordsMatch = url.pathname.match(/^\/api\/collections\/([^/]+)\/records(?:\/([^/]+))?$/);
  if (recordsMatch) return recordsApi(request, env, decodeURIComponent(recordsMatch[1]), recordsMatch[2] ? decodeURIComponent(recordsMatch[2]) : undefined, actor);
  const fileMatch = url.pathname.match(/^\/api\/files\/([^/]+)\/([^/]+)\/([^/]+)$/);
  if (fileMatch) {
    const row = await rowById(env, decodeURIComponent(fileMatch[1]), decodeURIComponent(fileMatch[2]));
    if (!row) return new Response("Not found", { status: 404 });
    const record = parseRow(row);
    const key = String(record.path || "").replace(/^\/uploads\//, "");
    return key ? serveR2(request, env, `uploads/${key}`) : new Response("Not found", { status: 404 });
  }
  if (url.pathname.startsWith("/uploads/")) {
    const media = await serveR2(request, env, `uploads/${url.pathname.slice(9)}`);
    return media.status === 404 ? env.ASSETS.fetch(request) : media;
  }
  if (url.pathname.startsWith("/api/")) return apiError(404, "The requested resource wasn't found.");

  const asset = await env.ASSETS.fetch(request);
  if (asset.status !== 404) return asset;
  return publicSite(request, env);
}

function corsResponse(request: Request, env: Env, response: Response): Response {
  const origin = request.headers.get("origin");
  if (!origin || origin !== env.CMS_ORIGIN) return response;
  const headers = new Headers(response.headers);
  headers.set("access-control-allow-origin", origin);
  headers.set("access-control-allow-methods", "GET, POST, PATCH, DELETE, OPTIONS");
  headers.set("access-control-allow-headers", "Authorization, Content-Type");
  headers.set("vary", "Origin");
  return new Response(response.body, { status: response.status, statusText: response.statusText, headers });
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    try {
      if (request.method === "OPTIONS" && request.headers.get("origin") === env.CMS_ORIGIN) {
        return corsResponse(request, env, new Response(null, { status: 204 }));
      }
      return corsResponse(request, env, await handle(request, env));
    } catch (error) {
      console.error("request failed", error);
      return apiError(500, "Internal Server Error");
    }
  },
} satisfies ExportedHandler<Env>;
