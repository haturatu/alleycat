import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { pb, PageRecord } from "../lib/pb";
import { AdminButton, AdminSelectField, AdminTextField } from "./components/AriaControls";

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
    if (!window.confirm("Delete this page?")) return;
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
        <AdminTextField
          ariaLabel="Search pages"
          className="admin-input"
          label=""
          value={query}
          type="search"
          placeholder="Search title, url, slug..."
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
            <th>Title</th>
            <th>URL</th>
            <th>Menu</th>
            <th>Status</th>
            <th>Actions</th>
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
                <AdminButton onPress={() => remove(page.id)}>Delete</AdminButton>
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
