#!/usr/bin/env python3
"""Add a uniform depth offset to taiwan_penghu water cells (idempotent)."""

from __future__ import annotations

import struct
from array import array
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BIN = ROOT / "scenarios_generated" / "theater_bathy" / "taiwan_penghu.bin"
OFFSET_FT = 200.0
ALREADY_DEEPENED_MIN_MAX_FT = 350.0


def load_bin(path: Path):
    raw = path.read_bytes()
    magic, ver, w, h = struct.unpack_from("<4sIII", raw, 0)
    if magic != b"BATH" or ver != 1:
        raise SystemExit(f"bad bathy header: {magic!r} v{ver}")
    ox, oy, cs = struct.unpack_from("<ddd", raw, 16)
    depths = array("f")
    depths.frombytes(raw[40:])
    return w, h, ox, oy, cs, depths


def save_bin(path: Path, w, h, ox, oy, cs, depths: array):
    hdr = struct.pack("<4sIII", b"BATH", 1, w, h) + struct.pack("<ddd", ox, oy, cs)
    path.write_bytes(hdr + depths.tobytes())


def main() -> None:
    if not BIN.is_file():
        raise SystemExit(f"missing {BIN} (run python tools/gen_bathy_zone.py --zone taiwan_penghu)")
    w, h, ox, oy, cs, depths = load_bin(BIN)
    water = [d for d in depths if d > 0]
    if not water:
        raise SystemExit("no water cells")
    prev_max = max(water)
    if prev_max >= ALREADY_DEEPENED_MIN_MAX_FT:
        print(f"{BIN.name}: max depth {prev_max:.0f}ft — offset already applied, skip")
        return
    n = 0
    for i, d in enumerate(depths):
        if d > 0:
            depths[i] = d + OFFSET_FT
            n += 1
    save_bin(BIN, w, h, ox, oy, cs, depths)
    new_max = max(d for d in depths if d > 0)
    print(f"{BIN.name}: +{OFFSET_FT:.0f}ft on {n} cells (max {prev_max:.0f} -> {new_max:.0f}ft)")


if __name__ == "__main__":
    main()
