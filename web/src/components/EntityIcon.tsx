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
  // The fallback letter scales with the tile. Fixed at text-xs it read as a
  // speck in the 96px tile the uploaders show.
  const letter = { ...dim, fontSize: Math.max(12, Math.round(size * 0.4)) };
  if (url) {
    return (
      <img
        src={url}
        alt=""
        style={dim}
        className="shrink-0 rounded-button object-cover outline outline-1 -outline-offset-1 outline-image"
      />
    );
  }
  return (
    <span
      style={letter}
      className="flex shrink-0 items-center justify-center rounded-button border border-hairline bg-surface-2 font-medium text-muted"
    >
      {(name.trim()[0] || '?').toUpperCase()}
    </span>
  );
}
