"""Preparación aislada de firma, sin Xvfb ni certificados reales."""
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import entorno_firma as firma


class XvfbFicticio:
    pid = 4242

    def __init__(self):
        self.terminated = False

    def poll(self):
        return None

    def terminate(self):
        self.terminated = True

    def wait(self, timeout=None):
        return 0


class EntornoFirmaTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "instancia con %"
        self.root.mkdir(mode=0o755)
        self.root.chmod(0o755)
        self.output_parent = Path(self.temp.name) / "salida privada con %"
        self.output_parent.mkdir(mode=0o700)
        self.fw = self.output_parent / "firma-web"
        self.binary = self.root / "binario-ficticio"
        self.binary.write_bytes(b"grxfirma-afirmauri-ficticio")
        self.binary.chmod(0o700)
        self.sha = hashlib.sha256(self.binary.read_bytes()).hexdigest()
        self.probes = []
        self.xauth_calls = []
        self.inputs = {}
        for role in firma.ROLES:
            if role == "tecnico":
                base = self.root / "material/mtls"
                p12, password = base / "cliente.p12", base / "cliente.p12.password"
            else:
                base = self.root / "recorrido/firmantes/certificados" / ("rrhh_" + role)
                p12, password = base / "cliente.p12", base / "pkcs12.password"
            base.mkdir(parents=True)
            p12.write_bytes(("p12-" + role).encode())
            password.write_text("clave-" + role)
            p12.chmod(0o600)
            password.chmod(0o600)
            self.inputs[role] = (p12, password)

    def argv(self, sha=None):
        return ["--raiz-instancia", str(self.root), "--salida", str(self.fw),
                "--origen", "https://127.0.0.1:18428",
                "--binario-afirmauri", str(self.binary), "--sha-afirmauri", sha or self.sha]

    def version(self, command, **kwargs):
        class Result:
            returncode = 0
            stdout = b"GrxFirma AfirmaURI 0.0.107"
            stderr = b""
        if command[0] == "/usr/bin/xauth":
            self.xauth_calls.append((command, kwargs))
            self.assertEqual(command[-1], "-")
            self.assertEqual(Path(command[3]).stat().st_mode & 0o777, 0o600)
            self.assertNotIn(b"MIT-MAGIC-COOKIE-1", " ".join(command).encode())
            return Result()
        self.probes.append(command)
        self.assertNotEqual(Path(command[0]), self.binary)
        self.assertEqual(hashlib.sha256(Path(command[0]).read_bytes()).hexdigest(), self.sha)
        self.assertEqual(Path(command[0]).stat().st_mode & 0o777, 0o700)
        return Result()

    def xvfb(self, command, **kwargs):
        self.assertIn("-auth", command)
        self.assertEqual(Path(command[command.index("-auth") + 1]).stat().st_mode & 0o777, 0o600)
        os.write(kwargs["pass_fds"][0], b"42\n")
        return XvfbFicticio()

    def test_sha_o_permisos_invalidos_no_crean_entorno(self):
        with patch.object(firma.subprocess, "run", side_effect=self.version) as probe:
            with self.assertRaises(firma.Corte):
                firma.main(self.argv("0" * 64))
            probe.assert_not_called()
        self.assertFalse(self.fw.exists())
        self.inputs["secretaria"][1].chmod(0o644)
        with patch.object(firma.subprocess, "run", side_effect=self.version), patch.object(firma.subprocess, "Popen") as xvfb:
            with self.assertRaises(firma.Corte):
                firma.main(self.argv())
            xvfb.assert_not_called()
        self.assertFalse(self.fw.exists())

    def test_cuatro_entornos_distintos_y_confianza_solo_origen(self):
        original = {role: p12.read_bytes() for role, (p12, _) in self.inputs.items()}
        with patch.object(firma.subprocess, "run", side_effect=self.version), patch.object(firma.subprocess, "Popen", side_effect=self.xvfb):
            self.assertEqual(firma.main(self.argv()), 0)
        fw = self.fw
        self.assertEqual(self.root.stat().st_mode & 0o777, 0o755)
        self.assertEqual(fw.stat().st_mode & 0o777, 0o700)
        self.assertFalse((self.root / "firma-web").exists())
        self.assertEqual({role: p12.read_bytes() for role, (p12, _) in self.inputs.items()}, original)
        self.assertEqual((fw / "display").read_text().strip(), "42")
        self.assertEqual(len(self.probes), 1)
        self.assertEqual(len(self.xauth_calls), 2)
        self.assertEqual(self.xauth_calls[0][1]["input"].split()[-1],
                         self.xauth_calls[1][1]["input"].split()[-1])
        self.assertIn(b"add :0", self.xauth_calls[0][1]["input"])
        self.assertIn(b"add :42", self.xauth_calls[1][1]["input"])
        self.assertEqual((fw / "xvfb.pid").stat().st_mode & 0o777, 0o600)
        p12s = []
        for role in firma.ROLES:
            base = fw / role
            p12s.append((base / "p12/firmante.p12").read_bytes())
            self.assertEqual((base / "p12.password").stat().st_mode & 0o777, 0o600)
            cfg = base / "home/.config/grxfirma"
            self.assertEqual(json.loads((cfg / "trusted-domains.json").read_text()), {"https://127.0.0.1:18428": "allowed"})
            self.assertFalse(json.loads((cfg / "config.json").read_text())["rest_habilitado"])
            self.assertTrue(json.loads((cfg / "settings.json").read_text())["confirmToSign"])
            desktop = (base / "xdg-data/applications/grxfirma-copia.desktop").read_text()
            self.assertIn('Exec="', desktop)
            self.assertIn("salida privada con %%", desktop)
            self.assertIn(" %u", desktop)
        self.assertEqual(len(set(p12s)), 4)
        launcher = (fw / "lanzar-grxfirma.py").read_text()
        self.assertIn("os.execve", launcher)
        self.assertIn("'XAUTHORITY':str(base / 'Xauthority')", launcher)
        self.assertIn("'XDG_CONFIG_HOME':str(home / 'home/.config')", launcher)
        self.assertNotIn("clave-tecnico", launcher)

    def test_no_reescribe_y_limpia_xvfb_propiedad_propia(self):
        fw = self.fw
        fw.mkdir()
        (fw / "conservar").write_text("original")
        with patch.object(firma.subprocess, "run", side_effect=self.version), patch.object(firma.subprocess, "Popen") as xvfb:
            with self.assertRaises(firma.Corte):
                firma.main(self.argv())
            xvfb.assert_not_called()
        self.assertEqual((fw / "conservar").read_text(), "original")
        (fw / "conservar").unlink()
        fw.rmdir()
        process = XvfbFicticio()
        with patch.object(firma.subprocess, "run", side_effect=self.version), patch.object(firma.subprocess, "Popen", return_value=process), patch.object(firma.select, "select", return_value=([], [], [])):
            with self.assertRaises(firma.Corte):
                firma.main(self.argv())
        self.assertTrue(process.terminated)
        self.assertFalse(fw.exists())
        self.assertEqual({role: p12.read_bytes() for role, (p12, _) in self.inputs.items()},
                         {role: ("p12-" + role).encode() for role in firma.ROLES})
        interrupted = XvfbFicticio()
        with patch.object(firma.subprocess, "run", side_effect=self.version), patch.object(firma.subprocess, "Popen", return_value=interrupted), patch.object(firma.select, "select", side_effect=KeyboardInterrupt):
            with self.assertRaises(KeyboardInterrupt):
                firma.main(self.argv())
        self.assertTrue(interrupted.terminated)
        self.assertFalse(fw.exists())

    def test_salida_obligatoria_y_parent_inseguro_o_enlace_rechazados(self):
        args = self.argv()
        with self.assertRaises(SystemExit):
            firma.main(args[:2] + args[4:])
        self.assertFalse(self.fw.exists())
        self.output_parent.chmod(0o777)
        try:
            with patch.object(firma.subprocess, "run", side_effect=self.version), patch.object(firma.subprocess, "Popen") as xvfb:
                with self.assertRaises(firma.Corte):
                    firma.main(args)
                xvfb.assert_not_called()
            self.assertFalse(self.fw.exists())
        finally:
            self.output_parent.chmod(0o700)
        self.fw.symlink_to(self.root, target_is_directory=True)
        with patch.object(firma.subprocess, "run", side_effect=self.version):
            with self.assertRaises(firma.Corte):
                firma.main(args)
        self.assertTrue(self.fw.is_symlink())
        self.fw.unlink()

    def test_ancestros_y_rutas_canonicas_rechazados_antes_de_sonda(self):
        args = self.argv()
        cases = ((self.root / "firma-web", self.root),
                 (self.output_parent / "../instancia con %/firma-web", self.root / "sub/.."))
        for output, source in cases:
            with self.subTest(output=str(output)):
                changed = args.copy()
                changed[1], changed[3] = str(source), str(output)
                with patch.object(firma.subprocess, "run", side_effect=self.version) as probe:
                    with self.assertRaises(firma.Corte):
                        firma.main(changed)
                    probe.assert_not_called()
        ancestor = Path(self.temp.name) / "abuelo"
        ancestor.mkdir(mode=0o700)
        ancestor.chmod(0o777)
        private = ancestor / "privado"
        private.mkdir(mode=0o700)
        changed = args.copy()
        changed[3] = str(private / "firma-web")
        with patch.object(firma.subprocess, "run", side_effect=self.version) as probe:
            with self.assertRaisesRegex(firma.Corte, "ancestro"):
                firma.main(changed)
            probe.assert_not_called()
        self.assertFalse((private / "firma-web").exists())
        Path(self.temp.name).chmod(0o777)
        try:
            with patch.object(firma.subprocess, "run", side_effect=self.version) as probe:
                with self.assertRaisesRegex(firma.Corte, "ancestro"):
                    firma.main(args)
                probe.assert_not_called()
        finally:
            Path(self.temp.name).chmod(0o700)
        original_lstat = Path.lstat

        def foreign_uid(path, *a, **kw):
            info = original_lstat(path, *a, **kw)
            if path == self.output_parent:
                fields = list(info)
                fields[4] = os.getuid() + 1
                return os.stat_result(fields)
            return info

        with patch.object(Path, "lstat", foreign_uid), patch.object(firma.subprocess, "run", side_effect=self.version) as probe:
            with self.assertRaisesRegex(firma.Corte, "ancestro"):
                firma.main(args)
            probe.assert_not_called()
        self.assertFalse(self.fw.exists())


if __name__ == "__main__":
    unittest.main()
