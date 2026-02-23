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
  savingLabel = "Saving...",
}: SaveButtonProps) {
  return (
    <button className="admin-primary" onClick={onClick} disabled={saving || disabled}>
      {saving ? savingLabel : idleLabel}
    </button>
  );
}
