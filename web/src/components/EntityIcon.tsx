// EntityIcon renders a tool/host icon as a rounded square: the uploaded image,
// or a first-letter fallback on a surface tile.
export default function EntityIcon({
  url,
  name,
  size = 32,
}: {
  url?: string;
  name: string;
  size?: number;
}) {
  const dim = { width: size, height: size };
  if (url) {
    return (
      <img src={url} alt="" style={dim} className="shrink-0 rounded-button border border-hairline object-cover" />
    );
  }
  return (
    <span
      style={dim}
      className="flex shrink-0 items-center justify-center rounded-button border border-hairline bg-surface-2 text-xs font-medium text-muted"
    >
      {(name.trim()[0] || '?').toUpperCase()}
    </span>
  );
}
