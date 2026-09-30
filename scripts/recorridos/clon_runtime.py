#!/usr/bin/env python3
"""Compile and run a pinned main revision using private local clone material."""

import argparse
import fcntl
import hashlib
import ipaddress
import importlib.util
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import signal
import socket
import ssl
import stat
import struct
import subprocess
import sys
import tarfile
import tempfile
import time
from urllib.parse import parse_qsl, urlsplit


class RuntimeErrorLocal(Exception):
    pass


def fail(message):
    raise RuntimeErrorLocal(message)


def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def write_json(path, value):
    descriptor, temporary_name = tempfile.mkstemp(prefix='.' + path.name + '-', suffix='.tmp', dir=path.parent)
    temporary = Path(temporary_name)
    try:
        with os.fdopen(descriptor, 'w') as stream:
            json.dump(value, stream, indent=2)
            stream.write('\n')
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def confined(path, root, directory=False):
    candidate = Path(path)
    if not candidate.is_absolute():
        candidate = root / candidate
    if '..' in candidate.parts:
        fail('Ruta privada con componentes no admitidos.')
    try:
        candidate.relative_to(root)
    except ValueError:
        fail('El material debe estar dentro del directorio privado del clon.')
    for item in [candidate, *candidate.parents]:
        if item == root.parent:
            break
        if item.is_symlink():
            fail('El material no admite enlaces simbólicos.')
    if directory:
        valid = candidate.is_dir()
    else:
        valid = candidate.is_file() and candidate.stat().st_nlink == 1
    if not valid:
        fail('Falta un archivo o directorio privado requerido.')
    return candidate


def writable_data_path(value, root, name):
    """Allow only nominal future data files; never create application content."""
    nominal = {
        'VEC_PERSONAL_CATALOG_PATH': ('personal-catalog.json', False),
        'VEC_BOLSA_DATA_DIR': ('bolsa', True),
        'VEC_BOLSA_DATA_PATH': ('bolsa/bolsa_store.json', False),
        'VEC_BOLSA_IMPORTACION_CONVOCA_CUSTODIA_DIR': ('importaciones', True),
    }
    relative, directory = nominal[name]
    base = confined(root / 'rw/data', root, directory=True)
    path = Path(value)
    if path != base / relative:
        fail('El archivo mutable no corresponde a su ruta nominal propia.')
    for item in [base, *reversed(path.relative_to(base).parents), path]:
        if not item.is_absolute():
            item = base / item
        if item.is_symlink():
            fail('La escritura propia no admite enlaces simbólicos.')
        if item.exists():
            metadata = item.stat()
            forbidden = 0o077 if item == base else 0o022
            if metadata.st_uid != os.getuid() or stat.S_IMODE(metadata.st_mode) & forbidden:
                fail('La escritura debe permanecer privada del usuario del clon.')
            if item != path or directory:
                if not item.is_dir():
                    fail('Un antecesor mutable no es un directorio propio.')
            elif not item.is_file() or metadata.st_nlink != 1:
                fail('El archivo mutable no es un archivo regular propio.')
    return path


def validate_state(repo, state):
    absolute = Path(os.path.abspath(state))
    for part in [absolute, *absolute.parents]:
        if part.is_symlink():
            fail('El directorio privado no admite enlaces simbólicos.')
        if (part / '.git').exists():
            fail('El directorio privado debe quedar fuera de cualquier repositorio Git.')
    if not absolute.exists():
        absolute.mkdir(parents=True, mode=0o700)
    metadata = absolute.stat()
    if metadata.st_uid != os.getuid() or stat.S_IMODE(metadata.st_mode) & 0o077:
        fail('El directorio privado debe pertenecer al usuario y tener permisos 0700.')
    if absolute == repo or repo in absolute.parents:
        fail('El directorio privado debe quedar fuera del repositorio.')
    return absolute


def git(repo, *args):
    result = subprocess.run(['git', '-C', str(repo), *args], capture_output=True, check=False)
    if result.returncode:
        fail('No se pudo comprobar la revisión fuente en Git.')
    return result.stdout.decode().strip()


def pinned_main(repo, commit):
    if not re.fullmatch(r'[0-9a-f]{40}', commit):
        fail('Indique el hash completo de una revisión de main.')
    if git(repo, 'rev-parse', commit + '^{commit}') != commit:
        fail('La revisión fuente no coincide con el hash solicitado.')
    for ref in ('refs/heads/main', 'refs/remotes/origin/main'):
        result = subprocess.run(['git', '-C', str(repo), 'merge-base', '--is-ancestor', commit, ref],
                                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if result.returncode == 0:
            return commit
    fail('La revisión fuente no pertenece a main.')


def read_environment(state):
    path = state / 'runtime-config.json'
    if path.exists():
        path = confined(path, state)
        try:
            values = json.loads(path.read_text())
        except (ValueError, UnicodeError):
            fail('Configuración JSON inválida.')
        if not isinstance(values, dict):
            fail('La configuración JSON debe ser un objeto de variables.')
    else:
        path = confined(state / 'runtime.env', state)
        values = {}
        for line in path.read_text().splitlines():
            if not line.strip() or line.lstrip().startswith('#'):
                continue
            if '=' not in line:
                fail('Formato runtime.env inválido.')
            name, value = line.split('=', 1)
            if name in values or not re.fullmatch(r'VEC_[A-Z0-9_]+', name):
                fail('Variable repetida o nombre inválido en runtime.env.')
            try:
                parsed = shlex.split(value, comments=False)
            except ValueError:
                fail('Valor inválido en runtime.env.')
            if len(parsed) != 1 or '$' in value or '`' in value:
                fail('runtime.env admite valores literales, sin expansión de shell.')
            values[name] = parsed[0]
    if path.stat().st_uid != os.getuid() or stat.S_IMODE(path.stat().st_mode) & 0o077:
        fail('La configuración privada requiere permisos 0600.')
    if not all(isinstance(k, str) and isinstance(v, str) and '\0' not in v and '\n' not in v
               for k, v in values.items()):
        fail('La configuración contiene valores no admitidos.')
    return values, digest(path)


def validate_dsn(value, pg_port, state):
    try:
        parsed = urlsplit(value)
        parameters = parse_qsl(parsed.query, keep_blank_values=True, strict_parsing=True)
        query = dict(parameters)
        if (parsed.scheme not in ('postgres', 'postgresql') or parsed.hostname != '127.0.0.1'
                or parsed.port != pg_port or not parsed.username or not parsed.path.strip('/')
                or parsed.fragment or len(query) != len(parameters)):
            fail('Todos los DSN deben apuntar exclusivamente al puerto PostgreSQL del clon.')
        allowed = {'sslmode', 'sslrootcert', 'sslcert', 'sslkey', 'connect_timeout', 'application_name'}
        if set(query) - allowed or query.get('sslmode') not in ('verify-full',):
            fail('Parámetros PostgreSQL no admitidos; se exige TLS verify-full al clon.')
        for name in ('sslrootcert', 'sslcert', 'sslkey'):
            if name in query:
                confined(query[name], state)
    except (ValueError, UnicodeError):
        fail('DSN PostgreSQL inválido.')


def validate_material(state, commit, port, pg_port):
    path = confined(state / 'material-manifest.json', state)
    manifest = json.loads(path.read_text())
    target = manifest.get('target', {})
    if (manifest.get('owner') != 'Codex-M' or target.get('source_commit') != commit
            or target.get('app_port') != port or target.get('pg_port') != pg_port):
        fail('El material pertenece a otra revisión o a otro destino del clon.')
    files = manifest.get('files', {})
    if not isinstance(files, dict) or not files:
        fail('El material no tiene inventario verificable.')
    for relative, expected in files.items():
        if not isinstance(relative, str) or Path(relative).is_absolute():
            fail('El inventario de material contiene una ruta inválida.')
        file = confined(relative, state)
        if digest(file) != expected:
            fail('El material cambió después de preparar su manifiesto.')
    for file in (state / 'material').rglob('*.json'):
        file = confined(file, state)
        data = json.loads(file.read_text())
        pending = [data]
        while pending:
            value = pending.pop()
            if isinstance(value, dict):
                pending.extend(value.values())
            elif isinstance(value, list):
                pending.extend(value)
            elif isinstance(value, str) and value.startswith(('postgres:', 'postgresql:')):
                validate_dsn(value, pg_port, state)
    return digest(path)


def inspect_smtp_resource(kind, name):
    result = subprocess.run(['/usr/bin/docker', kind, 'inspect', name], capture_output=True,
                            timeout=10, env={'PATH': '/usr/bin:/bin', 'LANG': 'C'})
    if result.returncode:
        fail('El recurso SMTP privado no está disponible.')
    data = json.loads(result.stdout)
    if not isinstance(data, list) or len(data) != 1 or not isinstance(data[0], dict):
        fail('El inventario SMTP privado es inválido.')
    return data[0]


def smtp_proxy_identity(record, state, address, port):
    data = json.loads(confined(record, state).read_text())
    command = ['/usr/bin/socat', '-T', '15',
               'TCP4-LISTEN:' + str(port) + ',bind=127.0.0.1,reuseaddr,fork,max-children=8',
               'TCP4:' + address + ':1025,connect-timeout=5']
    if (data.get('owner') != 'Codex-M' or data.get('state') != str(state)
            or data.get('command') != command or type(data.get('pid')) is not int or data['pid'] < 2):
        fail('La reserva del proxy SMTP pertenece a otro destino.')
    identity = process_identity(data['pid'])
    if (identity is None or identity['uid'] != os.getuid()
            or identity['exe'] != str(Path(command[0]).resolve())
            or identity['start_ticks'] != data.get('start_ticks')):
        fail('El proceso SMTP no coincide con la reserva privada.')
    actual = (Path('/proc') / str(data['pid']) / 'cmdline').read_bytes().rstrip(b'\0').split(b'\0')
    if actual != [item.encode() for item in command]:
        fail('El proceso SMTP cambió de destino.')
    return data['pid']


def validate_smtp(values, state, material_root=None):
    fields = {'VEC_SMTP_HOST', 'VEC_SMTP_PORT', 'VEC_SMTP_FROM', 'VEC_SMTP_CA_FILE', 'VEC_SMTP_MODO_TLS'}
    if not any(values.get(field) for field in fields):
        return
    if (any(not values.get(field) for field in fields) or values['VEC_SMTP_HOST'] != '127.0.0.1'
            or values['VEC_SMTP_FROM'] != 'rrhh@example.test'
            or values['VEC_SMTP_MODO_TLS'] != 'starttls'):
        fail('SMTP requiere el buzón sintético local y STARTTLS.')
    ca = confined(values['VEC_SMTP_CA_FILE'], material_root or state)
    if ca.stat().st_uid != os.getuid() or stat.S_IMODE(ca.stat().st_mode) & 0o077:
        fail('La CA SMTP debe ser privada del clon.')
    profile_file = confined(state / 'perfiles.json', state)
    if profile_file.stat().st_uid != os.getuid() or stat.S_IMODE(profile_file.stat().st_mode) & 0o077:
        fail('El recibo SMTP debe ser privado del clon.')
    profiles = json.loads(profile_file.read_text())
    proof = profiles.get('profiles', {}).get('usuarios_comunicaciones', {})
    clone_path = confined(state / 'clon.json', state)
    target_path = confined(state / 'comunicaciones/target.json', state)
    for path in (clone_path, target_path):
        if path.stat().st_uid != os.getuid() or stat.S_IMODE(path.stat().st_mode) & 0o077:
            fail('La reserva SMTP debe ser privada del clon.')
    clone = json.loads(clone_path.read_text())
    # El preparador y el lanzador validan la misma reserva, incluidos sus nombres.
    import importlib.util
    spec = importlib.util.spec_from_file_location('recorridos_comunicaciones', Path(__file__).with_name('clon_comunicaciones.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    try:
        target = module._target(state, clone.get('contenedor'), clone.get('puerto_pg'))
    except (module.PreparationError, OSError, ValueError, TypeError):
        fail('La reserva SMTP no corresponde al clon propio.')
    if (clone.get('propietario') != 'Codex-M' or clone.get('estado') != str(state)
            or proof.get('source_commit') != clone.get('commit')
            or proof.get('target_sha256') != digest(target_path)
            or any(proof.get(key) != value for key, value in target.items())
            or values['VEC_SMTP_PORT'] != str(target['smtp_port'])):
        fail('El recibo SMTP no coincide con su reserva privada.')
    if (proof.get('smtp_ready') is not True or proof.get('smtp_scope') != 'synthetic_local_sink'
            or proof.get('corporate_delivery') is not False or proof.get('starttls_verified') is not True
            or proof.get('external_recipient_rejected') is not True
            or proof.get('mailpit_container') != target['container']):
        fail('Falta el recibo del SMTP sintético verificado por el preparador.')
    approved_ca = confined(state / 'material/ca/ca.crt', state)
    if ca != (material_root or state) / 'material/ca/ca.crt' or digest(ca) != digest(approved_ca):
        fail('La CA SMTP no coincide con el material acreditado del clon.')
    sink = inspect_smtp_resource('container', proof['mailpit_container'])
    network_name = target['network']
    network = inspect_smtp_resource('network', network_name)
    image = inspect_smtp_resource('image', 'axllent/mailpit:v1.27.8')
    labels = {'vec.recorridos.owner': 'Codex-M', 'vec.recorridos.state': str(state)}
    if (any(sink.get('Config', {}).get('Labels', {}).get(k) != v for k, v in labels.items())
            or any(network.get('Labels', {}).get(k) != v for k, v in labels.items())
            or network.get('Internal') is not True or sink.get('State', {}).get('Running') is not True
            or not image.get('Id') or sink.get('Image') != image.get('Id')
            or proof.get('image_id') != image.get('Id')):
        fail('El buzón SMTP no pertenece al clon privado aislado.')
    host = sink.get('HostConfig', {})
    expected_ports = {'1025/tcp': [{'HostIp': '127.0.0.1', 'HostPort': str(target['smtp_port'])}],
                      '8025/tcp': [{'HostIp': '127.0.0.1', 'HostPort': str(target['http_port'])}]}
    command = sink.get('Config', {}).get('Cmd') or []
    recipients = r'^[A-Za-z0-9._+\-]+@example\.test$'
    expected_command = ['--disable-version-check', '--block-remote-css-and-fonts',
                        '--smtp-disable-rdns', '--smtp-require-starttls', '--smtp-tls-cert', '/tls/servidor.crt',
                        '--smtp-tls-key', '/tls/servidor.key', '--smtp-allowed-recipients', recipients,
                        '--smtp', '0.0.0.0:1025', '--listen', '0.0.0.0:8025', '--database', '/data/mailpit.db',
                        '--max', '100', '--quiet']
    if (host.get('PortBindings') != expected_ports or host.get('NetworkMode') != network_name
            or host.get('ReadonlyRootfs') is not True or command != expected_command):
        fail('El SMTP privado perdió sus límites de red o de TLS.')
    position = command.index('--smtp-allowed-recipients')
    if position + 1 >= len(command) or command[position + 1] != recipients:
        fail('El buzón SMTP admite destinos ajenos al ejercicio sintético.')
    mounts = {entry.get('Destination'): entry for entry in sink.get('Mounts', [])}
    if (mounts.get('/tls', {}).get('Source') != str(state / 'material/comunicaciones')
            or mounts.get('/tls', {}).get('RW') is not False):
        fail('El servidor SMTP no usa el certificado privado preparado.')
    networks = sink.get('NetworkSettings', {}).get('Networks', {})
    if set(networks) != {network_name}:
        fail('El SMTP privado está conectado a otra red.')
    address = networks[network_name].get('IPAddress', '')
    try:
        ip = ipaddress.IPv4Address(address)
        if not ip.is_private or ip.is_loopback or ip.is_unspecified or ip.is_multicast:
            fail('El proxy SMTP tiene un destino de red no admitido.')
    except ValueError:
        fail('El destino del buzón SMTP es inválido.')
    proxies = proof.get('loopback_proxies') or []
    matching = [item for item in proxies if item.get('port') == target['smtp_port']]
    if len(matching) != 1:
        fail('Falta la reserva del proxy SMTP propio.')
    if matching[0].get('record') != str(state / 'comunicaciones' / ('proxy-' + str(target['smtp_port']) + '.json')):
        fail('La reserva del proxy SMTP tiene otra ruta.')
    pid = smtp_proxy_identity(matching[0].get('record', ''), state, address, target['smtp_port'])
    if matching[0].get('pid') != pid:
        fail('El recibo SMTP no corresponde al proceso reservado.')


def runtime_environment(source, state, port, pg_port, projection_root=None):
    files_root = projection_root or state
    values, config_sha = read_environment(files_root)
    if projection_root is not None and values.get("VEC_PORTAL_PROCESO") != "interno":
        fail("El runtime aislado exige el portal interno explícito.")
    declared = set()
    for path in (source / 'config').glob('*.go'):
        if not path.name.endswith('_test.go'):
            declared.update(re.findall(r'"(VEC_[A-Z0-9_]+)"', path.read_text()))
    # Bootstrap owns some feature selectors. Accept only literal names actually
    # read by its environment accessors in the pinned source, including constants.
    bootstrap = '\n'.join(p.read_text() for p in (source / 'internal/app/bootstrap').glob('*.go')
                          if not p.name.endswith('_test.go'))
    constants = dict(re.findall(r'\b([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"(VEC_[A-Z0-9_]+)"', bootstrap))
    reads = re.findall(r'\b(?:os\.Getenv|getenv|envFirst)\(\s*(?:"(VEC_[A-Z0-9_]+)"|([A-Za-z_][A-Za-z0-9_]*))', bootstrap)
    # This selector reads os.Getenv(nombre); its nominal second argument is a name.
    if re.search(r'func selectorCapacidadRRHHDesarrollo\(cfg config\.Config, nombre string\)', bootstrap) and 'os.Getenv(nombre)' in bootstrap:
        reads.extend(re.findall(r'\bselectorCapacidadRRHHDesarrollo\(\s*[^,]+,\s*(?:"(VEC_[A-Z0-9_]+)"|([A-Za-z_][A-Za-z0-9_]*))', bootstrap))
    for literal, constant in reads:
        if literal:
            declared.add(literal)
        elif constant in constants:
            declared.add(constants[constant])
    if set(values) - declared:
        fail('La configuración contiene variables que no declara esta revisión de VEC.')
    if values.get('VEC_EXECUTION_PROFILE') != 'desarrollo' or values.get('VEC_AUTH_MODE') != 'desarrollo':
        fail('Este lanzador exige el perfil sintético de desarrollo con identidad mTLS propia.')
    required = ('VEC_DEVELOPMENT_GUARD', 'VEC_DEVELOPMENT_MATERIAL_DIR', 'VEC_TLS_CERT_FILE', 'VEC_TLS_KEY_FILE',
                'VEC_CT_DATABASE_URL', 'VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL',
                'VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL')
    if any(not values.get(k) for k in required):
        fail('Falta configuración de desarrollo, TLS o PostgreSQL del clon.')
    if values['VEC_DEVELOPMENT_GUARD'] != 'ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO':
        fail('Falta el reconocimiento explícito del material sintético.')
    validate_smtp(values, state, files_root)
    for name, value in values.items():
        if name.endswith('_DATABASE_URL'):
            validate_dsn(value, pg_port, files_root)
        elif name == 'VEC_SMTP_HOST':
            continue  # Validated above against the owned synthetic STARTTLS sink.
        elif name == 'VEC_HTTP_ALLOWED_CIDRS':
            if value != '127.0.0.1/32':
                fail('La aplicación del clon sólo admite el CIDR loopback exacto.')
        elif name.endswith(('_URL', '_HOST', '_CIDRS')) and value:
            fail('Este clon no admite endpoints adicionales ni redes configuradas.')
        elif name.endswith(('_FILE', '_PATH', '_DIR')) and value:
            if projection_root is not None and name in {
                    'VEC_PERSONAL_CATALOG_PATH', 'VEC_BOLSA_DATA_DIR', 'VEC_BOLSA_DATA_PATH',
                    'VEC_BOLSA_IMPORTACION_CONVOCA_CUSTODIA_DIR'}:
                values[name] = str(writable_data_path(value, files_root, name))
            else:
                values[name] = str(confined(value, files_root, directory=name.endswith('_DIR')))
    material = confined(values['VEC_DEVELOPMENT_MATERIAL_DIR'], files_root, directory=True)
    confined(material / 'ca/ca.crt', files_root)
    values['VEC_HTTP_ADDR'] = f'127.0.0.1:{port}'
    values['VEC_HTTP_ALLOWED_CIDRS'] = '127.0.0.1/32'
    if projection_root is not None:
        for name in values:
            if name.startswith('VEC_EXTERNO_'):
                fail('La configuración interna contiene una capacidad externa.')
        environment = {'PATH': '/usr/bin:/bin', 'HOME': '/home/runtime', 'TMPDIR': '/tmp', 'TZ': 'UTC'}
    else:
        values['VEC_PERSONAL_CATALOG_PATH'] = str(state / 'personal-catalog.json')
        environment = {'PATH': '/usr/bin:/bin', 'HOME': str(state), 'TMPDIR': str(state / 'tmp'), 'TZ': 'UTC'}
        (state / 'tmp').mkdir(mode=0o700, exist_ok=True)
    environment.update(values)
    return environment, config_sha


def container_module():
    spec = importlib.util.spec_from_file_location('runtime_container', Path(__file__).with_name('clon_runtime_container.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def projection_module():
    spec = importlib.util.spec_from_file_location('runtime_projection', Path(__file__).with_name('clon_interno_material.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def read_runtime_descriptor(state):
    top = json.loads(confined(state / 'material-manifest.json', state).read_text())
    descriptor = top.get('runtime_interno', {})
    if descriptor.get('mode') != 'interno':
        fail('Falta la proyección interna positiva del material.')
    root = confined(state / 'runtime-interno', state, directory=True)
    expected = {'material': 'runtime-interno/material', 'config': 'runtime-interno/runtime-config.json',
                'manifest': 'runtime-interno/material-manifest.json'}
    paths = {}
    for name, relative in expected.items():
        if descriptor.get(name) != relative:
            fail('La proyección interna tiene una ruta de otra superficie.')
        paths[name] = confined(state / relative, state, directory=name == 'material')
    sealed = json.loads(paths['manifest'].read_text())
    if (sealed.get('owner') != 'Codex-M' or sealed.get('portal') != 'interno'
            or sealed.get('mode') != 'interno' or sealed.get('target') != top.get('target')):
        fail('El manifiesto interno no corresponde al origen acreditado.')
    proof = sealed.get('source_proof', {})
    operator = {key: value for key, value in top.items() if key != 'runtime_interno'}
    operator_sha = hashlib.sha256(json.dumps(operator, sort_keys=True, separators=(',', ':')).encode()).hexdigest()
    if (proof.get('operator_manifest_normalization') != 'drop_runtime_interno_only'
            or proof.get('operator_manifest_sha256') != operator_sha
            or proof.get('operator_env_sha256') != digest(confined(state / 'runtime-config.json', state))
            or descriptor.get('manifest_sha256') != digest(paths['manifest'])
            or descriptor.get('source_commit') != top.get('target', {}).get('source_commit')):
        fail('La proyección perdió la relación con su material y configuración de origen.')
    if not re.fullmatch(r'[0-9a-f]{40}', descriptor.get('source_commit', '')):
        fail('La proyección requiere una revisión fuente completa.')
    allowed = sealed.get('files', {})
    positive = proof.get('positive_files', {})
    material_names = {
        'ca/ca.crt', 'tls/servidor.crt', 'tls/servidor.key', 'mtls/cliente.crt', 'mtls/intervencion.crt',
        'identidad/identidad.json', 'identidad/intervencion.json', 'manifiesto.json',
        'kms/clave-maestra.bin', 'kms/atestacion-ed25519.key', 'kms/atestacion-ed25519.pub',
        'kms/revalidacion-ed25519.key', 'kms/revalidacion-ed25519.pub', 'tsa/clave-hmac.bin',
        'idempotencia/configuracion.json', 'idempotencia/g1-localizador.bin', 'idempotencia/g1-huella-solicitud.bin',
        'idempotencia/g2-localizador.bin', 'idempotencia/g2-huella-solicitud.bin', 'pg/ca.crt',
        'identidad/usuarios-preferencias-interna.json', 'mtls/solicitante.crt', 'mtls/ratificador.crt',
        'identidad/solicitante.json', 'identidad/ratificador.json', 'identidad/centros.json',
        'identidad/consultas-rrhh.json', 'identidad/bolsa-bback.json', 'identidad/documentos.json',
        'catalogos/organizacion-publica.json', 'catalogos/rpt-publica.json',
    }
    generated = {'runtime-config.json', 'runtime.env', 'material/portal-proceso.json', 'material/desarrollo.env'}
    if (not positive or set(positive) - material_names
            or set(allowed) != generated | {'material/' + name for name in positive}):
        fail('La proyección contiene material fuera de su lista positiva interna.')
    for relative, evidence in positive.items():
        original = confined(state / 'material' / relative, state)
        projected = confined(paths['material'] / relative, root)
        if (top.get('files', {}).get('material/' + relative) != digest(original)
                or evidence.get('source_sha256') != digest(original)
                or evidence.get('projected_sha256') != digest(projected)
                or evidence.get('unchanged') is not (digest(original) == digest(projected))):
            fail('El material interno no acredita su copia del origen sellado.')
    contracts = proof.get('contracts', {})
    projection = projection_module()
    approved_sets = projection.APPROVED_SOURCE_CONTRACT_SETS
    source = confined(state / ('source-' + descriptor['source_commit']), state, directory=True)
    source_contracts = {path: digest(confined(source / path, source)) for path in approved_sets[0]}
    internal_contracts = {path: source_contracts[path] for path in projection.SOURCE_PATHS}
    if source_contracts not in approved_sets or contracts != internal_contracts:
        fail('La proyección no corresponde a los contratos de la fuente fijada.')
    public_names = {'catalogos/organizacion-publica.json', 'catalogos/rpt-publica.json'}
    present_public = set(positive) & public_names
    public_selectors = {'VEC_PERSONAL_ORGANIZACION_SOURCE_PATH': 'catalogos/organizacion-publica.json',
                        'VEC_RPT_CATALOGO_PATH': 'catalogos/rpt-publica.json'}
    projected_config = json.loads(paths['config'].read_text())
    operator_config_path = confined(state / 'runtime-config.json', state)
    operator_config = json.loads(operator_config_path.read_text())
    original_public = set(top.get('files', {})) & {'material/' + name for name in public_names}
    original_declares = (original_public or 'public_catalogs' in top
                         or any(key in operator_config for key in public_selectors))
    if original_declares or present_public or any(key in projected_config for key in public_selectors):
        approved_public = getattr(projection, 'APPROVED_PUBLIC_SOURCES', {})
        if (present_public != public_names or set(approved_public) != public_names
                or original_public != {'material/' + name for name in public_names}
                or top.get('files', {}).get('runtime-config.json') != digest(operator_config_path)
                or any(operator_config.get(key) != str(state / 'material' / name) for key, name in public_selectors.items())
                or any(projected_config.get(key) != str(paths['material'] / name) for key, name in public_selectors.items())):
            fail('La proyección necesita las dos fuentes públicas aprobadas completas.')
        if 'public_catalogs' in top:
            declaration = top['public_catalogs']
            if (not isinstance(declaration, dict) or declaration.get('source_commit') != descriptor['source_commit']
                    or declaration.get('files') != approved_public or declaration.get('sql_authority_changed') is not False):
                fail('La declaración original de las fuentes públicas es incoherente.')
        for name, evidence in approved_public.items():
            expected_sha = evidence['sha256']
            if (digest(confined(source / evidence['source_path'], source)) != expected_sha
                    or top['files'].get('material/' + name) != expected_sha
                    or digest(confined(state / 'material' / name, state)) != expected_sha
                    or positive[name].get('source_sha256') != expected_sha
                    or positive[name].get('projected_sha256') != expected_sha
                    or positive[name].get('unchanged') is not True):
                fail('Una fuente pública de la proyección difiere de su copia Git aprobada.')
    actual = set()
    for path in root.rglob('*'):
        if (root / 'rw') in path.parents or path == root / 'rw':
            continue
        if path.is_symlink():
            fail('La proyección interna no admite enlaces.')
        if path.is_file() and path != paths['manifest']:
            relative = str(path.relative_to(root))
            actual.add(relative)
            if allowed.get(relative) != digest(confined(path, root)):
                fail('La proyección expone un archivo fuera de su inventario acreditado.')
    if actual != set(allowed):
        fail('El inventario de la proyección interna cambió.')
    rw = []
    for item in descriptor.get('rw', []):
        if not isinstance(item, dict) or not isinstance(item.get('source'), str) or Path(item['source']).is_absolute():
            fail('La reserva de escritura interna es inválida.')
        source = confined(state / item['source'], root, directory=True)
        if source != root / 'rw' / str(item.get('kind')):
            fail('La escritura interna no corresponde a su directorio propio.')
        target = str(paths['material'] / 'comunicaciones') if item.get('kind') == 'comunicaciones' else str(source)
        if item.get('target') != target:
            fail('La escritura interna sale de su inventario positivo.')
        rw.append({'source': str(source), 'target': target, 'kind': item.get('kind')})
    if {item['kind'] for item in rw} != {'documentos', 'imagenes', 'data', 'comunicaciones'} or len(rw) != 4:
        fail('La proyección necesita sus cuatro directorios de escritura propios.')
    env_path = confined(root / 'runtime.env', root)
    ro_paths = [paths['material'], paths['config'], env_path, paths['manifest']]
    return {'root': str(root), 'mode': 'interno', 'portal': 'interno',
            'material_path': str(paths['material']), 'runtime_config_path': str(paths['config']),
            'manifest_path': str(paths['manifest']),
            'ro': [{'source': str(path), 'target': str(path)} for path in ro_paths], 'rw': rw}


def source_digest(source):
    h = hashlib.sha256()
    for path in sorted(source.rglob('*')):
        if path.is_symlink():
            fail('La revisión extraída contiene un enlace simbólico.')
        if path.is_file():
            h.update(str(path.relative_to(source)).encode() + b'\0' + digest(path).encode() + b'\n')
    return h.hexdigest()


def local_go(source, explicit):
    if explicit:
        go = Path(explicit).resolve()
    else:
        match = re.search(r'^toolchain (go[0-9.]+)$', (source / 'go.mod').read_text(), re.MULTILINE)
        candidates = []
        if match:
            candidates.append(Path.home() / 'go/pkg/mod' / ('golang.org/toolchain@v0.0.1-' + match[1] + '.linux-amd64/bin/go'))
        candidates.append(Path(shutil.which('go') or '/usr/local/go/bin/go'))
        go = next((p.resolve() for p in candidates if p.is_file()), candidates[-1])
    if not go.is_file():
        fail('No se encontró el compilador Go indicado.')
    return go


def elf_interpreter(path):
    """Read ELF program headers; a scratch executable cannot require PT_INTERP."""
    with path.open('rb') as stream:
        header = stream.read(64)
        if len(header) < 64 or header[:4] != b'\x7fELF' or header[4] != 2 or header[5] not in (1, 2):
            fail('El binario no es ELF de 64 bits acreditado.')
        endian = '<' if header[5] == 1 else '>'
        phoff = struct.unpack_from(endian + 'Q', header, 32)[0]
        phsize, count = struct.unpack_from(endian + 'HH', header, 54)
        if phsize < 56 or count > 65535 or phoff + phsize * count > path.stat().st_size:
            fail('Las cabeceras ELF del binario son inválidas.')
        for index in range(count):
            stream.seek(phoff + phsize * index)
            program = stream.read(56)
            kind = struct.unpack_from(endian + 'I', program)[0]
            if kind == 3:
                offset, size = struct.unpack_from(endian + 'Q', program, 8)[0], struct.unpack_from(endian + 'Q', program, 32)[0]
                if size < 1 or size > 4096 or offset + size > path.stat().st_size:
                    fail('El intérprete ELF del binario es inválido.')
                stream.seek(offset)
                return stream.read(size).rstrip(b'\0').decode('ascii')
    return None


def validate_artifact_claim(artifact, commit):
    if (artifact.get('source_commit') != commit or not re.fullmatch(r'[0-9a-f]{64}', artifact.get('sha256', ''))):
        fail('El artefacto aprobado no corresponde a la fuente y huella solicitadas.')
    path = Path(os.path.abspath(artifact['path']))
    if '..' in Path(artifact['path']).parts or any(item.is_symlink() for item in [path, *path.parents]):
        fail('El artefacto aprobado no admite enlaces ni rutas ambiguas.')
    metadata = path.stat()
    if (not stat.S_ISREG(metadata.st_mode) or metadata.st_uid != os.getuid()
            or metadata.st_nlink != 1 or stat.S_IMODE(metadata.st_mode) & 0o022):
        fail('El artefacto aprobado debe ser un archivo regular del operador.')
    if digest(path) != artifact['sha256'] or elf_interpreter(path) is not None:
        fail('La huella o el ELF estático del artefacto aprobado no coinciden.')
    with path.open('rb') as stream:
        header = stream.read(64)
    if struct.unpack_from(('<' if header[5] == 1 else '>') + 'H', header, 18)[0] != 62:
        fail('El artefacto aprobado no corresponde a la arquitectura Linux amd64 del clon.')
    return path


def inspect_artifact(path, source, commit, go):
    result = subprocess.run([str(go), 'version', '-m', '-json', str(path)],
                            env={'PATH': str(go.parent) + ':/usr/bin:/bin', 'HOME': '/nonexistent',
                                 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOSUMDB': 'off'},
                            capture_output=True, timeout=20)
    if result.returncode:
        fail('No se puede comprobar la información Go del artefacto aprobado.')
    info = json.loads(result.stdout)
    settings = {entry['Key']: entry['Value'] for entry in info.get('Settings', [])}
    module = re.search(r'^module\s+(\S+)\s*$', (source / 'go.mod').read_text(), re.MULTILINE)
    if (module is None or info.get('Main', {}).get('Path') != module[1]
            or info.get('Path') != module[1] + '/cmd/vec-server'
            or settings.get('CGO_ENABLED') != '0' or settings.get('GOOS') != 'linux'
            or settings.get('GOARCH') != 'amd64' or settings.get('vcs.modified') == 'true'
            or settings.get('vcs.revision', commit) != commit):
        fail('La información Go del artefacto no acredita el servidor estático y su fuente.')
    return {'go_version': info.get('GoVersion'), 'main_module': module[1], 'package': info['Path'],
            'buildinfo_sha256': hashlib.sha256(result.stdout).hexdigest(),
            'source_binding': 'operator_declared_commit', 'vcs_revision': settings.get('vcs.revision')}


def import_artifact(repo, state, source, binary, snapshot, commit, go, artifact, path, previous):
    if previous is not None and previous.get('binary_sha256') == artifact['sha256'] and previous.get('artifact_direction', {}).get('source_commit') == commit:
        inspect_artifact(binary, source, commit, go)
        return source, binary, previous
    if any((state / name).exists() for name in ('runtime-process.json', 'runtime-container.json', 'runtime-container-intent.json')):
        fail('Detenga y cierre la reserva propia antes de importar otro artefacto.')
    if binary.exists() and previous is None:
        fail('El binario existente carece de un manifiesto válido; no se sustituirá.')
    fd, temporary_name = tempfile.mkstemp(prefix='.vec-server-artifact-', dir=state)
    temporary = Path(temporary_name)
    backup = None
    published = False
    try:
        with os.fdopen(fd, 'wb') as output:
            incoming_fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
            with os.fdopen(incoming_fd, 'rb') as incoming:
                metadata = os.fstat(incoming.fileno())
                if (not stat.S_ISREG(metadata.st_mode) or metadata.st_uid != os.getuid()
                        or metadata.st_nlink != 1 or stat.S_IMODE(metadata.st_mode) & 0o022):
                    fail('El artefacto cambió de propietario o tipo antes de copiarse.')
                shutil.copyfileobj(incoming, output)
            output.flush()
            os.fsync(output.fileno())
            os.fchmod(output.fileno(), 0o755)
        if digest(temporary) != artifact['sha256'] or elf_interpreter(temporary) is not None:
            fail('El artefacto cambió durante la copia privada.')
        information = inspect_artifact(temporary, source, commit, go)
        manifest = {'version': 3, 'runtime_mode': 'interno', 'cgo_enabled': False, 'elf_interpreter': None,
                    'source_commit': commit, 'source_tree': git(repo, 'rev-parse', commit + '^{tree}'),
                    'source_sha256': snapshot['source_sha256'], 'binary_sha256': artifact['sha256'],
                    'build_mode': 'artifact_direction', 'build_command': None,
                    'artifact_direction': dict(information, path=str(path), sha256=artifact['sha256'], source_commit=commit)}
        if previous is not None:
            old_sha = previous['binary_sha256']
            history = state / ('runtime-manifest-' + commit + '-' + old_sha + '.json')
            if history.exists() and json.loads(confined(history, state).read_text()) != previous:
                fail('El manifiesto histórico del binario existente cambió.')
            if not history.exists():
                write_json(history, previous)
            candidate_backup = state / ('vec-server-' + commit + '-' + old_sha)
            if candidate_backup.exists() and digest(confined(candidate_backup, state)) != old_sha:
                fail('La copia histórica del binario existente cambió.')
            os.replace(confined(binary, state), candidate_backup)
            backup = candidate_backup
        os.replace(temporary, binary)
        published = True
        write_json(state / 'runtime-manifest.json', manifest)
        return source, binary, manifest
    except BaseException:
        if published:
            binary.unlink(missing_ok=True)
        if backup is not None and backup.exists():
            os.replace(backup, binary)
        raise
    finally:
        temporary.unlink(missing_ok=True)


def build(repo, state, commit, go_binary, artifact=None):
    manifest_path = state / 'runtime-manifest.json'
    source = state / ('source-' + commit)
    binary = state / ('vec-server-' + commit)
    artifact_path = validate_artifact_claim(artifact, commit) if artifact is not None else None
    previous = None
    if manifest_path.exists():
        manifest = json.loads(confined(manifest_path, state).read_text())
        if (manifest.get('source_commit') == commit and binary.is_file()
                and not binary.is_symlink() and digest(binary) == manifest.get('binary_sha256')
                and binary.stat().st_uid == os.getuid() and binary.stat().st_nlink == 1
                and source.is_dir() and not source.is_symlink()
                and source_digest(source) == manifest.get('source_sha256')
                and manifest.get('cgo_enabled') is False and manifest.get('elf_interpreter') is None
                and elf_interpreter(binary) is None):
            if artifact is None:
                return source, binary, manifest
            previous = manifest
    if (state / 'runtime-process.json').exists():
        fail('Detenga el proceso propio antes de recompilar.')
    snapshot_path = state / ('source-manifest-' + commit + '.json')
    if source.exists():
        if not snapshot_path.exists() or source.is_symlink():
            fail('Hay una fuente incompleta; revise el directorio privado.')
        snapshot = json.loads(confined(snapshot_path, state).read_text())
        if snapshot != {'source_commit': commit, 'source_sha256': source_digest(source)}:
            fail('La fuente privada cambió desde su extracción.')
    else:
        source.mkdir(mode=0o700)
        archive = state / 'source.tar'
        with archive.open('xb') as output:
            result = subprocess.run(['git', '-C', str(repo), 'archive', commit], stdout=output,
                                    stderr=subprocess.DEVNULL)
        if result.returncode:
            fail('No se pudo extraer la revisión fuente.')
        with tarfile.open(archive) as package:
            package.extractall(source, filter='data')
        archive.unlink()
        snapshot = {'source_commit': commit, 'source_sha256': source_digest(source)}
        write_json(snapshot_path, snapshot)
    go = local_go(source, go_binary)
    if artifact is not None:
        return import_artifact(repo, state, source, binary, snapshot, commit, go, artifact, artifact_path, previous)
    if binary.exists():
        fail('Hay un binario sin manifiesto válido; revise el estado privado.')
    go_path = Path.home() / 'go'
    environment = {'PATH': str(go.parent) + ':/usr/bin:/bin', 'HOME': str(state),
                   'TMPDIR': str(state / 'tmp'), 'GOCACHE': '/dev/shm/go-build',
                   'GOPATH': str(go_path), 'GOMODCACHE': str(go_path / 'pkg/mod'),
                   'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOSUMDB': 'off', 'CGO_ENABLED': '0'}
    (state / 'tmp').mkdir(mode=0o700, exist_ok=True)
    log_path = state / ('build-' + commit + '-cgo0.log')
    fd = os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as output:
        result = subprocess.run([str(go), 'build', '-p', '32', '-buildvcs=false', '-o', str(binary),
                                 './cmd/vec-server'], cwd=source, env=environment,
                                stdout=output, stderr=subprocess.STDOUT, timeout=900)
    if result.returncode:
        fail('La compilación falló; consulte el registro CGO0 de esta revisión.')
    interpreter = elf_interpreter(binary)
    if interpreter is not None:
        fail('El binario CGO0 requiere un intérprete externo.')
    manifest = {'version': 2, 'runtime_mode': 'interno', 'cgo_enabled': False, 'elf_interpreter': interpreter, 'source_commit': commit, 'source_tree': git(repo, 'rev-parse', commit + '^{tree}'),
                'binary_sha256': digest(binary), 'source_sha256': snapshot['source_sha256'], 'go_binary_sha256': digest(go),
                'build_command': 'go build -p 32 -buildvcs=false ./cmd/vec-server',
                'gocache': '/dev/shm/go-build'}
    if manifest_path.exists():
        previous = json.loads(confined(manifest_path, state).read_text())
        backup = state / ('runtime-manifest-' + previous['source_commit'] + '.json')
        if backup.exists() and json.loads(confined(backup, state).read_text()) != previous:
            fail('El manifiesto histórico no coincide con la revisión anterior.')
        if not backup.exists():
            write_json(backup, previous)
    write_json(manifest_path, manifest)
    return source, binary, manifest


def process_identity(pid):
    try:
        proc = Path('/proc') / str(pid)
        fields = (proc / 'stat').read_text().rsplit(')', 1)[1].split()
        if fields[0] == 'Z':
            return None
        return {'pid': pid, 'start_ticks': fields[19], 'exe': str((proc / 'exe').resolve(strict=True)),
                'uid': proc.stat().st_uid}
    except (OSError, ValueError, IndexError):
        return None



def own_process(state):
    path = state / 'runtime-process.json'
    if not path.exists():
        return None
    record = json.loads(confined(path, state).read_text())
    if record.get('container_mode') != 'interno':
        fail('El runtime requiere un recibo del contenedor interno propio.')
    module = container_module()
    try:
        module.verify_ownership(state, record)
    except module.ContainerError:
        fail('El contenedor o proceso del runtime no corresponde a su reserva.')
    return record


def stop(state):
    module = container_module()
    path = state / 'runtime-process.json'
    preimage = json.loads(confined(path, state).read_text()) if path.exists() else None
    container_record = module.read_record(state)
    removed = False
    if preimage is not None:
        keys = ('owner', 'state', 'container_mode', 'container_id', 'instance', 'pid',
                'start_ticks', 'source_commit', 'binary_sha256', 'image_id', 'uid', 'gid')
        if container_record is None and re.fullmatch(r'[a-f0-9]{64}', preimage.get('container_id', '')) and re.fullmatch(r'[a-f0-9]{32}', preimage.get('instance', '')):
            archived = state / ('runtime-container-stopped-' + preimage['container_id'] + '-' + preimage['instance'] + '.json')
            container_record = module.read_private_json(archived)
            if container_record is not None:
                try:
                    module.verify_record_boundary(state, container_record)
                    module.inspect(state, preimage['container_id'])
                except module.DockerNotFound:
                    removed = True
                except module.ContainerError:
                    fail('No se pudo acreditar la reserva retirada del contenedor propio.')
                else:
                    fail('El recibo retirado todavía tiene un contenedor existente.')
        if preimage.get('container_mode') != 'interno' or container_record is None or any(preimage.get(key) != container_record.get(key) for key in keys):
            fail('La reserva de proceso no corresponde al contenedor propio.')
    try:
        stopped = removed or module.stop(state)
    except module.ContainerError:
        fail('No se pudo detener el contenedor interno propio.')
    if stopped and preimage is not None and path.exists():
        current = json.loads(confined(path, state).read_text())
        if current != preimage:
            fail('La reserva de proceso cambió durante la parada.')
        path.unlink()
    return stopped


def verify_running(state, commit, port, pg_port):
    record = own_process(state)
    if record is None:
        fail('El clon no tiene un proceso interno activo.')
    descriptor = read_runtime_descriptor(state)
    root = Path(descriptor['root'])
    source = confined(state / ('source-' + commit), state, directory=True)
    material_sha = validate_material(root, commit, port, pg_port)
    environment, config_sha = runtime_environment(source, state, port, pg_port, root)
    if (record.get('source_commit') != commit or record.get('port') != port or record.get('pg_port') != pg_port
            or record.get('material_sha256') != material_sha or record.get('config_sha256') != config_sha
            or record.get('runtime_config_path') != descriptor['runtime_config_path']
            or record.get('runtime_manifest_path') != descriptor['manifest_path']):
        fail('El proceso interno perdió su configuración o material acreditados.')
    module = container_module()
    try:
        module.verify_record(state, record)
    except module.ContainerError:
        fail('La fuente, proyección o aislamiento del runtime cambió.')
    check_internal_https(state, port)
    if own_process(state) != record:
        fail('El runtime cambió durante la comprobación.')
    return record


def check_internal_https(state, port):
    # Client credentials stay on the host and are never mounted into the server.
    offline = confined(state / 'material', state, directory=True)
    context = ssl.create_default_context(cafile=str(confined(offline / 'ca/ca.crt', state)))
    context.load_cert_chain(str(confined(offline / 'mtls/cliente.crt', state)),
                            str(confined(offline / 'mtls/cliente.key', state)))
    with socket.create_connection(('127.0.0.1', port), timeout=3) as connection:
        with context.wrap_socket(connection, server_hostname='localhost') as tls:
            tls.sendall(b'GET /livez HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n')
            if not tls.recv(1024).split(b'\r\n', 1)[0].startswith(b'HTTP/1.1 200'):
                fail('El runtime interno no respondió HTTPS /livez 200.')


def start(source, binary, manifest, state, port, pg_port):
    if manifest.get('runtime_mode') != 'interno' or manifest.get('cgo_enabled') is not False or elf_interpreter(binary) is not None:
        fail('El runtime interno exige su binario estático CGO0 acreditado.')
    descriptor = read_runtime_descriptor(state)
    root = Path(descriptor['root'])
    material_sha = validate_material(root, manifest['source_commit'], port, pg_port)
    environment, config_sha = runtime_environment(source, state, port, pg_port, root)
    if own_process(state) is not None:
        fail('Ya hay un runtime interno propio; use restart o stop.')
    with socket.socket() as probe:
        try:
            probe.bind(('127.0.0.1', port))
        except OSError:
            fail('El puerto de aplicación está ocupado por otro proceso.')
    module = container_module()
    record = None
    succeeded = False
    try:
        record = module.start(state, source, binary, environment, manifest, descriptor, port, pg_port)
        record.update(material_sha256=material_sha, config_sha256=config_sha,
                      runtime_material_path=descriptor['material_path'], runtime_manifest_path=descriptor['manifest_path'])
        write_json(state / 'runtime-process.json', record)
        deadline = time.monotonic() + 60
        while time.monotonic() < deadline:
            try:
                module.verify_record(state, record)
                check_internal_https(state, port)
                module.verify_record(state, record)
                succeeded = True
                return record
            except (OSError, RuntimeErrorLocal, ssl.SSLError):
                time.sleep(0.2)
        fail('El runtime interno no alcanzó HTTPS /livez 200.')
    except module.ContainerError:
        fail('El runtime interno rechazó la creación o comprobación del contenedor.')
    finally:
        if not succeeded and record is not None:
            module.stop(state)
            path = state / 'runtime-process.json'
            if path.exists() and json.loads(confined(path, state).read_text()) == record:
                path.unlink()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['build', 'start', 'stop', 'restart', 'status', 'verify'])
    parser.add_argument('--repo', required=True, type=Path)
    parser.add_argument('--commit', required=True)
    parser.add_argument('--state', required=True, type=Path)
    parser.add_argument('--port', required=True, type=int)
    parser.add_argument('--pg-port', required=True, type=int)
    parser.add_argument('--mode', choices=['interno'])
    parser.add_argument('--go', help='Compilador local; por defecto usa el toolchain de go.mod si está instalado.')
    parser.add_argument('--artifact', type=Path)
    parser.add_argument('--artifact-sha256')
    parser.add_argument('--artifact-source')
    args = parser.parse_args()
    if not 1024 <= args.port <= 65535 or not 1024 <= args.pg_port <= 65535 or args.port == args.pg_port:
        fail('Puertos locales inválidos.')
    if args.action in ('build', 'start', 'restart', 'verify') and args.mode != 'interno':
        fail('Seleccione explícitamente --mode interno.')
    artifact = None
    if any((args.artifact, args.artifact_sha256, args.artifact_source)):
        if args.action not in ('build', 'start', 'restart') or not all((args.artifact, args.artifact_sha256, args.artifact_source)):
            fail('La importación exige archivo, huella y fuente explícitos en build, start o restart.')
        artifact = {'path': args.artifact, 'sha256': args.artifact_sha256, 'source_commit': args.artifact_source}
    repo = args.repo.resolve(strict=True)
    state = validate_state(repo, args.state)
    fd = os.open(state / 'runtime.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'a') as lock:
        operation = fcntl.LOCK_SH if args.action in ('status', 'verify') else fcntl.LOCK_EX
        try:
            fcntl.flock(lock, operation | fcntl.LOCK_NB)
        except BlockingIOError:
            fail('El runtime tiene otra operación incompatible en curso.')
        if args.action == 'stop':
            print(json.dumps({'stopped': stop(state)}))
            return
        if args.action == 'status':
            record = own_process(state)
            print(json.dumps({'running': record is not None, 'source_commit': (record or {}).get('source_commit')}))
            return
        commit = pinned_main(repo, args.commit)
        if artifact is not None:
            validate_artifact_claim(artifact, commit)
        if args.action == 'verify':
            record = verify_running(state, commit, args.port, args.pg_port)
            print(json.dumps({'verified': True, 'pid': record['pid'], 'source_commit': commit}))
            return
        if args.action == 'restart':
            stop(state)
        source, binary, manifest = build(repo, state, commit, args.go, artifact)
        if args.action == 'build':
            print(json.dumps(manifest))
        else:
            record = start(source, binary, manifest, state, args.port, args.pg_port)
            print(json.dumps({'running': True, 'pid': record['pid'], 'source_commit': commit,
                              'binary_sha256': manifest['binary_sha256'], 'port': args.port}))


if __name__ == '__main__':
    try:
        main()
    except (RuntimeErrorLocal, OSError, ValueError, subprocess.TimeoutExpired) as error:
        message = str(error) if isinstance(error, RuntimeErrorLocal) else 'Error local; consulte el estado privado del clon.'
        print(message, file=sys.stderr)
        sys.exit(1)
