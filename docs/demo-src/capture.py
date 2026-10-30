# Runs the real lootmap binary in a pty against the demo home and records a
# terminal session as (delay_seconds, bytes) events. Typing is simulated.
import os, pty, json, subprocess, select, time, sys

HOME = os.environ.get("DEMO_HOME", "/tmp/lootmap-demo/home")
BIN = os.environ.get("LOOTMAP_BIN", "/tmp/lootmap")
COLS, ROWS = 96, 23
env = dict(os.environ, HOME=HOME, COLUMNS=str(COLS), LINES=str(ROWS),
           TERM="xterm-256color", COLORTERM="truecolor")
env.pop("NO_COLOR", None)

events = []
def emit(delay, data): events.append([delay, data])

PROMPT = "\x1b[1;38;2;125;211;252m~\x1b[0m \x1b[38;2;148;163;184m❯\x1b[0m "

def type_cmd(cmd, pause_before=0.6):
    emit(pause_before, PROMPT)
    for i, ch in enumerate(cmd):
        emit(0.35 if i == 0 else 0.055, ch)
    emit(0.45, "\r\n")

def run(args, line_delay=0.035):
    import fcntl, struct, termios
    pid, fd = pty.fork()
    if pid == 0:
        os.execve(BIN, [BIN] + args, env)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", ROWS, COLS, 0, 0))
    out = b""
    while True:
        try:
            r, _, _ = select.select([fd], [], [], 5)
            if not r: break
            chunk = os.read(fd, 65536)
            if not chunk: break
            out += chunk
        except OSError:
            break
    os.waitpid(pid, 0)
    text = out.decode("utf-8")
    first = True
    for line in text.splitlines(keepends=True):
        emit(0.25 if first else line_delay, line)
        first = False
    return text

# Scene 1: scan
type_cmd("lootmap scan", 0.8)
run(["scan"])
emit(4.2, "")
# Scene 2: fix
emit(0.0, "\x1b[2J\x1b[H")
type_cmd("lootmap fix --yes", 0.5)
run(["fix", "--yes"], line_delay=0.12)
emit(2.8, "")
# Scene 3: rescan
emit(0.0, "\x1b[2J\x1b[H")
type_cmd("lootmap scan", 0.5)
run(["scan"])
emit(4.5, "")
json.dump({"cols": COLS, "rows": ROWS, "events": events}, open(sys.argv[1], "w"))
print(len(events), "events")
