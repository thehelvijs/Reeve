// The brand accent, as a literal: the pin is drawn into an SVG string, where a
// CSS variable in a fill attribute would not resolve.
export const PIN_DEFAULT = '#e4f222';

const HEX = /^#[0-9a-fA-F]{6}$/;

// pinFill is the colour a marker is drawn in. Anything that is not #rrggbb falls
// back to the accent rather than reaching the SVG: this string lands in an inline
// style, and the server is not the only thing that can put a value here (a stale
// cache, a hand-edited request).
export function pinFill(color?: string | null): string {
  if (color && HEX.test(color)) {
    return color;
  }
  return PIN_DEFAULT;
}

// pinSVG draws a map pin: a teardrop with a hole, tip at the coordinate. Sized
// large enough to read a colour at a glance, which a 14px dot was not.
export function pinSVG(color?: string | null): string {
  return (
    `<svg class="reeve-pin" viewBox="0 0 24 32" width="24" height="32" aria-hidden="true">` +
    `<path d="M12 1.5c-5.8 0-10.5 4.7-10.5 10.5 0 7.6 10.5 18.5 10.5 18.5S22.5 19.6 22.5 12c0-5.8-4.7-10.5-10.5-10.5z" ` +
    `fill="${pinFill(color)}"/>` +
    `<circle cx="12" cy="12" r="4"/>` +
    `</svg>`
  );
}
