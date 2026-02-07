import { useEffect, useState } from "react";
import { pb, PostRecord } from "../lib/pb";
import { siteConfig } from "../config";
import PostList from "../components/PostList";

export default function HomePage() {
  const [posts, setPosts] = useState<PostRecord[]>([]);

  useEffect(() => {
    const load = async () => {
      try {
        const res = await pb.collection("posts").getList<PostRecord>(1, 3, {
          filter: "published = true",
          sort: "-published_at",
        });
        setPosts(res.items);
      } catch {
        try {
          const res = await pb.collection("posts").getList<PostRecord>(1, 3, {
            filter: "published = true",
            sort: "-date",
          });
          setPosts(res.items);
        } catch {
          setPosts([]);
        }
      }
    };
    load();
  }, []);

  return (
    <>
      <header className="page-header">
        <img
          src={siteConfig.homeTopImage}
          alt={siteConfig.homeTopImageAlt}
          className="top-image"
        />
        <h1 className="page-title">{siteConfig.homeWelcome}</h1>
      </header>
      <PostList posts={posts} />
      <hr />
      <p>archive</p>
    </>
  );
}
