"""Puerta local de seguridad: no requiere navegador ni servidor."""

from __future__ import annotations

import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from recorrer import NoEjecutado, preparar


class Precondiciones(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.raiz = Path(self.temp.name)
        self.binario = self.raiz / "vec-server"
        self.binario.write_bytes(b"binario sintetico de prueba")
        self.certificado = self.raiz / "cert.pem"
        self.clave = self.raiz / "clave.pem"
        self.certificado.write_bytes(b"material ficticio")
        self.clave.write_bytes(b"material ficticio")
        self.acta = self.raiz / "acta.json"
        self.origen = "https://127.0.0.1:18443"
        self.datos = {
            "origen": self.origen,
            "clon_sintetico": True,
            "clon_ref": "clon:prueba",
            "hitos_verificados": ["H3", "H4", "H5"],
            "binario": str(self.binario),
            "binario_sha256": hashlib.sha256(self.binario.read_bytes()).hexdigest(),
        }

    def guardar(self):
        self.acta.write_text(json.dumps(self.datos), encoding="utf-8")

    def test_sin_acta_no_ejecuta(self):
        with self.assertRaisesRegex(NoEjecutado, "acta"):
            preparar(self.origen, self.acta, self.certificado, self.clave)

    def test_hito_o_huella_incompleta_no_ejecuta(self):
        self.datos["hitos_verificados"] = ["H3", "H4"]
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "H3, H4 y H5"):
            preparar(self.origen, self.acta, self.certificado, self.clave)
        self.datos["hitos_verificados"].append("H5")
        self.datos["binario_sha256"] = "0" * 64
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "binario"):
            preparar(self.origen, self.acta, self.certificado, self.clave)

    def test_acta_malformada_falla_cerrada(self):
        self.datos["hitos_verificados"] = 5
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "H3, H4 y H5"):
            preparar(self.origen, self.acta, self.certificado, self.clave)
        self.datos["hitos_verificados"] = ["H3", "H4", "H5"]
        self.datos["binario"] = None
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "binario"):
            preparar(self.origen, self.acta, self.certificado, self.clave)
        with self.assertRaisesRegex(NoEjecutado, "puerto"):
            preparar("https://127.0.0.1:invalido", self.acta, self.certificado, self.clave)

    def test_origen_externo_o_material_ausente_no_ejecuta(self):
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "origen"):
            preparar("https://cidonia.example", self.acta, self.certificado, self.clave)
        self.clave.unlink()
        with self.assertRaisesRegex(NoEjecutado, "clave"):
            preparar(self.origen, self.acta, self.certificado, self.clave)

    def test_acta_completa_permita_pasar_solo_la_puerta(self):
        self.guardar()
        with patch("recorrer.shutil.which", return_value="/usr/bin/google-chrome"):
            origen, chrome = preparar(self.origen, self.acta, self.certificado, self.clave)
        self.assertEqual(origen, self.origen)
        self.assertEqual(chrome, Path("/usr/bin/google-chrome"))


if __name__ == "__main__":
    unittest.main()
