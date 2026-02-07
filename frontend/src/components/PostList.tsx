import { Link } from "react-router-dom";
import { PostRecord } from "../lib/pb";
import { buildExcerpt, formatDate, parseTags, readingTimeMinutes } from "../utils/text";

export default function PostList({ posts }: { posts: PostRecord[] }) {
  return (
    <section className="postList">
      {posts.map((post) => {
        const tags = parseTags(post.tags);
        const body = post.body || post.content || "";
        const reading = readingTimeMinutes(body);
        return (
          <article className="post" key={post.id}>
            <header className="post-header">
              <h2 className="post-title">
                <Link to={`/posts/${post.slug}/`}>{post.title}</Link>
              </h2>
              <div className="post-details">
                {post.published_at && (
                  <p>
                    <time dateTime={post.published_at}>{formatDate(post.published_at)}</time>
                  </p>
                )}
                <p>{reading} min</p>
                {tags.length > 0 && (
                  <div className="post-tags">
                    {tags.map((tag) => (
                      <Link key={tag} className="badge" to={`/archive/${tag}/`}>
                        {tag}
                      </Link>
                    ))}
                  </div>
                )}
              </div>
            </header>
            <div className="post-excerpt body">{post.excerpt || buildExcerpt(body)}</div>
            <Link to={`/posts/${post.slug}/`} className="post-link">
              続きを読む
            </Link>
          </article>
        );
      })}
    </section>
  );
}
