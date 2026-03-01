import { Link } from "react-router-dom";

export default function Pagination({
  baseUrl,
  page,
  totalPages,
  query = "",
}: {
  baseUrl: string;
  page: number;
  totalPages: number;
  query?: string;
}) {
  if (totalPages <= 1) return null;
  const prev = page > 1 ? page - 1 : null;
  const next = page < totalPages ? page + 1 : null;

  const buildLink = (target: number) => {
    const path = target === 1 ? `${baseUrl}/` : `${baseUrl}/${target}/`;
    const q = query.trim();
    if (!q) return path;
    const params = new URLSearchParams({ q });
    return `${path}?${params.toString()}`;
  };

  return (
    <nav className="page-pagination pagination">
      <ul>
        {prev && (
          <li className="pagination-prev">
            <Link to={buildLink(prev)} rel="prev">
              <span>Previous</span>
              <strong>{prev}</strong>
            </Link>
          </li>
        )}
        {next && (
          <li className="pagination-next">
            <Link to={buildLink(next)} rel="next">
              <span>Next</span>
              <strong>{next}</strong>
            </Link>
          </li>
        )}
      </ul>
    </nav>
  );
}
