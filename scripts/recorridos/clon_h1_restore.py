#!/usr/bin/env python3
"""Restore only the externally pinned synthetic H1 into a newly reserved clone.

Never adopt a database from a supplied receipt. Failures preserve the private
pending record and owned volume/container for examination. No automatic cleanup
or retry is provided. A clean pg_control is technical cold-snapshot evidence;
the source README alone does not establish supported hot-backup provenance.
"""

import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import stat
import subprocess
import tarfile
import tempfile
import time

H1_SHA = 'd1c2e38a85f872e83b6ba9eca6b660b7a5d2ff5d39d4aca612a81ec30a496e5b'
IMAGE = 'postgres:18.4'
IMAGE_ID = 'sha256:1bf3d6960db467e87a506daef30feb41fecc23b7c5f96b157e873059f2ffb50a'
MAX_ARCHIVE = 2 * 1024**3
MAX_FILE = 1024**3
MAX_TOTAL = 4 * 1024**3
MAX_ENTRIES = 100_000
OWNER = 'Codex-M'
ROOT = '18/docker'
VOLUME_PARENT = Path('/dev/shm')
RECEIPT = 'h1-restore.json'
PG_COMMAND = ['-D', '/var/lib/postgresql/18/docker',
              '-c', 'listen_addresses=', '-c', 'unix_socket_directories=/var/run/postgresql',
              '-c', 'logging_collector=off', '-c', 'log_statement=none',
              '-c', 'archive_mode=off', '-c', 'shared_preload_libraries=']
BLOCKED_KEYS = {'include', 'include_dir', 'include_if_exists', 'archive_command',
                'archive_library', 'restore_command', 'recovery_end_command',
                'archive_cleanup_command', 'primary_conninfo', 'primary_slot_name',
                'shared_preload_libraries', 'session_preload_libraries',
                'local_preload_libraries', 'ssl', 'ssl_passphrase_command',
                'data_directory', 'hba_file', 'ident_file', 'external_pid_file'}


class RestoreError(Exception):
    pass


def fail(message):
    raise RestoreError(message)


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), ensure_ascii=True).encode() + b'\n'


def sha_stream(stream):
    stream.seek(0)
    digest = hashlib.sha256()
    for chunk in iter(lambda: stream.read(1024 * 1024), b''):
        digest.update(chunk)
    stream.seek(0)
    return digest.hexdigest()


def file_identity(info):
    return (info.st_dev, info.st_ino, info.st_mode, info.st_uid, info.st_gid,
            info.st_size, info.st_mtime_ns, info.st_ctime_ns, info.st_nlink)


def trusted(path, directory=False):
    path = Path(path)
    if not path.is_absolute() or '..' in path.parts or ',' in str(path):
        fail('Unsafe private path.')
    for parent in reversed(path.parents):
        info = parent.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid not in (0, os.getuid()) or info.st_mode & 0o022:
            fail('Private path has an untrusted ancestor.')
    info = path.lstat()
    if info.st_uid != os.getuid() or info.st_mode & 0o077:
        fail('Private path owner or permissions differ.')
    if directory:
        if not stat.S_ISDIR(info.st_mode):
            fail('Private state is not a directory.')
    elif not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
        fail('Input is not a private single-link regular file.')
    return path


def publish(state, name, value):
    """Publish canonical complete bytes exclusively and fsync both directory changes."""
    fd, tmp = tempfile.mkstemp(prefix='.' + name + '-', dir=state)
    try:
        with os.fdopen(fd, 'wb') as stream:
            stream.write(canonical(value))
            stream.flush()
            os.fsync(stream.fileno())
        os.link(tmp, state / name, follow_symlinks=False)
        syncdir(state)
    finally:
        os.unlink(tmp)
        syncdir(state)


def syncdir(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def normalized_name(raw):
    if not isinstance(raw, str) or len(raw) > 1024 or '\\' in raw or '\x00' in raw:
        fail('Invalid archive path.')
    if raw == '.':
        return '.'
    if raw.startswith('./'):
        raw = raw[2:]
    raw = raw.removesuffix('/')
    parts = raw.split('/')
    if any(p in ('', '.', '..') for p in parts) or raw.startswith('/'):
        fail('Archive path escapes its root.')
    if raw not in ('18', ROOT) and not raw.startswith(ROOT + '/'):
        fail('Archive entry is outside PG18/docker.')
    return raw


def validate_archive(source):
    """Scan every logical member before extraction, including all config contents."""
    seen, entries, total = set(), [], 0
    required = {'.', '18', ROOT, ROOT + '/PG_VERSION', ROOT + '/global/pg_control'}
    with tarfile.open(fileobj=source, mode='r:gz') as archive:
        for member in archive:
            name = normalized_name(member.name)
            if name in seen or len(entries) >= MAX_ENTRIES:
                fail('Duplicate or excessive archive entries.')
            seen.add(name)
            if not (member.isdir() or member.isreg()) or member.sparse is not None or member.pax_headers or member.linkname:
                fail('Links, special files, sparse files and extended metadata are forbidden.')
            if member.size < 0 or member.size > MAX_FILE or (member.isdir() and member.size):
                fail('Archive member size is invalid.')
            total += member.size
            if total > MAX_TOTAL:
                fail('Archive uncompressed size exceeds its bound.')
            if name in ('.', '18'):
                if not member.isdir() or member.mode & ~0o777 or member.mode & 0o002:
                    fail('Invalid ancestor directory metadata.')
            elif (member.uid != 999 or member.gid not in (0, 999)
                  or member.mode != (0o700 if member.isdir() else 0o600)):
                fail('PGDATA ownership or private modes differ.')
            if name in {ROOT + '/' + n for n in ('postmaster.pid', 'backup_label', 'backup_manifest', 'standby.signal', 'recovery.signal')}:
                fail('Snapshot requires an independently verified backup/recovery circuit.')
            if name == ROOT + '/PG_VERSION':
                if member.isdir() or member.size > 8 or archive.extractfile(member).read() != b'18\n':
                    fail('PG_VERSION is not 18.')
            if name == ROOT + '/global/pg_control' and (member.isdir() or member.size != 8192):
                fail('Invalid PostgreSQL control file.')
            if name.endswith(('/postgresql.conf', '/postgresql.auto.conf')):
                if member.isdir() or member.size > 1024 * 1024:
                    fail('Configuration size or type differs.')
                config = archive.extractfile(member).read().decode('utf-8')
                for line in config.splitlines():
                    line = line.strip()
                    if not line or line.startswith('#'):
                        continue
                    key = re.split(r'[=\s]', line, maxsplit=1)[0].lower()
                    if key in BLOCKED_KEYS:
                        fail('Snapshot references external configuration or executable providers.')
            entries.append((name, member))
        if not required <= seen:
            fail('Snapshot lacks required PG18 structure.')
        kinds = {name: m.isdir() for name, m in entries}
        for name, member in entries:
            if name == '.':
                continue
            parent = name.rsplit('/', 1)[0] if '/' in name else '.'
            if kinds.get(parent) is not True:
                fail('Archive parent directory is missing or is a file.')
    source.seek(0)
    return entries, total


def normalize_archive(source, destination, entries):
    """Reconstruct plain entries; omit the unsafe original volume-root mode."""
    with tarfile.open(fileobj=source, mode='r:gz') as original, tarfile.open(fileobj=destination, mode='w', format=tarfile.GNU_FORMAT) as output:
        for name, member in entries:
            if name == '.':
                continue
            clean = tarfile.TarInfo(name)
            clean.type = tarfile.DIRTYPE if member.isdir() else tarfile.REGTYPE
            clean.mode = 0o700 if member.isdir() else 0o600
            clean.uid = clean.gid = 999
            clean.mtime = 0
            clean.size = member.size
            output.addfile(clean, None if member.isdir() else original.extractfile(member))
    destination.flush()
    os.fsync(destination.fileno())
    destination.seek(0)


class _Docker:
    def __init__(self, state):
        self.config = state / 'h1-docker-client'
        self.config.mkdir(mode=0o700)
        syncdir(state)

    def run(self, *args, stdin=None, timeout=180):
        try:
            result = subprocess.run(['/usr/bin/docker', '--config', str(self.config),
                                     '--host', 'unix:///var/run/docker.sock', *args],
                                    env={'PATH': '/usr/bin:/bin', 'HOME': str(self.config)},
                                    stdin=stdin if stdin is not None else subprocess.DEVNULL,
                                    capture_output=True, timeout=timeout)
        except (OSError, subprocess.TimeoutExpired):
            fail('Local Docker operation unavailable or timed out; preserve pending state.')
        if result.returncode or len(result.stdout) > 1024 * 1024:
            fail('Local Docker operation failed; output is withheld.')
        return result.stdout.decode('utf-8').strip()


def image_check(docker):
    for reference in (IMAGE, IMAGE_ID):
        value = json.loads(docker.run('image', 'inspect', reference))
        if len(value) != 1 or value[0].get('Id') != IMAGE_ID:
            fail('Local PG18.4 image differs from its approved ID.')
    identity = docker.run('run', '--rm', '--pull', 'never', '--network', 'none', '--read-only',
                          '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges',
                          '--log-driver', 'none', '--pids-limit', '32', '--memory', '64m', '--cpus', '1',
                          '--entrypoint', '/usr/bin/id', IMAGE_ID, '-u', 'postgres')
    group = docker.run('run', '--rm', '--pull', 'never', '--network', 'none', '--read-only',
                       '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges',
                       '--log-driver', 'none', '--pids-limit', '32', '--memory', '64m', '--cpus', '1',
                       '--entrypoint', '/usr/bin/id', IMAGE_ID, '-g', 'postgres')
    if identity != '999' or group != '999':
        fail('Pinned image PostgreSQL UID differs.')


def volume_identity(volume, identity):
    info = volume.lstat()
    if not stat.S_ISDIR(info.st_mode) or (info.st_dev, info.st_ino) != identity:
        fail('Owned restore volume was replaced.')


def container_check(docker, container, state, volume, run_id):
    value = json.loads(docker.run('container', 'inspect', container))
    if len(value) != 1:
        fail('Container inspection differs.')
    obj = value[0]
    labels = obj.get('Config', {}).get('Labels', {})
    host = obj.get('HostConfig', {})
    mounts = obj.get('Mounts', [])
    binds = [m for m in mounts if m.get('Type') == 'bind']
    if (obj.get('Id') != container or obj.get('Image') != IMAGE_ID
        or not obj.get('State', {}).get('Running')
        or labels.get('vec.clon.owner') != OWNER or labels.get('vec.clon.state') != str(state)
        or labels.get('vec.clon.h1.run') != run_id or host.get('NetworkMode') != 'none'
        or host.get('PortBindings') or any(v for v in (obj.get('NetworkSettings', {}).get('Ports') or {}).values())
        or host.get('ReadonlyRootfs') is not True or host.get('Privileged') is not False
        or host.get('CapDrop') != ['ALL'] or host.get('CapAdd')
        or host.get('LogConfig', {}).get('Type') != 'none'
        or obj.get('Config', {}).get('Cmd') != PG_COMMAND
        or len(binds) != 1 or binds[0].get('Source') != str(volume)
        or binds[0].get('Destination') != '/var/lib/postgresql' or binds[0].get('RW') is not True
        or any(m.get('Type') not in ('bind', 'tmpfs') for m in mounts)
        or obj.get('Config', {}).get('User') != '999:999'
        or obj.get('Config', {}).get('Entrypoint') != ['postgres']):
        fail('Owned PostgreSQL container boundary differs.')
    return obj


def _probe(container, state, normalizer, normalizer_sha):
    import clon_sql
    ro = clon_sql.ReadOnlyDB(clon_sql.DockerDB(container, state=state, expected_image_id=IMAGE_ID))
    identity = ro.system_identity()
    identity.update(schema_sha=ro.schema_digest(normalizer, normalizer_sha),
                    roles_sha=ro.roles_digest(normalizer, normalizer_sha),
                    datacl_sha=ro.database_acl_digest())
    return identity


def restore(state, h1, normalizer_path, normalizer_sha):
    """Reserve, restore, prove cold-control state, start and capture a fresh preimage.

    `state` must exist private 0700 with trusted ancestors; no restore markers
    may exist. The receipt API matches H6Kit. Metadata lives separately.
    """
    state, h1 = trusted(state, True), trusted(h1)
    if stat.S_IMODE(state.stat().st_mode) != 0o700:
        fail('Restore state requires mode 0700.')
    for name in ('h1-restore-pending.json', RECEIPT, 'h1-restore-evidence.json', 'h1-docker-client', 'h1-volume.json', 'h1-container.json'):
        if os.path.lexists(state / name):
            fail('Prior restore state exists; no adoption or automatic retry.')
    fd = os.open(h1, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, 'rb') as source:
        initial = os.fstat(source.fileno())
        if initial.st_size > MAX_ARCHIVE or initial.st_nlink != 1 or not stat.S_ISREG(initial.st_mode):
            fail('H1 archive metadata differs.')
        if sha_stream(source) != H1_SHA:
            fail('H1 archive SHA differs from external approval.')
        entries, total = validate_archive(source)
        docker = _Docker(state)
        image_check(docker)
        run_id = secrets.token_hex(16)
        publish(state, 'h1-restore-pending.json', {'version': 1, 'kind': 'h1_restore_pending',
                    'run_id': run_id, 'estado_h1_sha': H1_SHA, 'pg_image_id': IMAGE_ID})
        volume = Path(tempfile.mkdtemp(prefix='vec-recorridos-', dir=VOLUME_PARENT))
        info = volume.lstat()
        if (volume.parent != VOLUME_PARENT or not re.fullmatch(r'vec-recorridos-[a-zA-Z0-9_-]+', volume.name)
            or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700
            or not stat.S_ISDIR(info.st_mode) or any(volume.iterdir())):
            fail('Exclusive fresh restore volume boundary differs.')
        identity = (info.st_dev, info.st_ino)
        publish(state, 'h1-volume.json', {'run_id': run_id, 'path': str(volume),
                'device': info.st_dev, 'inode': info.st_ino})
        if os.statvfs(volume).f_bavail * os.statvfs(volume).f_frsize < total + 256 * 1024**2:
            fail('Owned volume has insufficient free space.')
        with tempfile.TemporaryFile(dir=state) as normalized:
            normalize_archive(source, normalized, entries)
            if sha_stream(source) != H1_SHA or file_identity(os.fstat(source.fileno())) != file_identity(initial) or file_identity(h1.lstat()) != file_identity(initial):
                fail('H1 changed while preparing normalized input.')
            volume_identity(volume, identity)
            docker.run('run', '--rm', '-i', '--pull', 'never', '--network', 'none', '--read-only',
                       '--user', '0:0', '--log-driver', 'none', '--cap-drop', 'ALL', '--cap-add', 'CHOWN', '--cap-add', 'DAC_OVERRIDE',
                       '--cap-add', 'FOWNER', '--security-opt', 'no-new-privileges',
                       '--pids-limit', '64', '--memory', '256m', '--cpus', '1',
                       '--mount', f'type=bind,source={h1},destination=/h1,readonly',
                       '--mount', f'type=bind,source={volume},destination=/restore',
                       '--entrypoint', '/bin/sh', IMAGE_ID, '-ec',
                       'chown 999:999 /restore; chmod 700 /restore; exec tar --extract --file=- --directory=/restore --numeric-owner --same-owner --same-permissions',
                       stdin=normalized)
        volume_identity(volume, identity)
        control = docker.run('run', '--rm', '--pull', 'never', '--network', 'none', '--read-only',
                             '--user', '999:999', '--cap-drop', 'ALL', '--log-driver', 'none',
                             '--pids-limit', '32', '--memory', '128m', '--cpus', '1',
                             '--env', 'LC_ALL=C', '--security-opt', 'no-new-privileges',
                             '--mount', f'type=bind,source={volume},destination=/var/lib/postgresql,readonly',
                             '--entrypoint', 'pg_controldata', IMAGE_ID, '/var/lib/postgresql/18/docker')
        state_match = re.search(r'^Database cluster state:\s*(.+)$', control, re.M)
        system_match = re.search(r'^Database system identifier:\s*(\d+)$', control, re.M)
        if not state_match or state_match[1].strip() != 'shut down' or not system_match:
            fail('NO-GO: physical snapshot has no clean shutdown control evidence.')
        control_sha = hashlib.sha256(control.encode()).hexdigest()
        with tarfile.open(fileobj=source, mode='r:gz') as archive:
            control_member = next(m for name, m in entries if name == ROOT + '/global/pg_control')
            physical_control_sha = hashlib.sha256(archive.extractfile(control_member).read()).hexdigest()
        source.seek(0)
        container = docker.run('container', 'create', '--pull', 'never', '--network', 'none',
                     '--name', 'vec-h1-' + run_id, '--user', '999:999', '--read-only',
                     '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges',
                     '--pids-limit', '128', '--memory', '1g', '--cpus', '2',
                     '--log-driver', 'none', '--label', 'vec.clon.owner=' + OWNER,
                     '--label', 'vec.clon.state=' + str(state), '--label', 'vec.clon.h1.run=' + run_id,
                     '--mount', f'type=bind,source={volume},destination=/var/lib/postgresql',
                     '--tmpfs', '/var/run/postgresql:rw,noexec,nosuid,size=8m,uid=999,gid=999,mode=0700',
                     '--tmpfs', '/tmp:rw,noexec,nosuid,size=16m,uid=999,gid=999,mode=0700',
                     '--entrypoint', 'postgres', IMAGE_ID, *PG_COMMAND)
        if not re.fullmatch('[a-f0-9]{64}', container):
            fail('Docker did not return an immutable PostgreSQL ID.')
        publish(state, 'h1-container.json', {'run_id': run_id, 'pg_container_id': container})
        docker.run('container', 'start', container)
        for attempt in range(30):
            volume_identity(volume, identity)
            container_check(docker, container, state, volume, run_id)
            try:
                docker.run('exec', '--user', '999:999', container, 'pg_isready',
                           '-h', '/var/run/postgresql', '-p', '5432', '-U', 'postgres', '-d', 'postgres', timeout=10)
                break
            except RestoreError:
                if attempt == 29:
                    fail('PostgreSQL readiness timed out; preserve owned evidence.')
                time.sleep(1)
        observed = _probe(container, state, normalizer_path, normalizer_sha)
        expected = {'system_identifier': system_match[1], 'database_name': 'postgres',
                    'pg_container_id': container, 'pg_image': IMAGE, 'pg_image_id': IMAGE_ID, 'pg_volume': str(volume)}
        if any(observed.get(k) != v for k, v in expected.items()) or type(observed.get('database_oid')) is not int:
            fail('Fresh restored database identity differs.')
        if any(not re.fullmatch('[a-f0-9]{64}', observed.get(k, '')) for k in ('schema_sha', 'roles_sha', 'datacl_sha')):
            fail('Restored database fingerprints are invalid.')
        container_check(docker, container, state, volume, run_id)
        volume_identity(volume, identity)
        if sha_stream(source) != H1_SHA or file_identity(h1.lstat()) != file_identity(initial):
            fail('H1 input changed before confirmation.')
        receipt = {'version': 1, 'kind': 'h1_restore_confirmed', 'estado_h1_sha': H1_SHA,
                   **expected, 'database_oid': observed['database_oid'],
                   **{k: observed[k] for k in ('schema_sha', 'roles_sha', 'datacl_sha')}}
        publish(state, 'h1-restore-evidence.json', {'version': 1, 'run_id': run_id,
            'pg_image_id': IMAGE_ID, 'volume_device': identity[0], 'volume_inode': identity[1],
            'restore_exit0': True, 'pg_version': '18', 'control_state': 'shut down',
            'control_output_sha256': control_sha, 'pg_control_sha256': physical_control_sha, 'provenance': 'clean_shutdown_control_before_start',
            'provenance_limit': 'README alone does not attest snapshot capture protocol',
            'receipt_sha256': hashlib.sha256(canonical(receipt)).hexdigest()})
        publish(state, RECEIPT, receipt)
        return receipt
