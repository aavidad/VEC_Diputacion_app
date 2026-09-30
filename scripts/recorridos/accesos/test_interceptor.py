"""Prueba local: un 302 no debe llegar al segundo puerto loopback."""

import asyncio
import json
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest.mock import patch

import recorrer


CHROME = Path("/usr/bin/google-chrome")
PLAZO_CASO = 15
PLAZO_RECOGIDA = 2


def _inicio_proceso(pid):
    # El PID del hijo queda reservado hasta wait(): no puede reutilizarse.
    return Path(f"/proc/{pid}/stat").read_text().rsplit(")", 1)[1].split()[19]


def _ejecutar_aislado(caso, plazo=PLAZO_CASO, observar=None):
    """Acota el caso entero, incluidos cierres async y shutdown síncrono."""
    import playwright

    bwrap = shutil.which("bwrap")
    if bwrap is None:
        raise RuntimeError("Falta bwrap para aislar y recoger la prueba local.")
    carpeta = Path(__file__).resolve().parent
    paquetes = Path(playwright.__file__).resolve().parent.parent
    with tempfile.TemporaryDirectory(prefix="vec-interceptor-") as temporal:
        scratch = Path(temporal)
        marca = scratch / "recursos.json"
        comando = [bwrap, "--unshare-net", "--unshare-pid", "--die-with-parent"]
        # Solo código y herramientas de lectura; no se monta el HOME del operador.
        for ruta in ("/usr", "/bin", "/lib", "/lib64", "/opt/google/chrome",
                     "/etc/alternatives", "/etc/fonts", "/etc/passwd", "/etc/group",
                     str(paquetes), str(carpeta)):
            if Path(ruta).exists():
                comando += ["--ro-bind", ruta, ruta]
        comando += ["--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp",
                    "--bind", str(scratch), str(scratch), "--chdir", str(carpeta),
                    sys.executable, str(Path(__file__).resolve()), "--caso-aislado", caso, str(marca)]
        entorno = {"PATH": "/usr/bin:/bin", "LANG": "C.UTF-8", "HOME": str(scratch),
                   "TMPDIR": str(scratch), "XDG_CONFIG_HOME": str(scratch),
                   "XDG_CACHE_HOME": str(scratch), "PYTHONPATH": str(paquetes),
                   "PYTHONDONTWRITEBYTECODE": "1"}
        hijo = subprocess.Popen(comando, env=entorno, start_new_session=True,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        inicio = None
        recursos = None
        try:
            inicio = _inicio_proceso(hijo.pid)
            salida, errores = hijo.communicate(timeout=plazo)
            if hijo.returncode:
                raise AssertionError(errores.decode(errors="replace") or salida.decode(errors="replace"))
        except subprocess.TimeoutExpired as exc:
            if marca.is_file():
                recursos = json.loads(marca.read_text())
                recursos["red_visible_antes"] = _namespace_sigue_vivo(recursos["red"])
            raise TimeoutError(f"La prueba {caso} superó el límite global de {plazo} s, incluido el cierre.") from exc
        finally:
            # El hijo no se ha recogido si returncode sigue a None. Validamos
            # además generación, sesión y grupo antes de señalar el grupo propio.
            if hijo.returncode is None:
                acreditado = False
                fallo_identidad = None
                if inicio is not None:
                    try:
                        acreditado = (_inicio_proceso(hijo.pid) == inicio
                                      and os.getpgid(hijo.pid) == hijo.pid
                                      and os.getsid(hijo.pid) == hijo.pid)
                    except OSError as exc:
                        fallo_identidad = exc
                if acreditado:
                    os.killpg(hijo.pid, signal.SIGKILL)
                else:
                    # Popen conserva su hijo sin recoger: solo se termina el
                    # wrapper propio. bwrap recoge sus namespaces al morir.
                    hijo.kill()
                try:
                    hijo.communicate(timeout=PLAZO_RECOGIDA)
                except subprocess.TimeoutExpired as exc:
                    raise RuntimeError("No se recogió la prueba después de terminar su wrapper propio.") from exc
                if inicio is not None and not acreditado:
                    raise RuntimeError("No se ha podido acreditar el grupo; se recogió solo el wrapper propio.") from fallo_identidad
            if observar is not None and marca.is_file():
                observar(recursos if recursos is not None else json.loads(marca.read_text()))


def _namespace_sigue_vivo(referencia):
    for proceso in Path("/proc").iterdir():
        if proceso.name.isdigit():
            try:
                if os.readlink(proceso / "ns/net") == referencia:
                    return True
            except (FileNotFoundError, PermissionError, ProcessLookupError):
                pass
    return False


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
        _ejecutar_aislado("302")

    def test_fallo_proc_tras_popen_recoge_solo_wrapper_propio(self):
        creados = []
        crear = subprocess.Popen
        def registrar(*args, **kwargs):
            hijo = crear(*args, **kwargs)
            creados.append(hijo)
            return hijo
        with patch.object(subprocess, "Popen", side_effect=registrar), \
                patch.object(sys.modules[__name__], "_inicio_proceso", side_effect=OSError("/proc simulado")), \
                patch.object(os, "killpg") as grupo:
            with self.assertRaisesRegex(OSError, "/proc simulado"):
                _ejecutar_aislado("302")
        grupo.assert_not_called()
        self.assertEqual(len(creados), 1)
        self.assertIsNotNone(creados[0].returncode, "El wrapper debe quedar recogido.")
        self.assertFalse(Path(f"/proc/{creados[0].pid}").exists())

    def test_cierre_bloqueado_vence_y_recoge_chrome_servidor_y_socket(self):
        recursos = []
        inicio = time.monotonic()
        with self.assertRaisesRegex(TimeoutError, "límite global de 3 s, incluido el cierre"):
            _ejecutar_aislado("cierre_bloqueado", plazo=3, observar=recursos.append)
        self.assertLess(time.monotonic() - inicio, 3 + PLAZO_RECOGIDA + 1)
        self.assertEqual(len(recursos), 1, "El caso debe alcanzar el cierre de Chrome.")
        self.assertTrue(recursos[0]["chrome_abierto"])
        self.assertTrue(recursos[0]["servidor_abierto"])
        self.assertTrue(recursos[0]["red_visible_antes"], "La comprobación debe observar el namespace vivo.")
        # Desaparece el namespace entero: todos sus procesos y sockets son propios.
        self.assertFalse(_namespace_sigue_vivo(recursos[0]["red"]))

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
        _ejecutar_aislado("websocket")

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


async def _probar_cierre_bloqueado(marca):
    from playwright.async_api import async_playwright

    class Origen(BaseHTTPRequestHandler):
        def do_GET(self):
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"<!doctype html><title>Local</title>")

        def log_message(self, *_args):
            pass

    with Servidor(Origen) as origen:
        async with async_playwright() as pw:
            navegador = await pw.chromium.launch(executable_path=str(CHROME), headless=True)
            contexto = await navegador.new_context()
            pagina = await contexto.new_page()
            assert (await pagina.goto(origen, timeout=5000)).status == 200
            await contexto.close()
            # Un cierre síncrono bloquea también la cancelación de asyncio.
            async def cierre_bloqueado():
                marca.write_text(json.dumps({"chrome_abierto": navegador.is_connected(), "servidor_abierto": True,
                                              "red": os.readlink("/proc/self/ns/net")}))
                threading.Event().wait()
            navegador.close = cierre_bloqueado
            await navegador.close()


if __name__ == "__main__":
    if len(sys.argv) == 4 and sys.argv[1] == "--caso-aislado":
        caso = sys.argv[2]
        if caso == "cierre_bloqueado":
            asyncio.run(_probar_cierre_bloqueado(Path(sys.argv[3])))
        elif caso == "302":
            asyncio.run(InterceptorTest()._probar_302())
        elif caso == "websocket":
            asyncio.run(InterceptorTest()._probar_websocket())
        else:
            raise ValueError("Caso local desconocido.")
    else:
        unittest.main()
