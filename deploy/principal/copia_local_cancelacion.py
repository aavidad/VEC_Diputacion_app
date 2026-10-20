"""Enciende en la copia local la cancelación del expediente de CT (F1-96).

La cancelación (CT 000122 y AD3-87, ya presentes en el volcado de referencia)
solo se compone con su selector y dos catálogos versionados: el de reglas de
CT (regla c20, fases en las que se admite) y el de motivos de cancelación. Sin
ellos la ficha recibe 404 al consultarla y no muestra el apartado. Se añade
tras el kit de datos de la copia; un valor que el kit ya haya fijado se
conserva, para no pisar una configuración privada comprobada.
"""
from pathlib import Path

CATALOGOS = {
    'VEC_CT_REGLAS_SOURCE_PATH': 'data/catalogos/contratacion-temporal/reglas.v3.json',
    'VEC_CT_MOTIVOS_CANCELACION_SOURCE_PATH': 'data/demo/reglas/ct_motivos_cancelacion.demo.json',
}
SELECTOR = 'VEC_CT_CANCELACION_ENABLED'


def entorno_cancelacion(entorno, fuente):
    """Devuelve el entorno de módulos con la cancelación encendida."""
    nuevo = dict(entorno)
    for variable, relativa in CATALOGOS.items():
        if nuevo.get(variable):
            continue
        ruta = Path(fuente) / relativa
        if not ruta.is_file():
            raise ValueError('Falta el catálogo versionado de la cancelación')
        nuevo[variable] = str(ruta)
    nuevo[SELECTOR] = 'true'
    return nuevo


def encender(copia, guardar, sha256):
    """Escribe el entorno de módulos de la copia y deja su huella en el estado."""
    estado = copia.leer()
    ruta = copia.raiz / 'entorno_modulos.json'
    guardar(ruta, entorno_cancelacion(copia.entorno_modulos(), estado['fuente']))
    estado['entorno_modulos_sha256'] = sha256(ruta)
    guardar(copia.estado, estado)
