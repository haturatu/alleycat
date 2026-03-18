import { AdminButton } from "./AriaControls";
import LoadingSpinner from "./LoadingSpinner";

type SaveButtonProps = {
  onClick: () => void;
  saving: boolean;
  disabled?: boolean;
  idleLabel?: string;
  savingLabel?: string;
};

export default function SaveButton({
  onClick,
  saving,
  disabled = false,
  idleLabel = "Save",
  savingLabel = "Saving…",
}: SaveButtonProps) {
  return (
    <AdminButton className="admin-primary" onPress={onClick} disabled={saving || disabled}>
      <span className="admin-button-label">
        <span>{idleLabel}</span>
        {saving ? (
          <span className="admin-button-progress" aria-live="polite">
            <LoadingSpinner label={savingLabel} />
            <span>{savingLabel}</span>
          </span>
        ) : null}
      </span>
    </AdminButton>
  );
}
