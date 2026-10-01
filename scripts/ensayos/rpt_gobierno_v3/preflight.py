#!/usr/bin/env python3
"""Inventario de fuente RPT: nunca autoriza instalación, identidad ni HTTP."""

from __future__ import annotations

import hashlib
import json
import pathlib
import re
import subprocess
import sys


SQL_PROPIAS = (
    "deploy/postgresql/catalogos_configurables/migraciones/000004_gobierno_categorias_rpt.up.sql",
    "deploy/postgresql/autorizacion_atestada_v3/migraciones/000134_gobierno_categorias_rpt.up.sql",
)
DEPENDENCIAS = (
    "deploy/postgresql/contexto_actor_v1/migraciones/000021_vinculo_rpt_rrhh.up.sql",
    "deploy/postgresql/autorizacion/migraciones/000025_gobierno_categorias_rpt.up.sql",
    "deploy/postgresql/identidad_sesiones_v1/migraciones/000010_politica_high_rrhh_rpt_sintetica.up.sql",
)
RUTA_HTTP = "internal/vec/adapters/httpapi/catalogos_rpt_gobierno.go"
CONSTRUCTOR = "NuevasRutasGobiernoCategoriaRPT"


def git(root: pathlib.Path, *args: str) -> bytes:
    return subprocess.check_output(("git", *args), cwd=root, stderr=subprocess.DEVNULL)


def inventariar(root: pathlib.Path) -> dict:
    head = git(root, "rev-parse", "HEAD").decode().strip()
    hallazgos: list[str] = []
    huellas: dict[str, str] = {}
    for ruta in SQL_PROPIAS + DEPENDENCIAS:
        archivo = root / ruta
        if not archivo.is_file():
            hallazgos.append(f"Falta la fuente SQL causal: {ruta}")
            continue
        contenido = archivo.read_bytes()
        if contenido != git(root, "show", f"HEAD:{ruta}"):
            hallazgos.append(f"La fuente SQL difiere de HEAD: {ruta}")
        huellas[ruta] = hashlib.sha256(contenido).hexdigest()
    fuente_http = root / RUTA_HTTP
    if not fuente_http.is_file() or CONSTRUCTOR not in fuente_http.read_text():
        hallazgos.append("Falta el contrato HTTP de gobierno RRHH")
    llamadores = []
    for base in (root / "cmd", root / "internal"):
        if not base.is_dir():
            continue
        for archivo in base.rglob("*.go"):
            if archivo.name.endswith("_test.go") or archivo == fuente_http:
                continue
            if re.search(rf"\b{CONSTRUCTOR}\s*\(", archivo.read_text()):
                llamadores.append(str(archivo.relative_to(root)))
    if not llamadores:
        hallazgos.append("Las rutas de gobierno RRHH no están compuestas en la aplicación")
    return {
        "fuente_head": head,
        "sql_sha256": huellas,
        "llamadores_rrhh": sorted(llamadores),
        "estado": "bloqueado",
        "hallazgos": hallazgos,
        "limite": (
            "Inventario de fuente únicamente; faltan descriptor y recibos nominales, "
            "ensayo PostgreSQL causal, V3/COSE, HTTP y recuperación tras reinicio"
        ),
    }


def main() -> int:
    root = pathlib.Path(__file__).resolve().parents[3]
    try:
        resultado = inventariar(root)
    except (OSError, subprocess.CalledProcessError, UnicodeError) as exc:
        resultado = {
            "estado": "bloqueado",
            "hallazgos": [f"Fuente no verificable: {type(exc).__name__}"],
            "limite": "Sin ensayo PostgreSQL, decisión V3/COSE ni recorrido HTTP",
        }
    print(json.dumps(resultado, ensure_ascii=False, sort_keys=True, indent=2))
    return 2


if __name__ == "__main__":
    sys.exit(main())
