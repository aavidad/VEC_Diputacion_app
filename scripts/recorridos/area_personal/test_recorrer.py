"""Puerta local de seguridad: no requiere navegador ni servidor."""

from __future__ import annotations

import hashlib
import json
import shutil
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest.mock import patch

from recorrer import NoEjecutado, filtrar_red, preparar


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


class Redirecciones(unittest.TestCase):
    @unittest.skipUnless(shutil.which("google-chrome"), "falta Chrome del sistema")
    def test_302_local_a_otro_puerto_no_alcanza_destino(self):
        from playwright.sync_api import Error, sync_playwright

        alcanzadas = []

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                alcanzadas.append(self.path)
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"destino")

            def log_message(self, *_):
                pass

        destino = ThreadingHTTPServer(("127.0.0.1", 0), Destino)
        hilo_destino = threading.Thread(target=destino.serve_forever, daemon=True)
        hilo_destino.start()
        self.addCleanup(hilo_destino.join, 2)
        self.addCleanup(destino.server_close)
        self.addCleanup(destino.shutdown)

        class Origen(BaseHTTPRequestHandler):
            def do_GET(self):
                if self.path == "/salir":
                    self.send_response(302)
                    self.send_header("Location", f"http://127.0.0.1:{destino.server_port}/destino")
                    self.end_headers()
                else:
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(b"origen")

            def log_message(self, *_):
                pass

        servidor = ThreadingHTTPServer(("127.0.0.1", 0), Origen)
        hilo_origen = threading.Thread(target=servidor.serve_forever, daemon=True)
        hilo_origen.start()
        self.addCleanup(hilo_origen.join, 2)
        self.addCleanup(servidor.server_close)
        self.addCleanup(servidor.shutdown)

        origen = f"http://127.0.0.1:{servidor.server_port}"
        incidencias = []
        with sync_playwright() as playwright:
            navegador = playwright.chromium.launch(executable_path=shutil.which("google-chrome"), headless=True)
            try:
                contexto = navegador.new_context(service_workers="block")
                contexto.route("**/*", lambda ruta: filtrar_red(ruta, origen, incidencias))
                pagina = contexto.new_page()
                self.assertEqual(pagina.goto(origen + "/bien").status, 200)
                with self.assertRaises(Error):
                    pagina.goto(origen + "/salir", timeout=5000)
                self.assertEqual(alcanzadas, [])
                self.assertIn("redireccion_bloqueada", incidencias)
                contexto.close()
            finally:
                navegador.close()


if __name__ == "__main__":
    unittest.main()
