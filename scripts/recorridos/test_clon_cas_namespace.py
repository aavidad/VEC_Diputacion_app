"""Synthetic files and Docker/proc fixtures only; no Docker, SQL or TCP."""
import copy
from dataclasses import replace
import io
import json
import os
from pathlib import Path
import stat
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
        self.state, self.tls, self.volume = (root / n for n in ('state', 'tls', 'vec-recorridos-fixture'))
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
                'HostConfig': {'NetworkMode': 'none', 'Privileged': False, 'PortBindings': {},
                    'Tmpfs': {'/var/run/postgresql': 'rw,noexec,nosuid,size=8m,uid=999,gid=999,mode=0700',
                              '/tmp': 'rw,noexec,nosuid,size=16m,uid=999,gid=999,mode=0700'}},
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
        self.patch(ns.relay, 'directory', side_effect=lambda p: directory(self.volume.parent
                   if p == self.pg.volume.parent else p))
        self.volume_owner = self.volume_group = 999
        self.volume_permissions = None
        self.volume_open_flags = []
        original_stat, original_fstat, original_open = os.stat, os.fstat, os.open
        def volume_metadata(info):
            # A rootless user namespace cannot chown to an unmapped host UID999.
            # Simulate that ownership only for this synthetic volume, retaining
            # its real kernel dev/ino/mode and exercising the actual O_PATH open.
            try:
                current = original_stat(self.volume, follow_symlinks=False)
            except FileNotFoundError:
                return info
            if (info.st_dev, info.st_ino) != (current.st_dev, current.st_ino):
                return info
            return SimpleNamespace(**{name: self.volume_owner if name == 'st_uid' else
                                           self.volume_group if name == 'st_gid' else
                                           stat.S_IFDIR | self.volume_permissions
                                           if name == 'st_mode' and self.volume_permissions is not None
                                           else getattr(info, name)
                                      for name in dir(info) if name.startswith('st_')})
        def fixture_open(path, flags, *args, **kwargs):
            if path == self.pg.volume.name and kwargs.get('dir_fd') is not None:
                self.volume_open_flags.append(flags)
                self.assertTrue(flags & os.O_DIRECTORY)
                self.assertTrue(flags & os.O_NOFOLLOW)
                if not flags & os.O_PATH:
                    raise PermissionError('fixture PGDATA is UID999:999/0700')
            return original_open(path, flags, *args, **kwargs)
        self.patch(ns.os, 'open', side_effect=fixture_open)
        self.patch(ns.os, 'stat', side_effect=lambda *a, **kw: volume_metadata(original_stat(*a, **kw)))
        self.patch(ns.os, 'fstat', side_effect=lambda *a: volume_metadata(original_fstat(*a)))
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

    def test_uid999_pgdata_is_observed_with_opath_without_read_permission(self):
        self.assertNotEqual(os.getuid(), self.volume_owner)
        self.run_attest()
        self.assertTrue(self.volume_open_flags)
        self.assertTrue(all(flags & os.O_PATH and flags & os.O_DIRECTORY
                            and flags & os.O_NOFOLLOW for flags in self.volume_open_flags))
        # Every fixture open without O_PATH raises PermissionError, so a fallback
        # to O_RDONLY would fail this otherwise valid H1 attestation.
        self.assertEqual(len(self.pg_obj['Mounts']), 1)
        self.assertEqual(set(self.pg_obj['HostConfig']['Tmpfs']), {'/var/run/postgresql', '/tmp'})

    def test_volume_owner_group_or_mode_drift_is_refused_before_helper(self):
        for field, value in (('volume_owner', 1000), ('volume_group', 1000), ('volume_permissions', 0o755)):
            previous = getattr(self, field)
            setattr(self, field, value)
            with self.subTest(field=field), self.assertRaisesRegex(ns.Refused, 'namespace_volume_identity_drift'):
                self.run_attest()
            self.assertFalse(any(a[0] == 'run' for a, _, _ in self.calls))
            setattr(self, field, previous)
            self.calls.clear()

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


class MeasurementTests(unittest.TestCase):
    """Host orchestration fixtures; the production D authority gate stays shut."""

    patch = NamespaceTests.patch
    canonical = staticmethod(NamespaceTests.canonical)
    write = staticmethod(NamespaceTests.write)
    mutate = staticmethod(NamespaceTests.mutate)

    def setUp(self):
        NamespaceTests.setUp(self)
        root = Path(self.temp.name)
        self.inputs, self.modules = root / 'inputs', root / 'modules'
        self.inputs.mkdir(mode=0o700)
        self.modules.mkdir(mode=0o700)
        sources = {name[:-3]: '# inert synthetic module\n' + 'fixture=' + repr(name) + '\n'
                   for name in ns.CAS_MODULES}
        for name in ns.CAS_MODULES:
            self.write(self.modules / name, sources[name[:-3]].encode())
        # Fixed root is replaced only inside this test's no-network fixture.
        self.patch(ns, '__file__', new=str(self.modules / 'clon_cas_namespace.py'))
        self.program = ns._program(sources)
        nominal = tuple(replace(n, user='vec_externo_v3_fuente_autorizacion_desarrollo'
                                if n.channel == 'autorizacion' else n.user)
                        for n in self.nominal if n.channel != 'motivos')
        (self.tls / 'motivos.crt').unlink()
        (self.tls / 'motivos.key').unlink()
        self.nominal = nominal
        namespace = replace(self.request, nominal=nominal,
                            helper_sha256=ns.digest((ns.BOOT + '\n' + self.program).encode()))
        records = {name: ns.digest((self.state / name).read_bytes()) for name in
                   ('h1-restore.json', 'h1-restore-pending.json', 'h1-volume.json', 'h1-container.json')}
        for name in ns.CAS_INPUTS:
            data = (self.state / name).read_bytes() if (self.state / name).exists() else self.canonical(
                records if name == 'h1-records.json' else {'fixture': name})
            self.write(self.inputs / name, data)
        self.measurement = ns.MeasurementRequest(namespace,
            tuple(ns.FilePin(name, ns.digest((self.modules / name).read_bytes())) for name in ns.CAS_MODULES),
            self.inputs, tuple(ns.FilePin(name, ns.digest((self.inputs / name).read_bytes())) for name in ns.CAS_INPUTS),
            tuple(ns.Target(population, 'cta_' + 'a' * 32, 'per_' + 'b' * 32, 'prf_' + 'c' * 32,
                            ('pce_' if population == 'candidato' else 'pue_') + 'd' * 32)
                  for population in ('candidato', 'usuarios')))
        self.helper_obj['Mounts'] = [{'Type': 'bind', 'Source': source, 'Destination': destination,
                                     'RW': False, 'Propagation': 'rprivate'}
                                    for source, destination in ns.measurement_mounts(self.measurement).items()]
        self.hba = [dict(row, user_name=[nominal[i].user]) for i, row in enumerate(self.hba[:3])]
        self.measure_effect = None
        self.measure_output = None

    def docker_command(self, argv, data=None, **kwargs):
        if argv[-1] == self.program:
            args = argv[5:]
            self.calls.append((args, data, kwargs))
            self.assertEqual(args[:3], ['exec', '-i', '--user'])
            self.assertEqual(args[4:8], [self.helper_cid, '/usr/bin/python3', '-I', '-c'])
            payload = json.loads(data)
            self.assertEqual(set(payload['users']), set(ns.CAS_CHANNELS))
            value = self.result(payload)
            if self.measure_effect:
                self.measure_effect(value)
            return self.measure_output if self.measure_output is not None else self.canonical(value)
        return NamespaceTests.docker_command(self, argv, data, **kwargs)

    @staticmethod
    def result(payload):
        sessions = {}
        for i, (channel, user) in enumerate(payload['users'].items()):
            role = ns.CAS_ROLES[channel]
            row = {'session_user': user, 'current_user': user if role == 'none' else role,
                'role': role, 'transaction_read_only': 'on', 'transaction_isolation': 'read committed',
                'backend_pid': 1000 + i, 'database_name': 'postgres', 'database_oid': payload['pgid']['database_oid'],
                'system_identifier': payload['pgid']['system_identifier'], 'client_addr': '127.0.0.1',
                'server_addr': '127.0.0.1', 'server_port': 5432, 'login_safe': True,
                'ssl': True, 'tls_version': 'TLSv1.3', 'client_certificate': True}
            sessions[channel] = {'before': row, 'after': copy.deepcopy(row)}
        return {'version': 1, 'kind': 'cas_helper_result_v1',
            **{key: copy.deepcopy(payload[key]) for key in ('nonce', 'pgid', 'pins', 'tls_hashes')},
            'sessions': sessions, 'cas_sessions_bound': True, 'cas_applied': False, 'rollback_confirmed': True,
            'readings_sha256': ['a' * 64, 'a' * 64],
            'observations': {population: {'lectura_1': '2026-10-01T02:00:00Z',
                            'lectura_2': '2026-10-01T02:00:01Z'} for population in ('candidato', 'motivos')},
            'preimages': {'candidato': {'revision_control_rol': 0, 'huella_control_rol': '',
                'version_asignacion': 0, 'huella_asignacion': '', 'version_contexto': 0,
                'huella_contexto': '', 'secuencia_motivos': 0},
                'usuarios': {'version_contexto': 0, 'huella_contexto': ''}}}

    def synthetic_measure(self, request=None):
        # Only this fixture bypasses the missing D contract; no runtime knob.
        with patch.object(ns, 'D_RECEIPT_CONTRACT', {'synthetic_fixture': True}), \
             patch.object(ns, '_require_measurement_authority', return_value=None):
            return ns.measure(request or self.measurement)

    def clear_measurement(self):
        for name in (ns.CAS_ATTEMPT, ns.CAS_RECEIPT):
            (self.state / name).unlink(missing_ok=True)
        self.calls.clear()

    def test_missing_d_authority_cannot_be_enabled_by_caller_files_or_placeholder(self):
        for authority in (None, True, {'approved': True}):
            with self.subTest(authority=authority), patch.object(ns, 'D_RECEIPT_CONTRACT', authority):
                with self.assertRaisesRegex(ns.Refused, 'cas_execution_authority_pending'):
                    ns.measure(self.measurement)
            self.assertEqual(self.calls, [])
            self.assertFalse((self.state / ns.CAS_ATTEMPT).exists())
            self.assertFalse((self.state / ns.CAS_RECEIPT).exists())

    def test_absent_d_contract_refuses_before_any_private_io_or_lock(self):
        missing = replace(self.measurement, namespace=replace(self.measurement.namespace,
                          state=Path(self.temp.name) / 'absent-state'))
        with patch.object(ns, 'D_RECEIPT_CONTRACT', None), \
             patch.object(ns.installer, 'private_state') as private_state, \
             patch.object(ns.installer, 'installer_lock') as lock, \
             patch.object(ns.installer, 'read_owned') as read_owned, \
             patch.object(ns.relay, 'read_file') as read_module, \
             patch.object(ns, 'Docker') as docker:
            with self.assertRaisesRegex(ns.Refused, 'cas_execution_authority_pending'):
                ns.measure(missing)
            for boundary in (private_state, lock, read_owned, read_module, docker):
                boundary.assert_not_called()
        self.assertFalse(missing.namespace.state.exists())
        self.assertFalse((self.state / 'sql62-instalar.lock').exists())
        self.assertEqual(self.calls, [])

    def test_separate_success_receipt_pins_entire_helper_and_keeps_historical_false(self):
        historical = self.canonical({'kind': 'nominal_ro_namespace_sessions', 'cas_sessions_bound': False})
        self.write(self.state / ns.RECEIPT, historical)
        originals = {p.name: p.read_bytes() for p in self.state.iterdir() if p.is_file()}
        receipt = self.synthetic_measure()
        self.assertIs(receipt['cas_sessions_bound'], True)
        self.assertIs(receipt['cas_applied'], False)
        self.assertEqual((self.state / ns.RECEIPT).read_bytes(), historical)
        for name, data in originals.items():
            self.assertEqual((self.state / name).read_bytes(), data)
        self.assertEqual(receipt['helper_sha256'], self.measurement.namespace.helper_sha256)
        self.assertEqual(set(receipt['module_sha256']), set(ns.CAS_MODULES))
        self.assertEqual(receipt['helper_container_id'], self.helper_cid)
        self.assertEqual(set(receipt['result']['sessions']), set(ns.CAS_CHANNELS))
        self.assertEqual(len({v['before']['backend_pid'] for v in receipt['result']['sessions'].values()}), 3)
        self.assertEqual(receipt, json.loads((self.state / ns.CAS_RECEIPT).read_bytes()))
        run = next(args for args, _, _ in self.calls if args[0] == 'run')
        self.assertEqual(run[run.index('--network') + 1], 'container:' + self.cid)
        self.assertFalse(set(run) & {'-p', '--publish', '--privileged', '--net=host'})
        mounts = [run[i + 1] for i, arg in enumerate(run) if arg == '--mount']
        self.assertEqual(len(mounts), 14)
        self.assertTrue(all('readonly' in value and 'rprivate' in value for value in mounts))
        self.assertFalse(any(str(self.modules) in value for value in mounts))
        self.assertEqual(len([a for a, _, _ in self.calls if a[0] == 'exec' and a[-1] == self.program]), 1)

    def test_replayed_success_or_uncertain_attempt_never_opens_new_helper(self):
        self.synthetic_measure()
        self.calls.clear()
        with self.assertRaisesRegex(ns.Refused, 'cas_attempt_exists'):
            self.synthetic_measure()
        self.assertEqual(self.calls, [])
        self.clear_measurement()
        self.measure_output = b'incomplete fixture-secret'
        with self.assertRaises(ns.Refused) as caught:
            self.synthetic_measure()
        self.assertNotIn('fixture-secret', str(caught.exception))
        self.assertTrue((self.state / ns.CAS_ATTEMPT).exists())
        self.assertFalse((self.state / ns.CAS_RECEIPT).exists())
        self.calls.clear()
        with self.assertRaisesRegex(ns.Refused, 'cas_attempt_exists'):
            self.synthetic_measure()
        self.assertEqual(self.calls, [])

    def test_pending_is_durable_before_any_helper_creation(self):
        original = self.runner.side_effect
        def command(argv, data=None, **kwargs):
            if 'run' in argv:
                marker = json.loads((self.state / ns.CAS_ATTEMPT).read_bytes())
                self.assertEqual(marker['kind'], 'cas_measurement_pending_v1')
                self.assertEqual(marker['helper_sha256'], self.measurement.namespace.helper_sha256)
                self.assertEqual((self.state / ns.CAS_ATTEMPT).stat().st_mode & 0o777, 0o600)
            return original(argv, data, **kwargs)
        self.runner.side_effect = command
        self.start_output = b'partial-id'
        with self.assertRaisesRegex(ns.Refused, 'namespace_helper_creation_uncertain'):
            self.synthetic_measure()
        self.assertTrue((self.state / ns.CAS_ATTEMPT).exists())
        self.assertFalse(any(args[:2] == ['container', 'rm'] for args, _, _ in self.calls))

    def test_module_and_complete_program_hash_drift_refuse_before_docker(self):
        for pin in ns.CAS_MODULES:
            path = self.modules / pin
            original = path.read_bytes()
            self.write(path, original + b'# altered\n')
            with self.assertRaisesRegex(ns.Refused, 'cas_helper_module_drift'):
                self.synthetic_measure()
            self.assertEqual(self.calls, [])
            self.write(path, original)
        bad = replace(self.measurement, namespace=replace(self.measurement.namespace, helper_sha256='0' * 64))
        with self.assertRaisesRegex(ns.Refused, 'cas_helper_bundle_drift'):
            self.synthetic_measure(bad)
        self.assertEqual(self.calls, [])

    def test_module_bytes_or_inode_substitution_during_reader_refuse_receipt(self):
        path = self.modules / ns.CAS_MODULES[0]
        original = path.read_bytes()
        for same_bytes in (False, True):
            def mutate_module(value):
                path.rename(path.with_suffix('.old'))
                self.write(path, original if same_bytes else original + b'changed\n')
            self.measure_effect = mutate_module
            with self.subTest(same_bytes=same_bytes), self.assertRaises(ns.Refused):
                self.synthetic_measure()
            self.assertFalse((self.state / ns.CAS_RECEIPT).exists())
            path.unlink()
            path.with_suffix('.old').rename(path)
            self.clear_measurement()

    def test_three_login_roles_and_closed_output_are_required(self):
        effects = [lambda v: v['sessions'].update(motivos=v['sessions']['autorizacion']),
                   lambda v: v.update(rollback_confirmed=False), lambda v: v.update(cas_applied=True),
                   lambda v: v.update(cas_sessions_bound=False), lambda v: v.update(extra='private'),
                   lambda v: v['pins'].update(login_receipt_sha256='f' * 64),
                   lambda v: v['readings_sha256'].__setitem__(1, 'b' * 64),
                   lambda v: v['preimages']['usuarios'].update(alias='private')]
        for field, replacement in (('role', 'postgres'), ('current_user', 'postgres'),
                                   ('session_user', 'vec_foreign'), ('ssl', False),
                                   ('login_safe', False), ('transaction_read_only', 'off'),
                                   ('transaction_isolation', 'repeatable read'), ('backend_pid', 1001)):
            effects.append(lambda v, f=field, r=replacement:
                           v['sessions']['identidad']['after'].__setitem__(f, r))
        for effect in effects:
            self.measure_effect = effect
            with self.subTest(effect=effect), self.assertRaises(ns.Refused):
                self.synthetic_measure()
            self.assertFalse((self.state / ns.CAS_RECEIPT).exists())
            self.assertTrue((self.state / ns.CAS_ATTEMPT).exists())
            self.clear_measurement()

    def test_same_backend_pid_on_two_channels_is_refused(self):
        def shared(value):
            pid = value['sessions']['identidad']['before']['backend_pid']
            for row in value['sessions']['contexto'].values():
                row['backend_pid'] = pid
        self.measure_effect = shared
        with self.assertRaisesRegex(ns.Refused, 'cas_sessions_not_distinct'):
            self.synthetic_measure()

    def test_pg_helper_restart_or_namespace_change_refuses_output(self):
        effects = [lambda _: self.pg_obj['State'].update(StartedAt='restarted'),
                   lambda _: self.helper_obj['State'].update(StartedAt='restarted'),
                   lambda _: self.net.update({71: (101, 1, 42)}),
                   lambda _: self.net.update({72: (201, 1, 42)}),
                   lambda _: self.net.update({72: (200, 1, 99)}),
                   lambda _: self.helper_obj.update(Id='e' * 64),
                   lambda _: self.helper_obj['HostConfig'].update(NetworkMode='container:' + 'e' * 64)]
        for effect in effects:
            pg, helper, net = copy.deepcopy(self.pg_obj), copy.deepcopy(self.helper_obj), dict(self.net)
            self.measure_effect = effect
            with self.subTest(effect=effect), self.assertRaises(ns.Refused):
                self.synthetic_measure()
            self.assertFalse((self.state / ns.CAS_RECEIPT).exists())
            self.pg_obj, self.helper_obj, self.net = pg, helper, net
            self.clear_measurement()

    def test_hba_permission_mount_and_input_pin_fail_closed(self):
        self.hba[0]['auth_method'] = 'trust'
        with self.assertRaises(ns.Refused):
            self.synthetic_measure()
        self.assertFalse(any(a[0] == 'run' for a, _, _ in self.calls))
        self.calls.clear()
        self.hba[0]['auth_method'] = 'cert'
        self.helper_obj['HostConfig']['CapAdd'] = ['SYS_ADMIN']
        with self.assertRaises(ns.Refused):
            self.synthetic_measure()
        self.assertFalse(any(a[0] == 'exec' and a[-1] == self.program for a, _, _ in self.calls))
        self.clear_measurement()
        self.helper_obj['HostConfig']['CapAdd'] = None
        for name in ('source.json', 'aut26-receipt.json', 'login-receipt.json'):
            path = self.inputs / name
            data = path.read_bytes()
            self.write(path, b'{}')
            with self.assertRaisesRegex(ns.Refused, 'cas_input_pin_drift'):
                self.synthetic_measure()
            self.assertEqual(self.calls, [])
            self.write(path, data)

    def test_input_substitution_and_incomplete_output_preserve_uncertain_pending(self):
        path = self.inputs / 'alias.json'
        def replacement(value):
            path.rename(path.with_suffix('.old'))
            self.write(path, path.with_suffix('.old').read_bytes())
        self.measure_effect = replacement
        with self.assertRaises(ns.Refused):
            self.synthetic_measure()
        self.assertFalse((self.state / ns.CAS_RECEIPT).exists())
        self.assertTrue((self.state / ns.CAS_ATTEMPT).exists())
        self.clear_measurement()
        path.with_suffix('.old').unlink()
        self.measure_effect = None
        for output in (b'{}', b'[]', b'{"version":1,"version":1}', b'partial'):
            self.measure_output = output
            with self.subTest(output=output), self.assertRaises(ns.Refused):
                self.synthetic_measure()
            self.assertFalse((self.state / ns.CAS_RECEIPT).exists())
            self.clear_measurement()

    def test_typed_api_has_no_sql_command_callback_or_caller_module_path(self):
        with self.assertRaises(TypeError):
            ns.measure(self.measurement, lambda: True)
        for altered in (replace(self.measurement, targets=()),
                        replace(self.measurement, module_pins=(ns.FilePin('/tmp/user.py', 'a' * 64),)),
                        replace(self.measurement, namespace=replace(self.measurement.namespace, nominal=self.request.nominal))):
            with self.assertRaises(ns.Refused):
                self.synthetic_measure(altered)
            self.assertEqual(self.calls, [])


if __name__ == '__main__':
    unittest.main()
