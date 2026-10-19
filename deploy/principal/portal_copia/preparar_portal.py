#!/usr/bin/env python3
"""Prepara en una copia local la autoridad del aspirante sintético.

F0-21 sigue pendiente: las admisiones y la asignación son semilla declarada
de esta copia, y no acreditan actos de gobierno real. No instala migraciones.
"""
import argparse
import datetime as dt
from contextlib import contextmanager
import fcntl
import hashlib
import importlib.util
import json
import os
import re
import secrets
import stat
import subprocess
import sys
from pathlib import Path
from urllib.parse import urlencode, urlsplit, parse_qs

os.umask(0o077)
sys.dont_write_bytecode = True
BASE = Path('/dev/shm/vec/pg')
STATE = Path(os.environ['VEC_COPIA_PORTAL_ESTADO'])
REF = re.compile(r'^[a-z]+_[a-z0-9_-]{22,128}$')


def fail(codigo):
    raise RuntimeError(codigo)


def private_dir(p):
    s = p.lstat()
    if not stat.S_ISDIR(s.st_mode) or s.st_uid != os.getuid() or s.st_mode & 0o077 or p.resolve() != p:
        fail('directorio_privado_invalido')


def private_file(p):
    s = p.lstat()
    if not stat.S_ISREG(s.st_mode) or s.st_uid != os.getuid() or s.st_nlink != 1 or s.st_mode & 0o077:
        fail('fichero_privado_invalido')
    return p.read_bytes()


@contextmanager
def candado_copia(r, heredado=None):
    nombre = 'vec-copia-' + r.name
    if not re.fullmatch(r'vec-copia-[a-z0-9-]{2,80}', nombre):
        fail('nombre_copia_invalido')
    ruta = BASE / ('.' + nombre + '.lock')
    propio = heredado is None
    fd = os.open(ruta, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600) if propio else heredado
    try:
        archivo, enlace = os.fstat(fd), ruta.lstat()
        if (not stat.S_ISREG(archivo.st_mode) or archivo.st_uid != os.getuid()
                or archivo.st_mode & 0o077 or (archivo.st_dev, archivo.st_ino) != (enlace.st_dev, enlace.st_ino)):
            fail('candado_copia_incompatible')
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            fail('copia_ocupada')
        yield
    finally:
        if propio:
            os.close(fd)


def verificar_copia(copia, state, control):
    copia.leer()
    if not copia.existe():
        fail('postgresql_copia_ausente')
    inspeccion = subprocess.run(['docker', 'container', 'inspect', state['nombre']],
                                capture_output=True, timeout=10, check=True)
    datos = json.loads(inspeccion.stdout)[0]
    puerto = datos['NetworkSettings']['Ports'].get('5432/tcp')
    if (datos['Config'].get('Labels', {}).get('vec.copia') != str(copia.raiz)
            or datos['State'].get('Running') is not True
            or puerto != [{'HostIp': '127.0.0.1', 'HostPort': str(state['puerto_pg'])}]):
        fail('contenedor_copia_o_puerto_divergente')
    resultado = subprocess.run(['docker', 'exec', '-u', 'postgres', state['nombre'], 'psql', '-XAt',
                                '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres',
                                '-c', 'SELECT system_identifier FROM pg_control_system()'],
                               capture_output=True, timeout=10, check=True)
    sid = resultado.stdout.decode().strip()
    if not re.fullmatch(r'[0-9]{18,20}', sid):
        fail('sid_copia_invalido')
    esperado = {'nombre': state['nombre'], 'raiz': str(copia.raiz), 'puerto_pg': state['puerto_pg'], 'sid': sid}
    write_or_match(control, esperado)
    return sid


def write(p, value):
    p.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    data = value if isinstance(value, bytes) else (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + '\n').encode()
    with p.open('xb') as f:
        f.write(data)
        f.flush()
        os.fsync(f.fileno())
    p.chmod(0o600)
    directorio = os.open(p.parent, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(directorio)
    finally:
        os.close(directorio)


def write_or_match(p, value):
    data = value if isinstance(value, bytes) else (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + '\n').encode()
    if p.exists():
        if private_file(p) != data:
            fail('fuente_privada_divergente')
    else:
        write(p, data)


def invoke(args, log, *, env=None, input_bytes=None, cwd=None):
    with log.open('ab') as output:
        result = subprocess.run([str(a) for a in args], input=input_bytes, stdout=output,
                                stderr=subprocess.STDOUT, cwd=cwd, env=env, timeout=180, check=False)
    if result.returncode:
        fail('fallo_etapa_consultar_log_privado')


def module(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def ref(prefix):
    return prefix + secrets.token_hex(16)


def dsn(login, password, port, ca):
    return 'postgresql://' + login + ':' + password + '@localhost:' + str(port) + '/postgres?' + urlencode({
        'sslmode': 'verify-full', 'sslrootcert': str(ca)})


def cotejar_asignacion_retenida(actual, intento):
    if actual is None:
        return False
    documento = json.loads(intento['documento'])
    evento = actual.get('evento')
    return (actual.get('asignacion_ref') == intento['asignacion_ref']
            and actual.get('huella_sha256') == intento['huella_sha256']
            and actual.get('documento') == documento
            and actual.get('acto_ref') == intento['acto_ref']
            and actual.get('perfil_ref') == intento['perfil_ref']
            and actual.get('principal_id') == intento['principal_id']
            and actual.get('rol_ref') == intento['rol_ref']
            and documento.get('ambitos') == [{'clave': 'candidato_ref', 'valores': [intento['candidato_ref']]}]
            and isinstance(evento, dict)
            and evento.get('objeto_ref') == intento['asignacion_ref']
            and evento.get('huella_nueva') == intento['huella_sha256']
            and evento.get('version') == 1)


def confirmar_asignacion_retenida(actual, intento, respuesta_confirmada):
    if cotejar_asignacion_retenida(actual, intento):
        return
    if actual is not None:
        fail('asignacion_ajena_no_repetir')
    fail('postimagen_asignacion_divergente' if respuesta_confirmada else
         'publicador_asignacion_rechazo_reconciliar')


def entorno_portal(environment, dsns, external, sin_inscripcion):
    nuevo = dict(environment)
    nuevo.update({name: value for name, value in dsns.items() if name.startswith('VEC_')})
    nuevo['VEC_MATERIAL_PORTAL_EXTERNO_DIR'] = str(external)
    if sin_inscripcion:
        nuevo.pop('VEC_BOLSA_INSCRIPCIONES_ENABLED', None)
    else:
        nuevo['VEC_BOLSA_INSCRIPCIONES_ENABLED'] = 'true'
    nuevo['VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS'] = '2136'
    nuevo.pop('VEC_PORTAL_PROCESO', None)
    return nuevo


def publicar_entorno_retenido(copia, raiz, estado, intento_path, nuevo=None):
    ruta = raiz / 'entorno_modulos.json'
    if not intento_path.exists():
        if nuevo is None:
            return
        if copia.sha256(ruta) != estado['entorno_modulos_sha256']:
            fail('entorno_preimagen_cambio')
        contenido = (json.dumps(nuevo, ensure_ascii=False, indent=2) + '\n').encode()
        write(intento_path, {'revision': estado['revision'],
              'anterior': estado['entorno_modulos_sha256'], 'nuevo': nuevo,
              'huella_nueva': hashlib.sha256(contenido).hexdigest()})
    intento = json.loads(private_file(intento_path))
    contenido = (json.dumps(intento['nuevo'], ensure_ascii=False, indent=2) + '\n').encode()
    anterior, siguiente = intento['anterior'], intento['huella_nueva']
    actual = copia.sha256(ruta)
    if (intento['revision'] != estado['revision']
            or not re.fullmatch('[0-9a-f]{64}', anterior)
            or hashlib.sha256(contenido).hexdigest() != siguiente
            or (nuevo is not None and nuevo != intento['nuevo'])
            or estado['entorno_modulos_sha256'] not in (anterior, siguiente)
            or actual not in (anterior, siguiente)
            or (estado['entorno_modulos_sha256'] == siguiente and actual != siguiente)):
        fail('intento_entorno_divergente')
    if actual == anterior:
        copia.guardar(ruta, intento['nuevo'])
    estado['entorno_modulos_sha256'] = siguiente
    copia.guardar(raiz / 'estado.json', estado)


def base_env():
    return {'PATH': os.environ['PATH'], 'HOME': os.environ['HOME'], 'LANG': 'C.UTF-8'}


def pg(r, state, sql, log, verificar):
    verificar()
    invoke(['docker', 'exec', '-i', '-u', 'postgres', state['nombre'], 'psql', '-X', '-v', 'ON_ERROR_STOP=1',
            '-U', 'postgres', '-d', 'postgres'], log, input_bytes=sql.encode())


def step(work, name, function):
    marker = work / (name + '.ok')
    if marker.exists():
        private_file(marker)
        print(name + ': preparado')
        return
    function()
    write(marker, (name + '\n').encode())
    print(name + ': preparado')


def main_locked(a, r):
    state = json.loads(private_file(r / 'estado.json'))
    repo = Path(state['fuente'])
    go = json.loads(private_file(r / 'compilacion.json'))['go']
    if (state['raiz'] != str(r) or state['nombre'] != 'vec-copia-' + r.name
            or not 1024 <= state['puerto_pg'] <= 65535 or repo.resolve() != repo or not (r / 'vec-server').is_file()):
        fail('marca_de_copia_incompatible')
    private_dir(STATE)
    if any((p / '.git').exists() for p in (STATE, *STATE.parents)):
        fail('estado_portal_dentro_git')
    private_dir(r / 'material')
    copia = module(repo / 'deploy/principal/copia_local.py', 'copia_local_portal')
    copia.verificar_fuente(state['revision'])
    equipo, instancia = r.name.split('-', 1)
    instancia_copia = copia.Copia(equipo, instancia)
    instancia_copia.validar_binario(state)
    if not a.solo_accesos_rrhh and not instancia_copia.proceso(state):
        fail('mantener_aplicacion_y_sellado_activos')
    env_path = r / 'entorno_modulos.json'
    material = r / 'material'
    if not isinstance(state.get('creada_en'), (int, float)) or state['creada_en'] <= 0:
        fail('marca_de_copia_incompleta')
    suffix = hashlib.sha256((state['nombre'] + ':' + str(state['creada_en'])).encode()).hexdigest()[:12]
    work = STATE / ('portal-externo-trabajo-' + suffix)
    work.mkdir(mode=0o700, exist_ok=True)
    private_dir(work)
    control_sid = work / 'control-sid.json'
    def comprobar_copia():
        return verificar_copia(instancia_copia, state, control_sid)
    comprobar_copia()
    intento_entorno = work / 'entorno-intento.json'
    publicar_entorno_retenido(copia, r, state, intento_entorno)
    if copia.sha256(env_path) != state['entorno_modulos_sha256']:
        fail('entorno_modificado')
    environment = json.loads(private_file(env_path))
    if a.sin_inscripcion and environment.get('VEC_BOLSA_INSCRIPCIONES_ENABLED') == 'true':
        fail('clasificacion_con_inscripcion_previamente_activa')
    log = work / 'preparacion.log'
    if not log.exists():
        write(log, b'')
    private_file(log)

    def rrhh_stage():
        identidad = json.loads(private_file(material / 'identidad/identidad.json'))
        bolsa = json.loads(private_file(material / 'identidad/bolsa-bback.json'))
        sujeto, certificado = identidad.get('subject'), identidad.get('certificate_sha256')
        if (identidad.get('autoridad') != 'no_autoritativo' or identidad.get('roles') != ['tecnico_rrhh']
                or not isinstance(sujeto, str) or not sujeto or not isinstance(certificado, str)
                or not re.fullmatch('[0-9a-f]{64}', certificado)
                or bolsa.get('sujeto') != sujeto or bolsa.get('certificado_sha256') != certificado):
            fail('identidad_rrhh_primaria_invalida')
        # Contrato existente: referenciaAltaContratacionTemporalDesarrollo,
        # contratacion_temporal_alta_sellado_desarrollo.go. No crea una cuenta.
        base = sujeto + '\0' + certificado
        def referencia(prefix, discriminator):
            return prefix + hashlib.sha256(('vec.ct.alta.desarrollo.v1\0' + base + '\0' + discriminator).encode()).hexdigest()[:32]
        cuenta, perfil = referencia('cta_', 'cuenta'), referencia('prf_', 'perfil')
        keys = {
            'dsn_registro_identidad': 'VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL',
            'dsn_revalidacion_identidad': 'VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL',
            'dsn_contexto': 'VEC_CT_CONTEXTO_ACTOR_DATABASE_URL',
            'dsn_fuente_autorizacion': 'VEC_AUTORIZACION_FUENTE_DATABASE_URL',
            'dsn_registro_autorizacion': 'VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL',
            'dsn_motivos': 'VEC_AUTORIZACION_MOTIVOS_EVALUADOR_DATABASE_URL',
        }
        conexiones = {key: environment.get(name) for key, name in keys.items()}
        if any(not isinstance(value, str) or not value for value in conexiones.values()) or \
                len({urlsplit(value).username for value in conexiones.values()}) != 6:
            fail('conexiones_rrhh_no_separadas')
        motivos = json.loads((repo / 'deploy/principal/inscripcion_recorrido_20261013/motivos_autorizacion_v1.json').read_text())
        if set(motivos) != {'decidir', 'incorporar'}:
            fail('motivos_rrhh_fuente_invalida')
        decidir, incorporar = motivos['decidir'], motivos['incorporar']
        if (decidir.get('catalogo_id') != incorporar.get('catalogo_id') or
                decidir.get('catalogo_version') != incorporar.get('catalogo_version') or
                decidir.get('catalogo_huella_sha256') != incorporar.get('catalogo_huella_sha256') or
                decidir.get('entrada_clave') == incorporar.get('entrada_clave') or
                not re.fullmatch('[a-z_]+', decidir['catalogo_id']) or
                not isinstance(decidir['catalogo_version'], int) or
                not re.fullmatch('[0-9a-f]{64}', decidir['catalogo_huella_sha256']) or
                any(not re.fullmatch('[a-z0-9_]+', motivos[k]['entrada_clave']) for k in motivos)):
            fail('motivos_rrhh_fuente_invalida')
        # Solo SELECT: exige cuenta, perfil, catálogo y entradas publicados.
        cat, ver, sha = decidir['catalogo_id'], decidir['catalogo_version'], decidir['catalogo_huella_sha256']
        sql = ("BEGIN READ ONLY; SET LOCAL statement_timeout='5s'; SELECT "
               "(SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_ref='" + cuenta + "')=1 AND "
               "(SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref='" + perfil + "')=1 AND "
               "(SELECT count(*) FROM vec_autorizacion.motivo_v2_catalogo_publicado WHERE catalogo_id='" + cat +
               "' AND catalogo_version=" + str(ver) + " AND catalogo_huella_publicada_sha256='" + sha + "')=1 AND "
               "(SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada WHERE catalogo_id='" + cat +
               "' AND catalogo_version=" + str(ver) + " AND entrada_clave IN ('" + decidir['entrada_clave'] +
               "','" + incorporar['entrada_clave'] + "'))=2 AND NOT EXISTS "
               "(SELECT 1 FROM vec_autorizacion.motivo_v2_retirada WHERE catalogo_id='" + cat +
               "' AND catalogo_version=" + str(ver) + "); ROLLBACK;")
        comprobar_copia()
        probe = subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', state['nombre'], 'psql', '-XAt',
                                '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres'],
                               input=sql.encode(), capture_output=True, timeout=20, check=False)
        if probe.returncode or 't' not in probe.stdout.decode().splitlines():
            with log.open('ab') as output:
                output.write(probe.stderr)
            fail('preimagen_rrhh_no_acreditada')
        sources = work / 'inscripcion-rrhh'
        sources.mkdir(mode=0o700, exist_ok=True)
        private_dir(sources)
        write_or_match(sources / 'cuentas.json', {'cuentas': [{
            'certificado_sha256': certificado, 'sujeto': sujeto, 'cuenta_ref': cuenta, 'perfil_ref': perfil}]})
        write_or_match(sources / 'conexiones.json', conexiones)
        write_or_match(sources / 'motivos.json', motivos)
        write_or_match(material / 'motivos_autorizacion_v1.json', motivos)
        invoke([sys.executable, repo / 'scripts/operacion/inscripcion_bolsa/preparar_accesos_rrhh.py',
                '--material-dir', material, '--identidad', sources / 'cuentas.json',
                '--conexiones', sources / 'conexiones.json', '--motivos', sources / 'motivos.json'], log)
        private_file(material / 'identidad/inscripcion-bolsa-rrhh.json')

    if a.solo_accesos_rrhh:
        rrhh_stage()
        print('Material RRHH de inscripción preparado; selector sin cambios.')
        return
    clone = module(repo / 'scripts/recorridos/clon_material.py', 'clon_material_portal')
    internal_candidate = material / 'identidad/bolsa-candidato.json'
    external = STATE / ('portal-externo-material-' + suffix)
    refs_path = work / 'referencias.json'

    def material_stage():
        if internal_candidate.exists() or external.exists() or refs_path.exists():
            fail('material_parcial_revisar_antes_de_reintentar')
        # El certificado del aspirante nuevo queda en el host. La copia externa
        # recibirá solamente su parte pública a través del productor existente.
        subject = ref('per_')
        refs = {'cuenta_interna': ref('cta_'), 'cuenta_externa': ref('cta_'),
                'persona': ref('per_'), 'perfil_interno': ref('prf_'),
                'perfil_externo': ref('prf_'), 'candidato': ref('can_'),
                'sujeto': subject, 'provision': ref('pce_'), 'contexto': ref('vca_'),
                'vinculo': ref('vin_'), 'procedencia': ref('prc_')}
        if len(set(refs.values())) != len(refs) or not all(REF.fullmatch(x) for x in refs.values()):
            fail('referencias_invalidas')
        clone.certificate(material, 'candidato', subject, days=89)
        cert = subprocess.run(['openssl', 'x509', '-in', str(material / 'mtls/candidato.crt'), '-outform', 'DER'],
                              capture_output=True, check=True, timeout=10).stdout
        identity = {'version': 1, 'autoridad': 'no_autoritativo', 'certificate_sha256': hashlib.sha256(cert).hexdigest(),
                    'subject': subject, 'display_name': 'Elena Martín Campos', 'roles': ['candidato_bolsa']}
        candidate = {'version': 1, 'autoridad': 'no_autoritativo', 'candidato_ref': refs['candidato'],
                     'certificado': 'mtls/candidato.crt', 'cuenta_ref': refs['cuenta_interna'],
                     'identidad': 'identidad/candidato.json', 'perfil_ref': refs['perfil_interno'],
                     'persona_ref': refs['persona'], 'sujeto': subject}
        write(material / 'identidad/candidato.json', identity)
        write(internal_candidate, candidate)
        write(refs_path, refs)
        invoke([sys.executable, repo / 'scripts/operacion/inscripcion_bolsa/preparar_material_externo.py',
                '--material-interno', material, '--material-externo', external,
                '--perfil-externo-ref', refs['perfil_externo'], '--cuenta-externa-ref', refs['cuenta_externa'],
                '--aplicar'], log)

    step(work, 'material', material_stage)
    refs = json.loads(private_file(refs_path))
    if not external.exists():
        fail('material_externo_ausente')

    def source_stage():
        now = dt.datetime.now(dt.timezone.utc)
        start = now.strftime('%Y-%m-%dT%H:%M:%S.000000Z')
        end = (now + dt.timedelta(days=89)).strftime('%Y-%m-%dT%H:%M:%S.000000Z')
        proof = hashlib.sha256(b'VEC fuente ficticia local F0-28').hexdigest()
        def component(reference):
            return {'referencia': reference, 'version': 1, 'procedencia_ref': refs['procedencia'],
                    'procedencia_version': 1, 'procedencia_huella_sha256': proof,
                    'procedencia_autoridad': 'autoridad_maestra_acreditada', 'estado': 'activo',
                    'vigente_desde': start, 'vigente_hasta': end}
        snapshot = {'provision_ref': refs['provision'], 'poblacion': 'candidato', 'estado': 'activo',
                    'cuenta': component(refs['cuenta_externa']), 'persona': component(refs['persona']),
                    'perfil': component(refs['perfil_externo']), 'contexto': component(refs['contexto']),
                    'vinculo_candidato': dict(component(refs['vinculo']), candidato_ref=refs['candidato'])}
        write(work / 'fuente-identidad-contexto.json', {'version': 1, 'snapshot': snapshot,
              'preimagen': {'revision_control_rol': 0, 'huella_control_rol': '',
                            'version_asignacion': 0, 'huella_asignacion': '',
                            'version_contexto': 0, 'huella_contexto': '', 'secuencia_motivos': 0}})
        write(work / 'procedencia-copia.json', {'version': 1, 'tipo': 'dato_sintetico_de_copia',
              'fuente': 'F0-28', 'alcance': 'solo_ensayo_local', 'gobierno_real_acreditado': False,
              'nota': 'El enum del contrato ContextoActor exige autoridad_maestra_acreditada; este manifiesto declara que la fuente de ensayo es ficticia.'})
        # Exporta con la identidad del proceso externo y el material segregado.
        export_env = base_env()
        export_env.update({'VEC_PORTAL_PROCESO': 'externo', 'VEC_EXECUTION_PROFILE': 'desarrollo',
                           'VEC_AUTH_MODE': 'desarrollo',
                           'VEC_DEVELOPMENT_GUARD': 'ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO',
                           'VEC_DEVELOPMENT_MATERIAL_DIR': str(external),
                           'VEC_MATERIAL_PORTAL_EXTERNO_DIR': str(external)})
        export_env.pop('VEC_PORTAL_PROCESO_INTERNO', None)
        alias = work / 'alias-external.json'
        comprobar_copia()
        instancia_copia.validar_binario(state)
        with alias.open('xb') as out, log.open('ab') as errors:
            done = subprocess.run([str(r / 'vec-server'), 'exportar-seudonimos-portal-externo'],
                                  cwd=repo, env=export_env, stdout=out, stderr=errors, timeout=30)
        alias.chmod(0o600)
        if done.returncode or not json.loads(private_file(alias)).get('cuentas'):
            fail('alias_externo_no_exportado')

    step(work, 'fuente', source_stage)
    roles_path = work / 'conexiones-externas.json'
    mapping = {
        'VEC_BOLSA_INSCRIPCIONES_RRHH_LECTOR_DATABASE_URL': ('vec_bolsa_inscripciones_rrhh_lector', 'vec_bolsa_llamamientos_lector_inscripciones_rrhh'),
        'VEC_EXTERNO_BOLSA_DATABASE_URL': ('vec_externo_bolsa_desarrollo', 'vec_bolsa_llamamientos_portal_externo'),
        'VEC_EXTERNO_AUTORIZACION_FUENTE_DATABASE_URL': ('vec_externo_v3_fuente_autorizacion_desarrollo', 'vec_autorizacion_fuente_externa'),
        'VEC_EXTERNO_AUTORIZACION_REGISTRO_DATABASE_URL': ('vec_externo_v3_registro_autorizacion_desarrollo', 'vec_autorizacion_registro_externo'),
        'VEC_EXTERNO_AUTORIZACION_MOTIVOS_DATABASE_URL': ('vec_externo_v3_motivos_autorizacion_desarrollo', 'vec_autorizacion_motivos_externos'),
        'VEC_EXTERNO_PREFLIGHT_V3_DATABASE_URL': ('vec_externo_preflight_v3_desarrollo', 'vec_autorizacion_atestada_v3_preflight_externo'),
        'VEC_EXTERNO_BOLSA_INSCRIPCIONES_LECTOR_DATABASE_URL': ('vec_bolsa_inscripciones_lector', 'vec_bolsa_llamamientos_lector_inscripciones'),
        'VEC_EXTERNO_IDENTIDAD_REGISTRO_DATABASE_URL': ('vcl_ext_identidad_registro', 'vec_identidad_externa_v1_registrador'),
        'VEC_EXTERNO_IDENTIDAD_REVALIDACION_DATABASE_URL': ('vcl_ext_identidad_revalidacion', 'vec_identidad_externa_v1_revalidador'),
        'VEC_EXTERNO_CONTEXTO_DATABASE_URL': ('vcl_ext_contexto_runtime', 'vec_contexto_actor_v1_candidato_externo'),
        'VEC_EXTERNO_BOLSA_FRONTERA_DATABASE_URL': ('vcl_ext_bolsa_frontera', 'vec_bolsa_llamamientos_registrador_portal_externo'),
        'VEC_EXTERNO_CALENDARIOS_DATABASE_URL': ('vcl_ext_calendarios', 'vec_calendarios_lector'),
    }
    governance = {
        'identidad': ('vcl_ext_gob_identidad', 'vec_identidad_sesiones_v1_propietario'),
        'contexto': ('vcl_ext_gob_contexto', 'vec_contexto_actor_v1_propietario'),
        'asignacion': ('vcl_ext_gob_asignacion', 'vec_autorizacion_publicador_candidato_externo'),
        'confianza': ('vec_confianza_externa_publicador_f028', 'vec_autorizacion_atestada_v3_publicador_externo'),
    }

    def roles_stage():
        if roles_path.exists():
            fail('roles_parciales_reconciliar')
        plans = {}
        sql = ['BEGIN;', "SET LOCAL lock_timeout='5s';", "SET LOCAL statement_timeout='30s';"]
        expiration = (dt.datetime.now(dt.timezone.utc) + dt.timedelta(days=90)).strftime('%Y-%m-%dT%H:%M:%S.000000Z')
        for variable, (login, group) in mapping.items():
            plans[variable] = (login, group, False)
        for name, (login, group) in governance.items():
            plans['gobierno_' + name] = (login, group, group.endswith('_propietario'))
        dsns = {}
        for variable, (login, group, can_set) in plans.items():
            if not re.fullmatch(r'[a-z][a-z0-9_]{10,63}', login) or not re.fullmatch(r'vec_[a-z0-9_]+', group):
                fail('rol_invalido')
            secret = secrets.token_hex(32)
            sql.append("DO $pre$ BEGIN IF to_regrole('" + group + "') IS NULL OR to_regrole('" + login + "') IS NOT NULL THEN RAISE EXCEPTION 'rol_preimagen'; END IF; END $pre$;")
            sql.append("CREATE ROLE " + login + " LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '" + secret + "' VALID UNTIL '" + expiration + "';")
            sql.append('GRANT ' + group + ' TO ' + login + ' WITH INHERIT TRUE, SET ' + ('TRUE' if can_set else 'FALSE') + ', ADMIN FALSE;')
            dsns[variable] = dsn(login, secret, state['puerto_pg'], material / 'ca/ca.crt')
        # Son admisiones técnicas sintéticas de la copia, no un acto de gobierno.
        for variable in ('VEC_EXTERNO_BOLSA_DATABASE_URL',
                         'VEC_EXTERNO_AUTORIZACION_FUENTE_DATABASE_URL',
                         'VEC_EXTERNO_AUTORIZACION_REGISTRO_DATABASE_URL',
                         'VEC_EXTERNO_AUTORIZACION_MOTIVOS_DATABASE_URL'):
            login, group = mapping[variable]
            sql.append("INSERT INTO vec_identidad_externa_v1.llamante_autorizacion(login,grupo_esperado,aprobacion_ref,aprobada_por,aprobada_en) VALUES('"
                       + login + "','" + group + "','" + ref('apr_') + "','postgres',pg_catalog.clock_timestamp());")
        sql.append('COMMIT;')
        write(roles_path, dsns)
        pg(r, state, '\n'.join(sql), log, comprobar_copia)
        write(work / 'gobierno-identidad.dsn', (dsns['gobierno_identidad'] + '\n').encode())
        write(work / 'gobierno-contexto.dsn', (dsns['gobierno_contexto'] + '\n').encode())
        write(work / 'gobierno-asignacion.dsn', (dsns['gobierno_asignacion'] + '\n').encode())
        write(work / 'publicador-confianza.dsn', (dsns['gobierno_confianza'] + '\n').encode())

    step(work, 'roles', roles_stage)
    dsns = json.loads(private_file(roles_path))

    def provision_stage():
        binary = work / 'vec-provisionar-candidato-externo'
        if not binary.exists():
            copia.verificar_fuente(state['revision'])
            build_env = dict(base_env(), GOPROXY='off', GOSUMDB='off', GOCACHE='/dev/shm/vec/go-build', GOTOOLCHAIN='local')
            invoke([go, 'build', '-p', '6', '-o', binary, './cmd/vec-provisionar-candidato-externo'],
                   log, env=build_env, cwd=repo)
            binary.chmod(0o700)
        gov_env = dict(base_env(), VEC_PORTAL_PROCESO='interno', VEC_EXECUTION_PROFILE='desarrollo',
                       VEC_AUTH_MODE='desarrollo',
                       VEC_DEVELOPMENT_GUARD='ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO',
                       VEC_DEVELOPMENT_MATERIAL_DIR=str(material))
        source = work / 'fuente-identidad-contexto.json'
        for phase in ('identidad', 'contexto'):
            receipt = work / ('provision-' + phase + '.json')
            if receipt.exists():
                private_file(receipt)
                continue
            args = [str(binary), '--fuente', str(source), '--fase', phase,
                    '--dsn-archivo', str(work / ('gobierno-' + phase + '.dsn'))]
            if phase == 'identidad':
                args += ['--alias-archivo', str(work / 'alias-external.json')]
            comprobar_copia()
            with log.open('ab') as errors:
                plan = subprocess.run(args, cwd=repo, env=gov_env, stdout=subprocess.PIPE,
                                      stderr=errors, timeout=45, check=False)
            if plan.returncode:
                fail('plan_provision_rechazado')
            output = json.loads(plan.stdout)
            if not re.fullmatch('[0-9a-f]{64}', output.get('huella_sha256', '')) or not re.fullmatch('[0-9a-f]{64}', output.get('preimagen_sha256', '')):
                fail('plan_provision_invalido')
            args += ['--aprobar', output['huella_sha256'], '--preimagen', output['preimagen_sha256']]
            comprobar_copia()
            with log.open('ab') as errors:
                done = subprocess.run(args, cwd=repo, env=gov_env, stdout=subprocess.PIPE,
                                      stderr=errors, timeout=45, check=False)
            if done.returncode:
                fail('provision_reconciliar_preimagen')
            result = json.loads(done.stdout)
            if result.get('estado') not in ('publicado', 'confirmado'):
                fail('recibo_provision_no_confirmado')
            write(receipt, result)

    step(work, 'provision', provision_stage)
    # HZ37 tiene ya una cabeza gobernada; la fase `autorizacion` del CLI
    # intentaría publicar v1. Solo se publica una asignación nueva a esa cabeza.
    def assignment_stage():
        receipt = work / 'asignacion-externa.json'
        intent_path = work / 'asignacion-intento.json'
        perfil, persona, candidato = refs['perfil_externo'], refs['persona'], refs['candidato']
        if not all(REF.fullmatch(x) for x in (perfil, persona, candidato)):
            fail('referencias_asignacion_invalidas')

        def leer_actual():
            comprobar_copia()
            sql = ("SELECT jsonb_build_object('asignacion_ref',x.asignacion_ref,"
                   "'huella_sha256',x.huella_sha256,'documento',x.documento,"
                   "'acto_ref',a.acto_ref,'perfil_ref',x.perfil_activo_ref,"
                   "'principal_id',x.principal_id,'rol_ref',x.version_rol_ref,"
                   "'evento',(SELECT jsonb_build_object('objeto_ref',e.objeto_ref,"
                   "'huella_nueva',e.huella_nueva,'version',e.version)"
                   " FROM vec_autorizacion.publicacion_candidato_externo_evento e"
                   " WHERE e.tipo='asignacion' AND e.acto_ref=a.acto_ref))::text"
                   " FROM vec_autorizacion.asignacion_perfil_actual_externa a"
                   " JOIN vec_autorizacion.asignacion_perfil_externa x USING(perfil_activo_ref,asignacion_ref)"
                   " WHERE a.perfil_activo_ref='" + perfil + "'")
            q = subprocess.run(['docker', 'exec', '-u', 'postgres', state['nombre'], 'psql', '-XAt',
                                '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres', '-c', sql],
                               capture_output=True, timeout=20, check=False)
            if q.returncode:
                with log.open('ab') as f: f.write(q.stderr)
                fail('lectura_asignacion_no_disponible')
            filas = q.stdout.decode().splitlines()
            if len(filas) > 1:
                fail('asignacion_ambigua')
            return json.loads(filas[0]) if filas else None

        actual = leer_actual()
        if intent_path.exists():
            intent = json.loads(private_file(intent_path))
            document = intent.get('documento')
            datos_documento = json.loads(document) if isinstance(document, str) else {}
            if (intent.get('version') != 1 or intent.get('tipo') != 'semilla_sintetica_de_copia'
                    or not isinstance(document, str) or not re.fullmatch('[0-9a-f]{64}', intent.get('huella_sha256', ''))
                    or hashlib.sha256(document.encode()).hexdigest() != intent['huella_sha256']
                    or intent.get('rol_ref') != 'rol:candidato_bolsa_portal_historial_propio_desarrollo:v1'
                    or intent.get('perfil_ref') != perfil or intent.get('principal_id') != persona
                    or intent.get('candidato_ref') != candidato
                    or intent.get('asignacion_ref') != 'asignacion:' + intent.get('asignacion_id', '') + ':v1'
                    or not REF.fullmatch(intent.get('asignacion_id', ''))
                    or not REF.fullmatch(intent.get('acto_ref', ''))
                    or datos_documento.get('asignacion_id') != intent.get('asignacion_id')
                    or datos_documento.get('version') != 1 or datos_documento.get('estado') != 'activa'
                    or datos_documento.get('perfil_activo_ref') != perfil
                    or datos_documento.get('principal_id') != persona
                    or datos_documento.get('version_rol_ref') != intent.get('rol_ref')
                    or datos_documento.get('ambitos') != [{'clave': 'candidato_ref', 'valores': [candidato]}]
                    or datos_documento.get('emitida_por') != 'identidad:desarrollo:no-autoritativa'):
                fail('intento_asignacion_incompatible')
        else:
            if actual is not None:
                fail('asignacion_preexistente_requiere_conciliacion_admin')
            comprobar_copia()
            rol_sql = """SELECT vr.version_rol_ref||'|'||coalesce(c.estado,'') FROM vec_autorizacion.version_rol vr
LEFT JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
LEFT JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
WHERE vr.rol_id='candidato_bolsa_portal_historial_propio_desarrollo'
ORDER BY vr.version DESC LIMIT 1"""
            q = subprocess.run(['docker', 'exec', '-u', 'postgres', state['nombre'], 'psql', '-XAt',
                                '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres', '-c', rol_sql],
                               capture_output=True, timeout=20, check=False)
            if q.returncode:
                with log.open('ab') as f: f.write(q.stderr)
                fail('cabeza_hz37_no_disponible')
            cabeza = q.stdout.decode().strip()
            if cabeza != 'rol:candidato_bolsa_portal_historial_propio_desarrollo:v1|habilitada':
                fail('requiere_publicacion_por_administracion')
            helper = work / 'canon-asignacion'
            helper.mkdir(mode=0o700, exist_ok=True)
            build_env = dict(base_env(), GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOCACHE='/dev/shm/vec/go-build')
            copia.verificar_fuente(state['revision'])
            invoke([go, 'build', '-p', '6', '-o', helper / 'canon-asignacion',
                    './deploy/principal/portal_copia/canon_asignacion'],
                   log, env=build_env, cwd=repo)
            asignacion_id, acto_ref = ref('asg_'), ref('act_')
            args = [str(helper / 'canon-asignacion'), persona, perfil, candidato,
                    'rol:candidato_bolsa_portal_historial_propio_desarrollo:v1', asignacion_id]
            with log.open('ab') as errors:
                produced = subprocess.run(args, stdout=subprocess.PIPE, stderr=errors, timeout=20, check=False)
            if produced.returncode:
                fail('canon_asignacion_rechazado')
            canonical = json.loads(produced.stdout)
            document = canonical['documento']
            fingerprint = canonical['huella_sha256']
            if hashlib.sha256(document.encode()).hexdigest() != fingerprint:
                fail('canon_asignacion_no_cuadra')
            intent = {'version': 1, 'tipo': 'semilla_sintetica_de_copia', 'documento': document,
                      'huella_sha256': fingerprint, 'asignacion_id': asignacion_id,
                      'asignacion_ref': 'asignacion:' + asignacion_id + ':v1', 'acto_ref': acto_ref,
                      'rol_ref': 'rol:candidato_bolsa_portal_historial_propio_desarrollo:v1',
                      'perfil_ref': perfil, 'principal_id': persona, 'candidato_ref': candidato}
            write(intent_path, intent)  # Duradero antes del primer efecto.

        if actual is not None:
            confirmar_asignacion_retenida(actual, intent, True)
        else:
            comprobar_copia()
            rol_sql = "SELECT max(version) FROM vec_autorizacion.version_rol WHERE rol_id='candidato_bolsa_portal_historial_propio_desarrollo'"
            q = subprocess.run(['docker', 'exec', '-u', 'postgres', state['nombre'], 'psql', '-XAt',
                                '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres', '-c', rol_sql],
                               capture_output=True, timeout=20, check=False)
            if q.returncode or q.stdout.decode().strip() != '1':
                fail('requiere_publicacion_por_administracion')
            conn = urlsplit(dsns['gobierno_asignacion'])
            query = parse_qs(conn.query)
            postgres_env = dict(base_env(), PGHOST=conn.hostname, PGPORT=str(conn.port),
                                PGUSER=conn.username, PGPASSWORD=conn.password,
                                PGSSLMODE=query['sslmode'][0], PGSSLROOTCERT=query['sslrootcert'][0])
            sql = "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL timezone='UTC'; " + \
                  "SET LOCAL statement_timeout='15s'; SELECT asignacion_ref,version,huella_sha256 FROM " + \
                  "vec_autorizacion.publicar_asignacion_candidato_externo_v1(decode('" + intent['documento'].encode().hex() + \
                  "','hex'),'" + intent['huella_sha256'] + "',0,NULL,'identidad:desarrollo:no-autoritativa','" + \
                  intent['acto_ref'] + "'); COMMIT;"
            comprobar_copia()
            try:
                with log.open('ab') as errors:
                    result = subprocess.run(['docker', 'run', '--rm', '-i', '--pull=never', '--network=host',
                        '--memory=128m', '--cap-drop=ALL', '--user', str(os.getuid()) + ':' + str(os.getgid()),
                        '-v', str(material / 'ca/ca.crt') + ':/ca.crt:ro', '-e', 'PGHOST', '-e', 'PGPORT',
                        '-e', 'PGUSER', '-e', 'PGPASSWORD', '-e', 'PGSSLMODE', '-e', 'PGSSLROOTCERT=/ca.crt',
                        'postgres:18.4', 'psql', '-XAt', '-v', 'ON_ERROR_STOP=1', '-d', 'postgres'],
                        input=sql.encode(), stdout=subprocess.PIPE, stderr=errors,
                        env=postgres_env, timeout=30, check=False)
                confirmado = result.returncode == 0 and intent['huella_sha256'].encode() in result.stdout
            except subprocess.TimeoutExpired:
                confirmado = False
            actual = leer_actual()  # Reconcilia también un COMMIT con respuesta perdida.
            confirmar_asignacion_retenida(actual, intent, confirmado)
        write_or_match(receipt, {'tipo': 'semilla_sintetica_de_copia', 'version': 1,
                               'gobierno_real_acreditado': False, 'rol_ref': intent['rol_ref'],
                               'asignacion_ref': intent['asignacion_ref'], 'acto_tecnico_ref': intent['acto_ref'],
                               'asignacion_huella_sha256': intent['huella_sha256']})

    step(work, 'asignacion', assignment_stage)

    def trust_stage():
        prepared = work / 'confianza-externa.json'
        if prepared.exists():
            fail('confianza_parcial_reconciliar')
        trust_env = base_env()
        trust_env.update(environment)
        trust_env.update({'VEC_PORTAL_PROCESO': 'interno', 'VEC_EXECUTION_PROFILE': 'desarrollo',
                          'VEC_AUTH_MODE': 'desarrollo',
                          'VEC_DEVELOPMENT_GUARD': 'ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO',
                          'VEC_DEVELOPMENT_MATERIAL_DIR': str(material),
                          'VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS': '2136',
                          'VEC_CT_GOBIERNO_DATABASE_URL': environment['VEC_CT_GOBIERNO_DATABASE_URL']})
        consumidores = ['mi_bolsa', 'preferencias_bolsas']
        if not a.sin_inscripcion:
            consumidores.append('inscripcion_bolsa')
        args = [r / 'vec-server', 'preparar-portal-externo', '--material-externo', external,
                '--consumidores', ','.join(consumidores),
                '--publicador-confianza-dsn', work / 'publicador-confianza.dsn',
                '--seudonimos', work / 'alias-external.json', '--cuentas', refs['cuenta_externa']]
        comprobar_copia()
        instancia_copia.validar_binario(state)
        invoke(args, log, env=trust_env, cwd=repo)
        if not (external / 'externo/v3').is_dir():
            fail('confianza_externa_ausente')
        write(prepared, {'tipo': 'preparacion_local_no_autoritativa', 'consumidores': consumidores})

    step(work, 'confianza', trust_stage)
    step(work, 'accesos_rrhh', rrhh_stage)

    def config_stage():
        if not (material / 'auditoria-intentos-externa.json').exists() or not (material / 'auditoria-intentos-externa.dsn').exists():
            fail('auditoria_externa_ausente')
        original = json.loads(private_file(material / 'auditoria-intentos-externa.json'))
        source_dsn = private_file(material / 'auditoria-intentos-externa.dsn').decode().strip()
        if original.get('dsn_file') != 'auditoria-intentos-externa.dsn' or source_dsn == private_file(material / 'auditoria-intentos.dsn').decode().strip():
            fail('auditoria_externa_comparte_login')
        src = urlsplit(source_dsn)
        ext_audit = dsn(src.username, src.password, state['puerto_pg'], external / 'ca/ca.crt')
        if not (external / 'auditoria-intentos-externa.json').exists():
            write(external / 'externo/auditoria-intentos.dsn', (ext_audit + '\n').encode())
            original['dsn_file'] = 'externo/auditoria-intentos.dsn'
            write(external / 'auditoria-intentos-externa.json', original)
        new_env = entorno_portal(environment, dsns, external, a.sin_inscripcion)
        if json.loads(private_file(env_path)) != environment or copia.sha256(env_path) != state['entorno_modulos_sha256']:
            fail('entorno_preimagen_cambio')
        publicar_entorno_retenido(copia, r, state, intento_entorno, new_env)

    step(work, 'entorno', config_stage)
    if a.sin_inscripcion and 'VEC_BOLSA_INSCRIPCIONES_ENABLED' in json.loads(private_file(env_path)):
        fail('clasificacion_con_inscripcion_activa')
    print('Portal externo preparado para ensayo serial. Falta la fuente de convocatoria F1-40 y el recorrido visible; la candidatura se incorpora a Bolsa mediante RRHH tras su inscripción.')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--raiz', type=Path, required=True)
    p.add_argument('--candado-fd', type=int)
    p.add_argument('--solo-accesos-rrhh', action='store_true')
    p.add_argument('--sin-inscripcion', action='store_true')
    a = p.parse_args()
    r = a.raiz
    if not r.is_absolute() or r.resolve() != r or not r.is_relative_to(BASE):
        fail('raiz_fuera_de_copias')
    private_dir(r)
    with candado_copia(r, a.candado_fd):
        main_locked(a, r)


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        # Los detalles pueden contener rutas y DSN: solo código público.
        print('preparar_portal_fallido; consulte el log privado y revise el estado parcial', file=sys.stderr)
        sys.exit(1)
