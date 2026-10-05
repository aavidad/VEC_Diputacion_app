import http.client
import json
import tempfile
import threading
import unittest
from http.server import ThreadingHTTPServer
from pathlib import Path

from scripts.servir_preparacion_rrhh import COMUNES, cargar_recursos, handler_para


class PreparacionLocalTest(unittest.TestCase):
    def test_lista_cerrada_y_solo_sintetico(self):
        with tempfile.TemporaryDirectory() as carpeta:
            raiz = Path(carpeta)
            for ruta in (*COMUNES, "textos/es/formacion.json", "portal-empleado/modulos/formacion/index.html"):
                archivo = raiz / ruta
                archivo.parent.mkdir(parents=True, exist_ok=True)
                archivo.write_text("{}")
            (raiz / "textos/idiomas.json").write_text(json.dumps({"idiomas": [{"codigo": "es"}]}))
            escenario = raiz / "portal-empleado/modulos/formacion/escenario.json"
            escenario.write_text(json.dumps({"alcance": "preparacion_sintetica"}))
            (raiz / "privado.txt").write_text("fuera_de_la_lista")
            recursos, entrada = cargar_recursos(raiz, "formacion")
            self.assertIn(entrada, recursos)
            self.assertNotIn("/privado.txt", recursos)
            escenario.write_text(json.dumps({"alcance": "administrativo"}))
            with self.assertRaisesRegex(ValueError, "alcance_invalido"):
                cargar_recursos(raiz, "formacion")

    def test_visor_lista_admision_sirve_textos_de_motivos(self):
        raiz = Path(__file__).resolve().parents[2] / "web/static"
        recursos, entrada = cargar_recursos(raiz, "selectivos-lista-admision-visor")
        self.assertEqual(entrada, "/portal-empleado/modulos/seleccion/preparacion-lista-admision/")
        for ruta in ("/textos/es/motivos-seleccion-admision-ejemplo.json", "/textos/en/motivos-seleccion-admision-ejemplo.json",
                     "/portal-empleado/modulos/seleccion/preparacion-admision/controlador.js",
                     "/textos/es/selectivos-lista-admision-visor.json"):
            self.assertIn(ruta, recursos)
        # Textos de otros módulos no entran en este visor.
        self.assertNotIn("/textos/es/selectivos-admision.json", recursos)
        self.assertFalse(any(r.endswith(".test.mjs") for r in recursos))

    def test_host_rutas_y_escrituras(self):
        recursos = {"/entrada.js": (b"export const ok = true;", "text/javascript")}
        with ThreadingHTTPServer(("127.0.0.1", 0), handler_para(recursos)) as servidor:
            hilo = threading.Thread(target=servidor.serve_forever, daemon=True)
            hilo.start()
            try:
                def pedir(ruta, headers=None, metodo="GET"):
                    conexion = http.client.HTTPConnection("127.0.0.1", servidor.server_port, timeout=2)
                    try:
                        conexion.request(metodo, ruta, headers=headers or {})
                        respuesta = conexion.getresponse()
                        return respuesta.status, dict(respuesta.getheaders()), respuesta.read()
                    finally:
                        conexion.close()

                estado, headers, contenido = pedir("/entrada.js?v=1")
                self.assertEqual(estado, 200)
                self.assertEqual(contenido, recursos["/entrada.js"][0])
                self.assertEqual(headers["Cache-Control"], "no-store")
                self.assertNotIn("Set-Cookie", headers)
                self.assertNotIn("Access-Control-Allow-Origin", headers)
                self.assertEqual(pedir("/entrada.js", {"Host": "externo.invalid"})[0], 403)
                self.assertEqual(pedir("/entrada.js", {"Sec-Fetch-Site": "cross-site"})[0], 403)
                self.assertEqual(pedir("/../privado.txt")[0], 404)
                self.assertEqual(pedir("/entrada.js", metodo="POST")[0], 501)
            finally:
                servidor.shutdown()
                hilo.join(timeout=2)
