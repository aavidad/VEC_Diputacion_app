"""Pure composition tests: no Docker, database, provider execution or network."""
from dataclasses import replace
from types import SimpleNamespace
import io
import importlib.util
import json
import os
from pathlib import Path
import stat
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).with_name('clon_h6_orquestador.py')
spec = importlib.util.spec_from_file_location('clon_h6_orquestador', SCRIPT)
module = importlib.util.module_from_spec(spec)
import sys
sys.modules[spec.name] = module
spec.loader.exec_module(module)


class CompositionTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def test_order_and_recovery_do_not_allow_SQL_retry_or_approval_synthesis(self):
        ops = module.composition()['operations']
        self.assertEqual([op['id'] for op in ops], ['h1', 'sql62',
            'material_pre_ad132', 'canary', 'preview_ad132', 'approval_ad132',
            'apply_ad132', 'db_ready', 'material_final', 'runtime'])
        self.assertEqual(ops[1]['recovery'], 'never_retry')
        self.assertEqual(ops[5]['recovery'], 'never_synthesize')
        self.assertEqual(ops[6]['recovery'], 'revalidar_only')
        self.assertEqual(ops[9]['recovery'], 'no_SQL')

    def test_transport_is_PG_namespace_5432_without_host_publication(self):
        value = module.composition(self.root)
        transport = value['postgres_transport']
        self.assertEqual(transport['logical_port'], 5432)
        self.assertEqual(transport['network'], 'none')
        self.assertFalse(transport['host_published'])
        self.assertEqual(transport['app_network'], 'container:<PGID>')
        self.assertEqual(value['plan_status'], 'external_inputs_required')
        self.assertEqual(set(value['plan_inputs']), set(module.PLAN_INPUTS))

    def test_plan_without_external_inputs_never_outputs_historical45(self):
        with patch('sys.stdout') as output:
            with self.assertRaisesRegex(module.Refused, 'arguments_invalid_external_inputs_required'):
                module.main(['plan'])
            output.write.assert_not_called()

    def test_all_APIs_present_never_override_operational_review_blockers(self):
        with patch.object(module, 'source_api', return_value=set(
                key for values in module.API_REQUIREMENTS.values() for key in values)):
            result = module.composition(self.root)
            self.assertEqual(tuple(result['blockers']), module.COMPOSITION_BLOCKERS)
            self.assertFalse(result['ready'])
            self.assertFalse(result['executable'])
            for action in ('preparar', 'reiniciar'):
                with self.assertRaisesRegex(module.Refused, 'composition_incomplete'):
                    module.require_complete(action, self.root)

    def test_missing_components_are_explicit_and_never_create_state(self):
        before = list(self.root.iterdir())
        result = module.composition(self.root)
        self.assertIn('api_missing:clon_h1_restore.py', result['blockers'])
        self.assertIn('api_missing:clon_h6_kit.py', result['blockers'])
        self.assertEqual(before, list(self.root.iterdir()))
        self.assertEqual(result['sql_count'], 62)
        self.assertTrue(result['ad132_separate'])

    def test_syntax_inspection_does_not_execute_provider_side_effects(self):
        payload = self.root / 'provider.py'
        marker = self.root / 'executed'
        payload.write_text("raise RuntimeError('private-provider-detail')\n"
            "open(" + repr(str(marker)) + ", 'w').write('bad')\n"
            "def apply(): pass\nclass Kit:\n def verify(self): pass\n")
        payload.chmod(0o600)
        self.assertEqual(module.source_api(payload), {'apply', 'Kit', 'Kit.verify'})
        self.assertFalse(marker.exists())

    def test_symlinks_hardlinks_open_modes_and_oversize_source_are_rejected(self):
        payload = self.root / 'source.py'
        payload.write_text('def apply(): pass\n')
        payload.chmod(0o600)
        alias = self.root / 'alias.py'
        alias.symlink_to(payload)
        with self.assertRaises(OSError):
            module.source_api(alias)
        alias.unlink()
        os.link(payload, alias)
        with self.assertRaises(module.Refused):
            module.source_api(payload)
        alias.unlink()
        payload.chmod(0o666)
        with self.assertRaises(module.Refused):
            module.source_api(payload)
        payload.chmod(0o600)
        payload.write_bytes(b'#' * (module.MAX_SOURCE + 1))
        with self.assertRaises(module.Refused):
            module.source_api(payload)

    def test_source_cannot_silently_advance_to_main(self):
        with self.assertRaisesRegex(module.Refused, 'source_not_approved_H6_SQL'):
            module.composition(source_ref='a' * 40)

    def test_CLI_flags_and_legacy_ready_cannot_enable_mutation(self):
        (self.root / 'READY.json').write_text('{"approved":true}')
        environment = {'PATH': '/usr/bin:/bin', 'HOME': str(self.root),
                       'TMPDIR': str(self.root), 'VEC_H6_APPROVED': 'true',
                       'VEC_RECORRIDOS_ESTADO': str(self.root),
                       'PYTHONPYCACHEPREFIX': str(self.root / 'bytecode')}
        before = {p.name: p.read_bytes() for p in self.root.iterdir()}
        for action in ('preparar', 'reiniciar'):
            result = subprocess.run(['/usr/bin/python3', '-B', str(SCRIPT), action],
                env=environment, capture_output=True, text=True, timeout=10)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('H6-NO-GO composition_incomplete', result.stderr)
            self.assertFalse(json.loads(result.stdout)['executable'])
        self.assertEqual(before, {p.name: p.read_bytes() for p in self.root.iterdir()})

    def test_preflight_outputs_only_public_contract_and_never_secrets(self):
        result = module.composition(self.root)
        self.assertNotIn(str(self.root), json.dumps(result))
        self.assertNotIn('private-provider-detail', json.dumps(result))



class SQLPhaseTests(unittest.TestCase):
    """Real file contracts with fake restore/installer/DB; never run SQL."""
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.state = self.root / 'state'
        self.state.mkdir(mode=0o700)
        self.h1, self.installer, self.kit, self.sql = module.phase_apis()
        self.calls = []
        normalizer = self.root / 'h6_normalizar_pg_dump.py'
        normalizer.write_bytes(b'# fixture never executed\n')
        normalizer_sha = self.sql.sha(normalizer.read_bytes())
        guiones = self.root / 'guiones.sha256'
        guiones.write_text(normalizer_sha + '  h6_normalizar_pg_dump.py\n')
        guiones_sha = self.sql.sha(guiones.read_bytes())
        lock = self.root / 'release.lock'
        lock.write_text('KIT_GUIONES_SHA256 ' + guiones_sha + '\n')
        self.request = module.SQLRequest(self.root / 'package.tgz', lock, 'a'*64,
            self.sql.sha(lock.read_bytes()), self.root / 'H1.tgz', self.h1.H1_SHA,
            self.root, normalizer, normalizer_sha, guiones, guiones_sha, self.h1.IMAGE_ID)
        self.plan = {'source_ref': module.SOURCE, 'approved_sql_ref': module.SOURCE,
            'plan_family': self.sql.H6_PACKAGE_FAMILY, 'file_count': 62,
            'entries': [{'phase': 'H3' if n < 8 else 'H4' if n < 17 else 'H6',
                         'path': f'deploy/postgresql/fixture/{n:06d}.up.sql', 'sha256': 'b'*64}
                        for n in range(62)], 'plan_sha': 'c'*64, 'inventory_sha': 'd'*64,
            'package_sha': 'a'*64, 'lock_sha': self.request.approved_lock_sha256,
            'list_sha': 'e'*64, 'release_sha': 'f'*64, 'estado_h1_sha': self.h1.H1_SHA}
        self.receipt = {'version': 1, 'kind': 'h1_restore_confirmed', 'estado_h1_sha': self.h1.H1_SHA,
            'system_identifier': '123456789', 'database_name': 'postgres', 'database_oid': 5,
            'pg_container_id': '1'*64, 'pg_image': 'postgres:18.4', 'pg_image_id': self.h1.IMAGE_ID,
            'pg_volume': '/dev/shm/vec-recorridos-fixture',
            'schema_sha': '2'*64, 'roles_sha': '3'*64, 'datacl_sha': '4'*64}
        self.post = {'schema_sha': '5'*64, 'roles_sha': '6'*64, 'datacl_sha': '7'*64}
        self.identity = {key: self.receipt[key] for key in self.kit.IDENTITY_FIELDS}
        self.preflight = self.patch(self.sql, 'preflight_h6_package', return_value=(self.plan, [None]*62))
        self.restore = self.patch(self.h1, 'restore', side_effect=self.fake_restore)
        self.install = self.patch(self.installer, 'install', side_effect=self.fake_install)
        original_lstat = Path.lstat
        def volume_lstat(path):
            if str(path) == self.receipt['pg_volume']:
                return SimpleNamespace(st_mode=stat.S_IFDIR|0o700, st_dev=1, st_ino=2)
            return original_lstat(path)
        self.patch(Path, 'lstat', volume_lstat)
        outer = self
        class FakeDB:
            def __init__(self, cid, state, expected_image_id):
                outer.calls.append('db')
                outer.assertEqual((cid, state, expected_image_id),
                    (outer.receipt['pg_container_id'], outer.state, outer.h1.IMAGE_ID))
            def check_owner(self): outer.calls.append('owner')
            def system_identity(self):
                outer.calls.append('identity')
                return dict(outer.identity)
            def schema_digest(self, path, sha):
                outer.calls.append('schema'); return outer.post['schema_sha']
            def roles_digest(self, path, sha):
                outer.calls.append('roles'); return outer.post['roles_sha']
            def database_acl_digest(self):
                outer.calls.append('acl'); return outer.post['datacl_sha']
            def query(self, text): raise AssertionError('No SQL is allowed in these fixtures')
        self.patch(self.sql, 'DockerDB', FakeDB)
        self.patch(self.sql, 'apply', side_effect=AssertionError('No actual SQL in worker tests'))

    def patch(self, target, name, *args, **kwargs):
        value = patch.object(target, name, *args, **kwargs)
        result = value.start()
        self.addCleanup(value.stop)
        return result

    def write(self, name, value):
        p = self.state / name
        p.write_bytes(self.kit.canonical(value))
        p.chmod(0o600)

    def fake_restore(self, state, h1, normalizer, sha):
        self.calls.append('restore')
        self.assertTrue((state / module.PHASE_ATTEMPT).exists())
        self.assertFalse((state / 'sql-journal.json').exists())
        run = '0'*32
        self.write('h1-restore.json', self.receipt)
        self.write('h1-restore-pending.json', {'version': 1, 'kind': 'h1_restore_pending',
            'run_id': run, 'estado_h1_sha': self.h1.H1_SHA, 'pg_image_id': self.h1.IMAGE_ID})
        self.write('h1-volume.json', {'run_id': run, 'path': self.receipt['pg_volume'], 'device': 1, 'inode': 2})
        self.write('h1-container.json', {'run_id': run, 'pg_container_id': self.receipt['pg_container_id']})
        return dict(self.receipt)

    def fake_install(self, request, state, image):
        self.calls.append('install')
        with self.installer.private_state(state) as (directory, _):
            records_sha = self.installer.h1_records(directory, self.receipt, request)
            self.installer.reserve_attempt(directory, request, self.plan, self.receipt, image, records_sha)
        context = {key: self.plan[key] for key in (*self.sql.CONTEXT_KEYS[1:], 'lock_sha')}
        context['identidad_clon'] = request.approved_restore_receipt_sha256
        record = {'version': 2, 'run_id': '11111111-1111-4111-8111-111111111111',
            **context, 'source_commit': module.SOURCE, 'approved_sql_ref': module.SOURCE,
            'plan_family': self.sql.H6_PACKAGE_FAMILY, 'plan_sha': self.plan['plan_sha'],
            'inventory_sha': self.plan['inventory_sha'], 'entries': self.plan['entries'],
            'file_count': 62, 'revisions': [], 'pending': None, 'phase': 'awaiting_ad132',
            'installed': [{'position': n, **entry, 'confirmed_at': '2026-10-01T00:00:00+00:00',
                           'confirmation': 'psql_exit0_observed'}
                          for n, entry in enumerate(self.plan['entries'], 1)]}
        self.write('sql-journal.json', {**record, 'journal_sha': self.sql.record_hash(record)})
        return {'phase': 'awaiting_ad132', 'sql_count': 62,
                'journal_sha256': self.sql.sha((state/'sql-journal.json').read_bytes())}

    def prepare(self):
        return module.preparar_sql(self.request, self.state)

    def snapshot(self):
        return {p.name: (p.read_bytes(), p.stat().st_mtime_ns, p.stat().st_mode) for p in self.state.iterdir()}

    def test_prepare_once_observes_real_contracts_and_verify_is_read_only(self):
        result = self.prepare()
        self.assertEqual((result['phase'], result['sql_count'], result['ready']), ('awaiting_ad132', 62, False))
        self.assertLess(self.calls.index('restore'), self.calls.index('install'))
        self.assertEqual(result['acta_sha256'], self.sql.sha((self.state/module.PHASE_RECEIPT).read_bytes()))
        before = self.snapshot()
        self.calls.clear()
        verified = module.verificar_sql(self.request, self.state, result['acta_sha256'])
        self.assertTrue(verified['verified'])
        self.assertEqual(self.snapshot(), before)
        self.assertEqual(self.restore.call_count, 1)
        self.assertEqual(self.install.call_count, 1)
        self.assertIn('schema', self.calls); self.assertIn('roles', self.calls); self.assertIn('acl', self.calls)
        for name in ('DB_READY.json', 'READY.json', 'clon.json', 'material', 'runtime-process.json'):
            self.assertFalse((self.state/name).exists())
        with self.assertRaises(self.installer.Refused): self.prepare()
        self.assertEqual(self.snapshot(), before)
        self.assertEqual(self.restore.call_count, 1)

    def test_every_previous_file_even_dangling_pending_refuses_restore(self):
        for name in ('h1-restore-pending.json', 'h1-restore.json', 'sql62-intento.json',
                     'sql-journal.json', '.sql-confirming', module.PHASE_ATTEMPT, 'unknown'):
            with self.subTest(name=name):
                path = self.state/name
                path.symlink_to(self.root/'missing')
                with self.assertRaises(self.installer.Refused): self.prepare()
                path.unlink()
        self.restore.assert_not_called(); self.install.assert_not_called()

    def test_restore_failure_preserves_reservation_and_prevents_retry(self):
        self.restore.side_effect = RuntimeError('dummy secret must not be emitted')
        with self.assertRaises(RuntimeError): self.prepare()
        self.assertTrue((self.state/module.PHASE_ATTEMPT).exists())
        before = self.snapshot()
        with self.assertRaises(self.installer.Refused): self.prepare()
        self.assertEqual(self.snapshot(), before)
        self.assertEqual(self.restore.call_count, 1)

    def test_sql_failure_preserves_h1_and_uncertain_journal(self):
        def fail(*args):
            self.write('sql-journal.json', {'pending': 'uncertain'})
            raise RuntimeError('dummy')
        self.install.side_effect = fail
        with self.assertRaises(RuntimeError): self.prepare()
        before = self.snapshot()
        self.assertIn('h1-restore.json', before)
        self.assertNotIn(module.PHASE_RECEIPT, before)
        with self.assertRaises(self.installer.Refused): self.prepare()
        self.assertEqual(self.snapshot(), before)
        self.assertEqual(self.restore.call_count, 1); self.assertEqual(self.install.call_count, 1)

    def test_invalid_external_guiones_normalizer_h1_and_image_before_restore(self):
        for fields in ({'approved_normalizer_sha256': 'a'*64}, {'approved_guiones_sha256': 'a'*64},
                       {'approved_lock_sha256': 'a'*64}, {'approved_h1_sha256': 'a'*64},
                       {'expected_pg_image_id': 'postgres:18.4'}):
            with self.subTest(fields=fields), self.assertRaises((self.installer.Refused, self.sql.Refused)):
                module.preparar_sql(replace(self.request, **fields), self.state)
        self.restore.assert_not_called(); self.assertEqual(self.snapshot(), {})

    def test_verification_requires_external_acta_pin_and_rejects_changed_postimage(self):
        result = self.prepare()
        before = self.snapshot()
        for pin in (None, '', 'a'*64):
            with self.assertRaises(self.installer.Refused):
                module.verificar_sql(self.request, self.state, pin)
        for field in self.post:
            old = self.post[field]; self.post[field] = 'a'*64
            with self.subTest(field=field), self.assertRaises(self.installer.Refused):
                module.verificar_sql(self.request, self.state, result['acta_sha256'])
            self.post[field] = old
        self.assertEqual(self.snapshot(), before)

    def test_verification_rejects_changed_h1_identity_attempt_journal_pending_or_phase(self):
        result = self.prepare()
        original = self.snapshot()
        for name in ('h1-container.json', self.installer.ATTEMPT, module.PHASE_ATTEMPT, module.PHASE_RECEIPT):
            data = (self.state/name).read_bytes()
            self.write(name, {'wrong': True})
            with self.subTest(name=name), self.assertRaises((self.installer.Refused, self.sql.Refused)):
                module.verificar_sql(self.request, self.state, result['acta_sha256'])
            (self.state/name).write_bytes(data)
        for fields in ({'pending': {'position': 62}}, {'phase': 'ad132_confirmed'}, {'installed': []}):
            path = self.state/'sql-journal.json'; data = path.read_bytes()
            record = json.loads(data); record.update(fields); record['journal_sha'] = self.sql.record_hash(record)
            self.write(path.name, record)
            with self.subTest(fields=fields), self.assertRaises((self.installer.Refused, self.sql.Refused)):
                module.verificar_sql(self.request, self.state, result['acta_sha256'])
            path.write_bytes(data)
        (self.state/'.sql-confirming').write_bytes(b'pending')
        with self.assertRaises(self.installer.Refused):
            module.verificar_sql(self.request, self.state, result['acta_sha256'])
        (self.state/'.sql-confirming').unlink()
        self.identity['pg_container_id'] = 'a'*64
        with self.assertRaises(self.sql.Refused):
            module.verificar_sql(self.request, self.state, result['acta_sha256'])
        self.assertEqual({k:v[0] for k,v in self.snapshot().items()}, {k:v[0] for k,v in original.items()})
        self.assertEqual(self.install.call_count, 1)

    def test_CLI_uses_nominal_pins_and_never_full_preparation(self):
        args = ['preparar-sql', '--state-dir', str(self.state)]
        for key in module.SQLRequest.__dataclass_fields__:
            args.extend(['--'+key.replace('_','-'), str(getattr(self.request,key))])
        with patch('sys.stdout', new_callable=io.StringIO) as out:
            module.main(args)
        result = json.loads(out.getvalue())
        self.assertFalse(result['ready'])
        self.assertNotIn(str(self.state), out.getvalue())
        with self.assertRaisesRegex(module.Refused, 'composition_incomplete'):
            module.require_complete('preparar')


if __name__ == '__main__':
    unittest.main()
