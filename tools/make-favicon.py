#!/usr/bin/env python3
"""Genera el favicon.ico del login de Keycloak a partir de la geometría de tools/boceto/img/kaoru.svg
(el sello índigo con la K en blanco).

Keycloak enlaza siempre img/favicon.ico y un .ico no puede ser un SVG, así que el símbolo se
dibuja aquí con la biblioteca estándar (sin dependencias). Si el símbolo cambia, cambia aquí
y se vuelve a correr:  python3 tools/make-favicon.py
"""
import struct
import zlib
from pathlib import Path

SIZE, SCALE = 32, 4  # se dibuja a 4x y se promedia para suavizar bordes
SEAL = (0x4F, 0x5B, 0xD6)  # el acento oscuro: el sello se ve sobre pestañas claras y oscuras
MARK = (0xFF, 0xFF, 0xFF)  # la K, en on-accent
BOX, CORNER = (2.0, 2.0, 22.0, 22.0), 5.0  # el sello: rect x=2 y=2 20x20 rx=5
# La K va dentro del sello con translate(3.92 3.92) scale(0.66), igual que en el SVG.
OFFSET, FACTOR = 3.92, 0.66
SEGMENTS = [((7, 4), (7, 20)), ((7, 12), (17, 4)), ((7, 12), (17, 20))]
POINTS = [(7, 4), (7, 20), (17, 4), (17, 20)]
STROKE, RADIUS, VIEWBOX = 2.6, 2.2, 24.0


def distance_to_segment(px, py, a, b):
    (ax, ay), (bx, by) = a, b
    dx, dy = bx - ax, by - ay
    t = max(0.0, min(1.0, ((px - ax) * dx + (py - ay) * dy) / (dx * dx + dy * dy)))
    return ((px - ax - t * dx) ** 2 + (py - ay - t * dy) ** 2) ** 0.5


def in_seal(x, y):
    x0, y0, x1, y1 = BOX
    # Distancia al rectángulo interior: dentro si queda a menos del radio de la esquina.
    dx = max(x0 + CORNER - x, 0.0, x - (x1 - CORNER))
    dy = max(y0 + CORNER - y, 0.0, y - (y1 - CORNER))
    return dx * dx + dy * dy <= CORNER * CORNER


def in_mark(x, y):
    x, y = (x - OFFSET) / FACTOR, (y - OFFSET) / FACTOR
    if any(distance_to_segment(x, y, a, b) <= STROKE / 2 for a, b in SEGMENTS):
        return True
    return any(((x - cx) ** 2 + (y - cy) ** 2) ** 0.5 <= RADIUS for cx, cy in POINTS)


def render():
    rows = []
    for py in range(SIZE):
        row = bytearray([0])  # filtro PNG «ninguno»
        for px in range(SIZE):
            seal = mark = 0
            for sy in range(SCALE):
                for sx in range(SCALE):
                    x = (px + (sx + 0.5) / SCALE) * VIEWBOX / SIZE
                    y = (py + (sy + 0.5) / SCALE) * VIEWBOX / SIZE
                    if in_seal(x, y):
                        seal += 1
                        mark += in_mark(x, y)
            # Color: la mezcla de sello y K en las muestras cubiertas; alfa: cuánto cubre el sello.
            t = mark / seal if seal else 0.0
            color = [round(s + (m - s) * t) for s, m in zip(SEAL, MARK)]
            row += bytes([*color, round(255 * seal / SCALE**2)])
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
