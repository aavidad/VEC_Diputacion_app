"""Pruebas focales de la preparación offline de datos ficticios."""
import hashlib
import datetime as dt
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


GENERATOR = Path(__file__).with_name("generar_semilla_p.py")
ROLES = ("tecnico", "jefatura", "diputacion", "secretaria")


class SemillaFirmantesTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        acuse = {"estado": "permitido", "recibo": {
            "esquema": "vec.personal.unidad-inicial.v1", "auditoria_ref": "audit1",
            "recibo_ref": "recibo1", "unidad": {"organismo_ref": "org_ficticia",
                "unidad_ref": "unidad_ficticia", "nodo_ref": "nodo_ficticio", "revision": 1}}}
        self.private("acuse.json", acuse)
        self.private("clave.bin", b"k" * 32)
        self.config = {
            "version": 1, "copia_ref": "copia_ficticia",
            "organizacion_canonica_ref": "org_ficticia",
            "organizacion_procedencia_ref": "procedencia_ficticia",
            "organizacion_procedencia_sha256": "a" * 64,
            "organizacion_operativa_ref": "organizacion:ficticia",
            "unidad_ref": "unidad_ficticia", "nodo_ref": "nodo_ficticio",
            "hmac_g2_clave_archivo": str(self.root / "clave.bin"),
            "certificados": {role: "" for role in ROLES},
            "acuse_unidad_h10": str(self.root / "acuse.json"),
        }
        self.private("config.json", self.config)

    def private(self, name, value):
        path = self.root / name
        raw = value if isinstance(value, bytes) else json.dumps(value).encode()
        path.write_bytes(raw)
        path.chmod(0o600)
        return path

    def run_generator(self, output="salida", *extra):
        return subprocess.run([sys.executable, str(GENERATOR),
            "--config", str(self.root / "config.json"),
            "--salida", str(self.root / output), *extra],
            capture_output=True, text=True, timeout=10)

    def test_plantillas_publicas_generan_datos_privados_sin_asignaciones(self):
        for name in ("identidad.template.sql", "asignaciones.template.sql",
                     "firmantes.template.json", "cargos-roles.template.json"):
            self.assertEqual(GENERATOR.with_name(name).stat().st_mode & 0o777, 0o644)
        result = self.run_generator()
        self.assertEqual(result.returncode, 0, result.stderr)
        output = self.root / "salida"
        self.assertEqual({p.name for p in output.iterdir()}, {
            "datos-identidad-firmantes.sql", "cargos-firma-fuente.json",
            "firmantes-datos.pendiente.json", "preparacion.json"})
        self.assertTrue(all(p.stat().st_mode & 0o777 == 0o600 for p in output.iterdir()))
        self.assertEqual(output.stat().st_mode & 0o777, 0o700)
        self.assertIn("'2027-11-18T16:06:24Z'", (output / "datos-identidad-firmantes.sql").read_text())
        preparation = json.loads((output / "preparacion.json").read_text())
        self.assertFalse(preparation["asignaciones_sql_emitidas"])
        start = dt.datetime.fromisoformat(preparation["roles_vigente_desde"].replace("Z", "+00:00"))
        end = dt.datetime.fromisoformat(preparation["roles_vigente_hasta"].replace("Z", "+00:00"))
        self.assertGreaterEqual(end - start, dt.timedelta(days=60))
        self.assertLessEqual(end - start, dt.timedelta(days=90))

    def test_entradas_privadas_rechazan_permisos_y_enlaces(self):
        for name in ("config.json", "acuse.json", "clave.bin"):
            with self.subTest(name=name):
                path = self.root / name
                path.chmod(0o644)
                self.assertNotEqual(self.run_generator("salida_" + name).returncode, 0)
                self.assertFalse((self.root / ("salida_" + name)).exists())
                path.chmod(0o600)
        link = self.root / "acuse_enlace.json"
        link.symlink_to(self.root / "acuse.json")
        self.config["acuse_unidad_h10"] = str(link)
        self.private("config.json", self.config)
        self.assertNotEqual(self.run_generator("salida_enlace").returncode, 0)
        self.assertFalse((self.root / "salida_enlace").exists())

    def test_referencia_sql_inyectable_se_rechaza(self):
        self.config["organizacion_operativa_ref"] = "organizacion:ficticia'; DROP TABLE x; --"
        self.private("config.json", self.config)
        self.assertNotEqual(self.run_generator().returncode, 0)
        self.assertFalse((self.root / "salida").exists())

    def test_recibo_o_plan_ajeno_no_emite_asignaciones(self):
        self.assertEqual(self.run_generator("base").returncode, 0)
        source = self.root / "base" / "cargos-firma-fuente.json"
        cargos = json.loads(source.read_text())
        plan = {"cargos": [dict(row, version_rol_sha256="b" * 64)
                           for row in cargos["cargos"]]}
        plan_path = self.private("plan.json", plan)
        receipt = {"estado": "permitido", "recibo": {
            "auditoria_ref": "audit1", "plan_sha256": "c" * 64, "cargos": []}}
        receipt_path = self.private("recibo.json", receipt)
        args = ("--roles-recibo", str(receipt_path), "--roles-plan", str(plan_path),
                "--roles-fuente-publicada", str(source))
        self.assertNotEqual(self.run_generator("salida_recibo", *args).returncode, 0)
        self.assertFalse((self.root / "salida_recibo").exists())
        plan["cargos"][0]["organizacion_ref"] = "organizacion:ajena"
        self.private("plan.json", plan)
        receipt["recibo"]["plan_sha256"] = hashlib.sha256(plan_path.read_bytes()).hexdigest()
        self.private("recibo.json", receipt)
        self.assertNotEqual(self.run_generator("salida_plan", *args).returncode, 0)
        self.assertFalse((self.root / "salida_plan").exists())


if __name__ == "__main__":
    unittest.main()
