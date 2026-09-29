"""Prueba local: un 302 no debe llegar al segundo puerto loopback."""

import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import recorrer


CHROME = Path("/usr/bin/google-chrome")


class Servidor:
    def __init__(self, handler):
        self.http = ThreadingHTTPServer(("127.0.0.1", 0), handler)
        self.hilo = threading.Thread(target=self.http.serve_forever, daemon=True)

    def __enter__(self):
        self.hilo.start()
        return f"http://127.0.0.1:{self.http.server_port}"

    def __exit__(self, *_args):
        self.http.shutdown()
        self.http.server_close()
        self.hilo.join(timeout=2)


class InterceptorTest(unittest.TestCase):
    def test_302_hacia_otro_puerto_no_contacta_destino(self):
        from playwright.sync_api import Error, sync_playwright

        self.assertTrue(CHROME.is_file())
        llegadas = []

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                llegadas.append(self.path)
                self.send_response(200)
                self.end_headers()

            def log_message(self, *_args):
                pass

        with Servidor(Destino) as destino:
            class Origen(BaseHTTPRequestHandler):
                def do_GET(self):
                    if self.path == "/salto":
                        self.send_response(302)
                        self.send_header("Location", destino + "/prohibido")
                    else:
                        self.send_response(200)
                        self.send_header("Content-Type", "text/html")
                    self.end_headers()
                    if self.path != "/salto":
                        self.wfile.write(b"<!doctype html><title>Local</title>")

                def log_message(self, *_args):
                    pass

            with Servidor(Origen) as origen, sync_playwright() as pw:
                navegador = pw.chromium.launch(executable_path=str(CHROME), headless=True)
                try:
                    contexto = navegador.new_context(service_workers="block")
                    try:
                        contexto.route("**/*", lambda route: recorrer._interceptar_local(route, origen))
                        pagina = contexto.new_page()
                        permitido = pagina.goto(origen + "/permitido", timeout=5000)
                        self.assertEqual(permitido.status, 200)
                        try:
                            bloqueado = pagina.goto(origen + "/salto", timeout=5000)
                        except Error:
                            bloqueado = None
                        self.assertTrue(bloqueado is None or bloqueado.status != 200)
                        self.assertEqual(llegadas, [])
                    finally:
                        contexto.close()
                finally:
                    navegador.close()


if __name__ == "__main__":
    unittest.main()
