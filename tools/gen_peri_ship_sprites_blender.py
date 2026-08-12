#!/usr/bin/env python3
"""Generate peri IR ship sprites from Meshy models (Blender 4/5).

Usage (repo root):
  /Applications/Blender.app/Contents/MacOS/Blender --background --python \\
      tools/gen_peri_ship_sprites_blender.py

Outputs: assets/peri_ships/{class}_{aspect:03d}.png
  aspect 0..180 step 1 (bow-on → beam → stern-on).

IR: two styles via SHIP_CFG['ir_style']:
  zones  — flat EEVEE emission by hull/deck/super/engine/stack/radar
  detail — Workbench studio + cavity (portholes, masts, panel seams)
GLB/Y-up models get Rx(-90) so the camera sees a side elevation (like fishing).
"""

from __future__ import annotations

import math
import re
import sys
from pathlib import Path

import bmesh
import bpy
from mathutils import Matrix, Vector

HOME = Path.home()
MODELS_DIR = HOME / "ssn688" / "tools" / "vendor" / "peri_models_meshy"
_SCRIPT = Path(__file__).resolve() if "__file__" in dir() else None
if _SCRIPT and (_SCRIPT.parent.parent / "assets" / "peri_ships").is_dir():
    REPO = _SCRIPT.parent.parent
else:
    REPO = Path.cwd()
OUT_DIR = REPO / "assets" / "peri_ships"

MODEL_FILES = {
    "merchant": "Cargoship.blend",
    "tanker": "oil_tanker.glb",
    "fishing": "fishing.obj",
    "combatant": "combantant.glb",
}

# LOCKED framing (do not retune without an explicit ask):
#   orientation = ORIENT + yup/yaw180 below; waterline_z / frame_top_z as listed.
# After parent_clear / Y-up→Z-up, waterline/frame are in keel-up units.
SHIP_CFG = {
    # Cargoship.blend: Z-up after clear; length on Y → ORIENT y90.
    "merchant": {
        "loa": 19.0,
        "waterline_z": 1.70,
        "frame_top_z": 5.5,
        "yup": False,
        "yaw180": False,
        "join": True,
        "ir_style": "detail",
    },
    # oil_tanker.glb: pitch_m90 + yup; waterline just above rudder/prop.
    "tanker": {
        "loa": 30.0,
        "waterline_z": 1.20,
        "frame_top_z": 7.8,
        "yup": True,
        "yaw180": True,
        "join": False,
        "ir_style": "detail",
    },
    # fishing.obj: already Z-up.
    "fishing": {
        "loa": 2.8,
        "waterline_z": 0.20,
        "frame_top_z": 1.35,
        "yup": False,
        "yaw180": False,
        "join": False,
        "ir_style": "detail",
    },
    # combantant.glb: Z-up after parent_clear (do NOT yup); length already on X.
    "combatant": {
        "loa": 14.0,
        "waterline_z": 0.85,
        "frame_top_z": 4.2,
        "yup": False,
        "yaw180": False,
        "join": False,
        "ir_style": "detail",
    },
}

ASPECT_STEP = 1
ASPECT_MIN = 0
ASPECT_MAX = 180
RES_X, RES_Y = 512, 160
# Render larger then downsample for softer IR edges.
SUPERSAMPLE = 3

IR_LEVEL = {
    "hull": 0.16,
    "deck": 0.28,
    "super": 0.42,
    "engine": 0.82,
    "stack": 0.95,
    "radar": 1.00,
}

MAT_ORDER = ("hull", "deck", "super", "engine", "stack", "radar")

NAME_RADAR = re.compile(r"radar|radome|antenna|satcom|phased|sensor.?dome", re.I)
NAME_STACK = re.compile(r"stack|funnel|chimney|exhaust|smokestack", re.I)
NAME_ENGINE = re.compile(r"engine.?room|machinery|boiler", re.I)

ORIENT = {
    "merchant": "y90",  # after clear: length on Y → bring to +X
    "tanker": "pitch_m90",
    "fishing": "identity",
    "combatant": "identity",
}


def clear_scene() -> None:
    bpy.ops.object.select_all(action="SELECT")
    bpy.ops.object.delete(use_global=False)
    for block in (bpy.data.meshes, bpy.data.cameras, bpy.data.lights, bpy.data.materials):
        for b in list(block):
            if b.users == 0:
                block.remove(b)


def link(obj: bpy.types.Object) -> None:
    col = bpy.context.scene.collection
    if obj.name not in col.objects:
        col.objects.link(obj)


def import_model(path: Path) -> list[bpy.types.Object]:
    before = set(bpy.data.objects)
    suf = path.suffix.lower()
    if suf == ".blend":
        with bpy.data.libraries.load(str(path), link=False) as (data_from, data_to):
            data_to.objects = list(data_from.objects)
        for obj in data_to.objects:
            if obj is not None:
                link(obj)
    elif suf in (".glb", ".gltf"):
        bpy.ops.import_scene.gltf(filepath=str(path))
    elif suf == ".obj":
        if hasattr(bpy.ops.wm, "obj_import"):
            bpy.ops.wm.obj_import(filepath=str(path))
        else:
            bpy.ops.import_scene.obj(filepath=str(path))
    else:
        raise RuntimeError(f"unsupported model: {path}")
    return [o for o in bpy.data.objects if o not in before]


def mesh_objects(objs: list[bpy.types.Object]) -> list[bpy.types.Object]:
    out = []
    for o in objs:
        if o.type == "MESH":
            out.append(o)
        out.extend(c for c in o.children_recursive if c.type == "MESH")
    seen, uniq = set(), []
    for o in out:
        if o.name not in seen:
            seen.add(o.name)
            uniq.append(o)
    return uniq


def world_bbox(objs: list[bpy.types.Object]) -> tuple[Vector, Vector]:
    pts = []
    for o in objs:
        for c in o.bound_box:
            pts.append(o.matrix_world @ Vector(c))
    xs = [p.x for p in pts]
    ys = [p.y for p in pts]
    zs = [p.z for p in pts]
    return Vector((min(xs), min(ys), min(zs))), Vector((max(xs), max(ys), max(zs)))


def ensure_ir_materials() -> dict[str, bpy.types.Material]:
    """EEVEE emission materials — soft flat IR gray, no specular edges."""
    mats = {}
    for name in MAT_ORDER:
        g = IR_LEVEL[name]
        m = bpy.data.materials.get(f"IR_{name}") or bpy.data.materials.new(f"IR_{name}")
        m.use_nodes = True
        nt = m.node_tree
        for n in list(nt.nodes):
            nt.nodes.remove(n)
        out = nt.nodes.new("ShaderNodeOutputMaterial")
        em = nt.nodes.new("ShaderNodeEmission")
        em.inputs["Color"].default_value = (g, g, g, 1.0)
        em.inputs["Strength"].default_value = 1.0
        nt.links.new(em.outputs["Emission"], out.inputs["Surface"])
        m.diffuse_color = (g, g, g, 1.0)
        mats[name] = m
    return mats


def name_ir_bias(obj_name: str) -> str | None:
    if NAME_RADAR.search(obj_name):
        return "radar"
    if NAME_STACK.search(obj_name):
        return "stack"
    if NAME_ENGINE.search(obj_name):
        return "engine"
    return None


def ir_class_for_point(u: float, h: float, name_bias: str | None = None) -> str:
    if name_bias:
        return name_bias
    sternish = u < 0.35 or u > 0.65
    if h > 0.88:
        return "radar"
    if h > 0.78 and 0.25 < u < 0.75:
        return "radar"
    if 0.58 < h < 0.82 and (0.55 < u < 0.82 or 0.15 < u < 0.38):
        return "stack"
    if sternish and 0.05 < h < 0.38:
        return "engine"
    if h > 0.55:
        return "super"
    if h > 0.22:
        return "deck"
    return "hull"


def assign_ir_face_materials(
    meshes: list[bpy.types.Object],
    mats: dict[str, bpy.types.Material],
    frame_top_z: float,
) -> dict[str, int]:
    """Per-face IR emission materials (no mesh split — EEVEE reads slots)."""
    if not meshes:
        return {}
    mn, mx = world_bbox(meshes)
    span = mx - mn
    span.x = max(span.x, 1e-4)
    span.y = max(span.y, 1e-4)
    span.z = max(span.z, 1e-4)
    z_lo = max(0.0, mn.z)
    z_hi = min(max(mx.z, z_lo + 0.5), z_lo + max(frame_top_z, 0.5))
    z_span = max(z_hi - z_lo, 1e-4)
    len_axis = "x" if span.x >= span.y else "y"
    counts: dict[str, int] = {}

    for o in meshes:
        bias = name_ir_bias(o.name)
        me = o.data
        me.materials.clear()
        slot_index = {key: i for i, key in enumerate(MAT_ORDER)}
        for key in MAT_ORDER:
            me.materials.append(mats[key])

        bm = bmesh.new()
        bm.from_mesh(me)
        mw = o.matrix_world
        for f in bm.faces:
            c = mw @ f.calc_center_median()
            if len_axis == "x":
                u = (c.x - mn.x) / span.x
            else:
                u = (c.y - mn.y) / span.y
            h = max(0.0, min(1.5, (c.z - z_lo) / z_span))
            cls = ir_class_for_point(u, h, bias)
            if h > 1.05:
                cls = "radar"
            f.material_index = slot_index[cls]
            counts[cls] = counts.get(cls, 0) + 1
        bm.to_mesh(me)
        bm.free()
    return counts


def visible_frame_bbox(
    meshes: list[bpy.types.Object], frame_top_z: float
) -> tuple[Vector, Vector]:
    """Frame above-water silhouette. Use true X min/max — percentile trim on
    dense container meshes collapses to midship and crops bow/stern."""
    vis = [m for m in meshes if not m.hide_render]
    if not vis:
        return Vector((0, 0, 0)), Vector((1, 1, 1))
    mn, mx = world_bbox(vis)
    xs = []
    for o in vis:
        mw = o.matrix_world
        # Dense cargo meshes: sample enough verts to hit bow/stern extremities.
        nverts = len(o.data.vertices)
        step = max(1, nverts // 8000)
        for i, v in enumerate(o.data.vertices):
            if i % step:
                continue
            p = mw @ v.co
            if p.z < -0.05:
                continue
            xs.append(p.x)
    if xs:
        lo, hi = min(xs), max(xs)
    else:
        lo, hi = mn.x, mx.x
    return (
        Vector((lo, mn.y, max(0.0, mn.z))),
        Vector((hi, mx.y, min(frame_top_z, max(mx.z, 0.5)))),
    )


def setup_world_black() -> None:
    world = bpy.data.worlds.get("World") or bpy.data.worlds.new("World")
    bpy.context.scene.world = world
    world.use_nodes = True
    nt = world.node_tree
    for n in list(nt.nodes):
        nt.nodes.remove(n)
    out = nt.nodes.new("ShaderNodeOutputWorld")
    bg = nt.nodes.new("ShaderNodeBackground")
    bg.inputs["Color"].default_value = (0, 0, 0, 1)
    bg.inputs["Strength"].default_value = 0.0
    nt.links.new(bg.outputs["Background"], out.inputs["Surface"])
    if hasattr(world, "color"):
        world.color = (0, 0, 0)


def setup_render(ir_style: str = "zones") -> None:
    """zones: EEVEE flat emission. detail: Workbench studio+cavity (mesh relief)."""
    scene = bpy.context.scene
    scene.render.resolution_x = RES_X * SUPERSAMPLE
    scene.render.resolution_y = RES_Y * SUPERSAMPLE
    scene.render.resolution_percentage = 100
    scene.render.image_settings.file_format = "PNG"
    scene.render.image_settings.color_mode = "RGB"
    scene.render.image_settings.color_depth = "8"
    scene.render.film_transparent = False

    if ir_style == "detail":
        # Matches the earlier combatant look: readable masts/ports/panels via cavity.
        scene.render.engine = "BLENDER_WORKBENCH"
        shading = scene.display.shading
        shading.light = "STUDIO"
        shading.color_type = "TEXTURE"
        shading.show_cavity = True
        shading.cavity_type = "BOTH"
        if hasattr(shading, "cavity_ridge_factor"):
            shading.cavity_ridge_factor = 2.25
        if hasattr(shading, "cavity_valley_factor"):
            shading.cavity_valley_factor = 2.75
        if hasattr(shading, "curvature_ridge_factor"):
            shading.curvature_ridge_factor = 1.25
        if hasattr(shading, "curvature_valley_factor"):
            shading.curvature_valley_factor = 1.5
        if hasattr(shading, "show_object_outline"):
            shading.show_object_outline = False
        if hasattr(shading, "show_shadows"):
            shading.show_shadows = False
        if hasattr(scene.render, "filter_size"):
            scene.render.filter_size = 1.5
    else:
        try:
            scene.render.engine = "BLENDER_EEVEE_NEXT"
        except TypeError:
            scene.render.engine = "BLENDER_EEVEE"
        if hasattr(scene.render, "filter_size"):
            scene.render.filter_size = 2.0
        ee = getattr(scene, "eevee", None)
        if ee is not None and hasattr(ee, "taa_render_samples"):
            ee.taa_render_samples = 128

    if hasattr(scene, "view_settings"):
        scene.view_settings.view_transform = "Standard"
        scene.view_settings.look = "None"
        scene.view_settings.exposure = 0.0
        scene.view_settings.gamma = 1.0


def ensure_camera() -> bpy.types.Object:
    cam_data = bpy.data.cameras.new("PeriCam")
    cam_data.type = "ORTHO"
    cam = bpy.data.objects.new("PeriCam", cam_data)
    link(cam)
    bpy.context.scene.camera = cam
    return cam


def unparent_keep_transform(objs: list[bpy.types.Object]) -> None:
    """Clear parents without jumping — keep each object's world pose."""
    real = [o for o in objs if o is not None]
    if not real:
        return
    bpy.ops.object.select_all(action="DESELECT")
    for o in real:
        o.hide_set(False)
        o.hide_viewport = False
        o.hide_render = False
        o.select_set(True)
    bpy.context.view_layer.objects.active = real[0]
    bpy.ops.object.parent_clear(type="CLEAR_KEEP_TRANSFORM")
    bpy.context.view_layer.update()


def apply_object_transforms(meshes: list[bpy.types.Object]) -> None:
    """Bake location/rotation/scale into mesh data (identity object matrices)."""
    if not meshes:
        return
    bpy.ops.object.select_all(action="DESELECT")
    for o in meshes:
        o.hide_set(False)
        o.hide_viewport = False
        o.hide_render = False
        o.select_set(True)
    bpy.context.view_layer.objects.active = meshes[0]
    bpy.ops.object.transform_apply(location=True, rotation=True, scale=True)
    bpy.context.view_layer.update()


def join_meshes(meshes: list[bpy.types.Object]) -> list[bpy.types.Object]:
    """Join into one mesh so the ship yaws as a single rigid body."""
    real = [m for m in meshes if m.type == "MESH" and m.data is not None]
    if len(real) <= 1:
        return real
    bpy.ops.object.select_all(action="DESELECT")
    for o in real:
        o.hide_set(False)
        o.hide_viewport = False
        o.hide_render = False
        o.select_set(True)
    bpy.context.view_layer.objects.active = real[0]
    bpy.ops.object.join()
    joined = bpy.context.view_layer.objects.active
    return [joined] if joined is not None else real


def make_rigid_ship(meshes: list[bpy.types.Object], join: bool) -> list[bpy.types.Object]:
    """Import cleanup: clear parents (keep pose) → join into one rigid mesh.

    Do not transform_apply before join — on Cargoship.blend it collapses the
    hierarchy. parent_clear(KEEP_TRANSFORM) + join preserves the assembled ship.
    """
    unparent_keep_transform(meshes)
    if join or len(meshes) > 1:
        meshes = join_meshes(meshes)
        nverts = len(meshes[0].data.vertices) if meshes else 0
        print(f"rigid unit → {len(meshes)} mesh(es), {nverts} verts")
    return meshes


def apply_world_matrix(meshes: list[bpy.types.Object], mat: Matrix) -> None:
    for o in meshes:
        o.matrix_world = mat @ o.matrix_world
    bpy.context.view_layer.update()


def normalize_ship(meshes: list[bpy.types.Object], loa: float) -> None:
    unparent_keep_transform(meshes)
    mn, mx = world_bbox(meshes)
    span = mx - mn
    cur_loa = max(span.x, span.y, 1e-4)
    s = loa / cur_loa
    cx = 0.5 * (mn.x + mx.x)
    cy = 0.5 * (mn.y + mx.y)
    m = Matrix.Scale(s, 4) @ Matrix.Translation((-cx, -cy, -mn.z))
    apply_world_matrix(meshes, m)
    mn, mx = world_bbox(meshes)
    apply_world_matrix(meshes, Matrix.Translation((-0.5 * (mn.x + mx.x), -0.5 * (mn.y + mx.y), -mn.z)))


def orient_length_along_x(meshes: list[bpy.types.Object], class_name: str) -> None:
    """Put ship length on +X; convert Y-up assets to Z-up side-elevation rest pose."""
    unparent_keep_transform(meshes)

    kind = ORIENT.get(class_name, "identity")
    if kind == "y90":
        apply_world_matrix(meshes, Matrix.Rotation(math.pi / 2, 4, "Z"))
    elif kind == "pitch_m90":
        apply_world_matrix(meshes, Matrix.Rotation(-math.pi / 2, 4, "Y"))
        mn, mx = world_bbox(meshes)
        if (mx.y - mn.y) > (mx.x - mn.x):
            apply_world_matrix(meshes, Matrix.Rotation(math.pi / 2, 4, "Z"))
    elif kind != "identity":
        raise RuntimeError(f"unknown ORIENT {kind}")

    mn, mx = world_bbox(meshes)
    if (mx.z - mn.z) > (mx.x - mn.x) * 1.05:
        apply_world_matrix(meshes, Matrix.Rotation(-math.pi / 2, 4, "Y"))
        mn, mx = world_bbox(meshes)
        if (mx.y - mn.y) > (mx.x - mn.x):
            apply_world_matrix(meshes, Matrix.Rotation(math.pi / 2, 4, "Z"))
        mn, mx = world_bbox(meshes)

    apply_world_matrix(
        meshes,
        Matrix.Translation((-0.5 * (mn.x + mx.x), -0.5 * (mn.y + mx.y), -mn.z)),
    )

    cfg = SHIP_CFG[class_name]
    # Y-up (deck toward ±Y) → looking from −Y is top-down. Tip to Z-up like fishing.
    if cfg.get("yup"):
        apply_world_matrix(meshes, Matrix.Rotation(-math.pi / 2, 4, "X"))
        mn, mx = world_bbox(meshes)
        apply_world_matrix(
            meshes,
            Matrix.Translation((-0.5 * (mn.x + mx.x), -0.5 * (mn.y + mx.y), -mn.z)),
        )
    if cfg.get("yaw180"):
        apply_world_matrix(meshes, Matrix.Rotation(math.pi, 4, "Z"))
        mn, mx = world_bbox(meshes)
        apply_world_matrix(
            meshes,
            Matrix.Translation((-0.5 * (mn.x + mx.x), -0.5 * (mn.y + mx.y), -mn.z)),
        )
    print(f"orient {class_name}={kind} yup={cfg.get('yup')} bbox={world_bbox(meshes)}")


def set_waterline(meshes: list[bpy.types.Object], waterline_z: float) -> None:
    apply_world_matrix(meshes, Matrix.Translation((0, 0, -waterline_z)))
    for o in meshes:
        omn, omx = world_bbox([o])
        if omx.z < -0.02:
            o.hide_render = True
            o.hide_viewport = True


def rotate_aspect(meshes: list[bpy.types.Object], aspect_deg: float) -> None:
    """rot_z = 90 - aspect  (0=bow-on, 90=beam, 180=stern-on)."""
    apply_world_matrix(meshes, Matrix.Rotation(math.radians(90.0 - aspect_deg), 4, "Z"))


def compute_fixed_ortho(meshes: list[bpy.types.Object], frame_top_z: float) -> float:
    """Ortho width that fits beam (full LOA) + air draft. Same for every aspect so
    meters/pixel (and waterline) stay stable as the ship yaws."""
    mn, mx = visible_frame_bbox(meshes, frame_top_z)
    half_w = 0.5 * abs(mx.x - mn.x)
    pad_x = half_w * 0.18 + 0.20
    pad_top = frame_top_z * 0.10 + 0.08
    view_h = frame_top_z + pad_top
    aspect = RES_X / float(RES_Y)
    return max(view_h * aspect, 2.0 * (half_w + pad_x))


def frame_camera(
    cam: bpy.types.Object,
    meshes: list[bpy.types.Object],
    frame_top_z: float,
    ortho_scale: float,
) -> None:
    """Center on current aspect; keep fixed ortho with z=0 on the bottom edge."""
    mn, mx = visible_frame_bbox(meshes, frame_top_z)
    mid_x = 0.5 * (mn.x + mx.x)
    aspect = RES_X / float(RES_Y)
    # HORIZONTAL fit: ortho_scale is the visible world width.
    if hasattr(cam.data, "sensor_fit"):
        cam.data.sensor_fit = "HORIZONTAL"
    cam.data.ortho_scale = ortho_scale
    # Bottom of view at z=0 (waterline); top at ortho/aspect.
    half_v = ortho_scale / (2.0 * aspect)
    mid_z = half_v

    dist = max(abs(mx.y - mn.y), 1.0) + max(ortho_scale, 1.0) * 2.5
    target = Vector((mid_x, 0.0, mid_z))
    cam.location = Vector((mid_x, mn.y - dist, mid_z))
    cam.rotation_euler = (target - cam.location).to_track_quat("-Z", "Z").to_euler()
    bpy.context.view_layer.update()
    cam.data.clip_start = 0.05
    cam.data.clip_end = max(500.0, dist * 3)


def crush_background(path: Path, thresh: float = 0.08) -> None:
    """Crush near-black to pure black. Keep low thresh so dark details survive."""
    img = bpy.data.images.load(str(path))
    px = list(img.pixels)
    for i in range(0, len(px), 4):
        y = (px[i] + px[i + 1] + px[i + 2]) / 3.0
        if y < thresh:
            px[i] = px[i + 1] = px[i + 2] = 0.0
            px[i + 3] = 1.0
        else:
            # Force grayscale IR.
            px[i] = px[i + 1] = px[i + 2] = y
            px[i + 3] = 1.0
    img.pixels[:] = px
    img.filepath_raw = str(path)
    img.file_format = "PNG"
    img.save()
    bpy.data.images.remove(img)


def downsample_png(
    path: Path,
    src_w: int,
    src_h: int,
    dst_w: int,
    dst_h: int,
    soft_blur: bool = True,
) -> None:
    """Box-filter supersampled render; optional light blur for softer IR edges."""
    img = bpy.data.images.load(str(path))
    pixels = list(img.pixels)
    if len(pixels) != src_w * src_h * 4:
        img.scale(dst_w, dst_h)
        img.filepath_raw = str(path)
        img.file_format = "PNG"
        img.save()
        bpy.data.images.remove(img)
        return

    sx = src_w / dst_w
    sy = src_h / dst_h
    out = [0.0] * (dst_w * dst_h * 4)
    for y in range(dst_h):
        for x in range(dst_w):
            x0 = int(x * sx)
            y0 = int(y * sy)
            x1 = max(x0 + 1, int((x + 1) * sx))
            y1 = max(y0 + 1, int((y + 1) * sy))
            r = g = b = 0.0
            n = 0
            for yy in range(y0, y1):
                for xx in range(x0, x1):
                    i = (yy * src_w + xx) * 4
                    r += pixels[i]
                    g += pixels[i + 1]
                    b += pixels[i + 2]
                    n += 1
            j = (y * dst_w + x) * 4
            out[j] = r / n
            out[j + 1] = g / n
            out[j + 2] = b / n
            out[j + 3] = 1.0

    final = out
    if soft_blur:
        blurred = out[:]
        for y in range(dst_h):
            for x in range(dst_w):
                acc = [0.0, 0.0, 0.0]
                wsum = 0.0
                for dy in (-1, 0, 1):
                    for dx in (-1, 0, 1):
                        xx, yy = x + dx, y + dy
                        if xx < 0 or yy < 0 or xx >= dst_w or yy >= dst_h:
                            continue
                        w = 2.0 if dx == 0 and dy == 0 else 1.0
                        j = (yy * dst_w + xx) * 4
                        acc[0] += out[j] * w
                        acc[1] += out[j + 1] * w
                        acc[2] += out[j + 2] * w
                        wsum += w
                j = (y * dst_w + x) * 4
                if out[j] + out[j + 1] + out[j + 2] < 0.04:
                    continue
                blurred[j] = acc[0] / wsum
                blurred[j + 1] = acc[1] / wsum
                blurred[j + 2] = acc[2] / wsum
        final = blurred

    img.scale(dst_w, dst_h)
    img.pixels[:] = final
    img.filepath_raw = str(path)
    img.file_format = "PNG"
    img.save()
    bpy.data.images.remove(img)


def render_aspects(
    class_name: str,
    base_meshes: list[bpy.types.Object],
    cam: bpy.types.Object,
    frame_top_z: float,
    soft_blur: bool = True,
    crush_thresh: float = 0.025,
    aspect_min: int = ASPECT_MIN,
    aspect_max: int = ASPECT_MAX,
    skip_existing: bool = False,
) -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    rest = {o.name: o.matrix_world.copy() for o in base_meshes}
    # Rest pose is beam (length on X) — fix zoom from that so yaw doesn't rezoom.
    vis0 = [m for m in base_meshes if not m.hide_render]
    ortho = compute_fixed_ortho(vis0, frame_top_z)
    print(f"fixed_ortho {class_name}={ortho:.3f} aspects={aspect_min}..{aspect_max}")
    src_w, src_h = RES_X * SUPERSAMPLE, RES_Y * SUPERSAMPLE
    for aspect in range(aspect_min, aspect_max + 1, ASPECT_STEP):
        out = OUT_DIR / f"{class_name}_{aspect:03d}.png"
        if skip_existing and out.is_file():
            print(f"skip existing {out.name}")
            continue
        for o in base_meshes:
            o.matrix_world = rest[o.name].copy()
        bpy.context.view_layer.update()
        rotate_aspect(base_meshes, aspect)
        vis = [m for m in base_meshes if not m.hide_render]
        frame_camera(cam, vis, frame_top_z, ortho)
        bpy.context.scene.render.filepath = str(out)
        bpy.ops.render.render(write_still=True)
        if SUPERSAMPLE > 1:
            downsample_png(out, src_w, src_h, RES_X, RES_Y, soft_blur=soft_blur)
        crush_background(out, thresh=crush_thresh)
        print(f"wrote {out}")


def process_class(
    class_name: str,
    aspect_min: int = ASPECT_MIN,
    aspect_max: int = ASPECT_MAX,
    skip_existing: bool = False,
) -> None:
    clear_scene()
    setup_world_black()
    cfg = SHIP_CFG[class_name]
    ir_style = cfg.get("ir_style", "zones")
    setup_render(ir_style)
    cam = ensure_camera()
    path = MODELS_DIR / MODEL_FILES[class_name]
    if not path.is_file():
        print(f"MISSING {path}")
        return
    print(f"=== {class_name} ← {path.name} ir_style={ir_style} ===")
    imported = import_model(path)
    meshes = mesh_objects(imported)
    if not meshes:
        print(f"no meshes for {class_name}")
        return
    # Fresh rigid body: unparent (keep pose) → join, then yaw as one mesh.
    meshes = make_rigid_ship(meshes, join=bool(cfg.get("join")) or len(meshes) > 1)
    orient_length_along_x(meshes, class_name)
    normalize_ship(meshes, cfg["loa"])
    set_waterline(meshes, cfg["waterline_z"])
    visible = [m for m in meshes if not m.hide_render]

    if ir_style == "detail":
        # Keep original albedo/textures; Workbench cavity carries the relief.
        print("detail IR: Workbench studio+cavity (no flat zone paint)")
        soft_blur = False
        crush_thresh = 0.018
    else:
        mats = ensure_ir_materials()
        counts = assign_ir_face_materials(visible, mats, cfg["frame_top_z"])
        print("IR faces", counts, "meshes", len(visible))
        soft_blur = True
        crush_thresh = 0.025

    print("bbox", world_bbox(visible), "frame", visible_frame_bbox(visible, cfg["frame_top_z"]))
    render_aspects(
        class_name,
        visible,
        cam,
        cfg["frame_top_z"],
        soft_blur=soft_blur,
        crush_thresh=crush_thresh,
        aspect_min=aspect_min,
        aspect_max=aspect_max,
        skip_existing=skip_existing,
    )


def main() -> None:
    print(f"MODELS_DIR={MODELS_DIR} exists={MODELS_DIR.is_dir()}")
    print(f"OUT_DIR={OUT_DIR}")
    if not MODELS_DIR.is_dir():
        print("Models directory missing — abort")
        sys.exit(1)
    only = []
    aspect_min = ASPECT_MIN
    aspect_max = ASPECT_MAX
    skip_existing = False
    if "--" in sys.argv:
        after = sys.argv[sys.argv.index("--") + 1 :]
        only = [a for a in after if a in MODEL_FILES]
        if "--stern" in after:
            # Missing quarters: beam→stern (95..180). Bow→beam already on disk.
            aspect_min = 95
            aspect_max = 180
        if "--skip-existing" in after:
            skip_existing = True
        for i, a in enumerate(after):
            if a == "--from" and i + 1 < len(after):
                aspect_min = int(after[i + 1])
            if a == "--to" and i + 1 < len(after):
                aspect_max = int(after[i + 1])
    classes = only or ["merchant", "tanker", "fishing", "combatant"]
    for cls in classes:
        process_class(cls, aspect_min=aspect_min, aspect_max=aspect_max, skip_existing=skip_existing)
    print("done")


if __name__ == "__main__":
    main()
