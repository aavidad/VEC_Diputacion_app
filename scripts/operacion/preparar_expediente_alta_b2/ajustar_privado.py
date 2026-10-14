#!/usr/bin/env python3
"""Edición manual acotada de dos valores privados de la preparación B2."""
from __future__ import annotations

import argparse
import json
import os
import re
import stat
import sys
import tempfile
from pathlib import Path


FLAG = "VEC_PERSONAL_B2_GOBIERNO_ENABLED"
LINEA = re.compile(r"^([ \t]*(?:export[ \t]+)?VEC_PERSONAL_B2_GOBIERNO_ENABLED[ \t]*=[ \t]*)"
                   r"(?:'[^']*'|\"[^\"]*\"|[^\s#]*)([ \t]*(?:#.*)?)(\r?\n?)$")


def leer_privado(ruta: Path) -> tuple[str, int]:
    fd = os.open(ruta, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, "r", encoding="utf-8") as fuente:
        meta = os.fstat(fuente.fileno())
        if not stat.S_ISREG(meta.st_mode) or meta.st_nlink != 1 or meta.st_size > 1024 * 1024:
            raise ValueError("fichero privado no regular o demasiado grande")
        return fuente.read(), stat.S_IMODE(meta.st_mode)


def guardar_privado(ruta: Path, contenido: str, modo: int) -> None:
    fd, temporal = tempfile.mkstemp(prefix=".vec-b2-", dir=ruta.parent)
    try:
        os.fchmod(fd, modo)
        with os.fdopen(fd, "w", encoding="utf-8") as salida:
            salida.write(contenido)
            salida.flush()
            os.fsync(salida.fileno())
        os.replace(temporal, ruta)
        directorio = os.open(ruta.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directorio)
        finally:
            os.close(directorio)
    finally:
        if os.path.exists(temporal):
            os.unlink(temporal)


def editar_flag(texto: str) -> str:
    lineas = texto.splitlines(keepends=True)
    posiciones = [i for i, linea in enumerate(lineas) if LINEA.fullmatch(linea)]
    if len(posiciones) > 1:
        raise ValueError("hay varias asignaciones del flag; revisar lanzador")
    if posiciones:
        i = posiciones[0]
        match = LINEA.fullmatch(lineas[i])
        lineas[i] = match.group(1) + "true" + match.group(2) + match.group(3)
    else:
        ejecuciones = [i for i, linea in enumerate(lineas)
                       if re.match(r"^[ \t]*exec[ \t]+\S", linea)]
        if len(ejecuciones) != 1:
            raise ValueError("no hay un único exec del servidor; revisar lanzador antes de insertar el flag")
        lineas.insert(ejecuciones[0], f"export {FLAG}=true\n")
    return "".join(lineas)


def editar_cese(texto: str) -> str:
    datos = json.loads(texto)
    if not isinstance(datos, dict) or not isinstance(datos.get("personal_b2"), dict):
        raise ValueError("falta el objeto personal_b2 en el JSON privado")
    datos["personal_b2"]["cese_fecha_efecto"] = "primer_dia_sin_relacion"
    return json.dumps(datos, ensure_ascii=False, indent=2) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operacion", choices=("flag", "cese"))
    parser.add_argument("ruta", type=Path)
    args = parser.parse_args()
    ruta = args.ruta.absolute()
    texto, modo = leer_privado(ruta)
    resultado = editar_flag(texto) if args.operacion == "flag" else editar_cese(texto)
    if resultado != texto:
        guardar_privado(ruta, resultado, modo)
    print("Valor privado ajustado; compruebe el arranque y repita --comprobar")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError) as exc:
        print(f"FALTA: {exc}", file=sys.stderr)
        raise SystemExit(2) from None
