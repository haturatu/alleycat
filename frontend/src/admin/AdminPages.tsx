import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { pb, PageRecord } from "../lib/pb";

export default function AdminPages() {
  const [pages, setPages] = useState<PageRecord[]>([]);
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [loading, setLoading] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  const buildFilter = (value: string) => {
    const safe = value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
    return `title ~ "${safe}" || url ~ "${safe}" || slug ~ "${safe}"`;
  };

  useEffect(() => {
    let alive = true;
    const loadPages = async () => {
      setLoading(true);
      const trimmed = query.trim();
      const filter = trimmed ? buildFilter(trimmed) : undefined;
      try {
        const res = await pb.collection("pages").getList<PageRecord>(page, perPage, {
          filter,
          sort: "menuOrder",
        });
        if (!alive) return;
        setPages(res.items);
        setTotalPages(res.totalPages);
        setTotalItems(res.totalItems);
      } catch {
        if (!alive) return;
        setPages([]);
        setTotalPages(1);
        setTotalItems(0);
      } finally {
        if (alive) setLoading(false);
      }
    };
    loadPages();
    return () => {
      alive = false;
    };
  }, [page, perPage, query, reloadToken]);

  const remove = async (id: string) => {
    if (!window.confirm("削除しますか？")) return;
    await pb.collection("pages").delete(id);
    setReloadToken((n) => n + 1);
  };

  return (
    <section>
      <header className="admin-header">
        <h1>Pages</h1>
        <Link className="admin-primary" to="/pages/new">
          New
        </Link>
      </header>
      <div className="admin-toolbar">
        <input
          className="admin-input"
          type="search"
          placeholder="Search title, url, slug..."
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
      </div>
      <div className="admin-pagination admin-pagination-top">
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
      <table className="admin-table">
        <thead>
          <tr>
            <th>Title</th>
            <th>URL</th>
            <th>Menu</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {pages.map((page) => (
            <tr key={page.id}>
              <td>
                <Link to={`/pages/${page.id}`}>{page.title}</Link>
              </td>
              <td>{page.url}</td>
              <td>{page.menuVisible ? "visible" : "hidden"}</td>
              <td>{page.published ? "public" : "draft"}</td>
              <td className="admin-actions">
                <button onClick={() => remove(page.id)}>Delete</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="admin-pagination admin-pagination-bottom">
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
