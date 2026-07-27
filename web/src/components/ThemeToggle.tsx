import { otherTheme, setTheme, useTheme } from '../lib/theme';

// One control, in the app sidebar and in the portal header, so an anonymous
// visitor can switch too. It states what it will do rather than what is on.
//
// The glyph is a line-height tall and the padding matches Button's, so the box
// comes out the same height as the button beside it without a fixed size.
export default function ThemeToggle({ className = '' }: { className?: string }) {
  const theme = useTheme();
  const next = otherTheme(theme);
  return (
    <button
      type="button"
      onClick={() => setTheme(next)}
      title={`Switch to ${next} theme`}
      aria-label={`Switch to ${next} theme`}
      className={`inline-flex shrink-0 items-center justify-center rounded-button border border-hairline-strong bg-canvas px-2 py-2 text-muted transition-colors hover:bg-surface-2 hover:text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link ${className}`}
    >
      <Glyph theme={next} />
    </button>
  );
}

function Glyph({ theme }: { theme: 'dark' | 'light' }) {
  if (theme === 'dark') {
    return (
      <svg className="h-5 w-5" viewBox="0 0 20 20" fill="none" aria-hidden>
        <path
          d="M16.5 12A6.5 6.5 0 0 1 8 3.5a6.5 6.5 0 1 0 8.5 8.5Z"
          stroke="currentColor"
          strokeWidth="1.4"
          strokeLinejoin="round"
        />
      </svg>
    );
  }
  return (
    <svg className="h-5 w-5" viewBox="0 0 20 20" fill="none" aria-hidden>
      <circle cx="10" cy="10" r="3.6" stroke="currentColor" strokeWidth="1.4" />
      <path
        d="M10 2v1.6M10 16.4V18M2 10h1.6M16.4 10H18M4.34 4.34l1.13 1.13M14.53 14.53l1.13 1.13M15.66 4.34l-1.13 1.13M5.47 14.53l-1.13 1.13"
        stroke="currentColor"
        strokeWidth="1.4"
        strokeLinecap="round"
      />
    </svg>
  );
}
