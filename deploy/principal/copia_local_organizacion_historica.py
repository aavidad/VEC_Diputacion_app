"""Consulta de Organización histórica para «Llamar a la siguiente persona» (F1-103).

La ficha del expediente llama a la siguiente persona de la bolsa constituida.
Para emitir, Contratación consulta el puesto de la RPT en la Organización
histórica de Personal con el cliente M10 y el certificado de la técnica de
RRHH. En una copia nueva faltaba todo: el permiso, el montaje de la consulta
en el proceso combinado y el cliente. Contratación respondía 409.

Este módulo deja la copia como la describe
`organizacion_historica_combinado_20261015.md`, solo con datos ficticios:

- `sembrar` (antes de compilar): rol `consulta_organizacion_historica_rrhh` v1
  con la concesión de la fuente versionada, perfil nuevo de la técnica con su
  vínculo de contexto y su vínculo corporativo (gemelos de los de su perfil de
  Bolsa), asignación con organismo y unidades exactos, catálogo de cuatro
  motivos, cuatro LOGIN propios, cliente M10 y selector de gobierno OH. Es
  semilla de copia con procedencia no autoritativa, como los demás roles
  sembrados; no es un acto de Administración (F0-37).
- `preparar_material` (con la aplicación ya publicada): deriva la clave OH con
  `vec-preparar-material-interno`, escribe el montaje combinado y enciende la
  consulta. El guion reinicia después la aplicación.

La consulta conserva el contexto privado por cuenta: si cambia la versión del
perfil, hay que volver a preparar el material.
"""
import hashlib
import json
import os
import re
import secrets
import subprocess

FUENTE = 'deploy/principal/fuentes/personal_organizacion_historica_rrhh_v1.json'
SELECCION = 'deploy/principal/candidata_hz39/datos/seleccion.json'
ROL_ID = 'consulta_organizacion_historica_rrhh'
ROL_REF = 'rol:' + ROL_ID + ':v1'
# Nombre del rol como dato de la semilla; la pantalla no lo muestra.
ROL_NOMBRE = 'Consulta de la Organización histórica por RRHH'
# Unidades de los puestos del recorrido (caso 4.0: centro 520). Otra unidad se deniega.
UNIDADES = ('unidad:centro:520',)
CATALOGO_MOTIVOS = 'motivos_organizacion_historica_copia'
USOS_MOTIVO = ('consulta', 'denegado', 'entrada_invalida', 'no_disponible')
EMISOR = 'identidad:desarrollo:no-autoritativa'
PUBLICADOR = 'seguridad:desarrollo:no-autoritativa'
DIAS_ASIGNACION = 89
CERO_GO = '0001-01-01T00:00:00Z'
DIRECTORIO = 'organizacion-historica'
CLIENTE_M10 = 'cliente_m10_ct.json'
SEMILLA = 'organizacion-historica-semilla.json'
REFERENCIA = re.compile(r'(per|cta|prf|vca|vcr|prc|org)_[A-Za-z0-9_-]{22,128}')


def _sha(datos):
    return hashlib.sha256(datos if isinstance(datos, bytes) else datos.encode()).hexdigest()


def _json_privado(ruta):
    if ruta.is_symlink() or not ruta.is_file() or ruta.stat().st_mode & 0o077:
        raise ValueError(f'Material privado ausente o inseguro: {ruta.name}')
    return json.loads(ruta.read_text(encoding='utf-8'))


def _escribir(ruta, contenido):
    """Fichero nuevo 0600; nunca sobrescribe ni sigue enlaces."""
    descriptor = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, 'w', encoding='utf-8') as salida:
        salida.write(contenido)
        salida.flush()
        os.fsync(salida.fileno())


def _literal(valor):
    texto = str(valor)
    if '\x00' in texto or '\\' in texto:
        raise ValueError('Valor SQL no admitido')
    return "'" + texto.replace("'", "''") + "'"


def _json_sql(valor):
    return _literal(json.dumps(valor, ensure_ascii=False, separators=(',', ':'))) + '::jsonb'


def _consulta(copia, sql):
    resultado = subprocess.run(
        ['docker', 'exec', '-i', '-u', 'postgres', copia.nombre, 'psql', '-XAtq', '-v', 'ON_ERROR_STOP=1',
         '-U', 'postgres', '-d', 'postgres'], input=('SET search_path=pg_catalog;\n' + sql).encode(),
        stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=True, timeout=60)
    return [fila for fila in resultado.stdout.decode().splitlines() if fila]


def concesion(repo):
    """Concesión única de la fuente versionada, con los campos en el orden de Go."""
    fuente = json.loads((repo / FUENTE).read_text(encoding='utf-8'))
    if _sha(fuente['entradas_canon']) != fuente['huella_sha256'] or len(fuente['entradas']) != 1:
        raise ValueError('La fuente de Organización histórica no cuadra con su huella')
    c = fuente['entradas'][0]['concesion']
    salida = {clave: c[clave] for clave in ('accion', 'modulo_id', 'tipo_recurso', 'finalidades',
                                            'garantia_minima', 'campos_permitidos')}
    if c.get('obligaciones'):
        raise ValueError('La concesión de Organización histórica no lleva obligaciones')
    return salida


def organismo(repo):
    valor = json.loads((repo / SELECCION).read_text(encoding='utf-8'))['organismo_ref']
    if not REFERENCIA.fullmatch(valor):
        raise ValueError('Organismo del caso 4.0 inválido')
    return valor


def identidad(copia):
    """Cuenta, persona, perfil de Bolsa y vínculos de la técnica del certificado mtls/cliente.crt."""
    datos = _json_privado(copia.material / 'identidad/identidad.json')
    bback = _json_privado(copia.material / 'identidad/bolsa-bback.json')
    der = subprocess.run(['openssl', 'x509', '-in', str(copia.material / 'mtls/cliente.crt'), '-outform', 'DER'],
                         stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=True, timeout=30).stdout
    perfil = bback.get('perfil_ref', '')
    if (datos.get('certificate_sha256') != _sha(der) or bback.get('certificado_sha256') != _sha(der)
            or datos.get('autoridad') != 'no_autoritativo' or not REFERENCIA.fullmatch(perfil)):
        raise ValueError('La identidad de la técnica no corresponde al certificado de la copia')
    filas = _consulta(copia, f"""SELECT jsonb_build_object('contexto',to_jsonb(v),'corporativo',to_jsonb(c))::text
      FROM vec_contexto_actor_v1.vinculo_contexto_actual a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
      JOIN vec_contexto_actor_v1.vinculo_corporativo_actual ca
        ON ca.cuenta_ref=v.cuenta_ref AND ca.perfil_ref=v.perfil_ref
       AND ca.superficie='interna_corporativa' AND ca.uso='consulta_rrhh'
      JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones c
        ON c.vinculo_corporativo_ref=ca.vinculo_corporativo_ref AND c.version=ca.version
      WHERE v.perfil_ref={_literal(perfil)} AND v.estado='activo' AND c.estado='activo'
        AND c.vinculo_contexto_ref=v.vinculo_ref AND c.vinculo_contexto_version=v.version
        AND clock_timestamp()>=v.vigente_desde AND clock_timestamp()<v.vigente_hasta
        AND clock_timestamp()>=c.vigente_desde AND clock_timestamp()<c.vigente_hasta;""")
    if len(filas) != 1:
        raise ValueError('La técnica no tiene un vínculo corporativo de consulta inequívoco')
    fila = json.loads(filas[0])
    return fila['contexto'], fila['corporativo']


def _documentos(repo, persona, perfil, asignacion, organismo_ref):
    """Documentos sin instantes: la transacción añade la hora de la base."""
    rol = {'rol_id': ROL_ID, 'version': 1, 'nombre': ROL_NOMBRE, 'estado': 'publicada',
           'concesiones': [concesion(repo)], 'publicada_por': PUBLICADOR, 'retirada_en': CERO_GO}
    control = {'version_rol_ref': ROL_REF, 'revision': 1, 'estado': 'habilitada',
               'actualizado_por': PUBLICADOR}
    asignacion_doc = {'asignacion_id': asignacion, 'version': 1, 'perfil_activo_ref': perfil,
                      'principal_id': persona, 'version_rol_ref': ROL_REF, 'estado': 'activa',
                      'ambitos': [{'clave': 'organismo_ref', 'valores': [organismo_ref]},
                                  {'clave': 'unidad_clave', 'valores': list(UNIDADES)}],
                      'emitida_por': EMISOR}
    return rol, control, asignacion_doc


def motivos():
    """Cuatro motivos con clave opaca estable; las fechas llevan seis decimales."""
    claves = {uso: 'motivo_' + _sha('vec:copia:oh:motivo:' + uso)[:32] for uso in USOS_MOTIVO}
    huella = _sha('vec:copia:oh:catalogo:' + ','.join(claves[u] for u in USOS_MOTIVO))
    referencias = {uso: {'catalogo_id': CATALOGO_MOTIVOS, 'catalogo_version': 1,
                         'catalogo_huella_sha256': huella, 'entrada_clave': clave}
                   for uso, clave in claves.items()}
    entradas = [{'clave': clave, 'vigente_desde': '2026-10-01T00:00:00.000000Z', 'vigente_hasta': None}
                for clave in sorted(claves.values())]
    return huella, entradas, referencias


def sql_semilla(repo, contexto, corporativo, refs, organismo_ref):
    """Una transacción: rol, perfil y vínculos gemelos, asignación y motivos."""
    persona, perfil, asignacion = contexto['persona_ref'], refs['perfil'], refs['asignacion']
    rol, control, asignacion_doc = _documentos(repo, persona, perfil, asignacion, organismo_ref)
    huella_motivos, entradas, _ = motivos()
    vinculo, corp = refs['vinculo_contexto'], refs['vinculo_corporativo']
    columnas_contexto = ('procedencia_ref', 'procedencia_version', 'procedencia_huella_sha256',
                         'procedencia_autoridad')
    procedencia = ','.join(_literal(contexto[c]) for c in columnas_contexto)
    copia_corp = ('cuenta_ref', 'cuenta_version', 'persona_ref', 'persona_version', 'organizacion_ref',
                  'organizacion_version', 'organizacion_procedencia_ref', 'organizacion_procedencia_version',
                  'organizacion_procedencia_huella_sha256', 'organizacion_procedencia_autoridad', 'superficie',
                  'uso', 'procedencia_ref', 'procedencia_version', 'procedencia_huella_sha256',
                  'procedencia_autoridad', 'vigente_desde', 'vigente_hasta')
    if corporativo['organizacion_ref'] != organismo_ref or corporativo['persona_ref'] != persona:
        raise ValueError('El vínculo corporativo de la técnica no es del organismo del recorrido')
    return f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL search_path=pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec:copia:organizacion-historica:v1',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $contexto$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref={_literal(perfil)}) THEN
  RAISE EXCEPTION 'semilla OH: el perfil ya existe' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
  (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,
   procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES({_literal(perfil)},1,{_literal(persona)},{procedencia},'activo',
  {_literal(contexto['vigente_desde'])}::timestamptz,{_literal(contexto['vigente_hasta'])}::timestamptz);
 INSERT INTO vec_contexto_actor_v1.perfil_actual(perfil_ref,version) VALUES({_literal(perfil)},1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
  (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES({_literal(vinculo)},1,{_literal(contexto['cuenta_ref'])},{_literal(perfil)},{_literal(persona)},
  {procedencia},'activo',{_literal(contexto['vigente_desde'])}::timestamptz,
  {_literal(contexto['vigente_hasta'])}::timestamptz);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual(vinculo_ref,version) VALUES({_literal(vinculo)},1);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
  (vinculo_corporativo_ref,version,perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,
   estado,{','.join(copia_corp)})
 VALUES({_literal(corp)},1,{_literal(perfil)},1,{_literal(vinculo)},1,'activo',
  {','.join(_literal(corporativo[c]) + ('::timestamptz' if c.startswith('vigente') else '') for c in copia_corp)});
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual
  (cuenta_ref,superficie,uso,vinculo_corporativo_ref,version,perfil_ref)
 VALUES({_literal(corporativo['cuenta_ref'])},'interna_corporativa','consulta_rrhh',{_literal(corp)},1,{_literal(perfil)});
END $contexto$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $rol$
DECLARE ahora timestamptz := clock_timestamp(); rol jsonb; control jsonb; doc jsonb; t text; h text;
BEGIN
 -- Instantes de Go sin fracción; la vigencia de la asignación es de 89 días.
 t := to_char(date_trunc('second',ahora) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 h := to_char(date_trunc('second',ahora+interval '{DIAS_ASIGNACION} days') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id={_literal(ROL_ID)})
  OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref={_literal(perfil)}) THEN
  RAISE EXCEPTION 'semilla OH: rol o asignación ya presentes' USING ERRCODE='55000'; END IF;
 rol := {_json_sql(rol)} || jsonb_build_object('publicada_en',t);
 control := {_json_sql(control)} || jsonb_build_object('actualizado_en',t);
 doc := {_json_sql(asignacion_doc)}
  || jsonb_build_object('emitida_en',t,'vigente_desde',t,'vigente_hasta',h,'revocada_en',{_literal(CERO_GO)});
 INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
 VALUES({_literal(ROL_REF)},{_literal(ROL_ID)},1,
  encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol),'UTF8')),'hex'),
  date_trunc('second',ahora),rol);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol
  (version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 VALUES({_literal(ROL_REF)},1,'habilitada',
  encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex'),
  date_trunc('second',ahora),control);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
  (version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
 VALUES({_literal(ROL_REF)},1,date_trunc('second',ahora),{_literal(PUBLICADOR)},'acto:ct:desarrollo:control-rol:v1');
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES('asignacion:'||{_literal(asignacion)}||':v1',{_literal(asignacion)},1,{_literal(perfil)},{_literal(persona)},
  {_literal(ROL_REF)},encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),
  date_trunc('second',ahora),doc);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual
 VALUES({_literal(perfil)},'asignacion:'||{_literal(asignacion)}||':v1',ahora,{_literal(EMISOR)},
  'acto:ct:desarrollo:asignacion:v1');
END $rol$;
RESET ROLE;
DO $motivos$
DECLARE secuencia bigint; evento text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_catalogo_publicado
   WHERE catalogo_id={_literal(CATALOGO_MOTIVOS)} AND catalogo_version=1) THEN
  SELECT ultima_secuencia+1 INTO STRICT secuencia
   FROM vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id=true FOR UPDATE;
  evento := encode(sha256(convert_to({_literal(huella_motivos)}||':'||secuencia::text,'UTF8')),'hex');
  IF vec_autorizacion.publicar_motivos_autorizacion_v2('evento_'||substr(evento,1,32),secuencia,evento,
    {_literal(CATALOGO_MOTIVOS)},1,{_literal(huella_motivos)},'2026-10-01T00:00:00Z'::timestamptz,
    {_json_sql(entradas)}) IS NOT TRUE THEN
   RAISE EXCEPTION 'semilla OH: publicación de motivos rechazada' USING ERRCODE='55000'; END IF;
 END IF;
 IF (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada
   WHERE catalogo_id={_literal(CATALOGO_MOTIVOS)} AND catalogo_version=1)<>{len(USOS_MOTIVO)} THEN
  RAISE EXCEPTION 'semilla OH: catálogo de motivos distinto' USING ERRCODE='55000'; END IF;
END $motivos$;
COMMIT;
"""


def sembrar(copia, repo):
    """Semilla y configuración previa al primer arranque. Devuelve la huella de la semilla."""
    from copia_local import guardar, sha256
    from copia_local_conexiones import preparar_organizacion_historica

    estado = copia.leer()
    if estado['preparada'] or not copia.existe():
        raise ValueError('La Organización histórica se siembra en una copia nueva, antes del arranque')
    organismo_ref = organismo(repo)
    contexto, corporativo = identidad(copia)
    refs = {'perfil': 'prf_' + secrets.token_hex(16), 'vinculo_contexto': 'vca_' + secrets.token_hex(16),
            'vinculo_corporativo': 'vcr_' + secrets.token_hex(16), 'asignacion': 'asg_' + secrets.token_hex(16)}
    privado = copia.raiz / DIRECTORIO
    privado.mkdir(mode=0o700)
    semilla = {'tipo': 'semilla_sintetica_de_copia', 'gobierno_real_acreditado': False,
               'rol_ref': ROL_REF, 'persona_ref': contexto['persona_ref'], 'cuenta_ref': contexto['cuenta_ref'],
               'perfil_ref': refs['perfil'], 'asignacion_id': refs['asignacion'],
               'organismo_ref': organismo_ref, 'unidades': list(UNIDADES),
               'fuente_sha256': _sha((repo / FUENTE).read_bytes())}
    # Retenida antes del efecto; crear nunca reejecuta una instancia existente.
    _escribir(copia.raiz / SEMILLA, json.dumps(semilla, ensure_ascii=False, indent=2) + '\n')
    copia.sql_bytes(sql_semilla(repo, contexto, corporativo, refs, organismo_ref).encode(),
                    'organizacion-historica-semilla.log')
    conexiones = preparar_organizacion_historica(copia)
    _escribir(privado / 'conexiones.json', json.dumps(conexiones, sort_keys=True) + '\n')
    entorno = copia.raiz / 'entorno_modulos.json'
    if sha256(entorno) != estado['entorno_modulos_sha256']:
        raise ValueError('El entorno de módulos cambió durante la semilla')
    nuevo = json.loads(entorno.read_text())
    _escribir(privado / 'motivos-oh.dsn', conexiones['dsn_motivos'] + '\n')
    entrada = privado / 'entrada'
    entrada.mkdir(mode=0o700)
    consulta = motivos()[2]['consulta']
    _escribir(entrada / 'organizacion_historica_v3.json', json.dumps({
        'version': 1, 'catalogo_motivos': CATALOGO_MOTIVOS, 'motivo_consulta': consulta,
        'capacidad': {'archivo': 'organizacion_historica_consulta.hmac'},
        # unidad_clave vacía: decide la asignación con su lista exacta.
        'contextos': {contexto['cuenta_ref']: {'perfil_activo_ref': refs['perfil'], 'perfil_version': 1,
                                               'organismo_ref': organismo_ref, 'unidad_clave': ''}}},
        ensure_ascii=False) + '\n')
    _escribir(copia.raiz / CLIENTE_M10, json.dumps({
        'version': 1, 'organismo_ref': organismo_ref, 'origen': f"https://localhost:{estado['puerto_app']}",
        'directorio': str(copia.material), 'autoridad_ca': 'ca/ca.crt',
        'certificado_cliente': 'mtls/cliente.crt', 'clave_cliente': 'mtls/cliente.key',
        'maximo_bytes_pagina': 1048576}) + '\n')
    nuevo.update(VEC_ORGANIZACION_HISTORICA_GOBIERNO_ENABLED='true',
                 VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH=str(copia.raiz / CLIENTE_M10))
    guardar(entorno, nuevo)
    estado['entorno_modulos_sha256'] = sha256(entorno)
    guardar(copia.estado, estado)
    return _sha((copia.raiz / SEMILLA).read_bytes())


def preparar_material(copia, preparador):
    """Clave OH derivada y montaje combinado; requiere la aplicación publicada."""
    from copia_local import guardar, sha256
    from copia_local_conexiones import crear_preparador_gobierno, retirar_preparador_gobierno

    privado = copia.raiz / DIRECTORIO
    salida = privado / 'material'
    gobierno = privado / 'gobierno.dsn'
    try:
        _escribir(gobierno, crear_preparador_gobierno(copia) + '\n')
        with (copia.raiz / 'organizacion-historica-material.log').open('ab') as log:
            subprocess.run([str(preparador), '-material-idempotencia', str(copia.material / 'idempotencia'),
                            '-organizacion-historica-config',
                            str(privado / 'entrada/organizacion_historica_v3.json'),
                            '-organizacion-historica-motivos-dsn', str(privado / 'motivos-oh.dsn'),
                            '-dsn-archivo', str(gobierno), '-salida', str(salida)],
                           stdout=log, stderr=subprocess.STDOUT, check=True, timeout=120)
    finally:
        gobierno.unlink(missing_ok=True)
        retirar_preparador_gobierno(copia)
    conexiones = _json_privado(privado / 'conexiones.json')
    referencias = motivos()[2]
    _escribir(salida / 'organizacion_historica_combinado.json', json.dumps({
        'version': 1, **{campo: conexiones[campo] for campo in sorted(conexiones)},
        'motivo_denegado': referencias['denegado'], 'motivo_entrada_invalida': referencias['entrada_invalida'],
        'motivo_no_disponible': referencias['no_disponible']}, ensure_ascii=False) + '\n')
    estado = copia.leer()
    entorno = copia.raiz / 'entorno_modulos.json'
    if sha256(entorno) != estado['entorno_modulos_sha256']:
        raise ValueError('El entorno de módulos cambió durante el material OH')
    nuevo = json.loads(entorno.read_text())
    nuevo.update(VEC_ORGANIZACION_HISTORICA_CONSULTA_ENABLED='true',
                 VEC_ORGANIZACION_HISTORICA_MATERIAL_DIR=str(salida))
    guardar(entorno, nuevo)
    estado['entorno_modulos_sha256'] = sha256(entorno)
    estado['organizacion_historica_material_sha256'] = _sha(
        (salida / 'organizacion_historica_v3.json').read_bytes())
    guardar(copia.estado, estado)
