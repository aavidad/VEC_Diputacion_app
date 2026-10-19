#!/usr/bin/env python3
"""Sustitución reproducible de las versiones manuales por la marca del servidor."""
from pathlib import Path
import re

RAIZ = Path(__file__).resolve().parents[1] / 'web' / 'static'
MARCA = 'VEC_WEB_VERSION'
TIPOS = {'.js', '.html', '.css', '.json', '.webmanifest'}


def normalizar():
    versiones = set()
    fuentes = sorted(p for p in RAIZ.rglob('*') if p.suffix in TIPOS)
    for ruta in fuentes:
        versiones.update(re.findall(r'\?v=([\w.-]+)', ruta.read_text()))
    for ruta in fuentes:
        texto = ruta.read_text()
        nuevo = re.sub(r'\?v=[\w.-]+', '?v=' + MARCA, texto)
        if ruta.name in {'sw-public-assets.js', 'instalar.js', 'cache-publica-v1.json'}:
            nuevo = re.sub(r'(["\'])2026100[23]-pwa-(?:ci-v5|v1)\1',
                           lambda m: m[1] + MARCA + m[1], nuevo)
        if nuevo != texto:
            ruta.write_text(nuevo)
    for ruta in sorted(RAIZ.rglob('*.mjs')):
        texto = ruta.read_text()
        nuevo = re.sub(r'(\?v=|\\\?v=)(?:202[\w.-]+|1\\?\.9\\?\.4|VEC_WEB_VERSION)', lambda m: m[1] + MARCA, texto)
        for version in sorted(versiones, key=len, reverse=True):
            for comilla in ('"', "'"):
                nuevo = nuevo.replace(comilla + version + comilla, comilla + MARCA + comilla)
        if nuevo != texto:
            ruta.write_text(nuevo)


if __name__ == '__main__':
    normalizar()
