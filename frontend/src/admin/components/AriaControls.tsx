import type { CSSProperties, Key, ReactNode } from "react";
import {
  Button,
  Checkbox,
  Dialog,
  Input,
  Label,
  ListBox,
  ListBoxItem,
  Modal,
  ModalOverlay,
  Popover,
  Select,
  SelectValue,
  TextArea,
  TextField,
} from "react-aria-components";

type AdminButtonProps = {
  children: ReactNode;
  className?: string;
  disabled?: boolean;
  onPress?: () => void;
  type?: "button" | "submit";
  ariaLabel?: string;
  style?: CSSProperties;
};

export function AdminButton({
  children,
  className,
  disabled,
  onPress,
  type = "button",
  ariaLabel,
  style,
}: AdminButtonProps) {
  return (
    <Button
      aria-label={ariaLabel}
      className={className}
      isDisabled={disabled}
      onPress={onPress}
      style={style}
      type={type}
    >
      {children}
    </Button>
  );
}

type AdminTextFieldProps = {
  label: ReactNode;
  value: string;
  onChange: (value: string) => void;
  className?: string;
  inputClassName?: string;
  type?: "text" | "email" | "password" | "search" | "url" | "datetime-local" | "number";
  placeholder?: string;
  disabled?: boolean;
  required?: boolean;
  min?: number;
  max?: number;
  step?: number;
  enterKeyHint?: "enter" | "done" | "go" | "next" | "previous" | "search" | "send";
  onBlur?: () => void;
  onKeyDown?: React.KeyboardEventHandler<HTMLInputElement>;
  onFocus?: React.FocusEventHandler<HTMLInputElement>;
  ariaLabel?: string;
};

export function AdminTextField({
  label,
  value,
  onChange,
  className,
  inputClassName = "admin-input",
  type = "text",
  placeholder,
  disabled,
  required,
  min,
  max,
  step,
  enterKeyHint,
  onBlur,
  onKeyDown,
  onFocus,
  ariaLabel,
}: AdminTextFieldProps) {
  return (
    <TextField className={className} isDisabled={disabled} isRequired={required} type={type}>
      {label === "" ? null : <Label>{label}</Label>}
      <Input
        aria-label={ariaLabel}
        className={inputClassName}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        min={min}
        max={max}
        step={step}
        enterKeyHint={enterKeyHint}
        onBlur={onBlur}
        onKeyDown={onKeyDown}
        onFocus={onFocus}
      />
    </TextField>
  );
}

type AdminTextAreaFieldProps = {
  label: ReactNode;
  value: string;
  onChange: (value: string) => void;
  className?: string;
  rows?: number;
  placeholder?: string;
  disabled?: boolean;
};

export function AdminTextAreaField({
  label,
  value,
  onChange,
  className,
  rows = 3,
  placeholder,
  disabled,
}: AdminTextAreaFieldProps) {
  return (
    <TextField className={className} isDisabled={disabled}>
      <Label>{label}</Label>
      <TextArea
        className="admin-input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={rows}
        placeholder={placeholder}
      />
    </TextField>
  );
}

type AdminCheckboxFieldProps = {
  label: ReactNode;
  checked: boolean;
  onChange: (checked: boolean) => void;
  className?: string;
  disabled?: boolean;
  ariaLabel?: string;
};

export function AdminCheckboxField({
  label,
  checked,
  onChange,
  className = "admin-check admin-check-right",
  disabled,
  ariaLabel,
}: AdminCheckboxFieldProps) {
  return (
    <Checkbox
      aria-label={ariaLabel}
      className={className}
      isSelected={checked}
      onChange={onChange}
      isDisabled={disabled}
    >
      {({ isSelected }) => (
        <>
          {label === "" ? null : <span>{label}</span>}
          <span className="admin-checkbox-box" aria-hidden="true">
            {isSelected ? "✓" : ""}
          </span>
        </>
      )}
    </Checkbox>
  );
}

type SelectOption = {
  value: string | number;
  label: ReactNode;
};

type AdminSelectFieldProps = {
  label: ReactNode;
  value: string | number;
  onChange: (value: string) => void;
  options: SelectOption[];
  className?: string;
  disabled?: boolean;
  placeholder?: string;
  ariaLabel?: string;
};

export function AdminSelectField({
  label,
  value,
  onChange,
  options,
  className,
  disabled,
  placeholder,
  ariaLabel,
}: AdminSelectFieldProps) {
  return (
    <Select
      className={className}
      selectedKey={String(value)}
      onSelectionChange={(key: Key | null) => onChange(key == null ? "" : String(key))}
      isDisabled={disabled}
    >
      {label === "" ? null : <Label>{label}</Label>}
      <Button aria-label={ariaLabel} className="admin-input admin-select-trigger">
        <SelectValue>{({ selectedText }) => selectedText || placeholder || ""}</SelectValue>
        <span aria-hidden="true">▾</span>
      </Button>
      <Popover className="admin-select-popover">
        <ListBox className="admin-select-list">
          {options.map((option) => (
            <ListBoxItem className="admin-select-option" id={String(option.value)} key={String(option.value)}>
              {option.label}
            </ListBoxItem>
          ))}
        </ListBox>
      </Popover>
    </Select>
  );
}

type AdminDialogProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
};

export function AdminDialog({ open, onClose, title, children }: AdminDialogProps) {
  if (!open) return null;

  return (
    <ModalOverlay className="admin-modal-backdrop" isOpen={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <Modal className="admin-modal-shell is-open" isDismissable>
        <Dialog className="admin-modal" aria-label={title}>
          {children}
        </Dialog>
      </Modal>
    </ModalOverlay>
  );
}
