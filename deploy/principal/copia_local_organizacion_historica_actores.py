"""Añade a la copia un segundo lector nominal de Organización histórica."""

import hashlib
import json
import os
import secrets
import subprocess
import tempfile

import copia_local_organizacion_historica as oh


def _segundo(copia, organismo):
    manifiesto = oh._json_privado(copia.material / 'identidad/tecnico-rrhh-adicional.json')
    identidad = oh._json_privado(copia.material / 'identidad/tecnico-rrhh-adicional-identidad.json')
    semilla = oh._json_privado(copia.raiz / 'rrhh-adicional-semilla.json')
    if (manifiesto.get('version') != 1 or manifiesto.get('autoridad') != 'no_autoritativo'
            or manifiesto.get('identidad') != 'identidad/tecnico-rrhh-adicional-identidad.json'
            or manifiesto.get('certificado') != 'mtls/tecnico-rrhh-adicional.crt'
            or identidad.get('subject') != manifiesto.get('sujeto')
            or identidad.get('roles') != ['tecnico_rrhh']
            or semilla.get('persona_ref') != manifiesto.get('persona_ref')
            or semilla.get('perfil_ref') != manifiesto.get('perfil_ref')
            or semilla.get('gobierno_real_acreditado') is not False
            or semilla.get('rol_ref') != 'rol:seguimiento_cese_ct_desarrollo:v2'):
        raise ValueError('La segunda identidad y su semilla CT no coinciden')
    for clave, prefijo in (('cuenta_ref', 'cta_'), ('persona_ref', 'per_'), ('perfil_ref', 'prf_')):
        if not oh.REFERENCIA.fullmatch(manifiesto.get(clave, '')) or not manifiesto[clave].startswith(prefijo):
            raise ValueError('Referencia de la segunda identidad inválida')
    certificado = copia.material / manifiesto['certificado']
    if certificado.is_symlink() or not certificado.is_file() or certificado.stat().st_mode & 0o077:
        raise ValueError('Certificado de la segunda identidad ausente o inseguro')
    der = subprocess.run(['openssl', 'x509', '-in', str(certificado), '-outform', 'DER'],
                         stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                         check=True, timeout=30).stdout
    if identidad.get('certificate_sha256') != hashlib.sha256(der).hexdigest():
        raise ValueError('La huella de la segunda identidad no corresponde a su certificado')
    if semilla.get('organizacion_ref') == organismo:
        raise ValueError('La asignación CT no debe sustituir el organismo gobernado de OH')
    return manifiesto


def _contexto_f1(copia, segundo, organismo):
    """Solo acepta vínculos activos ya proyectados para esa cuenta y persona."""
    sql = f"""SELECT jsonb_build_object('contexto',to_jsonb(v),'corporativo',to_jsonb(c))::text
      FROM vec_contexto_actor_v1.vinculo_contexto_actual a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
      JOIN vec_contexto_actor_v1.vinculo_corporativo_actual ca
        ON ca.cuenta_ref=v.cuenta_ref AND ca.perfil_ref=v.perfil_ref
       AND ca.superficie='interna_corporativa' AND ca.uso='consulta_rrhh'
      JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones c
        ON c.vinculo_corporativo_ref=ca.vinculo_corporativo_ref AND c.version=ca.version
      WHERE v.cuenta_ref={oh._literal(segundo['cuenta_ref'])}
        AND v.persona_ref={oh._literal(segundo['persona_ref'])}
        AND v.perfil_ref={oh._literal(segundo['perfil_ref'])}
        AND v.estado='activo' AND c.estado='activo'
        AND c.organizacion_ref={oh._literal(organismo)}
        AND c.vinculo_contexto_ref=v.vinculo_ref AND c.vinculo_contexto_version=v.version
        AND clock_timestamp()>=v.vigente_desde AND clock_timestamp()<v.vigente_hasta
        AND clock_timestamp()>=c.vigente_desde AND clock_timestamp()<c.vigente_hasta;"""
    filas = oh._consulta(copia, sql)
    if len(filas) != 1:
        raise ValueError('La segunda cuenta carece de vínculo F1 corporativo vigente e inequívoco')
    datos = json.loads(filas[0])
    return datos['contexto'], datos['corporativo']


def sql_segunda(contexto, corporativo, refs, organismo):
    """Inserta solo perfil y asignación de la segunda persona; reutiliza rol OH."""
    cuenta, persona = contexto['cuenta_ref'], contexto['persona_ref']
    perfil, vinculo, corp, asignacion = (refs[k] for k in
                                          ('perfil', 'vinculo_contexto', 'vinculo_corporativo', 'asignacion'))
    if (corporativo['cuenta_ref'] != cuenta or corporativo['persona_ref'] != persona
            or corporativo['organizacion_ref'] != organismo):
        raise ValueError('El vínculo F1 no corresponde a la segunda cuenta y organismo')
    procedencia = ','.join(oh._literal(contexto[k]) for k in
                          ('procedencia_ref', 'procedencia_version', 'procedencia_huella_sha256',
                           'procedencia_autoridad'))
    columnas = ('cuenta_ref', 'cuenta_version', 'persona_ref', 'persona_version', 'organizacion_ref',
                'organizacion_version', 'organizacion_procedencia_ref', 'organizacion_procedencia_version',
                'organizacion_procedencia_huella_sha256', 'organizacion_procedencia_autoridad', 'superficie',
                'uso', 'procedencia_ref', 'procedencia_version', 'procedencia_huella_sha256',
                'procedencia_autoridad', 'vigente_desde', 'vigente_hasta')
    valores = ','.join(oh._literal(corporativo[k]) + ('::timestamptz' if k.startswith('vigente') else '')
                      for k in columnas)
    doc = {'asignacion_id': asignacion, 'version': 1, 'perfil_activo_ref': perfil,
           'principal_id': persona, 'version_rol_ref': oh.ROL_REF, 'estado': 'activa',
           'ambitos': [{'clave': 'organismo_ref', 'valores': [organismo]},
                       {'clave': 'unidad_clave', 'valores': list(oh.UNIDADES)}],
           'emitida_por': oh.EMISOR}
    return f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec:copia:organizacion-historica:v1',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $contexto$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
   JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
   WHERE v.cuenta_ref={oh._literal(cuenta)} AND v.persona_ref={oh._literal(persona)}
     AND v.perfil_ref={oh._literal(contexto['perfil_ref'])} AND v.estado='activo')
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref={oh._literal(perfil)})
 THEN RAISE EXCEPTION 'segunda OH: proyección F1 ausente o perfil existente' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
  (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,
   procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES({oh._literal(perfil)},1,{oh._literal(persona)},{procedencia},'activo',
  {oh._literal(contexto['vigente_desde'])}::timestamptz,{oh._literal(contexto['vigente_hasta'])}::timestamptz);
 INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES({oh._literal(perfil)},1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
  (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES({oh._literal(vinculo)},1,{oh._literal(cuenta)},{oh._literal(perfil)},{oh._literal(persona)},
  {procedencia},'activo',{oh._literal(contexto['vigente_desde'])}::timestamptz,
  {oh._literal(contexto['vigente_hasta'])}::timestamptz);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES({oh._literal(vinculo)},1);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
  (vinculo_corporativo_ref,version,perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,
   estado,{','.join(columnas)})
 VALUES({oh._literal(corp)},1,{oh._literal(perfil)},1,{oh._literal(vinculo)},1,'activo',{valores});
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual
  (cuenta_ref,superficie,uso,vinculo_corporativo_ref,version,perfil_ref)
 VALUES({oh._literal(cuenta)},'interna_corporativa','consulta_rrhh',{oh._literal(corp)},1,{oh._literal(perfil)});
END $contexto$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $asignacion$
DECLARE ahora timestamptz := clock_timestamp(); doc jsonb; t text; h text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r
   JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
   WHERE r.version_rol_ref={oh._literal(oh.ROL_REF)} AND r.documento->>'estado'='publicada'
     AND c.estado='habilitada')
 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual
   WHERE perfil_activo_ref={oh._literal(perfil)})
 THEN RAISE EXCEPTION 'segunda OH: rol ausente o asignación existente' USING ERRCODE='55000'; END IF;
 t := to_char(date_trunc('second',ahora) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 h := to_char(date_trunc('second',ahora+interval '89 days') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 doc := {oh._json_sql(doc)} || jsonb_build_object('emitida_en',t,'vigente_desde',t,
   'vigente_hasta',h,'revocada_en',{oh._literal(oh.CERO_GO)});
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,
   huella_sha256,emitida_en,documento)
 VALUES('asignacion:'||{oh._literal(asignacion)}||':v1',{oh._literal(asignacion)},1,
  {oh._literal(perfil)},{oh._literal(persona)},{oh._literal(oh.ROL_REF)},
  encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),
  date_trunc('second',ahora),doc);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual
 VALUES({oh._literal(perfil)},'asignacion:'||{oh._literal(asignacion)}||':v1',ahora,
  {oh._literal(oh.EMISOR)},'acto:ct:desarrollo:asignacion:v1');
END $asignacion$;
COMMIT;
"""


def extender_contextos(copia, repo):
    """Invocar tras proyectar F1 del segundo RRHH y antes de derivar material OH."""
    estado = copia.leer()
    if not estado.get('organizacion_historica_semilla_sha256') or not estado.get('rrhh_adicional_semilla_sha256'):
        raise ValueError('Faltan las dos semillas declaradas de la copia')
    organismo = oh.organismo(repo)
    segundo = _segundo(copia, organismo)
    contexto, corporativo = _contexto_f1(copia, segundo, organismo)
    refs = {clave: prefijo + secrets.token_hex(16) for clave, prefijo in
            (('perfil', 'prf_'), ('vinculo_contexto', 'vca_'),
             ('vinculo_corporativo', 'vcr_'), ('asignacion', 'asg_'))}
    ruta = copia.raiz / oh.DIRECTORIO / 'entrada/organizacion_historica_v3.json'
    entrada = oh._json_privado(ruta)
    contextos = entrada.get('contextos')
    if not isinstance(contextos, dict) or len(contextos) != 1 or segundo['cuenta_ref'] in contextos:
        raise ValueError('La entrada OH inicial no tiene una sola cuenta original')
    semilla = {'tipo': 'semilla_sintetica_de_copia', 'gobierno_real_acreditado': False,
               'cuenta_ref': segundo['cuenta_ref'], 'persona_ref': segundo['persona_ref'],
               'perfil_ref': refs['perfil'], 'asignacion_id': refs['asignacion'],
               'rol_ref': oh.ROL_REF, 'organismo_ref': organismo, 'unidades': list(oh.UNIDADES)}
    ruta_semilla = copia.raiz / 'organizacion-historica-segunda-semilla.json'
    oh._escribir(ruta_semilla, json.dumps(semilla, ensure_ascii=False, indent=2) + '\n')
    copia.sql_bytes(sql_segunda(contexto, corporativo, refs, organismo).encode(),
                    'organizacion-historica-segunda-semilla.log')
    contextos[segundo['cuenta_ref']] = {'perfil_activo_ref': refs['perfil'], 'perfil_version': 1,
                                        'organismo_ref': organismo, 'unidad_clave': ''}
    contenido = json.dumps(entrada, ensure_ascii=False) + '\n'
    with tempfile.NamedTemporaryFile(mode='w', encoding='utf-8', dir=ruta.parent,
                                     prefix='.contextos-oh-', delete=False) as temporal:
        os.fchmod(temporal.fileno(), 0o600)
        temporal.write(contenido)
        temporal.flush()
        os.fsync(temporal.fileno())
        temporal_ruta = temporal.name
    os.replace(temporal_ruta, ruta)
    return hashlib.sha256(ruta_semilla.read_bytes()).hexdigest()
