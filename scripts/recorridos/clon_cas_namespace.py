"""One-shot nominal RO session attestation in the approved PG network namespace.

No CAS, arbitrary query/callback, host PostgreSQL socket, relay, provisioning or
configuration is exposed. Missing nominal certificate HBA or TLS is a blocker.
The external request pins original RESTORE, SQL62 journal, physical PG identity,
H1 records and helper image/code. A failed attempt is preserved, never replayed.
Live use requires the two independent sensitive reviews of this exact source.
Sessions close before receipt publication. This receipt cannot bind subsequent
DB-API connections passed to medir(); cas_sessions_bound remains false.
"""
from __future__ import annotations

from contextlib import ExitStack
from dataclasses import dataclass
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import stat

try:
    from . import clon_sql, clon_relay_host as relay, clon_h6_sql_instalar as installer
except ImportError:
    import clon_sql
    import clon_relay_host as relay
    import clon_h6_sql_instalar as installer

CHANNELS = ('identidad', 'contexto', 'autorizacion', 'motivos')
ATTEMPT = 'cas-namespace-attempt.json'
RECEIPT = 'cas-namespace-sessions.json'
BOOT = 'import time; time.sleep(120)'
# This fixed program lives in the approved image's Python/psycopg environment.
# Its stdin contains only typed connection metadata and expected identities.
READER = r'''
import hashlib,json,sys
from contextlib import ExitStack
from pathlib import Path
import psycopg
SQL="""SELECT session_user::text,current_user::text,
 current_setting('role'),current_setting('transaction_read_only'),
 pg_backend_pid(),current_database(),
 (SELECT oid::bigint FROM pg_catalog.pg_database WHERE datname=current_database()),
 (SELECT system_identifier::text FROM pg_catalog.pg_control_system()),
 inet_client_addr()::text,inet_server_addr()::text,inet_server_port(),
 (SELECT rolcanlogin AND NOT (rolsuper OR rolcreatedb OR rolcreaterole
 OR rolreplication OR rolbypassrls) FROM pg_catalog.pg_roles WHERE rolname=session_user),
 s.ssl,s.version,s.client_dn IS NOT NULL FROM pg_catalog.pg_stat_ssl s
 WHERE s.pid=pg_backend_pid()"""
try:
    raw=sys.stdin.buffer.read(16385)
    if len(raw)>16384: raise ValueError()
    p=json.loads(raw)
    files={}
    for name,expected in p['tls_hashes'].items():
        data=Path('/tls/'+name).read_bytes()
        if not 0<len(data)<=65536: raise ValueError()
        files[name]=hashlib.sha256(data).hexdigest()
        if files[name]!=expected: raise ValueError()
    with ExitStack() as stack:
        cursors={}; initial={}
        for channel,user in p['users'].items():
            c=stack.enter_context(psycopg.connect(host=p['server_name'],hostaddr='127.0.0.1',
                port=5432,dbname='postgres',user=user,sslmode='verify-full',
                sslrootcert='/tls/ca.pem',sslcert='/tls/'+channel+'.crt',
                sslkey='/tls/'+channel+'.key',connect_timeout=5,autocommit=True,
                options='-c default_transaction_read_only=on -c statement_timeout=5000 -c lock_timeout=1000',
                application_name='vec-clon-nominal-ro'))
            stack.callback(c.rollback)
            cur=stack.enter_context(c.cursor())
            cur.execute('BEGIN READ ONLY')
            cursors[channel]=cur
            cur.execute(SQL); initial[channel]=cur.fetchall()
        for channel,cur in cursors.items():
            cur.execute(SQL)
            if cur.fetchall()!=initial[channel]: raise ValueError()
        # Roll back explicitly before the parent can accept this observation.
        for cur in cursors.values(): cur.execute('ROLLBACK')
    print(json.dumps({'version':1,'nonce':p['nonce'],'tls_hashes':files,
        'sessions':initial},sort_keys=True,separators=(',',':')))
except Exception:
    sys.exit(1)
'''
HELPER_SHA256 = hashlib.sha256((BOOT + '\n' + READER).encode()).hexdigest()
PREFLIGHT_SQL = """SELECT pg_catalog.jsonb_build_object(
 'ssl',current_setting('ssl'),
 'configuration_loaded',pg_catalog.pg_conf_load_time()::text,
 'hba_loaded',(SELECT bool_and(file_name=current_setting('hba_file') AND
 (pg_catalog.pg_stat_file(file_name)).modification <
 pg_catalog.date_trunc('second',pg_catalog.pg_conf_load_time()))
 FROM pg_catalog.pg_hba_file_rules),
 'system_identifier',(SELECT system_identifier::text FROM pg_catalog.pg_control_system()),
 'database_oid',(SELECT oid::bigint FROM pg_catalog.pg_database WHERE datname='postgres'),
 'hba',(SELECT jsonb_agg(jsonb_build_object('type',type,'database',database,
 'user_name',user_name,'address',address,'netmask',netmask,'auth_method',auth_method,
 'options',options,'error',error) ORDER BY rule_number)
 FROM pg_catalog.pg_hba_file_rules))::text"""


class Refused(RuntimeError):
    """Fixed error codes only; never provider errors or private inputs."""


def require(ok: bool, code: str) -> None:
    if not ok:
        raise Refused(code)


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def decode(raw: bytes):
    require(len(raw) <= clon_sql.MAX_JOURNAL, 'namespace_output_limit')
    return relay.decode(raw)


def publish(directory: int, name: str, value: dict) -> None:
    """Exclusive complete publication, with directory durability."""
    temporary = '.cas-' + secrets.token_hex(16)
    fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL |
                 os.O_NOFOLLOW | os.O_CLOEXEC, 0o600, dir_fd=directory)
    try:
        with os.fdopen(fd, 'wb') as stream:
            stream.write(installer.clon_h6_kit.canonical(value))
            stream.flush()
            os.fsync(stream.fileno())
        os.link(temporary, name, src_dir_fd=directory, dst_dir_fd=directory, follow_symlinks=False)
        os.fsync(directory)
    finally:
        os.unlink(temporary, dir_fd=directory)


@dataclass(frozen=True)
class PGID:
    container_id: str
    image_id: str
    system_identifier: str
    database_oid: int
    volume: Path
    volume_device: int
    volume_inode: int


@dataclass(frozen=True)
class Nominal:
    channel: str
    user: str
    cert_sha256: str
    key_sha256: str


@dataclass(frozen=True)
class Request:
    state: Path
    pg: PGID
    restore_sha256: str
    journal_sha256: str
    h1_records_sha256: str
    helper_image_id: str
    helper_image_digest: str
    helper_sha256: str
    tls_directory: Path
    ca_sha256: str
    server_name: str
    nominal: tuple[Nominal, ...]

    def validate(self) -> None:
        require(type(self.pg) is PGID and type(self.nominal) is tuple
                and all(type(n) is Nominal for n in self.nominal), 'namespace_typed_request')
        hashes = (self.pg.container_id, self.restore_sha256, self.journal_sha256,
                  self.h1_records_sha256, self.helper_sha256, self.ca_sha256,
                  *(v for n in self.nominal for v in (n.cert_sha256, n.key_sha256)))
        require(all(isinstance(v, str) and relay.HEX64.fullmatch(v) for v in hashes)
                and all(isinstance(v, str) and relay.IMAGE.fullmatch(v)
                        for v in (self.pg.image_id, self.helper_image_id)), 'namespace_pin_invalid')
        require(re.fullmatch(r'[a-z0-9][a-z0-9./_-]{0,180}@sha256:[a-f0-9]{64}',
                             self.helper_image_digest or '') is not None
                and self.helper_sha256 == HELPER_SHA256, 'namespace_helper_pin_invalid')
        require(len(self.nominal) == 4 and tuple(n.channel for n in self.nominal) == CHANNELS
                and len({n.user for n in self.nominal}) == 4
                and all(re.fullmatch(r'vec_[a-z0-9_]{1,100}', n.user or '') for n in self.nominal),
                'namespace_nominal_users_invalid')
        require(re.fullmatch(r'[a-z][a-z0-9.-]{0,120}', self.server_name or '') is not None
                and re.fullmatch(r'[1-9][0-9]{0,19}', self.pg.system_identifier or '') is not None
                and int(self.pg.system_identifier) < 2**64
                and type(self.pg.database_oid) is int and 0 < self.pg.database_oid < 2**32
                and type(self.pg.volume_device) is int and self.pg.volume_device >= 0
                and type(self.pg.volume_inode) is int and self.pg.volume_inode > 0,
                'namespace_pgid_invalid')
        for path in (self.state, self.tls_directory, self.pg.volume):
            require(type(path) is type(Path()) and path.is_absolute() and '..' not in path.parts
                    and not any(c in str(path) for c in ',\n\r'), 'namespace_path_invalid')
        require(re.fullmatch(r'/dev/shm/vec-recorridos-[A-Za-z0-9_-]+', str(self.pg.volume)) is not None
                and self.state != self.tls_directory and not self.tls_directory.is_relative_to(self.state),
                'namespace_private_material_separate')
        require(os.getuid() > 0, 'namespace_nonroot_required')


def tls_hashes(request: Request) -> dict[str, str]:
    return {'ca.pem': request.ca_sha256, **{n.channel + ext: value
            for n in request.nominal for ext, value in
            (('.crt', n.cert_sha256), ('.key', n.key_sha256))}}


def private_inputs(request: Request, directory: int) -> dict[str, tuple]:
    """Re-read pins, H1 ownership records and exact bytes; do not adopt state."""
    raw = {name: installer.read_owned(directory, name, clon_sql.MAX_JOURNAL)
           for name in ('h1-restore.json', 'sql-journal.json', 'h1-volume.json',
                        'h1-container.json', 'h1-restore-pending.json')}
    require(digest(raw['h1-restore.json']) == request.restore_sha256
            and digest(raw['sql-journal.json']) == request.journal_sha256, 'namespace_private_pin_drift')
    records = {name: digest(raw[name]) for name in
               ('h1-restore.json', 'h1-restore-pending.json', 'h1-volume.json', 'h1-container.json')}
    require(digest(installer.clon_h6_kit.canonical(records)) == request.h1_records_sha256,
            'namespace_h1_records_drift')
    value = {name: decode(data) for name, data in raw.items()}
    restore, journal = value['h1-restore.json'], value['sql-journal.json']
    pg = request.pg
    require(raw['h1-restore.json'] == installer.clon_h6_kit.canonical(restore)
            and set(restore) == installer.clon_h6_kit.RESTORE_FIELDS
            and type(restore['version']) is int and restore['version'] == 1
            and restore['kind'] == 'h1_restore_confirmed'
            and all(restore[k] == v for k, v in {
                'pg_container_id': pg.container_id, 'pg_image_id': pg.image_id,
                'system_identifier': pg.system_identifier, 'database_oid': pg.database_oid,
                'database_name': 'postgres', 'pg_image': 'postgres:18.4',
                'pg_volume': str(pg.volume)}.items()), 'namespace_restore_pgid_mismatch')
    pending, volume, container = (value[n] for n in
                ('h1-restore-pending.json', 'h1-volume.json', 'h1-container.json'))
    run = pending.get('run_id')
    require(isinstance(run, str) and re.fullmatch(r'[a-f0-9]{32}', run) is not None
            and pending == {'version': 1, 'kind': 'h1_restore_pending', 'run_id': run,
                            'estado_h1_sha': restore['estado_h1_sha'], 'pg_image_id': pg.image_id}
            and volume == {'run_id': run, 'path': str(pg.volume),
                           'device': pg.volume_device, 'inode': pg.volume_inode}
            and container == {'run_id': run, 'pg_container_id': pg.container_id},
            'namespace_h1_ownership_mismatch')
    plan = {k: journal[k] for k in ('plan_family', 'lock_sha', 'plan_sha', 'inventory_sha',
                                  'entries', 'file_count', 'approved_sql_ref')}
    plan['source_ref'] = journal['source_commit']
    require(journal.get('journal_sha') == clon_sql.record_hash(journal)
            and plan['plan_family'] == clon_sql.H6_PACKAGE_FAMILY
            and plan['source_ref'] == plan['approved_sql_ref'] == installer.SOURCE
            and plan['file_count'] == 62 and len(plan['entries']) == 62
            and clon_sql.plan_hash(plan['entries']) == plan['plan_sha']
            and journal.get('phase') == 'awaiting_ad132'
            and journal.get('identidad_clon') == request.restore_sha256,
            'namespace_sql62_required')
    clon_sql.validate_record(journal, plan, clon_sql.context_from_record(journal))
    clon_sql.validate_receipts(journal['installed'], plan, complete=True)
    snapshots = {}
    with installer.private_state(request.tls_directory) as (tls, _):
        require(set(os.listdir(tls)) == set(tls_hashes(request)), 'namespace_tls_mount_minimal')
        for name, expected in tls_hashes(request).items():
            data = installer.read_owned(tls, name, 65536)
            require(digest(data) == expected, 'namespace_tls_changed')
            snapshots[name] = installer.identity(os.stat(name, dir_fd=tls, follow_symlinks=False))
    return snapshots


def process_identity(pid: int) -> tuple[int, int, int]:
    """Linux process start tick and physical network namespace inode/device."""
    require(type(pid) is int and pid > 0, 'namespace_pid_invalid')
    raw = Path(f'/proc/{pid}/stat').read_text()
    start = int(raw[raw.rfind(')') + 2:].split()[19])
    namespace = os.stat(f'/proc/{pid}/ns/net')
    after = Path(f'/proc/{pid}/stat').read_text()
    require(start > 0 and start == int(after[after.rfind(')') + 2:].split()[19]),
            'namespace_process_missing')
    return start, namespace.st_dev, namespace.st_ino


class Docker:
    """Concrete local Docker only. No caller-supplied executor or shell."""
    def __init__(self, config: Path):
        self.config = config
        self.context = relay.directory(config)
        self.fd = self.context.__enter__()

    def close(self) -> None:
        self.context.__exit__(None, None, None)

    def command(self, *args: str, data: bytes | None = None, limit: int = 65536) -> bytes:
        with relay.directory(self.config) as fd:
            now, retained = os.fstat(fd), os.fstat(self.fd)
            require((now.st_dev, now.st_ino) == (retained.st_dev, retained.st_ino)
                    and now.st_uid == os.getuid() and stat.S_IMODE(now.st_mode) == 0o700
                    and not os.listdir(fd), 'namespace_docker_config_drift')
        # Existing bounded selector runner: no shell, finite output and time.
        return clon_sql._probe_command(['/usr/bin/docker', '--config', str(self.config),
                    '--host', 'unix:///var/run/docker.sock', *args], data, limit=limit, timeout=30)

    def inspect(self, cid: str) -> dict:
        require(relay.HEX64.fullmatch(cid) is not None, 'namespace_cid_invalid')
        # Never capture Config.Env, DSN, secrets or container logs.
        fields = ('NetworkMode', 'Privileged', 'ReadonlyRootfs', 'CapDrop', 'CapAdd',
                  'SecurityOpt', 'Devices', 'DeviceRequests', 'PidMode', 'IpcMode',
                  'PortBindings', 'PidsLimit', 'Memory', 'NanoCpus')
        host = ','.join('"' + f + '":{{json .HostConfig.' + f + '}}' for f in fields)
        projection = ('{"Id":{{json .Id}},"Image":{{json .Image}},'
            '"Config":{"Image":{{json .Config.Image}},"Labels":{{json .Config.Labels}},'
            '"User":{{json .Config.User}},"Entrypoint":{{json .Config.Entrypoint}},'
            '"Cmd":{{json .Config.Cmd}}},"HostConfig":{' + host +
            ',"LogConfig":{"Type":{{json .HostConfig.LogConfig.Type}}}},'
            '"State":{"Running":{{json .State.Running}},"Pid":{{json .State.Pid}},'
            '"StartedAt":{{json .State.StartedAt}}},"Mounts":{{json .Mounts}},'
            '"NetworkSettings":{"Ports":{{json .NetworkSettings.Ports}}}}')
        value = decode(self.command('container', 'inspect', '--format', projection, cid))
        require(isinstance(value, dict) and value.get('Id') == cid,
                'namespace_container_missing')
        return value

    def image(self, request: Request) -> None:
        projection = ('{"Id":{{json .Id}},"RepoDigests":{{json .RepoDigests}},'
                      '"Volumes":{{json .Config.Volumes}}}')
        value = decode(self.command('image', 'inspect', '--format', projection, request.helper_image_digest))
        require(value.get('Id') == request.helper_image_id
                and request.helper_image_digest in (value.get('RepoDigests') or [])
                and not value.get('Volumes'), 'namespace_helper_image_drift')

    def preflight(self, request: Request) -> str:
        pg = request.pg
        raw = self.command('exec', '-i', pg.container_id, 'psql', '-X', '-q', '-A', '-t',
                '-v', 'ON_ERROR_STOP=1', '-h', '/var/run/postgresql', '-p', '5432',
                '-U', 'postgres', '-d', 'postgres',
                data=('BEGIN READ ONLY; SET LOCAL statement_timeout=5000;\n' +
                      PREFLIGHT_SQL + ';\nROLLBACK;\n').encode())
        value = decode(raw)
        require(value.get('ssl') == 'on' and value.get('hba_loaded') is True
                and isinstance(value.get('configuration_loaded'), str) and value['configuration_loaded']
                and value.get('system_identifier') == pg.system_identifier
                and value.get('database_oid') == pg.database_oid, 'namespace_tls_or_pgid_missing')
        hba = value.get('hba')
        require(isinstance(hba, list) and all(isinstance(r, dict) and r.get('error') is None
                for r in hba), 'namespace_hba_invalid')
        host = [r for r in hba if r.get('type') != 'local']
        require(len(host) == 4, 'namespace_nominal_hba_missing')
        for n in request.nominal:
            matches = [r for r in host if r.get('user_name') == [n.user]]
            require(len(matches) == 1 and matches[0].get('type') == 'hostssl'
                    and matches[0].get('database') == ['postgres']
                    and matches[0].get('address') == '127.0.0.1'
                    and matches[0].get('netmask') == '255.255.255.255'
                    and matches[0].get('auth_method') == 'cert'
                    and matches[0].get('options') in (None, [], ['clientcert=verify-full']),
                    'namespace_nominal_hba_missing')
        return digest(installer.clon_h6_kit.canonical(value))

    def start(self, request: Request, name: str) -> str:
        args = ['run', '--detach', '--pull=never', '--name', name,
                '--network', 'container:' + request.pg.container_id,
                '--user', f'{os.getuid()}:{os.getgid()}', '--read-only', '--cap-drop', 'ALL',
                '--security-opt', 'no-new-privileges:true', '--pids-limit', '32',
                '--memory', '128m', '--cpus', '0.5', '--log-driver', 'none',
                '--label', 'vec.recorridos.owner=Codex-M', '--entrypoint', '/usr/bin/python3']
        for filename in tls_hashes(request):
            args += ['--mount', f'type=bind,src={request.tls_directory / filename},dst=/tls/{filename},readonly,bind-propagation=rprivate']
        args += [request.helper_image_digest, '-I', '-c', BOOT]
        raw = self.command(*args, limit=65)
        require(re.fullmatch(rb'[a-f0-9]{64}\n?', raw) is not None, 'namespace_helper_creation_uncertain')
        return raw.decode().strip()

    def read(self, request: Request, cid: str, nonce: str) -> dict:
        payload = {'nonce': nonce, 'tls_hashes': tls_hashes(request),
                   'server_name': request.server_name, 'users': {n.channel: n.user for n in request.nominal}}
        raw = self.command('exec', '-i', '--user', f'{os.getuid()}:{os.getgid()}',
                cid, '/usr/bin/python3', '-I', '-c', READER,
                data=installer.clon_h6_kit.canonical(payload), limit=16384)
        return decode(raw)

    def remove(self, cid: str) -> None:
        require(relay.HEX64.fullmatch(cid) is not None, 'namespace_cid_invalid')
        raw = self.command('container', 'rm', '--force', cid, limit=65)
        require(raw in (cid.encode(), (cid + '\n').encode()), 'namespace_helper_removal_uncertain')


def verify_pg(request: Request, docker: Docker) -> tuple:
    pg = request.pg
    obj = docker.inspect(pg.container_id)
    host, state, net = (obj[k] for k in ('HostConfig', 'State', 'NetworkSettings'))
    require(obj['Image'] == pg.image_id and obj['Config'].get('Image') in ('postgres:18.4', pg.image_id)
            and state.get('Running') is True
            and host.get('NetworkMode') == 'none' and host.get('Privileged') is False
            and not host.get('CapAdd') and not host.get('Devices') and not host.get('DeviceRequests')
            and not any((host.get('PortBindings') or {}).values())
            and not any((net.get('Ports') or {}).values())
            and obj['Config']['Labels'].get(clon_sql.OWNER_LABEL) == clon_sql.OWNER
            and obj['Config']['Labels'].get('vec.recorridos.state') == str(request.state),
            'namespace_pg_inspection_drift')
    mounts = obj.get('Mounts')
    require(isinstance(mounts, list) and len(mounts) == 1
            and mounts[0].get('Type') == 'bind' and mounts[0].get('Source') == str(pg.volume)
            and mounts[0].get('Destination') == '/var/lib/postgresql'
            and mounts[0].get('RW') is True, 'namespace_volume_mount_drift')
    with relay.directory(pg.volume) as fd:
        info = os.fstat(fd)
        require((info.st_dev, info.st_ino) == (pg.volume_device, pg.volume_inode),
                'namespace_volume_identity_drift')
    return state['Pid'], state['StartedAt'], *process_identity(state['Pid'])


def verify_helper(request: Request, docker: Docker, cid: str, pg_identity: tuple) -> tuple:
    obj = docker.inspect(cid)
    host, state, net = (obj[k] for k in ('HostConfig', 'State', 'NetworkSettings'))
    require(obj['Image'] == request.helper_image_id and state.get('Running') is True
            and host.get('NetworkMode') == 'container:' + request.pg.container_id
            and host.get('Privileged') is False and host.get('ReadonlyRootfs') is True
            and host.get('CapDrop') == ['ALL'] and not host.get('CapAdd')
            and host.get('SecurityOpt') == ['no-new-privileges:true']
            and not host.get('Devices') and not host.get('DeviceRequests')
            and not host.get('PidMode') and not host.get('IpcMode') == 'host'
            and not any((host.get('PortBindings') or {}).values())
            and not any((net.get('Ports') or {}).values())
            and obj['Config'].get('User') == f'{os.getuid()}:{os.getgid()}'
            and obj['Config'].get('Entrypoint') == ['/usr/bin/python3']
            and obj['Config'].get('Cmd') == ['-I', '-c', BOOT], 'namespace_helper_inspection_drift')
    require(host.get('PidsLimit') == 32 and host.get('Memory') == 128 * 1024**2
            and host.get('NanoCpus') == 500_000_000
            and host.get('LogConfig', {}).get('Type') == 'none', 'namespace_helper_limits_drift')
    expected = {str(request.tls_directory / name): '/tls/' + name for name in tls_hashes(request)}
    mounts = obj.get('Mounts')
    require(isinstance(mounts, list) and len(mounts) == len(expected)
            and len({m.get('Destination') for m in mounts}) == len(expected)
            and all(m.get('Type') == 'bind' and m.get('RW') is False
                    and m.get('Propagation') == 'rprivate'
                    and expected.get(m.get('Source')) == m.get('Destination') for m in mounts),
            'namespace_helper_mount_drift')
    identity = process_identity(state['Pid'])
    require(identity[1:] == pg_identity[-2:], 'namespace_network_inode_mismatch')
    return state['Pid'], state['StartedAt'], *identity


def validate_sessions(request: Request, value: dict, nonce: str) -> dict:
    require(set(value) == {'version', 'nonce', 'tls_hashes', 'sessions'}
            and type(value['version']) is int and value['version'] == 1 and value['nonce'] == nonce
            and value['tls_hashes'] == tls_hashes(request)
            and isinstance(value['sessions'], dict) and set(value['sessions']) == set(CHANNELS),
            'namespace_session_output_uncertain')
    result = {}
    for n in request.nominal:
        rows = value['sessions'][n.channel]
        require(isinstance(rows, list) and len(rows) == 1 and isinstance(rows[0], list)
                and len(rows[0]) == 15, 'namespace_session_output_uncertain')
        row = rows[0]
        require(row[:4] == [n.user, n.user, 'none', 'on'] and type(row[4]) is int and row[4] > 0
                and row[5:11] == ['postgres', request.pg.database_oid,
                                 request.pg.system_identifier, '127.0.0.1', '127.0.0.1', 5432]
                and type(row[6]) is int and type(row[10]) is int
                and row[11] is True and row[12] is True and row[14] is True
                and row[13] in ('TLSv1.2', 'TLSv1.3'),
                'namespace_nominal_session_invalid')
        result[n.channel] = {'session_user': n.user, 'backend_pid': row[4],
                             'pg_container_id': request.pg.container_id}
    require(len({v['backend_pid'] for v in result.values()}) == 4, 'namespace_sessions_not_distinct')
    return result


def attest(request: Request) -> dict:
    """Pre/post physical proof; result is historical evidence, never live access.

    No factory, callback, SQL or shell is accepted. Fixtures patch concrete local
    boundaries. Repeated calls refuse the attempt marker, including after failure.
    Original H1/SQL62 records remain unchanged. Docker config is fresh and empty.
    """
    helper = None
    try:
        require(type(request) is Request, 'namespace_typed_request')
        request.validate()
        with ExitStack() as stack:
            directory, retained = stack.enter_context(installer.private_state(request.state))
            stack.enter_context(installer.installer_lock(directory))
            require(not installer.present(directory, ATTEMPT) and not installer.present(directory, RECEIPT),
                    'namespace_attempt_exists')
            snapshots = private_inputs(request, directory)
            # Fresh, empty, private Docker client config; no host credentials/context.
            config_name = 'cas-docker-' + secrets.token_hex(16)
            os.mkdir(config_name, 0o700, dir_fd=directory)
            config = request.state / config_name
            docker = Docker(config)
            stack.callback(docker.close)
            docker.image(request)
            pg_identity = verify_pg(request, docker)
            preflight_seal = docker.preflight(request)
            require(verify_pg(request, docker) == pg_identity, 'namespace_pg_process_drift')
            nonce = secrets.token_hex(32)
            marker = {'version': 1, 'nonce': nonce, 'restore_sha256': request.restore_sha256,
                      'journal_sha256': request.journal_sha256,
                      'helper_name': 'vec-cas-nominal-' + nonce[:32]}
            publish(directory, ATTEMPT, marker)
            helper = docker.start(request, marker['helper_name'])
            helper_identity = verify_helper(request, docker, helper, pg_identity)
            require(verify_pg(request, docker) == pg_identity, 'namespace_pg_process_drift')
            require(private_inputs(request, directory) == snapshots, 'namespace_tls_identity_drift')
            value = docker.read(request, helper, nonce)
            require(verify_pg(request, docker) == pg_identity, 'namespace_pg_process_drift')
            require(verify_helper(request, docker, helper, pg_identity) == helper_identity,
                    'namespace_helper_process_drift')
            require(docker.preflight(request) == preflight_seal, 'namespace_configuration_changed')
            require(verify_pg(request, docker) == pg_identity, 'namespace_pg_process_drift')
            require(private_inputs(request, directory) == snapshots, 'namespace_tls_identity_drift')
            installer.stable_state(request.state, retained)
            sessions = validate_sessions(request, value, nonce)
            docker.remove(helper)
            helper = None
            require(verify_pg(request, docker) == pg_identity, 'namespace_pg_process_drift')
            require(private_inputs(request, directory) == snapshots, 'namespace_tls_identity_drift')
            installer.stable_state(request.state, retained)
            receipt = {'version': 1, 'kind': 'nominal_ro_namespace_sessions',
                       'cas_sessions_bound': False,
                       'restore_sha256': request.restore_sha256, 'journal_sha256': request.journal_sha256,
                       'helper_image_id': request.helper_image_id, 'helper_sha256': HELPER_SHA256,
                       'pg_container_id': request.pg.container_id, 'sessions': sessions}
            publish(directory, RECEIPT, receipt)
            return receipt
    except Exception as error:
        # Owned helper is preserved on uncertainty for examination; no name-based
        # removal/adoption. No provider output, credentials or source paths escape.
        if isinstance(error, Refused):
            raise
        raise Refused('namespace_attestation_not_accredited') from None
