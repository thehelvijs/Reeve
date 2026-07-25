#!/usr/bin/env python3
"""Generate Reeve PWA icons deterministically.

Near-black canvas (DESIGN.md --canvas) with a centered acid-lime rounded mark,
sized inside the maskable safe zone. Run: python3 scripts/gen_pwa_icons.py
"""
from pathlib import Path

from PIL import Image, ImageDraw

CANVAS = (8, 9, 10, 255)      # #08090a near-black
ACCENT = (228, 242, 34, 255)  # #e4f222 acid-lime
OUT_DIR = Path(__file__).resolve().parent.parent / "web" / "public"


def render(size: int) -> Image.Image:
    img = Image.new("RGBA", (size, size), CANVAS)
    draw = ImageDraw.Draw(img)
    # Mark inside the maskable safe zone (~55% of the canvas, centered).
    mark = round(size * 0.55)
    off = (size - mark) // 2
    radius = round(mark * 0.22)
    draw.rounded_rectangle([off, off, off + mark, off + mark], radius=radius, fill=ACCENT)
    # A near-black slot echoes the Reeve brand without extra assets.
    slot_w = round(mark * 0.14)
    slot_h = round(mark * 0.42)
    sx = size // 2 - slot_w // 2
    sy = size // 2 - slot_h // 2
    draw.rounded_rectangle([sx, sy, sx + slot_w, sy + slot_h], radius=slot_w // 2, fill=CANVAS)
    return img


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    for size in (192, 512):
        render(size).save(OUT_DIR / f"icon-{size}.png")
        print(f"wrote {OUT_DIR / f'icon-{size}.png'}")


if __name__ == "__main__":
    main()
