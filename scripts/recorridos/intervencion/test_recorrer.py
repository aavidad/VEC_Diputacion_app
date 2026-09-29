"""Prueba focal del cierre previo y del vínculo entre solicitud y recibo."""

from __future__ import annotations

import hashlib
import http.server
import tempfile
import threading
import unittest
from pathlib import Path
from types import SimpleNamespace
from urllib.error import HTTPError
from urllib.request import HTTPRedirectHandler, Request, build_opener

from recorrer import (FalloRecorrido, NoEjecutado, responder_sin_redireccion,
                      validar_configuracion, validar_recibo)


class RecorridoSinteticoTest(unittest.TestCase):
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
