#!/usr/bin/env python3
"""Deepen navigation channels on taiwan_penghu so strait/bay/south passages connect."""

from __future__ import annotations

import struct
from array import array
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BIN = ROOT / "scenarios_generated" / "theater_bathy" / "taiwan_penghu.bin"
TARGET_DEPTH_FT = 62.0

# World-yard corridors: deepen shallow water / cut narrow land gaps for surface ships.
CORRIDORS = [
    # NE bay pocket -> central strait (between islands)
    {"x0": 5000, "x1": 6400, "y0": 2400, "y1": 5200},
    # Strait south exit -> east-side open water
    {"x0": 7000, "x1": 10000, "y0": -800, "y1": 1700},
    # East of southern island down to enemy transit lanes
    {"x0": 8500, "x1": 12200, "y0": -9000, "y1": 1800},
    # East coast northbound return (east of main island)
    {"x0": 7000, "x1": 11500, "y0": 2400, "y1": 5400},
    # West strait pocket hook-up (optional merchant lane west of long island)
    {"x0": 1200, "x1": 3400, "y0": 200, "y1": 3600},
    {"x0": 2000, "x1": 5200, "y0": 200, "y1": 1400},
    {"x0": 3200, "x1": 5600, "y0": 1200, "y1": 2600},
]


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
    hdr = struct.pack("<4sIII", b"BATH", 1, w, h)
    hdr += struct.pack("<ddd", ox, oy, cs)
  # array.tobytes()
    path.write_bytes(hdr + depths.tobytes())


def patch():
    w, h, ox, oy, cs, depths = load_bin(BIN)
    patched = 0
    for box in CORRIDORS:
        for j in range(h):
            cy = oy + (j + 0.5) * cs
            if cy < box["y0"] or cy > box["y1"]:
                continue
            for i in range(w):
                cx = ox + (i + 0.5) * cs
                if cx < box["x0"] or cx > box["x1"]:
                    continue
                idx = j * w + i
                d = depths[idx]
                if d <= 0 or d < 40:
                    depths[idx] = TARGET_DEPTH_FT
                    patched += 1
                elif d < 50:
                    depths[idx] = TARGET_DEPTH_FT
                    patched += 1
    save_bin(BIN, w, h, ox, oy, cs, depths)
    print(f"patched {patched} cells in {BIN}")


if __name__ == "__main__":
    patch()
