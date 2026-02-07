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
      <table className="admin-table">
        <thead>
          <tr>
            <th>Title</th>
            <th>Date</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {posts.map((post) => (
            <tr key={post.id}>
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
