#!/usr/bin/env python3
"""Extrae la consulta SQL literal del preflight CT136 del hash Go indicado."""

import re
import sys


fuente = sys.stdin.read()
patron = re.compile(
    r"func preflightRegistradorFronteraAuditoriaConsultaDesarrollo\(.*?"
    r'const funcion = "([^"]+)"\s+const sql = `([^`]+)`',
    re.DOTALL,
)
coincidencia = patron.search(fuente)
if not coincidencia:
    raise SystemExit("no se encontró el preflight CT136 exacto")
funcion, consulta = coincidencia.groups()
if consulta.count("$1") != 1 or not funcion.startswith(
    "vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1("
):
    raise SystemExit("preflight CT136 cambiado: revisar extractor")
print(consulta.replace("$1", "'" + funcion.replace("'", "''") + "'") + ";")
