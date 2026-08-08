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

type LoginAttempt = {
  failures: number;
  windowStartedAt: number;
  blockedUntil: number;
  lastSeenAt: number;
};

const JSON_HEADERS = { "content-type": "application/json; charset=utf-8" };
const CONTENT_SECURITY_POLICY = "default-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'; object-src 'none'; img-src 'self' data: blob: https:; style-src 'self' 'unsafe-inline' https:; font-src 'self' data: https:; script-src 'self'; connect-src 'self' https:; frame-src 'self' https:; worker-src 'self' blob:";
const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;
const MAX_UPLOAD_PIXELS = 25_000_000;
const MAX_JSON_BODY_BYTES = 1 * 1024 * 1024;
const DB_PAGE_SIZE = 200;
const LOGIN_WINDOW_MS = 5 * 60 * 1000;
const LOGIN_BLOCK_MS = 15 * 60 * 1000;
const LOGIN_FAILURE_LIMIT = 10;
const MAX_LOGIN_TRACKERS = 10_000;
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

const loginAttempts = new Map<string, LoginAttempt>();
const PUBLIC_SETTINGS_FIELDS = new Set(Object.keys(DEFAULT_SETTINGS));

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
    if (!actor.id || !Number.isFinite(actor.exp) || actor.exp <= Math.floor(Date.now() / 1000)) return null;
    const row = await rowById(env, "cms_users", actor.id);
    if (!row) return null;
    const record = parseRow(row);
    const role = String(record.role || "viewer");
    if (!new Set(["admin", "editor", "viewer"]).has(role)) return null;
    return {
      ...actor,
      email: String(record.email || actor.email || ""),
      name: String(record.name || actor.name || ""),
      role,
    };
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

async function collectionRowsPage(env: Env, collection: string, page: number, perPage = DB_PAGE_SIZE): Promise<StoredRow[]> {
  const offset = Math.max(0, page - 1) * perPage;
  const result = await env.DB.prepare(
    "SELECT collection, id, data, created, updated FROM records WHERE collection = ? ORDER BY created DESC, id DESC LIMIT ? OFFSET ?",
  ).bind(collection, perPage, offset).all<StoredRow>();
  return result.results || [];
}

async function collectionRows(env: Env, collection: string): Promise<StoredRow[]> {
  const rows: StoredRow[] = [];
  for (let page = 1; ; page += 1) {
    const pageRows = await collectionRowsPage(env, collection, page);
    rows.push(...pageRows);
    if (pageRows.length < DB_PAGE_SIZE) return rows;
  }
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

function publicRecord(collection: string, record: Data): Data {
  if (collection !== "settings") return record;
  const safe: Data = {};
  for (const field of PUBLIC_SETTINGS_FIELDS) {
    if (field in record) safe[field] = record[field];
  }
  for (const field of ["id", "collectionId", "collectionName", "created", "updated"]) {
    if (field in record) safe[field] = record[field];
  }
  return safe;
}

async function bodyData(request: Request): Promise<{ data: Data; file?: File }> {
  const contentType = request.headers.get("content-type") || "";
  if (contentType.includes("multipart/form-data")) {
    const contentLength = Number(request.headers.get("content-length") || 0);
    if (Number.isFinite(contentLength) && contentLength > MAX_UPLOAD_BYTES + 1024 * 1024) {
      throw new Error("Multipart requests must be 11 MB or smaller.");
    }
    const form = await request.formData();
    const data: Data = {};
    let file: File | undefined;
    form.forEach((value, key) => {
      if (value instanceof File) {
        if (value.size > MAX_UPLOAD_BYTES) throw new Error("Files must be 10 MB or smaller.");
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
  const contentLength = Number(request.headers.get("content-length") || 0);
  if (Number.isFinite(contentLength) && contentLength > MAX_JSON_BODY_BYTES) {
    throw new Error("JSON requests must be 1 MB or smaller.");
  }
  const body = await readRequestBytes(request, MAX_JSON_BODY_BYTES);
  try {
    return { data: JSON.parse(new TextDecoder().decode(body)) as Data };
  } catch {
    throw new Error("Invalid JSON request body.");
  }
}

async function readRequestBytes(request: Request, maxBytes: number): Promise<Uint8Array> {
  if (!request.body) return new Uint8Array();
  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > maxBytes) {
        await reader.cancel();
        throw new Error(`Request body must be ${Math.floor(maxBytes / (1024 * 1024))} MB or smaller.`);
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  const body = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    body.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return body;
}

function safeFilename(value: string): string {
  const cleaned = value.normalize("NFKC").replace(/[^a-zA-Z0-9._-]/g, "_").replace(/^\.+/, "");
  return cleaned.slice(0, 180) || "upload.bin";
}

type ImageDimensions = { width: number; height: number };

function readImageDimensions(bytes: Uint8Array, contentType: string): ImageDimensions | null {
  if (contentType === "image/png" && bytes.length >= 24 && bytes.slice(0, 8).every((value, index) => value === [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a][index])) {
    return { width: readUint32BE(bytes, 16), height: readUint32BE(bytes, 20) };
  }
  if (contentType === "image/jpeg") return readJpegDimensions(bytes);
  if (contentType === "image/webp") return readWebpDimensions(bytes);
  return null;
}

function readUint32BE(bytes: Uint8Array, offset: number): number {
  return ((bytes[offset] << 24) | (bytes[offset + 1] << 16) | (bytes[offset + 2] << 8) | bytes[offset + 3]) >>> 0;
}

function readJpegDimensions(bytes: Uint8Array): ImageDimensions | null {
  if (bytes.length < 4 || bytes[0] !== 0xff || bytes[1] !== 0xd8) return null;
  let offset = 2;
  while (offset + 9 < bytes.length) {
    if (bytes[offset] !== 0xff) {
      offset += 1;
      continue;
    }
    const marker = bytes[offset + 1];
    offset += 2;
    if (marker === 0xd8 || marker === 0xd9) continue;
    if (offset + 2 > bytes.length) return null;
    const length = (bytes[offset] << 8) | bytes[offset + 1];
    if (length < 2 || offset + length > bytes.length) return null;
    const isFrame = (marker >= 0xc0 && marker <= 0xc3) || (marker >= 0xc5 && marker <= 0xc7) || (marker >= 0xc9 && marker <= 0xcb) || (marker >= 0xcd && marker <= 0xcf);
    if (isFrame && length >= 7) return { width: (bytes[offset + 5] << 8) | bytes[offset + 6], height: (bytes[offset + 3] << 8) | bytes[offset + 4] };
    offset += length;
  }
  return null;
}

function readWebpDimensions(bytes: Uint8Array): ImageDimensions | null {
  if (bytes.length < 30 || String.fromCharCode(...bytes.slice(0, 4)) !== "RIFF" || String.fromCharCode(...bytes.slice(8, 12)) !== "WEBP") return null;
  const type = String.fromCharCode(...bytes.slice(12, 16));
  if (type === "VP8X") {
    const width = 1 + bytes[24] + (bytes[25] << 8) + (bytes[26] << 16);
    const height = 1 + bytes[27] + (bytes[28] << 8) + (bytes[29] << 16);
    return { width, height };
  }
  if (type === "VP8 ") {
    if (bytes.length < 30 || bytes[23] !== 0x9d || bytes[24] !== 0x01 || bytes[25] !== 0x2a) return null;
    return { width: ((bytes[26] | (bytes[27] << 8)) & 0x3fff), height: ((bytes[28] | (bytes[29] << 8)) & 0x3fff) };
  }
  if (type === "VP8L") {
    if (bytes.length < 25 || bytes[20] !== 0x2f) return null;
    const width = 1 + bytes[21] + ((bytes[22] & 0x3f) << 8);
    const height = 1 + ((bytes[22] >> 6) | (bytes[23] << 2) | ((bytes[24] & 0x0f) << 10));
    return { width, height };
  }
  return null;
}

async function validateImageFile(file: File): Promise<void> {
  if (file.size > MAX_UPLOAD_BYTES) throw new Error("Files must be 10 MB or smaller.");
  const bytes = new Uint8Array(await file.arrayBuffer());
  const dimensions = readImageDimensions(bytes, file.type);
  if (!dimensions || dimensions.width <= 0 || dimensions.height <= 0) throw new Error("The uploaded file is not a supported image.");
  if (dimensions.width * dimensions.height > MAX_UPLOAD_PIXELS) throw new Error("Images must be 25 megapixels or smaller.");
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
  for (const field of uniqueFields) {
    const value = String(data[field] || "").trim();
    const duplicate = value
      ? await env.DB.prepare(
          `SELECT id FROM records WHERE collection = ? AND id != ? AND json_extract(data, '$.${field}') = ? LIMIT 1`,
        ).bind(collection, exceptId, value).first<{ id: string }>()
      : null;
    if (duplicate) {
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
  let records = (await collectionRows(env, collection)).map(parseRow).filter((record) => canRead(collection, record, actor)).map((record) => publicRecord(collection, record));
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
    return json(selectFields(publicRecord(collection, record), new URL(request.url).searchParams.get("fields") || ""));
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
      await validateImageFile(parsed.file);
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
  const clientKey = request.headers.get("cf-connecting-ip") || "unknown";
  const blocked = loginBlocked(clientKey);
  if (blocked > 0) {
    const response = apiError(429, "Too many login attempts. Try again later.");
    response.headers.set("retry-after", String(Math.ceil(blocked / 1000)));
    return response;
  }
  let input: { identity?: string; password?: string };
  try {
    input = await readJsonBody(request, MAX_JSON_BODY_BYTES);
  } catch (error) {
    return apiError(400, error instanceof Error ? error.message : "Invalid request body.");
  }
  const identity = String(input.identity || "").trim().toLowerCase();
  const effectiveAttemptKey = `${clientKey}:${identity}`;
  const effectiveBlocked = loginBlocked(effectiveAttemptKey);
  if (effectiveBlocked > 0) {
    const response = apiError(429, "Too many login attempts. Try again later.");
    response.headers.set("retry-after", String(Math.ceil(effectiveBlocked / 1000)));
    return response;
  }
  const rows = await collectionRows(env, "cms_users");
  const row = rows.find((item) => String((JSON.parse(item.data) as Data).email || "").toLowerCase() === identity);
  if (!row) {
    recordLoginFailure(clientKey);
    if (effectiveAttemptKey !== clientKey) recordLoginFailure(effectiveAttemptKey);
    return apiError(400, "Failed to authenticate.", { identity: { message: "Invalid login credentials." } });
  }
  const credential = await env.DB.prepare("SELECT password_hash FROM auth_credentials WHERE collection = 'cms_users' AND record_id = ?")
    .bind(row.id).first<{ password_hash: string }>();
  if (!credential || !(await bcrypt.compare(String(input.password || ""), credential.password_hash))) {
    recordLoginFailure(clientKey);
    if (effectiveAttemptKey !== clientKey) recordLoginFailure(effectiveAttemptKey);
    return apiError(400, "Failed to authenticate.", { identity: { message: "Invalid login credentials." } });
  }
  loginAttempts.delete(clientKey);
  loginAttempts.delete(effectiveAttemptKey);
  const record = parseRow(row);
  const actor = { id: row.id, email: String(record.email), name: String(record.name || ""), role: String(record.role || "viewer") };
  return json({ token: await createToken(actor, env.AUTH_SECRET), record });
}

async function readJsonBody<T>(request: Request, maxBytes: number): Promise<T> {
  const body = await readRequestBytes(request, maxBytes);
  try {
    return JSON.parse(new TextDecoder().decode(body)) as T;
  } catch {
    throw new Error("Invalid JSON request body.");
  }
}

function loginBlocked(key: string): number {
  const now = Date.now();
  const entry = loginAttempts.get(key);
  if (!entry) return 0;
  entry.lastSeenAt = now;
  if (entry.blockedUntil > now) return entry.blockedUntil - now;
  if (now - entry.windowStartedAt >= LOGIN_WINDOW_MS) {
    loginAttempts.delete(key);
    return 0;
  }
  return 0;
}

function recordLoginFailure(key: string): void {
  const now = Date.now();
  for (const [trackedKey, entry] of loginAttempts) {
    if (entry.lastSeenAt + LOGIN_BLOCK_MS <= now) loginAttempts.delete(trackedKey);
  }
  if (loginAttempts.size >= MAX_LOGIN_TRACKERS && !loginAttempts.has(key)) {
    const oldest = [...loginAttempts.entries()].sort((left, right) => left[1].lastSeenAt - right[1].lastSeenAt)[0];
    if (oldest) loginAttempts.delete(oldest[0]);
  }
  const current = loginAttempts.get(key);
  if (!current || now - current.windowStartedAt >= LOGIN_WINDOW_MS) {
    loginAttempts.set(key, { failures: 1, windowStartedAt: now, blockedUntil: 0, lastSeenAt: now });
    return;
  }
  current.failures += 1;
  current.lastSeenAt = now;
  if (current.failures >= LOGIN_FAILURE_LIMIT) current.blockedUntil = now + LOGIN_BLOCK_MS;
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
  let input: { email?: string; password?: string };
  try {
    input = await readJsonBody(request, MAX_JSON_BODY_BYTES);
  } catch (error) {
    return apiError(400, error instanceof Error ? error.message : "Invalid request body.");
  }
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

function sanitizeHtml(value: unknown): string {
  return String(value ?? "")
    .replace(/<script\b[^>]*>[\s\S]*?<\/script\s*>/gi, "")
    .replace(/<style\b[^>]*>[\s\S]*?<\/style\s*>/gi, "")
    .replace(/<iframe\b[^>]*>[\s\S]*?<\/iframe\s*>/gi, "")
    .replace(/<\/?(?:embed|object|form|base|meta|link)\b[^>]*>/gi, "")
    .replace(/\s(?:on[a-z]+|style|srcdoc)\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]+)/gi, "")
    .replace(/\s(?:href|src|action)\s*=\s*(?:"\s*javascript:[^"]*"|'\s*javascript:[^']*'|\s*javascript:[^\s>]+)/gi, "");
}

function asNumber(value: unknown, fallback: number): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

async function settings(env: Env): Promise<Data> {
  const rows = await collectionRowsPage(env, "settings", 1, 1);
  return rows.length ? { ...DEFAULT_SETTINGS, ...publicRecord("settings", parseRow(rows[0])) } : DEFAULT_SETTINGS;
}

function layout(config: Data, title: string, body: string, pages: Data[], request: Request): Response {
  const siteName = String(config.site_name || DEFAULT_SETTINGS.site_name);
  const theme = String(config.theme || "ember").replace(/[^a-z0-9_-]/gi, "");
  const canonical = new URL(request.url).origin + new URL(request.url).pathname;
  const menu = pages.filter(isPublished).sort((a, b) => Number(a.menuOrder || 0) - Number(b.menuOrder || 0))
    .filter((page) => page.menuVisible === true)
    .map((page) => `<a href="${escapeHtml(page.url)}">${escapeHtml(page.menuTitle || page.title)}</a>`).join("");
  const html = `<!doctype html><html lang="${escapeHtml(config.site_language || "ja")}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${escapeHtml(title === siteName ? title : `${title} | ${siteName}`)}</title><meta name="description" content="${escapeHtml(config.description)}"><link rel="canonical" href="${escapeHtml(canonical)}"><link rel="stylesheet" href="/themes/${theme}/styles.css"><link rel="stylesheet" href="/styles.css"><script src="/site.js" defer></script></head><body><header><nav><a href="/">${escapeHtml(siteName)}</a><a href="/archive/">Archive</a>${menu}</nav></header><main>${sanitizeHtml(body)}</main><footer>${sanitizeHtml(config.footer_html)}</footer></body></html>`;
  return new Response(html, { headers: { "content-type": "text/html; charset=utf-8", "cache-control": "public, max-age=60", "content-security-policy": CONTENT_SECURITY_POLICY, "x-content-type-options": "nosniff", "referrer-policy": "strict-origin-when-cross-origin", "permissions-policy": "camera=(), microphone=(), geolocation=()" } });
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
    let input: { title?: string };
    try {
      input = await readJsonBody(request, MAX_JSON_BODY_BYTES);
    } catch (error) {
      return apiError(400, error instanceof Error ? error.message : "Invalid request body.");
    }
    const slug = String(input.title || "").normalize("NFKD").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 100) || `post-${Date.now()}`;
    return json({ slug });
  }

  const recordsMatch = url.pathname.match(/^\/api\/collections\/([^/]+)\/records(?:\/([^/]+))?$/);
  if (recordsMatch) return recordsApi(request, env, decodeURIComponent(recordsMatch[1]), recordsMatch[2] ? decodeURIComponent(recordsMatch[2]) : undefined, actor);
  const fileMatch = url.pathname.match(/^\/api\/files\/([^/]+)\/([^/]+)\/([^/]+)$/);
  if (fileMatch) {
    if (request.method !== "GET" && request.method !== "HEAD") return apiError(405, "Method not allowed.");
    const collection = decodeURIComponent(fileMatch[1]);
    if (collection !== "media") return new Response("Not found", { status: 404 });
    const row = await rowById(env, collection, decodeURIComponent(fileMatch[2]));
    if (!row) return new Response("Not found", { status: 404 });
    const record = parseRow(row);
    if (!canRead(collection, record, actor)) return new Response("Not found", { status: 404 });
    const key = String(record.path || "").replace(/^\/uploads\//, "");
    if (!key || key.includes("..") || key.includes("\\")) return new Response("Not found", { status: 404 });
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
