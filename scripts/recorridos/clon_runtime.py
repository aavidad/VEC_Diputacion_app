#!/usr/bin/env python3
"""Compile and run a pinned main revision using private local clone material."""

import argparse
import fcntl
import hashlib
import ipaddress
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


def validate_smtp(values, state):
    fields = {'VEC_SMTP_HOST', 'VEC_SMTP_PORT', 'VEC_SMTP_FROM', 'VEC_SMTP_CA_FILE', 'VEC_SMTP_MODO_TLS'}
    if not any(values.get(field) for field in fields):
        return
    if (any(not values.get(field) for field in fields) or values['VEC_SMTP_HOST'] != '127.0.0.1'
            or values['VEC_SMTP_FROM'] != 'rrhh@example.test'
            or values['VEC_SMTP_MODO_TLS'] != 'starttls'):
        fail('SMTP requiere el buzón sintético local y STARTTLS.')
    ca = confined(values['VEC_SMTP_CA_FILE'], state)
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
    if ca != state / 'material/ca/ca.crt':
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


def runtime_environment(source, state, port, pg_port):
    values, config_sha = read_environment(state)
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
    validate_smtp(values, state)
    for name, value in values.items():
        if name.endswith('_DATABASE_URL'):
            validate_dsn(value, pg_port, state)
        elif name == 'VEC_SMTP_HOST':
            continue  # Validated above against the owned synthetic STARTTLS sink.
        elif name == 'VEC_HTTP_ALLOWED_CIDRS':
            if value != '127.0.0.1/32':
                fail('La aplicación del clon sólo admite el CIDR loopback exacto.')
        elif name.endswith(('_URL', '_HOST', '_CIDRS')) and value:
            fail('Este clon no admite endpoints adicionales ni redes configuradas.')
        elif name.endswith(('_FILE', '_PATH', '_DIR')) and value:
            values[name] = str(confined(value, state, directory=name.endswith('_DIR')))
    material = confined(values['VEC_DEVELOPMENT_MATERIAL_DIR'], state, directory=True)
    confined(material / 'ca/ca.crt', state)
    values['VEC_HTTP_ADDR'] = f'127.0.0.1:{port}'
    values['VEC_HTTP_ALLOWED_CIDRS'] = '127.0.0.1/32'
    values['VEC_PERSONAL_CATALOG_PATH'] = str(state / 'personal-catalog.json')
    environment = {'PATH': '/usr/bin:/bin', 'HOME': str(state), 'TMPDIR': str(state / 'tmp'), 'TZ': 'UTC'}
    environment.update(values)
    (state / 'tmp').mkdir(mode=0o700, exist_ok=True)
    return environment, config_sha


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


def build(repo, state, commit, go_binary):
    manifest_path = state / 'runtime-manifest.json'
    source = state / ('source-' + commit)
    binary = state / ('vec-server-' + commit)
    if manifest_path.exists():
        manifest = json.loads(confined(manifest_path, state).read_text())
        if (manifest.get('source_commit') == commit and binary.is_file()
                and not binary.is_symlink() and digest(binary) == manifest.get('binary_sha256')
                and source.is_dir() and not source.is_symlink()
                and source_digest(source) == manifest.get('source_sha256')):
            return source, binary, manifest
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
    if binary.exists():
        fail('Hay un binario sin manifiesto válido; revise el estado privado.')
    go = local_go(source, go_binary)
    go_path = Path.home() / 'go'
    environment = {'PATH': str(go.parent) + ':/usr/bin:/bin', 'HOME': str(state),
                   'TMPDIR': str(state / 'tmp'), 'GOCACHE': '/dev/shm/go-build',
                   'GOPATH': str(go_path), 'GOMODCACHE': str(go_path / 'pkg/mod'),
                   'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOSUMDB': 'off'}
    (state / 'tmp').mkdir(mode=0o700, exist_ok=True)
    log_path = state / 'build.log'
    fd = os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as output:
        result = subprocess.run([str(go), 'build', '-p', '32', '-buildvcs=false', '-o', str(binary),
                                 './cmd/vec-server'], cwd=source, env=environment,
                                stdout=output, stderr=subprocess.STDOUT, timeout=900)
    if result.returncode:
        fail('La compilación falló; consulte build.log en el directorio privado.')
    manifest = {'version': 1, 'source_commit': commit, 'source_tree': git(repo, 'rev-parse', commit + '^{tree}'),
                'binary_sha256': digest(binary), 'source_sha256': snapshot['source_sha256'], 'go_binary_sha256': digest(go),
                'build_command': 'go build -p 32 -buildvcs=false ./cmd/vec-server',
                'gocache': '/dev/shm/go-build'}
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
    identity = process_identity(record.get('pid'))
    if identity is None:
        path.unlink()
        return None
    expected = {key: record.get(key) for key in ('pid', 'start_ticks', 'exe', 'uid')}
    if identity != expected or identity['uid'] != os.getuid():
        fail('El PID guardado pertenece a otro proceso; no se enviará ninguna señal.')
    binary = confined(record['exe'], state)
    if digest(binary) != record.get('binary_sha256'):
        fail('El binario del proceso no coincide con su huella guardada.')
    return record


def stop(state):
    record = own_process(state)
    if record is None:
        return False
    # pidfd targets the verified process, including if its PID is subsequently reused.
    descriptor = os.pidfd_open(record['pid'])
    try:
        if own_process(state) != record:
            fail('El proceso cambió antes de detenerlo.')
        signal.pidfd_send_signal(descriptor, signal.SIGTERM)
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            if process_identity(record['pid']) is None:
                (state / 'runtime-process.json').unlink(missing_ok=True)
                return True
            time.sleep(0.1)
        fail('El proceso propio no terminó; no se ha forzado su cierre.')
    finally:
        os.close(descriptor)


def verify_running(state, commit, port, pg_port):
    record = own_process(state)
    if record is None:
        fail('El clon no tiene un proceso propio activo.')
    manifest = json.loads(confined(state / 'runtime-manifest.json', state).read_text())
    source = confined(state / ('source-' + commit), state, directory=True)
    if (record.get('source_commit') != commit or manifest.get('source_commit') != commit
            or record.get('port') != port or record.get('pg_port') != pg_port
            or manifest.get('binary_sha256') != record.get('binary_sha256')
            or source_digest(source) != manifest.get('source_sha256')):
        fail('El proceso activo no corresponde a su fuente y binario acreditados.')
    material_sha = validate_material(state, commit, port, pg_port)
    environment, config_sha = runtime_environment(source, state, port, pg_port)
    if record.get('material_sha256') != material_sha or record.get('config_sha256') != config_sha:
        fail('El material o la configuración cambiaron después del arranque.')
    material = Path(environment['VEC_DEVELOPMENT_MATERIAL_DIR'])
    context = ssl.create_default_context(cafile=str(material / 'ca/ca.crt'))
    context.load_cert_chain(str(confined(material / 'mtls/cliente.crt', state)),
                            str(confined(material / 'mtls/cliente.key', state)))
    with socket.create_connection(('127.0.0.1', port), timeout=3) as connection:
        with context.wrap_socket(connection, server_hostname='localhost') as tls:
            tls.sendall(b'GET /livez HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n')
            if not tls.recv(1024).split(b'\r\n', 1)[0].startswith(b'HTTP/1.1 200'):
                fail('El proceso activo no respondió HTTPS /livez 200.')
    if own_process(state) != record:
        fail('El proceso cambió durante su comprobación.')
    return record


def stop_started_child(child, descriptor):
    """Use the descriptor of the freshly spawned child, never a saved PID."""
    if child.poll() is not None:
        child.wait()
        return
    if descriptor is not None:
        try:
            signal.pidfd_send_signal(descriptor, signal.SIGTERM)
        except ProcessLookupError:
            pass
    else:
        # If descriptor allocation failed, retain the unreaped child identity:
        # waitid(WNOWAIT) cannot make its PID available to an unrelated process.
        try:
            ended = os.waitid(os.P_PID, child.pid, os.WEXITED | os.WNOHANG | os.WNOWAIT)
            if ended is None:
                os.kill(child.pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
    try:
        child.wait(timeout=5)
    except subprocess.TimeoutExpired:
        if descriptor is not None:
            signal.pidfd_send_signal(descriptor, signal.SIGKILL)
        else:
            os.kill(child.pid, signal.SIGKILL)  # Still our unreaped, directly created child.
        child.wait(timeout=5)


def start(source, binary, manifest, state, port, pg_port):
    if own_process(state):
        fail('Ya hay un proceso propio; use restart o stop.')
    material_sha = validate_material(state, manifest['source_commit'], port, pg_port)
    environment, config_sha = runtime_environment(source, state, port, pg_port)
    material = Path(environment['VEC_DEVELOPMENT_MATERIAL_DIR'])
    context = ssl.create_default_context(cafile=str(material / 'ca/ca.crt'))
    context.load_cert_chain(str(confined(material / 'mtls/cliente.crt', state)),
                            str(confined(material / 'mtls/cliente.key', state)))
    with socket.socket() as probe:
        try:
            probe.bind(('127.0.0.1', port))
        except OSError:
            fail('El puerto de aplicación está ocupado; no se tocará su proceso.')
    if signal.getsignal(signal.SIGCHLD) == signal.SIG_IGN:
        fail('El lanzador no puede conservar la identidad del proceso hijo.')
    # Check kernel support before spawning; hold the child descriptor from the
    # first instruction after Popen until publication and HTTPS verification.
    capability = os.pidfd_open(os.getpid())
    os.close(capability)
    log_path = state / 'runtime.log'
    fd = os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'ab') as output:
        child = subprocess.Popen([str(binary)], cwd=source, env=environment, stdin=subprocess.DEVNULL,
                                 stdout=output, stderr=subprocess.STDOUT, start_new_session=True)
    descriptor = None
    succeeded = False
    record = None
    record_path = state / 'runtime-process.json'
    try:
        descriptor = os.pidfd_open(child.pid)
        identity = None
        for _ in range(100):
            identity = process_identity(child.pid)
            if identity and identity['exe'] == str(binary):
                break
            if child.poll() is not None:
                fail('El servidor terminó al arrancar; consulte runtime.log privado.')
            time.sleep(0.01)
        if not identity or identity['exe'] != str(binary):
            fail('No se pudo comprobar la identidad del proceso recién iniciado.')
        record = dict(identity, source_commit=manifest['source_commit'], binary_sha256=manifest['binary_sha256'],
                      config_sha256=config_sha, material_sha256=material_sha, port=port, pg_port=pg_port)
        write_json(record_path, record)
        deadline = time.monotonic() + 60
        while time.monotonic() < deadline:
            if child.poll() is not None:
                fail('El servidor rechazó el arranque; consulte runtime.log privado.')
            try:
                with socket.create_connection(('127.0.0.1', port), timeout=1) as connection:
                    with context.wrap_socket(connection, server_hostname='localhost') as tls:
                        tls.sendall(b'GET /livez HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n')
                        first = tls.recv(1024).split(b'\r\n', 1)[0]
                        if first.startswith(b'HTTP/1.1 200'):
                            succeeded = True
                            return record
            except (OSError, ssl.SSLError):
                pass
            time.sleep(0.2)
        fail('El servidor no respondió HTTPS /livez 200; se detuvo solo el proceso propio.')
    finally:
        try:
            if not succeeded:
                stop_started_child(child, descriptor)
                # Remove only the record published by this invocation. A failure
                # or changed record cannot erase another process's reservation.
                if record is not None and record_path.exists():
                    if json.loads(confined(record_path, state).read_text()) == record:
                        record_path.unlink()
        finally:
            if descriptor is not None:
                os.close(descriptor)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['build', 'start', 'stop', 'restart', 'status', 'verify'])
    parser.add_argument('--repo', required=True, type=Path)
    parser.add_argument('--commit', required=True)
    parser.add_argument('--state', required=True, type=Path)
    parser.add_argument('--port', required=True, type=int)
    parser.add_argument('--pg-port', required=True, type=int)
    parser.add_argument('--go', help='Compilador local; por defecto usa el toolchain de go.mod si está instalado.')
    args = parser.parse_args()
    if not 1024 <= args.port <= 65535 or not 1024 <= args.pg_port <= 65535 or args.port == args.pg_port:
        fail('Puertos locales inválidos.')
    repo = args.repo.resolve(strict=True)
    state = validate_state(repo, args.state)
    fd = os.open(state / 'runtime.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        if args.action == 'stop':
            print(json.dumps({'stopped': stop(state)}))
            return
        if args.action == 'status':
            record = own_process(state)
            print(json.dumps({'running': record is not None, 'source_commit': (record or {}).get('source_commit')}))
            return
        commit = pinned_main(repo, args.commit)
        if args.action == 'verify':
            record = verify_running(state, commit, args.port, args.pg_port)
            print(json.dumps({'verified': True, 'pid': record['pid'], 'source_commit': commit}))
            return
        if args.action == 'restart':
            stop(state)
        source, binary, manifest = build(repo, state, commit, args.go)
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
