import type { ReactNode } from "react";

interface CardProps {
  children: ReactNode;
  className?: string;
}

export function Card({ children, className }: CardProps) {
  return <div className={["card", className].filter(Boolean).join(" ")}>{children}</div>;
}

interface CardHeadProps {
  icon?: ReactNode;
  title: ReactNode;
  count?: number;
  right?: ReactNode;
}

export function CardHead({ icon, title, count, right }: CardHeadProps) {
  return (
    <div className="card-head">
      <h2>
        {icon}
        {title}
        {count !== undefined && <span className="count">{count}</span>}
      </h2>
      {right && <div className="card-head-right">{right}</div>}
    </div>
  );
}

interface CardFootProps {
  left?: ReactNode;
  right?: ReactNode;
}

export function CardFoot({ left, right }: CardFootProps) {
  return (
    <div className="card-foot">
      <span>{left}</span>
      <span>{right}</span>
    </div>
  );
}
