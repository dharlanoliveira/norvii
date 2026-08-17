import type { ReactNode } from "react";

interface EmptyStateProps {
  readonly kicker: string;
  readonly title: string;
  readonly body: string;
  readonly icon: ReactNode;
  readonly action?: ReactNode;
}

export function EmptyState({
  kicker,
  title,
  body,
  icon,
  action,
}: EmptyStateProps) {
  return (
    <section className="empty-state">
      <span className="empty-state__icon" aria-hidden="true">
        {icon}
      </span>
      <p className="kicker">{kicker}</p>
      <h1>{title}</h1>
      <p className="empty-state__body">{body}</p>
      {action}
    </section>
  );
}
