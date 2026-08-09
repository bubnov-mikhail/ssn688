# Peri ship sprites (IR)

Grayscale orthographic silhouettes for the MAST periscope IR optic.
Aspects `0..90` step `5` → `{class}_{aspect:03d}.png` (512×160).

Rendered with **EEVEE emission** (supersampled). Styles per class in
`SHIP_CFG['ir_style']`:
- `detail` — Workbench studio + cavity (all classes)
- `zones` — flat hull/deck/super/engine/stack/radar bands (legacy)

Orientation / waterline are **locked** in `SHIP_CFG` + `ORIENT`.
Runtime flips via `PeriShipProj.BowRight`.

## Sources (`~/ssn688/tools/vendor/peri_models_meshy/`)

| Class     | File              |
|-----------|-------------------|
| merchant  | `Cargoship.blend` |
| tanker    | `oil_tanker.glb`  |
| fishing   | `fishing.obj`     |
| combatant | `combantant.glb`  |

Re-render:

```bash
/Applications/Blender.app/Contents/MacOS/Blender --background --python \
  tools/gen_peri_ship_sprites_blender.py
```
