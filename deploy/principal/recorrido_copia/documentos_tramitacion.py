"""Perfil documental propio de la técnica original en una copia ficticia."""

import hashlib
import json
import re

from .personal import _referencia


REFERENCIA = re.compile(r'[A-Za-z0-9][A-Za-z0-9:._-]{2,127}\Z')
HUELLA = re.compile(r'[0-9a-f]{64}\Z')
ROL = 'rol:ct_cargo_tecnico_documentos_dev:v1'


def _literal(valor):
    if not isinstance(valor, str) or not REFERENCIA.fullmatch(valor):
        raise ValueError('Referencia documental no canónica')
    return "'" + valor + "'"


def _privado(c, ruta):
    if (ruta.is_symlink() or not ruta.is_file() or ruta.stat().st_mode & 0o077
            or not ruta.resolve().is_relative_to(c.material.resolve())
            and not ruta.resolve().is_relative_to(c.carpeta.resolve())):
        raise ValueError('Fuente documental privada ajena a la copia')
    return c.leer(ruta)


def _origen(c):
    identidad = _privado(c, c.material / 'identidad/identidad.json')
    bolsa = _privado(c, c.material / 'identidad/bolsa-bback.json')
    certificado = c.material / 'mtls/cliente.crt'
    if (certificado.is_symlink() or not certificado.is_file()
            or certificado.stat().st_mode & 0o077):
        raise ValueError('Certificado original inseguro')
    der = c.ejecutar(('/usr/bin/openssl', 'x509', '-in', certificado, '-outform', 'DER'),
                    'documentos-tramitacion-certificado.log').stdout
    huella = hashlib.sha256(der).hexdigest()
    if (identidad.get('version') != 1 or identidad.get('autoridad') != 'no_autoritativo'
            or identidad.get('roles') != ['tecnico_rrhh']
            or identidad.get('certificate_sha256') != huella
            or bolsa.get('sujeto') != identidad.get('subject')
            or bolsa.get('certificado_sha256') != huella
            or not isinstance(identidad.get('subject'), str) or not identidad['subject']):
        raise ValueError('La identidad documental no es la técnica original')
    original = bolsa.get('perfil_ref')
    _literal(original)
    sujeto = identidad['subject']
    derivados = (_referencia('per_', sujeto, huella, 'persona'),
                 _referencia('cta_', sujeto, huella, 'cuenta'),
                 _referencia('prf_', sujeto, huella, 'perfil-bolsa-bback-v1'))
    if original != derivados[2]:
        raise ValueError('El manifiesto de Bolsa no corresponde a la identidad original')
    sql = f"""SELECT jsonb_build_object('vinculo',to_jsonb(v),'perfil',to_jsonb(p),
      'cuenta',to_jsonb(pc),'persona',to_jsonb(pe))::text
    FROM vec_contexto_actor_v1.vinculo_contexto_actual va
    JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
    JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=v.perfil_ref
    JOIN vec_contexto_actor_v1.perfil_versiones p ON (p.perfil_ref,p.version)=(pa.perfil_ref,pa.version)
    JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
    JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones pc
      ON (pc.cuenta_ref,pc.version)=(ca.cuenta_ref,ca.version)
    JOIN vec_contexto_actor_v1.persona_actual pea ON pea.persona_ref=v.persona_ref
    JOIN vec_contexto_actor_v1.persona_versiones pe
      ON (pe.persona_ref,pe.version)=(pea.persona_ref,pea.version)
    WHERE v.perfil_ref={_literal(original)} AND p.persona_ref=v.persona_ref
      AND v.estado='activo' AND p.estado='activo' AND pc.estado='activo' AND pe.estado='activo'
      AND clock_timestamp() BETWEEN v.vigente_desde AND v.vigente_hasta
      AND clock_timestamp() BETWEEN p.vigente_desde AND p.vigente_hasta
      AND clock_timestamp() BETWEEN pc.vigente_desde AND pc.vigente_hasta
      AND clock_timestamp() BETWEEN pe.vigente_desde AND pe.vigente_hasta;"""
    respuesta = c.consultar(sql)
    try:
        fila = json.loads(respuesta)
    except (ValueError, TypeError) as exc:
        raise ValueError('Contexto original de la técnica ambiguo o ausente') from exc
    v = fila['vinculo']
    persona, cuenta = v['persona_ref'], v['cuenta_ref']
    if ((persona, cuenta, v['perfil_ref']) != derivados
            or fila['perfil']['persona_ref'] != persona or fila['persona']['persona_ref'] != persona
            or fila['cuenta']['cuenta_ref'] != cuenta
            or original in (persona, cuenta)):
        raise ValueError('Contexto original de la técnica divergente')
    for valor in (persona, cuenta, v['vinculo_ref'], v['procedencia_ref'], original):
        _literal(valor)
    if not HUELLA.fullmatch(v['procedencia_huella_sha256']):
        raise ValueError('Procedencia original sin huella')
    return identidad, bolsa, fila, huella


def _acuse(c, refs, rol):
    """Lectura de filas actuales; no confunde un fichero local con efecto SQL."""
    perfil, vinculo, asignacion = (refs[k] for k in ('perfil_ref', 'vinculo_ref', 'asignacion_id'))
    sql = f"""SELECT count(*) FROM vec_contexto_actor_v1.perfil_actual pa
    JOIN vec_contexto_actor_v1.perfil_versiones p
      ON p.perfil_ref=pa.perfil_ref AND p.version=pa.version
    JOIN vec_contexto_actor_v1.vinculo_contexto_actual va ON va.vinculo_ref={_literal(vinculo)}
    JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
      ON v.vinculo_ref=va.vinculo_ref AND v.version=va.version
    JOIN vec_autorizacion.asignacion_perfil_actual aa ON aa.perfil_activo_ref=p.perfil_ref
    JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=aa.asignacion_ref
    JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
    JOIN vec_autorizacion.rol_administrable_exacto_v1 ra ON ra.version_rol_ref=r.version_rol_ref
    JOIN vec_autorizacion.control_vigencia_version_rol_actual ca
      ON ca.version_rol_ref=r.version_rol_ref
    JOIN vec_autorizacion.control_vigencia_version_rol cv
      ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
    WHERE pa.perfil_ref={_literal(perfil)} AND pa.version=1 AND va.version=1
      AND p.persona_ref={_literal(refs['persona_ref'])} AND p.estado='activo'
      AND v.cuenta_ref={_literal(refs['cuenta_ref'])} AND v.persona_ref=p.persona_ref
      AND v.perfil_ref=p.perfil_ref AND v.estado='activo'
      AND a.asignacion_id={_literal(asignacion)} AND a.version=1
      AND a.principal_id=p.persona_ref AND a.perfil_activo_ref=p.perfil_ref
      AND a.asignacion_ref={_literal('asignacion:' + asignacion + ':v1')}
      AND a.documento->>'estado'='activa'
      AND a.documento->'ambitos'=jsonb_build_array(jsonb_build_object(
        'clave','organizacion_ref','valores',jsonb_build_array({_literal(refs['organizacion_ref'])})))
      AND a.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
      AND (a.documento->>'vigente_desde')::timestamptz<=clock_timestamp()
      AND (a.documento->>'vigente_hasta')::timestamptz>clock_timestamp()
      AND p.vigente_desde<=clock_timestamp() AND p.vigente_hasta>clock_timestamp()
      AND v.vigente_desde<=clock_timestamp() AND v.vigente_hasta>clock_timestamp()
      AND r.version_rol_ref={_literal(ROL)} AND r.huella_sha256={_literal(rol['version_rol_sha256'])}
      AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
      AND cv.huella_sha256={_literal(rol['control_sha256'])}
      AND ra.huella_sha256=r.huella_sha256 AND ra.unidad_requerida IS FALSE
      AND ra.ambitos_fijos=a.documento->'ambitos'
      AND NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones q
                     WHERE q.perfil_ref=p.perfil_ref AND q.version<>1)
      AND NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones q
                     WHERE q.vinculo_ref=v.vinculo_ref AND q.version<>1)
      AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil q
                     WHERE q.perfil_activo_ref=p.perfil_ref AND q.asignacion_ref<>a.asignacion_ref);"""
    return c.consultar(sql).strip() == '1'


def preparar(c, *, rol_documental, perfiles_reservados):
    """Siembra solo un perfil nuevo; AUT53 ya publicó su rol y control."""
    estado = c.copia.leer()
    sid = estado.get('sid_copia')
    if (not estado.get('recorrido_completo') or not isinstance(sid, str)
            or not re.fullmatch(r'[0-9]{10,25}', sid)
            or c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();').strip() != sid):
        raise ValueError('Documentos exige SID exacto de copia completa')
    if (not isinstance(rol_documental, dict)
            or rol_documental.get('version_rol_ref') != ROL
            or any(not HUELLA.fullmatch(rol_documental.get(k, ''))
                   for k in ('version_rol_sha256', 'control_sha256'))):
        raise ValueError('Rol documental AUT53 no acreditado')
    if (not isinstance(perfiles_reservados, set)
            or any(not isinstance(x, str) or not REFERENCIA.fullmatch(x)
                   for x in perfiles_reservados)):
        raise ValueError('Perfiles nominales no acreditados')
    identidad, bolsa, fila, huella = _origen(c)
    v = fila['vinculo']
    original = bolsa['perfil_ref']
    firmantes = _privado(c, c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')
    organizacion = c.parametros.get('organizacion_operativa_ref')
    _literal(organizacion)
    if firmantes.get('retirada_en') is None:
        raise ValueError('La autoridad de firmantes no tiene retirada')
    nominales = firmantes.get('firmantes')
    if not isinstance(nominales, list) or len(nominales) != 4:
        raise ValueError('Faltan las cuatro identidades nominales')
    perfiles_nominales = set()
    for nominal in nominales:
        if (not isinstance(nominal, dict)
                or any(nominal.get(k) in (v['persona_ref'], v['cuenta_ref'], huella)
                       for k in ('persona_ref', 'cuenta_ref', 'certificado_sha256'))):
            raise ValueError('La técnica original comparte identidad nominal')
        for clave in ('perfil_firma_ref', 'perfil_consulta_ct_ref',
                      'perfil_documentos_ref', 'perfil_consulta_circuito_ref'):
            valor = nominal.get(clave)
            _literal(valor)
            perfiles_nominales.add(valor)
    if len(perfiles_nominales) != 16:
        raise ValueError('Perfiles nominales compartidos')
    base = sid + ':' + huella + ':' + ROL
    refs = {'persona_ref': v['persona_ref'], 'cuenta_ref': v['cuenta_ref'],
            'organizacion_ref': organizacion,
            'perfil_ref': 'prf_' + hashlib.sha256((base + ':perfil').encode()).hexdigest()[:32],
            'vinculo_ref': 'vca_' + hashlib.sha256((base + ':vinculo').encode()).hexdigest()[:32],
            'asignacion_id': 'asg_' + hashlib.sha256((base + ':asignacion').encode()).hexdigest()[:32]}
    perfil_ct = _referencia('prf_', identidad['subject'], huella, 'perfil')
    if refs['perfil_ref'] in perfiles_reservados | perfiles_nominales | {original, perfil_ct}:
        raise ValueError('Perfil Documentos coincide con tramitación, firma o custodia')
    fuente = {'sid_copia': sid, 'certificado_sha256': huella, 'sujeto': identidad['subject'],
              'perfil_original_ref': original, 'referencias': refs,
              'rol': {k: rol_documental[k] for k in ('version_rol_ref', 'version_rol_sha256', 'control_sha256')}}
    carpeta = c.carpeta / 'documentos'
    c.guardar(carpeta / 'tramitacion-fuente.json', fuente)
    sql = (c.fuente / 'deploy/principal/recorrido_copia/datos/documentos_tramitacion.sql').read_text()
    reemplazos = {'SID': sid, 'ORIGINAL': original, 'PERSONA': v['persona_ref'],
                  'CUENTA': v['cuenta_ref'], 'VINCULO_ORIGINAL': v['vinculo_ref'],
                  'PERFIL': refs['perfil_ref'], 'VINCULO': refs['vinculo_ref'],
                  'ASIGNACION': refs['asignacion_id'],
                  'ORGANIZACION': organizacion,
                  'ROL_SHA': rol_documental['version_rol_sha256'],
                  'CONTROL_SHA': rol_documental['control_sha256'],
                  'RETIRADA': firmantes['retirada_en']}
    for clave, valor in reemplazos.items():
        sql = sql.replace('__' + clave + '__', _literal(valor))
    if '__' in sql:
        raise ValueError('Plantilla SQL documental incompleta')
    semilla = carpeta / 'tramitacion-semilla.sql'
    semilla_previa = semilla.exists()
    acreditada = _acuse(c, refs, rol_documental)
    if acreditada and not semilla_previa:
        raise ValueError('Postimagen documental sin semilla declarada')
    c.guardar(semilla, sql.encode())
    if not acreditada:
        c.sql(sql.encode(), 'documentos-tramitacion-semilla.log')
    if not _acuse(c, refs, rol_documental):
        raise ValueError('Semilla documental sin postimagen en la copia')
    cuenta = {'certificado_sha256': huella, 'sujeto': identidad['subject'],
              'cuenta_ref': v['cuenta_ref'], 'perfil_ref': refs['perfil_ref']}
    c.guardar(carpeta / 'tramitacion-acuse.json', {
        'sid_copia': sid, 'fuente_sha256': c.sha(carpeta / 'tramitacion-fuente.json'),
        'sql_sha256': c.sha(carpeta / 'tramitacion-semilla.sql'),
        'postimagen': 'perfil_vinculo_asignacion_rol_control_vigentes', 'cuenta': cuenta})
    return cuenta
