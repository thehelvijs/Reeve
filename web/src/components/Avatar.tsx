function initials(name: string, email: string): string {
  const source = name.trim() || email;
  const parts = source.split(/[\s@._-]+/).filter(Boolean);
  if (parts.length === 0) {
    return '?';
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (parts[0][0] + parts[1][0]).toUpperCase();
}

export default function Avatar({
  url,
  name,
  email,
  size = 32,
}: {
  url?: string;
  name?: string;
  email: string;
  size?: number;
}) {
  const dim = { width: size, height: size };
  if (url) {
    return (
      <img
        src={url}
        alt=""
        style={dim}
        className="shrink-0 rounded-full border border-hairline object-cover"
      />
    );
  }
  return (
    <span
      style={dim}
      className="flex shrink-0 items-center justify-center rounded-full border border-hairline bg-surface-2 text-xs font-medium text-muted"
    >
      {initials(name ?? '', email)}
    </span>
  );
}
