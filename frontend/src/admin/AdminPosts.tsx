import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { pb, PostRecord } from "../lib/pb";
import { formatDate } from "../utils/text";
import {
  AdminButton,
  AdminCheckboxField,
  AdminSelectField,
  AdminTextField,
} from "./components/AriaControls";

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
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkLoading, setBulkLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [loading, setLoading] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  const buildFilter = (value: string) => {
    const safe = value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
    return `title ~ "${safe}" || slug ~ "${safe}" || tags ~ "${safe}" || category ~ "${safe}"`;
  };

  useEffect(() => {
    let alive = true;
    const loadPosts = async () => {
      setLoading(true);
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
        }
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
    if (!window.confirm(`Set ${selected.size} selected posts to ${value ? "published" : "unpublished"}?`)) return;
    setBulkLoading(true);
    const now = new Date().toISOString();
    const byId = new Map(posts.map((post) => [post.id, post]));
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
    setBulkLoading(false);
    setReloadToken((n) => n + 1);
  };

  const remove = async (id: string) => {
    if (!window.confirm("Delete this post?")) return;
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
  };

  return (
    <section>
      <header className="admin-header">
        <h1>Posts</h1>
        <Link className="admin-primary" to="/posts/new">
          New
        </Link>
      </header>
      <div className="admin-toolbar">
        <AdminTextField
          ariaLabel="Search posts"
          className="admin-input"
          label=""
          value={query}
          type="search"
          placeholder="Search title, slug, tags..."
          onChange={(value) => {
            setQuery(value);
            setPage(1);
          }}
        />
        <AdminSelectField
          ariaLabel="Rows per page"
          className="admin-field"
          label=""
          value={perPage}
          onChange={(value) => {
            setPerPage(Number(value));
            setPage(1);
          }}
          options={[
            { value: 20, label: "20 / page" },
            { value: 50, label: "50 / page" },
            { value: 100, label: "100 / page" },
          ]}
        />
        <div className="admin-toolbar-actions">
          <AdminButton
            className="admin-primary"
            disabled={bulkLoading || selected.size === 0}
            onPress={() => bulkSetPublished(true)}
          >
            Publish
          </AdminButton>
          <AdminButton
            disabled={bulkLoading || selected.size === 0}
            onPress={() => bulkSetPublished(false)}
          >
            Unpublish
          </AdminButton>
        </div>
      </div>
      <div className="admin-pagination admin-pagination-top">
        <span>
          Page {page} / {Math.max(1, totalPages)} ({totalItems} items)
        </span>
        <div className="admin-toolbar-actions">
          <AdminButton disabled={loading || page <= 1} onPress={() => setPage((p) => Math.max(1, p - 1))}>
            Prev
          </AdminButton>
          <AdminButton
            disabled={loading || page >= totalPages}
            onPress={() => setPage((p) => Math.min(totalPages, p + 1))}
          >
            Next
          </AdminButton>
        </div>
      </div>
      <div className="admin-table-wrap">
        <table className="admin-table">
        <thead>
          <tr>
            <th>
              <AdminCheckboxField
                ariaLabel="Select all"
                label=""
                checked={allFilteredSelected}
                onChange={toggleSelectAll}
                className="admin-check"
              />
            </th>
            <th>Title</th>
            <th>Date</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {posts.map((post) => (
            <tr key={post.id}>
              <td>
                <AdminCheckboxField
                  ariaLabel={`Select ${post.title}`}
                  label=""
                  checked={selected.has(post.id)}
                  onChange={() => toggleSelect(post.id)}
                  className="admin-check"
                />
              </td>
              <td>
                <Link to={`/posts/${post.id}`}>{post.title}</Link>
              </td>
              <td>{formatDate(post.published_at)}</td>
              <td>{post.published ? "public" : "draft"}</td>
              <td className="admin-actions">
                <AdminButton onPress={() => remove(post.id)}>Delete</AdminButton>
              </td>
            </tr>
          ))}
        </tbody>
        </table>
      </div>
      <div className="admin-pagination admin-pagination-bottom">
        <span>
          Page {page} / {Math.max(1, totalPages)} ({totalItems} items)
        </span>
        <div className="admin-toolbar-actions">
          <AdminButton disabled={loading || page <= 1} onPress={() => setPage((p) => Math.max(1, p - 1))}>
            Prev
          </AdminButton>
          <AdminButton
            disabled={loading || page >= totalPages}
            onPress={() => setPage((p) => Math.min(totalPages, p + 1))}
          >
            Next
          </AdminButton>
        </div>
      </div>
    </section>
  );
}
