import { FormEvent, useEffect, useMemo, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import PostList from "../components/PostList";
import Pagination from "../components/Pagination";
import { pb, PostRecord } from "../lib/pb";
import { parseTags } from "../utils/text";

const PAGE_SIZE = 10;

export default function ArchivePage() {
  const { slug, page } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const [posts, setPosts] = useState<PostRecord[]>([]);
  const [totalPages, setTotalPages] = useState(1);
  const [title, setTitle] = useState("Archive");
  const [tags, setTags] = useState<string[]>([]);
  const [searchInput, setSearchInput] = useState(searchParams.get("q")?.trim() ?? "");

  const { currentPage, tagSlug } = useMemo(() => {
    if (slug && /^\d+$/.test(slug)) {
      return { currentPage: Number(slug), tagSlug: null as string | null };
    }
    const resolvedPage = page && /^\d+$/.test(page) ? Number(page) : 1;
    return { currentPage: resolvedPage, tagSlug: slug ?? null };
  }, [slug, page]);

  const searchQuery = useMemo(() => searchParams.get("q")?.trim() ?? "", [searchParams]);

  useEffect(() => {
    setSearchInput(searchQuery);
  }, [searchQuery]);

  useEffect(() => {
    const load = async () => {
      const safeTag = tagSlug ? tagSlug.replace(/"/g, "") : null;
      const safeQuery = searchQuery.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
      const conditions: string[] = ["published = true"];
      if (safeTag) conditions.push(`tags ~ "${safeTag}"`);
      if (safeQuery) {
        conditions.push(
          `(title ~ "${safeQuery}" || slug ~ "${safeQuery}" || tags ~ "${safeQuery}" || excerpt ~ "${safeQuery}" || body ~ "${safeQuery}")`
        );
      }
      const filter = conditions.join(" && ");
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
  }, [currentPage, tagSlug, searchQuery]);

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

  const onSearchSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const next = new URLSearchParams(searchParams);
    const value = searchInput.trim();
    if (value) {
      next.set("q", value);
    } else {
      next.delete("q");
    }
    setSearchParams(next);
  };

  const clearSearch = () => {
    const next = new URLSearchParams(searchParams);
    next.delete("q");
    setSearchInput("");
    setSearchParams(next);
  };

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">{title}</h1>
        <p>
          RSS: <a href="/feed.xml">Atom</a>, <a href="/feed.json">JSON</a>
        </p>
        <div className="search" id="search">
          <form className="search-form" onSubmit={onSearchSubmit}>
            <input
              id="archive-search"
              className="search-input"
              type="search"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Search posts..."
              aria-label="Search posts"
            />
            <button className="search-submit" type="submit">
              Search
            </button>
            {searchQuery ? (
              <button className="search-clear" type="button" onClick={clearSearch}>
                Clear
              </button>
            ) : null}
          </form>
        </div>
      </header>
      <PostList posts={posts} />
      <Pagination baseUrl={baseUrl} page={currentPage} totalPages={totalPages} query={searchQuery} />
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
