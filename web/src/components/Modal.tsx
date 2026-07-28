import { useEffect, useRef, type ReactNode } from 'react';

// Overlay dialog shell. One shape for every modal so create/edit/reveal flows
// look and behave identically across the app.
export default function Modal({
  title,
  onClose,
  children,
  size = 'lg',
  headerRight,
  onSubmit,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
  size?: 'md' | 'lg';
  headerRight?: ReactNode;
  // Fires on Enter, so a modal's primary action is reachable from the keyboard.
  // Pass undefined while the action is unavailable; the modal then ignores Enter.
  onSubmit?: () => void;
}) {
  const dialog = useRef<HTMLDivElement>(null);

  // Focus moves into the dialog on open, otherwise the trigger button keeps it
  // and the button guard below swallows Enter.
  useEffect(() => {
    const el = dialog.current;
    if (!el) {
      return;
    }
    const first = el.querySelector<HTMLElement>('input, textarea, select');
    if (first) {
      first.focus();
    } else {
      el.focus();
    }
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
      if (e.key !== 'Enter' || !onSubmit || e.shiftKey || e.ctrlKey || e.metaKey || e.altKey) {
        return;
      }
      const el = e.target as HTMLElement | null;
      if (el instanceof HTMLTextAreaElement || el instanceof HTMLButtonElement || el instanceof HTMLAnchorElement) {
        return;
      }
      e.preventDefault();
      onSubmit();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose, onSubmit]);

  const width = size === 'md' ? 'max-w-md' : 'max-w-lg';
  return (
    <div
      className="reeve-veil fixed inset-0 z-[2000] flex items-center justify-center bg-black/60 p-4"
      onClick={onClose}
      role="presentation"
    >
      <div
        ref={dialog}
        tabIndex={-1}
        className={`reeve-pop w-full ${width} rounded-card border border-hairline bg-canvas p-6 shadow-pop focus:outline-none`}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={title}
      >
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-base font-semibold text-content">{title}</h2>
          {headerRight}
        </div>
        {children}
      </div>
    </div>
  );
}
