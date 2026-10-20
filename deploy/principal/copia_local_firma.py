"""Firma compuesta de una copia local: etapas privadas, plantillas v4, verificador y entorno.

La orden `firma` de copia_local.py llama a `preparar`. Las etapas del camino
largo (Administración, cargos, actos, plan y admisiones) dependen de material
privado y se declaran en un manifiesto fuera de Git; aquí solo se ejecutan en
orden y se registran. Lo común a toda copia sí vive aquí: publicar la v4 de
plantillas, comprobar `identidad/firma-vec.json`, instalar el verificador
GrxFirma en loopback y encender las variables de firma.

El verificador solo verifica (`-rest-solo-verificacion`); nunca firma. La
aplicación no da nada por firmado sin su dictamen.
"""
import datetime as dt
import email.utils
import hashlib
import http.client
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import signal
import ssl
import stat
import subprocess
import tempfile
import time

CLAVES_MANIFIESTO = {'version', 'etapas', 'verificador'}
CLAVES_VERIFICADOR = {'binario', 'binario_sha256', 'puerto'}
NOMBRE_ETAPA = re.compile(r'[a-z0-9][a-z0-9-]{0,40}')
MAXIMO_ETAPAS = 40
PLAZO_ETAPA = 1800
RECURSOS = ('plan-firma-publicado.json', 'circuito_rrhh_v2.json',
            'ct_plantillas_documentos.v4.json', 'clientes.crl', 'ct_subsanacion_politica.json')
# Fuentes del repositorio que la aplicación lee desde una copia privada.
FUENTES_REPO = {
    'circuito_rrhh_v2.json': 'config/contratacion_temporal_circuito_rrhh_v2.json',
    'ct_plantillas_documentos.v4.json': 'data/catalogos/contratacion-temporal/plantillas_documentos.v4.json',
    # F1-26: la política sintética de subsanación que ya usa la principal
    # (su motivo está publicado en el respaldo HZ37), sin actor ni secreto.
    'ct_subsanacion_politica.json': 'config/contratacion_temporal_subsanacion_politica_desarrollo.json',
}
OPERACION_V4 = 'PUBLICA-INFORME-PREVIO-V4'
CLAVES_PLAN = {'archivo', 'referencia', 'version', 'huella_sha256'}


def _sha256(ruta):
    with Path(ruta).open('rb') as fuente:
        return hashlib.file_digest(fuente, 'sha256').hexdigest()


def privado(ruta, *, directorio=False, ejecutable=False, padres=False):
    """Ruta absoluta, sin enlaces, del usuario y sin permisos de grupo ni otros.

    Con `padres`, además ninguna carpeta superior es modificable por otras cuentas
    (salvo con bit pegajoso): lo que se ejecuta no se puede cambiar entre la
    comprobación y el uso.
    """
    ruta = Path(ruta)
    if not ruta.is_absolute() or '..' in ruta.parts:
        raise ValueError('Ruta privada no absoluta')
    for parte in (*reversed(ruta.parents), ruta):
        if parte.is_symlink():
            raise ValueError('Ruta privada con enlace simbólico')
    for padre in ruta.parents if padres else ():
        datos = padre.stat()
        if (datos.st_uid not in (0, os.getuid())
                or (datos.st_mode & 0o022 and not datos.st_mode & stat.S_ISVTX)):
            raise ValueError(f'{padre}: carpeta modificable por otras cuentas')
    datos = ruta.stat()
    tipo = stat.S_ISDIR(datos.st_mode) if directorio else stat.S_ISREG(datos.st_mode)
    if not tipo or datos.st_uid != os.getuid() or stat.S_IMODE(datos.st_mode) & 0o077:
        raise ValueError(f'{ruta.name}: propietario, tipo o permisos no privados')
    if ejecutable and not datos.st_mode & stat.S_IXUSR:
        raise ValueError(f'{ruta.name}: no es ejecutable')
    return ruta


def leer_manifiesto(ruta, estado):
    """Valida el manifiesto privado. No admite claves desconocidas."""
    manifiesto = json.loads(privado(ruta, padres=True).read_text())
    if not isinstance(manifiesto, dict) or set(manifiesto) != CLAVES_MANIFIESTO or manifiesto['version'] != 1:
        raise ValueError('Manifiesto de firma con claves o versión distintas')
    etapas = manifiesto['etapas']
    if not isinstance(etapas, list) or len(etapas) > MAXIMO_ETAPAS:
        raise ValueError('Etapas de firma no válidas')
    nombres = set()
    for etapa in etapas:
        if (not isinstance(etapa, dict) or set(etapa) != {'nombre', 'orden'}
                or not isinstance(etapa['nombre'], str) or not NOMBRE_ETAPA.fullmatch(etapa['nombre'])
                or etapa['nombre'] in nombres or not isinstance(etapa['orden'], list) or not etapa['orden']
                or any(not isinstance(a, str) or '\x00' in a for a in etapa['orden'])):
            raise ValueError('Etapa de firma no válida')
        privado(etapa['orden'][0], ejecutable=True, padres=True)
        nombres.add(etapa['nombre'])
    verificador = manifiesto['verificador']
    if not isinstance(verificador, dict) or set(verificador) != CLAVES_VERIFICADOR:
        raise ValueError('Verificador del manifiesto no válido')
    puerto = verificador['puerto']
    ocupados = {estado.get('puerto_pg'), estado.get('puerto_app'), estado.get('puerto_smtp')}
    if not isinstance(puerto, int) or isinstance(puerto, bool) or not 1024 <= puerto <= 65535 or puerto in ocupados:
        raise ValueError('Puerto del verificador no válido o compartido con la copia')
    huella = verificador['binario_sha256']
    if not isinstance(huella, str) or not re.fullmatch('[0-9a-f]{64}', huella):
        raise ValueError('Huella del verificador no válida')
    if _sha256(privado(verificador['binario'], ejecutable=True, padres=True)) != huella:
        raise ValueError('El binario del verificador no coincide con su huella')
    return manifiesto


def ejecutar_etapas(copia, etapas):
    """Ejecuta las etapas privadas en orden. Una etapa hecha con la misma orden no se repite."""
    estado = copia.leer()
    carpeta = copia.raiz / 'firma-etapas'
    carpeta.mkdir(mode=0o700, exist_ok=True)
    entorno = {'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8', 'HOME': os.environ['HOME'],
               'VEC_COPIA_NOMBRE': copia.nombre, 'VEC_COPIA_RAIZ': str(copia.raiz),
               'VEC_COPIA_PUERTO_PG': str(estado['puerto_pg']), 'VEC_COPIA_PUERTO_APP': str(estado['puerto_app'])}
    hechas = []
    for etapa in etapas:
        nombre = etapa['nombre']
        # La marca cubre los argumentos y el contenido del ejecutable.
        huella = hashlib.sha256(json.dumps([etapa['orden'], _sha256(etapa['orden'][0])]).encode()).hexdigest()
        marca = carpeta / f'{nombre}.hecha'
        if marca.exists() and marca.read_text().strip() == huella:
            continue
        with (carpeta / f'{nombre}.log').open('ab') as log:
            proceso = subprocess.Popen(etapa['orden'], stdin=subprocess.DEVNULL, stdout=log,
                                       stderr=subprocess.STDOUT, env=entorno, cwd=carpeta,
                                       start_new_session=True)
            try:
                codigo = proceso.wait(timeout=PLAZO_ETAPA)
            except subprocess.TimeoutExpired:
                # Se termina el grupo entero: la etapa no deja procesos hijos vivos.
                os.killpg(proceso.pid, signal.SIGKILL)
                proceso.wait()
                raise ValueError(f'La etapa {nombre} superó su plazo; consulte firma-etapas/{nombre}.log en su copia')
        if codigo:
            raise ValueError(f'La etapa {nombre} falló; consulte firma-etapas/{nombre}.log en su copia')
        marca.write_text(huella + '\n')
        hechas.append(nombre)
    return hechas


def contrasena_pg(copia):
    resultado = subprocess.run(['docker', 'exec', copia.nombre, 'cat', '/run/vec-password'],
                               capture_output=True, check=False, timeout=30)
    clave = resultado.stdout.decode().strip()
    if resultado.returncode or not re.fullmatch('[0-9a-f]{64}', clave):
        raise ValueError('No se pudo leer la credencial de la base de la copia')
    return clave


def publicar_plantillas_v4(copia, repo, go, entorno_go):
    """F1-86: publica la v4 de plantillas (o la reconoce ya publicada) y copia su catálogo."""
    estado = copia.leer()
    binario = copia.raiz / 'publicar-plantillas'
    with (copia.raiz / 'compilacion-plantillas.log').open('wb') as log:
        subprocess.run([go, 'build', '-p', '6', '-o', str(binario), './scripts/operacion/publicar_plantillas_v3'],
                       cwd=repo, env=entorno_go, stdout=log, stderr=subprocess.STDOUT, check=True, timeout=600)
    binario.chmod(0o700)
    url = (f"postgres://postgres:{contrasena_pg(copia)}@localhost:{estado['puerto_pg']}/postgres"
           f"?sslmode=verify-full&sslrootcert={copia.material / 'ca/ca.crt'}")
    entorno = {'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8', 'VEC_PLANTILLAS_V3_BIN': str(binario),
               'VEC_PLANTILLAS_V3_DATABASE_URL': url}
    guion = repo / 'scripts/operacion/firma_ct_tres_pasos/publicar_informe.sh'
    resultado = subprocess.run([str(guion), '--aplicar', OPERACION_V4], env=entorno, capture_output=True,
                               check=False, timeout=120)
    try:
        recibo = json.loads(resultado.stdout)
    except ValueError:
        recibo = {}
    if not isinstance(recibo, dict):
        recibo = {}
    if resultado.returncode or recibo.get('version') != 4 or recibo.get('resultado') not in ('registrado', 'replay'):
        # La salida de error puede contener la conexión: no se reenvía.
        raise ValueError('La publicación de plantillas v4 no se confirmó')
    (copia.raiz / 'plantillas-v4-recibo.json').write_text(json.dumps(recibo, ensure_ascii=False) + '\n')
    return recibo


def instalar_fuentes_repo(copia, repo):
    """Copia el circuito RRHH v2 y el catálogo v4 de esta revisión a los recursos privados."""
    recursos = copia.material / 'identidad/firma-recursos'
    recursos.mkdir(mode=0o700, exist_ok=True)
    for destino, origen in FUENTES_REPO.items():
        shutil.copyfile(repo / origen, recursos / destino)
        (recursos / destino).chmod(0o600)
    return recursos


def preparar(copia, manifiesto_ruta, *, repo, go, entorno_go, guardar):
    """Compone la firma de una copia en marcha. Después hay que reiniciarla."""
    manifiesto = leer_manifiesto(manifiesto_ruta, copia.leer())
    ejecutar_etapas(copia, manifiesto['etapas'])
    publicar_plantillas_v4(copia, repo, go, entorno_go)
    instalar_fuentes_repo(copia, repo)
    recursos = recursos_firma(copia)
    # Un binario en ejecución no se puede sustituir: se para antes de instalar.
    parar_verificador(copia)
    instalar_verificador(copia, manifiesto['verificador'], recursos)
    componer_entorno(copia, entorno_firma(copia, recursos, manifiesto['verificador']['puerto']),
                     manifiesto['verificador'], guardar)


def recursos_firma(copia):
    """Comprueba firma-vec.json y sus recursos en material/identidad/firma-recursos."""
    recursos = copia.material / 'identidad/firma-recursos'
    privado(recursos, directorio=True)
    for nombre in RECURSOS:
        privado(recursos / nombre)
    configuracion = json.loads(privado(copia.material / 'identidad/firma-vec.json').read_text())
    plan = configuracion.get('plan') if isinstance(configuracion, dict) else None
    if (not isinstance(plan, dict) or set(plan) != CLAVES_PLAN
            or Path(str(plan['archivo'])) != recursos / 'plan-firma-publicado.json'
            or Path(str(configuracion.get('crl_path'))) != recursos / 'clientes.crl'
            or not re.fullmatch('[0-9a-f]{64}', str(plan['huella_sha256']))):
        raise ValueError('firma-vec.json no apunta al plan publicado y la CRL de la copia')
    if not configuracion.get('firmantes') or not configuracion.get('admisiones'):
        raise ValueError('firma-vec.json sin firmantes o admisiones')
    retirada = dt.datetime.fromisoformat(str(configuracion.get('retirada_en')).replace('Z', '+00:00'))
    if retirada.tzinfo is None or retirada <= dt.datetime.now(dt.timezone.utc):
        raise ValueError('firma-vec.json ya está retirado')
    return recursos


def _openssl(*args):
    resultado = subprocess.run(['/usr/bin/openssl', *map(str, args)], capture_output=True, timeout=15, check=False)
    if resultado.returncode:
        raise ValueError('Certificado o lista de revocación rechazados')
    return resultado.stdout.decode()


def comprobar_material_verificador(ca, cert, clave, crl):
    _openssl('x509', '-in', ca, '-checkend', '86400', '-noout')
    _openssl('x509', '-in', cert, '-checkend', '86400', '-noout')
    _openssl('verify', '-CAfile', ca, '-purpose', 'sslserver', cert)
    if _openssl('x509', '-in', cert, '-pubkey', '-noout') != _openssl('pkey', '-in', clave, '-pubout'):
        raise ValueError('La clave TLS del verificador no corresponde a su certificado')
    if 'IP Address:127.0.0.1' not in _openssl('x509', '-in', cert, '-ext', 'subjectAltName', '-noout'):
        raise ValueError('El certificado TLS del verificador no cubre 127.0.0.1')
    _openssl('crl', '-in', crl, '-verify', '-CAfile', ca, '-noout')
    fecha = lambda campo: email.utils.parsedate_to_datetime(
        _openssl('crl', '-in', crl, campo, '-noout').strip().split('=', 1)[1])
    ahora = dt.datetime.now(dt.timezone.utc)
    if not fecha('-lastupdate') <= ahora < fecha('-nextupdate'):
        raise ValueError('La lista de revocación de clientes no está vigente')


def instalar_verificador(copia, verificador, recursos):
    """Deja el verificador en <copia>/verificador con material propio y credencial nueva."""
    carpeta = copia.raiz / 'verificador'
    for sub in ('', 'crl', 'home', 'home/config', 'home/cache'):
        (carpeta / sub).mkdir(mode=0o700, exist_ok=True)
    binario = carpeta / 'grxfirma-verificador'
    shutil.copyfile(verificador['binario'], binario)
    binario.chmod(0o700)
    if _sha256(binario) != verificador['binario_sha256']:
        raise ValueError('La copia del verificador no coincide con su huella')
    for origen, destino in ((copia.material / 'ca/ca.crt', 'ca.crt'), (copia.material / 'tls/servidor.crt', 'tls.crt'),
                            (copia.material / 'tls/servidor.key', 'tls.key'),
                            (recursos / 'clientes.crl', 'crl/clientes.crl')):
        shutil.copyfile(origen, carpeta / destino)
        (carpeta / destino).chmod(0o600)
    comprobar_material_verificador(carpeta / 'ca.crt', carpeta / 'tls.crt', carpeta / 'tls.key',
                                   carpeta / 'crl/clientes.crl')
    token = carpeta / 'token'
    if not token.exists():
        descriptor = os.open(token, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, 'w') as salida:
            salida.write(secrets.token_hex(32))
    privado(token)
    return carpeta


def entorno_firma(copia, recursos, puerto):
    carpeta = copia.raiz / 'verificador'
    return {
        'VEC_CT_FIRMA_VEC_R5_ENABLED': 'true',
        'VEC_CT_CIRCUITO_RRHH_SOURCE_PATH': str(recursos / 'circuito_rrhh_v2.json'),
        'VEC_CT_PLANTILLAS_SOURCE_PATH': str(recursos / 'ct_plantillas_documentos.v4.json'),
        # F1-26: subsanación del reparo y ruta de Plantillas de documentos.
        'VEC_CT_SUBSANACION_POLITICA_FILE': str(recursos / 'ct_subsanacion_politica.json'),
        'VEC_CT_PLANTILLAS_GOBIERNO_ENABLED': 'true',
        'VEC_FIRMA_VERIFICACION_ENABLED': 'true',
        'VEC_FIRMA_VERIFICACION_URL': f'https://127.0.0.1:{puerto}',
        'VEC_FIRMA_VERIFICACION_CA_FILE': str(carpeta / 'ca.crt'),
        'VEC_FIRMA_VERIFICACION_TOKEN_FILE': str(carpeta / 'token'),
        'VEC_FIRMA_VERIFICACION_TIMEOUT': '20s',
    }


def componer_entorno(copia, nuevas, verificador, guardar):
    """Añade las variables de firma al entorno de módulos y deja su huella en el estado."""
    entorno = copia.entorno_modulos()
    if not entorno:
        raise ValueError('La firma necesita una copia creada con --modulos')
    entorno.update(nuevas)
    ruta = copia.raiz / 'entorno_modulos.json'
    descriptor, nombre = tempfile.mkstemp(dir=copia.raiz, prefix='.entorno-')
    temporal = Path(nombre)
    try:
        with os.fdopen(descriptor, 'w') as salida:
            salida.write(json.dumps(entorno, ensure_ascii=False, indent=2, sort_keys=True) + '\n')
        os.replace(temporal, ruta)
    finally:
        temporal.unlink(missing_ok=True)
    estado = copia.leer()
    estado['entorno_modulos_sha256'] = _sha256(ruta)
    estado['verificador'] = {'puerto': verificador['puerto'], 'binario_sha256': verificador['binario_sha256']}
    guardar(copia.estado, estado)


def _proceso_verificador(copia, estado):
    datos = estado.get('verificador') or {}
    pid = datos.get('pid')
    if not pid:
        return False
    proc = Path(f'/proc/{pid}')
    try:
        return ((proc / 'exe').resolve() == copia.raiz / 'verificador/grxfirma-verificador'
                and (proc / 'stat').read_text().split()[21] == datos.get('inicio_proceso'))
    except FileNotFoundError:
        return False


def salud_verificador(copia, puerto):
    carpeta = copia.raiz / 'verificador'
    contexto = ssl.create_default_context(cafile=str(carpeta / 'ca.crt'))
    conexion = http.client.HTTPSConnection('127.0.0.1', puerto, context=contexto, timeout=3)
    try:
        conexion.request('GET', '/health', headers={'Authorization': 'Bearer ' + (carpeta / 'token').read_text().strip()})
        respuesta = conexion.getresponse()
        cuerpo = json.loads(respuesta.read(4096) or b'{}')
    finally:
        conexion.close()
    if respuesta.status != 200 or not isinstance(cuerpo, dict) or cuerpo.get('ok') is not True or cuerpo.get('modo') != 'solo_verificacion':
        raise ValueError('El verificador no está en modo de solo verificación')


def arrancar_verificador(copia, guardar, puerto_libre):
    """Arranca el verificador de la copia si la firma está compuesta. Devuelve si lo arrancó."""
    estado = copia.leer()
    datos = estado.get('verificador')
    if not datos:
        return False
    if _proceso_verificador(copia, estado):
        salud_verificador(copia, datos['puerto'])
        return False
    carpeta = copia.raiz / 'verificador'
    binario = privado(carpeta / 'grxfirma-verificador', ejecutable=True)
    if _sha256(binario) != datos['binario_sha256']:
        raise ValueError('El verificador de la copia cambió; se conserva parado')
    # La CRL de clientes se renueva en firma-recursos; el verificador toma la vigente.
    crl = copia.material / 'identidad/firma-recursos/clientes.crl'
    if crl.exists() and _sha256(privado(crl)) != _sha256(carpeta / 'crl/clientes.crl'):
        shutil.copyfile(crl, carpeta / 'crl/clientes.crl')
        (carpeta / 'crl/clientes.crl').chmod(0o600)
    for nombre in ('ca.crt', 'tls.crt', 'tls.key', 'crl/clientes.crl', 'token'):
        privado(carpeta / nombre)
    comprobar_material_verificador(carpeta / 'ca.crt', carpeta / 'tls.crt', carpeta / 'tls.key',
                                   carpeta / 'crl/clientes.crl')
    puerto_libre(datos['puerto'])
    token = (carpeta / 'token').read_text().strip()
    if not re.fullmatch('[0-9a-f]{64}', token):
        raise ValueError('Credencial del verificador no válida')
    home = carpeta / 'home'
    # La credencial va en el entorno del proceso, nunca en sus argumentos.
    entorno = {'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8', 'GRXFIRMA_REST_TOKEN': token, 'HOME': str(home),
               'XDG_CONFIG_HOME': str(home / 'config'), 'XDG_CACHE_HOME': str(home / 'cache')}
    with (copia.raiz / 'verificador.log').open('ab') as log:
        proceso = subprocess.Popen(
            [str(binario), '-rest-solo-verificacion', '-direccion-rest', f"127.0.0.1:{datos['puerto']}",
             '-verificacion-anclas', str(carpeta / 'ca.crt'), '-verificacion-crl', str(carpeta / 'crl'),
             '-rest-tls-cert', str(carpeta / 'tls.crt'), '-rest-tls-key', str(carpeta / 'tls.key')],
            cwd=carpeta, env=entorno, stdin=subprocess.DEVNULL, stdout=log, stderr=subprocess.STDOUT,
            start_new_session=True)
    datos['pid'] = proceso.pid
    datos['inicio_proceso'] = Path(f'/proc/{proceso.pid}/stat').read_text().split()[21]
    estado['verificador'] = datos
    guardar(copia.estado, estado)
    for _ in range(80):
        if proceso.poll() is not None:
            raise ValueError('El verificador terminó; consulte verificador.log en su copia')
        try:
            salud_verificador(copia, datos['puerto'])
            return True
        except (OSError, ValueError):
            time.sleep(.25)
    raise ValueError('El verificador no respondió en 20 segundos')


def parar_verificador(copia):
    """Para solo el verificador propio: mismo ejecutable y mismo instante de arranque."""
    estado = copia.leer()
    if not _proceso_verificador(copia, estado):
        return False
    pid = estado['verificador']['pid']
    os.kill(pid, 15)
    for _ in range(100):
        if not _proceso_verificador(copia, estado):
            return True
        time.sleep(.1)
    raise ValueError('El verificador sigue activo; se conserva su copia')
