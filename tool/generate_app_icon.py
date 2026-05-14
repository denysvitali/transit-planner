#!/usr/bin/env python3
"""Rasterize the master SVG at assets/icon/app_icon.svg into the platform
specific PNGs Flutter expects (Android mipmaps, iOS AppIcon set, web/PWA
icons, favicon).

The generated PNGs are intentionally git-ignored: regenerate them before
building by running `python tool/generate_app_icon.py`. CI does the same.

Requires `rsvg-convert` (Debian/Ubuntu: `librsvg2-bin`) and Pillow.
"""

from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parent.parent
SOURCE_SVG = ROOT / "assets/icon/app_icon.svg"

GREEN = (15, 159, 110)  # AppPalette.green #0F9F6E
MASTER = 1024
MASKABLE_PAD_RATIO = 0.12   # web maskable / Android adaptive safe zone
ROUNDED_RADIUS_RATIO = 0.22  # iOS-style squircle approximation


def _rasterize_svg(svg: Path, size: int) -> Image.Image:
    """Render `svg` to an RGBA PIL image at `size`x`size` via rsvg-convert."""
    if shutil.which("rsvg-convert") is None:
        sys.exit(
            "rsvg-convert not found. Install librsvg2-bin "
            "(Debian/Ubuntu: `apt install librsvg2-bin`)."
        )
    proc = subprocess.run(
        ["rsvg-convert", "--width", str(size), "--height", str(size),
         "--format", "png", str(svg)],
        check=True, capture_output=True,
    )
    from io import BytesIO
    return Image.open(BytesIO(proc.stdout)).convert("RGBA")


def _rounded_mask(size: int, radius_ratio: float) -> Image.Image:
    mask = Image.new("L", (size, size), 0)
    r = int(size * radius_ratio)
    ImageDraw.Draw(mask).rounded_rectangle(
        (0, 0, size - 1, size - 1), radius=r, fill=255,
    )
    return mask


def _apply_rounded(img: Image.Image) -> Image.Image:
    mask = _rounded_mask(img.size[0], ROUNDED_RADIUS_RATIO)
    out = Image.new("RGBA", img.size, (0, 0, 0, 0))
    out.paste(img, (0, 0), mask)
    return out


def _make_maskable(master_square: Image.Image) -> Image.Image:
    """Pad the master inside a solid-green canvas so adaptive/maskable
    cropping (which trims ~12% on each side) lands on the route artwork."""
    size = master_square.size[0]
    pad = int(size * MASKABLE_PAD_RATIO)
    inner = size - 2 * pad
    canvas = Image.new("RGBA", (size, size), GREEN + (255,))
    shrunk = master_square.resize((inner, inner), Image.LANCZOS)
    canvas.alpha_composite(shrunk, dest=(pad, pad))
    return canvas


def write(img: Image.Image, path: Path, *, flatten_to: tuple | None = None) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    if flatten_to is not None:
        flat = Image.new("RGB", img.size, flatten_to)
        flat.paste(img, (0, 0), img)
        flat.save(path, format="PNG", optimize=True)
    else:
        img.save(path, format="PNG", optimize=True)
    print(f"  wrote {path.relative_to(ROOT)}")


def main() -> None:
    if not SOURCE_SVG.exists():
        sys.exit(f"missing source SVG: {SOURCE_SVG.relative_to(ROOT)}")

    master_square = _rasterize_svg(SOURCE_SVG, MASTER)
    master_rounded = _apply_rounded(master_square)
    master_maskable = _make_maskable(master_square)

    # ---- Android legacy launcher (square + system rounding) ----
    android_sizes = {
        "mipmap-mdpi": 48,
        "mipmap-hdpi": 72,
        "mipmap-xhdpi": 96,
        "mipmap-xxhdpi": 144,
        "mipmap-xxxhdpi": 192,
    }
    for folder, size in android_sizes.items():
        out = ROOT / "android/app/src/main/res" / folder / "ic_launcher.png"
        write(master_rounded.resize((size, size), Image.LANCZOS), out)

    # ---- iOS AppIcon set ----
    ios_dir = ROOT / "ios/Runner/Assets.xcassets/AppIcon.appiconset"
    ios_files = {
        "Icon-App-20x20@1x.png": 20,
        "Icon-App-20x20@2x.png": 40,
        "Icon-App-20x20@3x.png": 60,
        "Icon-App-29x29@1x.png": 29,
        "Icon-App-29x29@2x.png": 58,
        "Icon-App-29x29@3x.png": 87,
        "Icon-App-40x40@1x.png": 40,
        "Icon-App-40x40@2x.png": 80,
        "Icon-App-40x40@3x.png": 120,
        "Icon-App-60x60@2x.png": 120,
        "Icon-App-60x60@3x.png": 180,
        "Icon-App-76x76@1x.png": 76,
        "Icon-App-76x76@2x.png": 152,
        "Icon-App-83.5x83.5@2x.png": 167,
        "Icon-App-1024x1024@1x.png": 1024,
    }
    for name, size in ios_files.items():
        target = ios_dir / name
        scaled = master_square.resize((size, size), Image.LANCZOS)
        # The 1024 marketing icon must be opaque (App Store rejects alpha).
        write(scaled, target, flatten_to=GREEN if size == 1024 else None)

    # ---- Web / PWA ----
    web_icons = ROOT / "web/icons"
    write(master_rounded.resize((192, 192), Image.LANCZOS), web_icons / "Icon-192.png")
    write(master_rounded.resize((512, 512), Image.LANCZOS), web_icons / "Icon-512.png")
    write(master_maskable.resize((192, 192), Image.LANCZOS), web_icons / "Icon-maskable-192.png")
    write(master_maskable.resize((512, 512), Image.LANCZOS), web_icons / "Icon-maskable-512.png")
    write(master_rounded.resize((32, 32), Image.LANCZOS), ROOT / "web/favicon.png")

    print("Done.")


if __name__ == "__main__":
    main()
