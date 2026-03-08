import type { TranslationJobRecord } from "../../lib/pb";

type TranslationStatusModalProps = {
  open: boolean;
  job: TranslationJobRecord | null;
  loading: boolean;
  onClose: () => void;
};

const statusLabel = (status?: TranslationJobRecord["status"]) => {
  if (status === "completed") return "Completed";
  if (status === "failed") return "Failed";
  if (status === "running") return "Running";
  if (status === "queued") return "Queued";
  return "Preparing";
};

export default function TranslationStatusModal({
  open,
  job,
  loading,
  onClose,
}: TranslationStatusModalProps) {
  if (!open) return null;

  const total = Math.max(0, Number(job?.total_locales || 0));
  const completed = Math.max(0, Number(job?.completed_locales || 0));
  const failed = Math.max(0, Number(job?.failed_locales || 0));
  const pending = Math.max(0, total - completed - failed);
  const done = job?.status === "completed" || job?.status === "failed";

  return (
    <div className="admin-modal-backdrop" role="presentation" onClick={onClose}>
      <div
        className="admin-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="translation-status-title"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="admin-modal-head">
          <div>
            <h2 id="translation-status-title">Translation status</h2>
            <p className="admin-note">
              Source content is already saved. Translation continues in the background.
            </p>
          </div>
          <button type="button" className="admin-modal-close" onClick={onClose}>
            Close
          </button>
        </div>
        <div className="admin-modal-body">
          <div className="admin-status-grid">
            <div>
              <span className="admin-status-label">Status</span>
              <strong>{loading && !job ? "Loading..." : statusLabel(job?.status)}</strong>
            </div>
            <div>
              <span className="admin-status-label">Completed</span>
              <strong>{completed}</strong>
            </div>
            <div>
              <span className="admin-status-label">Pending</span>
              <strong>{pending}</strong>
            </div>
            <div>
              <span className="admin-status-label">Failed</span>
              <strong>{failed}</strong>
            </div>
          </div>
          {job?.last_error ? <p className="admin-error">{job.last_error}</p> : null}
          {done ? (
            <p className="admin-success">
              {job?.status === "completed"
                ? "All translation tasks finished."
                : "Translation finished with at least one failure."}
            </p>
          ) : (
            <p className="admin-note">You can close this modal and keep editing while translation runs.</p>
          )}
        </div>
      </div>
    </div>
  );
}
