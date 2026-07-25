import { useEffect, type ReactNode } from 'react';

// Overlay dialog shell. One shape for every modal so create/edit/reveal flows
// look and behave identically across the app.
export default function Modal({
  title,
  onClose,
  children,
  size = 'lg',
  headerRight,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
  size?: 'md' | 'lg';
  headerRight?: ReactNode;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose]);

  const width = size === 'md' ? 'max-w-md' : 'max-w-lg';
  return (
    <div
      className="fixed inset-0 z-[2000] flex items-center justify-center bg-black/60 p-4"
      onClick={onClose}
      role="presentation"
    >
      <div
        className={`w-full ${width} rounded-card border border-hairline bg-surface-1 p-6`}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={title}
      >
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-lg font-semibold tracking-tight text-content">{title}</h2>
          {headerRight}
        </div>
        {children}
      </div>
    </div>
  );
}
