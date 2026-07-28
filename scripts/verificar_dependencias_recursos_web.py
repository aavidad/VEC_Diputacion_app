#!/usr/bin/env python3
"""Comprueba que los recursos web locales cargables estén inventariados."""

from __future__ import annotations

import argparse
from html.parser import HTMLParser
from pathlib import Path, PurePosixPath
import posixpath
import re
import sys
from urllib.parse import unquote, urlsplit


EXTENSIONES_RECURSO = {
    ".css",
    ".gif",
    ".html",
    ".ico",
    ".jpeg",
    ".jpg",
    ".js",
    ".json",
    ".map",
    ".mp3",
    ".ogg",
    ".pdf",
    ".png",
    ".svg",
    ".ttf",
    ".webm",
    ".webp",
    ".woff",
    ".woff2",
}
ATRIBUTOS_RECURSO = {
    "audio": {"src"},
    "embed": {"src"},
    "iframe": {"src"},
    "img": {"src", "srcset"},
    "input": {"src"},
    "link": {"href"},
    "object": {"data"},
    "script": {"src"},
    "source": {"src", "srcset"},
    "track": {"src"},
    "video": {"poster", "src"},
}
PATRON_IMPORTACION_JS = re.compile(
    r"""(?:\bimport\s*\(|\bnew\s+(?:SharedWorker|Worker)\s*\()"""
    r"""\s*["']([^"']+)["']""",
    re.MULTILINE,
)
PATRON_CADENA_RECURSO = re.compile(
    r"""["']([^"'?#]+\.(?:css|gif|html|ico|jpe?g|js|json|map|mp3|ogg|pdf|png|"""
    r"""svg|ttf|webm|webp|woff2?)(?:\?[^"']*)?)["']""",
    re.IGNORECASE,
)
PATRON_URL_CSS = re.compile(r"""url\(\s*["']?([^"')]+)["']?\s*\)""", re.IGNORECASE)


class AnalizadorHTML(HTMLParser):
    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.referencias: list[str] = []

    def handle_starttag(
        self,
        etiqueta: str,
        atributos: list[tuple[str, str | None]],
    ) -> None:
        admitidos = ATRIBUTOS_RECURSO.get(etiqueta.lower())
        if not admitidos:
            return
        for nombre, valor in atributos:
            if not valor or nombre.lower() not in admitidos:
                continue
            if nombre.lower() == "srcset":
                self.referencias.extend(
                    candidato.strip().split()[0]
                    for candidato in valor.split(",")
                    if candidato.strip()
                )
            else:
                self.referencias.append(valor)


def leer_inventario(ruta: Path, prefijo: str) -> set[str]:
    return {
        str(PurePosixPath(prefijo) / linea.strip())
        for linea in ruta.read_text(encoding="utf-8").splitlines()
        if linea.strip()
    }


def normalizar_referencia(referencia: str, origen: str) -> str | None:
    referencia = referencia.strip()
    if not referencia or referencia.startswith(("#", "data:", "blob:", "//")):
        return None
    analizada = urlsplit(referencia)
    if analizada.scheme or analizada.netloc:
        return None
    ruta = unquote(analizada.path)
    if not ruta or "{" in ruta or "}" in ruta:
        return None
    if ruta.startswith("/locales/"):
        candidata = PurePosixPath(ruta.lstrip("/"))
    elif ruta.startswith("/"):
        candidata = PurePosixPath("static") / ruta.lstrip("/")
    else:
        candidata = PurePosixPath(origen).parent / ruta
    normalizada = PurePosixPath(posixpath.normpath(str(candidata)))
    if (
        normalizada.is_absolute()
        or not normalizada.parts
        or normalizada.parts[0] not in {"locales", "static"}
        or ".." in normalizada.parts
        or normalizada.suffix.lower() not in EXTENSIONES_RECURSO
    ):
        return None
    return str(normalizada)


def es_importacion_exclusiva_presentacion(ruta: str) -> bool:
    return any(
        "presentacion" in segmento.lower() or "demo" in segmento.lower()
        for segmento in PurePosixPath(ruta).parts
    )


def referencias_de_archivo(ruta: Path, origen: str) -> list[tuple[str, bool]]:
    texto = ruta.read_text(encoding="utf-8")
    if ruta.suffix.lower() == ".html":
        analizador = AnalizadorHTML()
        analizador.feed(texto)
        return [(referencia, True) for referencia in analizador.referencias]
    if ruta.suffix.lower() == ".css":
        return [(coincidencia, True) for coincidencia in PATRON_URL_CSS.findall(texto)]
    if ruta.suffix.lower() != ".js":
        return []

    importaciones = set(PATRON_IMPORTACION_JS.findall(texto))
    resultado = [(referencia, True) for referencia in importaciones]
    resultado.extend(
        (referencia, False)
        for referencia in PATRON_CADENA_RECURSO.findall(texto)
        if referencia not in importaciones
    )
    return resultado


def comprobar(
    raiz: Path,
    manifiesto: Path,
    manifiesto_locales: Path | None,
) -> list[str]:
    inventario = leer_inventario(manifiesto, "")
    if manifiesto_locales is not None:
        inventario.update(leer_inventario(manifiesto_locales, "locales"))

    errores: list[str] = []
    por_revisar = sorted(ruta for ruta in inventario if ruta.startswith("static/"))
    for origen in por_revisar:
        archivo = raiz / "web" / origen
        if archivo.suffix.lower() not in {".html", ".css", ".js"}:
            continue
        for referencia, obligatoria in referencias_de_archivo(archivo, origen):
            normalizada = normalizar_referencia(referencia, origen)
            if normalizada is None:
                continue
            if archivo.suffix.lower() == ".js" and es_importacion_exclusiva_presentacion(normalizada):
                continue
            archivo_referenciado = (
                raiz / normalizada
                if normalizada.startswith("locales/")
                else raiz / "web" / normalizada
            )
            if not obligatoria and not archivo_referenciado.is_file():
                continue
            if not archivo_referenciado.is_file():
                errores.append(
                    f"{origen}: dependencia local ausente: {normalizada}"
                )
            elif normalizada not in inventario:
                errores.append(
                    f"{origen}: dependencia local no inventariada: {normalizada}"
                )
    return errores


def main() -> int:
    argumentos = argparse.ArgumentParser()
    argumentos.add_argument("manifiesto")
    argumentos.add_argument("--locales")
    opciones = argumentos.parse_args()

    raiz = Path(__file__).resolve().parent.parent
    manifiesto = (raiz / opciones.manifiesto).resolve()
    manifiesto_locales = (
        (raiz / opciones.locales).resolve() if opciones.locales else None
    )
    errores = comprobar(raiz, manifiesto, manifiesto_locales)
    if errores:
        print("\n".join(errores), file=sys.stderr)
        return 1
    print(
        f"Dependencias locales de {opciones.manifiesto} inventariadas.",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
