import { Link, Outlet, useNavigate } from "react-router-dom";
import { useState } from "react";
import { pb } from "../lib/pb";
import { AdminButton } from "./components/AriaControls";
import "../styles/admin.css";
import "highlight.js/styles/github-dark.css";

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
          <AdminButton className="admin-ghost" onPress={() => { closeSidebar(); logout(); }}>
            Logout
          </AdminButton>
        </nav>
      </aside>
      <main className="admin-main">
        <AdminButton
          ariaLabel="Toggle menu"
          className="admin-sidebar-toggle"
          onPress={() => setSidebarOpen((open) => !open)}
          type="button"
        >
          ☰
        </AdminButton>
        <Outlet />
      </main>
    </div>
  );
}
