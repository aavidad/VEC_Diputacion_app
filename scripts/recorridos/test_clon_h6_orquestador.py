"""Pure composition tests: no Docker, database, provider execution or network."""
import importlib.util
import json
import os
from pathlib import Path
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


if __name__ == '__main__':
    unittest.main()
