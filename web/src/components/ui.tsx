import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from 'react';

// Primary: the single accent action per view. Dark label on acid-lime.
export function Button({
  variant = 'primary',
  className = '',
  type = 'button',
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary' | 'danger' }) {
  const base =
    'inline-flex items-center justify-center whitespace-nowrap rounded-button px-3 py-2 text-sm font-medium transition-colors duration-150 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent disabled:opacity-50 disabled:pointer-events-none';
  // Every variant carries a border, transparent on the primary: without it the
  // accent button is 2px shorter than the outlined ones and any row mixing them
  // sits crooked.
  const styles = {
    primary: 'border border-transparent bg-accent text-accent-fg hover:bg-accent-hover',
    secondary: 'border border-hairline text-muted hover:text-content hover:border-hairline-strong',
    danger: 'border border-hairline text-muted hover:text-red-400 hover:border-red-400/40',
  }[variant];
  return <button type={type} className={`${base} ${styles} ${className}`} {...props} />;
}

// Every editable section is a real form, so Enter confirms it from any field.
// The primary Button inside carries type="submit"; every other one defaults to
// type="button" and stays inert.
export function Form({
  onSubmit,
  className = '',
  children,
}: {
  onSubmit: () => void;
  className?: string;
  children: ReactNode;
}) {
  return (
    <form
      className={className}
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit();
      }}
    >
      {children}
    </form>
  );
}

export function Input({ className = '', ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={`w-full rounded-button bg-surface-1 border border-hairline px-3 py-2 text-sm text-content placeholder:text-muted focus:outline-none focus-visible:ring-2 focus-visible:ring-accent ${className}`}
      {...props}
    />
  );
}

export function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className="text-xs font-medium text-muted">{label}</span>
      {children}
      {hint && <span className="block text-xs text-muted/70">{hint}</span>}
    </label>
  );
}

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <div className={`rounded-card border border-hairline bg-surface-1 ${className}`}>{children}</div>
  );
}

export function Pill({
  children,
  tone = 'muted',
}: {
  children: ReactNode;
  tone?: 'muted' | 'up' | 'down' | 'warn';
}) {
  const styles = {
    muted: 'bg-surface-2 text-muted border-hairline',
    up: 'bg-green-500/10 text-green-400 border-green-500/20',
    down: 'bg-red-500/10 text-red-400 border-red-500/20',
    warn: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
  }[tone];
  return (
    <span className={`inline-flex items-center rounded-pill border px-2 py-0.5 text-xs ${styles}`}>
      {children}
    </span>
  );
}

export function ErrorText({ children }: { children: ReactNode }) {
  if (!children) {
    return null;
  }
  return <p className="text-sm text-red-400">{children}</p>;
}
