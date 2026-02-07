import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { pb } from "../lib/pb";
import { slugify } from "../utils/text";
import RichEditor from "./RichEditor";

export default function AdminPageEditor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [url, setUrl] = useState("");
  const [menuVisible, setMenuVisible] = useState(false);
  const [menuOrder, setMenuOrder] = useState(0);
  const [menuTitle, setMenuTitle] = useState("");
  const [body, setBody] = useState("");
  const [publishedAt, setPublishedAt] = useState("");
  const [published, setPublished] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!id || id === "new") return;
    pb.collection("pages")
      .getOne(id)
      .then((record) => {
        setError("");
        setTitle(record.title || "");
        setSlug(record.slug || "");
        setUrl(record.url || "");
        setMenuVisible(Boolean(record.menuVisible));
        setMenuOrder(record.menuOrder || 0);
        setMenuTitle(record.menuTitle || "");
        setBody(record.body || "");
        setPublishedAt(record.published_at ? record.published_at.slice(0, 16) : "");
        setPublished(Boolean(record.published));
      })
      .catch((err) => {
        setError("ページの取得に失敗しました。権限またはIDを確認してください。");
        console.error(err);
      });
  }, [id, navigate]);

  const save = async () => {
    const resolvedUrl = url || `/${slug}/`;
    const payload = {
      title,
      slug,
      url: resolvedUrl,
      menuVisible,
      menuOrder,
      menuTitle,
      body,
      published_at: publishedAt ? new Date(publishedAt).toISOString() : new Date().toISOString(),
      published,
    };

    if (!id || id === "new") {
      await pb.collection("pages").create(payload);
    } else {
      await pb.collection("pages").update(id, payload);
    }
    navigate("/pages");
  };

  return (
    <section>
      <header className="admin-header">
        <h1>{id === "new" ? "New Page" : "Edit Page"}</h1>
        <button className="admin-primary" onClick={save}>
          Save
        </button>
      </header>
      {error && <p className="admin-error">{error}</p>}
      <div className="admin-form">
        <label>
          Title
          <input value={title} onChange={(e) => setTitle(e.target.value)} />
        </label>
        <label>
          Slug
          <div className="admin-inline">
            <input value={slug} onChange={(e) => setSlug(e.target.value)} />
            <button type="button" onClick={() => setSlug(slugify(title))}>
              Auto
            </button>
          </div>
        </label>
        <label>
          URL
          <input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="/ab/" />
        </label>
        <label className="admin-check admin-check-right">
          <span>Show in menu</span>
          <input type="checkbox" checked={menuVisible} onChange={(e) => setMenuVisible(e.target.checked)} />
        </label>
        <label>
          Published at
          <input
            type="datetime-local"
            value={publishedAt}
            onChange={(e) => setPublishedAt(e.target.value)}
          />
        </label>
        <label>
          Menu order
          <input type="number" value={menuOrder} onChange={(e) => setMenuOrder(Number(e.target.value))} />
        </label>
        <label>
          Menu title
          <input value={menuTitle} onChange={(e) => setMenuTitle(e.target.value)} />
        </label>
        <label className="admin-check admin-check-right">
          <span>Published</span>
          <input type="checkbox" checked={published} onChange={(e) => setPublished(e.target.checked)} />
        </label>
        <div className="admin-field">
          <span>Content</span>
          <RichEditor value={body} onChange={setBody} />
        </div>
      </div>
    </section>
  );
}
