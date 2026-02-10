import http from "http";
import fs from "fs";
import { readFile, stat } from "fs/promises";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const publicDir = path.resolve(__dirname, "..", "public");
const defaultPublicDir = path.resolve(__dirname, "..", "default-public-asset");
const activePublicDir = (() => {
  const hasAssets = (dir) => {
    try {
      const entries = fs.readdirSync(dir);
      return entries.some((entry) => !entry.startsWith("."));
    } catch {
      return false;
    }
  };
  if (hasAssets(publicDir)) return publicDir;
  if (hasAssets(defaultPublicDir)) return defaultPublicDir;
  return publicDir;
})();

const PB_URL = process.env.PB_URL || "http://127.0.0.1:8090";
const PORT = Number(process.env.PORT || 5173);
const ADMIN_URL = process.env.ADMIN_URL || "http://admin:5174";
const DEFAULT_SITE_NAME = process.env.SITE_NAME || "Example Blog";
const DEFAULT_DESCRIPTION = process.env.SITE_DESCRIPTION || "A calm place to write.";
const DEFAULT_WELCOME = process.env.HOME_WELCOME || "Welcome to your blog";
const DEFAULT_TOP_IMAGE = process.env.HOME_TOP_IMAGE || "/default-hero.svg";
const DEFAULT_TOP_IMAGE_ALT = process.env.HOME_TOP_IMAGE_ALT || "Default hero image";
const FOOTER_HTML = process.env.FOOTER_HTML || "";
const ANALYTICS_URL = process.env.ANALYTICS_URL || "";
const ANALYTICS_SITE_ID = process.env.ANALYTICS_SITE_ID || "";
const ADS_CLIENT = process.env.ADS_CLIENT || "";

const mimeTypes = {
  ".css": "text/css",
  ".js": "application/javascript",
  ".mjs": "application/javascript",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".webp": "image/webp",
  ".svg": "image/svg+xml",
  ".woff2": "font/woff2",
  ".txt": "text/plain",
  ".ico": "image/x-icon",
  ".xml": "application/xml",
  ".json": "application/json",
};

const escapeHtml = (value = "") =>
  value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");

const formatDate = (value) => {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleDateString("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
};

const stripHtml = (value = "") =>
  value.replace(/<[^>]*>/g, " ").replace(/\s+/g, " ").trim();

const mediaFileRe = /(?:https?:\/\/[^"'\\s)]+)?\/api\/files\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_-]+)\/([^"'\\s)]+)/g;

const buildExcerpt = (value = "", length = 160) => {
  const text = stripHtml(value);
  if (text.length <= length) return text;
  return `${text.slice(0, length)}...`;
};

const parseTags = (value = "") =>
  value
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);

const fetchJson = async (url) => {
  const res = await fetch(url);
  if (!res.ok) {
    const body = await res.text();
    const error = new Error(`HTTP ${res.status}`);
    error.status = res.status;
    error.body = body;
    throw error;
  }
  return res.json();
};

const getPosts = async (params) => {
  const base = new URL(`${PB_URL}/api/collections/posts/records`);
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) base.searchParams.set(key, value);
  });
  return fetchJson(base.toString());
};

const getPagesMenu = async () => {
  try {
    const base = new URL(`${PB_URL}/api/collections/pages/records`);
    base.searchParams.set("perPage", "200");
    base.searchParams.set("filter", "published = true && menuVisible = true");
    base.searchParams.set("sort", "menuOrder");
    const data = await fetchJson(base.toString());
    return data.items || [];
  } catch {
    return [];
  }
};

const getPageByUrl = async (urlPath) => {
  const base = new URL(`${PB_URL}/api/collections/pages/records`);
  base.searchParams.set("filter", `url = \"${urlPath}\" && published = true`);
  base.searchParams.set("perPage", "1");
  const data = await fetchJson(base.toString());
  return data.items?.[0] || null;
};

const getPostBySlug = async (slug) => {
  const base = new URL(`${PB_URL}/api/collections/posts/records`);
  base.searchParams.set("filter", `slug = \"${slug}\" && published = true`);
  base.searchParams.set("perPage", "1");
  const data = await fetchJson(base.toString());
  return data.items?.[0] || null;
};

const getAdjacentPosts = async (post) => {
  if (!post) return { newer: null, older: null };
  const field = post.published_at ? "published_at" : post.date ? "date" : "";
  const value = post.published_at || post.date || "";
  if (!field || !value) return { newer: null, older: null };
  const safeValue = value.replace(/"/g, "");

  const fetchNearest = async (op, sort) => {
    try {
      const data = await getPosts({
        page: 1,
        perPage: 1,
        filter: `published = true && ${field} ${op} "${safeValue}"`,
        sort,
      });
      return data.items?.[0] || null;
    } catch {
      return null;
    }
  };

  const newer = await fetchNearest(">", field);
  const older = await fetchNearest("<", `-${field}`);
  return { newer, older };
};

const getMediaById = async (id) => {
  try {
    return await fetchJson(`${PB_URL}/api/collections/media/records/${id}`);
  } catch {
    return null;
  }
};

const getMediaByPath = async (mediaPath) => {
  try {
    const base = new URL(`${PB_URL}/api/collections/media/records`);
    base.searchParams.set("page", "1");
    base.searchParams.set("perPage", "1");
    base.searchParams.set(
      "filter",
      `path = \"${mediaPath.replace(/\\\\/g, "\\\\\\\\").replace(/\"/g, "\\\\\"")}\"`
    );
    const data = await fetchJson(base.toString());
    return data.items?.[0] || null;
  } catch {
    return null;
  }
};

const rewriteMediaUrls = async (body = "") => {
  const matches = [...body.matchAll(mediaFileRe)];
  if (matches.length === 0) return body;
  const cache = new Map();
  await Promise.all(
    matches.map(async (match) => {
      const collection = match[1];
      const id = match[2];
      if (collection !== "media" && collection !== "pbc_2708086759") return;
      if (cache.has(id)) return;
      const media = await getMediaById(id);
      const mediaPath = typeof media?.path === "string" ? media.path.trim() : "";
      const fallback = typeof media?.caption === "string" ? media.caption.trim() : "";
      if (mediaPath) {
        cache.set(id, mediaPath);
      } else if (fallback) {
        cache.set(id, fallback);
      }
    })
  );
  return body.replace(mediaFileRe, (full, collection, id, filename) => {
    const mediaPath = cache.get(id);
    if (!mediaPath) return `/api/files/${collection}/${id}/${filename}`;
    if (mediaPath.startsWith("http://") || mediaPath.startsWith("https://")) return mediaPath;
    return mediaPath.startsWith("/") ? mediaPath : `/${mediaPath}`;
  });
};

const themeStylesheet = (themeOverride) => {
  if (activePublicDir === publicDir) return "/styles.css";
  const raw = themeOverride || process.env.THEME || "ember";
  const theme = raw.trim().toLowerCase();
  return `/themes/${encodeURIComponent(theme)}/styles.css`;
};

const renderHead = (title = "Home", themeOverride = "") => `<!doctype html>
<html lang="ja">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>${escapeHtml(title)} - ${escapeHtml(DEFAULT_SITE_NAME)}</title>
    <meta name="supported-color-schemes" content="light dark" />
    <meta name="theme-color" content="hsl(220, 20%, 100%)" media="(prefers-color-scheme: light)" />
    <meta name="theme-color" content="hsl(220, 20%, 10%)" media="(prefers-color-scheme: dark)" />
    <link rel="stylesheet" href="${themeStylesheet(themeOverride)}" />
    <link rel="alternate" href="/feed.xml" type="application/atom+xml" title="${escapeHtml(DEFAULT_SITE_NAME)}" />
    <link rel="alternate" href="/feed.json" type="application/json" title="${escapeHtml(DEFAULT_SITE_NAME)}" />
    <link rel="icon" type="image/png" sizes="32x32" href="/favicon.png" />
    <meta name="description" content="${escapeHtml(DEFAULT_DESCRIPTION)}" />
    ${ANALYTICS_URL && ANALYTICS_SITE_ID ? `<script defer src="${escapeHtml(ANALYTICS_URL)}" data-website-id="${escapeHtml(ANALYTICS_SITE_ID)}"></script>` : ""}
    ${ADS_CLIENT ? `<script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${escapeHtml(ADS_CLIENT)}" crossorigin="anonymous"></script>` : ""}
  </head>
  <body>`;

const renderNav = (menuPages = []) => {
  const menuLinks = menuPages
    .map(
      (page) => `
        <li>
          <a href="${page.url}">${escapeHtml(page.menuTitle || page.title)}</a>
        </li>`
    )
    .join("");

  return `<nav class="navbar">
      <a href="/" class="navbar-home">
        <strong>${escapeHtml(DEFAULT_SITE_NAME)}</strong>
      </a>

      <ul class="navbar-links">
        <li><a href="/archive/">Archive</a></li>
        ${menuLinks}
        <li>
          <script>
            let theme = localStorage.getItem("theme") || (window.matchMedia("(prefers-color-scheme: dark)").matches
              ? "dark"
              : "light");
            document.documentElement.dataset.theme = theme;
            function changeTheme() {
              theme = theme === "dark" ? "light" : "dark";
              localStorage.setItem("theme", theme);
              document.documentElement.dataset.theme = theme;
            }
          </script>
          <button class="button" onclick="changeTheme()">
            <span class="icon">◐</span>
          </button>
        </li>
      </ul>
    </nav>`;
};

const renderFooter = () =>
  `${FOOTER_HTML ? `<footer class="footer">${FOOTER_HTML}</footer>` : ""}\n  </body>\n</html>`;

const proxyPocketBase = async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  if (!url.pathname.startsWith("/api/")) return false;

  const target = new URL(url.pathname, PB_URL);
  target.search = url.search;

  const body =
    req.method && !["GET", "HEAD"].includes(req.method) ? await readRequestBody(req) : undefined;

  const proxyRes = await fetch(target, {
    method: req.method,
    headers: req.headers,
    body,
  });

  res.writeHead(proxyRes.status, Object.fromEntries(proxyRes.headers.entries()));
  res.end(Buffer.from(await proxyRes.arrayBuffer()));
  return true;
};

const readRequestBody = (req) =>
  new Promise((resolve, reject) => {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => resolve(chunks.length ? Buffer.concat(chunks) : undefined));
    req.on("error", reject);
  });

const renderPagination = (baseUrl, pageNumber, totalPages) => {
  if (!totalPages || totalPages <= 1) return "";
  const prev = pageNumber > 1 ? pageNumber - 1 : null;
  const next = pageNumber < totalPages ? pageNumber + 1 : null;
  const linkFor = (page) => (page === 1 ? `${baseUrl}/` : `${baseUrl}/${page}/`);

  return `<nav class="page-pagination pagination">
    <ul>
      ${prev ? `<li class="pagination-prev"><a href="${linkFor(prev)}" rel="prev"><span>Previous</span><strong>${prev}</strong></a></li>` : ""}
      ${next ? `<li class="pagination-next"><a href="${linkFor(next)}" rel="next"><span>Next</span><strong>${next}</strong></a></li>` : ""}
    </ul>
  </nav>`;
};

const renderTagsNav = (tags) => {
  if (!tags.length) return "";
  return `<nav class="page-navigation">
    <h2>tags:</h2>
    <ul class="page-navigation-tags">
      ${tags.map((tag) => `<li><a href="/archive/${encodeURIComponent(tag)}/" class="badge">${escapeHtml(tag)}</a></li>`).join("")}
    </ul>
  </nav>`;
};

const collectTags = async () => {
  const tags = new Set();
  let page = 1;
  const perPage = 200;
  while (true) {
    const data = await getPosts({ page, perPage, filter: "published = true", sort: "-date" });
    (data.items || []).forEach((post) => {
      parseTags(post.tags || "").forEach((tag) => tags.add(tag));
    });
    if (!data.items || data.items.length < perPage) break;
    page += 1;
  }
  return Array.from(tags).sort();
};

const renderPostList = (items = []) =>
  `<section class="postList">
    ${items
      .map((post) => {
        const body = post.body || post.content || "";
        const excerpt = post.excerpt || buildExcerpt(body);
        const tags = parseTags(post.tags || "");
        const tagsHtml =
          tags.length > 0
            ? `<div class="post-tags">${tags
                .map((tag) => `<a class="badge" href="/archive/${encodeURIComponent(tag)}/">${escapeHtml(tag)}</a>`)
                .join("")}</div>`
            : "";
        return `<article class="post">
          <header class="post-header">
            <h2 class="post-title">
              <a href="/posts/${post.slug}/">${escapeHtml(post.title || post.slug)}</a>
            </h2>
            <div class="post-details">
              ${post.published_at || post.date ? `<p><time datetime="${post.published_at || post.date}">${formatDate(post.published_at || post.date)}</time></p>` : ""}
              <p>${Math.max(1, Math.ceil(stripHtml(body).length / 700))} min</p>
              ${tagsHtml}
            </div>
          </header>
          <div class="post-excerpt body">${excerpt}</div>
          <a href="/posts/${post.slug}/" class="post-link">Read →</a>
        </article>`;
      })
      .join("")}
  </section>`;

const renderHome = async (themeOverride = "") => {
  const menuPages = await getPagesMenu();
  let posts;
  try {
    posts = await getPosts({ page: 1, perPage: 3, filter: "published = true", sort: "-published_at" });
  } catch {
    posts = await getPosts({ page: 1, perPage: 3, filter: "published = true", sort: "-date" });
  }
  const items = posts.items || [];

  return (
    renderHead("Home", themeOverride) +
    renderNav(menuPages) +
    `<main class="body-home">
      <header class="page-header">
        ${DEFAULT_TOP_IMAGE ? `<img src="${escapeHtml(DEFAULT_TOP_IMAGE)}" alt="${escapeHtml(DEFAULT_TOP_IMAGE_ALT)}" class="top-image" />` : ""}
        <h1 class="page-title">${escapeHtml(DEFAULT_WELCOME)}</h1>
      </header>
      ${renderPostList(items)}
      <hr>
      <p>More posts can be found in <a href="/archive/">the archive</a>.</p>
    </main>` +
    renderFooter()
  );
};

const renderArchive = async (tag, pageNumber, themeOverride = "") => {
  const menuPages = await getPagesMenu();
  const filter = tag
    ? `published = true && tags ~ \"${tag}\"`
    : "published = true";
  let posts;
  try {
    posts = await getPosts({ page: pageNumber, perPage: 10, filter, sort: "-published_at" });
  } catch {
    posts = await getPosts({ page: pageNumber, perPage: 10, filter, sort: "-date" });
  }

  const title = tag ? `tag: ${tag}` : "Archive";
  const pagination = renderPagination(tag ? `/archive/${encodeURIComponent(tag)}` : "/archive", pageNumber, posts.totalPages || 1);
  const tagsNav = !tag && pageNumber === 1 ? renderTagsNav(await collectTags()) : "";

  return (
    renderHead(title, themeOverride) +
    renderNav(menuPages) +
    `<main class="body-tag">
      <header class="page-header">
        <h1 class="page-title">${escapeHtml(title)}</h1>
        <p>RSS: <a href="/feed.xml">Atom</a>, <a href="/feed.json">JSON</a></p>
        <div class="search" id="search"></div>
      </header>
      ${renderPostList(posts.items || [])}
      ${pagination}
      ${tagsNav}
    </main>` +
    renderFooter()
  );
};

const renderPost = async (slug, themeOverride = "") => {
  const menuPages = await getPagesMenu();
  const post = await getPostBySlug(slug);
  if (!post) return renderNotFound();
  const rawBody = post.body || post.content || "";
  const body = await rewriteMediaUrls(rawBody);
  const tags = parseTags(post.tags || "");
  const { newer, older } = await getAdjacentPosts(post);
  const navHtml =
    newer || older
      ? `<nav class="page-pagination pagination post-pagination">
    <ul>
      ${older ? `<li class="pagination-prev"><a href="/posts/${encodeURIComponent(older.slug)}/" rel="prev"><span>← Older post</span><strong>${escapeHtml(older.title || "Post")}</strong></a></li>` : ""}
      ${newer ? `<li class="pagination-next"><a href="/posts/${encodeURIComponent(newer.slug)}/" rel="next"><span>Newer post →</span><strong>${escapeHtml(newer.title || "Post")}</strong></a></li>` : ""}
    </ul>
  </nav>`
      : "";

  return (
    renderHead(post.title || "Post", themeOverride) +
    renderNav(menuPages) +
    `<main class="body-post">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">${escapeHtml(post.title || "")}</h1>
          <div class="post-details">
            ${post.published_at || post.date ? `<p><time datetime="${post.published_at || post.date}">${formatDate(post.published_at || post.date)}</time></p>` : ""}
            <p>${Math.max(1, Math.ceil(stripHtml(body).length / 700))} min</p>
            ${post.category ? `<p>${escapeHtml(post.category)}</p>` : ""}
            ${tags.length ? `<div class="post-tags">${tags
              .map((tag) => `<a class="badge" href="/archive/${encodeURIComponent(tag)}/">${escapeHtml(tag)}</a>`)
              .join("")}</div>` : ""}
          </div>
        </header>
        <div class="post-body body">${body}</div>
      </article>
      ${navHtml}
    </main>` +
    renderFooter()
  );
};

const renderPage = async (urlPath, themeOverride = "") => {
  const menuPages = await getPagesMenu();
  const page = await getPageByUrl(urlPath);
  if (!page) return renderNotFound();
  const rawBody = page.body || page.content || "";
  const body = await rewriteMediaUrls(rawBody);

  return (
    renderHead(page.title || "Page", themeOverride) +
    renderNav(menuPages) +
    `<main class="body-tag">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">${escapeHtml(page.title || "")}</h1>
        </header>
        <div class="post-body body">${body}</div>
      </article>
    </main>` +
    renderFooter()
  );
};

const renderNotFound = async () => {
  const menuPages = await getPagesMenu();
  return (
    renderHead("Not Found") +
    renderNav(menuPages) +
    `<main class="body-post">
      <article class="post">
        <header class="post-header">
          <h1 class="post-title">Not Found</h1>
        </header>
        <div class="post-body body">ページが見つかりませんでした。</div>
      </article>
    </main>` +
    renderFooter()
  );
};

const serveStatic = async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  let pathname = decodeURIComponent(url.pathname);
  if (pathname === "/") return false;

  if (pathname === "/theme-status") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(
      JSON.stringify({
        publicAssets: activePublicDir === publicDir,
      })
    );
    return true;
  }

  const safePath = path.normalize(pathname).replace(/^\.+/, "");
  if (safePath.startsWith("/uploads/")) {
    const media = await getMediaByPath(safePath);
    if (media?.file) {
      const fileUrl = `${PB_URL}/api/files/media/${media.id}/${encodeURIComponent(media.file)}`;
      const proxyRes = await fetch(fileUrl);
      if (proxyRes.ok) {
        res.writeHead(proxyRes.status, {
          "Content-Type": proxyRes.headers.get("content-type") || "application/octet-stream",
          "Cache-Control": proxyRes.headers.get("cache-control") || "public, max-age=300",
        });
        res.end(Buffer.from(await proxyRes.arrayBuffer()));
        return true;
      }
    }
  }
  const filePath = path.join(activePublicDir, safePath);
  if (!filePath.startsWith(activePublicDir)) return false;

  try {
    const fileStat = await stat(filePath);
    if (fileStat.isDirectory()) return false;
    const data = await readFile(filePath);
    const ext = path.extname(filePath).toLowerCase();
    res.writeHead(200, { "Content-Type": mimeTypes[ext] || "application/octet-stream" });
    res.end(data);
    return true;
  } catch {
    return false;
  }
};

const proxyAdmin = async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  if (!url.pathname.startsWith("/admin")) return false;

  const targetPath = url.pathname.replace("/admin", "") || "/";
  const target = new URL(targetPath, ADMIN_URL);
  target.search = url.search;

  const proxyHeaders = new Headers(req.headers);
  proxyHeaders.set("host", "localhost:5173");

  const proxyRes = await fetch(target, {
    method: req.method,
    headers: proxyHeaders,
  });

  const contentType = proxyRes.headers.get("content-type") || "";
  if (contentType.includes("text/html")) {
    const html = await proxyRes.text();
    const rewritten = html
      .replaceAll('"/@vite/', '"/admin/@vite/')
      .replaceAll('"/@react-refresh"', '"/admin/@react-refresh"')
      .replaceAll('"/src/', '"/admin/src/')
      .replaceAll('"/node_modules/', '"/admin/node_modules/');
    res.writeHead(proxyRes.status, { "Content-Type": "text/html; charset=utf-8" });
    res.end(rewritten);
    return true;
  }

  res.writeHead(proxyRes.status, Object.fromEntries(proxyRes.headers));
  res.end(await proxyRes.arrayBuffer());
  return true;
};

const server = http.createServer(async (req, res) => {
  try {
    if (await proxyPocketBase(req, res)) return;
    if (await serveStatic(req, res)) return;
    if (await proxyAdmin(req, res)) return;

    const url = new URL(req.url, `http://${req.headers.host}`);
    const pathName = url.pathname.endsWith("/") ? url.pathname : `${url.pathname}/`;
    const themeOverride = url.searchParams.get("theme") || "";

    if (pathName === "/") {
      const html = await renderHome(themeOverride);
      res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
      res.end(html);
      return;
    }

    if (pathName.startsWith("/archive/")) {
      const parts = pathName.split("/").filter(Boolean);
      let tag = parts[1] ? decodeURIComponent(parts[1]) : null;
      let pageNumber = 1;
      if (parts[1] && /^\d+$/.test(parts[1])) {
        pageNumber = Number(parts[1]) || 1;
        tag = null;
      } else if (parts[2]) {
        pageNumber = Number(parts[2]) || 1;
      }
      const html = await renderArchive(tag, pageNumber, themeOverride);
      res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
      res.end(html);
      return;
    }

    if (pathName.startsWith("/posts/")) {
      const parts = pathName.split("/").filter(Boolean);
      const slug = parts[1];
      const html = await renderPost(slug, themeOverride);
      res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
      res.end(html);
      return;
    }

    const html = await renderPage(pathName, themeOverride);
    res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
    res.end(html);
  } catch {
    if (!res.headersSent) {
      res.writeHead(500, { "Content-Type": "text/plain" });
      res.end("Internal Server Error");
      return;
    }
    res.end();
  }
});

server.listen(PORT, "0.0.0.0", () => {
  console.log(`SSR server running on :${PORT}`);
});
