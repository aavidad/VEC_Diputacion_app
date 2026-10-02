"""Pure archive and Docker contract tests; never launch Docker or restore H1."""

import contextlib
import hashlib
import io
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

import clon_h1_restore as restore
import clon_sql


class FakeDocker:
    instance = None
    control_state = 'shut down'
    wrong_image = False
    cid = 'c' * 64

    def __init__(self, state):
        self.state, self.calls = state, []
        FakeDocker.instance = self

    def run(self, *args, stdin=None, timeout=180):
        self.calls.append((args, stdin is not None))
        if args[:2] == ('image', 'inspect'):
            return json.dumps([{'Id': 'sha256:' + 'f' * 64 if self.wrong_image else restore.IMAGE_ID}])
        if args[0] == 'run':
            if '/usr/bin/id' in args:
                return '999'
            if 'pg_controldata' in args:
                return 'Database cluster state: ' + self.control_state + '\nDatabase system identifier: 12345'
            assert stdin is not None
            with tarfile.open(fileobj=stdin, mode='r:') as archive:
                self.normalized = [(m.name, m.type, m.uid, m.gid, m.mode) for m in archive]
            return ''
        if args[:2] == ('container', 'create'):
            self.create_args = args
            return self.cid
        if args[:2] == ('container', 'start'):
            return self.cid
        if args[:2] == ('container', 'inspect'):
            arguments = self.create_args
            self.volume = next(a.split(',source=', 1)[1].split(',destination=', 1)[0]
                               for a in arguments if a.startswith('type=bind,'))
            labels = dict(a.split('=', 1) for a in arguments if a.startswith(('vec.recorridos.', 'vec.clon.')))
            return json.dumps([{'Id': self.cid, 'Image': restore.IMAGE_ID, 'State': {'Running': True},
                'Config': {'Labels': labels, 'User': '999:999', 'Entrypoint': ['postgres'], 'Cmd': restore.PG_COMMAND},
                'HostConfig': {'NetworkMode': 'none', 'PortBindings': {}, 'ReadonlyRootfs': True,
                               'Privileged': False, 'CapDrop': ['ALL'], 'CapAdd': None, 'LogConfig': {'Type': 'none'}},
                'NetworkSettings': {'Ports': {}},
                'Mounts': [{'Type': 'bind', 'Source': self.volume, 'Destination': '/var/lib/postgresql', 'RW': True}]}])
        if args[0] == 'exec':
            return 'ready'
        raise AssertionError(args)


def fixture(extra=()):
    members = [('.', None), ('./18', None), ('./18/docker', None),
               ('./18/docker/global', None), ('./18/docker/global/pg_control', b'\0' * 8192),
               ('./18/docker/PG_VERSION', b'18\n'),
               ('./18/docker/postgresql.conf', b"listen_addresses = '*'\n"), *extra]
    stream = io.BytesIO()
    with tarfile.open(fileobj=stream, mode='w:gz') as archive:
        for name, data in members:
            if isinstance(name, tarfile.TarInfo):
                archive.addfile(name, io.BytesIO(data) if data else None)
                continue
            member = tarfile.TarInfo(name)
            member.type = tarfile.DIRTYPE if data is None else tarfile.REGTYPE
            member.mode = 0o700 if data is None else 0o600
            member.uid = member.gid = 999
            member.size = len(data) if data is not None else 0
            archive.addfile(member, None if data is None else io.BytesIO(data))
    stream.seek(0)
    return stream


class RestoreTests(unittest.TestCase):
    def setUp(self):
        # Tests use TMPDIR under a private trusted scratch, never /tmp's shared ancestry.
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.state = self.root / 'state'
        self.state.mkdir(mode=0o700)
        self.h1 = self.root / 'h1.tgz'
        self.h1.write_bytes(fixture().getvalue())
        self.h1.chmod(0o600)
        FakeDocker.control_state, FakeDocker.wrong_image = 'shut down', False

    def tearDown(self):
        self.tmp.cleanup()

    def test_rebuild_strips_root_mode_and_metadata(self):
        source = fixture()
        entries, total = restore.validate_archive(source)
        with tempfile.TemporaryFile() as output:
            restore.normalize_archive(source, output, entries)
            with tarfile.open(fileobj=output, mode='r:') as archive:
                members = archive.getmembers()
                self.assertNotIn('.', [m.name for m in members])
                self.assertTrue(all(m.uid == m.gid == 999 for m in members))
                self.assertTrue(all(m.mode == (0o700 if m.isdir() else 0o600) for m in members))
        self.assertGreater(total, 8192)

    def test_paths_are_closed(self):
        for path in ('/18/docker/a', '../18/docker/a', './18/docker/../a', '18//docker/a',
                     '18/docker/./a', '18/docker\\a', '17/docker/a', '././18/docker/a'):
            with self.subTest(path=path), self.assertRaises(restore.RestoreError):
                restore.normalized_name(path)

    def test_duplicate_normalized_names(self):
        with self.assertRaises(restore.RestoreError):
            restore.validate_archive(fixture([('18/docker/PG_VERSION', b'18\n')]))

    def test_rejects_links_special_and_modes(self):
        for typ in (tarfile.SYMTYPE, tarfile.LNKTYPE, tarfile.FIFOTYPE, tarfile.CHRTYPE, tarfile.BLKTYPE):
            item = tarfile.TarInfo('./18/docker/unsafe')
            item.type, item.mode, item.uid, item.gid = typ, 0o600, 999, 999
            item.linkname = 'PG_VERSION' if typ in (tarfile.SYMTYPE, tarfile.LNKTYPE) else ''
            with self.subTest(typ=typ), self.assertRaises(restore.RestoreError):
                restore.validate_archive(fixture([(item, None)]))
        for mode in (0o644, 0o660, 0o4600):
            item = tarfile.TarInfo('./18/docker/unsafe')
            item.mode, item.uid, item.gid = mode, 999, 999
            with self.subTest(mode=mode), self.assertRaises(restore.RestoreError):
                restore.validate_archive(fixture([(item, b'')]))

    def test_rejects_external_config_and_recovery_markers(self):
        for marker in ('postmaster.pid', 'backup_label', 'backup_manifest', 'standby.signal', 'recovery.signal'):
            with self.subTest(marker=marker), self.assertRaises(restore.RestoreError):
                restore.validate_archive(fixture([('./18/docker/' + marker, b'x')]))
        for key in restore.BLOCKED_KEYS:
            with self.subTest(key=key), self.assertRaises(restore.RestoreError):
                restore.validate_archive(fixture([('./18/docker/postgresql.auto.conf', (key + " = 'external'\n").encode())]))

    def test_rejects_limits_and_missing_parent(self):
        with patch.object(restore, 'MAX_TOTAL', 8192), self.assertRaises(restore.RestoreError):
            restore.validate_archive(fixture())
        with self.assertRaises(restore.RestoreError):
            restore.validate_archive(fixture([('./18/docker/missing/child', b'x')]))

    def test_exclusive_canonical_receipt(self):
        restore.publish(self.state, 'receipt.json', {'b': 2, 'a': 1})
        self.assertEqual((self.state / 'receipt.json').read_bytes(), b'{"a":1,"b":2}\n')
        self.assertEqual(stat.S_IMODE((self.state / 'receipt.json').stat().st_mode), 0o600)
        with self.assertRaises(FileExistsError):
            restore.publish(self.state, 'receipt.json', {'changed': True})

    def execute(self, **overrides):
        original_mkdtemp = tempfile.mkdtemp
        def owned_volume(*args, **kwargs):
            return original_mkdtemp(prefix='vec-recorridos-', dir=self.root)
        def probe(container, state, normalizer, digest):
            fake = FakeDocker.instance
            return {'system_identifier': '12345', 'database_name': 'postgres', 'database_oid': 5,
                'pg_container_id': container, 'pg_image': restore.IMAGE, 'pg_image_id': restore.IMAGE_ID,
                'pg_volume': fake.volume, 'schema_sha': 'a' * 64, 'roles_sha': 'b' * 64,
                'datacl_sha': 'd' * 64, **overrides}
        with patch.object(restore, '_Docker', FakeDocker), patch.object(restore, '_probe', probe), \
             patch.object(restore, 'H1_SHA', restore.sha_stream(fixture())), \
             patch.object(restore.tempfile, 'mkdtemp', owned_volume), patch.object(restore, 'VOLUME_PARENT', self.root):
            return restore.restore(self.state, self.h1, self.root / 'normalizer.py', 'e' * 64)

    def test_fresh_restore_receipt_and_pinned_boundary(self):
        receipt = self.execute()
        self.assertEqual(receipt['kind'], 'h1_restore_confirmed')
        self.assertEqual(set(receipt), {'version', 'kind', 'estado_h1_sha', 'system_identifier',
            'database_name', 'database_oid', 'pg_container_id', 'pg_image', 'pg_image_id',
            'pg_volume', 'schema_sha', 'roles_sha', 'datacl_sha'})
        self.assertTrue((self.state / 'h1-restore-pending.json').exists())
        self.assertTrue((self.state / 'h1-restore-evidence.json').exists())
        create_args = FakeDocker.instance.create_args
        self.assertIn('listen_addresses=127.0.0.1', create_args)
        self.assertIn('port=5432', create_args)
        self.assertNotIn('listen_addresses=', create_args)
        self.assertEqual(create_args[create_args.index('--network') + 1], 'none')
        self.assertIn('vec.recorridos.owner=' + restore.OWNER, create_args)
        self.assertIn('vec.recorridos.state=' + str(self.state), create_args)
        self.assertFalse(any(a.startswith(('vec.clon.owner=', 'vec.clon.state=')) for a in create_args))
        extraction_calls = [args for args, streamed in FakeDocker.instance.calls if streamed]
        self.assertEqual(len(extraction_calls), 1)
        extraction_args = extraction_calls[0]
        self.assertEqual(extraction_args[extraction_args.index('--memory') + 1], '1g')
        self.assertEqual(create_args[create_args.index('--memory') + 1], '1g')
        for args, streamed in FakeDocker.instance.calls:
            self.assertNotIn('rm', args[:2])
            self.assertNotIn('-p', args if args[0] != 'exec' else ())
            if args[0] == 'run' or args[:2] == ('container', 'create'):
                self.assertIn(restore.IMAGE_ID, args)
                self.assertIn('none', args)
                self.assertIn('never', args)
        with self.assertRaises(restore.RestoreError):
            self.execute()

    def test_container_boundary_changes_block_probe_and_receipt(self):
        original = FakeDocker.run
        def compromised(fake, *args, **kwargs):
            output = original(fake, *args, **kwargs)
            if args[:2] == ('container', 'inspect'):
                value = json.loads(output)
                value[0]['HostConfig']['NetworkMode'] = 'host'
                return json.dumps(value)
            return output
        with patch.object(FakeDocker, 'run', compromised), self.assertRaises(restore.RestoreError):
            self.execute()
        self.assertFalse((self.state / restore.RECEIPT).exists())
        self.assertFalse(any(args[0] == 'exec' for args, _ in FakeDocker.instance.calls))

    def test_tcp_listener_and_port_cannot_expand_boundary(self):
        original = FakeDocker.run
        for variant in ('external-listener', 'wrong-port'):
            self.state = self.root / ('state-' + variant)
            self.state.mkdir(mode=0o700)
            def compromised(fake, *args, **kwargs):
                output = original(fake, *args, **kwargs)
                if args[:2] == ('container', 'inspect'):
                    value = json.loads(output)
                    command = value[0]['Config']['Cmd']
                    if variant == 'external-listener':
                        command[command.index('listen_addresses=127.0.0.1')] = 'listen_addresses=0.0.0.0'
                    else:
                        command[command.index('port=5432')] = 'port=5433'
                    return json.dumps(value)
                return output
            with self.subTest(variant=variant), patch.object(FakeDocker, 'run', compromised), self.assertRaises(restore.RestoreError):
                self.execute()
            self.assertFalse((self.state / restore.RECEIPT).exists())
            self.assertFalse(any(args[0] == 'exec' for args, _ in FakeDocker.instance.calls))

    def test_common_authority_labels_cannot_be_replaced(self):
        original = FakeDocker.run
        for variant in ('legacy', 'owner', 'state'):
            self.state = self.root / ('state-' + variant)
            self.state.mkdir(mode=0o700)
            def compromised(fake, *args, **kwargs):
                output = original(fake, *args, **kwargs)
                if args[:2] == ('container', 'inspect'):
                    value = json.loads(output)
                    labels = value[0]['Config']['Labels']
                    if variant == 'legacy':
                        labels['vec.clon.owner'] = labels.pop('vec.recorridos.owner')
                        labels['vec.clon.state'] = labels.pop('vec.recorridos.state')
                    elif variant == 'owner':
                        labels['vec.recorridos.owner'] = 'foreign-owner'
                    else:
                        labels['vec.recorridos.state'] = str(self.root / 'foreign-state')
                    return json.dumps(value)
                return output
            with self.subTest(variant=variant), patch.object(FakeDocker, 'run', compromised), self.assertRaises(restore.RestoreError):
                self.execute()
            self.assertFalse((self.state / restore.RECEIPT).exists())
            self.assertFalse(any(args[0] == 'exec' for args, _ in FakeDocker.instance.calls))

    def test_no_receipt_when_control_not_cold(self):
        FakeDocker.control_state = 'in production'
        with self.assertRaises(restore.RestoreError):
            self.execute()
        self.assertTrue((self.state / 'h1-restore-pending.json').exists())
        self.assertFalse((self.state / restore.RECEIPT).exists())
        self.assertFalse(any(args[:2] == ('container', 'create') for args, _ in FakeDocker.instance.calls))

    def test_probe_identity_replacement_blocks_receipt(self):
        with self.assertRaises(restore.RestoreError):
            self.execute(system_identifier='67890')
        self.assertFalse((self.state / restore.RECEIPT).exists())

    def test_wrong_image_blocks_extraction(self):
        FakeDocker.wrong_image = True
        with self.assertRaises(restore.RestoreError):
            self.execute()
        self.assertFalse(any(stream for _, stream in FakeDocker.instance.calls))

    def test_failed_extract_preserves_pending_without_start(self):
        original = FakeDocker.run
        def failing(fake, *args, **kwargs):
            if kwargs.get('stdin') is not None:
                raise restore.RestoreError('synthetic transport failure')
            return original(fake, *args, **kwargs)
        with patch.object(FakeDocker, 'run', failing), self.assertRaises(restore.RestoreError):
            self.execute()
        self.assertTrue((self.state / 'h1-restore-pending.json').exists())
        self.assertTrue((self.state / 'h1-volume.json').exists())
        self.assertFalse((self.state / restore.RECEIPT).exists())
        self.assertFalse(any(args[:2] == ('container', 'create') for args, _ in FakeDocker.instance.calls))

    def test_invalid_fresh_fingerprint_blocks_receipt(self):
        with self.assertRaises(restore.RestoreError):
            self.execute(schema_sha='invalid')
        self.assertFalse((self.state / restore.RECEIPT).exists())

    def test_receipt_fsync_failure_is_reported(self):
        with patch.object(restore, 'syncdir', side_effect=OSError('synthetic fsync failure')), self.assertRaises(OSError):
            restore.publish(self.state, 'failed.json', {'version': 1})

    def test_input_sha_mismatch_before_docker(self):
        with patch.object(restore, '_Docker', side_effect=AssertionError('unexpected Docker')), \
             self.assertRaises(restore.RestoreError):
            restore.restore(self.state, self.h1, self.root / 'normalizer.py', 'e' * 64)

    def test_input_symlink_and_untrusted_directory(self):
        link = self.root / 'link'
        link.symlink_to(self.h1)
        with self.assertRaises(restore.RestoreError):
            restore.trusted(link)
        self.root.chmod(0o777)
        with self.assertRaises(restore.RestoreError):
            restore.trusted(self.h1)
        self.root.chmod(0o700)

    def test_changed_inode_is_rejected(self):
        volume = self.root / 'volume'
        volume.mkdir()
        info = volume.stat()
        volume.rename(self.root / 'old-volume')
        volume.mkdir()
        with self.assertRaises(restore.RestoreError):
            restore.volume_identity(volume, (info.st_dev, info.st_ino))


# Exact non-secret normalizer D bytes, pinned independently of the scratch file.
NORMALIZER_D_SHA = '529b99dcb8a6b0368ac545a0a2cb456b8410a9bd5f9fe4e4683e2c87694b6623'
NORMALIZER_D = r'''#!/usr/bin/env python3
"""Normaliza únicamente la clave aleatoria de los delimitadores psql de pg_dump 18."""
import re
import sys

texto = sys.stdin.buffer.read().decode("utf-8")
lineas = texto.splitlines(keepends=True)
patron = re.compile(r"^(\\(?:restrict|unrestrict)) ([A-Za-z0-9]+)(\r?\n)$")
claves = []
normalizadas = []
for linea in lineas:
    if linea.startswith(("\\restrict", "\\unrestrict")):
        coincide = patron.fullmatch(linea)
        if coincide is None:
            raise SystemExit("PARO: delimitador de pg_dump inesperado")
        claves.append((coincide.group(1), coincide.group(2)))
        normalizadas.append(f"{coincide.group(1)} CLAVE_NORMALIZADA{coincide.group(3)}")
    else:
        normalizadas.append(linea)
if len(claves) != 2 or claves[0][0] != "\\restrict" or claves[1][0] != "\\unrestrict" or claves[0][1] != claves[1][1]:
    raise SystemExit("PARO: faltan o no coinciden los delimitadores de pg_dump")
sys.stdout.buffer.write("".join(normalizadas).encode("utf-8"))
'''.encode('utf-8')


class RealProbeTests(unittest.TestCase):
    """Exercise _probe, DockerDB, ReadOnlyDB and normalization without Docker/SQL."""
    cid = 'c' * 64
    volume = '/dev/shm/vec-recorridos-probe-fixture'
    schema = b"\\restrict Schema123\nCREATE SCHEMA fixture AUTHORIZATION postgres;\n\\unrestrict Schema123\n"
    roles = b"\\restrict Roles456\nCREATE ROLE fixture NOLOGIN;\n\\unrestrict Roles456\n"

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.state = self.root / 'state'
        self.state.mkdir(mode=0o700)
        self.normalizer = self.root / 'h6_normalizar_pg_dump.py'
        self.normalizer.write_bytes(NORMALIZER_D)
        self.normalizer.chmod(0o600)
        self.identity = {'system_identifier': '12345', 'database_name': 'postgres', 'database_oid': 5}
        self.acl = 'd' * 64
        self.metadata = {
            'Id': self.cid, 'Image': restore.IMAGE_ID,
            'Config': {'Image': restore.IMAGE_ID, 'Labels': {
                clon_sql.OWNER_LABEL: restore.OWNER, 'vec.recorridos.state': str(self.state)}},
            'Running': True, 'NetworkMode': 'none', 'Ports': {},
            'Mounts': [{'Type': 'bind', 'Destination': '/var/lib/postgresql', 'Source': self.volume}],
        }
        self.commands = []

    @contextlib.contextmanager
    def docker_transport(self):
        # Only Docker is replaced. Real bounded pipes/selectors and D execute normally.
        original_popen = subprocess.Popen
        emitter = (
            'import json, sys\n'
            'responses = json.loads(sys.argv[1])\n'
            'payload = sys.stdin.buffer.read().hex()\n'
            'if payload not in responses: raise SystemExit(9)\n'
            'sys.stdout.buffer.write(bytes.fromhex(responses[payload]))\n'
        )

        def popen(command, **kwargs):
            self.commands.append(command)
            self.assertEqual(kwargs['env'], {'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8'})
            self.assertIs(kwargs['close_fds'], True)
            if command[0] != 'docker':
                self.assertEqual(command, [sys.executable, '-I', '-S', '-c', NORMALIZER_D.decode()])
                return original_popen(command, **kwargs)
            if command[1] == 'inspect':
                self.assertEqual(command, ['docker', 'inspect', '--format', clon_sql.PROBE_INSPECT_FORMAT, self.cid])
                responses = {'': json.dumps(self.metadata).encode().hex()}
            elif command[2] == '-i':
                self.assertEqual(command, ['docker', 'exec', '-i', self.cid, 'psql', '-h', '/var/run/postgresql',
                    '-p', '5432', '-U', 'postgres', '-d', 'postgres', '-X', '-q', '-A', '-t', '-v', 'ON_ERROR_STOP=1'])
                responses = {
                    ('BEGIN READ ONLY;\n' + clon_sql.SYSTEM_IDENTITY_SQL + ';\nCOMMIT;').encode().hex():
                        json.dumps(self.identity).encode().hex(),
                    ('BEGIN READ ONLY;\n' + clon_sql.DATABASE_ACL_SQL + ';\nCOMMIT;').encode().hex():
                        (self.acl + '\n').encode().hex(),
                }
            else:
                self.assertEqual(command[:5], ['docker', 'exec', '-e',
                    'PGOPTIONS=-c default_transaction_read_only=on -c statement_timeout=120000 -c lock_timeout=15000', self.cid])
                arguments = command[5:]
                if arguments == ['pg_dump', '-s', '-U', 'postgres', '-d', 'postgres']:
                    output = self.schema
                else:
                    self.assertEqual(arguments, ['pg_dumpall', '--globals-only', '--no-role-passwords', '-U', 'postgres'])
                    output = self.roles
                responses = {'': output.hex()}
            return original_popen([sys.executable, '-I', '-S', '-c', emitter, json.dumps(responses)], **kwargs)

        with patch.object(clon_sql.subprocess, 'Popen', side_effect=popen):
            yield

    def probe(self, normalizer_sha=NORMALIZER_D_SHA):
        with self.docker_transport():
            return restore._probe(self.cid, self.state, self.normalizer, normalizer_sha)

    def test_real_probe_returns_cid_image_identity_and_three_fingerprints(self):
        self.assertEqual(hashlib.sha256(NORMALIZER_D).hexdigest(), NORMALIZER_D_SHA)
        observed = self.probe()
        self.assertEqual(observed, {
            **self.identity, 'pg_container_id': self.cid, 'pg_image': restore.IMAGE,
            'pg_image_id': restore.IMAGE_ID, 'pg_volume': self.volume,
            'schema_sha': hashlib.sha256(b"\\restrict CLAVE_NORMALIZADA\nCREATE SCHEMA fixture AUTHORIZATION postgres;\n\\unrestrict CLAVE_NORMALIZADA\n").hexdigest(),
            'roles_sha': hashlib.sha256(b"\\restrict CLAVE_NORMALIZADA\nCREATE ROLE fixture NOLOGIN;\n\\unrestrict CLAVE_NORMALIZADA\n").hexdigest(),
            'datacl_sha': self.acl,
        })
        self.assertEqual(sum(command[:2] == ['docker', 'inspect'] for command in self.commands), 4)
        self.assertEqual(sum(command[0] != 'docker' for command in self.commands), 2)

    def test_real_probe_rejects_divergent_identity_labels_image_and_volume(self):
        variants = {
            'cid': {'Id': 'e' * 64},
            'image': {'Image': 'sha256:' + 'e' * 64},
            'configured-image': {'Config': {**self.metadata['Config'], 'Image': 'postgres:latest'}},
            'owner': {'Config': {**self.metadata['Config'], 'Labels': {
                clon_sql.OWNER_LABEL: 'foreign', 'vec.recorridos.state': str(self.state)}}},
            'state': {'Config': {**self.metadata['Config'], 'Labels': {
                clon_sql.OWNER_LABEL: restore.OWNER, 'vec.recorridos.state': str(self.root / 'foreign-state')}}},
            'volume': {'Mounts': [{'Type': 'bind', 'Destination': '/var/lib/postgresql', 'Source': '/private/foreign'}]},
            'volume-traversal': {'Mounts': [{'Type': 'bind', 'Destination': '/var/lib/postgresql', 'Source': '/dev/shm/../foreign'}]},
            'volume-destination': {'Mounts': [{'Type': 'bind', 'Destination': '/other', 'Source': self.volume}]},
        }
        original = self.metadata
        for name, change in variants.items():
            self.metadata = {**original, **change}
            self.commands.clear()
            with self.subTest(name=name), self.assertRaises(clon_sql.Refused):
                self.probe()
            self.assertEqual(len(self.commands), 1)
        self.metadata = original

    def test_real_probe_requires_private_state_before_transport(self):
        self.state.chmod(0o755)
        with self.assertRaises(clon_sql.Refused):
            self.probe()
        self.assertEqual(self.commands, [])

    def test_real_probe_rejects_invalid_postgresql_identity(self):
        original = self.identity
        for change in ({'system_identifier': '0'}, {'database_name': 'foreign'}, {'database_oid': True}):
            self.identity = {**original, **change}
            self.commands.clear()
            with self.subTest(change=change), self.assertRaises(clon_sql.Refused):
                self.probe()
            self.assertEqual(len(self.commands), 2)

    def test_real_probe_rejects_unapproved_normalizer_before_dump(self):
        for changed_bytes, pin in ((NORMALIZER_D, 'e' * 64), (NORMALIZER_D + b'# changed\n', NORMALIZER_D_SHA)):
            self.normalizer.write_bytes(changed_bytes)
            self.commands.clear()
            with self.subTest(pin=pin), self.assertRaises(clon_sql.Refused):
                self.probe(pin)
            self.assertEqual(len(self.commands), 2)  # Identity succeeds; no dump starts.

    def test_real_probe_rejects_invalid_acl_and_unpaired_dump_delimiters(self):
        self.acl = 'not-a-digest'
        with self.assertRaises(clon_sql.Refused):
            self.probe()
        self.acl = 'd' * 64
        self.roles = self.roles.replace(b'unrestrict Roles456', b'unrestrict Wrong789')
        with self.assertRaises(clon_sql.Refused):
            self.probe()


if __name__ == '__main__':
    unittest.main()
