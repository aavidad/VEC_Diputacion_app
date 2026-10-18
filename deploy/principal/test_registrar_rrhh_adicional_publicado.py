import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from deploy.principal import registrar_rrhh_adicional_publicado as modulo


class RegistroRRHHAdicionalTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.raiz = Path(self.tmp.name)
        self.material = self.raiz / "material"
        self.material.mkdir(mode=0o700)
        (self.material / "identidad").mkdir(mode=0o700)
        (self.material / "mtls").mkdir(mode=0o700)
        self.persona = "per_" + "1" * 32
        self.cuenta = "cta_" + "2" * 32
        self.base = "prf_" + "3" * 32
        self.perfil = "prf_" + "4" * 32
        self.vinculo = "vca_" + "5" * 32
        self.rol = "rol:seguimiento_cese_ct_desarrollo:v1"
        self.preparacion = self.raiz / "preparacion.json"
        self.recibo = self.raiz / "recibo.json"
        self._escribir(self.material / "identidad/tecnico-rrhh-adicional.json", {
            "certificado": "mtls/tecnico-rrhh-adicional.crt",
            "identidad": "identidad/tecnico-rrhh-adicional-identidad.json",
            "sujeto": "desarrollo:segunda-rrhh", "persona_ref": self.persona,
            "cuenta_ref": self.cuenta, "perfil_ref": self.base})
        self._escribir(self.material / "identidad/tecnico-rrhh-adicional-identidad.json", {
            "subject": "desarrollo:segunda-rrhh", "certificate_sha256": "a" * 64,
            "roles": ["tecnico_rrhh"]})
        self._escribir(self.material / "identidad/tecnico-rrhh-adicional-semilla.json", {
            "sujeto": "desarrollo:segunda-rrhh", "certificado_sha256": "a" * 64,
            "persona_ref": self.persona, "cuenta_ref": self.cuenta,
            "perfil_base_ref": self.base,
            "rol_base_ref": "rol:tecnico_rrhh_desarrollo:v1",
            "ca37_organizacion_ref": "org_" + "a" * 32,
            "ca37_recibo_ref": "recibo_ca_identidad:" + "9" * 32})
        certificado = self.material / "mtls/tecnico-rrhh-adicional.crt"
        certificado.write_bytes(b"CERTIFICADO SINTETICO")
        certificado.chmod(0o600)
        self.pre = {"preparacion": {"persona_ref": self.persona, "cuenta_ref": self.cuenta,
                    "unidad_ref": "unidad:desarrollo:rrhh", "altas": [{
                        "rol_version_ref": self.rol, "perfil_ref": self.perfil,
                        "vinculo_ref": self.vinculo, "huella_sha256": "b" * 64}]}}
        self.respuesta = {"recibo": {"operacion_ref": "lote_admin:" + "6" * 32,
                          "acto_ref": "acto_admin:" + "7" * 32,
                          "recibo_ref": "recibo_admin:" + "8" * 32,
                          "huella_solicitud_sha256": "c" * 64,
                          "fuentes_sha256": "d" * 64,
                          "confirmado_en": "2026-10-18T12:00:00Z",
                          "cambios": [{"objetivo_persona_ref": self.persona,
                            "perfil_ref": self.perfil, "vinculo_ref": self.vinculo,
                            "rol_version_ref": self.rol, "unidad_ref": "unidad:desarrollo:rrhh",
                            "estado_posterior": "activo", "version_posterior": 1,
                            "operacion_ref": "lote_admin:" + "6" * 32,
                            "acto_ref": "acto_admin:" + "7" * 32,
                            "recibo_ref": "recibo_admin:" + "8" * 32,
                            "huella_antes_sha256": "b" * 64,
                            "huella_despues_sha256": "e" * 64,
                            "vigente_desde": "2026-10-18T12:00:00Z",
                            "vigente_hasta": "2026-12-18T12:00:00Z"}]}}
        self._escribir(self.preparacion, self.pre)
        self._escribir(self.recibo, self.respuesta)

    def _escribir(self, ruta, dato):
        ruta.write_text(json.dumps(dato), encoding="utf-8")
        ruta.chmod(0o600)

    def test_guarda_solo_perfil_aleatorio_y_recibo_del_lote_real(self):
        with mock.patch.object(modulo, "huella", return_value="a" * 64):
            dato = modulo.registrar(self.material, self.preparacion, self.recibo, self.rol)
            self.assertEqual(dato["perfil_ref"], self.perfil)
            self.assertNotEqual(dato["perfil_ref"], self.base)
            self.assertEqual(dato["vinculo_ref"], self.vinculo)
            self.assertEqual(dato["ca37_organizacion_ref"], "org_" + "a" * 32)
            self.assertEqual(dato["acto_ref"], self.respuesta["recibo"]["acto_ref"])
            ruta = self.material / "identidad/tecnico-rrhh-adicional-publicado.json"
            self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
            self.assertEqual(modulo.registrar(self.material, self.preparacion, self.recibo, self.rol), dato)

    def test_rechaza_recibo_ajeno_sin_crear_manifiesto(self):
        self.respuesta["recibo"]["cambios"][0]["perfil_ref"] = self.base
        self._escribir(self.recibo, self.respuesta)
        with mock.patch.object(modulo, "huella", return_value="a" * 64):
            with self.assertRaisesRegex(ValueError, "no publicó"):
                modulo.registrar(self.material, self.preparacion, self.recibo, self.rol)
        self.assertFalse((self.material / "identidad/tecnico-rrhh-adicional-publicado.json").exists())


if __name__ == "__main__":
    unittest.main()
