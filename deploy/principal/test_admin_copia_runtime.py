"""Pruebas focales de la preparación privada de runtime ADMIN."""
import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parent / "admin_copia"
sys.path.insert(0, str(SCRIPT))
import preparar_runtime  # noqa: E402


def escribir(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data))


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


class RuntimeAdminTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.run = self.root / "run"
        self.kit = self.root / "kit"
        self.sid = "1234567890123456789"
        self.run.mkdir()
        self.kit.mkdir()
        self.raiz = self.root / "raiz.key"
        self.ca = self.root / "ca.pem"
        self.motivo = self.root / "motivo.json"
        self.raiz.write_bytes(b"semilla ficticia")
        self.ca.write_bytes(b"ca ficticia")
        escribir(self.motivo, {"catalogo_id": "motivos", "catalogo_version": 1,
                               "catalogo_huella_sha256": "a" * 64, "entrada_clave": "razon"})
        (self.kit / "plantilla-runtime.json").write_bytes((SCRIPT / "datos/plantilla_runtime.json").read_bytes())
        for name in preparar_runtime.RUNTIME_TEXTOS[:-1]:
            (self.kit / name).write_bytes((SCRIPT / name).read_bytes())
        escribir(self.kit / "source-manifest.json", {"version": 1, "files": {}})
        escribir(self.kit / "build-config.json", {"repo_revision": "b" * 40,
                                                       "sid_copia": self.sid, "host_admin": "admin.localhost"})
        escribir(self.kit / "contrato-copia-local-admin-v1.json", {"version": 2,
                "revision_candidata": "b" * 40, "sid_copia": self.sid,
                "puerto_admin": 18443, "conjunto_capacidades": 7, "vigencia_dias": 89})
        escribir(self.kit / "manifiesto-kit.json", {"estado": "compilado_sin_actos",
                "revision": "b" * 40, "sid_copia": self.sid, "puerto_admin": 18443,
                "textos_sha256": {name: sha(self.kit / name) for name in preparar_runtime.RUNTIME_TEXTOS},
                "contrato_sha256": sha(self.kit / "contrato-copia-local-admin-v1.json"),
                "fuente_sha256": sha(self.kit / "source-manifest.json")})
        cfg = {"proveedor": {"directorio_material": str(self.root / "material"),
                "ruta_configuracion_hmac": str(self.root / "hmac.json"),
                "espacio_identidad": "https://localhost/vec/desarrollo/identidad",
                "dominio_ref": "idh_test", "espacio_clave": "identidad.test",
                "dominio_hmac": "identidad.test.v1"},
               "bootstrap": {"audiencia_administrativa": "admin:test:v1"}}
        escribir(self.run / "config.json", cfg)
        escribir(self.run / "contrato-arranque.json", {"version": 1, "revision": "b" * 40,
                "sid_copia": self.sid, "puerto_admin": 18443, "conjunto_capacidades": 7,
                "kit_manifest_sha256": sha(self.kit / "manifiesto-kit.json"),
                "kit_contract_sha256": sha(self.kit / "contrato-copia-local-admin-v1.json"),
                "config_sha256": sha(self.run / "config.json")})
        originales, plan, certs, cuentas = [], [], [], []
        ca_der = self.run / "fuentes/ca.der"
        ca_der.parent.mkdir(parents=True, exist_ok=True)
        ca_der.write_bytes(b"ca derivada ficticia")
        ca_sha = sha(ca_der)
        for n in (1, 2):
            ref = f"persona{n}"
            originales.append({"persona_ref": ref, "sujeto_original": f"sujeto{n}",
                "cuenta_privilegiada_original": f"priv{n}", "cuenta_ordinaria_original": f"ord{n}"})
            der = self.run / f"fuentes/cliente-{n}.der"
            der.write_bytes(f"certificado {n}".encode())
            plan.append({"persona_ref": ref, "cuenta_ref": f"cuenta{n}",
                "certificado_admin": {"huella_sha256": sha(der), "ca_huella_sha256": ca_sha}})
            certs.append({"persona_ref": ref, "cert_der_sha256": sha(der), "ca_der_sha256": ca_sha})
            cuentas.append({"persona_ref": ref, "cuenta_privilegiada_ref": f"cuenta{n}",
                            "cuenta_ordinaria_ref": f"ordinaria{n}"})
        escribir(self.run / "fuentes/originales.json", {"personas": originales})
        escribir(self.run / "fuentes/certificados.json", {"certificados": certs})
        escribir(self.run / "fuentes/material-hmac.raw", {"dominio_hmac_ref": "idh_test",
                "clave_hmac_id": "test", "clave_hmac_version": 1, "fuente_ref": "fuente-test"})
        escribir(self.run / "fuentes/fuente.json", {"fuente_hmac": {"huella_sha256": sha(self.run / "fuentes/material-hmac.raw")},
                "politica_admin": {"ca_sha256": ca_sha}, "personas": [{"persona_ref": x["persona_ref"]} for x in originales]})
        escribir(self.run / "bootstrap/plan-v3.json", {"plan": {"personas": plan}})
        for phase in ("fuentes", "unidad", "bootstrap"):
            receipt = {"estado": "permitido", "auditoria_intento": "intento-test",
                       "recibo": {"auditoria_ref": "audit-test"}}
            if phase == "fuentes":
                receipt["recibo"]["is"] = {"datos": {"personas": cuentas}}
            escribir(self.run / phase / "acuse-primero.json", receipt)

    def ejecutar(self, sid=None):
        argv = ["preparar_runtime.py", "--run", str(self.run), "--kit", str(self.kit),
                "--instancia", "ensayo1", "--dia-gobierno", "20261020a",
                "--semilla-raiz", str(self.raiz), "--ca-pg", str(self.ca),
                "--pg-host", "127.0.0.1", "--pg-base", "postgres", "--puerto-pg", "55441",
                "--sid-copia", sid or self.sid, "--vigente-hasta-config", "2027-01-01T00:00:00Z",
                "--retirada-en", "2027-01-02T00:00:00Z", "--redes", "127.0.0.1/32",
                "--motivo-publicado", str(self.motivo)]
        with patch.object(sys, "argv", argv):
            preparar_runtime.main()

    def test_dos_admin_distintos_y_entrada_ligada(self):
        self.ejecutar()
        ident = json.loads((self.run / "vec-admin/identificadores/identificadores.json").read_text())
        entry = json.loads((self.run / "entrada-vec-admin.json").read_text())
        self.assertEqual({x["cuenta_ref"] for x in ident["entradas"]}, {"cuenta1", "cuenta2"})
        self.assertEqual(entry["sid_copia"], self.sid)
        self.assertEqual(entry["kit_dir"], str(self.kit))

    def test_sid_ajeno_cierra_sin_salida(self):
        with self.assertRaises(SystemExit):
            self.ejecutar("9999999999999999999")
        self.assertFalse((self.run / "vec-admin").exists())

    def test_certificado_repetido_cierra_sin_salida(self):
        path = self.run / "fuentes/certificados.json"
        data = json.loads(path.read_text())
        data["certificados"][1]["cert_der_sha256"] = data["certificados"][0]["cert_der_sha256"]
        escribir(path, data)
        with self.assertRaises(SystemExit):
            self.ejecutar()
        self.assertFalse((self.run / "vec-admin").exists())

    def test_no_reutiliza_salida_preparada(self):
        self.ejecutar()
        with self.assertRaises(SystemExit):
            self.ejecutar()

    def test_manifiesto_modificado_cierra_sin_salida(self):
        path = self.kit / "preparar_vec_admin.py"
        path.write_bytes(path.read_bytes() + b"\n# cambio ajeno\n")
        with self.assertRaises(SystemExit):
            self.ejecutar()
        self.assertFalse((self.run / "vec-admin").exists())

    def test_recibo_de_otro_plan_no_acredita_gobierno(self):
        gov = self.run / "gobierno/20261020a"
        escribir(gov / "plan.json", {"version": 2, "conjunto_version": 7,
                "clave_ordenes": list(range(12)), "preimagen_sha256": "a" * 64,
                "caduca_en": "2026-10-20T20:00:00Z", "configuracion": {"revision": "rev-test"}})
        escribir(gov / "material.json", {"claves": [{"audiencia": f"aud.{i}"} for i in range(12)]})
        escribir(gov / "configuracion-material.json", {"Gobierno": {"Revision": "rev-test"}})
        escribir(gov / "aprobacion-candidata.json", {"plan_sha256": sha(gov / "plan.json"),
                "material_sha256": sha(gov / "material.json"), "preimagen_sha256": "a" * 64,
                "caduca_en": "2026-10-20T20:00:00Z"})
        escribir(gov / "acuse-aplicar.json", {"estado": "permitido", "auditoria_intento": "intento",
                "recibo": {"auditoria_ref": "recibo", "plan_sha256": "b" * 64,
                           "material_sha256": sha(gov / "material.json"),
                           "preimagen_sha256": "a" * 64, "configuracion_ref": "rev-test"}})
        with self.assertRaisesRegex(ValueError, "recibo de gobierno ajeno"):
            preparar_runtime.validar_gobierno(self.run, "20261020a", aplicado=True)


if __name__ == "__main__":
    unittest.main()
