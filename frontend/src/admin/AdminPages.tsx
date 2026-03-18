import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { pb, PageRecord } from "../lib/pb";
import { AdminButton, AdminConfirmDialog, AdminSelectField, AdminTable, AdminTextField } from "./components/AriaControls";
import FormStatusMessage from "./components/FormStatusMessage";
import useAdminPageTitle from "./hooks/useAdminPageTitle";

export default function AdminPages() {
  const [pages, setPages] = useState<PageRecord[]>([]);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [loading, setLoading] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);
  const [deleteTargetId, setDeleteTargetId] = useState<string | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [error, setError] = useState("");
  const [searchParams, setSearchParams] = useSearchParams();

  useAdminPageTitle("Pages");

  const query = searchParams.get("q") ?? "";
  const page = Math.max(1, Number(searchParams.get("page") || "1") || 1);
  const parsedPerPage = Number(searchParams.get("perPage") || "20");
  const perPage = [20, 50, 100].includes(parsedPerPage) ? parsedPerPage : 20;

  const buildFilter = (value: string) => {
    const safe = value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
    return `title ~ "${safe}" || url ~ "${safe}" || slug ~ "${safe}"`;
  };

  const updateParams = (updates: Record<string, string | number | null>) => {
    const next = new URLSearchParams(searchParams);
    Object.entries(updates).forEach(([key, value]) => {
      if (value === null || value === "" || value === 1 || value === 20) {
        next.delete(key);
      } else {
        next.set(key, String(value));
      }
    });
    setSearchParams(next, { replace: true });
  };

  useEffect(() => {
    let alive = true;
    const loadPages = async () => {
      setLoading(true);
      setError("");
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
        setError("Pages could not be loaded. Refresh or adjust the current filters.");
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
    setDeleteLoading(true);
    setError("");
    try {
      await pb.collection("pages").delete(id);
      setReloadToken((n) => n + 1);
    } catch {
      setError("This page could not be deleted. Try again.");
    } finally {
      setDeleteLoading(false);
    }
  };

  return (
    <section>
      <header className="admin-header">
        <div>
          <p className="admin-eyebrow">Pages</p>
          <h1>Pages</h1>
        </div>
        <Link className="admin-primary" to="/pages/new">
          New Page
        </Link>
      </header>
      <FormStatusMessage error={error} />
      <AdminConfirmDialog
        open={deleteTargetId !== null}
        title="Delete page"
        message="This page will be removed immediately. Delete it?"
        confirmLabel={deleteLoading ? "Deleting…" : "Delete Page"}
        confirmDisabled={deleteLoading}
        onCancel={() => setDeleteTargetId(null)}
        onConfirm={() => {
          const next = deleteTargetId;
          setDeleteTargetId(null);
          if (next) void remove(next);
        }}
      />
      <div className="admin-stack">
        <section className="admin-toolbar admin-toolbar-section admin-filter-bar">
          <div className="admin-toolbar-heading">
            <p className="admin-section-label">Search and filter</p>
            <p className="admin-toolbar-note">
              {query.trim()
                ? `Filtering pages by "${query.trim()}".`
                : "Search titles, URLs, and slugs to narrow the current structure."}
            </p>
          </div>
          <AdminTextField
            ariaLabel="Search pages"
            className="admin-input"
            label="Search"
            value={query}
            type="search"
            placeholder="Search title, URL, or slug…"
            onChange={(value) => {
              updateParams({ q: value || null, page: null });
            }}
          />
          <AdminSelectField
            ariaLabel="Rows per page"
            className="admin-field"
            label="Rows per page"
            value={perPage}
            onChange={(value) => {
              updateParams({ perPage: Number(value), page: null });
            }}
            options={[
              { value: 20, label: "20 / page" },
              { value: 50, label: "50 / page" },
              { value: 100, label: "100 / page" },
            ]}
          />
        </section>
      </div>
      <div className="admin-pagination admin-pagination-top">
        <span className="admin-pagination-label">
          Page {page} / {Math.max(1, totalPages)} ({totalItems} items)
        </span>
        <div className="admin-toolbar-actions">
          <AdminButton className="admin-secondary" disabled={loading || page <= 1} onPress={() => updateParams({ page: page - 1 })}>
            Previous Page
          </AdminButton>
          <AdminButton
            className="admin-secondary"
            disabled={loading || page >= totalPages}
            onPress={() => updateParams({ page: Math.min(totalPages, page + 1) })}
          >
            Next Page
          </AdminButton>
        </div>
      </div>
      {loading ? <p className="admin-note">Loading pages…</p> : null}
      <div className="admin-list-shell">
        <div className="admin-table-utility admin-list-strip is-passive">
          <div className="admin-table-utility-copy">
            <p className="admin-section-label">Structure</p>
            <p className="admin-table-selection">{pages.filter((item) => item.menuVisible).length} menu entries visible</p>
            <p className="admin-note">
              {query.trim() ? `Filtered by "${query.trim()}".` : `${totalItems} total pages in the current structure.`}
            </p>
          </div>
        </div>
        <AdminTable
          ariaLabel="Pages"
          items={pages}
          columns={[
          {
            id: "title",
            name: "Title",
            mobileLabel: "Title",
            isRowHeader: true,
            render: (item) => <Link to={`/pages/${item.id}`}>{item.title}</Link>,
          },
          {
            id: "url",
            name: "URL",
            mobileLabel: "URL",
            className: "admin-table-url-column",
            width: "160px",
            render: (item) => item.url,
          },
          {
            id: "menu",
            name: "Menu",
            mobileLabel: "Menu",
            className: "admin-table-status-column",
            width: "120px",
            render: (item) => (
              <span className={item.menuVisible ? "admin-status-badge is-published" : "admin-status-badge is-draft"}>
                {item.menuVisible ? "Visible" : "Hidden"}
              </span>
            ),
          },
          {
            id: "status",
            name: "Status",
            mobileLabel: "Status",
            className: "admin-table-status-column",
            width: "126px",
            render: (item) => (
              <span className={item.published ? "admin-status-badge is-published" : "admin-status-badge is-draft"}>
                {item.published ? "Published" : "Draft"}
              </span>
            ),
          },
          {
            id: "actions",
            name: "Action",
            mobileLabel: "Action",
            width: "72px",
            render: (item) => (
              <div className="admin-actions">
                <AdminButton ariaLabel={`Delete ${item.title}`} className="admin-danger-button" onPress={() => setDeleteTargetId(item.id)}>
                  🗑
                </AdminButton>
              </div>
            ),
          },
          ]}
        />
      </div>
      {!loading && !error && pages.length === 0 ? (
        <div className="admin-empty-state">
          <p>No pages match the current filters.</p>
          <Link className="admin-primary" to="/pages/new">
            Create a Page
          </Link>
        </div>
      ) : null}
      <div className="admin-pagination admin-pagination-bottom">
        <span className="admin-pagination-label">
          Page {page} / {Math.max(1, totalPages)} ({totalItems} items)
        </span>
        <div className="admin-toolbar-actions">
          <AdminButton className="admin-secondary" disabled={loading || page <= 1} onPress={() => updateParams({ page: page - 1 })}>
            Previous Page
          </AdminButton>
          <AdminButton
            className="admin-secondary"
            disabled={loading || page >= totalPages}
            onPress={() => updateParams({ page: Math.min(totalPages, page + 1) })}
          >
            Next Page
          </AdminButton>
        </div>
      </div>
    </section>
  );
}
