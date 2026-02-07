import { Link } from "react-router-dom";

export default function Pagination({
  baseUrl,
  page,
  totalPages,
}: {
  baseUrl: string;
  page: number;
  totalPages: number;
}) {
  if (totalPages <= 1) return null;
  const prev = page > 1 ? page - 1 : null;
  const next = page < totalPages ? page + 1 : null;

  const buildLink = (target: number) =>
    target === 1 ? `${baseUrl}/` : `${baseUrl}/${target}/`;

  return (
    <nav className="page-pagination pagination">
      <ul>
        {prev && (
          <li className="pagination-prev">
            <Link to={buildLink(prev)} rel="prev">
              <span>前のページ</span>
              <strong>{prev}</strong>
            </Link>
          </li>
        )}
        {next && (
          <li className="pagination-next">
            <Link to={buildLink(next)} rel="next">
              <span>次のページ</span>
              <strong>{next}</strong>
            </Link>
          </li>
        )}
      </ul>
    </nav>
  );
}
