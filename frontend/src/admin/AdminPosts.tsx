import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { pb, PostRecord } from "../lib/pb";
import { formatDate } from "../utils/text";

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
      const trimmed = query.trim();
      const filter = trimmed ? buildFilter(trimmed) : undefined;
      try {
        const res = await pb.collection("posts").getList<PostRecord>(page, perPage, {
          filter,
          sort: "-published_at",
        });
        if (!alive) return;
        setPosts(res.items);
        setTotalPages(res.totalPages);
        setTotalItems(res.totalItems);
        return;
      } catch {
        // try fallback sort
      }
      try {
        const res = await pb.collection("posts").getList<PostRecord>(page, perPage, {
          filter,
          sort: "-created",
        });
        if (!alive) return;
        setPosts(res.items);
        setTotalPages(res.totalPages);
        setTotalItems(res.totalItems);
      } catch {
        if (!alive) return;
        setPosts([]);
        setTotalPages(1);
        setTotalItems(0);
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
    if (!window.confirm(`${selected.size}件を${value ? "公開" : "非公開"}にしますか？`)) return;
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
    if (!window.confirm("削除しますか？")) return;
    let mediaIds: string[] = [];
    try {
      const record = await pb.collection("posts").getOne(id);
      mediaIds = [
        ...extractMediaIds(record.body),
        ...extractMediaIds(record.content),
      ];
    } catch {
      mediaIds = [];
    }

    await pb.collection("posts").delete(id);

    if (mediaIds.length > 0) {
      try {
        const [postsAll, pagesAll] = await Promise.all([
          pb.collection("posts").getFullList({ fields: "body,content" }),
          pb.collection("pages").getFullList({ fields: "body,content" }),
        ]);
        const blobs = [
          ...postsAll.map((item: any) => `${item.body ?? ""} ${item.content ?? ""}`),
          ...pagesAll.map((item: any) => `${item.body ?? ""} ${item.content ?? ""}`),
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
        <input
          className="admin-input"
          type="search"
          placeholder="Search title, slug, tags..."
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setPage(1);
          }}
        />
        <select
          className="admin-input"
          value={perPage}
          onChange={(e) => {
            setPerPage(Number(e.target.value));
            setPage(1);
          }}
        >
          <option value={20}>20 / page</option>
          <option value={50}>50 / page</option>
          <option value={100}>100 / page</option>
        </select>
        <div className="admin-toolbar-actions">
          <button
            className="admin-primary"
            disabled={bulkLoading || selected.size === 0}
            onClick={() => bulkSetPublished(true)}
          >
            Publish
          </button>
          <button
            disabled={bulkLoading || selected.size === 0}
            onClick={() => bulkSetPublished(false)}
          >
            Unpublish
          </button>
        </div>
      </div>
      <table className="admin-table">
        <thead>
          <tr>
            <th>
              <input
                type="checkbox"
                checked={allFilteredSelected}
                onChange={toggleSelectAll}
                aria-label="Select all"
              />
            </th>
            <th>Title</th>
            <th>Date</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {posts.map((post) => (
            <tr key={post.id}>
              <td>
                <input
                  type="checkbox"
                  checked={selected.has(post.id)}
                  onChange={() => toggleSelect(post.id)}
                  aria-label={`Select ${post.title}`}
                />
              </td>
              <td>{post.title}</td>
              <td>{formatDate(post.published_at)}</td>
              <td>{post.published ? "public" : "draft"}</td>
              <td className="admin-actions">
                <Link to={`/posts/${post.id}`}>Edit</Link>
                <button onClick={() => remove(post.id)}>Delete</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="admin-pagination">
        <span>
          Page {page} / {Math.max(1, totalPages)} ({totalItems} items)
        </span>
        <div className="admin-toolbar-actions">
          <button disabled={loading || page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
            Prev
          </button>
          <button
            disabled={loading || page >= totalPages}
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          >
            Next
          </button>
        </div>
      </div>
    </section>
  );
}
