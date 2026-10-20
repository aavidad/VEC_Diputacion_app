"""Dato corporativo ficticio del perfil de Bolsa en esta copia."""

import hashlib
import json
from pathlib import Path
import re

from admin_copia.arranque import objeto_sin_duplicados


SID = re.compile(r'[0-9]{15,20}\Z')
PERFIL = re.compile(r'prf_[0-9a-f]{32}\Z')
ORGANISMO = re.compile(r'org_[0-9a-f]{32}\Z')


def _entradas(c):
    estado = c.copia.leer()
    sid = estado.get('sid_copia')
    if not isinstance(sid, str) or not SID.fullmatch(sid) or int(sid) <= 0:
        raise ValueError('Falta el identificador de sistema de esta copia')
    if c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();') != sid:
        raise ValueError('El identificador PostgreSQL difiere del estado de la copia')

    manifiesto = c.leer(c.material / 'identidad/bolsa-bback.json')
    perfil = manifiesto.get('perfil_ref') if isinstance(manifiesto, dict) else None
    seleccion = c.fuente / 'deploy/principal/candidata_hz39/datos/seleccion.json'
    if not seleccion.is_file() or seleccion.is_symlink():
        raise ValueError('Falta la selección pública de Organización histórica')
    datos = json.loads(seleccion.read_bytes(), object_pairs_hook=objeto_sin_duplicados)
    organismo = datos.get('organismo_ref') if isinstance(datos, dict) else None
    if (not isinstance(perfil, str) or not PERFIL.fullmatch(perfil)
            or not isinstance(organismo, str) or not ORGANISMO.fullmatch(organismo)):
        raise ValueError('Perfil de Bolsa u organismo del caso inválidos')
    return sid, perfil, organismo


def _observado(c, perfil, organismo, referencia):
    consulta = f"""SELECT pg_catalog.jsonb_build_object(
      'vinculo_corporativo_ref', v.vinculo_corporativo_ref,
      'version', v.version, 'cuenta_ref', v.cuenta_ref,
      'persona_ref', v.persona_ref, 'perfil_ref', v.perfil_ref,
      'organismo_ref', v.organizacion_ref, 'vigente_desde', v.vigente_desde,
      'vigente_hasta', v.vigente_hasta)::text
      FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
      JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones v
        ON v.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND v.version=a.version
      WHERE a.perfil_ref='{perfil}' AND a.superficie='interna_corporativa'
        AND a.uso='consulta_rrhh' AND a.vinculo_corporativo_ref='{referencia}'
        AND v.perfil_ref='{perfil}' AND v.organizacion_ref='{organismo}'
        AND v.estado='activo' AND pg_catalog.clock_timestamp()>=v.vigente_desde
        AND pg_catalog.clock_timestamp()<v.vigente_hasta;"""
    try:
        fila = json.loads(c.consultar(consulta))
    except (ValueError, TypeError) as error:
        raise ValueError('El vínculo corporativo no quedó acreditado tras el SQL') from error
    if (not isinstance(fila, dict) or fila.get('vinculo_corporativo_ref') != referencia
            or fila.get('perfil_ref') != perfil or fila.get('organismo_ref') != organismo
            or fila.get('version') != 1):
        raise ValueError('La consulta posterior no acredita el vínculo esperado')
    return fila


def preparar(c):
    """Inserta o coteja el vínculo y guarda lo observado tras confirmar SQL."""
    sid, perfil, organismo = _entradas(c)
    plantilla = Path(__file__).parent / 'datos/semilla_corporativa.sql'
    contenido = plantilla.read_bytes()
    prefijo = (f'\\set sid_copia {sid}\n\\set perfil_ref {perfil}\n'
               f'\\set organismo_ref {organismo}\n').encode('ascii')
    sql = prefijo + contenido
    referencia = 'vcr_' + hashlib.sha256(
        f'vec:f1-131:corporativo:{sid}:{perfil}'.encode('ascii')).hexdigest()[:32]
    c.sql(sql, 'corporativo-semilla.log')
    observado = _observado(c, perfil, organismo, referencia)
    recibo = {'version': 1, 'copia_ref': c.copia.nombre, 'sid_copia': sid,
              'sql_sha256': hashlib.sha256(sql).hexdigest(), **observado}
    c.guardar(c.carpeta / 'corporativo/recibo.json', recibo)
    return recibo
