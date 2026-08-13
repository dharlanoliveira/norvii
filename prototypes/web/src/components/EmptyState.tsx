import type { ReactNode } from "react";

interface EmptyStateProps {
  readonly eyebrow: string;
  readonly title: string;
  readonly body: string;
  readonly icon: ReactNode;
  readonly action?: ReactNode;
}

export function EmptyState({
  eyebrow,
  title,
  body,
  icon,
  action,
}: EmptyStateProps) {
  return (
    <section className="empty-state">
      <span className="empty-state-icon" aria-hidden="true">
        {icon}
      </span>
      <p className="eyebrow">{eyebrow}</p>
      <h2>{title}</h2>
      <p>{body}</p>
      {action}
    </section>
  );
}
