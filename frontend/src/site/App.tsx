import { Route, RouterProvider, createBrowserRouter, createRoutesFromElements } from "react-router-dom";
import ArchivePage from "@site/features/archive/ArchivePage";
import HomePage from "@site/features/home/HomePage";
import PageRoute from "@site/features/page/PageRoute";
import PostPage from "@site/features/post/PostPage";
import Layout from "@site/ui/Layout";

export default function SiteApp() {
  const base = import.meta.env.VITE_BASE || "/";
  const basename = base === "/" ? undefined : base.replace(/\/$/, "");
  const router = createBrowserRouter(
    createRoutesFromElements(
      <Route path="/" element={<Layout />}>
        <Route index element={<HomePage />} />
        <Route path="archive" element={<ArchivePage />} />
        <Route path="archive/:slug" element={<ArchivePage />} />
        <Route path="archive/:slug/:page" element={<ArchivePage />} />
        <Route path="posts/:slug/*" element={<PostPage />} />
        <Route path="*" element={<PageRoute />} />
      </Route>
    ),
    { basename }
  );

  return <RouterProvider router={router} />;
}
