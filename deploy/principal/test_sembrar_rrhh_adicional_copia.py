import json
from datetime import datetime, timedelta, timezone
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from deploy.principal import sembrar_rrhh_adicional_copia as modulo
from deploy.principal.preparar_rrhh_adicional_copia import referencia


class SemillaSegundoRRHHTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.raiz = Path(self.tmp.name)
        self.material = self.raiz / "material"
        self.material.mkdir(mode=0o700)
        (self.material / "identidad").mkdir(mode=0o700)
        (self.material / "mtls").mkdir(mode=0o700)
        self.acuse = self.raiz / "acuse.json"
        self.plan = self.raiz / "plan.json"
        self.ahora = datetime(2026, 10, 18, 12, tzinfo=timezone.utc)
        self.ca37_organizacion = "org_" + "a" * 32
        self.sujeto, self.sha = "desarrollo:segunda-rrhh", "a" * 64
        self.persona, self.cuenta = "per_" + "b" * 32, "cta_" + "d" * 32
        base = self.sujeto + "\0" + self.sha
        self._escribir(self.material / "identidad/tecnico-rrhh-adicional.json", {
            "certificado": "mtls/tecnico-rrhh-adicional.crt",
            "identidad": "identidad/tecnico-rrhh-adicional-identidad.json",
            "sujeto": self.sujeto, "persona_ref": self.persona,
            "cuenta_ref": "cta_" + "0" * 32,
            "perfil_ref": referencia("prf_", base + "\0perfil")})
        self._escribir(self.material / "identidad/tecnico-rrhh-adicional-identidad.json", {
            "subject": self.sujeto, "roles": ["tecnico_rrhh"], "certificate_sha256": self.sha})
        cert = self.material / "mtls/tecnico-rrhh-adicional.crt"
        cert.write_bytes(b"CERTIFICADO SINTETICO")
        cert.chmod(0o600)
        recibo = json.loads(Path("cmd/vec-provisionar-identidad-interna/testdata/recibo_aut57_permitido.json").read_text())
        r = recibo["recibo"]
        for autoridad in (r["is"], r["ca"]):
            autoridad["datos"]["persona_ref"] = self.persona
            autoridad["datos"]["cuenta_ordinaria_ref"] = self.cuenta
        r["ca"]["datos"]["organizacion_ref"] = self.ca37_organizacion
        self.recibo = recibo
        self.acuse_valido = json.loads(json.dumps(recibo))
        self._escribir(self.acuse, recibo)
        hasta = (self.ahora + timedelta(days=89)).isoformat().replace("+00:00", "Z")
        self._escribir(self.plan, {"huella_plan_sha256": r["plan_sha256"],
                                   "plan": {"operacion_ref": r["operacion_ref"],
                                            "persona": {"persona_ref": self.persona, "vigente_hasta": hasta},
                                            "organizacion": {"organizacion_ref": self.ca37_organizacion,
                                                             "vigente_hasta": hasta}}})

    def _escribir(self, ruta, dato):
        ruta.write_text(json.dumps(dato), encoding="utf-8")
        ruta.chmod(0o600)

    def test_ca37_real_y_sql_de_semilla_no_conceden_c22(self):
        with mock.patch.object(modulo, "huella", return_value=self.sha):
            dato = modulo.comprobar_acuse(self.material, self.acuse)
        self.assertEqual(dato["persona_ref"], self.persona)
        self.assertEqual(dato["cuenta_ref"], self.cuenta)
        self.assertEqual(dato["ca37_organizacion_ref"], self.ca37_organizacion)
        self.assertNotEqual(dato["ca37_organizacion_ref"], modulo.ORGANIZACION)
        self.assertEqual(modulo.comprobar_plan_ca37(self.plan, dato, self.ahora), self.ahora + timedelta(days=89))
        self.assertNotEqual(dato["cuenta_ref"], "cta_" + "0" * 32)
        ensayo, aplicacion = modulo.sentencia(dato, False), modulo.sentencia(dato, True)
        self.assertTrue(ensayo.endswith(b"ROLLBACK;\n"))
        self.assertTrue(aplicacion.endswith(b"COMMIT;\n"))
        self.assertIn(modulo.ROL_BASE.encode(), aplicacion)
        self.assertIn(modulo.ACCION_C22.encode(), aplicacion)
        self.assertIn(modulo.ORGANIZACION.encode(), aplicacion)
        self.assertNotIn(self.ca37_organizacion.encode(), aplicacion)
        self.assertIn(b"semilla RRHH: c22 ya concedida", aplicacion)
        self.assertNotIn(b"CREATE ROLE", aplicacion)
        self.assertNotIn(b"CREATE OR REPLACE", aplicacion)
        self.assertIn(b"efectos_identidad_interna_sintetica_v1", aplicacion)

    def test_rechaza_cuenta_no_acreditada_por_ca37(self):
        self.recibo["recibo"]["is"]["datos"]["cuenta_ordinaria_ref"] = "cta_" + "e" * 32
        self._escribir(self.acuse, self.recibo)
        with mock.patch.object(modulo, "huella", return_value=self.sha):
            with self.assertRaisesRegex(ValueError, "no corresponde"):
                modulo.comprobar_acuse(self.material, self.acuse)

    def test_rechaza_persona_cruzada_y_vigencia_inferior_a_60_dias(self):
        self.recibo["recibo"]["is"]["datos"]["persona_ref"] = "per_" + "e" * 32
        self._escribir(self.acuse, self.recibo)
        with mock.patch.object(modulo, "huella", return_value=self.sha):
            with self.assertRaisesRegex(ValueError, "no corresponde"):
                modulo.comprobar_acuse(self.material, self.acuse)
        self._escribir(self.acuse, self.acuse_valido)
        plan = json.loads(self.plan.read_text())
        plan["plan"]["persona"]["vigente_hasta"] = (self.ahora + timedelta(days=59)).isoformat().replace("+00:00", "Z")
        self._escribir(self.plan, plan)
        with mock.patch.object(modulo, "huella", return_value=self.sha):
            dato = modulo.comprobar_acuse(self.material, self.acuse)
        with self.assertRaisesRegex(ValueError, "60 a 90"):
            modulo.comprobar_plan_ca37(self.plan, dato, self.ahora)

    def test_fallo_de_ensayo_no_ejecuta_commit(self):
        copia = mock.Mock(material=self.material)
        copia.existe.return_value = True
        copia.sql_bytes.side_effect = RuntimeError("ensayo fallido")
        with mock.patch.object(modulo, "huella", return_value=self.sha):
            with self.assertRaisesRegex(RuntimeError, "ensayo fallido"):
                with mock.patch.object(modulo, "comprobar_plan_ca37", return_value=self.ahora + timedelta(days=89)):
                    modulo.sembrar(copia, self.acuse, self.plan)
        self.assertEqual(copia.sql_bytes.call_count, 1)
        self.assertFalse((self.material / "identidad/tecnico-rrhh-adicional-semilla.json").exists())


if __name__ == "__main__":
    unittest.main()
