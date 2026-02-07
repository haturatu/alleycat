import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import PostList from "../components/PostList";
import Pagination from "../components/Pagination";
import { pb, PostRecord } from "../lib/pb";
import { parseTags } from "../utils/text";

const PAGE_SIZE = 10;

export default function ArchivePage() {
  const { slug, page } = useParams();
  const [posts, setPosts] = useState<PostRecord[]>([]);
  const [totalPages, setTotalPages] = useState(1);
  const [title, setTitle] = useState("Archive");
  const [tags, setTags] = useState<string[]>([]);

  const { currentPage, tagSlug } = useMemo(() => {
    if (slug && /^\d+$/.test(slug)) {
      return { currentPage: Number(slug), tagSlug: null as string | null };
    }
    const resolvedPage = page && /^\d+$/.test(page) ? Number(page) : 1;
    return { currentPage: resolvedPage, tagSlug: slug ?? null };
  }, [slug, page]);

  useEffect(() => {
    const load = async () => {
      const safeTag = tagSlug ? tagSlug.replace(/"/g, "") : null;
      const filter = safeTag
        ? `published = true && tags ~ "${safeTag}"`
        : "published = true";
      try {
        const res = await pb.collection("posts").getList<PostRecord>(currentPage, PAGE_SIZE, {
          filter,
          sort: "-published_at",
        });
        setPosts(res.items);
        setTotalPages(res.totalPages);
      } catch {
        try {
          const res = await pb.collection("posts").getList<PostRecord>(currentPage, PAGE_SIZE, {
            filter,
            sort: "-date",
          });
          setPosts(res.items);
          setTotalPages(res.totalPages);
        } catch {
          setPosts([]);
          setTotalPages(1);
        }
      }
    };
    load();
  }, [currentPage, tagSlug]);

  useEffect(() => {
    setTitle(tagSlug ? `tag: ${tagSlug}` : "Archive");
  }, [tagSlug]);

  useEffect(() => {
    pb.collection("posts")
      .getFullList<PostRecord>({ filter: "published = true" })
      .then((items) => {
        const allTags = items.flatMap((post) => parseTags(post.tags));
        const unique = Array.from(new Set(allTags));
        setTags(unique);
      })
      .catch(() => setTags([]));
  }, []);

  const baseUrl = tagSlug ? `/archive/${tagSlug}` : "/archive";

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">{title}</h1>
        <p>
          RSS: <a href="/feed.xml">Atom</a>, <a href="/feed.json">JSON</a>
        </p>
      </header>
      <PostList posts={posts} />
      <Pagination baseUrl={baseUrl} page={currentPage} totalPages={totalPages} />
      {!tagSlug && tags.length > 0 && (
        <nav className="page-navigation">
          <h2>tags:</h2>
          <ul className="page-navigation-tags">
            {tags.map((tag) => (
              <li key={tag}>
                <a href={`/archive/${tag}/`} className="badge">
                  {tag}
                </a>
              </li>
            ))}
          </ul>
        </nav>
      )}
    </>
  );
}
