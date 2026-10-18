"""Publica el conjunto interno coherente del material histórico de RRHH."""

import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import stat
import subprocess
import tempfile


OBLIGATORIOS = (
    'ca/ca.crt', 'ca/ca.key', 'ca/serie',
    'tls/servidor.crt', 'tls/servidor.key',
    'mtls/cliente.crt', 'mtls/cliente.key', 'mtls/cliente.p12',
    'mtls/cliente.p12.password', 'mtls/intervencion.crt',
    'mtls/intervencion.key', 'mtls/intervencion.p12',
    'mtls/intervencion.p12.password', 'kms/clave-maestra.bin',
    'kms/atestacion-ed25519.key', 'kms/atestacion-ed25519.pub',
    'kms/revalidacion-ed25519.key', 'kms/revalidacion-ed25519.pub',
    'tsa/clave-hmac.bin', 'idempotencia/configuracion.json',
    'idempotencia/g2-localizador.bin', 'idempotencia/g2-huella-solicitud.bin',
    'idempotencia/g1-localizador.bin', 'idempotencia/g1-huella-solicitud.bin',
    'identidad/identidad.json', 'identidad/intervencion.json',
    'identidad/bolsa-bback.json',
    'manifiesto.json', 'desarrollo.env',
)

# No se trasladan scripts, respaldos, DSN ni descriptores de módulos. Cada
# productor prepara las conexiones opcionales para la instancia nueva.
OPCIONALES = (
    'identidad/centros.json', 'identidad/consultas-rrhh.json',
    'identidad/solicitante.json', 'identidad/ratificador.json',
    *(f'mtls/{nombre}.{extension}' for nombre in
      ('centros', 'consultas-rrhh', 'solicitante', 'ratificador') for extension in
      ('crt', 'key', 'p12', 'p12.password')),
    'catalogos/organizacion-publica.json', 'catalogos/rpt-publica.json',
)


def _openssl(*args, dato=None):
    try:
        return subprocess.check_output(['openssl', *map(str, args)], input=dato,
                                       stderr=subprocess.DEVNULL, timeout=15)
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired):
        raise ValueError('Material original incoherente o certificado no vigente') from None


def _huella_certificado(ruta):
    return hashlib.sha256(_openssl('x509', '-in', ruta, '-outform', 'DER')).hexdigest()


def _huella_publica(ruta, certificado=False):
    pem = _openssl('x509', '-in', ruta, '-pubkey', '-noout') if certificado else \
          _openssl('pkey', '-in', ruta, '-pubout')
    return hashlib.sha256(_openssl('pkey', '-pubin', '-outform', 'DER', dato=pem)).hexdigest()


def _huella_clave_pem(pem):
    publica = _openssl('pkey', '-pubout', dato=pem)
    return hashlib.sha256(_openssl('pkey', '-pubin', '-outform', 'DER', dato=publica)).hexdigest()


def _certificar(raiz):
    ca = raiz / 'ca/ca.crt'
    manifiesto = json.loads((raiz / 'manifiesto.json').read_text())
    if not isinstance(manifiesto, dict) or manifiesto.get('perfil') != 'desarrollo' or \
            manifiesto.get('autoridad') != 'no_autoritativo' or \
            manifiesto.get('migrable_a_produccion') is not False:
        raise ValueError('El manifiesto no corresponde a material de desarrollo')
    def campo_huella(campo):
        valor = manifiesto.get(campo)
        if not isinstance(valor, str) or not re.fullmatch('[0-9a-fA-F]{64}', valor):
            raise ValueError('El manifiesto contiene una huella inválida')
        return valor.lower()
    comprobaciones = (
        ('huella_ca_sha256', ca),
        ('huella_cliente_sha256', raiz / 'mtls/cliente.crt'),
        ('huella_intervencion_sha256', raiz / 'mtls/intervencion.crt'),
    )
    for campo, ruta in comprobaciones:
        if campo_huella(campo) != _huella_certificado(ruta):
            raise ValueError('Las huellas originales no corresponden al material')
    for nombre in ('cliente', 'intervencion'):
        cert = raiz / f'mtls/{nombre}.crt'
        key = raiz / f'mtls/{nombre}.key'
        _openssl('verify', '-purpose', 'sslclient', '-CAfile', ca, cert)
        if _huella_publica(cert, True) != _huella_publica(key):
            raise ValueError('Certificado y clave originales no coinciden')
        identidad = raiz / ('identidad/identidad.json' if nombre == 'cliente'
                           else 'identidad/intervencion.json')
        identidad_datos = json.loads(identidad.read_text())
        if (not isinstance(identidad_datos, dict) or
                not isinstance(identidad_datos.get('certificate_sha256'), str) or
                identidad_datos['certificate_sha256'].lower() != _huella_certificado(cert)):
            raise ValueError('La identidad original no corresponde al certificado')
        p12 = raiz / f'mtls/{nombre}.p12'
        password = raiz / f'mtls/{nombre}.p12.password'
        _openssl('pkcs12', '-in', p12, '-passin', f'file:{password}', '-noout')
        certificado_p12 = _openssl('pkcs12', '-in', p12, '-passin', f'file:{password}',
                                  '-clcerts', '-nokeys')
        if hashlib.sha256(_openssl('x509', '-outform', 'DER', dato=certificado_p12)).hexdigest() != _huella_certificado(cert):
            raise ValueError('El paquete PKCS#12 no contiene la identidad original')
        clave_p12 = _openssl('pkcs12', '-in', p12, '-passin', f'file:{password}',
                            '-nocerts', '-nodes')
        if _huella_clave_pem(clave_p12) != _huella_publica(key):
            raise ValueError('El paquete PKCS#12 no contiene la clave original')
        ca_p12 = _openssl('pkcs12', '-in', p12, '-passin', f'file:{password}',
                          '-cacerts', '-nokeys')
        if hashlib.sha256(_openssl('x509', '-outform', 'DER', dato=ca_p12)).hexdigest() != _huella_certificado(ca):
            raise ValueError('El paquete PKCS#12 no contiene la CA original')
    # El servidor de origen puede haber caducado; se reemplaza antes del arranque.
    _openssl('verify', '-no_check_time', '-purpose', 'sslserver', '-CAfile', ca,
             raiz / 'tls/servidor.crt')
    if _huella_publica(ca, True) != _huella_publica(raiz / 'ca/ca.key'):
        raise ValueError('La clave de la CA original no corresponde')
    if campo_huella('huella_servidor_sha256') != _huella_certificado(raiz / 'tls/servidor.crt'):
        raise ValueError('La huella original del servidor no corresponde')
    for nombre, campo in (
        ('atestacion', 'huella_publica_atestacion_kms_sha256'),
        ('revalidacion', 'huella_publica_revalidacion_kms_sha256'),
    ):
        privada = raiz / f'kms/{nombre}-ed25519.key'
        publica = raiz / f'kms/{nombre}-ed25519.pub'
        huella_publica = hashlib.sha256(_openssl('pkey', '-pubin', '-in', publica,
                                                 '-outform', 'DER')).hexdigest()
        if _huella_publica(privada) != huella_publica or campo_huella(campo) != huella_publica:
            raise ValueError('El material KMS original no corresponde al manifiesto')
    for nombre in ('kms/clave-maestra.bin', 'tsa/clave-hmac.bin',
                   'idempotencia/g1-localizador.bin', 'idempotencia/g1-huella-solicitud.bin',
                   'idempotencia/g2-localizador.bin', 'idempotencia/g2-huella-solicitud.bin'):
        if (raiz / nombre).stat().st_size != 32:
            raise ValueError('Un secreto original tiene una longitud inválida')
    identidad_rrhh = json.loads((raiz / 'identidad/identidad.json').read_text())
    bback = json.loads((raiz / 'identidad/bolsa-bback.json').read_text())
    if (not isinstance(bback, dict) or not isinstance(identidad_rrhh, dict) or
            bback.get('certificado_sha256') != identidad_rrhh.get('certificate_sha256') or
            bback.get('sujeto') != identidad_rrhh.get('subject')):
        raise ValueError('B-BACK no corresponde al certificado y sujeto originales')
    return manifiesto


def validar(origen, repo, base):
    """Valida antes de crear la instancia y devuelve las rutas y huellas congeladas."""
    origen = Path(origen)
    if not origen.is_absolute() or origen.resolve(strict=True) != origen or \
            origen == repo or origen.is_relative_to(repo) or origen.is_relative_to(base):
        raise ValueError('El material original debe estar fuera de Git y de las copias')
    rutas, directorios = {}, {Path('.')}
    for nombre in (*OBLIGATORIOS, *OPCIONALES):
        relativo = Path(nombre)
        ruta = origen / relativo
        try:
            info = ruta.lstat()
        except FileNotFoundError:
            if nombre in OBLIGATORIOS:
                raise ValueError('Falta una pieza del conjunto interno original') from None
            continue
        for padre in (origen, *(origen / p for p in reversed(relativo.parents) if p != Path('.'))):
            dato = padre.lstat()
            if (not stat.S_ISDIR(dato.st_mode) or dato.st_uid != os.getuid() or
                    dato.st_mode & 0o077):
                raise ValueError('Un directorio del material original está enlazado o no es privado')
            directorios.add(padre.relative_to(origen))
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or \
                info.st_mode & 0o077 or info.st_nlink != 1 or info.st_size > 16 * 1024 * 1024:
            raise ValueError('Una pieza seleccionada del material está enlazada o no es privada')
        # desarrollo.env se deriva para la instancia: ni se lee ni se copia.
        if nombre != 'desarrollo.env':
            contenido = ruta.read_bytes()
            if (b'DATABASE_URL' in contenido or
                    re.search(rb'postgres(?:ql)?://', contenido, re.I)):
                raise ValueError('Una pieza seleccionada contiene conexiones ajenas')
            rutas[relativo] = hashlib.sha256(contenido).hexdigest()
    _certificar(origen)
    return rutas, directorios


def preparar(origen, destino, repo, base):
    """Copia el conjunto completo y publica material coherente de forma atómica."""
    origen = Path(origen)
    rutas, directorios = validar(origen, repo, base)
    if destino.exists() or destino.is_symlink():
        raise ValueError('El material de la copia ya existe')
    temporal = Path(tempfile.mkdtemp(prefix='.material-rrhh-', dir=destino.parent))
    try:
        for relativo in directorios:
            (temporal / relativo).mkdir(parents=True, exist_ok=True, mode=0o700)
        for relativo, huella in rutas.items():
            objetivo = temporal / relativo
            objetivo.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
            with (origen / relativo).open('rb') as entrada, objetivo.open('xb') as salida:
                shutil.copyfileobj(entrada, salida)
            objetivo.chmod(0o600)
            if hashlib.sha256(objetivo.read_bytes()).hexdigest() != huella:
                raise ValueError('El material original cambió durante la copia')
        manifiesto = _certificar(temporal)
        subprocess.run([repo / 'scripts/generar_credenciales_desarrollo.sh',
                        '--renovar-servidor', temporal], check=True,
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30,
                       env={**os.environ, 'VEC_DESARROLLO_VIGENCIA_DIAS': '90'})
        _openssl('verify', '-purpose', 'sslserver', '-CAfile', temporal / 'ca/ca.crt',
                 temporal / 'tls/servidor.crt')
        manifiesto['huella_servidor_sha256'] = _huella_certificado(temporal / 'tls/servidor.crt').upper()
        (temporal / 'manifiesto.json').write_text(json.dumps(manifiesto, ensure_ascii=False,
                                                     separators=(',', ':')) + '\n')
        # El entorno que traía la fuente puede contener rutas o conexiones de otra copia.
        (temporal / 'desarrollo.env').write_text(
            'VEC_EXECUTION_PROFILE=desarrollo\nVEC_AUTH_MODE=desarrollo\n'
            'VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO\n'
            f'VEC_DEVELOPMENT_MATERIAL_DIR={shlex.quote(str(destino))}\n'
            f'VEC_TLS_CERT_FILE={shlex.quote(str(destino / "tls/servidor.crt"))}\n'
            f'VEC_TLS_KEY_FILE={shlex.quote(str(destino / "tls/servidor.key"))}\n')
        _certificar(temporal)
        os.rename(temporal, destino)
    finally:
        if temporal.exists():
            shutil.rmtree(temporal)
