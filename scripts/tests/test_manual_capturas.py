"""Pruebas focales del capturador de manuales."""

from __future__ import annotations

import importlib.util
import json
import stat
import tempfile
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from threading import Thread

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
                    "http://localhost:8000/ruta", "http://localhost:8000/?",
                    "http://localhost:8000/#"):
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
        otra_raiz = next((raiz for raiz in capturador._raices_git()
                          if raiz != capturador.RAIZ_REPOSITORIO),
                         capturador.RAIZ_REPOSITORIO)
        self.assertTrue(capturador._dentro_de_git(otra_raiz / "README.md"))
        with self.assertRaises(ValueError):
            capturador._certificados_externos(
                "https://localhost:8443", otra_raiz / "README.md",
                otra_raiz / "README.md", None)

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
        escenario["pantallas"][0]["ruta"] = "/portal-empleado/?vista=rrhh"
        escenario["pantallas"][0]["ruta_final"] = (
            "/portal-empleado/?vista=rrhh#contratacion-temporal")
        self.assertEqual(capturador.validar_escenario(escenario), escenario)
        self.assertTrue(capturador._vista_final_valida(
            "http://127.0.0.1:8000/portal-empleado/?vista=rrhh#contratacion-temporal",
            "http://127.0.0.1:8000", escenario["pantallas"][0]["ruta_final"]))
        self.assertFalse(capturador._vista_final_valida(
            "http://127.0.0.1:8000/portal-empleado/?vista=rrhh#otra",
            "http://127.0.0.1:8000", escenario["pantallas"][0]["ruta_final"]))
        self.assertFalse(capturador._vista_final_valida(
            "http://127.0.0.1:8000/portal-empleado/?vista=otra#contratacion-temporal",
            "http://127.0.0.1:8000", escenario["pantallas"][0]["ruta_final"]))
        for ruta in ("/portal-empleado/?token=secreto", "/portal-empleado/?vista=",
                     "/portal-empleado/?vista=rrhh&vista=otra",
                     "/portal-empleado/?vista=rrhh#A", "/portal-empleado/?",
                     "/portal-empleado/#"):
            copia = json.loads(json.dumps(escenario))
            copia["pantallas"][0]["ruta_final"] = ruta
            with self.subTest(ruta=ruta), self.assertRaises(ValueError):
                capturador.validar_escenario(copia)

    def test_ensayo_captura_dos_tamanos_y_manifiesto_sin_url(self):
        with tempfile.TemporaryDirectory() as temporal:
            salida = Path(temporal)
            manifiesto = capturador.ejecutar_ensayo(salida)
            datos = json.loads(manifiesto.read_text(encoding="utf-8"))
            self.assertEqual(datos["tipo"], "ensayo-sintetico")
            self.assertTrue(datos["pendiente_revision_visual"])
            self.assertEqual(len(datos["capturas"]), 2)
            self.assertNotIn("127.0.0.1", manifiesto.read_text(encoding="utf-8"))
            self.assertEqual(stat.S_IMODE(salida.stat().st_mode), 0o700)
            self.assertEqual(stat.S_IMODE(manifiesto.stat().st_mode), 0o600)
            for captura in datos["capturas"]:
                with Image.open(salida / captura["archivo"]) as imagen:
                    self.assertEqual(imagen.size, (captura["ancho"], captura["alto"]))
                self.assertEqual(stat.S_IMODE((salida / captura["archivo"]).stat().st_mode), 0o600)
                self.assertEqual([marca["numero"] for marca in captura["marcas"]], [1, 2])

    def test_no_sigue_redireccion_a_otro_origen(self):
        visitas = {"origen": 0, "destino": 0}

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                visitas["destino"] += 1
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"destino")

            def log_message(self, *_args):
                pass

        destino = ThreadingHTTPServer(("127.0.0.1", 0), Destino)

        class Origen(BaseHTTPRequestHandler):
            def do_GET(self):
                visitas["origen"] += 1
                self.send_response(302)
                self.send_header("Location", f"http://127.0.0.1:{destino.server_port}/externo")
                self.end_headers()

            def log_message(self, *_args):
                pass

        origen = ThreadingHTTPServer(("127.0.0.1", 0), Origen)
        hilos = [Thread(target=servidor.serve_forever, daemon=True)
                 for servidor in (destino, origen)]
        for hilo in hilos:
            hilo.start()
        escenario = {"pantallas": [{"clave": "redirigida", "ruta": "/", "pasos": [],
                                     "marcas": [{"numero": 1, "selector": "h1",
                                                 "texto": "Título", "tipo": "recuadro"}], "ocultar": []}]}
        try:
            with tempfile.TemporaryDirectory() as temporal:
                with self.assertRaises(Exception):
                    capturador.capturar(f"http://127.0.0.1:{origen.server_port}", escenario,
                                       Path(temporal), confirmar_sinteticos=True)
            self.assertEqual(visitas["origen"], 1)
            self.assertEqual(visitas["destino"], 0)
        finally:
            for servidor in (origen, destino):
                servidor.shutdown()
                servidor.server_close()
            for hilo in hilos:
                hilo.join(timeout=2)

    def test_carga_recurso_versionado_y_espera_estado_final(self):
        visitas = []

        class PaginaVersionada(BaseHTTPRequestHandler):
            def do_GET(self):
                visitas.append(self.path)
                self.send_response(200)
                if self.path.startswith("/app.js"):
                    contenido = (b'document.querySelector("main").innerHTML = "<h1 id=exito>Listo</h1>";'
                                 b'location.hash = "contratacion-temporal";')
                    self.send_header("Content-Type", "text/javascript")
                else:
                    contenido = (b'<main></main><input value="dato sintetico">'
                                 b'<script src="/app.js?v=1"></script>')
                    self.send_header("Content-Type", "text/html")
                self.send_header("Content-Length", str(len(contenido)))
                self.end_headers()
                self.wfile.write(contenido)

            def log_message(self, *_args):
                pass

        servidor = ThreadingHTTPServer(("127.0.0.1", 0), PaginaVersionada)
        hilo = Thread(target=servidor.serve_forever, daemon=True)
        hilo.start()
        escenario = {"pantallas": [{"clave": "versionada", "ruta": "/?vista=rrhh",
                                     "ruta_final": "/?vista=rrhh#contratacion-temporal", "pasos": [],
                                     "exito": "#exito", "marcas": [
                                         {"numero": 1, "selector": "#exito",
                                          "texto": "Estado final", "tipo": "recuadro"}], "ocultar": []}]}
        try:
            with tempfile.TemporaryDirectory() as temporal:
                capturador.capturar(f"http://127.0.0.1:{servidor.server_port}", escenario,
                                   Path(temporal), confirmar_sinteticos=True)
            self.assertEqual(visitas.count("/app.js?v=1"), 2)
            self.assertEqual(visitas.count("/?vista=rrhh"), 2)
        finally:
            servidor.shutdown()
            servidor.server_close()
            hilo.join(timeout=2)

    def test_rechaza_marca_tapada_por_capa(self):
        class PaginaTapada(BaseHTTPRequestHandler):
            def do_GET(self):
                html = (b'<h1 id="titulo">Ensayo</h1>'
                        b'<div style="position:fixed;inset:0;background:white;z-index:9"></div>')
                self.send_response(200)
                self.send_header("Content-Type", "text/html")
                self.end_headers()
                self.wfile.write(html)

            def log_message(self, *_args):
                pass

        servidor = ThreadingHTTPServer(("127.0.0.1", 0), PaginaTapada)
        hilo = Thread(target=servidor.serve_forever, daemon=True)
        hilo.start()
        escenario = {"pantallas": [{"clave": "tapada", "ruta": "/", "pasos": [],
                                     "marcas": [{"numero": 1, "selector": "#titulo",
                                                 "texto": "Título", "tipo": "recuadro"}], "ocultar": []}]}
        try:
            with tempfile.TemporaryDirectory() as temporal:
                with self.assertRaisesRegex(ValueError, "no visible"):
                    capturador.capturar(f"http://127.0.0.1:{servidor.server_port}", escenario,
                                       Path(temporal), confirmar_sinteticos=True)
        finally:
            servidor.shutdown()
            servidor.server_close()
            hilo.join(timeout=2)

    def test_rechaza_marca_sobre_campo_que_se_enmascara(self):
        class PaginaConCampo(BaseHTTPRequestHandler):
            def do_GET(self):
                html = b'<main><input id="campo" value="dato sintetico"></main>'
                self.send_response(200)
                self.send_header("Content-Type", "text/html")
                self.end_headers()
                self.wfile.write(html)

            def log_message(self, *_args):
                pass

        servidor = ThreadingHTTPServer(("127.0.0.1", 0), PaginaConCampo)
        hilo = Thread(target=servidor.serve_forever, daemon=True)
        hilo.start()
        escenario = {"pantallas": [{"clave": "campo", "ruta": "/", "pasos": [],
                                     "marcas": [{"numero": 1, "selector": "#campo",
                                                 "texto": "Campo", "tipo": "recuadro"}], "ocultar": []}]}
        try:
            with tempfile.TemporaryDirectory() as temporal:
                with self.assertRaisesRegex(ValueError, "marca intersecta"):
                    capturador.capturar(f"http://127.0.0.1:{servidor.server_port}", escenario,
                                       Path(temporal), confirmar_sinteticos=True)
        finally:
            servidor.shutdown()
            servidor.server_close()
            hilo.join(timeout=2)

    def test_rechaza_mascara_invisible_con_geometria_sobre_marca(self):
        class PaginaConMascaraInvisible(BaseHTTPRequestHandler):
            def do_GET(self):
                html = (b'<main><h1 id="titulo">Ensayo</h1></main>'
                        b'<div data-private style="visibility:hidden;position:absolute;'
                        b'left:0;top:0;width:600px;height:100px">Oculto</div>')
                self.send_response(200)
                self.send_header("Content-Type", "text/html")
                self.end_headers()
                self.wfile.write(html)

            def log_message(self, *_args):
                pass

        servidor = ThreadingHTTPServer(("127.0.0.1", 0), PaginaConMascaraInvisible)
        hilo = Thread(target=servidor.serve_forever, daemon=True)
        hilo.start()
        escenario = {"pantallas": [{"clave": "invisible", "ruta": "/", "pasos": [],
                                     "marcas": [{"numero": 1, "selector": "#titulo",
                                                 "texto": "Título", "tipo": "recuadro"}], "ocultar": []}]}
        try:
            with tempfile.TemporaryDirectory() as temporal:
                with self.assertRaisesRegex(ValueError, "marca intersecta"):
                    capturador.capturar(f"http://127.0.0.1:{servidor.server_port}", escenario,
                                       Path(temporal), confirmar_sinteticos=True)
        finally:
            servidor.shutdown()
            servidor.server_close()
            hilo.join(timeout=2)


if __name__ == "__main__":
    unittest.main()
