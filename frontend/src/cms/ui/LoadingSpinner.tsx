type LoadingSpinnerProps = {
  label?: string;
  className?: string;
};

export default function LoadingSpinner({
  label = "Loading",
  className = "admin-spinner",
}: LoadingSpinnerProps) {
  return <span aria-hidden="true" className={className} title={label} />;
}
