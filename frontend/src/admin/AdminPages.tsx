import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { pb, PageRecord } from "../lib/pb";

export default function AdminPages() {
  const [pages, setPages] = useState<PageRecord[]>([]);

  const load = () => {
    pb.collection("pages")
      .getFullList<PageRecord>({ sort: "menuOrder" })
      .then(setPages)
      .catch(() => setPages([]));
  };

  useEffect(() => {
    load();
  }, []);

  const remove = async (id: string) => {
    if (!window.confirm("削除しますか？")) return;
    await pb.collection("pages").delete(id);
    load();
  };

  return (
    <section>
      <header className="admin-header">
        <h1>Pages</h1>
        <Link className="admin-primary" to="/pages/new">
          New
        </Link>
      </header>
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
              <td>{page.title}</td>
              <td>{page.url}</td>
              <td>{page.menuVisible ? "visible" : "hidden"}</td>
              <td>{page.published ? "public" : "draft"}</td>
              <td className="admin-actions">
                <Link to={`/pages/${page.id}`}>Edit</Link>
                <button onClick={() => remove(page.id)}>Delete</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
