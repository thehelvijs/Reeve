import { Link } from 'react-router-dom';
import { UI_VERSION } from '../version';

// The brand mark. Lime is unreadable as text on a light canvas, so the accent
// is a highlight behind the word rather than the word itself — the one place
// the color appears that is not the view's primary button.
export default function Wordmark({ version = true }: { version?: boolean }) {
  return (
    <Link to="/" className="flex shrink-0 items-baseline gap-1.5">
      <span className="rounded-button bg-accent px-1.5 text-sm font-semibold text-accent-fg">Reeve</span>
      {version && <span className="text-[10px] font-medium tabular-nums text-muted">{UI_VERSION}</span>}
    </Link>
  );
}
