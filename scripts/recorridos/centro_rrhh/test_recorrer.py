"""Pruebas con datos sintéticos; no acceden a VEC ni crean peticiones."""

import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from types import SimpleNamespace

from recorrer import (
    NoEjecutado, interceptar_ruta, origen_local, preflight,
    verificar_entrega, verificar_recibo_centro,
)


class RecorridoCentroTest(unittest.TestCase):
    def test_origen_solo_loopback_https(self):
        self.assertEqual(origen_local("https://127.0.0.1:8443/"), "https://127.0.0.1:8443")
        for valor in ("http://127.0.0.1:8443", "https://cidonia.cloud",
                      "https://127.0.0.1:8443/ruta", "https://127.0.0.1:invalido"):
            with self.subTest(valor=valor), self.assertRaises(NoEjecutado):
                origen_local(valor)

    def test_sin_acreditacion_no_se_abre_navegador(self):
        with self.assertRaisesRegex(NoEjecutado, "acreditacion"):
            preflight(SimpleNamespace(origen="https://127.0.0.1:8443", acreditacion=None))

    def test_recibos_exigen_identidad_version_y_misma_alta(self):
        centro = {"peticion_ref": "peticion:centro:uno", "version": 2, "estado": "ratificada",
                  "actor_ref": "ratificador", "recibo_ref": "recibo:dos",
                  "registrado_en": "2026-09-29T12:00:00Z", "estado_local": "registrado"}
        self.assertEqual(verificar_recibo_centro(centro, "peticion:centro:uno", 2, "ratificada", "ratificador"),
                         ("recibo:dos", "2026-09-29T12:00:00Z"))
        with self.assertRaises(AssertionError):
            verificar_recibo_centro(centro, "peticion:centro:uno", 2, "ratificada", "solicitante")
        entrega = {"peticion": {"referencia": "peticion:centro:uno"}, "estado_entrega": "confirmada",
                   "recibo_alta": {"recibo_ref": "recibo:alta", "confirmada_en": "2026-09-29T12:01:00Z",
                                    "expediente_ref": "expediente:uno", "numero_visible": "2026/CT-1",
                                    "version": 1, "auditoria_ref": "auditoria:uno", "evento_ref": "evento:uno"}}
        original = verificar_entrega(entrega, "peticion:centro:uno")
        self.assertEqual(original, verificar_entrega(entrega, "peticion:centro:uno"))
        entrega["recibo_alta"]["expediente_ref"] = "expediente:dos"
        self.assertNotEqual(original, verificar_entrega(entrega, "peticion:centro:uno"))

    @unittest.skipUnless(Path("/usr/bin/google-chrome").is_file(), "falta Chrome del sistema")
    def test_302_a_otro_puerto_no_llega_al_destino(self):
        from playwright.sync_api import Error, sync_playwright

        contador = {"destino": 0}

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                contador["destino"] += 1
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"destino")

            def log_message(self, *_args):
                pass

        destino = ThreadingHTTPServer(("127.0.0.1", 0), Destino)
        puerto_destino = destino.server_port

        class Origen(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(302)
                self.send_header("Location", f"http://127.0.0.1:{puerto_destino}/contador")
                self.end_headers()

            def log_message(self, *_args):
                pass

        origen = ThreadingHTTPServer(("127.0.0.1", 0), Origen)
        hilos = [threading.Thread(target=s.serve_forever, daemon=True) for s in (destino, origen)]
        for hilo in hilos:
            hilo.start()
        try:
            permitido = f"http://127.0.0.1:{origen.server_port}"
            with sync_playwright() as pw:
                browser = pw.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
                try:
                    contexto = browser.new_context(service_workers="block")
                    contexto.route("**/*", lambda ruta: interceptar_ruta(ruta, permitido))
                    with self.assertRaises(Error):
                        contexto.new_page().goto(permitido + "/salto", wait_until="domcontentloaded")
                    self.assertEqual(contador["destino"], 0)
                finally:
                    browser.close()
        finally:
            for servidor in (origen, destino):
                servidor.shutdown()
                servidor.server_close()
            for hilo in hilos:
                hilo.join(timeout=2)


if __name__ == "__main__":
    unittest.main()
