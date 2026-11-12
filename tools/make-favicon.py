#!/usr/bin/env python3
"""Genera el favicon.ico del login de Keycloak a partir de la geometría de tools/boceto/img/kaoru.svg.

Keycloak enlaza siempre img/favicon.ico y un .ico no puede ser un SVG, así que el símbolo se
dibuja aquí con la biblioteca estándar (sin dependencias). Si el símbolo cambia, cambia aquí
y se vuelve a correr:  python3 tools/make-favicon.py
"""
import struct
import zlib
from pathlib import Path

SIZE, SCALE = 32, 4  # se dibuja a 4x y se promedia para suavizar bordes
COLOR = (0x5D, 0x69, 0xE2)  # entre el acento claro y el oscuro: se ve sobre ambas pestañas
SEGMENTS = [((7, 4), (7, 20)), ((7, 12), (17, 4)), ((7, 12), (17, 20))]
POINTS = [(7, 4), (7, 20), (17, 4), (17, 20)]
STROKE, RADIUS, VIEWBOX = 2.0, 1.8, 24.0


def distance_to_segment(px, py, a, b):
    (ax, ay), (bx, by) = a, b
    dx, dy = bx - ax, by - ay
    t = max(0.0, min(1.0, ((px - ax) * dx + (py - ay) * dy) / (dx * dx + dy * dy)))
    return ((px - ax - t * dx) ** 2 + (py - ay - t * dy) ** 2) ** 0.5


def inside(x, y):
    if any(distance_to_segment(x, y, a, b) <= STROKE / 2 for a, b in SEGMENTS):
        return True
    return any(((x - cx) ** 2 + (y - cy) ** 2) ** 0.5 <= RADIUS for cx, cy in POINTS)


def render():
    rows = []
    for py in range(SIZE):
        row = bytearray([0])  # filtro PNG «ninguno»
        for px in range(SIZE):
            hits = 0
            for sy in range(SCALE):
                for sx in range(SCALE):
                    x = (px + (sx + 0.5) / SCALE) * VIEWBOX / SIZE
                    y = (py + (sy + 0.5) / SCALE) * VIEWBOX / SIZE
                    hits += inside(x, y)
            row += bytes([*COLOR, round(255 * hits / SCALE**2)])
        rows.append(bytes(row))
    return b"".join(rows)


def chunk(kind, data):
    body = kind + data
    return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body))


def main():
    png = b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", SIZE, SIZE, 8, 6, 0, 0, 0))
    png += chunk(b"IDAT", zlib.compress(render(), 9)) + chunk(b"IEND", b"")
    # Un .ico con una sola imagen PNG dentro.
    ico = struct.pack("<HHH", 0, 1, 1) + struct.pack("<BBBBHHII", SIZE, SIZE, 0, 0, 1, 32, len(png), 22) + png
    out = Path(__file__).resolve().parent.parent / "infra/keycloak/themes/kaoru/login/resources/img/favicon.ico"
    out.write_bytes(ico)
    print(out)


if __name__ == "__main__":
    main()
