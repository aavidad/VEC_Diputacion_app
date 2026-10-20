"""Preparación de centros con productor Git y consultas simulados."""

import json
import importlib.util
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

spec = importlib.util.spec_from_file_location("vec_centros_prototipo", Path(__file__).parent / "recorrido_copia/centros.py")
centros = importlib.util.module_from_spec(spec)
spec.loader.exec_module(centros)


class CopiaFalsa:
    def __init__(self, raiz, repo):
        self.raiz = raiz
        self.nombre = "vec-copia-l-prueba"
        self.material = raiz / "material"
        (self.material / "ca").mkdir(parents=True)
        (self.material / "mtls").mkdir()
        (self.material / "identidad").mkdir()
        for ruta in ("ca/ca.crt", "ca/ca.key"):
            (self.material / ruta).write_text("ficticio")
        (self.material / "identidad/identidad.json").write_text(json.dumps({
            "subject": "desarrollo:rrhh:distinto", "roles": ["tecnico_rrhh"]}))
        self.estado = {"fuente": str(repo), "material_modo": "nuevo"}

    def leer(self):
        return self.estado

    def existe(self):
        return True


class CentrosTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.repo = Path(__file__).resolve().parents[2]
        self.copia = CopiaFalsa(Path(self.tmp.name), self.repo)
        self.centro = json.loads((self.repo / "deploy/principal/candidata_hz39/datos/seleccion.json").read_text())["centro_ref"]
        self.catalogo = {"version": 8, "revision": 3, "centers": ["centro:otro", self.centro],
                         "positions": [{"center": "centro:otro", "ref": "puesto:ajeno1"},
                                       {"center": self.centro, "ref": "puesto:primero"},
                                       {"center": self.centro, "ref": "puesto:segundo"}]}
        def certificar(material, nombre, sujeto, *, days):
            self.assertEqual(days, 90)
            for extension in (".key", ".crt", ".p12", ".p12.password"):
                (material / "mtls" / (nombre + extension)).write_text("ficticio")
        self.productor = SimpleNamespace(
            ROLES={"centro_solicitante": ("solicitante", "solicitante", "solicitante_centro"),
                   "ratificador": ("ratificador", "ratificador", "ratificador_centro")},
            read_catalog=Mock(return_value=self.catalogo),
            center_bindings=centros._productor(self.repo).center_bindings,
            fresh_profile_names=lambda: ({"centro_solicitante": "Lucía Navarro Molina",
                                          "ratificador": "Diego Serrano Vega"}, "ficticia"),
            certificate=Mock(side_effect=certificar),
            cert_hash=Mock(side_effect=["a" * 64, "b" * 64]))

    def test_prepara_dos_identidades_propias_y_centro_publicado(self):
        with patch.object(centros, "_productor", return_value=self.productor):
            resultado = centros.preparar(self.copia)
        self.assertEqual(resultado["centro_ref"], self.centro)
        self.assertEqual(resultado["puestos_ref"], ["puesto:primero", "puesto:segundo"])
        self.assertEqual(self.productor.certificate.call_count, 2)
        archivo = self.copia.material / "identidad/centros.json"
        manifiesto = json.loads(archivo.read_text())
        self.assertEqual(manifiesto["entradas"][0]["ratificador_subject"],
                         resultado["perfiles"][1]["subject"])
        self.assertEqual(archivo.stat().st_mode & 0o077, 0)
        self.assertEqual(resultado["perfiles_estado"], "pendientes_del_siguiente_arranque")
        self.assertNotEqual(resultado["perfiles"][0]["subject"], "desarrollo:rrhh:distinto")
        self.assertEqual(resultado["entorno"]["VEC_PERSONAL_ORGANIZACION_VERSION"], "8")

    def test_sin_dos_puestos_del_centro_no_crea_certificados(self):
        self.catalogo["positions"].pop()
        with patch.object(centros, "_productor", return_value=self.productor), \
                self.assertRaisesRegex(ValueError, "dos puestos"):
            centros.preparar(self.copia)
        self.productor.certificate.assert_not_called()

    def test_material_existente_se_conserva(self):
        (self.copia.material / "identidad/centros.json").write_text("{}")
        with self.assertRaisesRegex(ValueError, "ya se prepararon"):
            centros.preparar(self.copia)

    def test_huella_de_certificado_repetida_no_publica_manifiesto(self):
        self.productor.cert_hash.side_effect = ["a" * 64, "a" * 64]
        with patch.object(centros, "_productor", return_value=self.productor), \
                self.assertRaisesRegex(ValueError, "certificado"):
            centros.preparar(self.copia)
        self.assertFalse((self.copia.material / "identidad/centros.json").exists())


if __name__ == "__main__":
    unittest.main()
