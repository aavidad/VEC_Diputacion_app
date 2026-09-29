"""Pruebas focales del capturador de manuales."""

from __future__ import annotations

import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

from PIL import Image


RUTA = Path(__file__).resolve().parents[1] / "manuales" / "capturar_y_anotar.py"
ESPEC = importlib.util.spec_from_file_location("capturar_y_anotar", RUTA)
assert ESPEC and ESPEC.loader
capturador = importlib.util.module_from_spec(ESPEC)
ESPEC.loader.exec_module(capturador)


class ContratoCapturas(unittest.TestCase):
    def test_solo_permite_origen_loopback_sin_credenciales(self):
        self.assertEqual(capturador.validar_url_local("http://127.0.0.1:8080/", origen=True),
                         "http://127.0.0.1:8080")
        for url in ("https://example.org", "http://user:pass@localhost:8000",
                    "http://localhost:8000/?token=secreto", "file:///etc/passwd",
                    "http://localhost:8000/ruta"):
            with self.subTest(url=url), self.assertRaises(ValueError):
                capturador.validar_url_local(url, origen=True)
        self.assertFalse(capturador._peticion_local("http://127.0.0.1:9999/datos",
                                                   "http://127.0.0.1:8000"))
        self.assertTrue(capturador._peticion_local("http://127.0.0.1:8000/app.js?v=42",
                                                  "http://127.0.0.1:8000"))
        self.assertFalse(capturador._peticion_local("http://user:pass@127.0.0.1:8000/app.js",
                                                   "http://127.0.0.1:8000"))
        self.assertFalse(capturador._peticion_local("http://127.0.0.1:8000/app.js?access_token=secreto",
                                                   "http://127.0.0.1:8000"))
        with self.assertRaises(ValueError):
            capturador._certificados_externos("http://localhost:8000", Path("/tmp/a"), Path("/tmp/b"), None)
        with tempfile.TemporaryDirectory() as temporal:
            cert = Path(temporal) / "cliente.crt"
            key = Path(temporal) / "cliente.key"
            cert.write_text("certificado de prueba")
            key.write_text("clave de prueba")
            opciones = capturador._certificados_externos("https://localhost:8443", cert, key, None)
            self.assertEqual(opciones[0]["origin"], "https://localhost:8443")
            self.assertEqual(opciones[0]["certPath"], str(cert))

    def test_rechaza_datos_sensibles_y_acciones_que_escriben(self):
        escenario = {"pantallas": [{"clave": "inicio", "ruta": "/", "pasos": [],
                                     "marcas": [{"numero": 1, "selector": "h1",
                                                 "texto": "Título", "tipo": "recuadro"}], "ocultar": []}]}
        self.assertEqual(capturador.validar_escenario(escenario), escenario)
        for cambio in (lambda p: p.update(ruta="/?token=123"),
                       lambda p: p.update(pasos=[{"accion": "rellenar", "selector": "input"}]),
                       lambda p: p.update(pasos=[{"accion": "rellenar", "selector": "input",
                                                  "valor": "ana@example.org"}]),
                       lambda p: p["marcas"][0].update(texto="ana@example.org")):
            copia = json.loads(json.dumps(escenario))
            cambio(copia["pantallas"][0])
            with self.assertRaises(ValueError):
                capturador.validar_escenario(copia)
        escenario["pantallas"][0]["pasos"] = [
            {"accion": "rellenar", "selector": "#asunto", "valor": "Ejemplo sintético"},
            {"accion": "seleccionar", "selector": "#tipo", "valor": "temporal"}]
        self.assertEqual(capturador.validar_escenario(escenario), escenario)

    def test_ensayo_captura_dos_tamanos_y_manifiesto_sin_url(self):
        with tempfile.TemporaryDirectory() as temporal:
            salida = Path(temporal)
            manifiesto = capturador.ejecutar_ensayo(salida)
            datos = json.loads(manifiesto.read_text(encoding="utf-8"))
            self.assertEqual(datos["tipo"], "ensayo-sintetico")
            self.assertEqual(len(datos["capturas"]), 2)
            self.assertNotIn("127.0.0.1", manifiesto.read_text(encoding="utf-8"))
            for captura in datos["capturas"]:
                with Image.open(salida / captura["archivo"]) as imagen:
                    self.assertEqual(imagen.size, (captura["ancho"], captura["alto"]))
                self.assertEqual([marca["numero"] for marca in captura["marcas"]], [1, 2])


if __name__ == "__main__":
    unittest.main()
