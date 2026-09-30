"""Synthetic filesystem fixtures only: no Docker, VEC or PostgreSQL calls."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

spec = importlib.util.spec_from_file_location('rotation', Path(__file__).with_name('clon_rotacion_interna.py'))
rotation = importlib.util.module_from_spec(spec)
spec.loader.exec_module(rotation)
OLD, NEW = 'a' * 40, 'b' * 40


class Missing(Exception):
    pass


class RotationTests(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory()
        self.addCleanup(self.scratch.cleanup)
        self.state = Path(self.scratch.name)
        self.state.chmod(0o700)
        self.root = self.state / 'runtime-interno'
        self.put(self.state / 'clon.json', {'propietario': 'Codex-M', 'estado': str(self.state), 'commit': OLD})
        self.module = SimpleNamespace(PREFIX='vec.clon.runtime.', DockerNotFound=Missing,
            verify_record_boundary=Mock(), inspect=Mock(side_effect=Missing()),
            process_identity=Mock(return_value=None), docker=Mock(return_value=''))
        self.patcher = patch.object(rotation, 'runtime_module', return_value=self.module)
        self.patcher.start()
        self.addCleanup(self.patcher.stop)
        self.project(OLD)

    def put(self, path, value):
        path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        path.write_bytes(rotation.encoded(value) if isinstance(value, dict) else value)
        path.chmod(0o600)

    def project(self, source, comm='material/comunicaciones', revision=0):
        self.root.mkdir(mode=0o700)
        rw = []
        for kind in sorted(rotation.KINDS):
            relative = comm if kind == 'comunicaciones' else 'rw/' + kind
            folder = self.root / relative
            folder.mkdir(mode=0o700, parents=True, exist_ok=True)
            rw.append({'kind': kind, 'source': str(folder.relative_to(self.state)),
                       'target': str(self.root / 'material/comunicaciones') if kind == 'comunicaciones' else str(folder)})
        self.put(self.root / 'material/server.key', b'private synthetic server key')
        self.put(self.root / 'runtime-config.json', {'secret': 'fixture-not-for-output', 'source': source})
        self.put(self.root / 'material-manifest.json', {'owner': 'Codex-M', 'portal': 'interno', 'target': {'source_commit': source}, 'revision': revision})
        descriptor = {'mode': 'interno', 'portal': 'interno', 'source_commit': source,
                      'manifest_sha256': rotation.digest(rotation.read(self.root / 'material-manifest.json')), 'rw': rw}
        self.put(self.state / 'material-manifest.json', {'runtime_interno': descriptor})

    def old_data(self):
        self.put(self.root / 'rw/documentos/large.pdf', b'%PDF\n' + b'x' * (1024 * 1024))
        self.put(self.root / 'rw/imagenes/empty.lock', b'')
        self.put(self.root / 'rw/data/nested/store.json', b'{"synthetic":true}')
        self.put(self.root / 'material/comunicaciones/notice.json', b'{"message":"synthetic"}')
        (self.root / 'rw/data/nested').chmod(0o750)

    def begin(self):
        self.old_data()
        result = rotation.archive(self.state, OLD, NEW)
        self.archived = self.state / result['archive']
        return result

    def test_large_empty_and_four_roots_copy_independently_preserving_keys_and_metadata(self):
        keys = {}
        for i in range(19):
            path = self.state / 'material' / ('h1-key-' + str(i))
            self.put(path, ('H1 synthetic ' + str(i)).encode())
            keys[path] = path.read_bytes()
        result = self.begin()
        old_inventory = rotation.inventory(self.archived)
        self.assertEqual(result['phase'], 'archived')
        self.assertFalse(self.root.exists())
        self.project(NEW)
        self.assertEqual(rotation.restore(self.state, NEW)['phase'], 'restored')
        for relative in ['rw/documentos', 'rw/imagenes', 'rw/data', 'material/comunicaciones']:
            self.assertEqual(rotation.inventory(self.root / relative), rotation.inventory(self.archived / relative))
        for relative, info in old_inventory.items():
            target = self.root / relative
            if info['type'] == 'file' and relative.startswith(('rw/', 'material/comunicaciones/')):
                self.assertNotEqual(target.stat().st_ino, (self.archived / relative).stat().st_ino)
                self.assertEqual(target.stat().st_nlink, 1)
        for path, content in keys.items():
            self.assertEqual(path.read_bytes(), content)
        self.assertNotIn('fixture-not-for-output', json.dumps(result))
        self.assertEqual((self.state / rotation.RECEIPT).stat().st_mode & 0o777, 0o600)

    def test_archive_and_restore_replay_do_not_move_new_projection_or_duplicate_data(self):
        self.begin()
        self.project(NEW)
        self.assertEqual(rotation.archive(self.state, OLD, NEW)['phase'], 'archived')
        self.assertTrue(self.root.exists())
        first = rotation.restore(self.state, NEW)
        before = rotation.inventory(self.root)
        self.assertEqual(rotation.restore(self.state, NEW), first)
        self.assertEqual(rotation.inventory(self.root), before)

    def test_archive_recovers_move_completed_before_receipt_phase_publication(self):
        self.old_data()
        real = rotation.write_cas
        def publish(path, value, previous=None):
            if value['phase'] == 'archived':
                raise OSError('fixture interrupted after rename')
            return real(path, value, previous)
        with patch.object(rotation, 'write_cas', side_effect=publish), self.assertRaises(OSError):
            rotation.archive(self.state, OLD, NEW)
        self.assertFalse(self.root.exists())
        self.put(self.state / 'clon.json', {'propietario': 'Codex-M', 'estado': str(self.state), 'commit': NEW})
        self.assertEqual(rotation.archive(self.state, OLD, NEW)['phase'], 'archived')
        self.assertEqual(len(list((self.state / 'proyecciones').glob(OLD + '-*'))), 2)

    def test_metadata_publication_failure_can_retry_without_a_partial_final_file(self):
        self.old_data()
        real = rotation.rename_exclusive
        def rename(old, new):
            if str(new).endswith('.metadata/material-manifest.json'):
                raise OSError('fixture interrupted after partial temporary write')
            return real(old, new)
        with patch.object(rotation, 'rename_exclusive', side_effect=rename), self.assertRaises(OSError):
            rotation.archive(self.state, OLD, NEW)
        snapshots, = (self.state / 'proyecciones').glob('*.metadata')
        self.assertFalse((snapshots / 'material-manifest.json').exists())
        self.assertTrue(self.root.exists())
        self.assertEqual(rotation.archive(self.state, OLD, NEW)['phase'], 'archived')

    def test_partial_restore_recovers_absent_or_equal_entries_only(self):
        self.begin()
        self.project(NEW)
        real = rotation.copy_file
        count = 0
        def copy(*args):
            nonlocal count
            count += 1
            if count == 2:
                raise OSError('fixture copy interrupted')
            return real(*args)
        with patch.object(rotation, 'copy_file', side_effect=copy), self.assertRaises(OSError):
            rotation.restore(self.state, NEW)
        self.assertEqual(rotation.receipt(self.state)['phase'], 'restore_pending')
        self.assertEqual(rotation.restore(self.state, NEW)['phase'], 'restored')

    def test_readonly_directory_modes_restore_and_recover_after_partial_copy(self):
        self.old_data()
        self.put(self.root / 'rw/documentos/sealed/saved.pdf', b'%PDF fixture')
        (self.root / 'rw/documentos/sealed').chmod(0o500)
        result = rotation.archive(self.state, OLD, NEW)
        self.archived = self.state / result['archive']
        self.project(NEW)
        with patch.object(rotation, 'copy_file', side_effect=OSError('fixture interruption')), self.assertRaises(OSError):
            rotation.restore(self.state, NEW)
        self.assertEqual(rotation.receipt(self.state)['phase'], 'restore_pending')
        self.assertEqual(rotation.restore(self.state, NEW)['phase'], 'restored')
        self.assertEqual((self.root / 'rw/documentos/sealed').stat().st_mode & 0o777, 0o500)
        self.assertEqual(rotation.inventory(self.root / 'rw/documentos/sealed'), rotation.inventory(self.archived / 'rw/documentos/sealed'))
        self.assertEqual(rotation.restore(self.state, NEW)['phase'], 'restored')
        # Temporary permission cannot legitimise a later application change.
        (self.root / 'rw/documentos/sealed').chmod(0o700)
        with self.assertRaises(rotation.RotationError):
            rotation.restore(self.state, NEW)

    def test_new_writes_missing_restored_files_or_new_runtime_deny_retransfer(self):
        self.begin(); self.project(NEW); rotation.restore(self.state, NEW)
        target = self.root / 'rw/documentos/large.pdf'
        target.write_bytes(b'new runtime output')
        with self.assertRaises(rotation.RotationError):
            rotation.restore(self.state, NEW)
        self.assertEqual(target.read_bytes(), b'new runtime output')
        target.unlink()
        with self.assertRaises(rotation.RotationError):
            rotation.restore(self.state, NEW)
        self.put(self.state / 'runtime-container-stopped-fixture.json', {'source_commit': NEW})
        with self.assertRaises(rotation.RotationError):
            rotation.restore(self.state, NEW)

    def test_existing_target_conflict_is_rejected_before_any_copy(self):
        self.begin(); self.project(NEW)
        self.put(self.root / 'rw/data/different.json', b'new data')
        before = rotation.inventory(self.root)
        with self.assertRaises(rotation.RotationError):
            rotation.restore(self.state, NEW)
        self.assertEqual(rotation.inventory(self.root), before)

    def test_active_owned_container_or_unrecorded_object_blocks_archive(self):
        self.put(self.state / 'runtime-container.json', {'source_commit': OLD, 'container_id': 'c' * 64, 'pid': 12})
        self.module.inspect.side_effect = None
        self.module.inspect.return_value = {'State': {'Running': True}}
        with self.assertRaises(rotation.RotationError):
            rotation.archive(self.state, OLD, NEW)
        self.assertTrue(self.root.exists())
        (self.state / 'runtime-container.json').unlink()
        self.module.docker.return_value = 'c' * 64
        with self.assertRaises(rotation.RotationError):
            rotation.archive(self.state, OLD, NEW)

    def test_published_ready_or_runtime_lock_writer_denies_rotation(self):
        self.put(self.state / 'READY.json', {'portal_proceso': 'interno'})
        with self.assertRaises(rotation.RotationError):
            rotation.archive(self.state, OLD, NEW)
        (self.state / 'READY.json').unlink()
        with rotation.locked(self.state):
            with self.assertRaises(BlockingIOError):
                rotation.archive(self.state, OLD, NEW)

    def test_foreign_source_clone_receipt_and_rw_reference_are_denied(self):
        with self.assertRaises(rotation.RotationError):
            rotation.archive(self.state, NEW, OLD)
        parent = json.loads((self.state / 'material-manifest.json').read_bytes())
        parent['runtime_interno']['rw'][0]['source'] = 'material'
        self.put(self.state / 'material-manifest.json', parent)
        with self.assertRaises(rotation.RotationError):
            rotation.archive(self.state, OLD, NEW)
        self.put(self.state / 'clon.json', {'propietario': 'someone-else', 'estado': str(self.state)})
        with self.assertRaises(rotation.RotationError):
            rotation.archive(self.state, OLD, NEW)

    def test_symlink_hardlink_and_fifo_are_rejected_without_reading_payload(self):
        file = self.root / 'rw/data/plain'
        self.put(file, b'dummy')
        link = self.root / 'rw/data/link'
        for kind in ['symlink', 'hardlink', 'fifo']:
            if kind == 'symlink': link.symlink_to(file)
            elif kind == 'hardlink': os.link(file, link)
            else: os.mkfifo(link, 0o600)
            with self.subTest(kind=kind), self.assertRaises(rotation.RotationError):
                rotation.archive(self.state, OLD, NEW)
            link.unlink()

    def test_foreign_archive_destination_is_never_replaced(self):
        digest = rotation.digest(rotation.read(self.root / 'material-manifest.json'))
        occupied = self.state / 'proyecciones' / (OLD + '-' + digest)
        occupied.mkdir(mode=0o700, parents=True)
        self.put(occupied / 'foreign', b'must stay')
        with self.assertRaises(rotation.RotationError):
            rotation.archive(self.state, OLD, NEW)
        self.assertEqual((occupied / 'foreign').read_bytes(), b'must stay')

    def test_exclusive_rename_never_replaces_an_existing_empty_directory(self):
        first, second = self.state / 'first', self.state / 'second'
        first.mkdir(); second.mkdir()
        with self.assertRaises(OSError):
            rotation.rename_exclusive(first, second)
        self.assertTrue(first.is_dir()); self.assertTrue(second.is_dir())

    def same_archive(self, sha=None):
        sha = sha or rotation.digest(rotation.read(self.root / 'material-manifest.json'))
        return rotation.archive(self.state, OLD, OLD, allow_same_source=True,
                                expected_old_manifest_sha256=sha)

    def test_same_source_requires_explicit_exact_preimage_without_mutation(self):
        before = rotation.inventory(self.root)
        for options in ({}, {'allow_same_source': True},
                        {'allow_same_source': True, 'expected_old_manifest_sha256': 'f' * 64}):
            with self.subTest(options=options), self.assertRaises(rotation.RotationError):
                rotation.archive(self.state, OLD, OLD, **options)
            self.assertEqual(rotation.inventory(self.root), before)
            self.assertFalse((self.state / rotation.RECEIPT).exists())
        with self.assertRaises(rotation.RotationError):
            rotation.archive(self.state, OLD, NEW, allow_same_source=True,
                             expected_old_manifest_sha256='f' * 64)

    def test_same_source_distinct_manifest_preserves_four_roots_and_receipt_chain(self):
        self.old_data()
        sha1 = rotation.digest(rotation.read(self.root / 'material-manifest.json'))
        first = self.same_archive(sha1)
        self.archived = self.state / first['archive']
        self.project(OLD, revision=1)
        self.assertEqual(rotation.restore(self.state, OLD)['phase'], 'restored')
        for relative in rotation.receipt(self.state)['rw'].values():
            self.assertEqual(rotation.inventory(self.root / relative), rotation.inventory(self.archived / relative))
        target = self.root / 'rw/documentos/large.pdf'
        self.assertNotEqual(target.stat().st_ino, (self.archived / 'rw/documentos/large.pdf').stat().st_ino)
        self.assertEqual(self.same_archive(sha1)['phase'], 'restored')
        sha2 = rotation.digest(rotation.read(self.root / 'material-manifest.json'))
        second = self.same_archive(sha2)
        self.assertNotEqual(first['archive'], second['archive'])
        self.assertTrue((self.state / (first['archive'] + '.receipt.json')).exists())
        self.project(OLD, revision=2)
        self.assertEqual(rotation.restore(self.state, OLD)['phase'], 'restored')

    def test_same_source_unchanged_projection_and_changed_target_on_replay_deny(self):
        self.old_data(); self.same_archive(); self.project(OLD)
        with self.assertRaisesRegex(rotation.RotationError, 'projection_must_change'):
            rotation.restore(self.state, OLD)
        self.assertFalse((self.root / 'rw/documentos/large.pdf').exists())
        marker = self.root / 'material-manifest.json'
        value = json.loads(marker.read_bytes()); value['revision'] = 1
        self.put(marker, value)
        parent = json.loads((self.state / 'material-manifest.json').read_bytes())
        parent['runtime_interno']['manifest_sha256'] = rotation.digest(rotation.read(marker))
        self.put(self.state / 'material-manifest.json', parent)
        rotation.restore(self.state, OLD)
        value['revision'] = 2; self.put(marker, value)
        with self.assertRaisesRegex(rotation.RotationError, 'target_manifest_changed'):
            rotation.restore(self.state, OLD)

    def test_same_source_only_exact_archived_stopped_history_allowed(self):
        self.old_data()
        history = self.state / 'runtime-container-stopped-fixture.json'
        self.put(history, {'source_commit': OLD, 'container_id': 'c' * 64})
        self.same_archive(); self.project(OLD, revision=1)
        self.assertEqual(rotation.restore(self.state, OLD)['phase'], 'restored')
        self.put(history, {'source_commit': OLD, 'container_id': 'd' * 64})
        with self.assertRaisesRegex(rotation.RotationError, 'new_runtime_already_started'):
            rotation.restore(self.state, OLD)
        self.put(history, {'source_commit': OLD, 'container_id': 'c' * 64})
        self.put(self.state / 'runtime-container-stopped-new.json', {'source_commit': OLD})
        with self.assertRaisesRegex(rotation.RotationError, 'new_runtime_already_started'):
            rotation.restore(self.state, OLD)

    def test_same_source_partial_copy_recovers_but_new_writes_deny(self):
        self.old_data(); self.same_archive(); self.project(OLD, revision=1)
        with patch.object(rotation, 'copy_file', side_effect=OSError('fixture interruption')), self.assertRaises(OSError):
            rotation.restore(self.state, OLD)
        self.assertEqual(rotation.restore(self.state, OLD)['phase'], 'restored')
        target = self.root / 'rw/data/nested/store.json'
        target.write_bytes(b'new runtime write')
        with self.assertRaises(rotation.RotationError):
            rotation.restore(self.state, OLD)
        self.assertEqual(target.read_bytes(), b'new runtime write')


if __name__ == '__main__':
    unittest.main()
