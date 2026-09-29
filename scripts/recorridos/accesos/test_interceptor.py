"""Prueba local: un 302 no debe llegar al segundo puerto loopback."""

import asyncio
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import recorrer


CHROME = Path("/usr/bin/google-chrome")


class Servidor:
    def __init__(self, handler, clase=ThreadingHTTPServer):
        self.http = clase(("127.0.0.1", 0), handler)
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
        asyncio.run(self._probar_302())

    async def _probar_302(self):
        from playwright.async_api import Error, async_playwright

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

            with Servidor(Origen) as origen:
                async with async_playwright() as pw:
                    navegador = await pw.chromium.launch(executable_path=str(CHROME), headless=True)
                    try:
                        contexto = await navegador.new_context(service_workers="block")
                        try:
                            await contexto.route("**/*", lambda route: recorrer._interceptar_local(route, origen))
                            pagina = await contexto.new_page()
                            permitido = await pagina.goto(origen + "/permitido", timeout=5000)
                            self.assertEqual(permitido.status, 200)
                            try:
                                bloqueado = await pagina.goto(origen + "/salto", timeout=5000)
                            except Error:
                                bloqueado = None
                            self.assertTrue(bloqueado is None or bloqueado.status != 200)
                            self.assertEqual(llegadas, [])
                        finally:
                            await contexto.close()
                    finally:
                        await navegador.close()

    def test_websocket_hacia_otro_puerto_no_abre_conexion(self):
        asyncio.run(self._probar_websocket())

    async def _probar_websocket(self):
        from playwright.async_api import async_playwright

        conexiones = []
        interceptados = []

        class DestinoHTTP(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200)
                self.end_headers()

            def log_message(self, *_args):
                pass

        class DestinoContado(ThreadingHTTPServer):
            def get_request(self):
                conexion = super().get_request()
                conexiones.append(1)
                return conexion

        class Origen(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(200)
                self.send_header("Content-Type", "text/html")
                self.end_headers()
                self.wfile.write(b"<!doctype html><title>Local</title>")

            def log_message(self, *_args):
                pass

        with Servidor(DestinoHTTP, DestinoContado) as destino, Servidor(Origen) as origen:
            async with async_playwright() as pw:
                navegador = await pw.chromium.launch(executable_path=str(CHROME), headless=True)
                try:
                    contexto = await navegador.new_context(service_workers="block")
                    try:
                        await contexto.route("**/*", lambda route: recorrer._interceptar_local(route, origen))

                        async def cerrar(route):
                            interceptados.append(route.url)
                            await recorrer._cerrar_websocket(route)

                        await contexto.route_web_socket("**/*", cerrar)
                        pagina = await contexto.new_page()
                        self.assertEqual((await pagina.goto(origen + "/", timeout=5000)).status, 200)
                        url_ws = destino.replace("http://", "ws://") + "/socket"
                        await pagina.evaluate("url => { window.socketPrueba = new WebSocket(url); }", url_ws)
                        await pagina.wait_for_function("() => window.socketPrueba.readyState === WebSocket.CLOSED", timeout=5000)
                        self.assertEqual(interceptados, [url_ws])
                        self.assertEqual(conexiones, [])
                    finally:
                        await contexto.close()
                finally:
                    await navegador.close()


if __name__ == "__main__":
    unittest.main()
