import type { CSSProperties, Key, ReactNode } from "react";
import {
  Button,
  Checkbox,
  CheckboxGroup,
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
  Radio,
  RadioGroup,
  Row,
  Select,
  SelectValue,
  Tab,
  TabList,
  TabPanel,
  Tabs,
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

type AdminCheckboxGroupFieldProps = {
  label: ReactNode;
  values: string[];
  options: ChoiceOption[];
  onChange: (values: string[]) => void;
  ariaLabel?: string;
};

export function AdminCheckboxGroupField({
  label,
  values,
  options,
  onChange,
  ariaLabel,
}: AdminCheckboxGroupFieldProps) {
  return (
    <CheckboxGroup
      aria-label={ariaLabel}
      className="admin-field"
      value={values}
      onChange={(next) => onChange([...next])}
    >
      <Label>{label}</Label>
      <div className="admin-choice-group">
        {options.map((option) => (
          <Checkbox
            key={option.value}
            className="admin-choice-pill"
            isDisabled={option.disabled}
            value={option.value}
          >
            {({ isSelected }) => <span className={isSelected ? "is-selected" : undefined}>{option.label}</span>}
          </Checkbox>
        ))}
      </div>
    </CheckboxGroup>
  );
}

type AdminRadioGroupFieldProps = {
  label: ReactNode;
  value: string;
  options: ChoiceOption[];
  onChange: (value: string) => void;
  ariaLabel?: string;
};

export function AdminRadioGroupField({
  label,
  value,
  options,
  onChange,
  ariaLabel,
}: AdminRadioGroupFieldProps) {
  return (
    <RadioGroup aria-label={ariaLabel} className="admin-field" value={value} onChange={onChange}>
      <Label>{label}</Label>
      <div className="admin-choice-group">
        {options.map((option) => (
          <Radio
            key={option.value}
            className="admin-choice-pill"
            isDisabled={option.disabled}
            value={option.value}
          >
            {({ isSelected }) => <span className={isSelected ? "is-selected" : undefined}>{option.label}</span>}
          </Radio>
        ))}
      </div>
    </RadioGroup>
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

type ChoiceOption = {
  value: string;
  label: ReactNode;
  disabled?: boolean;
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

type AdminConfirmDialogProps = {
  open: boolean;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
  confirmDisabled?: boolean;
};

export function AdminConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  onConfirm,
  onCancel,
  confirmDisabled,
}: AdminConfirmDialogProps) {
  return (
    <AdminDialog open={open} onClose={onCancel} title={title}>
      <>
        <div className="admin-modal-head">
          <h2>{title}</h2>
          <AdminButton className="admin-modal-close" onPress={onCancel}>
            {cancelLabel}
          </AdminButton>
        </div>
        <div className="admin-modal-body">
          <p className="admin-note">{message}</p>
          <div className="admin-toolbar-actions">
            <AdminButton onPress={onCancel}>{cancelLabel}</AdminButton>
            <AdminButton className="admin-primary" disabled={confirmDisabled} onPress={onConfirm}>
              {confirmLabel}
            </AdminButton>
          </div>
        </div>
      </>
    </AdminDialog>
  );
}

type AdminTabsProps<T extends string> = {
  selectedKey: T;
  onSelectionChange: (key: T) => void;
  label: string;
  tabs: Array<{ id: T; label: ReactNode; panel: ReactNode }>;
};

export function AdminTabs<T extends string>({
  selectedKey,
  onSelectionChange,
  label,
  tabs,
}: AdminTabsProps<T>) {
  return (
    <Tabs
      aria-label={label}
      className="admin-tabs"
      selectedKey={selectedKey}
      onSelectionChange={(key) => onSelectionChange(String(key) as T)}
    >
      <TabList className="admin-markdown-tabs">
        {tabs.map((tab) => (
          <Tab className="admin-tab-trigger" id={tab.id} key={tab.id}>
            {tab.label}
          </Tab>
        ))}
      </TabList>
      {tabs.map((tab) => (
        <TabPanel className="admin-tab-panel" id={tab.id} key={tab.id}>
          {tab.panel}
        </TabPanel>
      ))}
    </Tabs>
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
