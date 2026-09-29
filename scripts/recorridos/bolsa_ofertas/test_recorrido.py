"""Puerta sintética del guion: nunca toca servicios ni inicia Chrome."""

import hashlib
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from recorrido import NoEjecutado, instalar_filtro_red, validar_configuracion, validar_origen


class PuertaRecorrido(unittest.TestCase):
    def test_rechaza_destinos_remotos_y_credenciales_en_url(self):
        for origen in ("https://vec.cidonia.cloud:443", "http://127.0.0.1:8443",
                       "https://usuario:clave@127.0.0.1:8443", "https://127.0.0.1:8443/api"):
            with self.subTest(origen=origen), self.assertRaises(NoEjecutado):
                validar_origen(origen)

    def test_sin_clon_h3_h5_o_binario_exacto_no_se_abre_navegador(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            binario = raiz / "vec-server"
            binario.write_bytes(b"binario sintetico")
            cert = raiz / "cert.pem"
            clave = raiz / "key.pem"
            cert.write_text("sintetico")
            clave.write_text("sintetico")
            clave.chmod(0o600)
            datos = {
                "origen": "https://127.0.0.1:8443", "clon": "aislado_h3_h4_h5",
                "hitos": ["H3", "H4", "H5"], "binario": str(binario),
                "binario_sha256": hashlib.sha256(binario.read_bytes()).hexdigest(),
                "rrhh": {"certificado": str(cert), "clave": str(clave)},
                "candidato": {"certificado": str(cert), "clave": str(clave)},
                "bolsa_ref": "bolsa:sintetica", "bolsa_sintetica_reservada": True,
                "oferta": {"categoria": "Auxiliar", "centro": "Centro sintético",
                           "fecha_inicio": "2026-10-15", "descripcion": "Ensayo"},
            }
            sin_clon = {**datos, "hitos": ["H3", "H4"]}
            with self.assertRaisesRegex(NoEjecutado, "clon aislado"):
                validar_configuracion(sin_clon, "alta")
            sin_binario = {**datos, "binario_sha256": "0" * 64}
            with self.assertRaisesRegex(NoEjecutado, "binario VEC exacto"):
                validar_configuracion(sin_binario, "alta")
            self.assertEqual(validar_configuracion(datos, "alta")["bolsa_ref"], "bolsa:sintetica")
            with self.assertRaisesRegex(NoEjecutado, "reinicio"):
                validar_configuracion(datos, "recuperar")

    def test_redirect_a_otro_puerto_no_contacta_destino(self):
        contactos = {"origen": 0, "destino": 0}

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                contactos["destino"] += 1
                self.send_response(200)
                self.end_headers()

            def log_message(self, *_):
                pass

        destino = ThreadingHTTPServer(("127.0.0.1", 0), Destino)

        class Origen(BaseHTTPRequestHandler):
            def do_GET(self):
                contactos["origen"] += 1
                self.send_response(302)
                self.send_header("Location", f"http://127.0.0.1:{destino.server_port}/fuera")
                self.end_headers()

            def log_message(self, *_):
                pass

        origen = ThreadingHTTPServer(("127.0.0.1", 0), Origen)
        hilos = [threading.Thread(target=servidor.serve_forever, daemon=True)
                 for servidor in (origen, destino)]
        for hilo in hilos:
            hilo.start()
        try:
            from playwright.sync_api import Error, sync_playwright

            with sync_playwright() as playwright:
                navegador = playwright.chromium.launch(headless=True, executable_path="/usr/bin/google-chrome",
                    args=["--disable-background-networking", "--disable-extensions", "--disable-sync"])
                try:
                    contexto = navegador.new_context(service_workers="block")
                    fallos = []
                    base = f"http://127.0.0.1:{origen.server_port}"
                    instalar_filtro_red(contexto, base, fallos)
                    try:
                        contexto.new_page().goto(base + "/", wait_until="domcontentloaded")
                    except Error:
                        pass
                    self.assertEqual(contactos, {"origen": 1, "destino": 0})
                    self.assertIn("redirección rechazada sin seguirla", fallos)
                finally:
                    navegador.close()
        finally:
            origen.shutdown()
            destino.shutdown()
            origen.server_close()
            destino.server_close()


if __name__ == "__main__":
    unittest.main()
