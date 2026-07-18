#!/usr/bin/env python3
"""Fachada compatible y CLI del capturador de la presentación web VEC.

La implementación vive en :mod:`scripts.revision_web`. Playwright continúa
importándose de forma perezosa dentro de ``ejecutar_revision``.
"""

from __future__ import annotations

import sys
from importlib import import_module

_api = import_module("scripts.revision_web" if __package__ else "revision_web")

__all__ = (
    "MANIFIESTO", "MANIFIESTO_FLUJOS", "MANIFIESTO_VISTAS",
    "RAIZ_REPOSITORIO", "RUTAS_MENU_ASPIRANTE", "RUTAS_MENU_RRHH",
    "SALIDA_PREDETERMINADA", "SUPERFICIES", "TAMANOS_VISTA",
    "Escenario", "Flujo", "PasoInteraccion", "Superficie", "TamanoVista", "Vista",
    "codigo_salida", "construir_url", "crear_informe_markdown", "ejecutar_revision",
    "filtrar_abortos_media_exitosos", "guardar_informes", "main",
    "normalizar_url_base", "resumir_resultados", "slug_castellano", "validar_manifiesto",
)

for _nombre in __all__:
    globals()[_nombre] = getattr(_api, _nombre)


if __name__ == "__main__":
    sys.exit(main())
