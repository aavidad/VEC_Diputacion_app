"""Local fixture tests: never run the real exporter or read its private material.

Run inside an OS sandbox with VEC_TEST_SCRATCH pointing to its private scratch.
The fixture producer uses dummy source accreditation; production has no override.
"""
from datetime import datetime, timezone
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("alias_export", Path(__file__).with_name("clon_alias_export.py"))
alias = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(alias)
m = alias.material
NOW = datetime.now(timezone.utc)


class AliasExportTests(unittest.TestCase):
    def setUp(self):
        scratch = os.environ.get("VEC_TEST_SCRATCH")
        self.assertTrue(scratch, "VEC_TEST_SCRATCH is required")
        self.temp = tempfile.TemporaryDirectory(dir=scratch)
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.inputs = self.base / "inputs"
        self.inputs.mkdir(mode=0o700)
        self.source = self.inputs / "source.json"
        self.ack = self.inputs / "ack.json"
        self.binary_dir = self.base / "binary"
        self.binary_dir.mkdir(mode=0o700)
        self.binary = self.binary_dir / "vec-server"
        self.root = self.base / "material"
        self.output = self.base / "output"
        self.refs = {name: prefix + hashlib.sha256(("fixture:" + name).encode()).hexdigest()[:32] for name, prefix in
                     (("cuenta", "cta_"), ("persona", "per_"), ("perfil", "prf_"), ("contexto", "vca_"),
                      ("vinculo", "vin_"), ("candidato", "can_"), ("procedencia", "prc_"))}
        master = {"datos_sinteticos": True, "personas": [{"persona_ref": self.refs["persona"], "nombre_visible": "Persona de prueba"}],
                  "candidatoBolsa": {key + "_ref": self.refs[key] for key in ("cuenta", "persona", "perfil", "candidato")}}
        self.write(self.source, m.encoded(master))
        self.write(self.ack, m.encoded({"fixture": True}))
        metadata = {"version": 1, "source_sha256": m.digest(self.source.read_bytes()), "autoridad_maestra": "fixture",
                    "responsable": "fixture", "acreditado_en": "2026-09-30T23:53:00Z", "refs": self.refs, "sujeto": m.SUBJECT,
                    "vigente_desde": "2026-10-01T00:00:00.000000Z", "vigente_hasta": "2026-10-15T00:00:00.000000Z"}
        self.addCleanup(patch.stopall)
        patch.object(m, "SOURCE_SHA", metadata["source_sha256"]).start()
        patch.object(m, "accreditation", return_value=metadata).start()
        patch.object(alias, "ACK_SHA", m.digest(self.ack.read_bytes())).start()
        m.produce(self.source, self.ack, self.root, now=NOW)
        patch.object(alias, "MATERIAL_SHA", m.digest((self.root / m.RECEIPT).read_bytes())).start()
        self.bound = m.binding(self.source.read_bytes(), self.ack.read_bytes(), NOW)
        self.files = {name: (self.root / name).read_bytes() for name in m.FILES}
        self.expected = m.encoded({"version": 1, "cuentas": [alias.expected_account(self.files, self.bound)]})
        self.git = patch.object(alias, "git_pins").start()
        self.fixture()

    @staticmethod
    def write(path, data, mode=0o600):
        path.write_bytes(data)
        path.chmod(mode)

    def fixture(self, action="success", padding=0):
        program = '''#!/usr/bin/python3 -I
import json, os, socket, sys, time
assert sys.argv[1:] == ['exportar-seudonimos-portal-externo']
assert os.environ['HOME'] == '/home/vec'
assert os.environ['PATH'] == '/usr/bin:/bin'
assert os.environ['VEC_PORTAL_PROCESO'] == 'externo'
assert os.environ['VEC_EXECUTION_PROFILE'] == 'desarrollo'
assert os.environ['VEC_AUTH_MODE'] == 'desarrollo'
assert os.environ['VEC_DEVELOPMENT_GUARD'] == 'ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO'
assert 'POISON_SECRET' not in os.environ
assert not any('DATABASE' in name or 'PG' == name[:2] for name in os.environ)
root = os.environ['VEC_DEVELOPMENT_MATERIAL_DIR']
assert root.endswith('/material/runtime-externo')
try:
    open(root + '/identidad/candidato.json', 'ab')
except OSError:
    pass
else:
    raise AssertionError('material writable')
assert not os.path.exists(os.path.dirname(root) + '/custodia')
assert not os.path.exists(os.path.dirname(os.path.dirname(root)) + '/binary')
sock = socket.socket()
try:
    sock.connect(('127.0.0.1', 55441))
except OSError:
    pass
else:
    raise AssertionError('network reachable')
sock.close()
'''
        if action == "success":
            program += "os.write(1, " + repr(self.expected) + ")\nos.write(2, b'fixture-secret-never-print')\n"
        elif action == "partial":
            program += "os.write(1, b'{\\\"version\\\":1,')\n"
        elif action == "nonzero":
            program += "os.write(1, " + repr(self.expected) + ")\nos.write(2, b'fixture-secret-never-print')\nsys.exit(7)\n"
        elif action == "timeout":
            program += "os.write(1, b'{')\ntime.sleep(10)\n"
        elif action == "overflow":
            program += "os.write(1, b'x' * 400000)\n"
        elif action == "stderr_overflow":
            program += "os.write(2, b'x' * 100000)\n"
        program += "\n#" + "p" * padding + "\n"
        self.write(self.binary, program.encode(), 0o700)
        patch.object(alias, "BINARY_SHA", m.digest(self.binary.read_bytes())).start()
        patch.object(alias, "BINARY_BYTES", self.binary.stat().st_size).start()
        act = ("Commit fuente: " + alias.COMMIT + "\nBinario SHA256: " + alias.BINARY_SHA + "\n").encode()
        self.write(self.binary.with_name("ACTA.txt"), act)
        patch.object(alias, "BUILD_ACT_SHA", m.digest(act)).start()

    def run_export(self, **kwargs):
        return alias.export(Path(__file__).resolve().parents[2], self.binary, self.source, self.ack, self.root, self.output,
                            now=NOW, **kwargs)

    def test_real_fixture_process_receipt_and_replay_are_bound_to_bytes(self):
        before = {str(p.relative_to(self.root)): (p.stat().st_ino, p.stat().st_mtime_ns, p.read_bytes())
                  for p in self.root.rglob('*') if p.is_file()}
        with patch.dict(os.environ, {"POISON_SECRET": "must-not-inherit", "PGPASSWORD": "must-not-inherit"}):
            receipt = self.run_export()
        self.assertEqual(receipt['proceso']['exit_code'], 0)
        self.assertGreater(receipt['proceso']['pid'], 0)
        self.assertEqual(receipt['proceso']['stderr'], 'redacted')
        self.assertNotIn('fixture-secret-never-print', json.dumps(receipt))
        self.assertEqual((self.output / alias.OUTPUT).read_bytes(), self.expected)
        self.assertEqual(receipt['salida_sha256'], m.digest(self.expected))
        self.assertEqual(receipt['binding']['binary_sha256'], m.digest(self.binary.read_bytes()))
        self.assertEqual(receipt['pending_sha256'], m.digest((self.output / alias.PENDING).read_bytes()))
        pin = m.digest((self.output / alias.RECEIPT).read_bytes())
        output_before = {p.name: (p.stat().st_ino, p.stat().st_mtime_ns, p.read_bytes()) for p in self.output.iterdir()}
        with patch.object(alias, 'execute', side_effect=AssertionError('replay must never execute')):
            self.assertEqual(receipt, self.run_export(receipt_sha=pin))
            with self.assertRaisesRegex(alias.ExportError, 'pin_required'):
                self.run_export()
        self.assertEqual(before, {str(p.relative_to(self.root)): (p.stat().st_ino, p.stat().st_mtime_ns, p.read_bytes())
                                  for p in self.root.rglob('*') if p.is_file()})
        self.assertEqual(output_before, {p.name: (p.stat().st_ino, p.stat().st_mtime_ns, p.read_bytes()) for p in self.output.iterdir()})
        for p in self.output.iterdir():
            self.assertEqual(p.stat().st_mode & 0o777, 0o600)

    def test_binary_larger_than_stdout_limit_can_start_without_relaxing_stdout_bound(self):
        self.fixture(padding=alias.MAX_OUTPUT * 2)
        self.assertGreater(self.binary.stat().st_size, alias.MAX_OUTPUT)
        receipt = self.run_export()
        self.assertEqual(receipt["salida_bytes"], len(self.expected))

    def test_preflight_pin_failures_have_no_output_effect(self):
        cases = [('binary', self.binary, b'changed', 'binary'), ('act', self.binary.with_name('ACTA.txt'), b'changed', 'act'),
                 ('ack', self.ack, b'{}', 'ack'), ('source', self.source, b'{}', 'source'),
                 ('receipt', self.root / m.RECEIPT, b'{}', 'receipt'),
                 ('material', self.root / 'runtime-externo/idempotencia/g2-huella-solicitud.bin', b'x' * 32, 'bytes')]
        for name, path, data, expected in cases:
            with self.subTest(name=name):
                old = path.read_bytes()
                self.write(path, data, 0o700 if path == self.binary else 0o600)
                with self.assertRaises((alias.ExportError, m.OfflineMaterialError)):
                    self.run_export()
                self.assertFalse(self.output.exists())
                self.write(path, old, 0o700 if path == self.binary else 0o600)
        self.git.side_effect = alias.ExportError('git_main_pin_changed')
        with self.assertRaisesRegex(alias.ExportError, 'git_main_pin_changed'):
            self.run_export()
        self.assertFalse(self.output.exists())

    def test_failure_partial_timeout_and_limits_keep_pending_and_block_reexecution(self):
        for action in ('partial', 'nonzero', 'timeout', 'overflow', 'stderr_overflow'):
            with self.subTest(action=action):
                self.output = self.base / ('output-' + action)
                self.fixture(action)
                with self.assertRaises((alias.ExportError, ValueError, subprocess.SubprocessError)):
                    self.run_export(timeout=0.7 if action == 'timeout' else 10)
                self.assertTrue((self.output / alias.PENDING).is_file())
                self.assertFalse((self.output / alias.RECEIPT).exists())
                self.assertLessEqual((self.output / alias.OUTPUT).stat().st_size, alias.MAX_OUTPUT)
                pending = (self.output / alias.PENDING).read_bytes()
                with patch.object(alias, 'execute', side_effect=AssertionError('uncertain attempt must not repeat')):
                    with self.assertRaisesRegex(alias.ExportError, 'attempt_incomplete'):
                        self.run_export()
                self.assertEqual(pending, (self.output / alias.PENDING).read_bytes())

    def test_stdout_closed_schema_and_hmac_validation(self):
        good = json.loads(self.expected)
        variants = [b'{"version":1,"version":1,"cuentas":[]}', self.expected + b'{}', b'{"version":true,"cuentas":[]}']
        for changed in (good | {'extra': True}, good | {'cuentas': good['cuentas'] * 2},
                        good | {'cuentas': [good['cuentas'][0] | {'cuenta_ref': 'cta_other'}]},
                        good | {'cuentas': [good['cuentas'][0] | {'clave_version': True}]},
                        good | {'cuentas': [good['cuentas'][0] | {'cuenta_id_hmac': 'a' * 64}]},
                        good | {'cuentas': [good['cuentas'][0] | {'extra': 1}]}):
            variants.append(m.encoded(changed))
        for data in variants:
            with self.subTest(data_length=len(data)):
                with self.assertRaises((alias.ExportError, m.OfflineMaterialError, ValueError)):
                    alias.validate_stdout(data, self.files, self.bound)

    def test_replay_output_receipt_pending_and_material_changes_never_execute(self):
        self.run_export()
        pin = m.digest((self.output / alias.RECEIPT).read_bytes())
        for name in (alias.OUTPUT, alias.RECEIPT, alias.PENDING):
            with self.subTest(name=name):
                path = self.output / name
                old = path.read_bytes()
                self.write(path, old + b' ')
                with patch.object(alias, 'execute', side_effect=AssertionError('no replay launch')):
                    with self.assertRaises(alias.ExportError):
                        self.run_export(receipt_sha=pin)
                self.write(path, old)
        self.write(self.binary, self.binary.read_bytes() + b'\n', 0o700)
        with patch.object(alias, 'execute', side_effect=AssertionError('no replay launch')):
            with self.assertRaises(alias.ExportError):
                self.run_export(receipt_sha=pin)

    def test_symlinks_hardlinks_unknown_material_and_overlap_are_rejected(self):
        preserved = self.binary_dir / 'held'
        self.binary.rename(preserved)
        self.binary.symlink_to(preserved)
        with self.assertRaises(OSError):
            self.run_export()
        self.assertFalse(self.output.exists())
        self.binary.unlink()
        os.link(preserved, self.binary)
        with self.assertRaises(alias.ExportError):
            self.run_export()
        self.binary.unlink()
        preserved.rename(self.binary)
        (self.root / 'runtime-externo/unknown').mkdir(mode=0o700)
        with self.assertRaises(m.OfflineMaterialError):
            self.run_export()
        self.assertFalse(self.output.exists())
        self.output = self.root / 'alias'
        with self.assertRaisesRegex(alias.ExportError, 'overlaps'):
            self.run_export()

    def test_postflight_changed_material_does_not_seal_execution(self):
        execute = alias.execute
        def changed(*args):
            result = execute(*args)
            self.write(self.root / 'runtime-externo/portal-proceso.json', b'{}')
            return result
        with patch.object(alias, 'execute', side_effect=changed):
            with self.assertRaises((alias.ExportError, m.OfflineMaterialError)):
                self.run_export()
        self.assertTrue((self.output / alias.PENDING).exists())
        self.assertFalse((self.output / alias.RECEIPT).exists())

    def test_sandbox_start_failure_preserves_attempt_and_no_fallback(self):
        with patch.object(alias.subprocess, 'Popen', side_effect=OSError('secret fixture path')):
            with self.assertRaises(OSError):
                self.run_export()
        self.assertTrue((self.output / alias.PENDING).exists())
        with patch.object(alias, 'execute', side_effect=AssertionError('no retry')):
            with self.assertRaisesRegex(alias.ExportError, 'attempt_incomplete'):
                self.run_export()


if __name__ == '__main__':
    unittest.main()
