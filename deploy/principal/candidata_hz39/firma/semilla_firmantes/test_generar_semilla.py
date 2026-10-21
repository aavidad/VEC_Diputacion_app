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
        checkout = self.root / "checkout_publico"
        checkout.mkdir(mode=0o700)
        names = ("generar_semilla_p.py", "identidad.template.sql",
                 "asignaciones.template.sql", "firmantes.template.json",
                 "cargos-roles.template.json")
        for name in names:
            target = checkout / name
            target.write_bytes(GENERATOR.with_name(name).read_bytes())
            target.chmod(0o644)
        self.generator = checkout / "generar_semilla_p.py"
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
        return subprocess.run([sys.executable, str(self.generator),
            "--config", str(self.root / "config.json"),
            "--salida", str(self.root / output), *extra],
            capture_output=True, text=True, timeout=10)

    def run_with_future_clock(self, output):
        bootstrap = """import datetime as dt, importlib.util, sys
base = dt.datetime
class Future(base):
    @classmethod
    def now(cls, tz=None):
        return base(2027, 1, 15, tzinfo=tz)
spec = importlib.util.spec_from_file_location('semilla', sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
module.dt.datetime = Future
sys.argv = sys.argv[1:]
module.main()
"""
        return subprocess.run([sys.executable, "-c", bootstrap, str(self.generator),
            "--config", str(self.root / "config.json"),
            "--salida", str(self.root / output)],
            capture_output=True, text=True, timeout=10)

    def coherent_roles(self, end):
        base = self.run_generator("base")
        self.assertEqual(base.returncode, 0, base.stderr)
        cargos = json.loads((self.root / "base" / "cargos-firma-fuente.json").read_text())
        for row in cargos["cargos"]:
            row["vigente_hasta"] = end
        source = self.private("fuente.json", cargos)
        plan = {"cargos": [dict(row, version_rol_sha256=f"{index:064x}")
                           for index, row in enumerate(cargos["cargos"], 1)]}
        plan_path = self.private("plan.json", plan)
        receipt = {"estado": "permitido", "recibo": {
            "auditoria_ref": "auditoria:ficticia",
            "plan_sha256": hashlib.sha256(plan_path.read_bytes()).hexdigest(),
            "cargos": [{"version_rol_ref": "rol:" + row["rol_id"] + ":v1",
                        "version_rol_sha256": f"{index:064x}",
                        "control_sha256": "d" * 64}
                       for index, row in enumerate(cargos["cargos"], 1)]}}
        receipt_path = self.private("recibo.json", receipt)
        return ("--roles-recibo", str(receipt_path), "--roles-plan", str(plan_path),
                "--roles-fuente-publicada", str(source))

    def test_plantillas_publicas_generan_datos_privados_sin_asignaciones(self):
        writable = self.generator.with_name("identidad.template.sql")
        writable.chmod(0o664)
        rejected = self.run_generator("salida_abierta")
        self.assertNotEqual(rejected.returncode, 0)
        self.assertIn("plantilla pública inválida", rejected.stderr)
        self.assertFalse((self.root / "salida_abierta").exists())
        writable.chmod(0o644)
        for name in ("config.json", "acuse.json", "clave.bin"):
            self.assertEqual((self.root / name).stat().st_mode & 0o777, 0o600)
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
        base = self.run_generator("base")
        self.assertEqual(base.returncode, 0, base.stderr)
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

    def test_fuente_aut53_coherente_limitada_a_90_dias(self):
        now = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
        start = now - dt.timedelta(hours=1)
        args = self.coherent_roles((start + dt.timedelta(days=90)).isoformat().replace("+00:00", "Z"))
        valid = self.run_generator("valida", *args)
        self.assertEqual(valid.returncode, 0, valid.stderr)
        self.assertTrue((self.root / "valida" / "semilla-asignaciones-firma.sql").exists())
        invalid_end = (start + dt.timedelta(days=90, hours=1)).isoformat().replace("+00:00", "Z")
        source_path = self.root / "fuente.json"
        source = json.loads(source_path.read_text())
        for row in source["cargos"]:
            row["vigente_hasta"] = invalid_end
        self.private("fuente.json", source)
        plan_path = self.root / "plan.json"
        plan = json.loads(plan_path.read_text())
        for row in plan["cargos"]:
            row["vigente_hasta"] = invalid_end
        self.private("plan.json", plan)
        receipt_path = self.root / "recibo.json"
        receipt = json.loads(receipt_path.read_text())
        receipt["recibo"]["plan_sha256"] = hashlib.sha256(plan_path.read_bytes()).hexdigest()
        self.private("recibo.json", receipt)
        invalid = self.run_generator("excesiva", *args)
        self.assertNotEqual(invalid.returncode, 0)
        self.assertFalse((self.root / "excesiva").exists())
        for rows in (source["cargos"], plan["cargos"]):
            for row in rows:
                row["vigente_desde"] = row["vigente_desde"].replace("Z", "+00:00")
                row["vigente_hasta"] = row["vigente_hasta"].replace("Z", "+00:00")
        self.private("fuente.json", source)
        self.private("plan.json", plan)
        receipt["recibo"]["plan_sha256"] = hashlib.sha256(plan_path.read_bytes()).hexdigest()
        self.private("recibo.json", receipt)
        noncanonical = self.run_generator("no_utc_canonica", *args)
        self.assertNotEqual(noncanonical.returncode, 0)
        self.assertFalse((self.root / "no_utc_canonica").exists())

    def test_retirada_vencida_rechaza_certificados_incompletos(self):
        result = self.run_with_future_clock("retirada_vencida")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.root / "retirada_vencida").exists())


if __name__ == "__main__":
    unittest.main()
