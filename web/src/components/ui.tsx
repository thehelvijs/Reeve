import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TdHTMLAttributes,
} from 'react';

// buttonBox is the geometry every button-shaped control shares: padding, radius,
// type scale and a border. Exported so a control that cannot be a <button> — a
// download link, say — occupies the same box instead of an approximation that
// ends up a few pixels short of the button beside it.
export const buttonBox =
  'inline-flex items-center justify-center whitespace-nowrap rounded-button border border-hairline-strong bg-canvas px-3 py-2 text-sm font-medium text-content transition-colors duration-150 hover:bg-surface-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-link';

// Primary: the single accent action per view. Dark label on acid-lime.
export function Button({
  variant = 'primary',
  className = '',
  type = 'button',
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary' | 'danger' }) {
  // Every variant carries a border, transparent on the primary: without it the
  // accent button is 2px shorter than the outlined ones and any row mixing them
  // sits crooked.
  const styles = {
    primary: 'border-transparent bg-accent font-semibold text-accent-fg hover:bg-accent-hover',
    secondary: 'border-hairline-strong bg-canvas text-content hover:bg-surface-2',
    danger: 'border-down-line bg-canvas text-down hover:border-down hover:bg-down-soft',
  }[variant];
  const base =
    'inline-flex items-center justify-center whitespace-nowrap rounded-button border px-3 py-2 text-sm font-medium transition-colors duration-150 focus:outline-none focus-visible:ring-2 focus-visible:ring-link disabled:opacity-50 disabled:pointer-events-none';
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
      className={`w-full rounded-button bg-canvas border border-hairline-strong px-3 py-2 text-sm text-content placeholder:text-muted focus:outline-none focus-visible:ring-2 focus-visible:ring-link ${className}`}
      {...props}
    />
  );
}

// Select carries the input's box so a dropdown beside a text field or a button
// lines up with it. Three pages had grown their own copy of this class string.
export function Select({ className = '', ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      className={`rounded-button border border-hairline-strong bg-canvas px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link disabled:opacity-50 ${className}`}
      {...props}
    />
  );
}

// The label is `block` so it sits above its control even when the control is not
// full width: a bare <span> shares a line with a <select>, which put "Auto-update"
// to the left of its dropdown while every other field in the app labelled from
// above.
export function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className="block text-xs font-semibold text-content">{label}</span>
      {children}
      {hint && <span className="block text-xs text-muted">{hint}</span>}
    </label>
  );
}

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <div className={`rounded-card border border-hairline bg-canvas ${className}`}>{children}</div>
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
    muted: 'bg-idle-soft text-idle border-idle-line',
    up: 'bg-up-soft text-up border-up-line',
    down: 'bg-down-soft text-down border-down-line',
    warn: 'bg-warn-soft text-warn border-warn-line',
  }[tone];
  return (
    <span
      className={`inline-flex items-center rounded-pill border px-2 py-0.5 text-xs font-medium ${styles}`}
    >
      {children}
    </span>
  );
}

// Eyebrow labels a group of tiles or a section of a page. Small caps, muted,
// so a heading never competes with the page title.
export function Eyebrow({ children }: { children: ReactNode }) {
  return <h2 className="text-eyebrow font-semibold uppercase text-muted">{children}</h2>;
}

// Section is the one shape a block of a page takes: a named heading, the count
// of what is under it, a line saying what it is for, and one control on the
// right. Everything below the page title is one of these, so a card, a table
// and a form all sit at the same altitude and a screen reader gets an outline.
export function Section({
  title,
  count,
  description,
  action,
  children,
}: {
  title: string;
  count?: number;
  description?: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0">
          <h2 className="text-sm font-medium text-content">
            {title}
            {count !== undefined && <span className="text-muted"> ({count})</span>}
          </h2>
          {description && <p className="mt-0.5 text-xs text-muted">{description}</p>}
        </div>
        {action && <div className="flex shrink-0 items-center gap-2">{action}</div>}
      </div>
      <div className="mt-2">{children}</div>
    </section>
  );
}

// Facts is the label/value grid a detail page uses for what a thing *is*, as
// opposed to what it is doing. A definition list, because that is what it is.
export function Facts({ items }: { items: { label: string; value: ReactNode }[] }) {
  return (
    <dl className="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3 lg:grid-cols-4">
      {items.map((f) => (
        <div key={f.label} className="min-w-0">
          <dt className="text-eyebrow font-semibold uppercase text-muted">{f.label}</dt>
          <dd className="mt-0.5 truncate text-sm text-content">{f.value}</dd>
        </div>
      ))}
    </dl>
  );
}

// Table is the default shape for a list of records: every column is named, so a
// bar or a pill in a row can be read without guessing what it measures.
export function Table({
  head,
  children,
  id,
  className = '',
}: {
  head: ReactNode;
  children: ReactNode;
  id?: string;
  className?: string;
}) {
  return (
    <div className={`overflow-x-auto rounded-card border border-hairline ${className}`}>
      <table id={id} className="w-full border-collapse text-left text-sm">
        <thead>
          <tr className="border-b border-hairline bg-surface-1">{head}</tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  );
}

export function Th({ children, className = '' }: { children?: ReactNode; className?: string }) {
  return (
    <th
      scope="col"
      className={`whitespace-nowrap px-4 py-2 text-eyebrow font-semibold uppercase text-muted ${className}`}
    >
      {children}
    </th>
  );
}

export function Tr({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <tr className={`border-b border-hairline last:border-0 hover:bg-surface-1 ${className}`}>
      {children}
    </tr>
  );
}

export function Td({ children, className = '', ...props }: TdHTMLAttributes<HTMLTableCellElement>) {
  return (
    <td className={`px-4 py-2.5 align-middle ${className}`} {...props}>
      {children}
    </td>
  );
}

// Tabs filter a list in place, or split one page into sections. A count is part
// of the label where there is something to count: it says how much is behind a
// tab before it is opened, and reads as zero rather than empty. A tab that holds
// a form rather than a list leaves it out.
export function Tabs<T extends string>({
  tabs,
  active,
  onChange,
  label,
}: {
  tabs: { key: T; label: string; count?: number }[];
  active: T;
  onChange: (key: T) => void;
  label: string;
}) {
  return (
    <div className="border-b border-hairline" role="tablist" aria-label={label}>
      <div className="-mb-px flex gap-4 overflow-x-auto">
        {tabs.map((t) => {
          const selected = t.key === active;
          let tone = 'border-transparent text-muted hover:border-hairline-strong hover:text-content';
          if (selected) {
            tone = 'border-accent font-semibold text-content';
          }
          return (
            <button
              key={t.key}
              type="button"
              role="tab"
              aria-selected={selected}
              onClick={() => onChange(t.key)}
              className={`flex items-center gap-1.5 whitespace-nowrap border-b-2 px-1 py-2 text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-link ${tone}`}
            >
              {t.label}
              {t.count !== undefined && (
                <span className="rounded-pill bg-surface-2 px-1.5 text-xs font-medium tabular-nums text-muted">
                  {t.count}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}

export function ErrorText({ children }: { children: ReactNode }) {
  if (!children) {
    return null;
  }
  return <p className="text-sm text-down">{children}</p>;
}
