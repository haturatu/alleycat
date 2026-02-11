import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { pb } from "../lib/pb";
import { buildExcerpt, parseTags, slugify } from "../utils/text";
import RichEditor from "./RichEditor";

export default function AdminPostEditor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [body, setBody] = useState("");
  const [excerpt, setExcerpt] = useState("");
  const [tags, setTags] = useState("");
  const [category, setCategory] = useState("");
  const [author, setAuthor] = useState("");
  const [publishedAt, setPublishedAt] = useState("");
  const [published, setPublished] = useState(true);
  const [featuredImage, setFeaturedImage] = useState<File | null>(null);
  const [attachments, setAttachments] = useState<File[]>([]);
  const [authors, setAuthors] = useState<Array<{ id: string; name?: string; email?: string }>>(
    []
  );
  const [categories, setCategories] = useState<string[]>([]);
  const [tagOptions, setTagOptions] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [excerptLength, setExcerptLength] = useState(0);

  useEffect(() => {
    if (!id || id === "new") return;
    pb.collection("posts")
      .getOne(id)
      .then((record) => {
        setError("");
        setTitle(record.title || "");
        setSlug(record.slug || "");
        setBody(record.body || "");
        setExcerpt(record.excerpt || "");
        setTags(record.tags || "");
        setCategory(record.category || "");
        setAuthor(record.author || "");
        setPublishedAt(record.published_at ? record.published_at.slice(0, 16) : "");
        setPublished(Boolean(record.published));
      })
      .catch((err) => {
        setError("記事の取得に失敗しました。権限またはIDを確認してください。");
        console.error(err);
      });
  }, [id, navigate]);

  useEffect(() => {
    pb.collection("cms_users")
      .getFullList({ sort: "name" })
      .then((items) => setAuthors(items as Array<{ id: string; name?: string; email?: string }>))
      .catch(() => setAuthors([]));
  }, []);

  useEffect(() => {
    pb.collection("posts")
      .getFullList({ fields: "category,tags" })
      .then((items: any[]) => {
        const categorySet = new Set<string>();
        const tagSet = new Set<string>();
        items.forEach((item) => {
          if (item.category) categorySet.add(String(item.category));
          parseTags(item.tags).forEach((tag) => tagSet.add(tag));
        });
        setCategories(Array.from(categorySet));
        setTagOptions(Array.from(tagSet));
      })
      .catch(() => {
        setCategories([]);
        setTagOptions([]);
      });
  }, []);

  useEffect(() => {
    pb.collection("settings")
      .getList(1, 1)
      .then((res) => {
        const length = Number(res.items?.[0]?.excerpt_length ?? 0);
        setExcerptLength(Number.isFinite(length) ? length : 0);
      })
      .catch(() => setExcerptLength(0));
  }, []);

  const save = async () => {
    const autoExcerpt = excerptLength > 0;
    const finalExcerpt = autoExcerpt ? buildExcerpt(body, excerptLength) : excerpt;
    const form = new FormData();
    form.set("title", title);
    form.set("slug", slug);
    form.set("body", body);
    form.set("excerpt", finalExcerpt || buildExcerpt(body));
    form.set("tags", tags);
    form.set("category", category);
    form.set("author", author);
    form.set("published_at", publishedAt ? new Date(publishedAt).toISOString() : new Date().toISOString());
    form.set("published", String(published));
    if (featuredImage) {
      form.set("featured_image", featuredImage);
    }
    if (attachments.length > 0) {
      attachments.forEach((file) => form.append("attachments", file));
    }

    if (!id || id === "new") {
      await pb.collection("posts").create(form);
    } else {
      await pb.collection("posts").update(id, form);
    }
    navigate("/posts");
  };

  return (
    <section>
      <header className="admin-header">
        <h1>{id === "new" ? "New Post" : "Edit Post"}</h1>
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
          Published at
          <input
            type="datetime-local"
            value={publishedAt}
            onChange={(e) => setPublishedAt(e.target.value)}
          />
        </label>
        <label>
          Category
          <input
            list="category-list"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          />
        </label>
        <datalist id="category-list">
          {categories.map((item) => (
            <option key={item} value={item} />
          ))}
        </datalist>
        <label>
          Author
          <select value={author} onChange={(e) => setAuthor(e.target.value)}>
            <option value="">(none)</option>
            {authors.map((user) => (
              <option key={user.id} value={user.id}>
                {user.name || user.email || user.id}
              </option>
            ))}
          </select>
        </label>
        <label>
          Tags (comma separated)
          <input value={tags} onChange={(e) => setTags(e.target.value)} />
        </label>
        {tagOptions.length > 0 && (
          <div className="admin-tag-suggestions">
            {tagOptions
              .filter((tag) => !parseTags(tags).includes(tag))
              .slice(0, 20)
              .map((tag) => (
                <button
                  type="button"
                  key={tag}
                  onClick={() => {
                    const next = [...parseTags(tags), tag].join(", ");
                    setTags(next);
                  }}
                >
                  {tag}
                </button>
              ))}
          </div>
        )}
        <label>
          Featured image
          <input
            type="file"
            accept="image/*"
            onChange={(e) => setFeaturedImage(e.target.files ? e.target.files[0] : null)}
          />
        </label>
        <label>
          Attachments
          <input
            type="file"
            multiple
            onChange={(e) =>
              setAttachments(e.target.files ? Array.from(e.target.files) : [])
            }
          />
        </label>
        <label>
          Excerpt
          <textarea
            value={excerptLength > 0 ? buildExcerpt(body, excerptLength) : excerpt}
            onChange={(e) => setExcerpt(e.target.value)}
            rows={3}
            disabled={excerptLength > 0}
          />
        </label>
        {excerptLength > 0 && (
          <p className="admin-note">
            Excerpt is auto-generated from content ({excerptLength} chars).
          </p>
        )}
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
