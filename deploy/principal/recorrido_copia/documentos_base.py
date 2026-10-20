"""Base privada de Documentos desde las cuentas nominales de esta copia."""

from pathlib import Path
from datetime import datetime, timedelta, timezone
import hashlib
import json
import re

from .firmantes import _roles

ROLES = ('tecnico', 'jefatura', 'diputacion', 'secretaria')
HEX = re.compile(r'[0-9a-f]{64}\Z')
REF = re.compile(r'[A-Za-z0-9][A-Za-z0-9:._-]{2,127}\Z')
CATALOGO = re.compile(r'[a-z][a-z0-9._-]{0,127}\Z')
CLAVE = re.compile(r'motivo_[0-9a-f]{32}\Z')
REF_MOTIVO = 'motivo:documentos:listado:copia-desarrollo'
CATALOGO_MOTIVO = 'motivos_documentos_copia_desarrollo'
CLAVE_MOTIVO = 'motivo_' + hashlib.sha256(REF_MOTIVO.encode()).hexdigest()[:32]
CUSTODIA = {
    'informe_definitivo': 'contratacion_temporal.informe_definitivo_firmado.v1',
    'resolucion': 'contratacion_temporal.resolucion_firmada.v1',
}


def _propio(c, ruta):
    ruta = Path(ruta)
    if (not ruta.is_absolute() or ruta.is_symlink() or not ruta.is_file()
            or ruta.stat().st_mode & 0o077
            or not (ruta.resolve().is_relative_to(c.carpeta.resolve())
                    or ruta.resolve().is_relative_to(c.material.resolve()))):
        raise ValueError('Fuente documental ajena a la copia')
    return c.leer(ruta)


def _literal(v):
    if not isinstance(v, str) or not REF.fullmatch(v):
        raise ValueError('Referencia documental inválida')
    return "'" + v + "'"


def _motivo(c):
    """Consulta o publica el dato aprobado con el publicador V2 ya instalado."""
    valor = c.parametros.get('documentos_motivo_listar')
    if valor is not None:
        if not isinstance(valor, dict) or set(valor) != {
                'catalogo_id', 'catalogo_version', 'catalogo_huella_sha256', 'entrada_clave'}:
            raise ValueError('documentos_motivo_listar incompleto')
    else:
        carpeta = c.carpeta / 'documentos'
        catalogo_ruta = carpeta / 'motivo-catalogo.json'
        if catalogo_ruta.exists():
            catalogo = _propio(c, catalogo_ruta)
        else:
            ahora = c.ahora()
            meta_firma = _propio(c, c.carpeta / 'firmantes/metadata.json')
            fin = datetime.fromisoformat(meta_firma['autoridad_hasta'].replace('Z', '+00:00'))
            if (fin.tzinfo is None or fin.utcoffset() != timedelta(0)
                    or not timedelta(days=60) <= fin - ahora <= timedelta(days=90)):
                raise ValueError('Motivo Documentos fuera de la vigencia de la copia')
            catalogo = {'referencia': REF_MOTIVO, 'catalogo_id': CATALOGO_MOTIVO,
                        'version': 1, 'entrada_clave': CLAVE_MOTIVO,
                        'accion': 'vec.documentos.listar',
                        'finalidad': 'consulta_documentos',
                        'vigente_desde': (ahora - timedelta(minutes=1)).strftime('%Y-%m-%dT%H:%M:%S.000000Z'),
                        'vigente_hasta': fin.astimezone(timezone.utc).strftime('%Y-%m-%dT%H:%M:%S.000000Z')}
            c.guardar(catalogo_ruta, catalogo)
        if (catalogo.get('referencia') != REF_MOTIVO or catalogo.get('catalogo_id') != CATALOGO_MOTIVO
                or catalogo.get('version') != 1 or catalogo.get('entrada_clave') != CLAVE_MOTIVO
                or catalogo.get('accion') != 'vec.documentos.listar'
                or catalogo.get('finalidad') != 'consulta_documentos'):
            raise ValueError('Motivo documental propio modificado')
        valor = {'catalogo_id': CATALOGO_MOTIVO, 'catalogo_version': 1,
                 'catalogo_huella_sha256': c.sha(catalogo_ruta), 'entrada_clave': CLAVE_MOTIVO}
        _publicar_motivo(c, catalogo, valor)
    if (not isinstance(valor['catalogo_id'], str) or not CATALOGO.fullmatch(valor['catalogo_id'])
            or type(valor['catalogo_version']) is not int or valor['catalogo_version'] < 1
            or not isinstance(valor['catalogo_huella_sha256'], str)
            or not HEX.fullmatch(valor['catalogo_huella_sha256'])
            or not isinstance(valor['entrada_clave'], str) or not CLAVE.fullmatch(valor['entrada_clave'])):
        raise ValueError('Referencia de motivo Documentos inválida')
    sql = ("SELECT count(*) FROM vec_autorizacion.motivo_v2_catalogo_publicado c "
           "JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version) "
           "WHERE c.catalogo_id=" + _literal(valor['catalogo_id']) +
           " AND c.catalogo_version=" + str(valor['catalogo_version']) +
           " AND c.catalogo_huella_publicada_sha256=" + _literal(valor['catalogo_huella_sha256']) +
           " AND e.entrada_clave=" + _literal(valor['entrada_clave']) +
           " AND e.vigente_desde<=clock_timestamp() "
           "AND (e.vigente_hasta IS NULL OR clock_timestamp()<e.vigente_hasta) "
           "AND NOT EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r "
           "WHERE r.catalogo_id=c.catalogo_id AND r.catalogo_version=c.catalogo_version);")
    if c.consultar(sql).strip() != '1':
        raise ValueError('Motivo Documentos no publicado, retirado o sin vigencia en la copia')
    if valor['catalogo_id'] == CATALOGO_MOTIVO:
        total = c.consultar("SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada WHERE catalogo_id='"
                           + CATALOGO_MOTIVO + "' AND catalogo_version=1;").strip()
        if total != '1':
            raise ValueError('Catálogo propio de motivos Documentos ambiguo')
    return valor


def _publicar_motivo(c, catalogo, valor):
    """Ensambla solo datos del recorrido; la función V2 instalada hace el efecto."""
    consulta = ("SELECT catalogo_huella_publicada_sha256 FROM "
                "vec_autorizacion.motivo_v2_catalogo_publicado WHERE catalogo_id='" +
                CATALOGO_MOTIVO + "' AND catalogo_version=1;")
    actual = c.consultar(consulta).strip()
    if actual:
        if actual != valor['catalogo_huella_sha256']:
            raise ValueError('Motivo Documentos publicado con otra huella')
        return
    marca = c.carpeta / 'documentos/motivo-publicacion-incierta.json'
    if marca.exists():
        raise ValueError('Publicación del motivo incierta: conciliar la operación original')
    sid = c.copia.leer().get('sid_copia')
    if not isinstance(sid, str) or not re.fullmatch(r'[1-9][0-9]{14,19}', sid):
        raise ValueError('SID de la copia ausente')
    entrada = json.dumps([{'clave': CLAVE_MOTIVO,
                          'vigente_desde': catalogo['vigente_desde'],
                          'vigente_hasta': catalogo['vigente_hasta']}], separators=(',', ':'))
    # Los valores son locales, validados o canonizados; no hay interpolación de entrada libre.
    sql = f"""BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $pre$ BEGIN
 IF current_user<>'postgres' OR (SELECT system_identifier::text FROM pg_catalog.pg_control_system())<>'{sid}'
 OR pg_catalog.to_regprocedure('vec_autorizacion.publicar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,timestamp with time zone,jsonb)') IS NULL
 THEN RAISE EXCEPTION 'origen de motivo Documentos inválido'; END IF;
END $pre$;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:documentos:motivo:listado:v1',0));
DO $publicar$ DECLARE secuencia bigint; evento_huella text; BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_catalogo_publicado
   WHERE catalogo_id='{CATALOGO_MOTIVO}' AND catalogo_version=1) THEN
  SELECT ultima_secuencia+1 INTO STRICT secuencia FROM vec_autorizacion.motivo_v2_checkpoint_origen
   WHERE control_id=true FOR UPDATE;
  evento_huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{valor['catalogo_huella_sha256']}'||':'||secuencia::text,'UTF8')),'hex');
  IF vec_autorizacion.publicar_motivos_autorizacion_v2('evento_'||pg_catalog.substr(evento_huella,1,32),
    secuencia,evento_huella,'{CATALOGO_MOTIVO}',1,'{valor['catalogo_huella_sha256']}',
    '{catalogo['vigente_desde']}'::timestamptz,'{entrada}'::jsonb) IS NOT TRUE
  THEN RAISE EXCEPTION 'motivo Documentos rechazado'; END IF;
 END IF;
 IF (SELECT catalogo_huella_publicada_sha256 FROM vec_autorizacion.motivo_v2_catalogo_publicado
  WHERE catalogo_id='{CATALOGO_MOTIVO}' AND catalogo_version=1) IS DISTINCT FROM '{valor['catalogo_huella_sha256']}'
 OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada WHERE catalogo_id='{CATALOGO_MOTIVO}'
  AND catalogo_version=1 AND entrada_clave='{CLAVE_MOTIVO}')<>1
 THEN RAISE EXCEPTION 'motivo Documentos divergente'; END IF;
END $publicar$;
COMMIT;
"""
    c.guardar(marca, {'catalogo_id': CATALOGO_MOTIVO,
                      'catalogo_huella_sha256': valor['catalogo_huella_sha256']})
    c.sql(sql.encode(), 'documentos-motivo-publicar.log')


def _cuentas(c, datos, roles):
    if set(datos.get('referencias', {})) != set(ROLES) or len(datos.get('firmantes', [])) != 4:
        raise ValueError('Cuatro firmantes propios requeridos para Documentos')
    por_perfil = {f.get('perfil_documentos_ref'): f for f in datos['firmantes']}
    if len(por_perfil) != 4:
        raise ValueError('Perfiles Documentos compartidos')
    cuentas, filas = [], []
    for rol in ROLES:
        r = datos['referencias'][rol]
        f = por_perfil.get(r['perfil']['documentos'])
        rol_ref = 'rol:ct_cargo_' + rol + '_documentos_dev:v1'
        if (not f or rol_ref not in roles or f.get('persona_ref') != r.get('persona_ref')
                or f.get('cuenta_ref') != r.get('cuenta_ref')
                or f.get('certificado_sha256') != r.get('certificado_sha256')
                or not HEX.fullmatch(r.get('certificado_sha256', ''))
                or r['perfil']['documentos'] == r['perfil']['firma']):
            raise ValueError('Cuenta documental sin rol AUT53 propio')
        persona, cuenta, perfil = r['persona_ref'], r['cuenta_ref'], r['perfil']['documentos']
        for valor in (persona, cuenta, perfil, rol_ref, r['vinculo']['documentos']):
            _literal(valor)
        cuentas.append({'certificado_sha256': r['certificado_sha256'], 'sujeto': persona,
                        'cuenta_ref': cuenta, 'perfil_ref': perfil})
        filas.append((rol, perfil, persona, cuenta, r['vinculo']['documentos'], rol_ref))
    if (len({x['sujeto'] for x in cuentas}) != 4 or
            len({x['cuenta_ref'] for x in cuentas}) != 4 or
            len({x['certificado_sha256'] for x in cuentas}) != 4):
        raise ValueError('Sujetos o certificados Documentos repetidos')
    values = ','.join('(' + ','.join(_literal(x) for x in fila) + ')' for fila in filas)
    sql = f"""WITH esperado(rol,perfil,persona,cuenta,vinculo,version_rol) AS (VALUES {values})
SELECT count(*) FROM esperado e
JOIN vec_autorizacion.asignacion_perfil_actual x ON x.perfil_activo_ref=e.perfil
JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=x.asignacion_ref
JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
JOIN vec_autorizacion.control_vigencia_version_rol cv ON (cv.version_rol_ref,cv.revision)=(ca.version_rol_ref,ca.revision)
JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=e.perfil
JOIN vec_contexto_actor_v1.perfil_versiones p ON (p.perfil_ref,p.version)=(pa.perfil_ref,pa.version)
JOIN vec_contexto_actor_v1.vinculo_contexto_actual va ON va.vinculo_ref=e.vinculo
JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v ON (v.vinculo_ref,v.version)=(va.vinculo_ref,va.version)
WHERE a.perfil_activo_ref=e.perfil AND a.principal_id=e.persona
AND a.documento->>'estado'='activa' AND (a.documento->>'vigente_desde')::timestamptz<=clock_timestamp()
AND (a.documento->>'vigente_hasta')::timestamptz>clock_timestamp()
AND r.version_rol_ref=e.version_rol AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
AND p.persona_ref=e.persona AND p.estado='activo' AND p.vigente_desde<=clock_timestamp()
AND v.cuenta_ref=e.cuenta AND v.persona_ref=e.persona AND v.perfil_ref=e.perfil
AND v.estado='activo' AND v.vigente_desde<=clock_timestamp();"""
    if c.consultar(sql).strip() != '4':
        raise ValueError('Asignaciones Documentos vigentes no acreditadas')
    # Documentos usa el resolvedor base, distinto del resolvedor nominal R5.
    # Su contrato sólo admite aquí la hoja RRHH original. Los otros tres
    # firmantes conservan sus perfiles de custodia y sus admisiones R5.
    identidad = _propio(c, c.material / 'identidad/identidad.json')
    tecnico = cuentas[0]
    if (identidad.get('version') != 1 or identidad.get('autoridad') != 'no_autoritativo'
            or identidad.get('roles') != ['tecnico_rrhh']
            or identidad.get('certificate_sha256') != tecnico['certificado_sha256']
            or not isinstance(identidad.get('subject'), str) or not identidad['subject']):
        raise ValueError('La cuenta Documentos no corresponde a la identidad RRHH original')
    return [{**tecnico, 'sujeto': identidad['subject']}]


def _custodia(c):
    ruta = c.fuente / 'internal/vec/adapters/conservacion/politicas_conservacion.v2.json'
    import json
    catalogo = json.loads(ruta.read_bytes())
    tipos = {x.get('tipo') for x in catalogo.get('politicas', [])
             if x.get('custodia') == 'firmado' and x.get('procedimiento') == 'contratacion_temporal.expediente'}
    if set(CUSTODIA.values()) != tipos:
        raise ValueError('Tipos reservados de custodia firmada distintos del catálogo')
    return {'documentos': dict(CUSTODIA)}


def preparar(c):
    """Conserva la base derivable; no crea LOGIN ni activa Documentos."""
    estado = c.copia.leer()
    if not estado.get('recorrido_completo') or c.consultar(
            'SELECT system_identifier::text FROM pg_catalog.pg_control_system();').strip() != estado.get('sid_copia'):
        raise ValueError('Documentos exige copia completa y SID propio')
    firmantes = _propio(c, c.carpeta / 'firmantes/metadata.json')
    actos = _propio(c, c.carpeta / 'actos/metadata.json')
    plan = _propio(c, c.carpeta / 'plan/metadata.json')
    firma = _propio(c, c.material / 'identidad/firma-vec.json')
    datos = _propio(c, c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')
    if (firmantes.get('copia_ref') != c.copia.nombre or actos.get('copia_ref') != c.copia.nombre
            or plan.get('copia_ref') != c.copia.nombre or actos.get('actos_confirmados') != 14
            or plan.get('plan_publicado') is not True or firma.get('autoridad') != 'no_autoritativo'
            or firma.get('firmantes') != datos.get('firmantes') or len(firma.get('admisiones', [])) != 4):
        raise ValueError('Firma y actos de esta copia incompletos')
    for clave, huella in (('roles_acuse', 'roles_acuse_sha256'),
                          ('consulta_v2_acuse', 'consulta_v2_acuse_sha256')):
        ruta = Path(firmantes[clave])
        if (not ruta.resolve().is_relative_to((c.carpeta / 'firmantes').resolve())
                or c.sha(ruta) != firmantes[huella]):
            raise ValueError('Acuse AUT53 ajeno a la copia')
    roles = _roles(firmantes['roles_acuse'], firmantes['roles_plan'], 16)
    _roles(firmantes['consulta_v2_acuse'], firmantes['consulta_v2_plan'], 4)
    cuentas = _cuentas(c, datos, roles)
    custodia = _custodia(c)
    motivo = _motivo(c)
    base = {'version': 1, 'autoridad': 'no_autoritativo', 'cuentas': cuentas,
            'motivos': {'listar': motivo},
            'almacen': {'tipo': 'ficheros', 'tamano_maximo': 32 * 1024 * 1024,
                        'retencion_minima_dias': 0},
            'custodia_firmado': custodia}
    destino = c.carpeta / 'documentos/base.json'
    c.guardar(destino, base)
    return destino
