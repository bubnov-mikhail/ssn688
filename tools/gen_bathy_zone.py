#!/usr/bin/env python3
"""Generate BATH v1 grids from ETOPO 2022 for scenario theaters.

Usage:
  python tools/gen_bathy_zone.py                    # all Taiwan mission theaters
  python tools/gen_bathy_zone.py --zone taiwan_strait
  python tools/gen_bathy_zone.py --list

Land fraction target: 10–30% (warn if outside).
"""

from __future__ import annotations

import argparse
import csv
import math
import struct
import subprocess
import urllib.request
from array import array
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT_DIR = ROOT / "scenarios_generated" / "theater_bathy"

M_PER_DEG_LAT = 111_320.0
YD_PER_M = 1.0936133
FT_PER_M = 3.2808399
CELL_YD = 250.0
NM_YD = 2025.37183


@dataclass
class ZoneSpec:
    id: str
    title_en: str
    title_ru: str
    coast_type: str
    center_lat: float
    center_lon: float  # east positive (121 E)
    span_nm: float = 20.0
    land_min: float = 0.10
    land_max: float = 0.30
    depth_offset_ft: float = 0.0


TAIWAN_ZONES: list[ZoneSpec] = [
    ZoneSpec(
        "taiwan_east",
        "East Coast Offshore",
        "Восточное побережье (офшор)",
        "простой берег вдоль острова",
        23.97,
        121.68,
    ),
    ZoneSpec(
        "taiwan_strait",
        "Kinmen / Fujian Littoral",
        "Прибрежье Фуцзянь / Цзиньмэнь",
        "материковый берег + острова, узкий фарватер",
        24.48,
        118.70,
        span_nm=16.0,
    ),
    ZoneSpec(
        "taiwan_penghu",
        "Penghu Archipelago",
        "Архипелаг Пэнху",
        "мелкие острова, лабиринт проливов",
        23.58,
        119.52,
        span_nm=13.0,
        depth_offset_ft=200.0,
    ),
    ZoneSpec(
        "taiwan_south",
        "Bashi Channel Approach",
        "Подход к проливу Баши",
        "мыс, глубокий канал, излом берега",
        21.97,
        120.92,
        span_nm=9.0,
    ),
    ZoneSpec(
        "taiwan_north",
        "Northeast Coast / Yilan Shelf",
        "Северо-восток / шельф Илань",
        "заливы, выступы, сложный рельеф",
        24.75,
        121.95,
        span_nm=16.0,
    ),
    ZoneSpec(
        "taiwan_lanyu",
        "Orchid Island (Lanyu) Waters",
        "Воды о. Ланьюй",
        "одиночный вулканический остров",
        22.05,
        121.55,
        span_nm=12.0,
    ),
    ZoneSpec(
        "taiwan_overview",
        "Taiwan Campaign Region",
        "Регион кампании (обзор)",
        "полный регион для overview-карты",
        23.40,
        120.35,
        span_nm=240.0,
        land_min=0.05,
        land_max=0.45,
    ),
]

# mission 2..6 theater assignment (mission 1 keeps taiwan_east)
MISSION_THEATER = {
    "tw_twin_exercises": "taiwan_strait",
    "tw_attribution": "taiwan_penghu",
    "tw_contested": "taiwan_south",
    "tw_combined_asw": "taiwan_north",
    "tw_break_pressure": "taiwan_lanyu",
}


def elev_to_game_ft(z_m: float) -> float:
    if z_m >= 0:
        return -max(5.0, z_m * FT_PER_M)
    return -z_m * FT_PER_M


def lon_360(lon_e: float) -> float:
    return lon_e if lon_e >= 0 else 360.0 + lon_e


def bbox(spec: ZoneSpec) -> tuple[float, float, float, float]:
    half = spec.span_nm * NM_YD / 2.0
    m_per_deg_lon = M_PER_DEG_LAT * math.cos(math.radians(spec.center_lat))
    dlat = half / (M_PER_DEG_LAT * YD_PER_M)
    dlon = half / (m_per_deg_lon * YD_PER_M)
    lat_min = spec.center_lat - dlat
    lat_max = spec.center_lat + dlat
    lon_min = lon_360(spec.center_lon) - dlon
    lon_max = lon_360(spec.center_lon) + dlon
    return lat_min, lat_max, lon_min, lon_max


def fetch_etopo(lat_min: float, lat_max: float, lon_min: float, lon_max: float) -> list[tuple[float, float, float]]:
    url = (
        "https://oceanwatch.pifsc.noaa.gov/erddap/griddap/ETOPO_2022_v1_15s.csv?"
        f"z%5B({lat_min}):({lat_max})%5D%5B({lon_min}):({lon_max})%5D"
    )
    print(f"  ETOPO {url[:90]}…")
    with urllib.request.urlopen(url, timeout=240) as resp:
        text = resp.read().decode("utf-8")
    rows: list[tuple[float, float, float]] = []
    rdr = csv.reader(text.splitlines())
    next(rdr)
    next(rdr)
    for row in rdr:
        if len(row) < 3 or not row[2]:
            continue
        rows.append((float(row[0]), float(row[1]), float(row[2])))
    return rows


def build_bathy(spec: ZoneSpec) -> tuple[bytes, dict]:
    lat_min, lat_max, lon_min, lon_max = bbox(spec)
    rows = fetch_etopo(lat_min, lat_max, lon_min, lon_max)
    lats = sorted({lat for lat, _, _ in rows})
    lons = sorted({lon for _, lon, _ in rows})
    zmap = {(lat, lon): z for lat, lon, z in rows}

    center_lon = lon_360(spec.center_lon)
    m_per_deg_lon = M_PER_DEG_LAT * math.cos(math.radians(spec.center_lat))
    dlat = lats[1] - lats[0]
    dlon = lons[1] - lons[0]
    cell_x = dlon * m_per_deg_lon * YD_PER_M
    cell_y = dlat * M_PER_DEG_LAT * YD_PER_M
    xs = [(lon - center_lon) * m_per_deg_lon * YD_PER_M for lon in lons]
    ys = [(lat - spec.center_lat) * M_PER_DEG_LAT * YD_PER_M for lat in lats]
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
        lat = spec.center_lat + (wy / YD_PER_M) / M_PER_DEG_LAT
        for i in range(w):
            wx = ox + (i + 0.5) * CELL_YD
            lon = center_lon + (wx / YD_PER_M) / m_per_deg_lon
            depths.append(elev_to_game_ft(sample_z(lat, lon)))

    if spec.depth_offset_ft:
        for i in range(len(depths)):
            if depths[i] > 0:
                depths[i] += spec.depth_offset_ft

    land = sum(1 for d in depths if d <= 0)
    land_frac = land / len(depths)
    buf = bytearray()
    buf += b"BATH"
    buf += struct.pack("<I", 1)
    buf += struct.pack("<II", w, h)
    buf += struct.pack("<ddd", ox, oy, CELL_YD)
    buf += depths.tobytes()
    meta = {
        "id": spec.id,
        "w": w,
        "h": h,
        "origin_x": ox,
        "origin_y": oy,
        "cell": CELL_YD,
        "land_frac": land_frac,
        "land_pct": round(land_frac * 100, 1),
        "max_depth_ft": max(depths),
        "title_en": spec.title_en,
        "title_ru": spec.title_ru,
        "coast_type": spec.coast_type,
        "center_lat": spec.center_lat,
        "center_lon": spec.center_lon,
    }
    return bytes(buf), meta


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--zone", action="append", help="zone id (repeatable)")
    ap.add_argument("--list", action="store_true")
    ap.add_argument(
        "--preview",
        action="store_true",
        help="after build, render PNG previews with mission routes (needs scenario JSON)",
    )
    ap.add_argument(
        "--scenario",
        default=str(ROOT / "scenarios_generated" / "taiwan_formosa_watch.json"),
        help="scenario for route overlays when --preview",
    )
    args = ap.parse_args()

    if args.list:
        for z in TAIWAN_ZONES:
            print(f"{z.id:16} {z.title_en:32} {z.coast_type}")
        print("\nMission mapping:")
        for mid, tid in MISSION_THEATER.items():
            print(f"  {mid} -> {tid}")
        return

    zones = TAIWAN_ZONES
    if args.zone:
        want = set(args.zone)
        zones = [z for z in TAIWAN_ZONES if z.id in want]
        missing = want - {z.id for z in zones}
        if missing:
            raise SystemExit(f"unknown zones: {missing}")

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    summary = []
    for spec in zones:
        print(f"\n=== {spec.id} ({spec.title_en}) ===")
        data, meta = build_bathy(spec)
        out = OUT_DIR / f"{spec.id}.bin"
        out.write_bytes(data)
        ok = spec.land_min <= meta["land_frac"] <= spec.land_max
        flag = "OK" if ok else "WARN land%"
        print(
            f"  {flag} land={meta['land_pct']}% grid={meta['w']}x{meta['h']} "
            f"maxDepth={meta['max_depth_ft']:.0f}ft -> {out.name}"
        )
        summary.append(meta)

    idx_path = OUT_DIR / "zones_summary.txt"
    with idx_path.open("w", encoding="utf-8") as f:
        for m in summary:
            f.write(
                f"{m['id']}\t{m['land_pct']}%\t{m['w']}x{m['h']}\t"
                f"{m['title_en']}\t{m['coast_type']}\n"
            )
    print(f"\nwrote summary {idx_path}")

    if args.preview:
        scenario = Path(args.scenario)
        if not scenario.is_file():
            print(f"preview skipped: scenario not found ({scenario})")
        else:
            print("\nrendering location previews with routes…")
            cmd = [
                "go",
                "run",
                "./tools/render_theater_routes.go",
                "-scenario",
                str(scenario),
            ]
            try:
                subprocess.run(cmd, cwd=ROOT, check=True)
            except (subprocess.CalledProcessError, FileNotFoundError) as exc:
                print(f"preview render failed: {exc}")


if __name__ == "__main__":
    main()
