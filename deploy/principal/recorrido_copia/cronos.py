"""Cronos de la persona empleada en una copia completa (F3-01, corte C1).

Prepara solo lo que la copia no trae: ocho LOGIN propios con un grupo cada
uno, las ocho filas técnicas de origen de consumo del ejecutor, la
configuración privada `identidad/cronos-empleado.json` y los selectores.
Usa el rol sintético de Cronos ya publicado en la referencia HZ37 y el catálogo
de motivos de CRN18. La persona técnica adicional de la copia recibe, como
semilla técnica declarada (no gobierno), un perfil propio, su empleado en
Personal y la asignación de ese rol. No instala SQL ni publica gobierno.
"""

from datetime import timedelta
import json
import re
import secrets
from urllib.parse import parse_qs, quote, urlencode, urlsplit

from copia_local_conexiones import _nombre_instancia, _privado, _rol
from copia_local_usuarios import sql as sql_logins


# Un LOGIN por conexión del fichero privado; mismos grupos que el material
# HZ37 de Cronos. Índices 130–137, posteriores a los de Dietas (110–120).
GRUPOS = {
    'dsn_registro_identidad': 'vec_identidad_sesiones_v1_registrador',
    'dsn_revalidacion_identidad': 'vec_identidad_sesiones_v1_revalidador',
    'dsn_contexto': 'vec_contexto_actor_v1_runtime',
    'dsn_fuente_autorizacion': 'vec_autorizacion_fuente',
    'dsn_registro_autorizacion': 'vec_autorizacion_registro',
    'dsn_motivos': 'vec_autorizacion_motivos_evaluador',
    'dsn_cronos': 'vec_cronos_v1_ejecutor',
    'dsn_cronos_auditor': 'vec_cronos_v1_auditor',
}
PRIMER_INDICE = 130

# Audiencia y operación de los ocho consumos del empleado (AD3-53 y AD3-70).
# AD172 exige una fila técnica por cada uno para el LOGIN ejecutor; no amplía
# la autorización de negocio, que sigue en el rol y su asignación.
CONSUMOS = (
    ('vec_cronos_v1.marcaje_propio.v1', 'cronos.marcaje.propio.registrar'),
    ('vec_cronos_v1.marcaje_remoto_disponibilidad.v1', 'cronos.marcaje.remoto.disponibilidad.consultar'),
    ('vec_cronos_v1.marcaje_remoto_recibo.v1', 'cronos.marcaje.remoto.recibo.consultar'),
    ('vec_cronos_v1.saldo_propio.consultar.v1', 'cronos.saldo.propio.consultar'),
    ('vec_cronos_v1.movimientos_propio.consultar.v1', 'cronos.movimientos.propio.consultar'),
    ('vec_cronos_v1.correccion_propia.solicitar.v1', 'cronos.correccion.solicitar'),
    ('vec_cronos_v1.permisos_propio.consultar.v1', 'cronos.permisos.propio.consultar'),
    ('vec_cronos_v1.permiso_propio.solicitar.v1', 'cronos.permiso.solicitar'),
)
CANAL = 'interna_corporativa'
PROCESO = 'vec-server'

ROL = 'rol:cronos_empleado_sintetico:v1'
CATALOGO = ('motivos_cronos_v1', 1, '82136314c0c8b9946b49debcb4497b62cbe9be7e0501c92dc4278ef528f5a822')
# Entradas de CRN18: consulta, actuación y denegación son distintas.
MOTIVO_CONSULTA = 'motivo_a28bf9dd817c4ef9ccba903beb358c3f'
MOTIVO_ACTUACION = 'motivo_1de68e77a196b06508201b9753248d16'
MOTIVO_DENEGADO = 'motivo_0dc9dd0b008c4ebf09a65d5a64563693'
MOTIVOS = {'saldo': MOTIVO_CONSULTA, 'disponibilidad': MOTIVO_CONSULTA,
           'recuperacion': MOTIVO_CONSULTA, 'movimientos': MOTIVO_CONSULTA,
           'permisos': MOTIVO_CONSULTA, 'marcaje': MOTIVO_ACTUACION,
           'correccion': MOTIVO_ACTUACION, 'permiso': MOTIVO_ACTUACION}
CANAL_REMOTO = {'politica_version_ref': 'politica:canal:cronos:v1',
                'canal_ref': 'portal-empleado-web', 'calidad_ref': 'mtls-certificado'}
# Persona empleada sintética del volcado: solo aporta calendario y jornada.
EMPLEADO_REFERENCIA = 'emp_cae2338320061380b9ae00c5f6ab1fed'
FUENTE = 'fuente:copia:f3-01:jornada-sintetica'
REF = re.compile(r'(per|prf|emp|cta)_[0-9a-f]{32}\Z')


def preparar(c):
    """Deja Cronos del empleado listo para el siguiente arranque de la copia."""
    copia = c.copia
    estado = copia.leer()
    if (estado.get('recorrido_completo') is not True
            or c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();')
               != estado.get('sid_copia')):
        raise ValueError('Cronos requiere la copia completa y su identificador PostgreSQL')
    motivos(c)
    empleado = persona_empleada(c)
    dsns = conexiones(c)
    origenes(c, urlsplit(dsns['dsn_cronos']).username)
    archivo = configuracion(c, empleado, dsns)
    activar(c)
    resultado = {'version': 1, 'tarea_ref': 'F3-01', 'copia_ref': copia.nombre,
                 'sid_copia': estado['sid_copia'], 'rol': ROL,
                 'cuenta_ref': empleado['cuenta_ref'], 'perfil_ref': empleado['perfil_ref'],
                 'configuracion_sha256': c.sha(archivo),
                 'gobierno': 'semilla técnica de copia de desarrollo, no gobierno',
                 'pendiente': ['resolucion_permisos', 'correcciones', 'notificaciones',
                               'calendario', 'equipo_informes', 'justificantes']}
    c.guardar(c.carpeta / 'etapa-cronos.json', resultado)
    return resultado


def persona_empleada(c):
    """Usa la persona técnica adicional de la copia y su certificado propio.

    La copia no trae ninguna persona con certificado que sea a la vez empleada
    en Personal. La semilla siguiente lo resuelve solo en esta copia.
    """
    identidad = c.leer(c.material / 'identidad/tecnico-rrhh-adicional-identidad.json')
    manifiesto = c.leer(c.material / 'identidad/tecnico-rrhh-adicional.json')
    if (identidad.get('autoridad') != 'no_autoritativo'
            or manifiesto.get('sujeto') != identidad.get('subject')
            or not re.fullmatch(r'[0-9a-f]{64}', identidad.get('certificate_sha256', ''))
            or any(not REF.fullmatch(str(manifiesto.get(k))) for k in ('persona_ref', 'cuenta_ref'))):
        raise ValueError('Identidad técnica adicional de la copia incoherente')
    persona, cuenta = manifiesto['persona_ref'], manifiesto['cuenta_ref']
    origen = c.consultar(f"""SELECT jsonb_build_object('ref',v.procedencia_ref,'version',v.procedencia_version,
        'huella',v.procedencia_huella_sha256,'autoridad',v.procedencia_autoridad)::text
      FROM vec_contexto_actor_v1.vinculo_contexto_actual a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
      WHERE v.cuenta_ref='{cuenta}' AND v.persona_ref='{persona}'
        AND v.perfil_ref='{manifiesto.get('perfil_ref', '')}' AND v.estado='activo'
        AND clock_timestamp() < v.vigente_hasta - interval '60 days';""").splitlines()
    if len(origen) != 1:
        raise ValueError('La persona técnica adicional no tiene un vínculo de cuenta inequívoco')
    refs = refs_semilla(c, persona)
    semilla(c, persona, cuenta, json.loads(origen[0]), refs)
    return {'certificado_sha256': identidad['certificate_sha256'], 'sujeto': identidad['subject'],
            'cuenta_ref': cuenta, 'perfil_ref': refs['perfil_ref'], 'persona_ref': persona,
            'empleado_ref': refs['empleado_ref']}


def refs_semilla(c, persona):
    """Fija las referencias antes del efecto; una repetición usa las mismas."""
    ruta = c.carpeta / 'cronos/semilla.json'
    (c.carpeta / 'cronos').mkdir(mode=0o700, exist_ok=True)
    if ruta.exists() or ruta.is_symlink():
        refs = c.leer(ruta)
    else:
        sufijo = lambda: secrets.token_hex(16)
        refs = {'version': 1, 'tipo': 'semilla_sintetica_de_copia', 'gobierno_real_acreditado': False,
                'persona_ref': persona, 'perfil_ref': 'prf_' + sufijo(),
                'vinculo_ref': 'vca_' + sufijo(), 'empleado_ref': 'emp_' + sufijo(),
                'proyeccion_ref': 'pep_' + sufijo(), 'asignacion_id': 'asg_' + sufijo(), 'rol_ref': ROL}
        c.guardar(ruta, refs)
    if (refs.get('persona_ref') != persona or refs.get('rol_ref') != ROL
            or any(not REF.fullmatch(refs.get(k, '')) for k in ('perfil_ref', 'empleado_ref'))
            or not re.fullmatch(r'vca_[0-9a-f]{32}', refs['vinculo_ref'])
            or not re.fullmatch(r'pep_[0-9a-f]{32}', refs['proyeccion_ref'])
            or not re.fullmatch(r'asg_[0-9a-f]{32}', refs['asignacion_id'])):
        raise ValueError('Semilla de Cronos divergente')
    return refs


def semilla(c, persona, cuenta, origen, refs):
    """Perfil propio, empleado en Personal y asignación del rol de Cronos.

    Semilla técnica de copia de desarrollo, no gobierno: el acto real de
    Administración y de Personal queda pendiente. Cada propietario escribe lo
    suyo en su transacción y vuelve a comprobarlo; repetir no duplica nada.
    """
    lit = lambda v: "'" + str(v).replace("'", "''") + "'"
    procedencia = ','.join((lit(origen['ref']), str(int(origen['version'])), lit(origen['huella']),
                            lit(origen['autoridad'])))
    perfil, vinculo, empleado = refs['perfil_ref'], refs['vinculo_ref'], refs['empleado_ref']
    # Lectura previa como superusuario de la copia: la RLS de Cronos impide al
    # propietario leer las filas de otra persona empleada.
    calendarios = c.consultar("SELECT coalesce(jsonb_agg(jsonb_build_object('anio',anio,'calendario_ref',calendario_ref)),'[]') "
                              "FROM vec_cronos_v1.calendario_asignacion WHERE empleado_ref='" + EMPLEADO_REFERENCIA + "';")
    jornadas = c.consultar("SELECT coalesce(jsonb_agg(jsonb_build_object('fecha',fecha,'version',version,'turno_ref',turno_ref,"
                           "'politica_version_ref',politica_version_ref,'zona_horaria',zona_horaria,"
                           "'minutos_previstos',minutos_previstos) ORDER BY fecha),'[]') "
                           "FROM vec_cronos_v1.programacion_jornada WHERE empleado_ref='" + EMPLEADO_REFERENCIA + "';")
    if json.loads(calendarios) == []:
        raise ValueError('La referencia no trae calendario sintético de Cronos')
    c.sql(f"""BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ({lit(perfil)},1,{lit(persona)},{procedencia},'activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '89 days')
 ON CONFLICT DO NOTHING;
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES ({lit(perfil)},1) ON CONFLICT DO NOTHING;
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ({lit(vinculo)},1,{lit(cuenta)},{lit(perfil)},{lit(persona)},{procedencia},'activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '89 days')
 ON CONFLICT DO NOTHING;
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ({lit(vinculo)},1) ON CONFLICT DO NOTHING;
COMMIT;
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
DO $proyeccion$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia
   WHERE proyeccion_ref={lit(refs['proyeccion_ref'])}) THEN
  PERFORM vec_personal.publicar_proyeccion_empleado_persona_v1({lit(refs['proyeccion_ref'])},1,
   {lit(persona)},{lit(empleado)},'activa',clock_timestamp()-interval '1 hour',
   clock_timestamp()+interval '89 days',NULL,{lit(origen['ref'])},{int(origen['version'])},{lit(origen['huella'])});
 END IF;
END $proyeccion$;
COMMIT;
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
-- Mismo calendario y jornada sintéticos que la persona empleada del volcado.
-- RLS: el propietario solo ve y añade filas del empleado de su sesión.
SELECT set_config('vec.cronos.empleado_ref',{lit(empleado)},true);
INSERT INTO vec_cronos_v1.calendario_asignacion(empleado_ref,anio,calendario_ref,fuente_ref,publicada_en)
 SELECT {lit(empleado)},a.anio,a.calendario_ref,{lit(FUENTE)},clock_timestamp()
   FROM jsonb_to_recordset({lit(calendarios)}::jsonb) a(anio integer,calendario_ref text)
  WHERE NOT EXISTS(SELECT 1 FROM vec_cronos_v1.calendario_asignacion b
                    WHERE b.empleado_ref={lit(empleado)} AND b.anio=a.anio);
INSERT INTO vec_cronos_v1.programacion_jornada(programacion_ref,empleado_ref,fecha,version,turno_ref,
  politica_version_ref,fuente_ref,zona_horaria,minutos_previstos,publicada_en)
 SELECT 'programacion:cronos:copia-'||substr({lit(empleado)},5,16)||'-'||to_char(j.fecha,'YYYYMMDD'),
        {lit(empleado)},j.fecha,j.version,j.turno_ref,j.politica_version_ref,{lit(FUENTE)},
        j.zona_horaria,j.minutos_previstos,clock_timestamp()
   FROM jsonb_to_recordset({lit(jornadas)}::jsonb) j(fecha date,version integer,turno_ref text,
        politica_version_ref text,zona_horaria text,minutos_previstos integer)
  WHERE NOT EXISTS(SELECT 1 FROM vec_cronos_v1.programacion_jornada k
                    WHERE k.empleado_ref={lit(empleado)} AND k.fecha=j.fecha AND k.version=j.version);
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL ROLE vec_autorizacion_propietario;
DO $semilla$
DECLARE c record; doc jsonb; ahora timestamptz := clock_timestamp();
BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref={lit(perfil)}
   AND asignacion_ref='asignacion:'||{lit(refs['asignacion_id'])}||':v1') THEN RETURN; END IF;
 SELECT v.* INTO STRICT c FROM vec_autorizacion.control_vigencia_version_rol_actual a
  JOIN vec_autorizacion.control_vigencia_version_rol v USING(version_rol_ref,revision)
  WHERE a.version_rol_ref={lit(ROL)};
 IF c.estado IS DISTINCT FROM 'habilitada'
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref={lit(ROL)}
   AND documento->>'estado'='publicada' AND jsonb_array_length(documento->'concesiones')=8)
  OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref={lit(perfil)})
 THEN RAISE EXCEPTION 'semilla Cronos: preimagen incompatible' USING ERRCODE='55000'; END IF;
 doc:=jsonb_build_object('asignacion_id',{lit(refs['asignacion_id'])},'version',1,
  'perfil_activo_ref',{lit(perfil)},'principal_id',{lit(persona)},'version_rol_ref',{lit(ROL)},
  'estado','activa','ambitos',jsonb_build_array(jsonb_build_object('clave','empleado_ref',
   'valores',jsonb_build_array({lit(empleado)}))),
  'emitida_por','identidad:desarrollo:no-autoritativa',
  'emitida_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_hasta',to_char((ahora+interval '88 days') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'revocada_en','0001-01-01T00:00:00Z');
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES('asignacion:'||{lit(refs['asignacion_id'])}||':v1',{lit(refs['asignacion_id'])},1,{lit(perfil)},{lit(persona)},{lit(ROL)},
  encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),ahora,doc);
 INSERT INTO vec_autorizacion.asignacion_perfil_actual
 VALUES({lit(perfil)},'asignacion:'||{lit(refs['asignacion_id'])}||':v1',ahora,
  'identidad:desarrollo:no-autoritativa','acto:cronos:copia:asignacion:v1');
END $semilla$;
COMMIT;
""".encode(), 'cronos-semilla.log')
    canonico = c.consultar("SELECT resultado||'|'||coalesce(empleado_ref,'') FROM "
                           "vec_personal.resolver_empleado_canonico_persona_v1(" + lit(persona) + ",clock_timestamp());")
    calendario = c.consultar(f"SELECT count(*) FROM vec_cronos_v1.calendario_asignacion WHERE empleado_ref={lit(empleado)};")
    if (canonico != 'empleado|' + empleado or calendario in ('', '0') or c.consultar(
            f"SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref={lit(perfil)};") != '1'):
        raise ValueError('Semilla de Cronos no confirmada')


def motivos(c):
    """Exige el catálogo de CRN18 publicado, vigente y con sus tres entradas."""
    catalogo, version, huella = CATALOGO
    filas = c.consultar(f"""SELECT e.entrada_clave FROM vec_autorizacion.motivo_v2_catalogo_publicado p
      JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version)
      WHERE p.catalogo_id='{catalogo}' AND p.catalogo_version={version}
        AND p.catalogo_huella_publicada_sha256='{huella}'
        AND e.vigente_desde<=clock_timestamp() AND (e.vigente_hasta IS NULL OR clock_timestamp()<e.vigente_hasta)
        AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r
          WHERE r.catalogo_id=p.catalogo_id AND r.catalogo_version=p.catalogo_version)
      ORDER BY 1;""").splitlines()
    if filas != sorted({MOTIVO_CONSULTA, MOTIVO_ACTUACION, MOTIVO_DENEGADO}):
        raise ValueError('Falta el catálogo de motivos de Cronos (CRN18)')


def referencia_motivo(clave):
    catalogo, version, huella = CATALOGO
    return {'catalogo_id': catalogo, 'catalogo_version': version,
            'catalogo_huella_sha256': huella, 'entrada_clave': clave}


def conexiones(c):
    """Crea una vez los ocho LOGIN o coteja los conservados; nunca repite un alta."""
    copia = c.copia
    estado = copia.leer()
    nombre = _nombre_instancia(copia)
    _privado(c.raiz, directorio=True)
    _privado(c.material, directorio=True)
    ca = c.material / 'ca/ca.crt'
    _privado(ca, directorio=False)
    puerto = estado.get('puerto_pg')
    if type(puerto) is not int or not 1024 <= puerto <= 65535:
        raise ValueError('Puerto PostgreSQL de Cronos inválido')
    nombres = [_rol(nombre, PRIMER_INDICE + i) for i in range(len(GRUPOS))]
    carpeta = c.carpeta / 'cronos'
    carpeta.mkdir(mode=0o700, exist_ok=True)
    _privado(carpeta, directorio=True)
    resultado = carpeta / 'conexiones.json'
    pendiente = carpeta / 'conexiones-pendientes.json'
    if resultado.exists() or resultado.is_symlink():
        valores = c.leer(resultado)
        if set(valores) != set(GRUPOS):
            raise ValueError('Inventario de conexiones de Cronos divergente')
        for campo, login in zip(GRUPOS, nombres):
            u = urlsplit(valores[campo])
            if (u.scheme != 'postgresql' or u.hostname != 'localhost' or u.port != puerto
                    or u.path != '/postgres' or u.username != login or not u.password
                    or u.fragment or parse_qs(u.query, strict_parsing=True) != {
                        'sslmode': ['verify-full'], 'sslrootcert': [str(ca)]}):
                raise ValueError('Conexión de Cronos ajena a la copia o sin TLS verificado')
        return valores
    if pendiente.exists() or pendiente.is_symlink():
        raise ValueError('Alta de Cronos incierta; conciliar las ocho cuentas originales')
    claves = [secrets.token_hex(32) for _ in nombres]
    vence = (c.ahora() + timedelta(days=90)).strftime('%Y-%m-%dT%H:%M:%SZ')
    c.guardar(pendiente, {'version': 1, 'copia_ref': copia.nombre,
                          'sid_copia': estado['sid_copia'], 'logins': nombres})
    c.sql(sql_logins(nombres, claves, list(GRUPOS.values()), vence), 'cronos-conexiones.log')
    valores = {campo: 'postgresql://' + quote(login) + ':' + quote(clave)
               + '@localhost:' + str(puerto) + '/postgres?' + urlencode({
                   'sslmode': 'verify-full', 'sslrootcert': str(ca)})
               for campo, login, clave in zip(GRUPOS, nombres, claves)}
    c.guardar(resultado, valores)
    return valores


def origenes(c, login):
    """Inserta las ocho filas AD172 del ejecutor o coteja que ya son exactas."""
    if not re.fullmatch(r'vcl_[a-z0-9_]{1,59}', login or ''):
        raise ValueError('LOGIN ejecutor de Cronos inválido')
    esperado = sorted('|'.join((login, a, o, PROCESO, CANAL)) for a, o in CONSUMOS)
    consulta = ("SELECT login_nombre||'|'||audiencia_consumo||'|'||operacion||'|'||proceso||'|'||canal_permitido "
                "FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 "
                "WHERE login_nombre='" + login + "' OR audiencia_consumo LIKE 'vec\\_cronos\\_v1.%' ORDER BY 1;")
    actual = c.consultar(consulta).splitlines()
    if actual == esperado:
        return
    if actual:
        raise ValueError('Origen de consumos de Cronos divergente')
    valores = ','.join(f"('{login}','{a}','{o}','{PROCESO}','{CANAL}')" for a, o in CONSUMOS)
    c.sql(("BEGIN;\nSET LOCAL search_path=pg_catalog;\nSET LOCAL lock_timeout='5s';\n"
           "SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;\n"
           "INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1"
           "(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) VALUES "
           + valores + ";\nCOMMIT;\n").encode(), 'cronos-origenes.log')
    if c.consultar(consulta).splitlines() != esperado:
        raise ValueError('Origen de consumos de Cronos no confirmado')


def configuracion(c, empleado, dsns):
    """Escribe el fichero privado que lee el arranque; la ausencia cierra Cronos."""
    datos = {'version': 1, 'autoridad': 'no_autoritativo',
             'cuentas': [{'certificado_sha256': empleado['certificado_sha256'],
                          'sujeto': empleado['sujeto'], 'cuenta_ref': empleado['cuenta_ref'],
                          'perfil_ref': empleado['perfil_ref']}],
             **dsns, 'zona_horaria': 'Europe/Madrid',
             'auditoria_intentos': {'motivo_denegado': referencia_motivo(MOTIVO_DENEGADO)},
             'motivos': {clave: referencia_motivo(valor) for clave, valor in MOTIVOS.items()},
             'canal_remoto': CANAL_REMOTO}
    archivo = c.material / 'identidad/cronos-empleado.json'
    c.guardar(archivo, datos)
    return archivo


def activar(c):
    """Enciende el selector del empleado; resolución y avisos siguen apagados."""
    from copia_local import guardar, sha256
    copia = c.copia
    entorno = copia.entorno_modulos()
    if not entorno:
        raise ValueError('La copia no tiene entorno de módulos')
    entorno['VEC_CRONOS_EMPLEADO_ENABLED'] = 'true'
    for selector in ('VEC_CRONOS_RESOLUCION_ENABLED', 'VEC_CRONOS_NOTIFICACIONES_ENABLED'):
        if entorno.get(selector, 'false') != 'false':
            raise ValueError('Selector de Cronos sin sus migraciones')
    visibles = entorno.get('VEC_PORTAL_MODULOS_VISIBLES', '')
    if visibles and 'cronos' not in visibles.split(','):
        entorno['VEC_PORTAL_MODULOS_VISIBLES'] = visibles + ',cronos'
    archivo = copia.raiz / 'entorno_modulos.json'
    guardar(archivo, entorno)
    estado = copia.leer()
    estado['entorno_modulos_sha256'] = sha256(archivo)
    guardar(copia.estado, estado)
