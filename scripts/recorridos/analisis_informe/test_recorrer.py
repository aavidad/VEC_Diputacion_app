"""Puertas puras del recorrido; no abren navegador ni tocan servicios."""

import argparse
import hashlib
import json
import tempfile
import unittest
from pathlib import Path

from recorrer import (Corte, NoEjecutado, comprobar_precondiciones, comprobar_recibo,
                      instalar_filtro_red, origen_local, solicitud_permitida)


class PuertasRecorrido(unittest.TestCase):
    def test_solo_origen_https_loopback(self):
        self.assertEqual(origen_local("https://127.0.0.1:8443"), "https://127.0.0.1:8443")
        for origen in ("http://127.0.0.1:8443", "https://vec.cidonia.cloud",
                       "https://127.0.0.1:8443/api", "https://usuario@localhost:8443"):
            with self.subTest(origen=origen), self.assertRaises(NoEjecutado):
                origen_local(origen)

    def test_filtro_rechaza_otro_puerto_y_redireccion_sin_seguirla(self):
        origen = "https://127.0.0.1:8443"
        self.assertTrue(solicitud_permitida(origen + "/portal-empleado/", origen))
        self.assertFalse(solicitud_permitida("https://127.0.0.1:8444/otro", origen))

        class Respuesta:
            status = 302
            url = origen + "/portal-empleado/"
            headers = {"location": "https://127.0.0.1:8444/otro"}

        class Ruta:
            request = type("Peticion", (), {"url": origen + "/portal-empleado/"})()
            llamadas = []

            def fetch(self, **opciones):
                self.llamadas.append(("fetch", opciones))
                return Respuesta()

            def abort(self):
                self.llamadas.append(("abort", None))

            def fulfill(self, **_opciones):
                self.llamadas.append(("fulfill", None))

        class Contexto:
            def route(self, _patron, funcion):
                self.filtrar = funcion

        contexto = Contexto()
        fallos = []
        instalar_filtro_red(contexto, origen, fallos)
        ruta = Ruta()
        contexto.filtrar(ruta)
        self.assertEqual(ruta.llamadas[0], ("fetch", {"max_redirects": 0, "timeout": 20_000}))
        self.assertEqual(ruta.llamadas[1:], [("abort", None)])
        self.assertEqual(len(fallos), 1)

    def test_sin_clon_h3_h5_no_abre_navegador(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            cert = raiz / "cert.pem"
            clave = raiz / "clave.pem"
            cert.write_text("solo prueba", encoding="utf-8")
            clave.write_text("solo prueba", encoding="utf-8")
            binario = raiz / "vec-server"
            binario.write_bytes(b"binario sintetico")
            clon = raiz / "clon.json"
            clon.write_text(json.dumps({"tipo": "clon_local_h3_h5", "sintetico": True,
                                        "origen": "https://localhost:8443", "expediente_ref": "expediente:uno",
                                        "hitos": ["H3", "H4"], "commit_binario": "a" * 40,
                                        "sha256_binario": hashlib.sha256(binario.read_bytes()).hexdigest()}), encoding="utf-8")
            cfg = argparse.Namespace(origen="https://localhost:8443", expediente_ref="expediente:uno",
                                     certificado=cert, clave=clave, certificado_denegado=cert,
                                     clave_denegada=clave, clon=clon, binario=binario,
                                     modo="recuperar", esperado=raiz / "antes.json",
                                     efectos=False, datos=None)
            with self.assertRaisesRegex(NoEjecutado, "H3-H5"):
                comprobar_precondiciones(cfg)
            clon.write_text(json.dumps({"tipo": "clon_local_h3_h5", "sintetico": True,
                                        "origen": "https://localhost:8443", "expediente_ref": "expediente:uno",
                                        "hitos": ["H3", "H4", "H5"], "commit_binario": "a" * 40,
                                        "sha256_binario": "0" * 64}), encoding="utf-8")
            with self.assertRaisesRegex(NoEjecutado, "SHA256"):
                comprobar_precondiciones(cfg)

    def test_recibo_ajeno_o_version_repetida_corta(self):
        recibo = {"expediente_ref": "expediente:uno", "version_resultante": 3,
                  "recibo_ref": "recibo:uno", "confirmada_en": "2026-09-30T00:00:00Z"}
        self.assertEqual(comprobar_recibo(recibo, "expediente:uno", 2, "analisis")["version"], 3)
        for alteracion in ({"expediente_ref": "expediente:dos"}, {"version_resultante": 2},
                           {"recibo_ref": ""}):
            with self.subTest(alteracion=alteracion), self.assertRaises(Corte):
                comprobar_recibo(recibo | alteracion, "expediente:uno", 2, "analisis")


if __name__ == "__main__":
    unittest.main()
