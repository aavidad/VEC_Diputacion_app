"""Pure archive and Docker contract tests; never launch Docker or restore H1."""

import io
import json
import os
from pathlib import Path
import stat
import tarfile
import tempfile
import unittest
from unittest.mock import patch

import clon_h1_restore as restore


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
            labels = dict(a.split('=', 1) for a in arguments if a.startswith('vec.clon.'))
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


if __name__ == '__main__':
    unittest.main()
