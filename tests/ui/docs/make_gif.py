"""Assembles docs/media/demo.gif from the screenshots (no ffmpeg needed). Run after the capture spec."""
from pathlib import Path
from PIL import Image

media = Path(__file__).resolve().parents[3] / "docs" / "media"
frames = [("01-home", 1600), ("02-pairing", 2200), ("03-connected", 1600), ("04-incoming", 2200), ("05-transferring", 2400), ("06-received", 3000)]
imgs, durations = [], []
for name, ms in frames:
    im = Image.open(media / f"{name}.png").convert("RGB")
    im = im.resize((960, round(im.height * 960 / im.width)), Image.LANCZOS)
    imgs.append(im.quantize(colors=96, method=Image.MEDIANCUT, dither=Image.NONE))
    durations.append(ms)
imgs[0].save(media / "demo.gif", save_all=True, append_images=imgs[1:], duration=durations, loop=0, optimize=True)
print("demo.gif", round((media / "demo.gif").stat().st_size / 1024), "KiB")
