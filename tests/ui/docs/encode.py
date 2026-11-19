"""Turns docs/media/demo.webm (Playwright's recording) into a small, widely playable demo.mp4.
Needs ffmpeg on PATH, or `pip install imageio-ffmpeg`."""
import shutil, subprocess, sys
from pathlib import Path

media = Path(__file__).resolve().parents[3] / "docs" / "media"
ffmpeg = shutil.which("ffmpeg")
if not ffmpeg:
    try:
        import imageio_ffmpeg
        ffmpeg = imageio_ffmpeg.get_ffmpeg_exe()
    except ImportError:
        sys.exit("ffmpeg not found: install it or run `pip install imageio-ffmpeg`")
src, dst = media / "demo.webm", media / "demo.mp4"
subprocess.run([ffmpeg, "-y", "-loglevel", "error", "-i", str(src), "-an", "-c:v", "libx264", "-preset", "slow", "-crf", "30",
                "-pix_fmt", "yuv420p", "-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2,fps=25", "-movflags", "+faststart", str(dst)], check=True)
src.unlink()
print("demo.mp4", round(dst.stat().st_size / 1024), "KiB")
