#!/usr/bin/env python3
"""Rebuild assets/bathy.bin from NOAA ETOPO 2022 for Santa Catalina.

Viewport is a 20x20 nautical mile square centered on 33.30 N, 118.45 W.
Source: NOAA NCEI ETOPO 2022 via ERDDAP (oceanwatch.pifsc.noaa.gov).
"""

from __future__ import annotations

import csv
import math
import struct
import urllib.request
from array import array
from pathlib import Path

CENTER_LAT = 33.30
CENTER_LON = 241.55
LAT_MIN, LAT_MAX = 33.133632770391664, 33.46636722960833
LON_MIN, LON_MAX = 241.35095027000483, 241.7490497299952

M_PER_DEG_LAT = 111320.0
M_PER_DEG_LON = 111320.0 * math.cos(math.radians(CENTER_LAT))
YD_PER_M = 1.0936133
FT_PER_M = 3.2808399
CELL_YD = 250.0

ERDDAP_URL = (
    "https://oceanwatch.pifsc.noaa.gov/erddap/griddap/ETOPO_2022_v1_15s.csv?"
    f"z%5B({LAT_MIN}):({LAT_MAX})%5D%5B({LON_MIN}):({LON_MAX})%5D"
)

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "assets" / "bathy.bin"


def elev_to_game_ft(z_m: float) -> float:
    # Game convention: positive = water depth (ft); <= 0 = land.
    if z_m >= 0:
        return -max(5.0, z_m * FT_PER_M)
    return -z_m * FT_PER_M


def main() -> None:
    print("Downloading ETOPO 2022 subset…")
    with urllib.request.urlopen(ERDDAP_URL, timeout=180) as resp:
        text = resp.read().decode("utf-8")

    rows: list[tuple[float, float, float]] = []
    rdr = csv.reader(text.splitlines())
    next(rdr)
    next(rdr)
    for row in rdr:
        if len(row) < 3 or not row[2]:
            continue
        rows.append((float(row[0]), float(row[1]), float(row[2])))

    lats = sorted({lat for lat, _, _ in rows})
    lons = sorted({lon for _, lon, _ in rows})
    zmap = {(lat, lon): z for lat, lon, z in rows}
    print(f"source grid {len(lons)}x{len(lats)} samples={len(rows)}")

    dlat = lats[1] - lats[0]
    dlon = lons[1] - lons[0]
    cell_x = dlon * M_PER_DEG_LON * YD_PER_M
    cell_y = dlat * M_PER_DEG_LAT * YD_PER_M
    xs = [(lon - CENTER_LON) * M_PER_DEG_LON * YD_PER_M for lon in lons]
    ys = [(lat - CENTER_LAT) * M_PER_DEG_LAT * YD_PER_M for lat in lats]
    min_x, max_x = xs[0] - cell_x / 2, xs[-1] + cell_x / 2
    min_y, max_y = ys[0] - cell_y / 2, ys[-1] + cell_y / 2

    w = int(math.ceil((max_x - min_x) / CELL_YD))
    h = int(math.ceil((max_y - min_y) / CELL_YD))
    ox, oy = min_x, min_y

    def sample_z(lat: float, lon: float) -> float:
        if lat <= lats[0]:
            j0 = 0
        elif lat >= lats[-1]:
            j0 = len(lats) - 2
        else:
            j0 = next(j for j in range(len(lats) - 1) if lats[j] <= lat <= lats[j + 1])
        if lon <= lons[0]:
            i0 = 0
        elif lon >= lons[-1]:
            i0 = len(lons) - 2
        else:
            i0 = next(i for i in range(len(lons) - 1) if lons[i] <= lon <= lons[i + 1])
        i1, j1 = i0 + 1, j0 + 1
        tx = (lon - lons[i0]) / (lons[i1] - lons[i0])
        ty = (lat - lats[j0]) / (lats[j1] - lats[j0])
        z00 = zmap[(lats[j0], lons[i0])]
        z10 = zmap[(lats[j0], lons[i1])]
        z01 = zmap[(lats[j1], lons[i0])]
        z11 = zmap[(lats[j1], lons[i1])]
        return z00 * (1 - tx) * (1 - ty) + z10 * tx * (1 - ty) + z01 * (1 - tx) * ty + z11 * tx * ty

    depths = array("f")
    for j in range(h):
        wy = oy + (j + 0.5) * CELL_YD
        lat = CENTER_LAT + (wy / YD_PER_M) / M_PER_DEG_LAT
        for i in range(w):
            wx = ox + (i + 0.5) * CELL_YD
            lon = CENTER_LON + (wx / YD_PER_M) / M_PER_DEG_LON
            depths.append(elev_to_game_ft(sample_z(lat, lon)))

    buf = bytearray()
    buf += b"BATH"
    buf += struct.pack("<I", 1)
    buf += struct.pack("<II", w, h)
    buf += struct.pack("<ddd", ox, oy, CELL_YD)
    buf += depths.tobytes()
    OUT.write_bytes(buf)

    land = sum(1 for d in depths if d <= 0)
    water = len(depths) - land
    print(
        f"wrote {OUT} ({len(buf)} bytes) {w}x{h} cell={CELL_YD} "
        f"land={land} water={water} maxDepthFt={max(depths):.0f}"
    )


if __name__ == "__main__":
    main()
