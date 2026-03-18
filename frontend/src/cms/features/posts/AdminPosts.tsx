import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { pb, PostRecord } from "@shared/lib/pb";
import { formatDate } from "@shared/utils/text";
import {
  AdminButton,
  AdminConfirmDialog,
  AdminSelectField,
  AdminTable,
  AdminTextField,
} from "@cms/ui/AriaControls";
import FormStatusMessage from "@cms/ui/FormStatusMessage";
import useAdminPageTitle from "@cms/useAdminPageTitle";

const extractMediaIds = (value?: string) => {
  const ids = new Set<string>();
  const text = value ?? "";
  const re = /(?:https?:\/\/[^/"']+)?\/api\/files\/media\/([a-zA-Z0-9_-]+)\//g;
  let match;
  while ((match = re.exec(text)) !== null) {
    ids.add(match[1]);
  }
  return Array.from(ids);
};

export default function AdminPosts() {
  const [posts, setPosts] = useState<PostRecord[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkLoading, setBulkLoading] = useState(false);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [loading, setLoading] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);
  const [publishConfirmOpen, setPublishConfirmOpen] = useState(false);
  const [pendingPublishValue, setPendingPublishValue] = useState<boolean | null>(null);
  const [deleteTargetId, setDeleteTargetId] = useState<string | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [error, setError] = useState("");
  const [searchParams, setSearchParams] = useSearchParams();

  useAdminPageTitle("Posts");

  const query = searchParams.get("q") ?? "";
  const page = Math.max(1, Number(searchParams.get("page") || "1") || 1);
  const parsedPerPage = Number(searchParams.get("perPage") || "20");
  const perPage = [20, 50, 100].includes(parsedPerPage) ? parsedPerPage : 20;

  const buildFilter = (value: string) => {
    const safe = value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
    return `title ~ "${safe}" || slug ~ "${safe}" || tags ~ "${safe}" || category ~ "${safe}"`;
  };

  const updateParams = (updates: Record<string, string | number | null>) => {
    const next = new URLSearchParams(searchParams);
    Object.entries(updates).forEach(([key, value]) => {
      if (value === null || value === "" || value === 1 || value === 20) {
        next.delete(key);
      } else {
        next.set(key, String(value));
      }
    });
    setSearchParams(next, { replace: true });
  };

  useEffect(() => {
    let alive = true;
    const loadPosts = async () => {
      setLoading(true);
      setError("");
      try {
        const trimmed = query.trim();
        const filter = trimmed ? buildFilter(trimmed) : undefined;
        let res: { items: PostRecord[]; totalPages: number; totalItems: number } | null = null;
        try {
          res = await pb.collection("posts").getList<PostRecord>(page, perPage, {
            filter,
            sort: "-published_at",
          });
        } catch {
          try {
            res = await pb.collection("posts").getList<PostRecord>(page, perPage, {
              filter,
              sort: "-created",
            });
          } catch {
            res = null;
          }
        }
        if (!alive) return;
        if (res) {
          setPosts(res.items);
          setTotalPages(res.totalPages);
          setTotalItems(res.totalItems);
        } else {
          setPosts([]);
          setTotalPages(1);
          setTotalItems(0);
          setError("Posts could not be loaded. Refresh or adjust the current filters.");
        }
      } catch {
        if (!alive) return;
        setPosts([]);
        setTotalPages(1);
        setTotalItems(0);
        setError("Posts could not be loaded. Refresh or adjust the current filters.");
      } finally {
        if (alive) setLoading(false);
      }
    };
    loadPosts();
    return () => {
      alive = false;
    };
  }, [page, perPage, query, reloadToken]);

  useEffect(() => {
    setSelected(new Set());
  }, [posts, page, perPage, query]);

  const allFilteredSelected =
    posts.length > 0 && posts.every((post) => selected.has(post.id));

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const toggleSelectAll = () => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allFilteredSelected) {
        posts.forEach((post) => next.delete(post.id));
        return next;
      }
      posts.forEach((post) => next.add(post.id));
      return next;
    });
  };

  const bulkSetPublished = async (value: boolean) => {
    if (selected.size === 0) return;
    setBulkLoading(true);
    setError("");
    const now = new Date().toISOString();
    const byId = new Map(posts.map((post) => [post.id, post]));
    try {
      await Promise.all(
        Array.from(selected).map(async (id) => {
          const post = byId.get(id);
          const payload: Record<string, unknown> = { published: value };
          if (value && (!post?.published_at || post.published_at === "")) {
            payload.published_at = now;
          }
          await pb.collection("posts").update(id, payload);
        })
      );
      setSelected(new Set());
      setReloadToken((n) => n + 1);
    } catch {
      setError("Selected posts could not be updated. Try again.");
    } finally {
      setBulkLoading(false);
    }
  };

  const remove = async (id: string) => {
    setDeleteLoading(true);
    setError("");
    let mediaIds: string[] = [];
    let translationIds: string[] = [];
    try {
      const record = await pb.collection("posts").getOne(id);
      mediaIds = [
        ...extractMediaIds(record.body),
        ...extractMediaIds(record.content),
      ];
      const safeId = id.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
      const translations = await pb.collection("post_translations").getFullList({
        fields: "id,body,content",
        filter: `source_post = "${safeId}"`,
      });
      translationIds = translations.map((item: any) => item.id);
      translations.forEach((item: any) => {
        mediaIds.push(...extractMediaIds(item.body));
        mediaIds.push(...extractMediaIds(item.content));
      });
    } catch {
      mediaIds = [];
      translationIds = [];
    }

    try {
      await Promise.all(
        translationIds.map((translationId) => pb.collection("post_translations").delete(translationId))
      );
      await pb.collection("posts").delete(id);

      if (mediaIds.length > 0) {
        try {
          const [postsAll, pagesAll, translationsAll] = await Promise.all([
            pb.collection("posts").getFullList({ fields: "body,content" }),
            pb.collection("pages").getFullList({ fields: "body,content" }),
            pb.collection("post_translations").getFullList({ fields: "body,content" }),
          ]);
          const blobs = [
            ...postsAll.map((item: any) => `${item.body ?? ""} ${item.content ?? ""}`),
            ...pagesAll.map((item: any) => `${item.body ?? ""} ${item.content ?? ""}`),
            ...translationsAll.map((item: any) => `${item.body ?? ""} ${item.content ?? ""}`),
          ];
          for (const mediaId of mediaIds) {
            const marker = `/api/files/media/${mediaId}/`;
            const inUse = blobs.some((text) => text.includes(marker));
            if (!inUse) {
              await pb.collection("media").delete(mediaId);
            }
          }
        } catch {
          // ignore media cleanup errors
        }
      }

      setReloadToken((n) => n + 1);
    } catch {
      setError("This post could not be deleted. Try again.");
    } finally {
      setDeleteLoading(false);
    }
  };

  return (
    <section>
      <header className="admin-header">
        <div>
          <p className="admin-eyebrow">Posts</p>
          <h1>Posts</h1>
        </div>
        <Link className="admin-primary" to="/posts/new">
          New Post
        </Link>
      </header>
      <FormStatusMessage error={error} />
      <AdminConfirmDialog
        open={publishConfirmOpen && pendingPublishValue !== null}
        title={pendingPublishValue ? "Publish selected posts" : "Unpublish selected posts"}
        message={`Set ${selected.size} selected posts to ${pendingPublishValue ? "published" : "unpublished"}?`}
        confirmLabel={bulkLoading ? "Updating…" : "Update Posts"}
        confirmDisabled={bulkLoading || pendingPublishValue === null}
        onCancel={() => {
          setPublishConfirmOpen(false);
          setPendingPublishValue(null);
        }}
        onConfirm={() => {
          const next = pendingPublishValue;
          setPublishConfirmOpen(false);
          setPendingPublishValue(null);
          if (next !== null) void bulkSetPublished(next);
        }}
      />
      <AdminConfirmDialog
        open={deleteTargetId !== null}
        title="Delete post"
        message="This post and its translations will be removed immediately. Delete them?"
        confirmLabel={deleteLoading ? "Deleting…" : "Delete Post"}
        confirmDisabled={deleteLoading}
        onCancel={() => setDeleteTargetId(null)}
        onConfirm={() => {
          const next = deleteTargetId;
          setDeleteTargetId(null);
          if (next) void remove(next);
        }}
      />
      <div className="admin-stack">
        <section className="admin-toolbar admin-toolbar-section admin-filter-bar">
          <div className="admin-toolbar-heading">
            <p className="admin-section-label">Search and filter</p>
            <p className="admin-toolbar-note">
              {query.trim()
                ? `Filtering posts by "${query.trim()}".`
                : "Narrow the queue by title, slug, tags, or category."}
            </p>
          </div>
          <AdminTextField
            ariaLabel="Search posts"
            className="admin-input"
            label="Search"
            value={query}
            type="search"
            placeholder="Search title, slug, tags, or category…"
            onChange={(value) => {
              updateParams({ q: value || null, page: null });
            }}
          />
          <AdminSelectField
            ariaLabel="Rows per page"
            className="admin-field"
            label="Rows per page"
            value={perPage}
            onChange={(value) => {
              updateParams({ perPage: Number(value), page: null });
            }}
            options={[
              { value: 20, label: "20 / page" },
              { value: 50, label: "50 / page" },
              { value: 100, label: "100 / page" },
            ]}
          />
        </section>
      </div>
      {loading ? <p className="admin-note">Loading posts…</p> : null}
      <div className="admin-list-shell">
        {selected.size > 0 ? (
          <div className="admin-table-utility admin-list-strip is-active">
            <div className="admin-table-utility-copy">
              <p className="admin-table-selection">{selected.size} selected</p>
              <p className="admin-note">Publish or unpublish the selected rows.</p>
            </div>
            <div className="admin-toolbar-actions">
              <AdminButton
                className="admin-primary"
                disabled={bulkLoading}
                onPress={() => {
                  setPendingPublishValue(true);
                  setPublishConfirmOpen(true);
                }}
              >
                Publish
              </AdminButton>
              <AdminButton
                className="admin-secondary"
                disabled={bulkLoading}
                onPress={() => {
                  setPendingPublishValue(false);
                  setPublishConfirmOpen(true);
                }}
              >
                Unpublish
              </AdminButton>
            </div>
          </div>
        ) : null}
        <AdminTable
          ariaLabel="Posts"
          items={posts}
          columns={[
          {
            id: "select",
            className: "admin-table-select-column",
            name: (
              <input
                aria-label="Select all"
                checked={allFilteredSelected}
                className="admin-table-checkbox"
                onChange={toggleSelectAll}
                type="checkbox"
              />
            ),
            width: "50px",
            render: (item) => {
              const isChecked = selected.has(item.id);
              return (
              <input
                aria-label={`Select ${item.title}`}
                checked={isChecked}
                className="admin-table-checkbox"
                onChange={() => toggleSelect(item.id)}
                type="checkbox"
              />
            )},
          },
          {
            id: "title",
            name: "Title",
            mobileLabel: "Title",
            isRowHeader: true,
            render: (item) => (
              <div className="admin-table-title">
                <Link to={`/posts/${item.id}`}>{item.title}</Link>
                <span className="admin-table-subline">/{item.slug || item.id}/</span>
              </div>
            ),
          },
          {
            id: "date",
            name: "Date",
            mobileLabel: "Date",
            className: "admin-table-meta-column",
            width: "104px",
            render: (item) => formatDate(item.published_at) || "Not scheduled",
          },
          {
            id: "status",
            name: "Status",
            mobileLabel: "Status",
            className: "admin-table-status-column",
            width: "126px",
            render: (item) => (
              <span className={item.published ? "admin-status-badge is-published" : "admin-status-badge is-draft"}>
                {item.published ? "Published" : "Draft"}
              </span>
            ),
          },
          {
            id: "actions",
            name: "Action",
            mobileLabel: "Action",
            width: "72px",
            render: (item) => (
              <div className="admin-actions">
                <AdminButton ariaLabel={`Delete ${item.title}`} className="admin-danger-button" onPress={() => setDeleteTargetId(item.id)}>
                  🗑
                </AdminButton>
              </div>
            ),
          },
          ]}
        />
      </div>
      {!loading && !error && posts.length === 0 ? (
        <div className="admin-empty-state">
          <p>No posts match the current filters.</p>
          <Link className="admin-primary" to="/posts/new">
            Create a Post
          </Link>
        </div>
      ) : null}
      <div className="admin-pagination admin-pagination-bottom">
        <span className="admin-pagination-label">
          Page {page} / {Math.max(1, totalPages)} ({totalItems} items)
        </span>
        <div className="admin-toolbar-actions">
          <AdminButton className="admin-secondary" disabled={loading || page <= 1} onPress={() => updateParams({ page: page - 1 })}>
            Previous Page
          </AdminButton>
          <AdminButton
            className="admin-secondary"
            disabled={loading || page >= totalPages}
            onPress={() => updateParams({ page: Math.min(totalPages, page + 1) })}
          >
            Next Page
          </AdminButton>
        </div>
      </div>
    </section>
  );
}
