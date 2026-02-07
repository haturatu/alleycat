import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { fileUrl, pb, PostRecord } from "../lib/pb";
import { formatDate, parseTags, readingTimeMinutes } from "../utils/text";

export default function PostPage() {
  const { slug } = useParams();
  const [post, setPost] = useState<PostRecord | null>(null);

  useEffect(() => {
    if (!slug) return;
    const safeSlug = slug.replace(/"/g, "");
    pb.collection("posts")
      .getFirstListItem<PostRecord>(
        `slug = "${safeSlug}" && published = true`,
        { expand: "author" }
      )
      .then(setPost)
      .catch(() => setPost(null));
  }, [slug]);

  if (!post) {
    return (
      <article className="post">
        <header className="post-header">
          <h1 className="post-title">Not Found</h1>
        </header>
        <div className="post-body body">記事が見つかりませんでした。</div>
      </article>
    );
  }

  const tags = parseTags(post.tags);
  const body = post.body || post.content || "";
  const featuredImageUrl = fileUrl("posts", post.id, post.featured_image);
  const attachmentUrls = (post.attachments ?? []).map((file) =>
    fileUrl("posts", post.id, file)
  );
  const authorName = (post as PostRecord & { expand?: { author?: { name?: string; email?: string } } })
    .expand?.author?.name || (post as PostRecord & { expand?: { author?: { email?: string } } }).expand?.author?.email;

  return (
    <>
      <article className="post">
        <header className="post-header">
          <h1 className="post-title">{post.title}</h1>
          <div className="post-details">
            {post.published_at && (
              <p>
                <time dateTime={post.published_at}>{formatDate(post.published_at)}</time>
              </p>
            )}
            <p>{readingTimeMinutes(body)} min</p>
            {post.category && <p>{post.category}</p>}
            {authorName && <p>{authorName}</p>}
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
        {featuredImageUrl && (
          <div className="post-body body">
            <img src={featuredImageUrl} alt="" />
          </div>
        )}
        <div className="post-body body" dangerouslySetInnerHTML={{ __html: body }} />
        {attachmentUrls.length > 0 && (
          <div className="post-body body">
            <h3>Attachments</h3>
            <ul>
              {attachmentUrls.map((url, index) => (
                <li key={url}>
                  <a href={url} target="_blank" rel="noreferrer">
                    Attachment {index + 1}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        )}
      </article>
    </>
  );
}
