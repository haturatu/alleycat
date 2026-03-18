import { Route, RouterProvider, createBrowserRouter, createRoutesFromElements } from "react-router-dom";
import AdminLogin from "@cms/features/auth/AdminLogin";
import RequireAdmin from "@cms/features/auth/RequireAdmin";
import AdminLayout from "@cms/features/layout/AdminLayout";
import AdminPageEditor from "@cms/features/pages/AdminPageEditor";
import AdminPages from "@cms/features/pages/AdminPages";
import AdminPostEditor from "@cms/features/posts/AdminPostEditor";
import AdminPosts from "@cms/features/posts/AdminPosts";
import AdminSettings from "@cms/features/settings/AdminSettings";

export default function CmsApp() {
  const base = import.meta.env.VITE_BASE || "/";
  const basename = base === "/" ? undefined : base.replace(/\/$/, "");
  const router = createBrowserRouter(
    createRoutesFromElements(
      <>
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
      </>
    ),
    { basename }
  );

  return <RouterProvider router={router} />;
}
