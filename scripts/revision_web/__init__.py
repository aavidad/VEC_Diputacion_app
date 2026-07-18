"""API estable del capturador y revisor de presentación web."""

from .auditoria import filtrar_abortos_media_exitosos
from .ejecucion import ejecutar_revision, main
from .informes import (
    codigo_salida,
    crear_informe_markdown,
    guardar_informes,
    resumir_resultados,
)
from .modelo import (
    MANIFIESTO,
    MANIFIESTO_FLUJOS,
    MANIFIESTO_VISTAS,
    RAIZ_REPOSITORIO,
    RUTAS_MENU_ASPIRANTE,
    RUTAS_MENU_RRHH,
    SALIDA_PREDETERMINADA,
    SUPERFICIES,
    TAMANOS_VISTA,
    Escenario,
    Flujo,
    PasoInteraccion,
    Superficie,
    TamanoVista,
    Vista,
    construir_url,
    normalizar_url_base,
    slug_castellano,
    validar_manifiesto,
)

__all__ = (
    "MANIFIESTO", "MANIFIESTO_FLUJOS", "MANIFIESTO_VISTAS",
    "RAIZ_REPOSITORIO", "RUTAS_MENU_ASPIRANTE", "RUTAS_MENU_RRHH",
    "SALIDA_PREDETERMINADA", "SUPERFICIES", "TAMANOS_VISTA",
    "Escenario", "Flujo", "PasoInteraccion", "Superficie", "TamanoVista", "Vista",
    "codigo_salida", "construir_url", "crear_informe_markdown", "ejecutar_revision",
    "filtrar_abortos_media_exitosos", "guardar_informes", "main",
    "normalizar_url_base", "resumir_resultados", "slug_castellano", "validar_manifiesto",
)
