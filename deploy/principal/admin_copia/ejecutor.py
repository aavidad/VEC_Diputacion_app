"""Conductor nativo de Administración para una copia local sintética propia."""

import datetime as dt
import hashlib
import http.client
import json
import os
from pathlib import Path
import re
import shutil
import socket
import ssl
import stat
import subprocess
import sys
import time

from recorrido_copia.comun import Contexto


DESARROLLO = {
    'VEC_EXECUTION_PROFILE': 'desarrollo', 'VEC_AUTH_MODE': 'desarrollo',
    'VEC_DEVELOPMENT_GUARD': 'ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO',
}
FASES = ('fuentes', 'unidad', 'bootstrap')
AUDIENCIAS_EXTERNAS = (
    'vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1',
    'vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1',
)


def exigir(condicion, mensaje):
    if not condicion:
        raise ValueError(mensaje)


def privado(ruta, directorio=False):
    ruta = Path(ruta)
    dato = ruta.lstat()
    exigir(ruta.is_absolute() and ruta.resolve(strict=True) == ruta
           and dato.st_uid == os.getuid() and not dato.st_mode & 0o077
           and (stat.S_ISDIR(dato.st_mode) if directorio else
                stat.S_ISREG(dato.st_mode) and dato.st_nlink == 1),
           'Material ADMIN ausente o con permisos inseguros')
    return ruta


def documento(ruta):
    return json.loads(privado(ruta).read_text())


def script(c, kit, nombre, *argumentos):
    manifiesto = documento(kit / 'manifiesto-kit.json')
    ruta = privado(kit / nombre)
    exigir(manifiesto['textos_sha256'].get(nombre) == c.sha(ruta),
           'Script ADMIN distinto del kit de esta copia')
    c.ejecutar([sys.executable, ruta, *argumentos], 'admin-' + nombre + '.log',
               env=DESARROLLO)


def binario(c, kit, nombre):
    ruta = privado(kit / 'bin' / nombre)
    manifiesto = documento(kit / 'manifiesto-kit.json')
    exigir(manifiesto['binarios_sha256'].get('bin/' + nombre) == c.sha(ruta),
           'Binario ADMIN distinto del kit de esta copia')
    return ruta


def sql(c, ruta, nombre='admin-sql.log'):
    privado(ruta)
    c.sql(ruta.read_bytes(), nombre)


def cotejar(c, estado):
    exigir(estado.get('nombre') == c.copia.nombre and estado.get('raiz') == str(c.raiz)
           and estado.get('fuente') == str(c.fuente)
           and estado.get('administracion', {}).get('versionado') is True
           and type(estado['administracion'].get('puerto')) is int,
           'Copia ADMIN o selector ajeno')
    sid = estado.get('sid_copia')
    exigir(isinstance(sid, str) and re.fullmatch(r'[0-9]{15,20}', sid),
           'Falta SID confirmado por la copia')
    exigir(c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();') == sid,
           'SID PostgreSQL distinto del estado de la copia')
    return sid


def publicar_autorizacion(c, run, fase, encargo):
    """Registra la aprobación técnica del encargo y del plan exacto; no firma."""
    config = documento(run / 'config.json')
    plan = documento(run / fase / ('plan-v3.json' if fase == 'bootstrap' else 'plan.json'))
    huella = plan.get('huella_plan_sha256')
    exigir(isinstance(huella, str) and re.fullmatch('[0-9a-f]{64}', huella),
           'Plan H10 sin huella confirmada')
    destino = Path(config['dba'][fase]['aprobacion_externa'])
    exigir(destino.parent == run / 'aprobaciones', 'Destino de aprobación ajeno a la copia')
    cuerpo = {'huella_plan_sha256': huella,
              'aprobacion_ref': config['dba'][fase]['aprobacion_ref'],
              'backup_ref': config['backup']['ref']}
    c.guardar(destino, cuerpo)
    c.guardar(run / 'aprobaciones' / (fase + '-contexto.json'), {
        'encargo_local_ref': encargo, 'fase': fase, 'plan_sha256': huella,
        'preimagen_admin_sha256': config['backup']['sha256'],
        'sid_copia': documento(run / 'contrato-arranque.json')['sid_copia'],
        'archivo_autorizacion_sha256': c.sha(destino),
        'registrado_en': c.ahora().isoformat(), 'alcance': 'desarrollo_sintetico_local',
        'firma_o_recibo_juridico': False})


def etapa_h10(c, kit, run, fase, encargo):
    config = run / 'config.json'
    script(c, kit, 'arranque.py', '--config', config, '--etapa', 'preparar_' + fase)
    publicar_autorizacion(c, run, fase, encargo)
    script(c, kit, 'arranque.py', '--config', config, '--etapa', 'configurar_' + fase)
    sql(c, run / 'dba' / (fase + '-password.sql'))
    # Ante resultado incierto, conservar el plan y exigir recuperación explícita.
    script(c, kit, 'arranque.py', '--config', config, '--etapa', 'aplicar_' + fase)
    acuse = documento(run / fase / 'acuse-primero.json')
    exigir(acuse.get('estado') == 'permitido' and acuse.get('auditoria_intento')
           and acuse.get('recibo', {}).get('auditoria_ref'),
           'Fase H10 sin recibo auditado')


def motivo(c, destino):
    consulta = ("SELECT jsonb_build_object('catalogo_id',c.catalogo_id,"
        "'catalogo_version',c.catalogo_version,'catalogo_huella_sha256',"
        "c.catalogo_huella_publicada_sha256,'entrada_clave',e.entrada_clave)::text "
        "FROM vec_autorizacion.motivo_v2_catalogo_publicado c "
        "JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version) "
        "WHERE e.vigente_desde<=clock_timestamp() "
        "AND (e.vigente_hasta IS NULL OR clock_timestamp()<e.vigente_hasta) "
        "AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r "
        "WHERE r.catalogo_id=c.catalogo_id AND r.catalogo_version=c.catalogo_version)"
        + " ORDER BY c.catalogo_id,c.catalogo_version DESC,e.entrada_clave LIMIT 1;")
    filas = c.consultar(consulta).splitlines()
    exigir(len(filas) == 1, 'No hay motivo publicado y vigente en la copia')
    valor = json.loads(filas[0])
    exigir(set(valor) == {'catalogo_id', 'catalogo_version',
                         'catalogo_huella_sha256', 'entrada_clave'}
           and re.fullmatch('[0-9a-f]{64}', valor['catalogo_huella_sha256']),
           'Motivo publicado inválido')
    c.guardar(destino, valor)
    return valor


def raiz_h(c, kit, run):
    carpeta = run / 'raiz'
    carpeta.mkdir(mode=0o700)
    semilla = carpeta / 'semilla-root-dev.bin'
    c.ejecutar([binario(c, kit, 'h-root-seed.test'), '-test.run',
               '^TestDerivarSemillaRaizHPrivada$', '-test.v'], 'admin-raiz.log',
              env={'VEC_H_MATERIAL_DIR': str(c.material),
                   'VEC_H_HMAC_CONFIG': str(c.material / 'idempotencia/configuracion.json'),
                   'VEC_H_ROOT_OUT': str(semilla)})
    privado(semilla)
    registro = (c.carpeta / 'admin-raiz.log').read_text()
    huellas = re.findall(r'spki_h=([0-9a-f]{64})', registro)
    exigir(len(huellas) == 1, 'Raíz H sin huella inequívoca')
    consulta = ("SELECT count(*) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p "
      "JOIN vec_autorizacion_atestada_v3.configuracion_raiz x ON "
      "x.configuracion_revision=p.configuracion_revision "
      "JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON "
      "(r.clave_id,r.version)=(x.raiz_clave_id,x.raiz_version) "
      "WHERE p.orden=(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual) "
      f"AND r.huella_spki_sha256='{huellas[0]}' "
      "AND r.valida_desde<=clock_timestamp() AND r.valida_hasta>clock_timestamp();")
    exigir(c.consultar(consulta) == '1', 'Raíz H no publicada para esta copia')
    return semilla


def gobierno(c, kit, run, dia, sid):
    comunes = ('--run', run, '--dia', dia, '--sid-copia', sid)
    script(c, kit, 'preparar_gobierno.py', '--fase', 'entrada', *comunes)
    sql(c, run / 'dba' / ('roles-gobierno-' + dia + '.sql'))
    config = run / ('config-gobierno-' + dia + '.json')
    cli = binario(c, kit, 'vec-gobierno-usuarios-admin')
    textos = kit / 'source/web/static/textos/es/admin-gobierno-usuarios.json'
    privado(textos)

    def fase(nombre, extra=()):
        c.ejecutar([cli, '-fase', nombre, '-config', config, '-textos', textos,
                   '-timeout', '30s', *extra], 'admin-gobierno-' + nombre + '.log',
                  env=DESARROLLO)

    fase('preparar')
    script(c, kit, 'preparar_gobierno.py', '--fase', 'configurar', *comunes)
    sql(c, run / 'dba' / ('config-gobierno-' + dia + '.sql'))
    fase('aplicar', ('-acuse', 'acuse-aplicar.json'))
    from admin_copia.preparar_runtime import validar_gobierno
    validar_gobierno(run, dia, aplicado=True)
    verificar_gobierno(c, fase)


def verificar_gobierno(c, fase):
    """Espera una lectura de la cadena; nunca repite la publicación confirmada."""
    try:
        fase('verificar', ('-acuse', 'acuse-aplicar.json'))
    except ValueError as fallo:
        if str(fallo) != 'recorrido_operacion_fallida_consulte_registro':
            raise
        try:
            raw = privado(c.carpeta / 'admin-gobierno-verificar.log').read_text().splitlines()
            ultimo = json.loads(raw[-1])
        except (ValueError, OSError, IndexError):
            raise fallo
        if ultimo.get('codigo') != 'verificacion_no_disponible':
            raise
        # El sellador de la aplicación puede no haber consumido aún el evento.
        # Solo se relee una vez; cualquier otro fallo conserva la copia y sus actos.
        time.sleep(1)
        fase('verificar', ('-acuse', 'acuse-aplicar.json'))


def runtime(c, kit, run, estado, sid, semilla, dia, sufijo):
    ahora = c.ahora()
    vigente = (ahora + dt.timedelta(days=88)).strftime('%Y-%m-%dT%H:%M:%SZ')
    retirada = (ahora + dt.timedelta(days=89)).strftime('%Y-%m-%dT%H:%M:%SZ')
    ca = c.material / 'ca/ca.crt'
    motivo(c, run / 'motivo-publicado.json')
    script(c, kit, 'preparar_runtime.py', '--run', run, '--kit', kit,
           '--instancia', sufijo, '--dia-gobierno', dia, '--semilla-raiz', semilla,
           '--ca-pg', ca, '--pg-host', 'localhost', '--pg-base', 'postgres',
           '--puerto-pg', estado['puerto_pg'], '--sid-copia', sid,
           '--vigente-hasta-config', vigente, '--retirada-en', retirada,
           '--redes', '127.0.0.0/8', '--motivo-publicado', run / 'motivo-publicado.json')



def overlay_externo(run, dia):
    exigir(re.fullmatch(r'[0-9]{8}[a-z0-9]{1,12}', dia), 'Día de overlay inválido')
    return privado(run / 'gobierno' / dia / 'version-bolsa-externa.json')


def preparar(c):
    estado = c.copia.leer()
    sid = cotejar(c, estado)
    exigir(estado.get('preparada') and not estado.get('admin_preparada')
           and c.copia.proceso(estado), 'La aplicación propia debe estar arrancada')
    admin = c.raiz / 'admin'
    exigir(not admin.exists() and not admin.is_symlink(),
           'ADMIN ya iniciado: conservar resultados y resolver el estado anterior')
    privado(c.raiz, True)
    privado(c.material, True)
    privado(c.material / 'ca/ca.crt')
    compilacion = documento(c.raiz / 'compilacion.json')
    puerto = estado['administracion']['puerto']
    sufijo = (c.copia.nombre.removeprefix('vec-copia-').replace('-', '_')[:10]
              + hashlib.sha256(c.copia.nombre.encode()).hexdigest()[:6])
    catalogo = estado['administracion'].get('catalogo_entrada_unidad')
    exigir(isinstance(catalogo, str) and re.fullmatch('[a-z0-9_]{3,80}', catalogo),
           'Falta la entrada publicada de unidad ADMIN')
    admin.mkdir(mode=0o700)
    kit = admin / 'kit'
    script_fuente = c.fuente / 'deploy/principal/admin_copia/kit.py'
    c.ejecutar([sys.executable, script_fuente, '--repo', c.fuente,
               '--revision', estado['revision'], '--sid-copia', sid,
               '--salida', kit, '--go', compilacion['go'],
               '--modcache', compilacion['gomodcache'], '--gocache', compilacion['gocache'],
               '--encargo-ref', 'encargo:f1-131', '--puerto-admin', puerto,
               '--catalogo-entrada-unidad', catalogo], 'admin-kit.log')
    manifiesto = documento(kit / 'manifiesto-kit.json')
    exigir(manifiesto.get('estado') == 'compilado_sin_actos'
           and manifiesto.get('revision') == estado['revision']
           and manifiesto.get('sid_copia') == sid,
           'Kit ADMIN incompleto o ajeno a la copia')
    preimagen = admin / 'preimagen-admin.dump'
    script(c, kit, 'guardar_preimagen.py', '--contenedor', c.copia.nombre,
           '--sid-copia', sid, '--salida', preimagen)
    script(c, kit, 'preparar_fixture.py', '--sid-copia', sid,
           '--respaldo-hz37', Path(estado['origen']) / 'principal-hz37-20261014.dump',
           '--preimagen-admin', preimagen, '--salida', admin)
    sql(c, admin / 'fixture-admin-hz37.sql', 'admin-fixture.log')
    run = admin / 'h10'
    script(c, kit, 'configuracion.py', '--kit', kit, '--revision', estado['revision'],
           '--sid-copia', sid, '--puerto-admin', puerto, '--instancia', sufijo,
           '--contenedor', c.copia.nombre, '--puerto-pg', estado['puerto_pg'],
           '--backup-admin', preimagen, '--material', c.material,
           '--ca-pg', c.material / 'ca/ca.crt', '--catalogo-entrada-unidad', catalogo,
           '--salida', run)
    script(c, kit, 'arranque.py', '--config', run / 'config.json', '--etapa', 'comprobar')
    for fase in FASES:
        etapa_h10(c, kit, run, fase, 'encargo:f1-131')
    script(c, kit, 'arranque.py', '--config', run / 'config.json', '--etapa', 'verificar')
    semilla = raiz_h(c, kit, run)
    dia = c.ahora().strftime('%Y%m%d') + hashlib.sha256(sufijo.encode()).hexdigest()[:6]
    # La entrada existe antes de que el gobierno lea sus identificadores.
    runtime(c, kit, run, estado, sid, semilla, dia, sufijo)
    gobierno(c, kit, run, dia, sid)
    entrada = run / 'entrada-vec-admin.json'
    for fase in ('tls', 'activos', 'pools'):
        script(c, kit, 'preparar_vec_admin.py', '--entrada', entrada, '--fase', fase, '--dia', dia)
    sql(c, run / 'vec-admin/dba-logins.sql', 'admin-runtime.log')
    for fase in ('claves', 'base', 'overlay', 'runtime', 'entorno'):
        script(c, kit, 'preparar_vec_admin.py', '--entrada', entrada, '--fase', fase, '--dia', dia)
    motivo_externo = motivo(c, run / 'motivo-externo-publicado.json')
    motivos = {audiencia: motivo_externo for audiencia in AUDIENCIAS_EXTERNAS}
    motivos_ruta = run / 'motivos-externos.json'
    c.guardar(motivos_ruta, motivos)
    script(c, kit, 'preparar_overlay_externo.py', '--run', run, '--dia', dia,
           '--motivos-publicados', motivos_ruta)
    overlay = run / ('config-gobierno-externo-' + dia + '.json')
    c.ejecutar([binario(c, kit, 'vec-gobierno-usuarios-admin'), '-fase',
               'version-bolsa-externa', '-config', overlay,
               '-textos', kit / 'source/web/static/textos/es/admin-gobierno-usuarios.json',
               '-timeout', '30s', '-acuse', 'acuse-aplicar.json'],
              'admin-overlay-externo.log', env=DESARROLLO)
    overlay_runtime = overlay_externo(run, dia)
    externo = run / 'vec-admin/config' / f'entorno-{dia}.env'
    privado(externo)
    fuente_bin = binario(c, kit, 'vec-admin')
    destino_bin = c.raiz / 'vec-admin'
    with fuente_bin.open('rb') as origen, destino_bin.open('xb') as destino:
        shutil.copyfileobj(origen, destino)
    destino_bin.chmod(0o700)
    exigir(c.sha(destino_bin) == c.sha(fuente_bin), 'Binario ADMIN copiado con otra huella')
    c.guardar(c.raiz / 'admin_preparacion.json', {
        'VEC_ADMIN_ENTORNO_FILE': str(externo), 'VEC_ADMIN_RUN': str(run),
        'VEC_ADMIN_ESTADO': 'preparado_con_acuses',
        'VEC_ADMIN_VERSION_BOLSA_EXTERNA_CONFIG_FILE': str(overlay_runtime)})


def agregar_entorno_firma(c, datos, run, variables):
    ruta = c.raiz / 'admin_firma_env.json'
    if not ruta.exists():
        exigir('admin_firma_env_sha256' not in datos, 'Falta el entorno de firma confirmado')
        return
    from recorrido_copia.montajes_firma import ENTORNO
    exigir(datos.get('admin_firma_env_sha256') == c.sha(privado(ruta)),
           'Entorno de firma distinto del confirmado')
    extra = documento(ruta)
    hashes = datos.get('admin_firma_archivos_sha256', {})
    exigir(set(extra) == set(ENTORNO) and set(hashes) == set(ENTORNO),
           'Claves de entorno de firma inesperadas')
    for clave, nombre in ENTORNO.items():
        esperado = run / 'vec-admin/config' / nombre
        exigir(extra[clave] == str(esperado) and c.sha(privado(esperado)) == hashes[clave],
               'Configuración de firma de otra copia o modificada')
        variables[clave] = str(esperado)


def entorno(c, estado):
    datos = documento(c.raiz / 'admin_preparacion.json')
    run = c.raiz / 'admin/h10'
    ruta = Path(datos['VEC_ADMIN_ENTORNO_FILE'])
    exigir(datos['VEC_ADMIN_RUN'] == str(run)
           and datos['VEC_ADMIN_ESTADO'] == 'preparado_con_acuses'
           and ruta.parent == run / 'vec-admin/config',
           'Entorno ADMIN de otra copia')
    variables = {}
    for linea in privado(ruta).read_text().splitlines():
        clave, separador, valor = linea.partition('=')
        exigir(separador and re.fullmatch(r'VEC_ADMIN_[A-Z0-9_]+', clave)
               and clave not in variables and '\x00' not in valor,
               'Archivo de entorno ADMIN inválido')
        variables[clave] = valor
    puerto = estado['administracion']['puerto']
    exigir(variables.get('VEC_ADMIN_ESCUCHA') == f'127.0.0.1:{puerto}',
           'Escucha ADMIN de otra copia')
    host = documento(run / 'entrada-vec-admin.json')['host']
    exigir(host == 'admin.localhost'
           and variables.get('VEC_ADMIN_HOST') in (host, f'{host}:{puerto}'),
           'Autoridad ADMIN inesperada')
    variables['VEC_ADMIN_HOST'] = f'{host}:{puerto}'
    externo = Path(datos['VEC_ADMIN_VERSION_BOLSA_EXTERNA_CONFIG_FILE'])
    dia = documento(run / 'entrada-vec-admin.json')['dia_gobierno']
    exigir(externo == overlay_externo(run, dia), 'Overlay ADMIN de otra copia')
    variables['VEC_ADMIN_VERSION_BOLSA_EXTERNA_CONFIG_FILE'] = str(externo)
    extra = c.raiz / 'admin_extra_env.json'
    if extra.exists():
        exigir(datos.get('admin_extra_env_sha256') == c.sha(extra),
               'Entorno ADMIN adicional distinto del confirmado')
        adicionales = documento(extra)
        esperadas = {'VEC_ADMIN_GOBIERNO_ROLES_CONFIG_FILE': 'gobierno-roles.json',
                     'VEC_ADMIN_VERSION_BOLSA_B103_CONFIG_FILE': 'version-bolsa-b103.json'}
        exigir(set(adicionales) == set(esperadas), 'Claves ADMIN adicionales inesperadas')
        for clave, nombre in esperadas.items():
            ruta_extra = Path(adicionales[clave])
            exigir(ruta_extra.name == nombre and ruta_extra.parent.parent == run / 'gobierno'
                   and re.fullmatch(r'[0-9]{8}[a-z0-9]{1,12}', ruta_extra.parent.name),
                   'Overlay ADMIN adicional de otra copia')
            privado(ruta_extra)
            variables[clave] = str(ruta_extra)
    agregar_entorno_firma(c, datos, run, variables)
    variables.update(DESARROLLO)
    variables.update(PATH='/usr/bin:/bin', LANG='C.UTF-8', HOME=str(c.raiz))
    return variables


def activo(c, estado):
    pid = estado.get('pid_admin')
    if type(pid) is not int or pid <= 0:
        return False
    try:
        return (Path(f'/proc/{pid}/exe').resolve() == c.raiz / 'vec-admin'
                and Path(f'/proc/{pid}/stat').read_text().split()[21]
                == estado.get('inicio_proceso_admin'))
    except FileNotFoundError:
        return False


def arrancar(c):
    estado = c.copia.leer()
    cotejar(c, estado)
    exigir(estado.get('admin_preparada') and not activo(c, estado),
           'ADMIN no preparado o ya activo')
    kit = c.raiz / 'admin/kit'
    binario(c, kit, 'vec-admin')
    propio = privado(c.raiz / 'vec-admin')
    exigir(c.sha(propio) == c.sha(kit / 'bin/vec-admin'),
           'Ejecutable ADMIN de la copia divergente')
    variables = entorno(c, estado)
    puerto = estado['administracion']['puerto']
    with socket.socket() as s:
        s.bind(('127.0.0.1', puerto))
    with (c.raiz / 'admin.log').open('ab') as log:
        proceso = subprocess.Popen([str(propio)], cwd=c.fuente, env=variables,
                                   stdin=subprocess.DEVNULL, stdout=log,
                                   stderr=subprocess.STDOUT, start_new_session=True)
    estado = c.copia.leer()
    estado['pid_admin'] = proceso.pid
    estado['inicio_proceso_admin'] = Path(f'/proc/{proceso.pid}/stat').read_text().split()[21]
    from copia_local import guardar
    guardar(c.copia.estado, estado)


def comprobar(c):
    estado = c.copia.leer()
    cotejar(c, estado)
    exigir(activo(c, estado), 'ADMIN de esta copia no está activo')
    run = c.raiz / 'admin/h10'
    contexto = ssl.create_default_context(cafile=str(privado(run / 'vec-admin/tls/srv-ca.pem')))
    contexto.load_cert_chain(str(privado(run / 'fuentes/cliente-1.pem')),
                            str(privado(run / 'fuentes/cliente-1.key')))
    puerto = estado['administracion']['puerto']
    for _ in range(60):
        try:
            conexion = http.client.HTTPSConnection('admin.localhost', puerto,
                                                    context=contexto, timeout=3)
            conexion.sock = contexto.wrap_socket(
                socket.create_connection(('127.0.0.1', puerto), timeout=3),
                server_hostname='admin.localhost')
            try:
                conexion.request('GET', '/livez', headers={'Host': f'admin.localhost:{puerto}'})
                respuesta = conexion.getresponse()
                respuesta.read(65536)
                exigir(respuesta.status in (200, 204), 'ADMIN no confirmó salud HTTPS')
                return
            finally:
                conexion.close()
        except OSError:
            exigir(activo(c, c.copia.leer()), 'ADMIN terminó antes de responder HTTPS')
            time.sleep(.5)
    raise ValueError('ADMIN no respondió por HTTPS propio')


def ejecutar(copia, fase):
    exigir(fase in ('preparar', 'arrancar', 'comprobar'), 'Fase ADMIN desconocida')
    c = Contexto(copia, {})
    {'preparar': preparar, 'arrancar': arrancar, 'comprobar': comprobar}[fase](c)
