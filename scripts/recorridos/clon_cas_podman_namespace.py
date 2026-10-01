"""Prepared Podman v2 physical acquisition; no installed execution authority.

The transport and process observer are injected into the fixture seam. No shell,
Docker, remote transport, SQL, socket client or subprocess is installed here.
Public acquisition remains closed even if a caller replaces the gate marker.
JSON measurements are evidence of a completed observation, never permission.
"""
from __future__ import annotations

from contextlib import contextmanager
from dataclasses import dataclass
import fcntl
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat

try:
    from . import clon_cas_podman_contract as contract
except ImportError:
    import clon_cas_podman_contract as contract

Refused = contract.Refused
EXECUTION_AUTHORITY = None
ATTEMPT = 'cas-podman-v2-attempt.json'
RECEIPT = 'cas-podman-v2-measurement.json'
RESULT = 'cas-podman-v2-helper-result.json'
MODULES = ('clon_cas_podman_contract.py', 'clon_cas_podman_helper.py',
           'clon_preimagenes_nominales.py')
INPUTS = {'source_sha256': 'source.json', 'alias_sha256': 'alias.json',
          'restore_sha256': 'h1-restore.json', 'journal_sha256': 'sql-journal.json',
          'h1_records_sha256': 'h1-records.json',
          'aut26_receipt_sha256': 'aut26-receipt.json',
          'login_receipt_sha256': 'login-receipt.json', 'hba_sha256': 'pg_hba.conf',
          'acl_sha256': 'acl.json', 'rls_sha256': 'rls.json'}
HEX = re.compile(r'[a-f0-9]{64}')
IMAGE = re.compile(r'sha256:[a-f0-9]{64}')
MAX_FILE = 1 << 20
MAX_TOTAL = 4 << 20
SQL_BARRIER_TIMEOUT = 30


def require(ok, code):
    if not ok:
        raise Refused(code)


def keys(value, expected, code):
    require(type(value) is dict and set(value) == set(expected), code)


def integer(value, minimum=0):
    return type(value) is int and minimum <= value < 2**63


def mapped_id(maps, inner):
    matches = [m['host_id'] + inner - m['container_id'] for m in maps
               if m['container_id'] <= inner < m['container_id'] + m['size']]
    require(len(matches) == 1 and matches[0] > 0, 'podman_unmapped_id')
    return matches[0]


def parse_id_map(raw):
    require(type(raw) is str and 0 < len(raw) <= 4096, 'podman_map_size')
    result = []
    for line in raw.splitlines():
        fields = line.split()
        require(len(fields) == 3 and all(re.fullmatch(r'[0-9]+', x) for x in fields),
                'podman_map_format')
        inner, host, size = map(int, fields)
        require(0 <= inner < 2**32 and 0 < host < 2**32 and 0 < size < 2**32
                and max(inner, host) + size <= 2**32, 'podman_map_range')
        result.append({'container_id': inner, 'host_id': host, 'size': size})
    require(0 < len(result) <= 16, 'podman_map_count')
    for key in ('container_id', 'host_id'):
        ordered = sorted(result, key=lambda item: item[key])
        require(all(a[key] + a['size'] <= b[key] for a, b in zip(ordered, ordered[1:])),
                'podman_map_overlap')
    return result


def process_starttick(raw):
    require(type(raw) is str and len(raw) <= 8192 and ') ' in raw, 'podman_proc_stat')
    fields = raw[raw.rfind(') ') + 2:].split()
    require(len(fields) >= 20 and re.fullmatch(r'[0-9]+', fields[19]) is not None,
            'podman_proc_stat')
    tick = int(fields[19])
    require(integer(tick, 1), 'podman_proc_starttick')
    return tick


def mount_permissions(raw, paths):
    require(type(raw) is str and 0 < len(raw) <= MAX_FILE, 'podman_mountinfo_limit')
    found = {}
    for line in raw.splitlines():
        fields = line.split()
        require('-' in fields and len(fields) >= 10, 'podman_mountinfo_format')
        split = fields.index('-')
        require(split >= 6 and len(fields) == split + 4, 'podman_mountinfo_format')
        # Paths here are a closed, whitespace-free destination set.
        if fields[4] in paths:
            require(fields[4] not in found, 'podman_mountinfo_duplicate')
            options = set(fields[5].split(','))
            superoptions = set(fields[-1].split(','))
            require(len(options & {'rw', 'ro'}) == 1 and len(superoptions & {'rw', 'ro'}) == 1,
                    'podman_mountinfo_permissions')
            found[fields[4]] = 'rw' in options and 'rw' in superoptions
    require(set(found) == set(paths), 'podman_mountinfo_missing')
    return found


def observe_process(pid, started_at, paths):
    """Linux /proc observation only, with start tick revalidation around reads.

    Namespace symlinks and /proc/PID/root are intentional kernel references.
    Every destination below the retained proc root is walked without symlinks.
    This function does not authorize or start a process and is not used by CLI.
    """
    require(integer(pid, 1), 'podman_pid_invalid')
    proc = Path('/proc') / str(pid)
    tick = process_starttick((proc / 'stat').read_text())
    uid_map = parse_id_map((proc / 'uid_map').read_text())
    gid_map = parse_id_map((proc / 'gid_map').read_text())
    net = os.stat(proc / 'ns/net')
    mount_namespace = os.stat(proc / 'ns/mnt')
    permissions = mount_permissions((proc / 'mountinfo').read_text(), paths)
    root = os.open(proc / 'root', os.O_PATH | os.O_DIRECTORY | os.O_CLOEXEC)
    effective = {}
    try:
        for destination in paths:
            path = PurePosixPath(destination)
            require(path.is_absolute() and '..' not in path.parts, 'podman_effective_mount_path')
            fd = os.dup(root)
            try:
                for index, part in enumerate(path.parts[1:]):
                    options = os.O_PATH | os.O_NOFOLLOW | os.O_CLOEXEC
                    if index < len(path.parts) - 2:
                        options |= os.O_DIRECTORY
                    child = os.open(part, options, dir_fd=fd)
                    os.close(fd)
                    fd = child
                info = os.fstat(fd)
                require(not stat.S_ISLNK(info.st_mode), 'podman_effective_mount_symlink')
                effective[destination] = {'dev': info.st_dev, 'ino': info.st_ino,
                                          'rw': permissions[destination]}
            finally:
                os.close(fd)
        require(process_starttick((proc / 'stat').read_text()) == tick
                and identity(os.stat(proc / 'ns/net')) == identity(net)
                and identity(os.stat(proc / 'ns/mnt')) == identity(mount_namespace),
                'podman_process_changed_during_observation')
    finally:
        os.close(root)
    return {'process': {'pid': pid, 'StartedAt': started_at, 'starttick': tick,
                       'dev': net.st_dev, 'ino': net.st_ino},
            'mount_namespace': {'dev': mount_namespace.st_dev, 'ino': mount_namespace.st_ino},
            'uid_map': uid_map, 'gid_map': gid_map, 'effective_mounts': effective}


def closed_path(path):
    require(type(path) is Path or isinstance(path, Path), 'podman_path_type')
    require(path.is_absolute() and path != Path('/') and '..' not in path.parts
            and not any(c in str(path) for c in '\x00\n\r,:\\'), 'podman_path_invalid')
    return path


@contextmanager
def directory(path):
    """Walk absolute paths with retained directory descriptors, without links."""
    closed_path(path)
    fd = os.open('/', os.O_PATH | os.O_DIRECTORY | os.O_CLOEXEC)
    try:
        for part in path.parts[1:]:
            child = os.open(part, os.O_PATH | os.O_DIRECTORY | os.O_NOFOLLOW |
                            os.O_CLOEXEC, dir_fd=fd)
            os.close(fd)
            fd = child
        yield fd
    finally:
        os.close(fd)


def identity(info):
    return (info.st_dev, info.st_ino, info.st_uid, info.st_gid,
            stat.S_IMODE(info.st_mode), info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def directory_identity(path):
    with directory(path) as fd:
        info = os.fstat(fd)
        return {'dev': info.st_dev, 'ino': info.st_ino}


def readable(info, uid, gid):
    shift = 6 if info.st_uid == uid else 3 if info.st_gid == gid else 0
    return bool(stat.S_IMODE(info.st_mode) & (4 << shift))


def file_snapshot(path, expected, uid, gid):
    closed_path(path)
    with directory(path.parent) as parent:
        fd = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK |
                     os.O_CLOEXEC, dir_fd=parent)
        try:
            before = os.fstat(fd)
            require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1
                    and 0 < before.st_size <= MAX_FILE and readable(before, uid, gid),
                    'podman_input_not_minimal_readable_regular')
            raw = bytearray()
            while len(raw) <= MAX_FILE:
                part = os.read(fd, min(65536, MAX_FILE + 1 - len(raw)))
                if not part:
                    break
                raw.extend(part)
            named = os.stat(path.name, dir_fd=parent, follow_symlinks=False)
            require(len(raw) == before.st_size and identity(before) == identity(os.fstat(fd))
                    == identity(named), 'podman_input_replaced')
            require(contract.digest(bytes(raw)) == expected, 'podman_input_pin_drift')
            return identity(before)
        finally:
            os.close(fd)


def parent_only_tree(source, destination):
    """The anonymous volume may contain only the nested bind's directory chain.

    Its externally approved digest covers ordered relative directory names.
    The nested PGDATA content is observed separately and is never read here.
    """
    root = PurePosixPath('/var/lib/postgresql')
    target = PurePosixPath(destination)
    require(target != root and target.is_relative_to(root)
            and '..' not in target.parts, 'podman_pgdata_not_nested')
    chain = target.relative_to(root).parts
    names = []
    with directory(source) as root_fd:
        fd = os.open('.', os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC, dir_fd=root_fd)
        try:
            for index, part in enumerate(chain):
                entries = os.listdir(fd)
                # A nonexistent mountpoint cannot be silently made by this code.
                require(entries == [part], 'podman_parent_volume_not_directory_chain')
                child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW |
                                os.O_CLOEXEC, dir_fd=fd)
                os.close(fd)
                fd = child
                names.append('/'.join(chain[:index + 1]) + '/')
            require(not os.listdir(fd), 'podman_parent_volume_has_content')
        finally:
            os.close(fd)
    return contract.digest(contract.canonical({'directories': names}))


@dataclass(frozen=True)
class Bind:
    source: Path
    destination: str
    sha256: str


@dataclass(frozen=True)
class Layout:
    state: Path
    controller_socket: Path
    inputs: tuple[Bind, ...]


def snapshot_inputs(request, layout):
    """Approve the complete closure and exact files; never mount directories."""
    runtime, helper = request['runtime'], request['helper']
    uid = mapped_id(runtime['uid_map'], helper['uid'])
    gid = mapped_id(runtime['gid_map'], helper['gid'])
    expected = {'/tls/' + name: sha for name, sha in request['tls_hashes'].items()}
    expected.update({'/inputs/' + name: request['pins'][key] for key, name in INPUTS.items()})
    destinations = [bind.destination for bind in layout.inputs]
    module_destinations = {'/helper/' + name for name in MODULES}
    require(len(destinations) == len(set(destinations))
            and set(destinations) == set(expected) | module_destinations,
            'podman_input_mount_set')
    module_hashes = {b.destination.split('/')[-1]: b.sha256 for b in layout.inputs
                     if b.destination in module_destinations}
    require(contract.digest(contract.canonical(module_hashes)) == helper['code_sha256'],
            'podman_helper_closure_pin')
    observed, total = {}, 0
    sources = set()
    for bind in layout.inputs:
        require(type(bind) is Bind and HEX.fullmatch(bind.sha256 or '') is not None,
                'podman_input_pin_invalid')
        require(bind.source not in sources and bind.source != layout.controller_socket
                and not bind.source.is_relative_to(layout.state), 'podman_input_source_set')
        sources.add(bind.source)
        require(bind.destination in module_destinations or expected[bind.destination] == bind.sha256,
                'podman_input_external_pin')
        observed[bind.destination] = file_snapshot(bind.source, bind.sha256, uid, gid)
        total += observed[bind.destination][5]
    require(total <= MAX_TOTAL, 'podman_input_total_limit')
    return observed


def validate_controller(request, socket_path, info):
    runtime = request['runtime']
    keys(info, ('rootless', 'remote', 'socket_path', 'owner_uid', 'owner_gid'),
         'podman_controller_schema')
    require(info == {'rootless': True, 'remote': False, 'socket_path': str(socket_path),
                    'owner_uid': runtime['owner_uid'], 'owner_gid': runtime['owner_gid']},
            'podman_local_rootless_controller_required')
    closed_path(socket_path)
    with directory(socket_path.parent) as parent:
        parent_info = os.fstat(parent)
        require(parent_info.st_uid == runtime['owner_uid']
                and stat.S_IMODE(parent_info.st_mode) == 0o700,
                'podman_socket_parent_private')
        socket = os.stat(socket_path.name, dir_fd=parent, follow_symlinks=False)
        require(stat.S_ISSOCK(socket.st_mode) and socket.st_uid == runtime['owner_uid']
                and not stat.S_IMODE(socket.st_mode) & 0o007, 'podman_socket_not_authorized')
        return identity(socket)


def validate_image(image, image_id, repo_digest, *, helper=False):
    keys(image, ('Id', 'RepoDigests', 'Config'), 'podman_image_schema')
    keys(image['Config'], ('Volumes', 'User'), 'podman_image_config_schema')
    require(IMAGE.fullmatch(image_id or '') is not None and image['Id'] == image_id
            and type(image['RepoDigests']) is list and repo_digest in image['RepoDigests'],
            'podman_local_image_pin')
    if helper:
        require(not image['Config']['Volumes'], 'podman_helper_image_volumes')


def mount_projection(mount):
    expected = ('Type', 'Source', 'Destination', 'RW', 'Propagation', 'Options')
    if mount.get('Type') == 'volume':
        expected += ('Name', 'Driver')
    keys(mount, expected, 'podman_mount_unexpected_fields')
    require(type(mount['RW']) is bool and mount['Propagation'] == 'rprivate'
            and type(mount['Options']) is list
            and set(mount['Options']) <= {'rw', 'ro', 'rbind', 'rprivate'},
            'podman_mount_options')
    require(not ({'rw', 'ro'} <= set(mount['Options']))
            and ('ro' not in mount['Options'] if mount['RW'] else 'rw' not in mount['Options']),
            'podman_mount_permissions_conflict')
    return mount


def common_inspection(value, cid, image_id):
    keys(value, ('Id', 'Image', 'Config', 'HostConfig', 'State', 'Mounts', 'NetworkSettings'),
         'podman_container_schema')
    keys(value['Config'], ('Image', 'User', 'Entrypoint', 'Cmd'), 'podman_config_schema')
    keys(value['State'], ('Running', 'Pid', 'StartedAt', 'ExitCode'), 'podman_state_schema')
    keys(value['NetworkSettings'], ('Ports',), 'podman_network_schema')
    require(HEX.fullmatch(cid or '') is not None and value['Id'] == cid
            and value['Image'] == image_id and value['State']['Running'] is True
            and integer(value['State']['Pid'], 1) and type(value['State']['StartedAt']) is str
            and value['State']['StartedAt'] and not value['NetworkSettings']['Ports'],
            'podman_container_identity')
    host = value['HostConfig']
    keys(host, ('NetworkMode', 'Privileged', 'ReadonlyRootfs', 'CapDrop', 'CapAdd',
                'SecurityOpt', 'Devices', 'DeviceRequests', 'PidMode', 'IpcMode',
                'PortBindings', 'PidsLimit', 'Memory', 'NanoCpus', 'LogConfig'),
         'podman_host_schema')
    require(host['Privileged'] is False and not host['CapAdd'] and not host['Devices']
            and not host['DeviceRequests'] and not host['PortBindings']
            and host['PidMode'] in ('', 'private') and host['IpcMode'] != 'host',
            'podman_host_boundary')
    require(type(value['Mounts']) is list, 'podman_mounts_schema')
    for mount in value['Mounts']:
        mount_projection(mount)
    return host, value['State']


def validate_postgres(request, inspected, volume, image, observed):
    pg, runtime = request['pgid'], request['runtime']
    validate_image(image, pg['pg_image_id'], pg['pg_repo_digest'])
    host, state = common_inspection(inspected, pg['pg_container_id'], pg['pg_image_id'])
    require(host['NetworkMode'] == 'none' and inspected['Config']['Image'] in
            (pg['pg_image_id'], pg['pg_repo_digest']), 'podman_postgres_image_or_network')
    expected = runtime['mounts']
    a, p = expected['anonymous'], expected['pgdata']
    keys(volume, ('Name', 'Driver', 'Mountpoint', 'Anonymous', 'Options'),
         'podman_volume_schema')
    require(volume == {'Name': a['name'], 'Driver': 'local', 'Mountpoint': a['source'],
                       'Anonymous': True, 'Options': {}}, 'podman_anonymous_local_volume')
    require(len(inspected['Mounts']) == 2, 'podman_postgres_mount_count')
    by_dest = {m['Destination']: m for m in inspected['Mounts']}
    require(len(by_dest) == 2 and set(by_dest) == {a['destination'], p['destination']},
            'podman_postgres_mount_destinations')
    actual_a, actual_p = by_dest[a['destination']], by_dest[p['destination']]
    require(actual_a['Type'] == 'volume' and actual_a['Name'] == a['name']
            and HEX.fullmatch(a['name']) is not None and actual_a['Driver'] == 'local'
            and actual_a['Source'] == a['source'] and actual_a['RW'] is True
            and actual_p['Type'] == 'bind' and actual_p['Source'] == p['source']
            and actual_p['RW'] is True and p['source'] == pg['pgdata_bind_source']
            and p['destination'] == pg['pgdata_bind_destination'], 'podman_postgres_mount_identity')
    require(Path(p['source']).name == 'h6-pgdata', 'podman_pgdata_private_origin_name')
    with directory(Path(p['source']).parent) as fd:
        parent = os.fstat(fd)
        require(parent.st_uid == runtime['owner_uid']
                and stat.S_IMODE(parent.st_mode) == 0o700, 'podman_pgdata_parent_private')
    for mount in (a, p):
        require(directory_identity(Path(mount['source'])) == {'dev': mount['dev'], 'ino': mount['ino']},
                'podman_mount_source_inode')
    with directory(Path(p['source'])) as fd:
        info = os.fstat(fd)
        require(stat.S_IMODE(info.st_mode) == 0o700
                and info.st_uid == mapped_id(runtime['uid_map'], 999)
                and info.st_gid == mapped_id(runtime['gid_map'], 999), 'podman_pgdata_private_owner')
    require(parent_only_tree(Path(a['source']), p['destination']) == a['tree_sha256'],
            'podman_parent_tree_pin')
    validate_process(observed, state, runtime)
    require(observed['process'] == runtime['postgres_process']
            and observed['mount_namespace'] == runtime['postgres_mount_namespace'],
            'podman_postgres_process_pin')
    require(observed['effective_mounts'] == {
        a['destination']: {'dev': a['dev'], 'ino': a['ino'], 'rw': True},
        p['destination']: {'dev': p['dev'], 'ino': p['ino'], 'rw': True}},
        'podman_effective_mount_map')
    return observed['process']


def validate_process(observed, state, runtime):
    keys(observed, ('process', 'mount_namespace', 'uid_map', 'gid_map', 'effective_mounts'),
         'podman_process_observation_schema')
    process = observed['process']
    keys(process, ('pid', 'StartedAt', 'starttick', 'dev', 'ino'), 'podman_process_schema')
    require(process['pid'] == state['Pid'] and process['StartedAt'] == state['StartedAt']
            and integer(process['starttick'], 1)
            and {'dev': process['dev'], 'ino': process['ino']} == runtime['network_namespace']
            and observed['uid_map'] == runtime['uid_map']
            and observed['gid_map'] == runtime['gid_map'], 'podman_process_or_rootless_map_drift')
    keys(observed['mount_namespace'], ('dev', 'ino'), 'podman_mount_namespace_schema')
    require(integer(observed['mount_namespace']['dev'])
            and integer(observed['mount_namespace']['ino'], 1), 'podman_mount_namespace_invalid')


def validate_helper(request, inspected, cid, observed, binds, input_snapshot):
    helper, runtime = request['helper'], request['runtime']
    require(cid != request['pgid']['pg_container_id'], 'podman_helper_postgres_collision')
    host, state = common_inspection(inspected, cid, helper['image_id'])
    require(inspected['Config'] == {'Image': helper['image_id'], 'User': '10002:10002',
             'Entrypoint': ['python3'], 'Cmd': ['/helper/clon_cas_podman_helper.py']}
            and host['NetworkMode'] == 'container:' + request['pgid']['pg_container_id']
            and host['ReadonlyRootfs'] is True and host['CapDrop'] == ['ALL']
            and host['SecurityOpt'] == ['no-new-privileges']
            and host['PidsLimit'] == 32 and host['Memory'] == 128 * 1024**2
            and host['NanoCpus'] == 500_000_000 and host['LogConfig'] == {'Type': 'none'},
            'podman_helper_boundary')
    expected = {b.destination: str(b.source) for b in binds}
    mounts = inspected['Mounts']
    require(len(mounts) == len(expected) and len({m['Destination'] for m in mounts}) == len(expected)
            and all(m['Type'] == 'bind' and m['RW'] is False
                    and expected.get(m['Destination']) == m['Source'] for m in mounts),
            'podman_helper_minimal_ro_mounts')
    validate_process(observed, state, runtime)
    require(observed['mount_namespace'] != runtime['postgres_mount_namespace']
            and observed['process']['pid'] != runtime['postgres_process']['pid'],
            'podman_helper_private_process_and_mount_namespace')
    effective = observed['effective_mounts']
    require(type(effective) is dict and '/' in effective, 'podman_helper_rootfs_observation_missing')
    root = effective['/']
    keys(root, ('dev', 'ino', 'rw'), 'podman_helper_rootfs_schema')
    require(integer(root['dev']) and integer(root['ino'], 1) and root['rw'] is False,
            'podman_helper_effective_rootfs_writable')
    require({k: v for k, v in effective.items() if k != '/'} == {
            destination: {'dev': value[0], 'ino': value[1], 'rw': False}
            for destination, value in input_snapshot.items()}, 'podman_helper_effective_mounts')
    return observed['process']


def physical_snapshot(request, layout, backend):
    """Transport returns narrow projections; proc observer is a separate source."""
    inputs = snapshot_inputs(request, layout)
    pg = request['pgid']
    pg_inspect = backend.inspect(pg['pg_container_id'])
    pg_observed = backend.observe(pg_inspect['State']['Pid'], pg_inspect['State']['StartedAt'])
    postgres = validate_postgres(request, pg_inspect,
        backend.volume(request['runtime']['mounts']['anonymous']['name']),
        backend.image(pg['pg_image_id']), pg_observed)
    helper_inspect = backend.inspect(backend.helper_id)
    helper_observed = backend.observe(helper_inspect['State']['Pid'], helper_inspect['State']['StartedAt'])
    helper = validate_helper(request, helper_inspect, backend.helper_id,
                             helper_observed, layout.inputs, inputs)
    runtime = request['runtime']
    return {'postgres_process': postgres, 'helper_process': helper,
            'network_namespace': runtime['network_namespace'],
            'postgres_mount_namespace': pg_observed['mount_namespace'],
            'helper_mount_namespace': helper_observed['mount_namespace'],
            'uid_map': pg_observed['uid_map'], 'gid_map': pg_observed['gid_map'],
            'mounts': runtime['mounts'], 'mount_inventory_sha256': runtime['mount_inventory_sha256']}


def postgres_snapshot(request, backend):
    pg = request['pgid']
    inspected = backend.inspect(pg['pg_container_id'])
    observed = backend.observe(inspected['State']['Pid'], inspected['State']['StartedAt'])
    validate_postgres(request, inspected,
        backend.volume(request['runtime']['mounts']['anonymous']['name']),
        backend.image(pg['pg_image_id']), observed)
    return observed


def helper_spec(request, layout):
    """Fixed specification handed only to the injected fixture transport."""
    return {'image_id': request['helper']['image_id'], 'pull': 'never',
            'network': 'container:' + request['pgid']['pg_container_id'],
            'user': '10002:10002', 'read_only': True, 'cap_drop': ['ALL'],
            'security_opt': ['no-new-privileges'], 'pids_limit': 32,
            'memory': 128 * 1024**2, 'nano_cpus': 500_000_000, 'log_driver': 'none',
            'entrypoint': ['python3'], 'command': ['/helper/clon_cas_podman_helper.py'],
            'mounts': [{'type': 'bind', 'source': str(bind.source),
                        'destination': bind.destination, 'rw': False,
                        'propagation': 'rprivate'} for bind in layout.inputs]}


def publish_raw(fd, name, raw):
    require(type(raw) is bytes and 0 < len(raw) <= MAX_FILE, 'podman_evidence_limit')
    leaf = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW |
                   os.O_CLOEXEC, 0o600, dir_fd=fd)
    try:
        view = memoryview(raw)
        while view:
            count = os.write(leaf, view)
            require(count > 0, 'podman_evidence_write')
            view = view[count:]
        os.fsync(leaf)
    finally:
        os.close(leaf)
    os.fsync(fd)


def publish(fd, name, value):
    publish_raw(fd, name, contract.canonical(value))


def controller_normalizer_bundle(request, backend):
    """Pin both controller-side programs, never an assertion from helper JSON."""
    bundle = backend.normalizer_bundle()
    keys(bundle, ('h6_comun.sh', 'h6_normalizar_pg_dump.py'), 'podman_normalizer_bundle_files')
    require(all(type(raw) is bytes and 0 < len(raw) <= MAX_FILE for raw in bundle.values()),
            'podman_normalizer_bundle_size')
    hashes = {name: contract.digest(raw) for name, raw in bundle.items()}
    require(hashes == request['normalizer_bundle']
            and contract.digest(contract.canonical(hashes)) == request['pins']['normalizer_sha256'],
            'podman_controller_normalizer_pin')
    return hashes


def sql_barrier_observer(request, layout, backend, expected_physical):
    """Private fixture adapter: controller observes at two open-session barriers.

    The helper can request a stage and provide session identities. It cannot
    supply the authoritative SQL state or normalizer evidence. There is no live
    transport here; the future controller must enforce the passed hard timeout.
    """
    observations, sessions = {}, {}
    def observe(stage, helper_request, initial_sessions):
        require(stage == ('before' if not observations else 'after')
                and stage not in observations, 'podman_sql_barrier_order')
        require(contract.canonical(helper_request) == contract.canonical(request),
                'podman_sql_barrier_request_changed')
        controller_normalizer_bundle(request, backend)
        initial = contract.decode(contract.canonical(initial_sessions))
        if sessions:
            require(initial == sessions, 'podman_sql_barrier_sessions_changed')
        else:
            sessions.update(initial)
        require(physical_snapshot(request, layout, backend) == expected_physical,
                'podman_sql_barrier_physical_changed')
        acquired = backend.observe_sql(stage, request, initial,
                                       timeout_seconds=SQL_BARRIER_TIMEOUT)
        attestation = contract.validate_sql_postimage_observation(acquired, request, initial, stage)
        controller_normalizer_bundle(request, backend)
        require(physical_snapshot(request, layout, backend) == expected_physical,
                'podman_sql_barrier_physical_changed')
        # Retain a detached observation before returning another detached value.
        observations[stage] = contract.decode(contract.canonical(attestation))
        return contract.decode(contract.canonical(attestation))
    return observe, observations, sessions


def close_started_helper(backend, original_error):
    """Always attempt both closure and confirmation; preserve the primary error."""
    close_ok = confirmed = False
    cleanup_error = None
    try:
        close_ok = backend.close_helper() is True
    except BaseException as error:
        cleanup_error = error
    try:
        confirmed = backend.helper_closed() is True
    except BaseException as error:
        if cleanup_error is None:
            cleanup_error = error
    if not (close_ok and confirmed):
        if original_error is not None:
            original_error.add_note('podman_helper_close_unconfirmed')
        elif cleanup_error is not None:
            raise cleanup_error
        else:
            raise Refused('podman_helper_close_unconfirmed')
    return close_ok and confirmed


def _measure_fixture(request, layout, backend):
    """Non-CLI seam for simulated transport tests, not an authority provider.

    No concrete backend exists in this delivery. Caller-supplied JSON, result or
    a previous receipt cannot reach this through acquire()/main().
    """
    request = contract.validate_request(request)
    require(type(layout) is Layout, 'podman_layout_contract')
    state_path = closed_path(layout.state)
    require(not any((p / '.git').exists() for p in (state_path, *state_path.parents)),
            'podman_private_state_in_git')
    initial_inputs = snapshot_inputs(request, layout)
    socket_identity = validate_controller(request, layout.controller_socket, backend.controller())
    validate_image(backend.image(request['helper']['image_id']),
                   request['helper']['image_id'], request['helper']['repo_digest'], helper=True)
    controller_normalizer_bundle(request, backend)
    pg_before = postgres_snapshot(request, backend)
    with directory(state_path) as path_fd:
        fd = os.open('.', os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC, dir_fd=path_fd)
        try:
            state = os.fstat(fd)
            require(state.st_uid == request['runtime']['owner_uid']
                    and stat.S_IMODE(state.st_mode) == 0o700, 'podman_state_private')
            require(not {ATTEMPT, RECEIPT, RESULT} & set(os.listdir(fd)), 'podman_attempt_requires_new_state')
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            pending = {'version': 2, 'kind': contract.PENDING_KIND, 'request': request,
                       'request_sha256': contract.digest(contract.canonical(request))}
            contract.validate_pending(pending)
            publish(fd, ATTEMPT, pending)
            require(snapshot_inputs(request, layout) == initial_inputs
                    and postgres_snapshot(request, backend) == pg_before,
                    'podman_prestart_inputs_or_postgres_changed')
            # First helper effect follows the exclusively created durable attempt.
            start_attempted, primary_error = False, None
            try:
                # Set before the call: a failed start may already own a helper.
                start_attempted = True
                backend.start_helper(helper_spec(request, layout))
                before = physical_snapshot(request, layout, backend)
                observer, sql_observations, sessions = sql_barrier_observer(request, layout, backend, before)
                result_raw = backend.measure(request, observer)
                result = contract.bind_result(request, contract.decode(result_raw))
                require(set(sql_observations) == {'before', 'after'}
                        and result['sql_postimage_observations'] == sql_observations
                        and all(pair['before'] == sessions[channel]
                                for channel, pair in result['sessions'].items()),
                        'podman_sql_observations_not_acquired')
                after = physical_snapshot(request, layout, backend)
                require(before == after and snapshot_inputs(request, layout) == initial_inputs,
                        'podman_physical_or_inputs_changed')
                require(validate_controller(request, layout.controller_socket, backend.controller())
                        == socket_identity, 'podman_controller_socket_changed')
            except BaseException as error:
                primary_error = error
                raise
            finally:
                if start_attempted:
                    close_started_helper(backend, primary_error)
            require(snapshot_inputs(request, layout) == initial_inputs, 'podman_inputs_changed_after_close')
            require(postgres_snapshot(request, backend) == pg_before,
                    'podman_postgres_changed_after_close')
            with directory(state_path) as final_fd:
                named = os.fstat(final_fd)
                require((named.st_dev, named.st_ino) == (state.st_dev, state.st_ino),
                        'podman_state_replaced')
            receipt = {'version': 2, 'kind': contract.MEASUREMENT_KIND,
                       'request_sha256': pending['request_sha256'], 'result': result,
                       'result_bytes_sha256': contract.digest(result_raw),
                       'result_canonical_sha256': contract.digest(contract.canonical(result)),
                       'physical': {'before': before, 'after': after}}
            contract.validate_measurement(receipt, request, result_raw=result_raw)
            publish_raw(fd, RESULT, result_raw)
            publish(fd, RECEIPT, receipt)
            return receipt
        finally:
            os.close(fd)


def acquire(*args, **kwargs):
    """No provider is accredited. Reject before parsing, path IO or transport."""
    require(EXECUTION_AUTHORITY is not None, 'podman_execution_authority_pending')
    # Assigning a marker/object is not authority; the missing provider stays closed.
    raise Refused('podman_execution_authority_pending')


def main(argv=None):
    try:
        acquire()
    except Refused:
        print(json.dumps({'status': 'refused', 'code': 'podman_execution_authority_pending'}))
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
