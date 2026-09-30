"""Pure one-shot composition tests: fake kit, fake Docker, no SQL execution."""
from dataclasses import replace
from datetime import datetime, timezone
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import clon_h6_sql_instalar as installer


class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='sql62-')
        self.addCleanup(self.temp.cleanup)
        self.state = Path(self.temp.name)
        self.state.chmod(0o700)
        self.cid = 'c' * 64
        self.image = 'sha256:' + 'a' * 64
        self.calls = []
        self.receipt = {
            'version': 1, 'kind': 'h1_restore_confirmed', 'estado_h1_sha': 'b' * 64,
            'system_identifier': '1234', 'database_name': 'postgres', 'database_oid': 1,
            'pg_container_id': self.cid, 'pg_image': 'postgres:18.4', 'pg_image_id': self.image,
            'pg_volume': '/dev/shm/vec-recorridos-dummy', 'schema_sha': 'd' * 64,
            'roles_sha': 'e' * 64, 'datacl_sha': 'f' * 64,
        }
        data = installer.clon_h6_kit.canonical(self.receipt)
        self.write('h1-restore.json', data)
        self.request = installer.clon_h6_kit.Request(
            package_tar=self.state / 'package.tar.gz', release_lock=self.state / 'release.lock',
            approved_package_sha256='1' * 64, approved_lock_sha256='2' * 64,
            h1_state_file=self.state / 'H1.tar.gz', approved_h1_sha256='b' * 64,
            git_repo=Path('/repo'), restore_receipt=self.state / 'h1-restore.json',
            approved_restore_receipt_sha256=installer.clon_sql.sha(data),
            normalizer_path=self.state / 'h6_normalizar_pg_dump.py', approved_normalizer_sha256='3' * 64,
            guiones_manifest=self.state / 'guiones.sha256', approved_guiones_sha256='4' * 64)
        self.inventory = {'propietario': 'Codex-M', 'estado': str(self.state),
                          'contenedor': self.cid, 'puerto_pg': 5432, 'commit': installer.SOURCE,
                          'identidad_clon': self.request.approved_restore_receipt_sha256}
        self.write('clon.json', json.dumps(self.inventory).encode())
        self.rows = [{'path': f'dummy/{i}.up.sql', 'sha256': '5' * 64,
                      'phase': 'H3' if i < 8 else 'H4' if i < 17 else 'H6',
                      'sql': 'DUMMY NEVER EXECUTED'} for i in range(62)]
        self.plan = {'source_ref': installer.SOURCE, 'approved_sql_ref': installer.SOURCE,
                     'plan_family': 'h6_package_62', 'file_count': 62, 'plan_sha': '6' * 64,
                     'inventory_sha': '7' * 64, 'entries': [{k: v for k, v in row.items() if k != 'sql'}
                                                        for row in self.rows],
                     'package_sha': '1' * 64, 'lock_sha': '2' * 64, 'list_sha': '8' * 64,
                     'release_sha': '9' * 64, 'estado_h1_sha': 'b' * 64}
        outer = self

        class FakeDB:
            def __init__(self, container, state, expected_image_id):
                outer.calls.append(('docker_ctor', container, state, expected_image_id))
            def check_owner(self):
                outer.calls.append(('owner',))

        class FakeKit:
            def __init__(self, request):
                outer.calls.append(('kit', request))
            def validate(self, plan, context):
                outer.calls.append(('validate', plan, context))
                return True
            def identity(self, db, context):
                outer.calls.append(('identity', isinstance(db, installer.clon_sql.ReadOnlyDB)))
                return context['identidad_clon']
            def confirm(self, db, last, completed, context):
                outer.calls.append(('preimage', last, completed))
                return True
            def verify(self, *args):
                raise AssertionError('AD132/readiness is outside this installer')

        self.FakeKit = FakeKit
        self.original_apply = installer.clon_sql.apply
        self.preflight = self.start_patch(installer.clon_sql, 'preflight_h6_package', return_value=(self.plan, self.rows))
        self.start_patch(installer.clon_sql, 'DockerDB', FakeDB)
        self.start_patch(installer.clon_h6_kit, 'H6Kit', FakeKit)
        self.apply = self.start_patch(installer.clon_sql, 'apply', side_effect=self.fake_apply)

    def start_patch(self, obj, name, *args, **kwargs):
        p = patch.object(obj, name, *args, **kwargs)
        value = p.start()
        self.addCleanup(p.stop)
        return value

    def write(self, name, data):
        path = self.state / name
        path.write_bytes(data)
        path.chmod(0o600)

    def fake_apply(self, db, rows, state, source, plan, context, kit):
        self.calls.append(('apply',))
        self.assertEqual(source, installer.SOURCE)
        self.assertIs(kit.__class__, self.FakeKit)
        self.assertTrue((state / installer.ATTEMPT).is_file())
        self.assertFalse((state / 'sql-journal.json').exists())
        record = {'version': 2, 'run_id': '11111111-1111-4111-8111-111111111111', **context,
                  'source_commit': source, 'approved_sql_ref': source, 'plan_sha': plan['plan_sha'],
                  'inventory_sha': plan['inventory_sha'], 'entries': plan['entries'], 'file_count': 62,
                  'plan_family': plan['plan_family'], 'revisions': [], 'pending': None,
                  'phase': 'awaiting_ad132', 'installed': [
                      {'position': i, **row, 'confirmed_at': datetime.now(timezone.utc).isoformat(),
                       'confirmation': 'psql_exit0_observed'} for i, row in enumerate(plan['entries'], 1)]}
        self.write('sql-journal.json', json.dumps({**record, 'journal_sha': installer.clon_sql.record_hash(record)}).encode())
        return record

    def run_install(self, request=None, image=None):
        return installer.install(request or self.request, self.state, image or self.image)

    def test_success_immutable_id_and_preimage_before_apply(self):
        result = self.run_install()
        self.assertEqual(result['phase'], 'awaiting_ad132')
        self.assertEqual(result['sql_count'], 62)
        self.assertEqual(result['journal_sha256'], installer.clon_sql.sha((self.state / 'sql-journal.json').read_bytes()))
        names = [c[0] for c in self.calls]
        self.assertLess(names.index('preimage'), names.index('apply'))
        self.assertIn(('docker_ctor', self.cid, self.state, self.image), self.calls)
        self.assertIn(('preimage', None, 0), self.calls)
        self.assertEqual(self.preflight.call_args.kwargs, {'source_ref': installer.SOURCE, 'git_repo': Path('/repo')})
        self.assertFalse((self.state / 'READY.json').exists())
        self.assertFalse((self.state / 'DB_READY.json').exists())
        with self.assertRaises(installer.Refused):
            self.run_install()
        self.assertEqual(self.apply.call_count, 1)

    def test_every_existing_state_refuses_before_provider_and_docker(self):
        for name in installer.BLOCKERS:
            with self.subTest(name=name):
                self.write(name, b'pending partial complete corrupt all refuse')
                with self.assertRaises(installer.Refused):
                    self.run_install()
                self.assertEqual(self.calls, [])
                (self.state / name).unlink()
        self.apply.assert_not_called()

    def test_dangling_journal_symlink_refuses(self):
        (self.state / 'sql-journal.json').symlink_to(self.state / 'absent')
        with self.assertRaises(installer.Refused):
            self.run_install()
        self.apply.assert_not_called()

    def test_restore_sha_mismatch_has_no_docker_or_apply(self):
        with self.assertRaises(installer.clon_sql.Refused):
            self.run_install(replace(self.request, approved_restore_receipt_sha256='0' * 64))
        self.assertEqual(self.calls, [])
        self.apply.assert_not_called()

    def test_external_image_mismatch_and_invalid_format(self):
        for image in ('postgres:18.4', 'sha256:' + '0' * 64):
            with self.subTest(image=image), self.assertRaises(installer.Refused):
                self.run_install(image=image)
        self.assertEqual(self.calls, [])

    def test_owned_inventory_and_perms_and_duplicate_keys(self):
        for raw in (json.dumps({**self.inventory, 'contenedor': 'vec-fake'}).encode(),
                    json.dumps({**self.inventory, 'estado': '/elsewhere'}).encode(),
                    json.dumps({**self.inventory, 'puerto_pg': 5433}).encode(),
                    b'{"propietario":"Codex-M","propietario":"Codex-M"}'):
            self.write('clon.json', raw)
            with self.assertRaises((installer.Refused, installer.clon_sql.Refused)):
                self.run_install()
        self.write('clon.json', json.dumps(self.inventory).encode())
        (self.state / 'clon.json').chmod(0o644)
        with self.assertRaises(installer.Refused):
            self.run_install()
        self.assertEqual(self.calls, [])

    def test_legacy_cid_constructor_failure_is_closed(self):
        self.start_patch(installer.clon_sql, 'DockerDB', side_effect=installer.clon_sql.Refused('legacy rejects CID'))
        with self.assertRaises(installer.clon_sql.Refused):
            self.run_install()
        self.apply.assert_not_called()
        self.assertFalse((self.state / installer.ATTEMPT).exists())

    def test_preimage_mismatch_has_no_attempt_or_apply(self):
        self.start_patch(self.FakeKit, 'confirm', return_value=False)
        with self.assertRaises(installer.Refused):
            self.run_install()
        self.apply.assert_not_called()
        self.assertFalse((self.state / installer.ATTEMPT).exists())

    def test_uncertain_commit_preserves_every_marker_and_refuses_retry(self):
        def uncertain(*args, **kwargs):
            self.write('sql-journal.json', b'{"pending":"dummy"}')
            self.write('.sql-confirming', b'pending')
            raise installer.clon_sql.Refused('private dummy content')
        self.apply.side_effect = uncertain
        with self.assertRaises(installer.clon_sql.Refused):
            self.run_install()
        before = {name: (self.state / name).read_bytes() for name in ('sql-journal.json', '.sql-confirming', installer.ATTEMPT)}
        with self.assertRaises(installer.Refused):
            self.run_install()
        self.assertEqual(before, {name: (self.state / name).read_bytes() for name in before})
        self.assertEqual(self.apply.call_count, 1)

    def test_status_read_only_does_not_create_locks_or_call_providers(self):
        before = set(self.state.iterdir())
        self.assertEqual(installer.status(self.state), 'journal_absent')
        self.assertEqual(before, set(self.state.iterdir()))
        self.write('sql-journal.json', b'complete or corrupt')
        self.assertEqual(installer.status(self.state), 'state_present')
        self.assertEqual(self.calls, [])
        self.preflight.assert_not_called()

    def test_state_symlink_and_state_in_git_fail(self):
        alias = self.state / 'alias'
        alias.symlink_to(self.state, target_is_directory=True)
        with self.assertRaises(OSError):
            installer.status(alias)
        (self.state / '.git').mkdir()
        with self.assertRaises(installer.Refused):
            self.run_install()
        self.apply.assert_not_called()

    def test_request_cannot_enable_ad132_or_select_callback(self):
        with self.assertRaises(installer.Refused):
            self.run_install(replace(self.request, ad132_request=object()))
        with self.assertRaises(TypeError):
            installer.install(self.request, self.state, self.image, callback=lambda: True)
        self.assertEqual(self.calls, [])

    def test_cli_mandatory_arguments_never_echo_private_values(self):
        with self.assertRaisesRegex(installer.Refused, 'arguments_invalid'), patch('sys.stderr', new_callable=io.StringIO) as err:
            installer.main(['install', '--callback', '/dummy-private'])
        self.assertEqual(err.getvalue(), '')

    def test_concurrent_installer_lock_refuses(self):
        with installer.private_state(self.state) as (directory, _):
            with installer.installer_lock(directory):
                with self.assertRaises(BlockingIOError):
                    self.run_install()
        self.apply.assert_not_called()

    def test_post_reservation_failure_retains_attempt(self):
        with patch.object(installer, 'stable_state', side_effect=[None, OSError('dummy fs failure')]):
            with self.assertRaises(OSError):
                self.run_install()
        self.assertTrue((self.state / installer.ATTEMPT).exists())
        with self.assertRaises(installer.Refused):
            self.run_install()
        self.apply.assert_not_called()

    def test_complete_record_with_uncertain_marker_never_reports_success(self):
        def incomplete(*args, **kwargs):
            record = self.fake_apply(*args, **kwargs)
            self.write('.sql-confirming', b'pending')
            return record
        self.apply.side_effect = incomplete
        with self.assertRaisesRegex(installer.Refused, 'sql62_confirmation_incomplete'):
            self.run_install()
        self.assertTrue((self.state / installer.ATTEMPT).exists())
        self.assertTrue((self.state / '.sql-confirming').exists())

    def test_cli_constructs_exact_request_and_prints_only_progress_and_digest(self):
        args = ['install', '--state-dir', str(self.state), '--expected-pg-image-id', self.image]
        for key in installer.clon_h6_kit.Request.__dataclass_fields__:
            if key != 'ad132_request':
                args += ['--' + key.replace('_', '-'), str(getattr(self.request, key))]
        with patch('sys.stdout', new_callable=io.StringIO) as output:
            installer.main(args)
        lines = output.getvalue().splitlines()
        self.assertEqual(lines[:2], ['SQL62 preflight', 'SQL62 62/62 awaiting_ad132'])
        self.assertEqual(len(lines), 3)
        self.assertTrue(lines[2].startswith(str(self.state / 'sql-journal.json') + ' SHA256 '))
        self.assertNotIn('DUMMY', output.getvalue())
        self.assertNotIn(self.cid, output.getvalue())


if __name__ == '__main__':
    unittest.main()
