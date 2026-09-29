"""Prueba focal del cierre previo y del vínculo entre solicitud y recibo."""

from __future__ import annotations

import hashlib
import http.server
import io
import tempfile
import threading
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch
from urllib.error import HTTPError
from urllib.request import HTTPRedirectHandler, Request, build_opener

import recorrer
from recorrer import (FalloRecorrido, NoEjecutado, responder_sin_redireccion,
                      validar_configuracion, validar_recibo, raiz_git_estable)


class RecorridoSinteticoTest(unittest.TestCase):
    def test_raiz_externa_tras_integrar_y_en_worktree(self):
        principal = Path("/home/alberto/Trabajo/VEC_Diputacion_app")
        worktree = principal / ".worktrees" / "codexm-rec-intervencion-20260930"
        self.assertEqual(raiz_git_estable(principal), principal)
        self.assertEqual(raiz_git_estable(worktree), principal)

    def test_contexto_intercepta_primer_pedido_popup_y_cierra_websocket(self):
        class Ruta:
            request = SimpleNamespace(url="https://127.0.0.1:9444/fuera")

            def __init__(self):
                self.abortada = False

            def abort(self):
                self.abortada = True

        class Canal:
            def __init__(self):
                self.codigo = None

            def close(self, *, code):
                self.codigo = code

        class Contexto:
            def __init__(self):
                self.rutas = None
                self.websockets = None

            def route(self, patron, manejador):
                self.rutas = manejador

            def route_web_socket(self, patron, manejador):
                self.websockets = manejador

            def abrir_popup(self):
                pedido_inicial = Ruta()
                self.rutas(pedido_inicial)
                return pedido_inicial

        class Navegador:
            def __init__(self):
                self.creado = Contexto()

            def new_context(self, **opciones):
                return self.creado

        datos = {"origen": "https://127.0.0.1:8443", "rrhh_cert": "/externo/rrhh.crt",
                 "rrhh_key": "/externo/rrhh.key"}
        ctx = recorrer.contexto(Navegador(), datos, "rrhh")
        self.assertTrue(ctx.abrir_popup().abortada)
        canal = Canal()
        ctx.websockets(canal)
        self.assertEqual(canal.codigo, 1008)

    def test_error_tras_post_nunca_declara_no_ejecutado(self):
        for error in (OSError("fallo de escritura"), ValueError("respuesta incorrecta")):
            with self.subTest(tipo=type(error).__name__):
                salida = io.StringIO()

                def fallar(_fase, _datos, _ruta, estado):
                    estado["navegador"] = True
                    estado["post_posible"] = True
                    raise error

                with patch.object(recorrer, "cargar_json", return_value={}), \
                     patch.object(recorrer, "validar_configuracion", return_value={}), \
                     patch.object(recorrer, "ejecutar", side_effect=fallar), \
                     patch("sys.argv", ["recorrer.py", "registrar", "--config", "/externo/config.json",
                                        "--evidencia", "/externo/evidencia.json"]), \
                     patch("sys.stderr", salida):
                    self.assertEqual(recorrer.main(), 1)
                self.assertIn('"estado": "FALLO_CON_EFECTO_POSIBLE"', salida.getvalue())
                self.assertNotIn(str(error), salida.getvalue())

    def test_redireccion_a_otro_puerto_no_llega_al_destino(self):
        impactos = {"origen": 0, "destino": 0}

        class Destino(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                impactos["destino"] += 1
                self.send_response(200)
                self.end_headers()

            def log_message(self, *_):
                pass

        destino = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Destino)

        class Origen(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                impactos["origen"] += 1
                self.send_response(302)
                self.send_header("Location", f"http://127.0.0.1:{destino.server_port}/fuera")
                self.end_headers()

            def log_message(self, *_):
                pass

        origen = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Origen)
        servidores = [threading.Thread(target=s.serve_forever, daemon=True)
                      for s in (origen, destino)]
        for hilo in servidores:
            hilo.start()

        class SinRedireccion(HTTPRedirectHandler):
            def redirect_request(self, *_):
                return None

        class Ruta:
            def __init__(self, url):
                self.request = SimpleNamespace(url=url)
                self.abortada = False
                self.cumplida = False

            def fetch(self, *, max_redirects, timeout):
                self.test.assertEqual(max_redirects, 0)
                self.test.assertEqual(timeout, 30_000)
                try:
                    respuesta = build_opener(SinRedireccion()).open(Request(self.request.url), timeout=1)
                except HTTPError as error:
                    respuesta = error
                return SimpleNamespace(status=respuesta.status, url=respuesta.url)

            def abort(self):
                self.abortada = True

            def fulfill(self, *, response):
                self.cumplida = True

        try:
            url = f"http://127.0.0.1:{origen.server_port}"
            ruta = Ruta(url + "/inicio")
            ruta.test = self
            responder_sin_redireccion(ruta, url)
            self.assertTrue(ruta.abortada)
            self.assertFalse(ruta.cumplida)
            self.assertEqual(impactos, {"origen": 1, "destino": 0})
        finally:
            for servidor in (origen, destino):
                servidor.shutdown()
                servidor.server_close()
            for hilo in servidores:
                hilo.join(timeout=1)

    def test_preparacion_falla_sin_hitos_y_separa_actores(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            chrome = raiz / "chrome"
            binario = raiz / "vec-server"
            cert_int, cert_rrhh = raiz / "int.crt", raiz / "rrhh.crt"
            key_int, key_rrhh = raiz / "int.key", raiz / "rrhh.key"
            for ruta, texto in ((chrome, b"chrome"), (binario, b"binario"),
                                (cert_int, b"cert-int"), (cert_rrhh, b"cert-rrhh"),
                                (key_int, b"key-int"), (key_rrhh, b"key-rrhh")):
                ruta.write_bytes(texto)
            datos = {
                "origen": "https://127.0.0.1:8443", "chrome": str(chrome),
                "binario": str(binario),
                "binario_sha256": hashlib.sha256(b"binario").hexdigest(),
                "hitos_clon": [], "uso_sintetico": True,
                "intervencion_cert": str(cert_int), "intervencion_key": str(key_int),
                "rrhh_cert": str(cert_rrhh), "rrhh_key": str(key_rrhh),
                "favorable": {"expediente_ref": "expediente:sintetico:a", "version_esperada": 5},
                "reparo": {"expediente_ref": "expediente:sintetico:b", "version_esperada": 5},
            }
            salida = raiz / "evidencia.json"
            with self.assertRaisesRegex(NoEjecutado, "H3–H5"):
                validar_configuracion(datos.copy(), salida, "registrar")
            datos["hitos_clon"] = ["H3", "H4", "H5"]
            self.assertEqual(validar_configuracion(datos.copy(), salida, "registrar")["origen"],
                             "https://127.0.0.1:8443")
            datos["rrhh_cert"] = str(cert_int)
            with self.assertRaisesRegex(NoEjecutado, "certificados distintos"):
                validar_configuracion(datos.copy(), salida, "registrar")

    def test_recibo_ligado_y_fecha_inmutable(self):
        solicitud = {"expediente_ref": "expediente:sintetico:a", "version_esperada": 5,
                     "resultado": "favorable"}
        recibo = {"expediente_ref": solicitud["expediente_ref"], "version_resultante": 6,
                  "fase_resultante": "fiscalizacion", "estado_resultante": "en_curso",
                  "resultado": "favorable", "recibo_ref": "recibo:sintetico:a",
                  "registrada_en": "2026-09-29T00:00:00Z",
                  "auditoria_ref": "auditoria:sintetica:a", "evento_ref": "evento:sintetico:a",
                  "actor_ref": "actor:intervencion:sintetico"}
        self.assertEqual(validar_recibo(recibo, solicitud, "favorable")["recibo_ref"],
                         "recibo:sintetico:a")
        with self.assertRaises(FalloRecorrido):
            validar_recibo({**recibo, "version_resultante": 7}, solicitud, "favorable")


if __name__ == "__main__":
    unittest.main()
