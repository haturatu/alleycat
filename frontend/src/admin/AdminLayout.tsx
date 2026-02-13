import { Link, Outlet, useNavigate } from "react-router-dom";
import { useState } from "react";
import { pb } from "../lib/pb";
import "../styles/admin.css";

export default function AdminLayout() {
  const navigate = useNavigate();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const logout = () => {
    pb.authStore.clear();
    navigate("/");
  };

  const closeSidebar = () => setSidebarOpen(false);

  return (
    <div className="admin-shell" data-sidebar-open={sidebarOpen}>
      <aside className="admin-sidebar">
        <h2>Admin</h2>
        <nav>
          <Link to="/posts" onClick={closeSidebar}>
            Posts
          </Link>
          <Link to="/pages" onClick={closeSidebar}>
            Pages
          </Link>
          <Link to="/settings" onClick={closeSidebar}>
            Settings
          </Link>
          <button className="admin-ghost" onClick={() => { closeSidebar(); logout(); }}>
            Logout
          </button>
        </nav>
      </aside>
      <main className="admin-main">
        <button
          className="admin-sidebar-toggle"
          onClick={() => setSidebarOpen((open) => !open)}
          type="button"
          aria-label="Toggle menu"
        >
          ☰
        </button>
        <Outlet />
      </main>
    </div>
  );
}
