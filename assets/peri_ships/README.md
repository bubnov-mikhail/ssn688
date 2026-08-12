# Peri ship sprites (IR)

Grayscale orthographic silhouettes for the MAST periscope IR optic.
Aspects `0..180` step `1` → `{class}_{aspect:03d}.png` (512×160).
`0` = bow-on, `90` = beam, `180` = stern-on.

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

Re-render all aspects:

```bash
/Applications/Blender.app/Contents/MacOS/Blender --background --python \
  tools/gen_peri_ship_sprites_blender.py
```

Fill missing 1° frames only (keep existing):

```bash
/Applications/Blender.app/Contents/MacOS/Blender --background --python \
  tools/gen_peri_ship_sprites_blender.py -- --skip-existing
```

Stern quarters only (`95..180`):

```bash
/Applications/Blender.app/Contents/MacOS/Blender --background --python \
  tools/gen_peri_ship_sprites_blender.py -- --stern
```
