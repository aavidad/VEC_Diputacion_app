#!/usr/bin/env python3
"""Own the Docker process boundary for the private internal clone only.

Host networking is required for the owned loopback PostgreSQL/SMTP services.
It does not isolate the network namespace. The parent seals endpoint policy and
performs HTTPS health checks; this module never mounts offline client secrets.
"""

import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import signal
import stat
import struct
import subprocess
import tempfile
import time


OWNER = 'Codex-M'
PREFIX = 'vec.clon.runtime.'
HOME = '/home/runtime'
RW_KINDS = {'documentos', 'imagenes', 'data', 'comunicaciones'}


class ContainerError(Exception):
    pass


class DockerNotFound(ContainerError):
    pass


def fail(message):
    raise ContainerError(message)


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def object_sha(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(',', ':')).encode()).hexdigest()


def checked_path(value, root=None, directory=False):
    path = Path(value)
    if not path.is_absolute() or '..' in path.parts or ',' in str(path):
        fail('Invalid absolute runtime path.')
    if root is not None and path != root and root not in path.parents:
        fail('Runtime mount escapes its owned root.')
    for item in [path, *path.parents]:
        if item.is_symlink():
            fail('Runtime paths cannot contain symbolic links.')
    metadata = path.stat()
    if metadata.st_uid != os.getuid():
        fail('Runtime path belongs to another user.')
    if directory:
        if not stat.S_ISDIR(metadata.st_mode):
            fail('Runtime directory is missing.')
    elif not stat.S_ISREG(metadata.st_mode) or metadata.st_nlink != 1:
        fail('Runtime file is not a private regular file.')
    return path


def private_json(path, value):
    # No replacement of a foreign reservation, including a symlink.
    descriptor, name = tempfile.mkstemp(prefix='.' + path.name + '-', dir=path.parent)
    temporary = Path(name)
    try:
        with os.fdopen(descriptor, 'w') as stream:
            json.dump(value, stream, sort_keys=True, indent=2)
            stream.write('\n')
            stream.flush()
            os.fsync(stream.fileno())
        # link publishes the complete file atomically and refuses an existing name.
        os.link(temporary, path, follow_symlinks=False)
    finally:
        temporary.unlink(missing_ok=True)


def read_record(state):
    return read_private_json(Path(state) / 'runtime-container.json')


def read_private_json(path):
    if not path.exists() and not path.is_symlink():
        return None
    checked_path(path, path.parent)
    if stat.S_IMODE(path.stat().st_mode) != 0o600:
        fail('Runtime container record requires mode 0600.')
    return json.loads(path.read_text())


def docker(state, *args):
    config = Path(state) / 'runtime-docker-client'
    config.mkdir(mode=0o700, exist_ok=True)
    checked_path(config, Path(state), directory=True)
    # No inherited Docker contexts, remote daemon, credential helpers or proxies.
    try:
        result = subprocess.run(['docker', '--config', str(config), '--host', 'unix:///var/run/docker.sock', *args],
                                env={'PATH': '/usr/bin:/bin', 'HOME': str(config)},
                                stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=60)
    except subprocess.TimeoutExpired:
        fail('Local Docker operation timed out; creation intent is preserved for recovery.')
    if result.returncode:
        if args[:2] == ('container', 'inspect') and re.search(r'No such (?:object|container):', result.stderr):
            raise DockerNotFound('The immutable container ID no longer exists.')
        fail('Local Docker operation failed; no secret-bearing output is displayed.')
    return result.stdout.strip()


def inspect(state, container_id):
    if not re.fullmatch(r'[a-f0-9]{64}', container_id):
        fail('An immutable Docker container ID is required.')
    result = json.loads(docker(state, 'container', 'inspect', container_id))
    if len(result) != 1 or result[0].get('Id') != container_id:
        fail('Docker returned a different container.')
    return result[0]


def source_sha(source):
    digest = hashlib.sha256()
    for path in sorted(Path(source).rglob('*')):
        if path.is_symlink():
            fail('Pinned source contains a symbolic link.')
        if path.is_file():
            digest.update(str(path.relative_to(source)).encode() + b'\0' + sha(path).encode() + b'\n')
    return digest.hexdigest()


def static_elf(path):
    with Path(path).open('rb') as stream:
        header = stream.read(64)
        if len(header) < 64 or header[:4] != b'\x7fELF' or header[4] not in (1, 2) or header[5] not in (1, 2):
            return False
        endian = '<' if header[5] == 1 else '>'
        if header[4] == 2:
            offset = struct.unpack_from(endian + 'Q', header, 32)[0]
            size, count = struct.unpack_from(endian + 'HH', header, 54)
        else:
            offset = struct.unpack_from(endian + 'I', header, 28)[0]
            size, count = struct.unpack_from(endian + 'HH', header, 42)
        if size < 4 or count > 128 or offset + size * count > os.fstat(stream.fileno()).st_size:
            return False
        for index in range(count):
            stream.seek(offset + size * index)
            if struct.unpack(endian + 'I', stream.read(4))[0] == 3:  # PT_INTERP
                return False
        return True


def projection_sha(projection):
    files = {}
    for entry in projection['ro']:
        path = Path(entry['source'])
        for item in sorted(path.rglob('*')) if path.is_dir() else [path]:
            if item.is_file():
                files[str(item)] = sha(item)
    return object_sha(files)


def approved_plan(state, source, binary, manifest, projection):
    """Validate recorded boundaries without requiring unchanged live material."""
    commit = manifest.get('source_commit', '')
    if not re.fullmatch(r'[a-f0-9]{40}', commit):
        fail('Invalid recorded source revision.')
    root = state / 'runtime-interno'
    if (source != state / ('source-' + commit) or binary != state / ('vec-server-' + commit)
            or projection.get('root') != str(root) or projection.get('mode') != 'interno'
            or projection.get('portal') != 'interno'):
        fail('Recorded runtime paths or realm are invalid.')
    env_path = Path(projection.get('runtime_config_path', str(root / 'runtime-config.json')))
    if env_path != root / 'runtime-config.json':
        fail('Unapproved projected runtime configuration.')
    material_path = Path(projection.get('material_path', str(root / 'material')))
    marker_path = Path(projection.get('manifest_path', str(root / 'material-manifest.json')))
    if material_path != root / 'material' or marker_path != root / 'material-manifest.json':
        fail('Unapproved projected material paths.')
    allowed_ro = {material_path, env_path, root / 'runtime.env', marker_path}
    mounts = [{'source': str(source), 'target': str(source), 'rw': False},
              {'source': str(binary), 'target': str(binary), 'rw': False}]
    given_ro = set()
    for entry in projection.get('ro', []):
        path = Path(entry.get('source', ''))
        if path not in allowed_ro or entry.get('target') != str(path) or path in given_ro:
            fail('Unapproved recorded read-only mount.')
        given_ro.add(path)
        mounts.append({'source': str(path), 'target': str(path), 'rw': False})
    if given_ro != allowed_ro:
        fail('Recorded read-only projection is incomplete.')
    given_rw = set()
    for entry in projection.get('rw', []):
        kind = entry.get('kind')
        path = Path(entry.get('source', ''))
        target = root / 'material/comunicaciones' if kind == 'comunicaciones' else path
        if (kind not in RW_KINDS or kind in given_rw or path != root / 'rw' / kind
                or entry.get('target') != str(target)):
            fail('Unapproved recorded writable mount.')
        given_rw.add(kind)
        mounts.append({'source': str(path), 'target': str(target), 'rw': True})
    if given_rw != RW_KINDS:
        fail('The four owned writable directories are required.')
    return sorted(mounts, key=lambda item: item['target'])


def verify_record_boundary(state, record):
    if (record.get('owner') != OWNER or record.get('state') != str(state) or record.get('container_mode') != 'interno'
            or record.get('uid') != os.getuid() or record.get('gid') != os.getgid()
            or record.get('source_commit') != record['manifest'].get('source_commit')
            or record.get('name') != container_name(state, record['source_commit'])
            or not re.fullmatch(r'sha256:[a-f0-9]{64}', record.get('image_id', ''))
            or not re.fullmatch(r'[a-f0-9]{32}', record.get('instance', ''))):
        fail('Container reservation belongs to another owner or boundary.')
    mounts = approved_plan(state, Path(record['source']), Path(record['binary']), record['manifest'], record['projection'])
    if record.get('mounts') != mounts:
        fail('Container reservation has unapproved boundaries.')


def validate_inputs(state, source, binary, manifest, projection):
    state = checked_path(state, directory=True)
    if stat.S_IMODE(state.stat().st_mode) & 0o077:
        fail('Runtime state must be private.')
    commit = manifest.get('source_commit', '')
    if not re.fullmatch(r'[a-f0-9]{40}', commit):
        fail('Pinned source revision is required.')
    source = checked_path(source, state, directory=True)
    binary = checked_path(binary, state)
    if source != state / ('source-' + commit) or binary != state / ('vec-server-' + commit):
        fail('Runtime source or binary path belongs to another revision.')
    if (manifest.get('cgo_enabled') is not False or 'elf_interpreter' not in manifest
            or manifest['elf_interpreter'] is not None or not static_elf(binary)
            or sha(binary) != manifest.get('binary_sha256')
            or source_sha(source) != manifest.get('source_sha256')):
        fail('Runtime static binary or source proof does not match.')
    root = checked_path(projection.get('root', ''), state, directory=True)
    plan = approved_plan(state, source, binary, manifest, projection)
    if root != state / 'runtime-interno' or projection.get('mode') != 'interno' or projection.get('portal') != 'interno':
        fail('Only the internal runtime projection is admitted.')
    mounts = [{'source': str(source), 'target': str(source), 'rw': False},
              {'source': str(binary), 'target': str(binary), 'rw': False}]
    allowed_ro = {Path(mount['source']) for mount in plan if not mount['rw']} - {source, binary}
    given_ro = set()
    for entry in projection.get('ro', []):
        path = Path(entry.get('source', ''))
        if path not in allowed_ro or entry.get('target') != str(path) or path in given_ro:
            fail('Unapproved read-only runtime mount.')
        checked_path(path, root, directory=path == root / 'material')
        given_ro.add(path)
        mounts.append({'source': str(path), 'target': str(path), 'rw': False})
    if given_ro != allowed_ro:
        fail('The internal projection is incomplete.')
    marker = json.loads((root / 'material-manifest.json').read_text())
    if marker.get('portal') != 'interno':
        fail('Material belongs to another portal realm.')
    for path in (root / 'material').rglob('*'):
        checked_path(path, root, directory=path.is_dir())
        relative = str(path.relative_to(root / 'material')).lower()
        if (any(part in relative for part in ('externo', 'external', 'admin', 'backup', '.p12', '.pfx'))
                or re.search(r'(^|/)(ca[^/]*\.key|cliente[^/]*\.(key|pem)|client[^/]*\.(key|pem))$', relative)
                or (path.is_file() and relative.startswith('ca/') and b'PRIVATE KEY' in path.read_bytes())):
            fail('Offline or foreign-realm secrets cannot enter the runtime projection.')
    given_rw = set()
    for entry in projection.get('rw', []):
        kind = entry.get('kind')
        path = Path(entry.get('source', ''))
        target = root / 'material/comunicaciones' if kind == 'comunicaciones' else path
        if (kind not in RW_KINDS or kind in given_rw or path != root / 'rw' / kind
                or entry.get('target') != str(target)):
            fail('Unapproved writable runtime mount.')
        checked_path(path, root, directory=True)
        # A writable tree cannot contain a path that bypasses the bind boundary.
        for child in path.rglob('*'):
            checked_path(child, path, directory=child.is_dir())
        given_rw.add(kind)
        mounts.append({'source': str(path), 'target': str(target), 'rw': True})
    if given_rw != RW_KINDS:
        fail('The four owned writable directories are required.')
    return sorted(mounts, key=lambda item: item['target'])


def build_image(state, source_commit):
    uid, gid = os.getuid(), os.getgid()
    if uid == 0:
        fail('The internal runtime cannot run as root.')
    with tempfile.TemporaryDirectory(prefix='runtime-image-', dir=state) as context_name:
        context = Path(context_name)
        (context / 'home/runtime').mkdir(parents=True, mode=0o755)
        (context / 'passwd').write_text(f'runtime:x:{uid}:{gid}:runtime:{HOME}:/nonexistent\n')
        (context / 'group').write_text(f'runtime:x:{gid}:\n')
        (context / 'Dockerfile').write_text(
            'FROM scratch\nCOPY passwd /etc/passwd\nCOPY group /etc/group\n'
            'COPY home /home\n' + f'USER {uid}:{gid}\n' + f'ENV HOME={HOME} TMPDIR=/tmp TZ=UTC\n')
        iidfile = context / 'image-id'
        docker(state, 'build', '--network=none', '--pull=false', '--label', PREFIX + 'owner=' + OWNER,
               '--label', PREFIX + 'state=' + str(state), '--label', PREFIX + 'source=' + source_commit,
               '--iidfile', str(iidfile), str(context))
        image_id = iidfile.read_text().strip()
    if not re.fullmatch(r'sha256:[a-f0-9]{64}', image_id):
        fail('Docker did not produce an immutable scratch image ID.')
    image = json.loads(docker(state, 'image', 'inspect', image_id))[0]
    labels = image.get('Config', {}).get('Labels', {})
    if (image.get('Id') != image_id or labels.get(PREFIX + 'owner') != OWNER
            or labels.get(PREFIX + 'state') != str(state) or labels.get(PREFIX + 'source') != source_commit
            or image.get('RootFS', {}).get('Type') != 'layers'):
        fail('Scratch image proof does not match.')
    return image_id


def process_identity(pid):
    try:
        if not isinstance(pid, int) or pid <= 0:
            return None
        proc = Path('/proc') / str(pid)
        fields = (proc / 'stat').read_text().rsplit(')', 1)[1].split()
        if fields[0] == 'Z':
            return None
        status = dict(line.split(':', 1) for line in (proc / 'status').read_text().splitlines() if ':' in line)
        if not isolated_process_status(status):
            return None
        exe = os.readlink(proc / 'exe')
        if (proc / 'cmdline').read_bytes().split(b'\0') != [exe.encode(), b'']:
            return None
        return {'pid': pid, 'start_ticks': fields[19], 'exe': exe,
                'uid': proc.stat().st_uid, 'binary_sha256': sha(proc / 'exe')}
    except (OSError, ValueError, IndexError):
        return None


def isolated_process_status(status):
    """Kernel evidence supplements Docker's declared root, caps and PID policy."""
    zero_caps = ('CapInh', 'CapPrm', 'CapEff', 'CapBnd', 'CapAmb')
    try:
        namespace_pids = status.get('NSpid', '').split()
        return (status.get('Seccomp', '').strip() == '2' and status.get('NoNewPrivs', '').strip() == '1'
                and status.get('Uid', '').split() == [str(os.getuid())] * 4
                and status.get('Gid', '').split() == [str(os.getgid())] * 4
                and len(namespace_pids) >= 2 and namespace_pids[-1] == '1'
                and all(int(status[key].strip(), 16) == 0 for key in zero_caps))
    except (KeyError, ValueError):
        return False


def expected_labels(state, record):
    return {PREFIX + 'owner': OWNER, PREFIX + 'state': str(state), PREFIX + 'mode': 'interno',
            PREFIX + 'source': record['source_commit'], PREFIX + 'image_id': record['image_id'],
            PREFIX + 'uid': str(os.getuid()), PREFIX + 'mounts': object_sha(record['mounts']),
            PREFIX + 'instance': record['instance']}


def verify_configuration(state, record, proof):
    config, host = proof.get('Config', {}), proof.get('HostConfig', {})
    labels = config.get('Labels', {})
    if any(labels.get(key) != value for key, value in expected_labels(state, record).items()):
        fail('Container ownership labels do not match.')
    if (proof.get('Id') != record['container_id'] or proof.get('Image') != record['image_id']
            or config.get('Image') != record['image_id'] or config.get('User') != f'{os.getuid()}:{os.getgid()}'
            or config.get('WorkingDir') != record['source'] or config.get('Entrypoint') != [record['binary']]
            or config.get('Cmd') not in (None, []) or proof.get('Path') != record['binary']
            or proof.get('Args') not in (None, []) or object_sha(sorted(config.get('Env', []))) != record['env_sha256']):
        fail('Container source, executable, user or environment changed.')
    if (host.get('ReadonlyRootfs') is not True or host.get('Privileged') is not False
            or host.get('NetworkMode') != 'host' or host.get('PidMode') not in ('', None)
            or host.get('UsernsMode') not in ('', None) or host.get('AutoRemove') is not False
            or host.get('IpcMode') not in ('private', '') or host.get('Init') not in (False, None)
            or host.get('CapDrop') != ['ALL'] or host.get('CapAdd') not in (None, [])
            or host.get('SecurityOpt') != ['no-new-privileges'] or host.get('Devices') not in (None, [])
            or host.get('DeviceRequests') not in (None, []) or host.get('VolumesFrom') not in (None, [])
            or host.get('RestartPolicy', {}).get('Name') != 'no'
            or host.get('PidsLimit') != 64 or host.get('Memory') != 1073741824
            or host.get('NanoCpus') != 2000000000
            or sorted(host.get('Ulimits') or [], key=lambda item: item['Name']) != [
                {'Name': 'fsize', 'Hard': 67108864, 'Soft': 67108864},
                {'Name': 'nofile', 'Hard': 1024, 'Soft': 1024}]
            or host.get('Tmpfs') != {'/tmp': 'rw,noexec,nosuid,nodev,size=16m,mode=1777'}):
        fail('Container isolation configuration changed.')
    observed = []
    for mount in proof.get('Mounts', []):
        if mount.get('Type') == 'tmpfs' and mount.get('Destination') == '/tmp':
            continue
        if mount.get('Type') != 'bind' or mount.get('Propagation') not in ('rprivate', 'private'):
            fail('Container has an unapproved mount type or propagation.')
        observed.append({'source': mount.get('Source'), 'target': mount.get('Destination'), 'rw': mount.get('RW')})
    if sorted(observed, key=lambda item: item['target']) != record['mounts']:
        fail('Container exposes an unapproved mount.')


def verify_record(state, record=None):
    state = checked_path(state, directory=True)
    if record is None:
        record = read_record(state)
    if record is None:
        fail('No owned runtime container is recorded.')
    expected_mounts = validate_inputs(state, record['source'], record['binary'], record['manifest'], record['projection'])
    if (record.get('uid') != os.getuid() or record.get('mounts') != expected_mounts
            or record.get('projection_sha256') != projection_sha(record['projection'])):
        fail('Container record has a foreign user or mount plan.')
    verify_ownership(state, record)
    return record


def verify_ownership(state, record):
    verify_record_boundary(state, record)
    proof = inspect(state, record['container_id'])
    verify_configuration(state, record, proof)
    if proof.get('State', {}).get('Running') is not True or proof['State'].get('Pid') != record['pid']:
        fail('Recorded container is not the current live process.')
    identity = process_identity(record['pid'])
    if identity != {key: record[key] for key in ('pid', 'start_ticks', 'exe', 'uid', 'binary_sha256')}:
        fail('Host PID identity changed; no signal will be sent.')
    if identity['exe'] != record['binary'] or identity['binary_sha256'] != record['manifest']['binary_sha256']:
        fail('The container does not execute the pinned Go binary directly.')
    return record


def container_name(state, commit):
    return 'vec-clon-interno-' + object_sha([str(state), commit])[:24]


def start(state, source, binary, environment, manifest, projection, port, pg_port):
    state, source, binary = Path(state), Path(source), Path(binary)
    mounts = validate_inputs(state, source, binary, manifest, projection)
    recover_intent(state)
    if read_record(state) is not None:
        fail('An owned container reservation already exists.')
    if (environment.get('HOME') != HOME or environment.get('TMPDIR') != '/tmp'
            or environment.get('TZ') != 'UTC' or environment.get('PATH') != '/usr/bin:/bin'
            or environment.get('VEC_HTTP_ADDR') != f'127.0.0.1:{port}'
            or environment.get('VEC_HTTP_ALLOWED_CIDRS') != '127.0.0.1/32'):
        fail('The sealed runtime environment has invalid process limits.')
    if not all(isinstance(key, str) and re.fullmatch(r'(VEC_[A-Z0-9_]+|HOME|TMPDIR|TZ|PATH)', key)
               and isinstance(value, str) and not any(char in value for char in '\0\n\r')
               for key, value in environment.items()):
        fail('The sealed runtime environment is invalid.')
    if not hasattr(os, 'pidfd_open') or not hasattr(signal, 'pidfd_send_signal'):
        fail('The host lacks the process identity primitive required for this runtime.')
    capability = os.pidfd_open(os.getpid())
    os.close(capability)
    image_id = build_image(state, manifest['source_commit'])
    record = {'source': str(source), 'binary': str(binary), 'source_commit': manifest['source_commit'],
              'manifest': manifest, 'projection': projection, 'mounts': mounts, 'image_id': image_id,
              'uid': os.getuid(), 'gid': os.getgid(), 'port': port, 'pg_port': pg_port,
              'projection_sha256': projection_sha(projection),
              'env_sha256': object_sha(sorted(f'{key}={value}' for key, value in environment.items())),
              'network_namespace_isolated': False, 'version': 1}
    record.update(instance=secrets.token_hex(16), container_mode='interno',
                  owner=OWNER, state=str(state),
                  runtime_config_path=projection.get('runtime_config_path', str(Path(projection['root']) / 'runtime-config.json')),
                  material_path=projection.get('material_path', str(Path(projection['root']) / 'material')),
                  manifest_path=projection.get('manifest_path', str(Path(projection['root']) / 'material-manifest.json')))
    env_bytes = ''.join(f'{key}={value}\n' for key, value in sorted(environment.items())).encode()
    env_path = state / ('runtime-container-env-' + hashlib.sha256(env_bytes).hexdigest() + '.env')
    if env_path.exists() or env_path.is_symlink():
        checked_path(env_path, state)
        if sha(env_path) != hashlib.sha256(env_bytes).hexdigest() or stat.S_IMODE(env_path.stat().st_mode) != 0o600:
            fail('Private Docker environment export changed.')
    else:
        descriptor = os.open(env_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, 'wb') as stream:
            stream.write(env_bytes)
    command = ['create', '--name', container_name(state, manifest['source_commit']), '--read-only',
               '--network=host', '--ipc=private', '--user', f'{os.getuid()}:{os.getgid()}',
               '--cap-drop=ALL', '--security-opt=no-new-privileges', '--restart=no',
               '--tmpfs', '/tmp:rw,noexec,nosuid,nodev,size=16m,mode=1777',
               '--pids-limit=64', '--memory=1g', '--cpus=2', '--ulimit', 'nofile=1024:1024',
               '--ulimit', 'fsize=67108864:67108864', '--workdir', str(source), '--entrypoint', str(binary),
               '--env-file', str(env_path)]
    for key, value in expected_labels(state, record).items():
        command.extend(['--label', f'{key}={value}'])
    for mount in mounts:
        command.extend(['--mount', f'type=bind,source={mount["source"]},target={mount["target"]},bind-propagation=rprivate'
                        + ('' if mount['rw'] else ',readonly')])
    command.append(image_id)
    record['name'] = container_name(state, manifest['source_commit'])
    intent_path = state / 'runtime-container-intent.json'
    private_json(intent_path, record)
    succeeded = False
    try:
        container_id = docker(state, *command)
        record['container_id'] = container_id
        # Validate the immutable object before allowing even its first instruction.
        verify_configuration(state, record, inspect(state, container_id))
        docker(state, 'start', container_id)
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            proof = inspect(state, container_id)
            pid = proof.get('State', {}).get('Pid')
            identity = process_identity(pid)
            if identity and identity['exe'] == str(binary):
                record.update(identity)
                break
            if proof.get('State', {}).get('Running') is not True:
                fail('The internal runtime exited during startup.')
            time.sleep(0.05)
        if 'pid' not in record:
            fail('The direct runtime process could not be verified.')
        verify_record(state, record)
        private_json(state / 'runtime-container.json', record)
        succeeded = True
        intent_path.unlink()
        return record
    finally:
        if not succeeded:
            # Failed publication cannot destroy a reservation created elsewhere.
            if 'container_id' in record:
                _remove_started(state, record)
                intent_path.unlink(missing_ok=True)
            else:
                recover_intent(state)


def recover_intent(state):
    """Recover an interrupted create by name, then act only on its immutable ID."""
    intent_path = Path(state) / 'runtime-container-intent.json'
    record = read_private_json(intent_path)
    if record is None:
        return False
    verify_record_boundary(Path(state), record)
    # Listing does not mutate foreign objects. A daemon failure preserves intent.
    matches = docker(state, 'container', 'ls', '-aq', '--no-trunc', '--filter', 'name=^/' + record['name'] + '$').splitlines()
    if not matches:
        intent_path.unlink()
        return False
    if len(matches) != 1:
        fail('Runtime creation intent has ambiguous Docker objects.')
    record['container_id'] = matches[0]
    try:
        proof = inspect(state, record['container_id'])
    except DockerNotFound:
        intent_path.unlink()
        return False
    # A stable name collision cannot authorise removal of the foreign object.
    if proof.get('Config', {}).get('Labels', {}).get(PREFIX + 'instance') != record.get('instance'):
        intent_path.unlink()
        fail('Runtime container name belongs to another invocation.')
    verify_configuration(state, record, proof)
    existing = read_record(state)
    if existing is not None:
        if existing.get('container_id') != record['container_id'] or existing.get('instance') != record['instance']:
            fail('Runtime intent conflicts with another reservation.')
        verify_ownership(Path(state), existing)
        intent_path.unlink()
        return False
    _remove_started(Path(state), record)
    intent_path.unlink()
    return True


def _remove_started(state, record):
    proof = inspect(state, record['container_id'])
    verify_configuration(state, record, proof)
    if proof.get('State', {}).get('Running'):
        if 'pid' not in record:
            identity = process_identity(proof['State'].get('Pid'))
            if not identity or identity['exe'] != record['binary'] or identity['uid'] != os.getuid():
                fail('Startup cleanup cannot verify the owned process.')
            record = dict(record, **identity)
        _terminate(state, record)
    docker(state, 'rm', record['container_id'])


def _terminate(state, record):
    verify_ownership(state, record)
    descriptor = os.pidfd_open(record['pid'])
    try:
        verify_ownership(state, record)
        signal.pidfd_send_signal(descriptor, signal.SIGTERM)
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            if process_identity(record['pid']) is None:
                return
            time.sleep(0.1)
        fail('The owned runtime did not stop; it has not been forcibly killed.')
    finally:
        os.close(descriptor)


def stop(state):
    state = Path(state)
    record = read_record(state)
    if record is None:
        return recover_intent(state)
    verify_record_boundary(state, record)
    try:
        proof = inspect(state, record['container_id'])
    except DockerNotFound:
        archive_reservation(state, record)
        return True
    verify_configuration(state, record, proof)
    if proof.get('State', {}).get('Running'):
        _terminate(state, record)
    else:
        # A stopped container can be removed by immutable ID only after replaying
        # its source/mount proof. No saved PID is ever signalled in this branch.
        expected = approved_plan(state, Path(record['source']), Path(record['binary']), record['manifest'], record['projection'])
        if record['mounts'] != expected:
            fail('Stopped container mount plan changed.')
    docker(state, 'rm', record['container_id'])
    archive_reservation(state, record)
    return True


def archive_reservation(state, record):
    """Retain the own receipt after removal without ever touching its saved PID."""
    verify_record_boundary(state, record)
    if not re.fullmatch(r'[a-f0-9]{64}', record.get('container_id', '')):
        fail('Runtime receipt has no immutable container ID.')
    if read_record(state) != record:
        fail('Container reservation changed while stopping.')
    path = state / ('runtime-container-stopped-' + record['container_id'] + '-' + record['instance'] + '.json')
    existing = read_private_json(path)
    if existing is None:
        private_json(path, record)
    elif existing != record:
        fail('Stopped container receipt conflicts with existing history.')
    if read_record(state) != record:
        fail('Container reservation changed before archiving.')
    (state / 'runtime-container.json').unlink()
