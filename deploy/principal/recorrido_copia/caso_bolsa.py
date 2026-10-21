"""Enlaza el aspirante de la copia con la bolsa recién cargada por su API."""

import json
import re

from . import carga_bolsa
from .roles_admin import literal


def preparar(c):
    recibo = carga_bolsa.preparar(c)
    primero = recibo['aspirante']
    consulta = ("SELECT jsonb_build_object('candidato',v.candidato_ref,"
        "'participacion',e.participacion_ref,'instantanea_ref',b.instantanea_ref)::text "
        "FROM vec_bolsa_llamamientos.constitucion b "
        "JOIN vec_bolsa_llamamientos.constitucion_entrada e "
        "USING(instantanea_ref,version_instantanea) "
        "JOIN vec_bolsa_llamamientos.vinculo_candidato v ON v.acta_ref=b.acta_ref "
        "AND v.instantanea_ref=e.instantanea_ref AND v.version_instantanea=e.version_instantanea "
        "AND v.participacion_ref=e.participacion_ref "
        f"WHERE b.bolsa_ref={literal(recibo['bolsa_ref'])} "
        f"AND b.acta_ref={literal(recibo['acta_ref'])} AND b.version_bolsa={recibo['version']} "
        f"AND e.orden=1 AND e.fila_numero={primero['fila_numero']};")
    filas = c.consultar(consulta).splitlines()
    if len(filas) != 1:
        raise ValueError('La primera posición cargada no tiene un vínculo único')
    vinculo = json.loads(filas[0])
    if (not re.fullmatch(r'can_[A-Za-z0-9_-]{22,128}', vinculo.get('candidato', ''))
            or not re.fullmatch(r'participacion:[A-Za-z0-9:_-]{1,200}', vinculo.get('participacion', ''))):
        raise ValueError('El vínculo de la primera posición no es válido')
    c.guardar(c.raiz / 'caso40-seleccion.json', {
        'copia_ref': c.copia.nombre, 'sid_copia': c.copia.leer()['sid_copia'],
        'carga': {k: v for k, v in recibo.items() if k != 'aspirante'},
        'seleccion': {'bolsa': recibo['bolsa_ref'], 'candidato': vinculo['candidato'],
                      'participacion': vinculo['participacion'], 'nombre': primero['nombre'],
                      'identidad': 'aspirante-caso40'}})
    # La identidad se prepara con PostgreSQL activo y la aplicación parada.
    c.copia.parar()
    c.copia.pg()
    import copia_local_caso40 as caso40
    from copia_local import guardar, sha256
    propuesto = caso40.preparar_aspirante(c.copia, c.fuente)
    entorno = caso40.entorno_portal(c.copia.entorno_modulos(), propuesto)
    ruta = c.raiz / 'entorno_modulos.json'
    guardar(ruta, entorno)
    estado = c.copia.leer()
    estado['entorno_modulos_sha256'] = sha256(ruta)
    guardar(c.copia.estado, estado)
    c.copia.arrancar()
    caso40.registrar_contacto(c.copia, c.fuente)
    return recibo
