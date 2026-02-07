import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { pb, PageRecord } from "../lib/pb";

export default function PageRoute() {
  const location = useLocation();
  const [page, setPage] = useState<PageRecord | null>(null);

  useEffect(() => {
    const path = location.pathname.endsWith("/") ? location.pathname : `${location.pathname}/`;
    const safePath = path.replace(/"/g, "");
    pb.collection("pages")
      .getFirstListItem<PageRecord>(
        `url = "${safePath}" && published = true`
      )
      .then(setPage)
      .catch(() => setPage(null));
  }, [location.pathname]);

  if (!page) {
    return (
      <article className="post">
        <header className="post-header">
          <h1 className="post-title">Not Found</h1>
        </header>
        <div className="post-body body">ページが見つかりませんでした。</div>
      </article>
    );
  }

  return (
    <article className="post">
      <header className="post-header">
        <h1 className="post-title">{page.title}</h1>
      </header>
      <div className="post-body body" dangerouslySetInnerHTML={{ __html: page.body }} />
    </article>
  );
}
