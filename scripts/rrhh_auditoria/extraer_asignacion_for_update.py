#!/usr/bin/env python3
"""Extrae el SELECT FOR UPDATE del helper de asignación del hash Go."""

import re
import sys


fuente = sys.stdin.read()
coincidencia = re.search(
    r"func leerAsignacionActualPostgreSQLDesarrollo\(.*?"
    r"consultador\.QueryRow\(ctx, `([^`]+)`, perfilRef\)",
    fuente,
    re.DOTALL,
)
if not coincidencia:
    raise SystemExit("no se encontró la lectura de asignación actual")
consulta = coincidencia.group(1)
if consulta.count("$1") != 1 or "FOR UPDATE OF vigente" not in consulta:
    raise SystemExit("lectura FOR UPDATE cambiada: revisar extractor")
print(consulta.replace("$1", "'perfil:auditoria:sintetico'") + ";")
