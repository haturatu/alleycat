import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useState } from "react";
import { pb } from "../lib/pb";
import { AdminButton, AdminDialog } from "./components/AriaControls";
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

  const sidebarContent = (
    <>
        <h2>Admin</h2>
        <nav>
        <NavLink className={({ isActive }) => (isActive ? "is-current" : undefined)} to="/posts" onClick={closeSidebar}>
          Posts
        </NavLink>
        <NavLink className={({ isActive }) => (isActive ? "is-current" : undefined)} to="/pages" onClick={closeSidebar}>
          Pages
        </NavLink>
        <NavLink className={({ isActive }) => (isActive ? "is-current" : undefined)} to="/settings" onClick={closeSidebar}>
          Settings
        </NavLink>
        <AdminButton className="admin-ghost" onPress={() => { closeSidebar(); logout(); }}>
          Logout
        </AdminButton>
      </nav>
    </>
  );

  return (
    <div className="admin-shell" data-sidebar-open={sidebarOpen}>
      <aside className="admin-sidebar">{sidebarContent}</aside>
      <div className="admin-sidebar-mobile">
        <AdminDialog open={sidebarOpen} onClose={closeSidebar} title="Admin navigation">
          <div className="admin-modal-head">
            <h2>Menu</h2>
            <AdminButton className="admin-modal-close" onPress={closeSidebar}>
              Close
            </AdminButton>
          </div>
          <div className="admin-modal-body admin-mobile-nav">{sidebarContent}</div>
        </AdminDialog>
      </div>
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
