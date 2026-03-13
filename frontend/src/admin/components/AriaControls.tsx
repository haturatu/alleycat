import type { CSSProperties, Key, ReactNode } from "react";
import {
  Button,
  Checkbox,
  ComboBox,
  Cell,
  Column,
  Dialog,
  FileTrigger,
  Input,
  Label,
  ListBox,
  ListBoxItem,
  Modal,
  ModalOverlay,
  Popover,
  Row,
  Select,
  SelectValue,
  Table,
  TableBody,
  TableHeader,
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
    <TextField
      aria-label={ariaLabel}
      className={className}
      isDisabled={disabled}
      isRequired={required}
      type={type}
    >
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
  slot?: string;
};

export function AdminCheckboxField({
  label,
  checked,
  onChange,
  className = "admin-check admin-check-right",
  disabled,
  ariaLabel,
  slot,
}: AdminCheckboxFieldProps) {
  return (
    <Checkbox
      aria-label={ariaLabel}
      className={className}
      isSelected={checked}
      onChange={onChange}
      isDisabled={disabled}
      slot={slot}
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

type ComboBoxOption = {
  value: string;
  label: ReactNode;
};

type AdminTableColumn<T> = {
  id: string;
  name: ReactNode;
  width?: string;
  render: (item: T) => ReactNode;
  isRowHeader?: boolean;
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
      aria-label={ariaLabel}
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

type AdminComboBoxFieldProps = {
  label: ReactNode;
  value: string;
  onInputChange: (value: string) => void;
  options: ComboBoxOption[];
  className?: string;
  placeholder?: string;
  disabled?: boolean;
  onSelectionChange?: (value: string) => void;
  onBlur?: () => void;
  onKeyDown?: React.KeyboardEventHandler<HTMLInputElement>;
  enterKeyHint?: "enter" | "done" | "go" | "next" | "previous" | "search" | "send";
  ariaLabel?: string;
};

export function AdminComboBoxField({
  label,
  value,
  onInputChange,
  options,
  className,
  placeholder,
  disabled,
  onSelectionChange,
  onBlur,
  onKeyDown,
  enterKeyHint,
  ariaLabel,
}: AdminComboBoxFieldProps) {
  return (
    <ComboBox
      aria-label={ariaLabel}
      className={className}
      inputValue={value}
      isDisabled={disabled}
      menuTrigger="input"
      onInputChange={onInputChange}
      onSelectionChange={(key) => onSelectionChange?.(key == null ? "" : String(key))}
    >
      <Label>{label}</Label>
      <div className="admin-combobox-row">
        <Input
          aria-label={ariaLabel}
          className="admin-input"
          enterKeyHint={enterKeyHint}
          onBlur={onBlur}
          onKeyDown={onKeyDown}
          placeholder={placeholder}
        />
        <Button className="admin-input admin-select-trigger" aria-label={`${String(label)} options`}>
          <span aria-hidden="true">▾</span>
        </Button>
      </div>
      <Popover className="admin-select-popover">
        <ListBox className="admin-select-list">
          {options.map((item) => (
            <ListBoxItem className="admin-select-option" id={item.value} key={item.value} textValue={String(item.label)}>
              {item.label}
            </ListBoxItem>
          ))}
        </ListBox>
      </Popover>
    </ComboBox>
  );
}

type AdminFileTriggerFieldProps = {
  label: ReactNode;
  buttonLabel: string;
  description?: ReactNode;
  acceptedFileTypes?: string[];
  allowsMultiple?: boolean;
  onSelect: (files: File[] | null) => void;
  disabled?: boolean;
};

export function AdminFileTriggerField({
  label,
  buttonLabel,
  description,
  acceptedFileTypes,
  allowsMultiple,
  onSelect,
  disabled,
}: AdminFileTriggerFieldProps) {
  return (
    <div className="admin-field">
      <span>{label}</span>
      <div className="admin-inline admin-file-row">
        <FileTrigger
          acceptedFileTypes={acceptedFileTypes}
          allowsMultiple={allowsMultiple}
          onSelect={(files) => onSelect(files ? Array.from(files) : null)}
        >
          <AdminButton className="admin-primary" disabled={disabled}>
            {buttonLabel}
          </AdminButton>
        </FileTrigger>
        {description ? <p className="admin-note">{description}</p> : null}
      </div>
    </div>
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

type AdminTableProps<T extends { id: string }> = {
  ariaLabel: string;
  columns: AdminTableColumn<T>[];
  items: T[];
};

export function AdminTable<T extends { id: string }>({
  ariaLabel,
  columns,
  items,
}: AdminTableProps<T>) {
  return (
    <div className="admin-table-wrap">
      <Table aria-label={ariaLabel} className="admin-table">
        <TableHeader>
          {columns.map((column) => (
            <Column
              className="admin-table-column"
              id={column.id}
              isRowHeader={column.isRowHeader}
              key={column.id}
              style={column.width ? { width: column.width } : undefined}
            >
              {column.name}
            </Column>
          ))}
        </TableHeader>
        <TableBody items={items}>
          {(item) => (
            <Row id={item.id}>
              {columns.map((column) => (
                <Cell className="admin-table-cell" key={column.id}>
                  {column.render(item)}
                </Cell>
              ))}
            </Row>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
