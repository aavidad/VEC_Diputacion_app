"""Coteja en solo lectura las cuatro admisiones nominales de la copia."""

import hashlib
import json
from pathlib import Path
import re

from .firmantes import _roles

ROLES = ('tecnico', 'jefatura', 'diputacion', 'secretaria')
SEGURA = re.compile(r'[A-Za-z0-9_:-]{1,200}\Z')
HEX = re.compile(r'[0-9a-f]{64}\Z')


def _fuente(c, ruta):
    ruta = Path(ruta)
    if (not ruta.is_absolute() or ruta.is_symlink() or not ruta.is_file()
            or ruta.stat().st_mode & 0o077
            or not ruta.resolve().is_relative_to(c.carpeta.resolve())):
        raise ValueError('Fuente privada de admisiones ajena a la copia')
    return c.leer(ruta)


def _literal(valor):
    if not isinstance(valor, str) or not SEGURA.fullmatch(valor):
        raise ValueError('Referencia de admisión no canónica')
    return "'" + valor + "'"


def _proyeccion(esperado, actual):
    if isinstance(esperado, dict) and isinstance(actual, dict):
        return all(k in actual and _proyeccion(v, actual[k]) for k, v in esperado.items())
    if isinstance(esperado, list) and isinstance(actual, list):
        return len(esperado) == len(actual) and all(_proyeccion(a, b) for a, b in zip(esperado, actual))
    return esperado == actual


def _consulta(valores):
    filas = ','.join('(' + ','.join(_literal(v) for v in fila) + ')' for fila in valores)
    return f"""WITH esperado(rol,perfil,persona,cuenta,certificado,vinculo_contexto,vinculo_certificado,recibo,decision,auditoria) AS (VALUES {filas})
SELECT jsonb_agg(to_jsonb(fila) ORDER BY fila.rol) FROM (
 SELECT e.rol,a.perfil_activo_ref AS perfil_ref,a.asignacion_ref,a.version AS asignacion_version,
        a.huella_sha256 AS asignacion_sha256,r.version_rol_ref AS rol_version_ref,
        r.huella_sha256 AS rol_sha256,control.revision AS control_revision,
        control.huella_sha256 AS control_sha256,
        e.persona AS persona_ref,e.cuenta AS cuenta_ref,e.certificado AS certificado_sha256,
        cert.vinculo_ref AS vinculo_certificado_ref,cert.recibo_ref,cert.decision_ref,
        cert.auditoria_ref,convert_from(cert.documento_canonico,'UTF8')::jsonb AS recibo_db
 FROM esperado e
 JOIN vec_autorizacion.asignacion_perfil_actual actual ON actual.perfil_activo_ref=e.perfil
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=actual.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual control_actual ON control_actual.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol control ON (control.version_rol_ref,control.revision)=(control_actual.version_rol_ref,control_actual.revision)
 JOIN vec_contexto_actor_v1.perfil_actual perfil_actual ON perfil_actual.perfil_ref=e.perfil
 JOIN vec_contexto_actor_v1.perfil_versiones perfil ON (perfil.perfil_ref,perfil.version)=(perfil_actual.perfil_ref,perfil_actual.version)
 JOIN vec_contexto_actor_v1.vinculo_contexto_actual vinculo_actual ON vinculo_actual.vinculo_ref=e.vinculo_contexto
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vinculo ON (vinculo.vinculo_ref,vinculo.version)=(vinculo_actual.vinculo_ref,vinculo_actual.version)
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual cuenta_actual ON cuenta_actual.cuenta_ref=e.cuenta
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cuenta ON (cuenta.cuenta_ref,cuenta.version)=(cuenta_actual.cuenta_ref,cuenta_actual.version)
 JOIN vec_contexto_actor_v1.certificado_firmante_nominal_actual certificado_actual ON certificado_actual.certificado_der_sha256=e.certificado
 JOIN vec_contexto_actor_v1.certificado_firmante_nominal_versiones cert ON
      (cert.certificado_der_sha256,cert.vinculo_ref,cert.version,cert.huella_sha256)=
      (certificado_actual.certificado_der_sha256,certificado_actual.vinculo_ref,certificado_actual.version,certificado_actual.huella_sha256)
 JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 audit ON audit.auditoria_ref=e.auditoria
 WHERE a.perfil_activo_ref=e.perfil AND a.principal_id=e.persona AND a.version=1
   AND a.documento->>'estado'='activa'
   AND a.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
   AND (a.documento->>'vigente_desde')::timestamptz<=clock_timestamp()
   AND (a.documento->>'vigente_hasta')::timestamptz>clock_timestamp()
   AND r.rol_id='ct_cargo_'||e.rol||'_firma_dev' AND r.version=1 AND r.documento->>'estado'='publicada'
   AND control.estado='habilitada'
   AND perfil.persona_ref=e.persona AND perfil.estado='activo' AND perfil.vigente_desde<=clock_timestamp()
   AND perfil.vigente_hasta>=(a.documento->>'vigente_hasta')::timestamptz
   AND vinculo.cuenta_ref=e.cuenta AND vinculo.persona_ref=e.persona AND vinculo.perfil_ref=e.perfil
   AND vinculo.estado='activo' AND vinculo.vigente_desde<=clock_timestamp()
   AND vinculo.vigente_hasta>=(a.documento->>'vigente_hasta')::timestamptz
   AND cuenta.estado='activo' AND cuenta.vigente_desde<=clock_timestamp()
   AND cuenta.vigente_hasta>=(a.documento->>'vigente_hasta')::timestamptz
   AND cert.certificado_der_sha256=e.certificado AND cert.vinculo_ref=e.vinculo_certificado
   AND cert.persona_ref=e.persona AND cert.cuenta_ref=e.cuenta AND cert.estado='vigente'
   AND cert.vigente_desde<=clock_timestamp() AND cert.vigente_hasta>=(a.documento->>'vigente_hasta')::timestamptz
   AND cert.recibo_ref=e.recibo AND cert.decision_ref=e.decision AND cert.auditoria_ref=e.auditoria
) fila;"""


def preparar(c):
    """Devuelve cuatro filas acreditadas sin crear una concesión nueva."""
    semilla = _fuente(c, c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')
    firmantes_meta = _fuente(c, c.carpeta / 'firmantes/metadata.json')
    actos = _fuente(c, c.carpeta / 'actos/metadata.json')
    if (firmantes_meta.get('copia_ref') != c.copia.nombre or actos.get('copia_ref') != c.copia.nombre
            or actos.get('actos_confirmados') != 14 or len(actos.get('recibos', [])) != 14
            or set(semilla.get('referencias', {})) != set(ROLES)
            or len(semilla.get('firmantes', [])) != 4):
        raise ValueError('Semilla y catorce actos propios incompletos')
    roles_v1 = None
    for nombre, huella, plan, cantidad in (
            ('roles_acuse', 'roles_acuse_sha256', 'roles_plan', 16),
            ('consulta_v2_acuse', 'consulta_v2_acuse_sha256', 'consulta_v2_plan', 4)):
        ruta = Path(firmantes_meta[nombre])
        if (not ruta.resolve().is_relative_to((c.carpeta / 'firmantes').resolve())
                or c.sha(ruta) != firmantes_meta[huella]):
            raise ValueError('Recibo AUT53 alterado o ajeno')
        cotejo = _roles(ruta, firmantes_meta[plan], cantidad)
        if cantidad == 16:
            roles_v1 = cotejo
    perfiles = {x.get('perfil_firma_ref'): x for x in semilla['firmantes']}
    if len(perfiles) != 4:
        raise ValueError('Los perfiles de firma no son únicos')
    valores, recibos = [], {}
    for rol in ROLES:
        ref = semilla['referencias'][rol]
        perfil = ref['perfil']['firma']
        firmante = perfiles.get(perfil)
        ruta = c.carpeta / 'actos/recibos' / (rol + '-certificado.json')
        respuesta = _fuente(c, ruta)
        recibo = respuesta.get('recibo', {})
        if (not firmante or firmante.get('persona_ref') != ref['persona_ref']
                or firmante.get('cuenta_ref') != ref['cuenta_ref']
                or firmante.get('certificado_sha256') != ref['certificado_sha256']
                or not HEX.fullmatch(ref['certificado_sha256'])
                or recibo.get('estado') != 'vigente' or recibo.get('version') != 1
                or recibo.get('certificado_der_sha256') != ref['certificado_sha256']
                or recibo.get('organizacion_destino', {}).get('organizacion_ref') != semilla['organizacion_ca_ref']
                or not all(recibo.get(k) for k in ('vinculo_ref', 'recibo_ref', 'decision_ref', 'auditoria_ref'))):
            raise ValueError('Certificado sin recibo nominal propio')
        recibos[rol] = recibo
        valores.append((rol, perfil, ref['persona_ref'], ref['cuenta_ref'], ref['certificado_sha256'],
                        ref['vinculo']['firma'], recibo['vinculo_ref'], recibo['recibo_ref'],
                        recibo['decision_ref'], recibo['auditoria_ref']))
    if any(len({fila[i] for fila in valores}) != 4 for i in range(1, 10)):
        raise ValueError('Identidades o certificados compartidos')
    sid = c.copia.leer().get('sid_copia')
    actual = c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();').strip()
    if not isinstance(sid, str) or sid != actual:
        raise ValueError('PostgreSQL no corresponde a la copia')
    filas = json.loads(c.consultar(_consulta(valores)))
    if not isinstance(filas, list) or len(filas) != 4 or {f.get('rol') for f in filas} != set(ROLES):
        raise ValueError('Cuatro admisiones vigentes ausentes')
    resumen = []
    claves = ('perfil_ref', 'asignacion_ref', 'asignacion_version', 'asignacion_sha256',
              'rol_version_ref', 'rol_sha256', 'control_revision', 'control_sha256',
              'persona_ref', 'cuenta_ref', 'certificado_sha256', 'vinculo_certificado_ref',
              'recibo_ref', 'decision_ref', 'auditoria_ref')
    for fila in filas:
        rol = fila['rol']
        recibo = recibos[rol]
        durable = fila.pop('recibo_db', None)
        publicado = roles_v1.get(fila.get('rol_version_ref'))
        if (not publicado or publicado['version_rol_sha256'] != fila.get('rol_sha256')
                or publicado['control_sha256'] != fila.get('control_sha256')
                or recibo.get('estado_replay') not in ('nueva', 'replay')
                or not _proyeccion({k: v for k, v in recibo.items() if k != 'estado_replay'}, durable)
                or any(k not in fila for k in claves)):
            raise ValueError('Recibo durable divergente del acto nominal')
        resumen.append({k: fila[k] for k in claves})
    destino = c.carpeta / 'admisiones'
    c.guardar(destino / 'admisiones-bd.json', resumen)
    meta = {'version': 1, 'copia_ref': c.copia.nombre, 'sid_copia': sid,
            'filas': 4, 'admisiones_sha256': c.sha(destino / 'admisiones-bd.json'),
            'certificados': [{'rol': rol, 'recibo_ref': recibos[rol]['recibo_ref'],
                              'respuesta_sha256': c.sha(c.carpeta / 'actos/recibos' / (rol + '-certificado.json'))}
                             for rol in ROLES], 'alcance': 'cotejo de lectura; semilla técnica AUT53'}
    c.guardar(destino / 'metadata.json', meta)
    return meta
