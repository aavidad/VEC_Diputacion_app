"""Podman projections and private filesystem fixtures; never execute a runtime."""
from copy import deepcopy
from contextlib import redirect_stdout
import io
import json
import os
from pathlib import Path
import socket
import tempfile
import unittest
from unittest.mock import patch

try:
    from . import clon_cas_podman_namespace as ns
    from . import test_clon_cas_podman_contract as fixtures
except ImportError:
    import clon_cas_podman_namespace as ns
    import test_clon_cas_podman_contract as fixtures

FIXTURE_NORMALIZERS = {'h6_comun.sh': b'# inert H6 controller shell fixture\n',
                       'h6_normalizar_pg_dump.py': b'# inert H6 normalizer fixture\n'}


def inspection(req, cid, helper, binds=()):
    pg = req['pgid']
    host = {'NetworkMode': 'container:' + pg['pg_container_id'] if helper else 'none',
            'Privileged': False, 'ReadonlyRootfs': helper, 'CapDrop': ['ALL'],
            'CapAdd': [], 'SecurityOpt': ['no-new-privileges'], 'Devices': [],
            'DeviceRequests': [], 'PidMode': 'private', 'IpcMode': 'private',
            'PortBindings': {}, 'PidsLimit': 32, 'Memory': 128 * 1024**2,
            'NanoCpus': 500_000_000, 'LogConfig': {'Type': 'none'}}
    if helper:
        mounts = [{'Type': 'bind', 'Source': str(b.source), 'Destination': b.destination,
                   'RW': False, 'Propagation': 'rprivate', 'Options': ['ro']} for b in binds]
    else:
        a, p = (req['runtime']['mounts'][k] for k in ('anonymous', 'pgdata'))
        mounts = [{'Type': 'volume', 'Name': a['name'], 'Driver': 'local',
                   'Source': a['source'], 'Destination': a['destination'], 'RW': True,
                   'Propagation': 'rprivate', 'Options': ['rw']},
                  {'Type': 'bind', 'Source': p['source'], 'Destination': p['destination'],
                   'RW': True, 'Propagation': 'rprivate', 'Options': ['rw']}]
    return {'Id': cid, 'Image': req['helper']['image_id'] if helper else pg['pg_image_id'],
            'Config': {'Image': req['helper']['image_id'] if helper else pg['pg_image_id'],
                       'User': '10002:10002' if helper else '999:999',
                       'Entrypoint': ['python3'] if helper else ['postgres'],
                       'Cmd': ['/helper/clon_cas_podman_helper.py'] if helper else []},
            'HostConfig': host, 'State': {'Running': True, 'Pid': 101 if helper else 100,
                 'StartedAt': fixtures.T, 'ExitCode': 0}, 'Mounts': mounts,
            'NetworkSettings': {'Ports': {}}}


class FakePodman:
    """No command runner or live socket use: pre-recorded narrow projections."""
    def __init__(self, req, layout):
        self.req, self.layout = req, layout
        self.helper_id = '9' * 64
        self.events = []
        self.pg = inspection(req, req['pgid']['pg_container_id'], False)
        self.helper = inspection(req, self.helper_id, True, layout.inputs)
        self.returned = fixtures.result(req)
        self.close_ok = True
        self.closed = False
        self.drift = None
        self.raw_result = None
        self.normalizers = deepcopy(FIXTURE_NORMALIZERS)
        self.sql_state = deepcopy(req['receipt_bindings']['post'])
        self.helper_open = False
        self.open_session_count = 0
        self.fail_at = None
        self.failure = RuntimeError('fixture_primary_failure')
        self.close_error = None
        self.confirm_ok = True
        self.skip_observers = False
        self.observer_order = ('before', 'after')

    def fail(self, stage):
        if self.fail_at == stage:
            raise self.failure

    def normalizer_bundle(self):
        return deepcopy(self.normalizers)

    def observe_sql(self, stage, request, initial_sessions, *, timeout_seconds):
        assert self.helper_open and self.open_session_count == 3
        assert timeout_seconds == ns.SQL_BARRIER_TIMEOUT
        self.events.append('sql-' + stage)
        self.fail('sql-' + stage)
        pg = self.req['pgid']
        process = self.observe(self.pg['State']['Pid'], self.pg['State']['StartedAt'])['process']
        return {'stage': stage, 'nonce': self.req['nonce'],
                'request_sha256': ns.contract.digest(ns.contract.canonical(self.req)),
                'sessions_sha256': ns.contract.digest(ns.contract.canonical(initial_sessions)),
                'pg_container_id': pg['pg_container_id'], 'pg_image_id': pg['pg_image_id'],
                'system_identifier': pg['system_identifier'], 'database_oid': pg['database_oid'],
                'normalizer_sha256': self.req['pins']['normalizer_sha256'],
                'normalizer_bundle': deepcopy(self.req['normalizer_bundle']),
                'postgres_process': process,
                'network_namespace': {'dev': process['dev'], 'ino': process['ino']},
                'mount_inventory_sha256': ns.contract.digest(ns.contract.canonical(self.req['runtime']['mounts'])),
                **self.sql_state}

    def controller(self):
        return {'rootless': True, 'remote': False, 'socket_path': str(self.layout.controller_socket),
                'owner_uid': self.req['runtime']['owner_uid'],
                'owner_gid': self.req['runtime']['owner_gid']}

    def image(self, image_id):
        helper = image_id == self.req['helper']['image_id']
        return {'Id': image_id,
                'RepoDigests': [self.req['helper']['repo_digest'] if helper
                                else self.req['pgid']['pg_repo_digest']],
                'Config': {'Volumes': None, 'User': ''}}

    def volume(self, name):
        a = self.req['runtime']['mounts']['anonymous']
        return {'Name': name, 'Driver': 'local', 'Mountpoint': a['source'],
                'Anonymous': True, 'Options': {}}

    def inspect(self, cid):
        if cid == self.helper_id:
            self.fail('physical')
        return deepcopy(self.helper if cid == self.helper_id else self.pg)

    def observe(self, pid, started_at):
        runtime = self.req['runtime']
        if pid == 100:
            process = deepcopy(runtime['postgres_process'])
            mounts = {m['destination']: {'dev': m['dev'], 'ino': m['ino'], 'rw': True}
                      for m in runtime['mounts'].values()}
            namespace = runtime['postgres_mount_namespace']
        else:
            process = {'pid': 101, 'StartedAt': started_at, 'starttick': 5678, 'dev': 4, 'ino': 123}
            namespace = {'dev': 4, 'ino': 125}
            mounts = {b.destination: {'dev': b.source.stat().st_dev,
                       'ino': b.source.stat().st_ino, 'rw': False} for b in self.layout.inputs}
            mounts['/'] = {'dev': 4, 'ino': 99999, 'rw': False}
        if self.drift and 'measure' in self.events:
            self.drift(process, mounts)
        return {'process': process, 'mount_namespace': deepcopy(namespace),
                'uid_map': deepcopy(runtime['uid_map']), 'gid_map': deepcopy(runtime['gid_map']),
                'effective_mounts': mounts}

    def start_helper(self, spec):
        # The attempt is complete and durable before this simulated effect.
        pending = ns.contract.decode((self.layout.state / ns.ATTEMPT).read_bytes())
        ns.contract.validate_pending(pending)
        assert spec == ns.helper_spec(self.req, self.layout)
        assert spec['user'] == '10002:10002' and spec['pull'] == 'never'
        self.events.append('start')
        self.helper_open = True
        self.fail('start-partial')

    def measure(self, request, state_observer):
        self.events.append('measure')
        self.fail('measure')
        self.open_session_count = 3
        try:
            initial = {channel: pair['before'] for channel, pair in self.returned['sessions'].items()}
            if not self.skip_observers:
                acquired = {stage: state_observer(stage, request, initial) for stage in self.observer_order}
                self.returned['sql_postimage_observations'] = acquired
                self.returned['postimage'] = {key: acquired['after'][key] for key in ns.contract.STATE_FIELDS}
            return self.raw_result if self.raw_result is not None else ns.contract.canonical(self.returned)
        finally:
            self.open_session_count = 0

    def close_helper(self):
        assert not (self.layout.state / ns.RECEIPT).exists()
        self.events.append('close')
        if self.close_error is not None:
            raise self.close_error
        self.closed = self.close_ok
        if self.closed:
            self.helper_open = False
        return self.close_ok

    def helper_closed(self):
        self.events.append('confirm_close')
        return self.closed and self.confirm_ok


class FilesystemTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='namespace-fixture-')
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        req = fixtures.request()
        runtime = req['runtime']
        runtime['owner_uid'], runtime['owner_gid'] = os.getuid(), os.getgid()
        for kind, owner in (('uid_map', os.getuid()), ('gid_map', os.getgid())):
            runtime[kind][0]['host_id'] = owner
        source, state, sockdir, anonymous, pgdata = (root / n for n in
            ('sources', 'state', 'socket', 'anonymous', 'h6-pgdata'))
        for p in (source, state, sockdir, anonymous, pgdata):
            p.mkdir(mode=0o700)
        (anonymous / '18' / 'docker').mkdir(parents=True)
        sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.addCleanup(sock.close)
        sockpath = sockdir / 'podman.sock'
        sock.bind(str(sockpath))
        sockpath.chmod(0o600)
        req['pgid']['pgdata_bind_source'] = str(pgdata)
        a, p = runtime['mounts']['anonymous'], runtime['mounts']['pgdata']
        a['source'], p['source'] = str(anonymous), str(pgdata)
        a.update(ns.directory_identity(anonymous))
        p.update(ns.directory_identity(pgdata))
        a['tree_sha256'] = ns.parent_only_tree(anonymous, p['destination'])
        runtime['mount_inventory_sha256'] = ns.contract.digest(ns.contract.canonical(runtime['mounts']))
        binds = []
        for name in req['tls_hashes']:
            path = source / name
            data = ('fixture-tls:' + name).encode()
            path.write_bytes(data)
            path.chmod(0o444)
            sha = ns.contract.digest(data)
            req['tls_hashes'][name] = sha
            binds.append(ns.Bind(path, '/tls/' + name, sha))
        for key, name in ns.INPUTS.items():
            path = source / name
            data = ('fixture-input:' + name).encode()
            path.write_bytes(data)
            path.chmod(0o444)
            sha = ns.contract.digest(data)
            req['pins'][key] = sha
            binds.append(ns.Bind(path, '/inputs/' + name, sha))
        hashes = {}
        for name in ns.MODULES:
            path = source / name
            data = ('# inert fixture closure:' + name).encode()
            path.write_bytes(data)
            path.chmod(0o444)
            sha = ns.contract.digest(data)
            hashes[name] = sha
            binds.append(ns.Bind(path, '/helper/' + name, sha))
        req['helper']['code_sha256'] = ns.contract.digest(ns.contract.canonical(hashes))
        req['receipt_bindings']['receipt_sha256']['a'] = req['pins']['aut26_receipt_sha256']
        req['receipt_bindings']['receipt_sha256']['l'] = req['pins']['login_receipt_sha256']
        req['normalizer_bundle'] = {name: ns.contract.digest(raw) for name, raw in FIXTURE_NORMALIZERS.items()}
        req['pins']['normalizer_sha256'] = ns.contract.digest(ns.contract.canonical(req['normalizer_bundle']))
        self.req, self.root = req, root
        self.layout = ns.Layout(state, sockpath, tuple(binds))
        self.backend = FakePodman(req, self.layout)
        # Simulate only PGDATA's remapped UID999 ownership; the sandbox cannot
        # chown a host subuid. File bytes, modes, inode identities and sockets are real fixtures.
        mapped = ns.mapped_id
        self.owner_patch = patch.object(ns, 'mapped_id', side_effect=lambda rows, inner:
            os.getuid() if inner == 999 and rows == self.req['runtime']['uid_map'] else
            os.getgid() if inner == 999 else mapped(rows, inner))
        self.owner_patch.start()
        self.addCleanup(self.owner_patch.stop)

    def observe_pg(self):
        return self.backend.observe(100, fixtures.T)

    def validate_pg(self, inspected=None, volume=None, image=None, observed=None):
        pg = self.req['pgid']
        return ns.validate_postgres(self.req, self.backend.pg if inspected is None else inspected,
            self.backend.volume(self.req['runtime']['mounts']['anonymous']['name']) if volume is None else volume,
            self.backend.image(pg['pg_image_id']) if image is None else image,
            self.observe_pg() if observed is None else observed)

    def test_complete_simulated_lifecycle(self):
        receipt = ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertEqual(self.backend.events, ['start', 'measure', 'sql-before', 'sql-after', 'close', 'confirm_close'])
        self.assertEqual(ns.contract.validate_measurement(receipt, self.req), receipt)
        self.assertEqual(receipt['physical']['before'], receipt['physical']['after'])
        self.assertEqual(receipt['result']['cas_applied'], False)
        self.assertTrue((self.layout.state / ns.ATTEMPT).is_file())
        self.assertEqual(ns.contract.decode((self.layout.state / ns.RECEIPT).read_bytes()), receipt)

    def test_original_helper_bytes_preserved_separately_from_canonical_hash(self):
        raw = json.dumps(self.backend.returned, indent=2).encode() + b'\n'
        self.backend.raw_result = raw
        receipt = ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertEqual(receipt['result_bytes_sha256'], ns.contract.digest(raw))
        self.assertNotEqual(receipt['result_bytes_sha256'], receipt['result_canonical_sha256'])
        self.assertEqual((self.layout.state / ns.RESULT).read_bytes(), raw)
        ns.contract.validate_measurement(receipt, self.req, result_raw=raw)

    def test_self_consistent_report_without_original_bytes_cannot_publish(self):
        self.backend.raw_result = deepcopy(self.backend.returned)
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertTrue((self.layout.state / ns.ATTEMPT).exists())
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())
        self.assertFalse((self.layout.state / ns.RESULT).exists())

    def test_failed_rollback_or_close_retains_attempt_without_receipt(self):
        for close_failure in (False, True):
            with self.subTest(close_failure=close_failure):
                # Every attempt has a new state; incomplete states are never retried.
                state = self.root / ('state-fail-' + str(close_failure))
                state.mkdir(mode=0o700)
                layout = ns.Layout(state, self.layout.controller_socket, self.layout.inputs)
                backend = FakePodman(self.req, layout)
                if close_failure:
                    backend.close_ok = False
                else:
                    backend.returned['rollback_confirmed'] = False
                with self.assertRaises(ns.Refused):
                    ns._measure_fixture(self.req, layout, backend)
                self.assertTrue((state / ns.ATTEMPT).is_file())
                self.assertFalse((state / ns.RECEIPT).exists())

    def test_partial_start_and_each_operation_failure_close_in_finally(self):
        for stage in ('start-partial', 'physical', 'measure', 'sql-before', 'sql-after'):
            with self.subTest(stage=stage):
                state = self.root / ('state-finally-' + stage)
                state.mkdir(mode=0o700)
                layout = ns.Layout(state, self.layout.controller_socket, self.layout.inputs)
                backend = FakePodman(self.req, layout)
                backend.fail_at = stage
                with self.assertRaises(RuntimeError) as caught:
                    ns._measure_fixture(self.req, layout, backend)
                self.assertIs(caught.exception, backend.failure)
                self.assertEqual(backend.events[-2:], ['close', 'confirm_close'])
                self.assertFalse(backend.helper_open)
                self.assertEqual(backend.open_session_count, 0)
                self.assertTrue((state / ns.ATTEMPT).exists())
                self.assertFalse((state / ns.RECEIPT).exists())

    def test_invalid_result_still_closes_and_does_not_publish(self):
        self.backend.raw_result = b'{}'
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertEqual(self.backend.events[-2:], ['close', 'confirm_close'])
        self.assertFalse(self.backend.helper_open)
        self.assertTrue((self.layout.state / ns.ATTEMPT).exists())
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())

    def test_cleanup_failure_preserves_primary_error_and_checks_closure(self):
        self.backend.fail_at = 'start-partial'
        self.backend.close_error = ValueError('fixture_close_failure')
        with self.assertRaises(RuntimeError) as caught:
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertIs(caught.exception, self.backend.failure)
        self.assertIn('podman_helper_close_unconfirmed', caught.exception.__notes__)
        self.assertEqual(self.backend.events[-2:], ['close', 'confirm_close'])
        self.assertTrue((self.layout.state / ns.ATTEMPT).exists())
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())

    def test_successful_close_without_confirmation_cannot_publish(self):
        self.backend.confirm_ok = False
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertEqual(self.backend.events[-2:], ['close', 'confirm_close'])
        self.assertTrue((self.layout.state / ns.ATTEMPT).exists())
        self.assertFalse((self.layout.state / ns.RESULT).exists())
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())

    def test_helper_report_cannot_replace_external_observations(self):
        self.backend.skip_observers = True
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertNotIn('sql-before', self.backend.events)
        self.assertEqual(self.backend.events[-2:], ['close', 'confirm_close'])
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())

    def test_external_sql_state_change_rejected_despite_old_helper_report(self):
        self.backend.sql_state['schema_sha256'] = '0' * 64
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertIn('sql-before', self.backend.events)
        self.assertNotIn('sql-after', self.backend.events)
        self.assertEqual(self.backend.events[-2:], ['close', 'confirm_close'])
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())

    def test_simulated_controller_timeout_at_second_barrier_closes_helper(self):
        self.backend.fail_at = 'sql-after'
        self.backend.failure = TimeoutError('fixture_controller_timeout')
        with self.assertRaises(TimeoutError) as caught:
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertIs(caught.exception, self.backend.failure)
        self.assertEqual(self.backend.events[-2:], ['close', 'confirm_close'])
        self.assertEqual(self.backend.open_session_count, 0)
        self.assertTrue((self.layout.state / ns.ATTEMPT).exists())
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())

    def test_both_normalizer_programs_pinned_before_start(self):
        for name in FIXTURE_NORMALIZERS:
            with self.subTest(name=name):
                self.backend.normalizers = deepcopy(FIXTURE_NORMALIZERS)
                self.backend.normalizers[name] += b'# changed\n'
                with self.assertRaises(ns.Refused):
                    ns._measure_fixture(self.req, self.layout, self.backend)
                self.assertEqual(self.backend.events, [])
                self.assertFalse((self.layout.state / ns.ATTEMPT).exists())

    def test_reordered_or_repeated_barriers_close_and_reject(self):
        self.backend.observer_order = ('after', 'before')
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertEqual(self.backend.events[-2:], ['close', 'confirm_close'])
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())

    def test_attempt_is_exclusive_and_cannot_be_replayed(self):
        ns._measure_fixture(self.req, self.layout, self.backend)
        self.backend.events.clear()
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertEqual(self.backend.events, [])
        (self.layout.state / ns.RECEIPT).unlink()
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)

    def test_namespace_pid_and_mount_drift_prevent_receipt(self):
        self.backend.drift = lambda process, mounts: process.update(starttick=7777)
        with self.assertRaises(ns.Refused):
            ns._measure_fixture(self.req, self.layout, self.backend)
        self.assertTrue((self.layout.state / ns.ATTEMPT).exists())
        self.assertFalse((self.layout.state / ns.RECEIPT).exists())

    def test_postgres_mounts_refuse_remote_volume_and_extra_mounts(self):
        changes = [('Anonymous', False), ('Driver', 'nfs'), ('Name', 'human-name'),
                   ('Options', {'device': 'remote'})]
        volume = self.backend.volume(self.req['runtime']['mounts']['anonymous']['name'])
        for key, replacement in changes:
            changed = deepcopy(volume)
            changed[key] = replacement
            with self.subTest(key=key), self.assertRaises(ns.Refused):
                self.validate_pg(volume=changed)
        changed = deepcopy(self.backend.pg)
        changed['Mounts'].append(deepcopy(changed['Mounts'][0]))
        with self.assertRaises(ns.Refused):
            self.validate_pg(inspected=changed)
        changed = deepcopy(self.backend.pg)
        changed['Mounts'][1]['RW'] = False
        with self.assertRaises(ns.Refused):
            self.validate_pg(inspected=changed)

    def test_extra_fields_are_rejected_before_projection(self):
        for key in ('Env', 'Binds', 'Tmpfs'):
            changed = deepcopy(self.backend.pg)
            changed['HostConfig'][key] = []
            with self.subTest(key=key), self.assertRaises(ns.Refused):
                self.validate_pg(inspected=changed)
        changed = deepcopy(self.backend.pg)
        changed['Mounts'][0]['Unexpected'] = True
        with self.assertRaises(ns.Refused):
            self.validate_pg(inspected=changed)

    def test_source_inode_and_effective_mount_inode_must_agree(self):
        observed = self.observe_pg()
        observed['effective_mounts'][self.req['pgid']['pgdata_bind_destination']]['ino'] += 1
        with self.assertRaises(ns.Refused):
            self.validate_pg(observed=observed)
        source = Path(self.req['pgid']['pgdata_bind_source'])
        source.rename(source.with_name('old-pgdata'))
        source.mkdir(mode=0o700)
        with self.assertRaises(ns.Refused):
            self.validate_pg()

    def test_anonymous_parent_tree_rejects_file_and_symlink(self):
        source = Path(self.req['runtime']['mounts']['anonymous']['source'])
        (source / 'unapproved').write_bytes(b'fixture')
        with self.assertRaises(ns.Refused):
            self.validate_pg()
        (source / 'unapproved').unlink()
        (source / '18' / 'docker').rmdir()
        (source / '18' / 'docker').symlink_to(self.root / 'h6-pgdata', target_is_directory=True)
        with self.assertRaises(OSError):
            self.validate_pg()

    def test_inputs_reject_hardlink_symlink_and_unreadable_remapped_user(self):
        bind = self.layout.inputs[0]
        os.link(bind.source, bind.source.with_name('extra-link'))
        with self.assertRaises(ns.Refused):
            ns.snapshot_inputs(self.req, self.layout)
        bind.source.with_name('extra-link').unlink()
        original = bind.source.with_name('old-input')
        bind.source.rename(original)
        bind.source.symlink_to(original)
        with self.assertRaises(OSError):
            ns.snapshot_inputs(self.req, self.layout)
        bind.source.unlink()
        original.rename(bind.source)
        bind.source.chmod(0o400)
        with self.assertRaises(ns.Refused):
            ns.snapshot_inputs(self.req, self.layout)

    def test_helper_effective_boundaries_not_only_requested_spec(self):
        inputs = ns.snapshot_inputs(self.req, self.layout)
        observed = self.backend.observe(101, fixtures.T)
        for area, key, replacement in [('Config', 'User', '0:0'),
                ('HostConfig', 'ReadonlyRootfs', False), ('HostConfig', 'CapAdd', ['SYS_ADMIN']),
                ('HostConfig', 'NetworkMode', 'host'), ('HostConfig', 'SecurityOpt', []),
                ('HostConfig', 'PidsLimit', 0)]:
            changed = deepcopy(self.backend.helper)
            changed[area][key] = replacement
            with self.subTest(area=area, key=key), self.assertRaises(ns.Refused):
                ns.validate_helper(self.req, changed, self.backend.helper_id, observed,
                                   self.layout.inputs, inputs)
        changed = deepcopy(self.backend.helper)
        changed['Mounts'][0]['Source'] = str(self.layout.controller_socket)
        with self.assertRaises(ns.Refused):
            ns.validate_helper(self.req, changed, self.backend.helper_id, observed,
                               self.layout.inputs, inputs)
        observed['effective_mounts']['/']['rw'] = True
        with self.assertRaises(ns.Refused):
            ns.validate_helper(self.req, self.backend.helper, self.backend.helper_id, observed,
                               self.layout.inputs, inputs)

    def test_controller_socket_authorized_private_and_local_only(self):
        for field, replacement in [('remote', True), ('rootless', False),
                                   ('socket_path', 'ssh://fixture'), ('owner_uid', 0)]:
            changed = self.backend.controller()
            changed[field] = replacement
            with self.subTest(field=field), self.assertRaises(ns.Refused):
                ns.validate_controller(self.req, self.layout.controller_socket, changed)
        self.layout.controller_socket.chmod(0o666)
        with self.assertRaises(ns.Refused):
            ns.validate_controller(self.req, self.layout.controller_socket, self.backend.controller())

    def test_image_id_is_not_repo_digest_or_tag(self):
        pg = self.req['pgid']
        for image_id in (pg['pg_repo_digest'], pg['pg_image'], 'd' * 64):
            with self.subTest(image_id=image_id), self.assertRaises(ns.Refused):
                ns.validate_image(self.backend.image(pg['pg_image_id']), image_id, pg['pg_repo_digest'])


class PureTests(unittest.TestCase):
    def test_gate_rejects_before_io_even_replaced_marker(self):
        for marker in (None, True, {'approved': True}, object()):
            with self.subTest(marker=type(marker)), patch.object(ns, 'EXECUTION_AUTHORITY', marker), \
                    patch.object(ns, 'directory', side_effect=AssertionError('unexpected IO')):
                with self.assertRaises(ns.Refused):
                    ns.acquire(object(), backend=object())
        with redirect_stdout(io.StringIO()) as output:
            self.assertEqual(ns.main(['--approved=true']), 1)
        self.assertIn('podman_execution_authority_pending', output.getvalue())

    def test_proc_parsers_reject_reused_ticks_conflicts_and_overlaps(self):
        raw = '100 (name with ) spaces) S ' + '0 ' * 18 + '12345 0'
        self.assertEqual(ns.process_starttick(raw), 12345)
        maps = ns.parse_id_map('0 1000 1\n1 100000 65536\n')
        self.assertEqual(ns.mapped_id(maps, 10002), 110001)
        for value in ('0 1000 2\n1 100000 100\n', '0 0 1', '0 1000 0', '0 1000 1 extra'):
            with self.subTest(value=value), self.assertRaises(ns.Refused):
                ns.parse_id_map(value)
        line = '36 25 0:31 / /tls/key.pem ro,nosuid - tmpfs tmpfs rw\n'
        self.assertEqual(ns.mount_permissions(line, ['/tls/key.pem']), {'/tls/key.pem': False})
        for value in (line + line, line.replace('ro,nosuid', 'ro,rw'), ''):
            with self.subTest(value=value), self.assertRaises(ns.Refused):
                ns.mount_permissions(value, ['/tls/key.pem'])


if __name__ == '__main__':
    unittest.main()
