"""Generate original PNG pixels; no downloaded images or third-party artwork."""

import base64
from pathlib import Path
import struct
import zlib


def chunk(kind, data):
    return (struct.pack(">I", len(data)) + kind + data
            + struct.pack(">I", zlib.crc32(kind + data)))


def png(size):
    rows = bytearray()
    state = 1
    for _ in range(size):
        rows.append(0)  # PNG scanline filter: none.
        for _ in range(size * 3):
            state = (1664525 * state + 1013904223) & 0xffffffff
            rows.append(state >> 24)
    return (b"\x89PNG\r\n\x1a\n"
            + chunk(b"IHDR", struct.pack(">IIBBBBB", size, size, 8, 2, 0, 0, 0))
            + chunk(b"IDAT", zlib.compress(rows, level=0))
            + chunk(b"IEND", b""))


css = ["/* Synthetic fixture: original generated PNG pixels, no third-party content.\n"
       "   Reproduce: python3 test-data/corpus/generate-data-uri.py */\n"]
for size, quote in [(128, '"'), (256, '')]:
    image = base64.b64encode(png(size)).decode("ascii")
    css.append(f".generated-{size} {{ background-image: "
               f"url({quote}data:image/png;base64,{image}{quote}); }}\n")
Path(__file__).with_name("generated-data-uri.css").write_text("".join(css), encoding="ascii")
