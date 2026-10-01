"""Synthetic files and Docker/proc fixtures only; no Docker, SQL or TCP."""
import copy
from dataclasses import replace
import io
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import clon_cas_namespace as ns


class NamespaceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(dir=os.environ['TMPDIR'], prefix='namespace-')
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.state, self.tls, self.volume = (root / n for n in ('state', 'tls', 'volume'))
        for p in (self.state, self.tls, self.volume):
            p.mkdir(mode=0o700)
        self.cid, self.helper_cid = 'c' * 64, 'd' * 64
        self.pg_image, self.helper_image = 'sha256:' + 'a' * 64, 'sha256:' + 'b' * 64
        self.image_digest = 'local/nominal@sha256:' + 'e' * 64
        self.pg = ns.PGID(self.cid, self.pg_image, '12345', 5,
                Path('/dev/shm/vec-recorridos-fixture'), self.volume.stat().st_dev, self.volume.stat().st_ino)
        self.nominal = tuple(ns.Nominal(c, 'vec_ro_' + c, ns.digest((c + '-crt').encode()),
                ns.digest((c + '-key').encode())) for c in ns.CHANNELS)
        self.write(self.tls / 'ca.pem', b'fixture-ca')
        for n in self.nominal:
            self.write(self.tls / (n.channel + '.crt'), (n.channel + '-crt').encode())
            self.write(self.tls / (n.channel + '.key'), (n.channel + '-key').encode())
        restore = {'version': 1, 'kind': 'h1_restore_confirmed', 'estado_h1_sha': '1' * 64,
                'system_identifier': self.pg.system_identifier, 'database_name': 'postgres',
                'database_oid': 5, 'pg_container_id': self.cid, 'pg_image': 'postgres:18.4',
                'pg_image_id': self.pg_image, 'pg_volume': str(self.pg.volume),
                'schema_sha': '2' * 64, 'roles_sha': '3' * 64, 'datacl_sha': '4' * 64}
        self.write(self.state / 'h1-restore.json', self.canonical(restore))
        self.restore_hash = ns.digest(self.canonical(restore))
        run = 'f' * 32
        values = {'h1-restore-pending.json': {'version': 1, 'kind': 'h1_restore_pending',
                'run_id': run, 'estado_h1_sha': '1' * 64, 'pg_image_id': self.pg_image},
                'h1-volume.json': {'run_id': run, 'path': str(self.pg.volume),
                                  'device': self.pg.volume_device, 'inode': self.pg.volume_inode},
                'h1-container.json': {'run_id': run, 'pg_container_id': self.cid}}
        for name, value in values.items():
            self.write(self.state / name, self.canonical(value))
        records = {n: ns.digest((self.state / n).read_bytes()) for n in
                   ('h1-restore.json', 'h1-restore-pending.json', 'h1-volume.json', 'h1-container.json')}
        entries = [{'path': f'fixture/{i}.up.sql', 'sha256': '6' * 64,
                    'phase': 'H3' if i < 8 else 'H4' if i < 17 else 'H6'} for i in range(62)]
        journal = {'version': 2, 'identidad_clon': self.restore_hash, 'estado_h1_sha': '1' * 64,
                'package_sha': '7' * 64, 'list_sha': '8' * 64, 'release_sha': '9' * 64,
                'lock_sha': 'a' * 64, 'source_commit': ns.installer.SOURCE,
                'approved_sql_ref': ns.installer.SOURCE, 'plan_sha': ns.clon_sql.plan_hash(entries),
                'inventory_sha': 'b' * 64, 'entries': entries, 'file_count': 62,
                'plan_family': ns.clon_sql.H6_PACKAGE_FAMILY, 'revisions': [], 'pending': None,
                'phase': 'awaiting_ad132', 'installed': [{'position': i, **e,
                    'confirmed_at': '2026-10-01T00:00:00+00:00', 'confirmation': 'psql_exit0_observed'}
                    for i, e in enumerate(entries, 1)]}
        journal['journal_sha'] = ns.clon_sql.record_hash(journal)
        self.write(self.state / 'sql-journal.json', self.canonical(journal))
        self.request = ns.Request(self.state, self.pg, self.restore_hash,
                ns.digest(self.canonical(journal)), ns.digest(self.canonical(records)),
                self.helper_image, self.image_digest, ns.HELPER_SHA256, self.tls,
                ns.digest(b'fixture-ca'), 'pg-clone.local', self.nominal)
        self.pg_obj = {'Id': self.cid, 'Image': self.pg_image,
                'Config': {'Image': self.pg_image, 'Labels': {ns.clon_sql.OWNER_LABEL: 'Codex-M',
                                                          'vec.recorridos.state': str(self.state)}},
                'HostConfig': {'NetworkMode': 'none', 'Privileged': False, 'PortBindings': {}},
                'State': {'Running': True, 'Pid': 71, 'StartedAt': 'fixture-start'},
                'Mounts': [{'Type': 'bind', 'Source': str(self.pg.volume),
                           'Destination': '/var/lib/postgresql', 'RW': True}],
                'NetworkSettings': {'Ports': {'5432/tcp': None}}}
        self.helper_obj = {'Id': self.helper_cid, 'Image': self.helper_image,
                'Config': {'User': f'{os.getuid()}:{os.getgid()}', 'Entrypoint': ['/usr/bin/python3'],
                           'Cmd': ['-I', '-c', ns.BOOT]},
                'HostConfig': {'NetworkMode': 'container:' + self.cid, 'Privileged': False,
                    'ReadonlyRootfs': True, 'CapDrop': ['ALL'], 'SecurityOpt': ['no-new-privileges:true'],
                    'PidsLimit': 32, 'Memory': 128 * 1024**2, 'NanoCpus': 500_000_000,
                    'LogConfig': {'Type': 'none'}, 'PortBindings': {}},
                'State': {'Running': True, 'Pid': 72, 'StartedAt': 'fixture-start'},
                'Mounts': [{'Type': 'bind', 'Source': str(self.tls / name), 'Destination': '/tls/' + name,
                            'RW': False, 'Propagation': 'rprivate'} for name in ns.tls_hashes(self.request)],
                'NetworkSettings': {'Ports': {}}}
        self.hba = [{'type': 'hostssl', 'database': ['postgres'], 'user_name': [n.user],
                'address': '127.0.0.1', 'netmask': '255.255.255.255', 'auth_method': 'cert',
                'options': None, 'error': None} for n in self.nominal]
        self.calls = []
        self.read_effect = None
        self.preflight_effect = None
        self.preflight_calls = 0
        self.read_output = None
        self.start_output = (self.helper_cid + '\n').encode()
        self.remove_output = None
        self.net = {71: (100, 1, 42), 72: (200, 1, 42)}
        directory = ns.relay.directory
        self.patch(ns.relay, 'directory', side_effect=lambda p: directory(self.volume if p == self.pg.volume else p))
        self.physical_process_identity = ns.process_identity
        self.patch(ns, 'process_identity', side_effect=lambda p: self.net[p])
        self.runner = self.patch(ns.clon_sql, '_probe_command', side_effect=self.docker_command)

    def patch(self, obj, field, **kwargs):
        p = patch.object(obj, field, **kwargs)
        value = p.start()
        self.addCleanup(p.stop)
        return value

    @staticmethod
    def canonical(value):
        return ns.installer.clon_h6_kit.canonical(value)

    @staticmethod
    def write(path, data):
        path.write_bytes(data)
        path.chmod(0o600)

    def docker_command(self, argv, data=None, **kwargs):
        self.assertEqual(argv[:2], ['/usr/bin/docker', '--config'])
        self.assertEqual(argv[3:5], ['--host', 'unix:///var/run/docker.sock'])
        args = argv[5:]
        self.calls.append((args, data, kwargs))
        if args[:2] == ['image', 'inspect']:
            return self.canonical({'Id': self.helper_image, 'RepoDigests': [self.image_digest], 'Volumes': None})
        if args[:2] == ['container', 'inspect']:
            obj = self.pg_obj if args[-1] == self.cid else self.helper_obj
            return self.canonical(copy.deepcopy(obj))
        if args[0] == 'run':
            return self.start_output
        if args[0] == 'exec' and args[-1] == ns.READER:
            payload = json.loads(data)
            value = {'version': 1, 'nonce': payload['nonce'], 'tls_hashes': ns.tls_hashes(self.request),
                     'sessions': {n.channel: [[n.user, n.user, 'none', 'on', 1000 + i,
                        'postgres', 5, '12345', '127.0.0.1', '127.0.0.1', 5432, True, True, 'TLSv1.3', True]]
                        for i, n in enumerate(self.nominal)}}
            if self.read_effect:
                self.read_effect(value)
            return self.read_output if self.read_output is not None else self.canonical(value)
        if args[0] == 'exec':
            self.assertEqual(data, ('BEGIN READ ONLY; SET LOCAL statement_timeout=5000;\n' +
                                   ns.PREFLIGHT_SQL + ';\nROLLBACK;\n').encode())
            self.preflight_calls += 1
            value = {'ssl': 'on', 'configuration_loaded': 'fixture-loaded-before-hba',
                     'hba_loaded': True, 'system_identifier': '12345', 'database_oid': 5, 'hba': self.hba}
            if self.preflight_effect:
                self.preflight_effect(value)
            return self.canonical(value)
        if args[:2] == ['container', 'rm']:
            self.assertEqual(args, ['container', 'rm', '--force', self.helper_cid])
            return self.remove_output if self.remove_output is not None else (self.helper_cid + '\n').encode()
        raise AssertionError('Unapproved fixture Docker command')

    def run_attest(self, request=None):
        return ns.attest(request or self.request)

    def clear_attempt(self):
        for name in (ns.ATTEMPT, ns.RECEIPT):
            (self.state / name).unlink(missing_ok=True)
        self.calls.clear()

    def test_success_four_distinct_sessions_in_pinned_namespace_minimal_receipt(self):
        originals = {p.name: p.read_bytes() for p in self.state.iterdir()}
        result = self.run_attest()
        self.assertEqual(result['pg_container_id'], self.cid)
        self.assertIs(result['cas_sessions_bound'], False)
        self.assertEqual(set(result['sessions']), set(ns.CHANNELS))
        self.assertEqual(len({s['backend_pid'] for s in result['sessions'].values()}), 4)
        self.assertEqual(json.loads((self.state / ns.RECEIPT).read_bytes()), result)
        self.assertEqual((self.state / ns.RECEIPT).stat().st_mode & 0o777, 0o600)
        for name, data in originals.items():
            self.assertEqual((self.state / name).read_bytes(), data)
        self.assertNotIn('preimagen', json.dumps(result))
        self.assertNotIn('pg_volume', result)
        self.assertEqual(len([c for c in self.calls if c[0][0] == 'run']), 1)

    def test_run_is_no_host_port_nonroot_readonly_no_capabilities_minimal_bind(self):
        self.run_attest()
        args = next(a for a, _, _ in self.calls if a[0] == 'run')
        for option, value in [('--network', 'container:' + self.cid), ('--pull', 'never'),
                              ('--user', f'{os.getuid()}:{os.getgid()}'), ('--cap-drop', 'ALL'),
                              ('--security-opt', 'no-new-privileges:true'), ('--log-driver', 'none')]:
            if option == '--pull':
                self.assertIn('--pull=never', args)
            else:
                self.assertEqual(args[args.index(option) + 1], value)
        self.assertIn('--read-only', args)
        self.assertFalse(set(args) & {'--privileged', '-p', '--publish', '--publish-all', '--net=host'})
        mounts = [args[i + 1] for i, a in enumerate(args) if a == '--mount']
        self.assertEqual(len(mounts), 9)
        self.assertTrue(all('readonly' in m and 'rprivate' in m for m in mounts))
        self.assertNotIn(str(self.state), mounts)
        self.assertEqual(args[-4:], [self.image_digest, '-I', '-c', ns.BOOT])

    def test_replay_success_and_failed_attempt_never_open_another_session(self):
        self.run_attest()
        count = len(self.calls)
        with self.assertRaisesRegex(ns.Refused, 'namespace_attempt_exists'):
            self.run_attest()
        self.assertEqual(len(self.calls), count)
        self.clear_attempt()
        self.read_output = b'uncertain'
        with self.assertRaises(ns.Refused):
            self.run_attest()
        count = len(self.calls)
        with self.assertRaisesRegex(ns.Refused, 'namespace_attempt_exists'):
            self.run_attest()
        self.assertEqual(len(self.calls), count)
        self.assertFalse((self.state / ns.RECEIPT).exists())

    def test_missing_tls_or_hba_blocks_before_helper_creation(self):
        for effect in (lambda v: v.update(ssl='off'), lambda v: v.update(hba=[]),
                       lambda v: v.update(hba_loaded=False),
                       lambda v: v['hba'][0].update(auth_method='trust'),
                       lambda v: v['hba'][0].update(options=['map=foreign']),
                       lambda v: v['hba'][0].update(user_name=['all']),
                       lambda v: v['hba'][0].update(address='0.0.0.0')):
            original = copy.deepcopy(self.hba)
            self.preflight_effect = effect
            with self.subTest(effect=effect), self.assertRaises(ns.Refused):
                self.run_attest()
            self.assertFalse(any(a[0] == 'run' for a, _, _ in self.calls))
            self.assertFalse((self.state / ns.ATTEMPT).exists())
            self.hba = original
            self.calls.clear()

    def test_foreign_container_image_owner_network_ports_and_privilege_refused(self):
        cases = [('Id', 'f' * 64), ('Image', self.helper_image), ('Config.Labels.vec.recorridos.owner', 'foreign'),
                 ('HostConfig.NetworkMode', 'bridge'), ('HostConfig.Privileged', True),
                 ('HostConfig.PortBindings', {'5432/tcp': [{'HostPort': '55531'}]}),
                 ('NetworkSettings.Ports', {'5432/tcp': [{'HostIp': '127.0.0.1', 'HostPort': '55531'}]}),
                 ('Mounts', [])]
        for field, value in cases:
            original = copy.deepcopy(self.pg_obj)
            # Labels have dot-containing keys; handle that one explicitly.
            if field.endswith('vec.recorridos.owner'):
                self.pg_obj['Config']['Labels']['vec.recorridos.owner'] = value
            else:
                self.mutate(self.pg_obj, field, value)
            with self.subTest(field=field), self.assertRaises(ns.Refused):
                self.run_attest()
            self.assertFalse(any(a[0] == 'run' for a, _, _ in self.calls))
            self.pg_obj = original
            self.calls.clear()

    @staticmethod
    def mutate(obj, field, value):
        parts = field.split('.')
        for part in parts[:-1]:
            obj = obj[part]
        obj[parts[-1]] = value

    def test_helper_namespace_privileges_mounts_and_process_identity_refused(self):
        cases = [('Image', self.pg_image), ('HostConfig.NetworkMode', 'container:' + 'f' * 64),
                 ('HostConfig.Privileged', True),
                 ('HostConfig.ReadonlyRootfs', False), ('HostConfig.CapDrop', []),
                 ('HostConfig.CapAdd', ['SYS_ADMIN']), ('HostConfig.SecurityOpt', []),
                 ('HostConfig.PortBindings', {'5432/tcp': [{'HostPort': '5432'}]}),
                 ('HostConfig.PidMode', 'host'), ('HostConfig.Memory', 0),
                 ('Config.User', '0:0'), ('Mounts', [])]
        for field, value in cases:
            original = copy.deepcopy(self.helper_obj)
            self.mutate(self.helper_obj, field, value)
            with self.subTest(field=field), self.assertRaises(ns.Refused):
                self.run_attest()
            self.assertFalse((self.state / ns.RECEIPT).exists())
            self.assertFalse(any(a[0] == 'exec' and a[-1] == ns.READER for a, _, _ in self.calls))
            self.helper_obj = original
            self.clear_attempt()
        self.net[72] = (200, 1, 99)
        with self.assertRaisesRegex(ns.Refused, 'namespace_network_inode_mismatch'):
            self.run_attest()

    def test_restart_during_reader_changes_pg_or_helper_and_refuses_result(self):
        for effect in (lambda _: self.pg_obj['State'].update(StartedAt='restarted'),
                       lambda _: self.net.update({71: (101, 1, 42)}),
                       lambda _: self.net.update({72: (201, 1, 42)})):
            self.read_effect = effect
            with self.subTest(effect=effect), self.assertRaises(ns.Refused):
                self.run_attest()
            self.assertFalse((self.state / ns.RECEIPT).exists())
            self.clear_attempt()
            self.pg_obj['State']['StartedAt'] = 'fixture-start'
            self.net = {71: (100, 1, 42), 72: (200, 1, 42)}

    def test_configuration_reload_or_hba_change_during_reader_refuses_result(self):
        def reloaded(value):
            if self.preflight_calls == 2:
                value['configuration_loaded'] = 'fixture-reloaded'
        self.preflight_effect = reloaded
        with self.assertRaisesRegex(ns.Refused, 'namespace_configuration_changed'):
            self.run_attest()
        self.assertFalse((self.state / ns.RECEIPT).exists())

    def test_volume_replacement_and_tls_bytes_or_inode_replacement_refused(self):
        def replace_volume(_):
            self.volume.rename(self.volume.with_name('old-volume'))
            self.volume.mkdir(mode=0o700)
        self.read_effect = replace_volume
        with self.assertRaisesRegex(ns.Refused, 'namespace_volume_identity_drift'):
            self.run_attest()
        self.clear_attempt()
        self.volume.rmdir()
        self.volume.with_name('old-volume').rename(self.volume)
        for same_bytes in (False, True):
            path = self.tls / 'identidad.key'
            original = path.read_bytes()
            def replace_tls(_, same_bytes=same_bytes):
                old = path.with_suffix('.old')
                path.rename(old)
                self.write(path, original if same_bytes else b'altered-fixture')
                old.unlink()
            self.read_effect = replace_tls
            with self.subTest(same_bytes=same_bytes), self.assertRaises(ns.Refused):
                self.run_attest()
            self.assertFalse((self.state / ns.RECEIPT).exists())
            self.write(path, original)
            self.clear_attempt()

    def test_uncertain_or_mixed_session_outputs_no_receipt_no_provider_details(self):
        for output in (b'{}', b'{"version":1,"version":1}', b'not-json fixture-secret', b'[]'):
            self.read_output = output
            with self.subTest(output=output), self.assertRaises(ns.Refused) as found:
                self.run_attest()
            self.assertNotIn('fixture-secret', str(found.exception))
            self.assertFalse((self.state / ns.RECEIPT).exists())
            self.clear_attempt()
        self.read_output = None
        for column, replacement in ((0, 'postgres'), (3, 'off'), (7, '99999'), (8, None),
                                     (12, False), (13, 'TLSv1.1'), (14, False)):
            self.read_effect = lambda v, c=column, r=replacement: v['sessions']['identidad'][0].__setitem__(c, r)
            with self.subTest(column=column), self.assertRaises(ns.Refused):
                self.run_attest()
            self.clear_attempt()

    def test_external_pins_and_typed_api_refused_before_docker(self):
        for r in (replace(self.request, restore_sha256='0' * 64),
                  replace(self.request, journal_sha256='0' * 64),
                  replace(self.request, h1_records_sha256='0' * 64),
                  replace(self.request, helper_sha256='0' * 64),
                  replace(self.request, nominal=tuple(replace(n, user='postgres') for n in self.nominal)),
                  replace(self.request, pg=replace(self.pg, container_id='container-name'))):
            with self.subTest(request=r), self.assertRaises(ns.Refused):
                self.run_attest(r)
            self.assertEqual(self.calls, [])
        with self.assertRaises(TypeError):
            ns.attest(self.request, lambda: True)

    def test_uncertain_docker_run_does_not_adopt_or_delete_a_named_container(self):
        self.start_output = b'partial-id'
        with self.assertRaisesRegex(ns.Refused, 'namespace_helper_creation_uncertain'):
            self.run_attest()
        self.assertFalse(any(a[:2] == ['container', 'rm'] for a, _, _ in self.calls))
        self.assertTrue((self.state / ns.ATTEMPT).exists())

    def test_uncertain_removal_is_not_a_success_receipt(self):
        self.remove_output = b'foreign-container'
        with self.assertRaisesRegex(ns.Refused, 'namespace_helper_removal_uncertain'):
            self.run_attest()
        self.assertTrue((self.state / ns.ATTEMPT).exists())
        self.assertFalse((self.state / ns.RECEIPT).exists())

    def test_linux_process_identity_rejects_pid_reuse_during_namespace_observation(self):
        def proc(start):
            return '71 (synthetic reader) ' + ' '.join(['S'] + ['0'] * 18 + [str(start)])
        with patch.object(ns, 'Path') as path, patch.object(ns.os, 'stat') as metadata:
            path.return_value.read_text.side_effect = [proc(100), proc(100)]
            metadata.return_value = SimpleNamespace(st_dev=1, st_ino=42)
            self.assertEqual(self.physical_process_identity(71), (100, 1, 42))
            path.return_value.read_text.side_effect = [proc(100), proc(101)]
            with self.assertRaisesRegex(ns.Refused, 'namespace_process_missing'):
                self.physical_process_identity(71)

    def test_reader_opens_four_fixed_tls_readonly_channels_and_closes_before_output(self):
        events, connections, parameters = [], [], []
        outer = self
        class Connection:
            def __init__(self, user):
                self.user = user
                self.pid = 2000 + len(connections)
                connections.append(self)
            def __enter__(self):
                return self
            def __exit__(self, *args):
                events.append(('closed', self.user))
            def rollback(self):
                events.append(('rollback', self.user))
            def cursor(self):
                return Cursor(self)
        class Cursor:
            def __init__(self, connection):
                self.connection = connection
            def __enter__(self):
                return self
            def __exit__(self, *args):
                pass
            def execute(self, sql):
                events.append(('sql', self.connection.user, sql))
            def fetchall(self):
                return [(self.connection.user, self.connection.user, 'none', 'on', self.connection.pid,
                         'postgres', 5, '12345', '127.0.0.1', '127.0.0.1', 5432, True, True, 'TLSv1.3', True)]
        def connect(**values):
            parameters.append(values)
            return Connection(values['user'])
        payload = {'nonce': 'f' * 64, 'tls_hashes': ns.tls_hashes(self.request),
                   'server_name': self.request.server_name, 'users': {n.channel: n.user for n in self.nominal}}
        output = io.StringIO()
        path_type = Path
        def fixture_path(path):
            outer.assertTrue(path.startswith('/tls/'))
            return outer.tls / path_type(path).name
        # Execute only our fixed source with a pure driver, inside the test sandbox.
        with patch.dict('sys.modules', {'psycopg': SimpleNamespace(connect=connect)}), \
             patch('sys.stdin', SimpleNamespace(buffer=io.BytesIO(self.canonical(payload)))), \
             patch('sys.stdout', output), patch('pathlib.Path', side_effect=fixture_path):
            exec(compile(ns.READER, '<fixed-nominal-reader>', 'exec'), {})
        value = json.loads(output.getvalue())
        self.assertEqual(set(value['sessions']), set(ns.CHANNELS))
        self.assertEqual(len(parameters), 4)
        for p in parameters:
            self.assertEqual((p['hostaddr'], p['port'], p['dbname']), ('127.0.0.1', 5432, 'postgres'))
            self.assertEqual(p['host'], self.request.server_name)
            self.assertEqual(p['sslmode'], 'verify-full')
            self.assertEqual(p['sslrootcert'], '/tls/ca.pem')
            self.assertIs(p['autocommit'], True)
            self.assertIn('default_transaction_read_only=on', p['options'])
        self.assertEqual(len([e for e in events if e[0] == 'closed']), 4)
        self.assertEqual(len([e for e in events if e[0] == 'sql' and e[2] == 'BEGIN READ ONLY']), 4)
        self.assertEqual(len([e for e in events if e[0] == 'sql' and e[2] == 'ROLLBACK']), 4)

    def test_journal_must_be_completed_sql62_with_no_pending_and_original_restore(self):
        original = (self.state / 'sql-journal.json').read_bytes()
        for key, replacement in (('phase', 'ad132_confirmed'), ('pending', {'position': 62}),
                                 ('file_count', 61), ('identidad_clon', '0' * 64),
                                 ('installed', []), ('source_commit', '0' * 40)):
            journal = json.loads(original)
            journal[key] = replacement
            journal['journal_sha'] = ns.clon_sql.record_hash(journal)
            raw = self.canonical(journal)
            self.write(self.state / 'sql-journal.json', raw)
            with self.subTest(key=key), self.assertRaises(ns.Refused):
                self.run_attest(replace(self.request, journal_sha256=ns.digest(raw)))
            self.assertEqual(self.calls, [])
        self.write(self.state / 'sql-journal.json', original)

    def test_docker_configuration_replacement_and_extra_tls_bind_material_refused(self):
        self.write(self.tls / 'unapproved.pem', b'fixture-extra')
        with self.assertRaises(ns.Refused):
            self.run_attest()
        self.assertEqual(self.calls, [])
        (self.tls / 'unapproved.pem').unlink()
        def tamper(_):
            config = next(self.state.glob('cas-docker-*'))
            self.write(config / 'config.json', b'{"credsStore":"foreign"}')
        self.read_effect = tamper
        with self.assertRaisesRegex(ns.Refused, 'namespace_docker_config_drift'):
            self.run_attest()
        self.assertFalse((self.state / ns.RECEIPT).exists())


if __name__ == '__main__':
    unittest.main()
