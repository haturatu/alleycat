import { useEffect, useMemo, useState } from "react";
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

  const load = () => {
    const loadPosts = async () => {
      try {
        const items = await pb.collection("posts").getFullList<PostRecord>({ sort: "-published_at" });
        setPosts(items);
      } catch {
        try {
          const items = await pb.collection("posts").getFullList<PostRecord>({ sort: "-created" });
          setPosts(items);
        } catch {
          try {
            const res = await pb.collection("posts").getList<PostRecord>(1, 100);
            setPosts(res.items);
          } catch {
            setPosts([]);
          }
        }
      }
    };
    loadPosts();
  };

  useEffect(() => {
    load();
  }, []);

  useEffect(() => {
    setSelected((prev) => {
      if (prev.size === 0) return prev;
      const ids = new Set(posts.map((post) => post.id));
      const next = new Set(Array.from(prev).filter((id) => ids.has(id)));
      return next;
    });
  }, [posts]);

  const filteredPosts = useMemo(() => {
    const trimmed = query.trim().toLowerCase();
    if (!trimmed) return posts;
    return posts.filter((post) => {
      const haystack = [
        post.title,
        post.slug,
        post.tags,
        post.category,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return haystack.includes(trimmed);
    });
  }, [posts, query]);

  const allFilteredSelected =
    filteredPosts.length > 0 && filteredPosts.every((post) => selected.has(post.id));

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
        filteredPosts.forEach((post) => next.delete(post.id));
        return next;
      }
      filteredPosts.forEach((post) => next.add(post.id));
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
    load();
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

    load();
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
          onChange={(e) => setQuery(e.target.value)}
        />
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
          {filteredPosts.map((post) => (
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
    </section>
  );
}
