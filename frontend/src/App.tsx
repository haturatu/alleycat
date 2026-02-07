import { BrowserRouter, Route, Routes } from "react-router-dom";
import Layout from "./components/Layout";
import HomePage from "./pages/HomePage";
import ArchivePage from "./pages/ArchivePage";
import PostPage from "./pages/PostPage";
import PageRoute from "./pages/PageRoute";
import AdminLogin from "./admin/AdminLogin";
import AdminLayout from "./admin/AdminLayout";
import AdminPosts from "./admin/AdminPosts";
import AdminPages from "./admin/AdminPages";
import AdminPostEditor from "./admin/AdminPostEditor";
import AdminPageEditor from "./admin/AdminPageEditor";
import RequireAdmin from "./admin/RequireAdmin";
import AdminSettings from "./admin/AdminSettings";

export default function App() {
  const isAdminApp = import.meta.env.VITE_ADMIN === "true";
  const base = import.meta.env.VITE_BASE || "/";
  const basename = base === "/" ? undefined : base.replace(/\/$/, "");

  return (
    <BrowserRouter basename={basename}>
      {isAdminApp ? (
        <Routes>
          <Route path="/" element={<AdminLogin />} />
          <Route
            element={
              <RequireAdmin>
                <AdminLayout />
              </RequireAdmin>
            }
          >
            <Route path="/posts" element={<AdminPosts />} />
            <Route path="/posts/:id" element={<AdminPostEditor />} />
            <Route path="/pages" element={<AdminPages />} />
            <Route path="/pages/:id" element={<AdminPageEditor />} />
            <Route path="/settings" element={<AdminSettings />} />
          </Route>
        </Routes>
      ) : (
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={<HomePage />} />
            <Route path="archive" element={<ArchivePage />} />
            <Route path="archive/:slug" element={<ArchivePage />} />
            <Route path="archive/:slug/:page" element={<ArchivePage />} />
            <Route path="posts/:slug/*" element={<PostPage />} />
            <Route path="*" element={<PageRoute />} />
          </Route>
        </Routes>
      )}
    </BrowserRouter>
  );
}
