#!/usr/bin/env python3
"""Compile and run a pinned main revision using private local clone material."""

import argparse
import fcntl
import hashlib
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
    temporary = path.with_suffix('.tmp')
    fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as stream:
        json.dump(value, stream, indent=2)
        stream.write('\n')
    os.replace(temporary, path)


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
    for name, value in values.items():
        if name.endswith('_DATABASE_URL'):
            validate_dsn(value, pg_port, state)
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
    log_path = state / 'runtime.log'
    fd = os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'ab') as output:
        child = subprocess.Popen([str(binary)], cwd=source, env=environment, stdin=subprocess.DEVNULL,
                                 stdout=output, stderr=subprocess.STDOUT, start_new_session=True)
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
    write_json(state / 'runtime-process.json', record)
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        if child.poll() is not None:
            (state / 'runtime-process.json').unlink(missing_ok=True)
            fail('El servidor rechazó el arranque; consulte runtime.log privado.')
        try:
            with socket.create_connection(('127.0.0.1', port), timeout=1) as connection:
                with context.wrap_socket(connection, server_hostname='localhost') as tls:
                    tls.sendall(b'GET /livez HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n')
                    first = tls.recv(1024).split(b'\r\n', 1)[0]
                    if first.startswith(b'HTTP/1.1 200'):
                        return record
        except (OSError, ssl.SSLError):
            pass
        time.sleep(0.2)
    stop(state)
    fail('El servidor no respondió HTTPS /livez 200; se detuvo solo el proceso propio.')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['build', 'start', 'stop', 'restart', 'status'])
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
