# Recording the README demo

`docs/demo.gif` is a real recording of `lootmap` running against a **demo home
directory full of fake keys** (`home.txtar`). No real credentials were
used. To re-record it (Linux or macOS):

```bash
# 1. Build lootmap
go build -ldflags "-X main.version=0.1.0" -o /tmp/lootmap ./cmd/lootmap

# 2. Create the demo home with deterministic fake secrets
rm -rf /tmp/lootmap-demo && mkdir -p /tmp/lootmap-demo
go run ./docs/demo-src/gen docs/demo-src/home.txtar /tmp/lootmap-demo/home
chmod 644 /tmp/lootmap-demo/home/.ssh/id_ed25519
find /tmp/lootmap-demo/home -exec touch -d '3 hours ago' {} +   # skip live-session guard

# 3. Record and render (needs Python 3, pyte, Pillow, ffmpeg, JetBrains Mono)
pip install pyte pillow
python3 docs/demo-src/capture.py /tmp/session.json
FONT_DIR=/path/to/JetBrainsMono/fonts/ttf python3 docs/demo-src/render.py /tmp/session.json docs/demo.gif
```
