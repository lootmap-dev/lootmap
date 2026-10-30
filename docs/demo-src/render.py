# Renders a recorded session (capture.py) into PNG frames with a window frame,
# then encodes an optimized GIF with ffmpeg.
import json, sys, os, subprocess, shutil
import pyte
from PIL import Image, ImageDraw, ImageFont

src, out = sys.argv[1], sys.argv[2]
d = json.load(open(src))
COLS, ROWS = d["cols"], d["rows"]
FONT_DIR = os.environ.get("FONT_DIR", "/tmp/fonts/fonts/ttf")
FONT = os.path.join(FONT_DIR, "JetBrainsMono-Regular.ttf")
FONTB = os.path.join(FONT_DIR, "JetBrainsMono-Bold.ttf")
SIZE = 15
font, fontb = ImageFont.truetype(FONT, SIZE), ImageFont.truetype(FONTB, SIZE)
fallback = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf", SIZE)
CW = int(round(font.getlength("M")))
LH = int(SIZE * 1.45)
PAD_X, PAD_TOP, PAD_BOT, BAR = 22, 14, 18, 34
MARGIN = 26
W = COLS * CW + 2 * PAD_X
H = BAR + PAD_TOP + ROWS * LH + PAD_BOT

BG_OUT = (226, 232, 240)
BG = (13, 17, 23)
BAR_BG = (22, 27, 34)
FG = (214, 222, 235)
PALETTE = {
    "default": FG, "black": (40, 44, 52), "red": (248, 81, 73), "green": (63, 185, 80),
    "brown": (210, 153, 34), "yellow": (210, 153, 34), "blue": (88, 166, 255),
    "magenta": (188, 140, 255), "cyan": (57, 197, 207), "white": (201, 209, 217),
    "brightblack": (110, 118, 129), "brightred": (255, 107, 107), "brightgreen": (86, 211, 100),
    "brightyellow": (255, 209, 102), "brightbrown": (255, 209, 102), "brightblue": (121, 192, 255), "brightmagenta": (210, 168, 255),
    "brightcyan": (86, 212, 221), "brightwhite": (240, 246, 252),
}

def color(c, bold=False):
    if isinstance(c, str) and len(c) == 6 and all(ch in "0123456789abcdefABCDEF" for ch in c):
        return tuple(int(c[i:i+2], 16) for i in (0, 2, 4))
    if bold and c in ("red", "green", "yellow", "brown", "blue", "magenta", "cyan", "white"):
        c = "bright" + ("yellow" if c == "brown" else c)
    return PALETTE.get(c, FG)

def has_glyph(ch):
    try:
        return font.getmask(ch).getbbox() is not None or ch == " "
    except Exception:
        return False

def rounded(img, r):
    mask = Image.new("L", img.size, 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, img.size[0]-1, img.size[1]-1], r, fill=255)
    return mask

def render(screen, cursor):
    win = Image.new("RGB", (W, H), BG)
    dr = ImageDraw.Draw(win)
    dr.rectangle([0, 0, W, BAR], fill=BAR_BG)
    for i, c in enumerate([(255, 95, 86), (255, 189, 46), (39, 201, 63)]):
        cx = 20 + i * 20
        dr.ellipse([cx-6, BAR//2-6, cx+6, BAR//2+6], fill=c)
    title = "lootmap — demo home (fake keys)"
    tw = fallback.getlength(title)
    dr.text(((W - tw) / 2, BAR/2 - SIZE*0.6), title, font=fallback, fill=(139, 148, 158))
    y0 = BAR + PAD_TOP
    for y in range(ROWS):
        line = screen.buffer[y]
        for x in range(COLS):
            ch = line[x]
            if ch.data in (" ", "") and ch.bg == "default":
                continue
            fg = color(ch.fg, ch.bold)
            px, py = PAD_X + x * CW, y0 + y * LH
            if ch.bg != "default":
                dr.rectangle([px, py, px + CW, py + LH], fill=color(ch.bg))
            f = fontb if ch.bold else font
            if not has_glyph(ch.data):
                f = fallback
            dr.text((px, py + (LH - SIZE) / 2 - 1), ch.data, font=f, fill=fg)
    if cursor:
        cx, cy = screen.cursor.x, screen.cursor.y
        px, py = PAD_X + cx * CW, y0 + cy * LH
        dr.rectangle([px, py + 2, px + CW - 1, py + LH - 2], fill=(125, 211, 252))
    canvas = Image.new("RGB", (W + 2*MARGIN, H + 2*MARGIN), BG_OUT)
    shadow = Image.new("RGB", (W, H), (180, 188, 200))
    canvas.paste(shadow, (MARGIN, MARGIN + 6), rounded(shadow, 12))
    canvas.paste(win, (MARGIN, MARGIN), rounded(win, 12))
    return canvas

screen = pyte.Screen(COLS, ROWS)
stream = pyte.Stream(screen)
fdir = os.environ.get("FRAMES_DIR", "/tmp/lootmap-frames"); shutil.rmtree(fdir, ignore_errors=True); os.makedirs(fdir)
events = d["events"]
frames = []  # (path, duration)
typing = False
for i, (delay, data) in enumerate(events):
    if frames:
        frames[-1][1] += delay
    if data:
        stream.feed(data)
        # cursor visible while a prompt line is being typed
        typing = data != "\r\n" and (len(data) == 1 or "❯" in data)
        p = f"{fdir}/{len(frames):04d}.png"
        render(screen, typing).save(p)
        frames.append([p, 0.0])
frames[-1][1] += 0.5
with open(f"{fdir}/list.txt", "w") as fh:
    for p, dur in frames:
        fh.write(f"file '{p}'\nduration {max(dur, 0.02):.3f}\n")
    fh.write(f"file '{frames[-1][0]}'\n")
total = sum(x[1] for x in frames)
print(len(frames), "frames", round(total, 1), "s", W + 2*MARGIN, "x", H + 2*MARGIN)
vf = "fps=12,split[a][b];[a]palettegen=max_colors=96:stats_mode=full[p];[b][p]paletteuse=dither=none"
subprocess.run(["ffmpeg", "-y", "-loglevel", "error", "-f", "concat", "-safe", "0", "-i", f"{fdir}/list.txt",
                "-vf", vf, "-loop", "0", out], check=True)
print(os.path.getsize(out) // 1024, "KB")
