import { AdminButton } from "./AriaControls";

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
    <AdminButton className="admin-primary" onPress={onClick} disabled={saving || disabled}>
      {saving ? savingLabel : idleLabel}
    </AdminButton>
  );
}
