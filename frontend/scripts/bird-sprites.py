"""
Generates the pairing magpie layers in src/assets/photos/bird from magpie-pairing.webp.

  python3 scripts/bird-sprites.py        (needs Pillow and numpy)

Output: body.webp (still), and wing-up / wing-right / tail sprite sheets (FRAMES poses side by side;
frame 0 is the unmodified artwork). A wing "bends": the rotation is zero at the shoulder (hidden under
the torso) and grows towards the tip, so the root never detaches from the body.
"""
import os
import numpy as np
from PIL import Image, ImageChops, ImageDraw

SRC = 'src/assets/photos/magpie-pairing.webp'
OUT = 'src/assets/photos/bird'
W, H, FRAMES = 280, 281, 8
# Peak tip rotation in degrees on the up/down stroke, and upstroke foreshortening.
WING = dict(raise_deg=3.0, lower_deg=2.0, shrink=0.008)
TAIL = dict(raise_deg=3.5, lower_deg=3.5, shrink=0.0)

os.makedirs(OUT, exist_ok=True)
im = Image.open(SRC).convert('RGBA').resize((W, H), Image.LANCZOS)
k = W / 720
r_, g_, b_, a_ = im.split()
blue = ImageChops.subtract(b_, r_).point(lambda v: 255 if v > 45 else 0)
opaque = a_.point(lambda v: 255 if v > 0 else 0)
blue = ImageChops.multiply(blue, opaque)
body = ImageChops.subtract(opaque, blue)


def rect(x0, y0, x1, y1):
    m = Image.new('L', (W, H), 0)
    ImageDraw.Draw(m).rectangle([x0 * k, y0 * k, x1 * k, y1 * k], fill=255)
    return m


def disc(cx, cy, r):
    m = Image.new('L', (W, H), 0)
    ImageDraw.Draw(m).ellipse([(cx - r) * k, (cy - r) * k, (cx + r) * k, (cy + r) * k], fill=255)
    return m


M = 12  # overlap so cut seams never open
tail_r = rect(0, 390 - M, 322 + M, 720)
right_r = ImageChops.subtract(rect(322 - M, 310 - M, 720, 720), rect(0, 390, 322 - M, 720))
up_r = ImageChops.lighter(rect(0, 0, 322 + M, 390 + M), rect(322 - M, 0, 720, 310 + M))
torso = ImageChops.subtract(body, rect(420, 150, 650, 275))
solid = Image.new('RGBA', (W, H), (44, 94, 150, 255))
art = Image.composite(im, solid, blue)


def layer(mask, fill):
    m = ImageChops.multiply(blue, mask)
    if fill is not None:  # blue under the torso, so the root stays covered while the wing bends
        m = ImageChops.lighter(m, ImageChops.multiply(ImageChops.multiply(torso, mask), fill))
    out = art.copy()
    out.putalpha(ImageChops.multiply(m, ImageChops.lighter(a_, torso)))
    return out


def warp(img, pivot, length, f, raise_deg, lower_deg, shrink, r0, p):
    a = np.asarray(img, dtype=np.float32) / 255.0
    a[..., :3] *= a[..., 3:4]
    ys, xs = np.mgrid[0:H, 0:W].astype(np.float32)
    dx, dy = xs - pivot[0], ys - pivot[1]
    r, ang = np.hypot(dx, dy), np.arctan2(dy, dx)
    g = np.clip((r - r0) / (length - r0), 0, 1) ** p
    th = np.deg2rad(-raise_deg * f if f > 0 else -lower_deg * f) * g
    s = 1 - shrink * max(f, 0) * g
    rs, sa = r / s, ang - th
    sx, sy = pivot[0] + rs * np.cos(sa), pivot[1] + rs * np.sin(sa)
    x0, y0 = np.floor(sx).astype(int), np.floor(sy).astype(int)
    fx, fy = (sx - x0)[..., None], (sy - y0)[..., None]

    def at(yy, xx):
        ok = ((xx >= 0) & (xx < W) & (yy >= 0) & (yy < H))[..., None]
        return np.where(ok, a[np.clip(yy, 0, H - 1), np.clip(xx, 0, W - 1)], 0)

    o = (at(y0, x0) * (1 - fx) * (1 - fy) + at(y0, x0 + 1) * fx * (1 - fy)
         + at(y0 + 1, x0) * (1 - fx) * fy + at(y0 + 1, x0 + 1) * fx * fy)
    al = o[..., 3:4]
    rgb = np.where(al > 1e-4, o[..., :3] / np.maximum(al, 1e-4), 0)
    return Image.fromarray((np.concatenate([rgb, al], -1).clip(0, 1) * 255 + 0.5).astype(np.uint8), 'RGBA')


def sheet(name, img, pivot, length, amp, r0, p):
    out = Image.new('RGBA', (W * FRAMES, H), (0, 0, 0, 0))
    for i in range(FRAMES):
        out.paste(warp(img, pivot, length, np.sin(2 * np.pi * i / FRAMES), r0=r0, p=p, **amp), (i * W, 0))
    out.save(f'{OUT}/{name}.webp', 'WEBP', quality=88, method=6)


sheet('wing-up', layer(up_r, disc(360, 340, 95)), (360 * k, 340 * k), 330 * k, WING, 45 * k, 1.3)
sheet('wing-right', layer(right_r, disc(400, 385, 95)), (400 * k, 385 * k), 330 * k, WING, 45 * k, 1.3)
sheet('tail', layer(tail_r, None), (300 * k, 425 * k), 420 * k, TAIL, 30 * k, 1.0)
body_img = im.copy()
body_img.putalpha(ImageChops.multiply(a_, body))
body_img.save(f'{OUT}/body.webp', 'WEBP', quality=90, method=6)
print({f: os.path.getsize(f'{OUT}/{f}') // 1024 for f in sorted(os.listdir(OUT))}, 'KB')
