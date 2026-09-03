#!/usr/bin/env python3
"""Render theater bathy + mission routes for land-crossing review.

Usage:
  python tools/render_theater_routes.py
  python tools/render_theater_routes.py --scenario scenarios_generated/taiwan_formosa_watch.json

Writes PNG previews to scenarios_generated/theater_previews/
"""

from __future__ import annotations

import argparse
import base64
import json
import math
import struct
from pathlib import Path

try:
    from PIL import Image, ImageDraw, ImageFont
except ImportError as e:
    raise SystemExit("pip install pillow") from e

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "scenarios_generated" / "theater_previews"
CELL_BLOCKED_FT = 40.0
CLEARANCE_YD = 2800.0

# distinct colors per route (cycle)
ROUTE_COLORS = [
    (255, 220, 60),
    (80, 200, 255),
    (255, 120, 80),
    (160, 255, 120),
    (220, 120, 255),
    (255, 180, 220),
    (180, 220, 255),
    (255, 255, 180),
    (120, 255, 200),
    (255, 140, 140),
    (200, 200, 200),
]


class Bathy:
    def __init__(self, raw: bytes):
        magic, ver, w, h = struct.unpack_from("<4sIII", raw, 0)
        if magic != b"BATH":
            raise ValueError("not BATH")
        ox, oy, cs = struct.unpack_from("<ddd", raw, 16)
        self.w, self.h = w, h
        self.ox, self.oy, self.cs = ox, oy, cs
        self.depths = struct.unpack_from(f"<{w*h}f", raw, 40)

    def depth_at(self, x: float, y: float) -> float:
        fx = (x - self.ox) / self.cs
        fy = (y - self.oy) / self.cs
        if fx < 0 or fy < 0 or fx >= self.w or fy >= self.h:
            return -50.0
        i = int(fx)
        j = int(fy)
        return self.depths[j * self.w + i]

    def blocked(self, x: float, y: float) -> bool:
        fx = (x - self.ox) / self.cs
        fy = (y - self.oy) / self.cs
        if fx < 0 or fy < 0 or fx >= self.w or fy >= self.h:
            return True
        d = self.depths[int(fy) * self.w + int(fx)]
        return d <= 0 or d < CELL_BLOCKED_FT

    def bounds(self) -> tuple[float, float, float, float]:
        return (
            self.ox,
            self.oy,
            self.ox + self.w * self.cs,
            self.oy + self.h * self.cs,
        )


def depth_color(d: float) -> tuple[int, int, int]:
    if d <= 0:
        return (48, 72, 40)
    depth = min(d, 6000.0)
    t = math.log1p(depth) / math.log1p(6000.0)
    # shallow sand -> deep blue
    r = int(30 + (1 - t) * 90)
    g = int(60 + (1 - t) * 100)
    b = int(40 + t * 180)
    return (r, g, b)


def world_to_px(x: float, y: float, b: Bathy, img_w: int, img_h: int, pad: int = 8) -> tuple[int, int]:
    min_x, min_y, max_x, max_y = b.bounds()
    sx = pad + (x - min_x) / (max_x - min_x) * (img_w - 2 * pad)
    sy = pad + (max_y - y) / (max_y - min_y) * (img_h - 2 * pad)
    return int(sx), int(sy)


def sample_segment(b: Bathy, x0, y0, x1, y1) -> list[tuple[float, float]]:
    hits = []
    dist = math.hypot(x1 - x0, y1 - y0)
    steps = max(2, int(dist / 200) + 1)
    for s in range(steps + 1):
        t = s / steps
        x = x0 + (x1 - x0) * t
        y = y0 + (y1 - y0) * t
        if b.blocked(x, y):
            hits.append((x, y))
    return hits


def render_theater(
    theater_id: str,
    bathy: Bathy,
    routes: list[dict],
    title: str,
    subtitle: str,
    out_path: Path,
) -> dict:
    img_w, img_h = 900, 900
    img = Image.new("RGB", (img_w, img_h), (12, 14, 18))
    px = img.load()

    # bathy raster (downsample cells)
    step = max(1, min(bathy.w, bathy.h) // 220)
    for j in range(0, bathy.h, step):
        for i in range(0, bathy.w, step):
            cx = bathy.ox + (i + 0.5) * bathy.cs
            cy = bathy.oy + (j + 0.5) * bathy.cs
            d = bathy.depth_at(cx, cy)
            col = depth_color(d)
            x0, y0 = world_to_px(cx, cy, bathy, img_w, img_h)
            for dy in range(step):
                for dx in range(step):
                    xi = min(img_w - 1, x0 + dx)
                    yi = min(img_h - 1, y0 + dy)
                    px[xi, yi] = col

    draw = ImageDraw.Draw(img)
    total_hits = 0
    route_stats = []

    for ri, route in enumerate(routes):
        wps = route.get("waypoints", [])
        if len(wps) < 2:
            continue
        col = ROUTE_COLORS[ri % len(ROUTE_COLORS)]
        rid = route.get("id", f"route_{ri}")
        hits: list[tuple[float, float]] = []
        pts = []
        for wp in wps:
            pts.append(world_to_px(wp["x"], wp["y"], bathy, img_w, img_h))
        for i in range(1, len(wps)):
            a, b = wps[i - 1], wps[i]
            hits.extend(sample_segment(bathy, a["x"], a["y"], b["x"], b["y"]))
        mode = route.get("mode", "pingpong")
        if mode == "pingpong" and len(wps) >= 2:
            a, b = wps[-1], wps[-2]
            hits.extend(sample_segment(bathy, a["x"], a["y"], b["x"], b["y"]))
        if len(pts) >= 2:
            draw.line(pts, fill=col, width=2)
        for px_, py_ in pts[:: max(1, len(pts) // 8)]:
            draw.ellipse((px_ - 3, py_ - 3, px_ + 3, py_ + 3), outline=col, width=1)
        for hx, hy in hits:
            sx, sy = world_to_px(hx, hy, bathy, img_w, img_h)
            draw.rectangle((sx - 4, sy - 4, sx + 4, sy + 4), fill=(255, 40, 40), outline=(255, 200, 200))
        total_hits += len(hits)
        route_stats.append({"id": rid, "hits": len(hits), "wps": len(wps)})

    # legend block
    try:
        font = ImageFont.load_default()
    except Exception:
        font = None
    land = sum(1 for d in bathy.depths if d <= 0)
    land_pct = 100.0 * land / len(bathy.depths)
    header = f"{theater_id} — {title}"
    sub = f"{subtitle} | land {land_pct:.1f}% | routes {len(routes)} | land hits {total_hits}"
    draw.rectangle((0, 0, img_w, 36), fill=(20, 24, 30))
    draw.text((8, 4), header, fill=(220, 230, 240), font=font)
    draw.text((8, 18), sub, fill=(140, 160, 180), font=font)

    out_path.parent.mkdir(parents=True, exist_ok=True)
    img.save(out_path, format="PNG", optimize=True)
    return {"theater": theater_id, "land_pct": land_pct, "hits": total_hits, "routes": route_stats, "path": str(out_path)}


def load_bathy_from_theater(th: dict) -> Bathy:
    b64 = th["bathy"]["data_b64"]
    return Bathy(base64.b64decode(b64))


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--scenario", default=str(ROOT / "scenarios_generated" / "taiwan_formosa_watch.json"))
    args = ap.parse_args()
    scenario = Path(args.scenario)
    with scenario.open(encoding="utf-8") as f:
        doc = json.load(f)

    theaters = {th["id"]: th for th in doc.get("theaters", [])}
    zone_meta = {
        "taiwan_east": ("East Coast Offshore", "простой берег"),
        "taiwan_strait": ("Taiwan Strait", "два берега, пролив"),
        "taiwan_penghu": ("Penghu Archipelago", "острова"),
        "taiwan_south": ("Bashi Approach", "мыс / канал"),
        "taiwan_north": ("NE Coast / Yilan", "заливы"),
        "taiwan_lanyu": ("Orchid Island", "одиночный остров"),
    }

    manifest = []
    for mission in doc.get("missions", []):
        mid = mission["id"]
        tid = mission.get("theater_id", "taiwan_east")
        th = theaters.get(tid)
        if not th:
            print(f"skip {mid}: no theater {tid}")
            continue
        bathy = load_bathy_from_theater(th)
        title_en, coast = zone_meta.get(tid, (tid, ""))
        mtitle = mission.get("title", {}).get("en", mid)
        out = OUT / f"{tid}__{mid}.png"
        stats = render_theater(
            tid,
            bathy,
            mission.get("routes", []),
            f"{mtitle}",
            f"{title_en} — {coast}",
            out,
        )
        stats["mission"] = mid
        manifest.append(stats)
        print(f"{out.name}: land={stats['land_pct']:.1f}% route_land_hits={stats['hits']}")

    # one overview per unique theater (first mission using it)
    seen = set()
    for mission in doc.get("missions", []):
        tid = mission.get("theater_id")
        if tid in seen:
            continue
        seen.add(tid)
        th = theaters[tid]
        bathy = load_bathy_from_theater(th)
        title_en, coast = zone_meta.get(tid, (tid, ""))
        out = OUT / f"{tid}__overview.png"
        stats = render_theater(tid, bathy, mission.get("routes", []), title_en, coast, out)
        stats["mission"] = mission["id"] + " (overview)"
        manifest.append(stats)

    idx = OUT / "manifest.json"
    idx.write_text(json.dumps(manifest, indent=2), encoding="utf-8")
    print(f"wrote {len(manifest)} previews -> {OUT}")


if __name__ == "__main__":
    main()
