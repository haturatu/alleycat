type FormStatusMessageProps = {
  error?: string;
  success?: string;
};

export default function FormStatusMessage({ error, success }: FormStatusMessageProps) {
  return (
    <>
      {error ? <p className="admin-error">{error}</p> : null}
      {success ? <p className="admin-success">{success}</p> : null}
    </>
  );
}
