"""Gobierno B103 y AUT60 de roles simples en la copia sintética propia."""

import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import sys
from datetime import timedelta

from admin_copia import catalogo_acciones as cat
from admin_copia.ejecutor import DESARROLLO, binario, privado


CATALOGO = 'catalogo:bolsa:carga_convoca:b1'
ROL_BOLSA = 'tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo'
ROL_ORGANIZACION = 'consulta_organizacion_historica_rrhh'
ACCION_ORIGEN = 'bolsa.origen_ct.consultar'
AUDIENCIAS = {
    'b103': (
        ('vec_autorizacion.versionar_rol_bolsa.propuesta.v1',
         'administracion.perfiles.version_bolsa.proponer'),
        ('vec_autorizacion.versionar_rol_bolsa.cierre.v1',
         'administracion.perfiles.version_bolsa.aprobar')),
    'roles': (
        ('vec_autorizacion.gobierno_rol_nuevo.propuesta.v1',
         'administracion.perfiles.definicion.proponer'),
        ('vec_autorizacion.gobierno_rol_nuevo.cierre.v1',
         'administracion.perfiles.definicion.aprobar')),
}
POOL = {
    'b103': (('version-bolsa-b103-ejecutor', 'vec_admin_version_rol_bolsa_b103_ejecutor'),
             ('catalogo-acciones-lector-b103', 'vec_admin_catalogo_acciones_lector')),
    'roles': (('gobierno-roles-ejecutor', 'vec_admin_gobierno_roles_ejecutor'),
              ('catalogo-roles-lector', 'vec_admin_catalogo_acciones_lector')),
}
RUTAS = {
    'b103': '/api/admin/perfiles/v1/gobierno-version-bolsa-rrhh',
    'roles': '/api/admin/perfiles/v1/gobierno-roles',
}


def exigir(condicion, mensaje):
    if not condicion:
        raise ValueError(mensaje)


def literal(valor):
    exigir(isinstance(valor, str) and re.fullmatch(r'[A-Za-z0-9_.:/-]{1,200}', valor),
           'Referencia SQL de ADMIN inválida')
    return "'" + valor + "'"


def _documento(c, ruta):
    return c.leer(privado(ruta))


def _admisiones(c):
    base = c.carpeta / 'catalogo'
    meta = _documento(c, base / 'admisiones.json')
    sid = c.copia.leer()['sid_copia']
    actos = meta.get('actos', [])
    exigir(meta.get('version') == 1 and meta.get('copia_ref') == c.copia.nombre
           and meta.get('sid_copia') == sid and len(actos) == 2
           and [a.get('nombre') for a in actos] == ['organizacion', 'origen_ct']
           and meta.get('recibos_reales') == 2 and meta.get('replays_cotejados') == 2,
           'Dos admisiones AUT58 no confirmadas en esta copia')
    for a in actos:
        carpeta = base / a['nombre']
        primero = _documento(c, carpeta / 'acuse-primero.json')
        replay = _documento(c, carpeta / 'acuse-replay.json')
        exigir(a.get('sid_copia') == sid and primero.get('recibo') == replay.get('recibo')
               and replay.get('replay') is True
               and c.sha(carpeta / 'acuse-primero.json') == a.get('acuse_primero_sha256')
               and c.sha(carpeta / 'acuse-replay.json') == a.get('acuse_replay_sha256')
               and primero.get('recibo', {}).get('catalogo_sha256') == a.get('catalogo_sha256'),
               'Acuse AUT58 o SID divergente')
    return actos[-1]


def _cabeza(c, ultimo):
    filas = c.consultar(cat.SQL_CABEZA).splitlines()
    exigir(len(filas) == 1, 'Cabeza del catálogo ambigua')
    cabeza = json.loads(filas[0])
    exigir(cabeza.get('version') == ultimo['catalogo_version']
           and cabeza.get('huella') == ultimo['catalogo_sha256']
           and hashlib.sha256(cabeza['canon'].encode()).hexdigest() == cabeza['huella'],
           'Cabeza del catálogo distinta de la segunda admisión')
    catalogo = json.loads(cabeza['canon'])
    exigir(catalogo.get('referencia') == CATALOGO and catalogo.get('version') == cabeza['version'],
           'Catálogo publicado ajeno')
    return cabeza, catalogo


def _entrada(catalogo, fuente, accion):
    propias = [e for e in fuente['entradas'] if e['concesion']['accion'] == accion]
    exigir(len(propias) == 1 and fuente['huella_sha256'] == hashlib.sha256(
           fuente['entradas_canon'].encode()).hexdigest(), 'Fuente Git de rol inválida')
    e = propias[0]
    publicadas = [x for x in catalogo['entradas'] if x['referencia'] == e['referencia']
                  and x['version'] == e['version']]
    exigir(len(publicadas) == 1 and publicadas[0] == e,
           'Entrada del rol no admitida en el catálogo AUT58')
    return {'entrada_ref': e['referencia'], 'entrada_version': e['version'],
            'entrada_huella_sha256': hashlib.sha256(
                cat.go_json(cat.entrada_go(e)).encode()).hexdigest()}


def _rol(c, rol):
    consulta = ("SELECT jsonb_build_object('ref',v.version_rol_ref,'huella',v.huella_sha256,"
        "'version',v.version,'documento',v.documento,'control_revision',cv.revision,"
        "'control_huella',cv.huella_sha256,'control_estado',cv.estado)::text "
        "FROM vec_autorizacion.version_rol v JOIN "
        "vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref) "
        "JOIN vec_autorizacion.control_vigencia_version_rol cv ON "
        "(cv.version_rol_ref,cv.revision)=(ca.version_rol_ref,ca.revision) "
        f"WHERE v.rol_id={literal(rol)} ORDER BY v.version DESC LIMIT 1;")
    filas = c.consultar(consulta).splitlines()
    exigir(len(filas) <= 1, 'Cabeza del rol ambigua')
    return json.loads(filas[0]) if filas else None


def _asignaciones(c, version_ref):
    consulta = ("SELECT COALESCE(jsonb_agg(jsonb_build_object('asignacion_ref',a.asignacion_ref,"
        "'huella_sha256',a.huella_sha256,'documento',a.documento) "
        "ORDER BY a.perfil_activo_ref),'[]'::jsonb)::text "
        "FROM vec_autorizacion.asignacion_perfil_actual x "
        "JOIN vec_autorizacion.asignacion_perfil a ON "
        "(a.perfil_activo_ref,a.asignacion_ref)=(x.perfil_activo_ref,x.asignacion_ref) "
        "JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref "
        f"JOIN vec_autorizacion.version_rol base ON base.version_rol_ref={literal(version_ref)} "
        "AND r.rol_id=base.rol_id AND r.version<=base.version "
        "WHERE a.documento->>'estado'='activa' "
        "AND (a.documento->>'vigente_desde')::timestamptz<=clock_timestamp() "
        "AND (a.documento->>'vigente_hasta')::timestamptz>clock_timestamp();")
    datos = json.loads(c.consultar(consulta))
    exigir(1 <= len(datos) <= 16 and len({x['asignacion_ref'] for x in datos}) == len(datos),
           'Asignaciones RRHH activas ausentes o ambiguas')
    return datos


def _sql_pools(c, run, dia, sid):
    """Los cuatro LOGIN usan grupos distintos y un mismo SID comprobado al ejecutar."""
    entrada = _documento(c, run / 'entrada-vec-admin.json')
    exigir(entrada.get('sid_copia') == sid and entrada.get('dia_gobierno') == dia
           and entrada.get('runner') == str(run), 'Entrada ADMIN ajena a la copia')
    puerto = c.copia.leer()['puerto_pg']
    ca = c.material / 'ca/ca.crt'
    privado(ca)
    carpeta = c.carpeta / 'roles_admin'
    carpeta.mkdir(mode=0o700, exist_ok=True)
    dba = carpeta / 'pools.sql'
    if dba.exists():
        raise ValueError('Pools ADMIN ya preparados; conservar el estado anterior')
    fecha = (c.ahora() + timedelta(days=89)).strftime('%Y-%m-%dT%H:%M:%SZ')
    sql = ["BEGIN;", "DO $sid$ BEGIN IF current_database() IS DISTINCT FROM 'postgres' "
           f"OR (SELECT system_identifier::text FROM pg_catalog.pg_control_system()) "
           f"IS DISTINCT FROM {literal(sid)} THEN RAISE EXCEPTION 'SID ADMIN ajeno'; "
           "END IF; END $sid$;", "SET LOCAL password_encryption='scram-sha-256';"]
    nombres = {}
    for tipo in ('b103', 'roles'):
        nombres[tipo] = []
        for indice, (archivo, grupo) in enumerate(POOL[tipo]):
            login = f'vec_adm_{tipo}_{indice}_{dia}'
            clave = secrets.token_urlsafe(32)
            c.guardar(run / 'vec-admin/pools' / (archivo + '.json'), {
                'dsn': f'host=localhost port={puerto} dbname=postgres user={login} '
                       f'password={clave} sslmode=verify-full sslrootcert={ca}'})
            sql.extend((f"DO $g$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname={literal(login)}) "
                        f"OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname={literal(grupo)}) "
                        "THEN RAISE EXCEPTION 'LOGIN ADMIN ocupado o grupo ausente'; END IF; END $g$;",
                        f"CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE "
                        f"NOREPLICATION NOBYPASSRLS VALID UNTIL {literal(fecha)} "
                        f"PASSWORD {cat.q(cat.scram(clave))};",
                        f"GRANT {grupo} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;"))
            nombres[tipo].append(login)
        sql.append('SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;')
        for audiencia, operacion in AUDIENCIAS[tipo]:
            sql.append("INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1"
                       "(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) "
                       f"VALUES({literal(nombres[tipo][0])},{literal(audiencia)},"
                       f"{literal(operacion)},{literal(entrada['procesos']['usuarios'])},"
                       "'administracion_privilegiada');")
        sql.append('RESET ROLE;')
    sql.append('COMMIT;')
    c.guardar(dba, ('\n'.join(sql) + '\n').encode())
    c.sql(privado(dba).read_bytes(), 'roles-admin-pools.log')
    return carpeta


def _overlay_b103(c, run, dia, base):
    """Conserva las dos claves del día en archivos B103 propios y pools segregados."""
    gov = run / 'gobierno' / dia
    original = _documento(c, gov / 'version-bolsa.json')
    entradas = original['confianza']['entradas_capacidad']
    exigir(len(entradas) == 2, 'Overlay B1 sin dos capacidades publicadas')
    nuevas = []
    for x in entradas:
        origen = Path(x['material_archivo'])
        exigir(origen.parent == gov and origen.name.startswith('version-bolsa-'),
               'Material B1 de otra copia')
        destino = gov / origen.name.replace('version-bolsa-', 'version-bolsa-b103-', 1)
        c.guardar(destino, privado(origen).read_bytes())
        nuevas.append({**x, 'material_archivo': str(destino)})
    original['confianza']['entradas_capacidad'] = nuevas
    original['pool_gobierno'] = str(run / 'vec-admin/pools/version-bolsa-b103-ejecutor.json')
    original['pool_catalogo'] = str(run / 'vec-admin/pools/catalogo-acciones-lector-b103.json')
    destino = gov / 'version-bolsa-b103.json'
    c.guardar(destino, original)
    return destino


def _overlays(c, run, dia):
    kit = c.raiz / 'admin/kit'
    base = _documento(c, run / ('config-gobierno-' + dia + '.json'))
    exigir(base.get('conjunto_capacidades') == 7, 'Gobierno ADMIN distinto del conjunto 7')
    entrada = _documento(c, run / 'entrada-vec-admin.json')
    motivo = entrada['motivo']
    config = dict(base)
    for tipo, clave in (('b103', 'version_bolsa'), ('roles', 'gobierno_roles')):
        config[clave] = {
            'pool_gobierno': str(run / 'vec-admin/pools' / (POOL[tipo][0][0] + '.json')),
            'pool_catalogo': str(run / 'vec-admin/pools' / (POOL[tipo][1][0] + '.json')),
            'motivos': {a: motivo for a, _ in AUDIENCIAS[tipo]}}
    ruta = run / ('config-gobierno-roles-b103-' + dia + '.json')
    c.guardar(ruta, config)
    cli = binario(c, kit, 'vec-gobierno-usuarios-admin')
    textos = privado(kit / 'source/web/static/textos/es/admin-gobierno-usuarios.json')
    for fase in ('version-bolsa', 'gobierno-roles'):
        c.ejecutar([cli, '-fase', fase, '-config', ruta, '-textos', textos,
                   '-timeout', '30s', '-acuse', 'acuse-aplicar.json'],
                  'roles-admin-' + fase + '.log', env=DESARROLLO)
    b103 = _overlay_b103(c, run, dia, base)
    roles = privado(run / 'gobierno' / dia / 'gobierno-roles.json')
    return b103, roles


def _activar(c, b103, roles):
    from copia_local import guardar
    from copia_local_administracion import ejecutar as ejecutar_admin, parar as parar_admin
    extra = c.raiz / 'admin_extra_env.json'
    c.guardar(extra, {'VEC_ADMIN_VERSION_BOLSA_B103_CONFIG_FILE': str(b103),
                     'VEC_ADMIN_GOBIERNO_ROLES_CONFIG_FILE': str(roles)})
    preparacion = _documento(c, c.raiz / 'admin_preparacion.json')
    exigir('admin_extra_env_sha256' not in preparacion, 'Overlay ADMIN ya activado')
    parar_admin(c.copia)
    nueva = {**preparacion, 'admin_extra_env_sha256': c.sha(extra)}
    guardar(c.raiz / 'admin_preparacion.json', nueva)
    try:
        ejecutar_admin(c.copia, 'arrancar')
        ejecutar_admin(c.copia, 'comprobar')
    except Exception as error:
        # Los recibos y el overlay fallido se conservan; se restaura el entorno
        # anterior antes de volver a servir Administración.
        parar_admin(c.copia)
        extra.rename(c.carpeta / 'admin-extra-fallido.json')
        guardar(c.raiz / 'admin_preparacion.json', preparacion)
        ejecutar_admin(c.copia, 'arrancar')
        ejecutar_admin(c.copia, 'comprobar')
        raise ValueError('Overlay de roles rechazado; Administración recuperada') from error


def _proponer_y_cerrar(c, carpeta, tipo, propuesta):
    ruta = RUTAS[tipo]
    preparacion = carpeta / (tipo + '-propuesta.json')
    pendiente = carpeta / (tipo + '-pendiente.json')
    respuesta = carpeta / (tipo + '-propuesta-respuesta.json')
    cierre = carpeta / (tipo + '-cierre.json')
    final = carpeta / (tipo + '-cierre-respuesta.json')
    c.guardar(preparacion, propuesta)
    if not respuesta.exists():
        exigir(not pendiente.exists(), 'Propuesta ADMIN incierta; conciliar operación original')
        c.guardar(pendiente, {'operacion_ref': propuesta['operacion_ref'],
                              'peticion_sha256': c.sha(preparacion)})
        estado, datos = c.admin(1, 'POST', ruta + '/propuestas', c.leer(preparacion))
        exigir(estado in (200, 201) and datos.get('propuesta_ref') == propuesta['operacion_ref']
               and re.fullmatch(r'[0-9a-f]{64}', datos.get('huella_sha256', '')),
               'Propuesta ADMIN sin confirmación nominal')
        c.guardar(respuesta, datos)
    aprobado = _documento(c, respuesta)
    motivo = propuesta['motivo']
    cuerpo = {'operacion_ref': 'cierre_admin:' + secrets.token_hex(16),
              'propuesta_ref': aprobado['propuesta_ref'],
              'propuesta_huella_sha256': aprobado['huella_sha256'], 'motivo': motivo}
    c.guardar(cierre, cuerpo)
    if not final.exists():
        incierto = carpeta / (tipo + '-cierre-pendiente.json')
        exigir(not incierto.exists(), 'Cierre ADMIN incierto; conciliar operación original')
        c.guardar(incierto, {'operacion_ref': cuerpo['operacion_ref'],
                             'peticion_sha256': c.sha(cierre)})
        estado, datos = c.admin(2, 'POST', ruta + '/cierres', c.leer(cierre))
        exigir(estado == 200 and datos.get('propuesta_ref') == aprobado['propuesta_ref']
               and (datos.get('recibo_ref') or datos.get('recibo', {}).get('auditoria_ref')),
               'Cierre ADMIN sin recibo nominal')
        c.guardar(final, datos)
    return c.leer(final)


def cotejar_destino_bolsa(c, version_ref):
    """No anuncia el avance si la cuenta nueva sigue ligada a una versión antigua."""
    bback = _documento(c, c.material / 'identidad/bolsa-bback.json')
    perfil = bback.get('perfil_ref')
    exigir(isinstance(perfil, str) and re.fullmatch(r'prf_[0-9a-f]{32}', perfil),
           'Perfil propio de Bolsa ausente')
    consulta = ("SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual x "
        "JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=x.asignacion_ref "
        f"WHERE x.perfil_activo_ref={literal(perfil)} AND a.version_rol_ref={literal(version_ref)} "
        "AND a.documento->>'estado'='activa' "
        "AND (a.documento->>'vigente_desde')::timestamptz<=clock_timestamp() "
        "AND (a.documento->>'vigente_hasta')::timestamptz>clock_timestamp();")
    exigir(c.consultar(consulta).strip() == '1',
           'El perfil nuevo de Bolsa no recibió la versión gobernada')
    return {'perfil_ref': perfil, 'version_rol_ref': version_ref}


def preparar(c):
    """Publica origen CT en el rol de Bolsa y un rol simple de Organización."""
    ultimo = _admisiones(c)
    estado = c.copia.leer()
    sid = estado['sid_copia']
    exigir(c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();') == sid,
           'SID cambió después de AUT58')
    _, catalogo = _cabeza(c, ultimo)
    run = c.raiz / 'admin/h10'
    entrada = _documento(c, run / 'entrada-vec-admin.json')
    dia = entrada['dia_gobierno']
    carpeta = _sql_pools(c, run, dia, sid)
    b103, roles = _overlays(c, run, dia)
    _activar(c, b103, roles)
    from .actos import perfiles_admin
    perfiles = perfiles_admin(c)
    exigir(len(set(perfiles)) == 2, 'Se exigen dos perfiles ADMIN distintos')

    fuente = json.loads((c.fuente / 'deploy/principal/fuentes/bolsa_origen_ct_rrhh_v2.json').read_bytes())
    seleccion = _entrada(catalogo, fuente, ACCION_ORIGEN)
    base = _rol(c, ROL_BOLSA)
    exigir(base and base['control_estado'] == 'habilitada', 'Rol RRHH de Bolsa no habilitado')
    acciones = {x['accion'] for x in base['documento']['concesiones']}
    referencias = []
    if ACCION_ORIGEN not in acciones:
        asignaciones = _asignaciones(c, base['ref'])
        cuerpo = {'operacion_ref': 'propuesta_admin:' + secrets.token_hex(16),
                  'catalogo_ref': CATALOGO, 'catalogo_version': catalogo['version'],
                  'catalogo_huella_sha256': ultimo['catalogo_sha256'],
                  'base_ref': base['ref'], 'base_huella_sha256': base['huella'],
                  'control_revision': int(base['control_revision']),
                  'control_huella_sha256': base['control_huella'],
                  'selecciones': [seleccion], 'asignaciones': asignaciones,
                  'motivo': entrada['motivo'],
                  'referencia_acto': 'acto:rrhh:perfiles:f1-131'}
        respuesta = _proponer_y_cerrar(c, carpeta, 'b103', cuerpo)
        referencias.append(respuesta.get('recibo_ref') or respuesta['recibo']['auditoria_ref'])
        nuevo = _rol(c, ROL_BOLSA)
        exigir(nuevo['version'] == base['version'] + 1
               and ACCION_ORIGEN in {x['accion'] for x in nuevo['documento']['concesiones']},
               'B103 no publicó la concesión de origen CT')
    else:
        exigir(base['version'] >= 2, 'Concesión de origen sin sucesión de rol')

    ruta_fuente = c.fuente / 'deploy/principal/fuentes/personal_organizacion_historica_rrhh_v1.json'
    if ruta_fuente.exists():
        fuente_organizacion = json.loads(ruta_fuente.read_bytes())
        accion = fuente_organizacion['entradas'][0]['concesion']['accion']
        seleccion = _entrada(catalogo, fuente_organizacion, accion)
        existente = _rol(c, ROL_ORGANIZACION)
        if existente is None:
            cuerpo = {'operacion_ref': 'propuesta_admin:' + secrets.token_hex(16),
                      'catalogo_ref': CATALOGO, 'catalogo_version': catalogo['version'],
                      'catalogo_huella_sha256': ultimo['catalogo_sha256'],
                      **seleccion, 'rol_id': ROL_ORGANIZACION,
                      'nombre': 'Consulta de Organización histórica RRHH',
                      'motivo': entrada['motivo'],
                      'referencia_acto': 'acto:rrhh:organizacion:f1-131'}
            respuesta = _proponer_y_cerrar(c, carpeta, 'roles', cuerpo)
            referencias.append(respuesta.get('recibo_ref') or respuesta['recibo']['auditoria_ref'])
            existente = _rol(c, ROL_ORGANIZACION)
        exigir(existente and existente['version'] == 1
               and existente['control_estado'] == 'habilitada'
               and {x['accion'] for x in existente['documento']['concesiones']} == {accion},
               'Rol simple de Organización no publicado con la concesión exacta')
    destino_bolsa = cotejar_destino_bolsa(c, _rol(c, ROL_BOLSA)['ref'])
    resultado = {'version': 1, 'copia_ref': c.copia.nombre, 'sid_copia': sid,
                 'destino_bolsa': destino_bolsa,
                 'catalogo_version': catalogo['version'],
                 'catalogo_sha256': ultimo['catalogo_sha256'],
                 'overlay_b103_sha256': c.sha(b103), 'overlay_roles_sha256': c.sha(roles),
                 'dos_perfiles_admin': perfiles, 'recibos': referencias,
                 'asignacion_directa': False}
    c.guardar(carpeta / 'resultado.json', resultado)
    return resultado
