type FormStatusMessageProps = {
  error?: string;
  success?: string;
};

export default function FormStatusMessage({ error, success }: FormStatusMessageProps) {
  return (
    <>
      {error ? (
        <p aria-live="assertive" className="admin-error" role="alert">
          {error}
        </p>
      ) : null}
      {success ? (
        <p aria-live="polite" className="admin-success" role="status">
          {success}
        </p>
      ) : null}
    </>
  );
}
