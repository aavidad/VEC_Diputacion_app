import http.client
import json
import tempfile
import threading
import unittest
from http.server import ThreadingHTTPServer
from pathlib import Path

from scripts.servir_revision_notas import (
    COMUNES, MAX_RECURSO, PREFIJO, PROPIOS, cargar_recursos, handler_para,
)


def escribir(raiz, ruta, contenido=b"recurso_publico"):
    archivo = raiz / ruta
    archivo.parent.mkdir(parents=True, exist_ok=True)
    archivo.write_bytes(contenido)
    return archivo


def preparar(raiz, idiomas=("es", "en")):
    for ruta in (*COMUNES, *PROPIOS):
        escribir(raiz, ruta)
    escribir(raiz, "textos/idiomas.json", json.dumps({
        "por_defecto": idiomas[0], "idiomas": [{"codigo": i} for i in idiomas],
    }).encode())
    for idioma in idiomas:
        escribir(raiz, f"textos/{idioma}/seleccion.json", b'{"titulo":"visor"}')


class RevisionNotasLocalTest(unittest.TestCase):
    def test_catalogos_declarados_y_lista_cerrada(self):
        with tempfile.TemporaryDirectory() as carpeta:
            raiz = Path(carpeta)
            preparar(raiz, ("es", "en", "fr"))
            for ruta in (
                "privado.json", f"{PREFIJO}/solicitud.json", f"{PREFIJO}/resultado.json",
                f"{PREFIJO}/testdata/preparacion.json", f"{PREFIJO}/modelo.test.mjs",
                "textos/de/seleccion.json", f"{PREFIJO}/escenario.json",
            ):
                escribir(raiz, ruta)
            recursos, entrada = cargar_recursos(raiz)
            esperados = {f"/{r}" for r in (*COMUNES, *PROPIOS)}
            esperados.update(f"/textos/{i}/seleccion.json" for i in ("es", "en", "fr"))
            esperados.add(entrada)
            self.assertEqual(set(recursos), esperados)
            self.assertEqual(recursos[entrada], recursos[f"/{PREFIJO}/index.html"])
            (raiz / "textos/fr/seleccion.json").unlink()
            with self.assertRaises(FileNotFoundError):
                cargar_recursos(raiz)

    def test_rechaza_escape_symlink_directorio_y_recurso_excesivo(self):
        with tempfile.TemporaryDirectory() as carpeta:
            raiz = Path(carpeta) / "web"
            preparar(raiz)
            archivo = raiz / f"{PREFIJO}/vista.js"
            archivo.unlink()
            externo = escribir(Path(carpeta), "privado.js")
            archivo.symlink_to(externo)
            with self.assertRaisesRegex(ValueError, "recurso_invalido"):
                cargar_recursos(raiz)
            archivo.unlink()
            archivo.mkdir()
            with self.assertRaisesRegex(ValueError, "recurso_invalido"):
                cargar_recursos(raiz)
            archivo.rmdir()
            archivo.write_bytes(b"x" * (MAX_RECURSO + 1))
            with self.assertRaisesRegex(ValueError, "recurso_demasiado_grande"):
                cargar_recursos(raiz)
            archivo.write_bytes(b"")
            with self.assertRaisesRegex(ValueError, "recurso_invalido"):
                cargar_recursos(raiz)

    def test_recursos_reales_incluyen_importaciones_y_estilos(self):
        raiz = Path(__file__).resolve().parents[2] / "web/static"
        recursos, entrada = cargar_recursos(raiz)
        self.assertEqual(entrada, f"/{PREFIJO}/")
        self.assertIn(b"preparacion-notas.css?v=", recursos[entrada][0])
        self.assertIn("/textos/idiomas.json", recursos)
        for ruta in PROPIOS:
            self.assertTrue(recursos[f"/{ruta}"][0])

    def test_http_hereda_host_csp_no_store_y_solo_lectura(self):
        with tempfile.TemporaryDirectory() as carpeta:
            raiz = Path(carpeta)
            preparar(raiz)
            recursos, entrada = cargar_recursos(raiz)
            with ThreadingHTTPServer(("127.0.0.1", 0), handler_para(recursos)) as servidor:
                hilo = threading.Thread(target=servidor.serve_forever, daemon=True)
                hilo.start()
                try:
                    def pedir(ruta, metodo="GET", headers=None):
                        conexion = http.client.HTTPConnection("127.0.0.1", servidor.server_port, timeout=2)
                        try:
                            conexion.request(metodo, ruta, headers=headers or {})
                            respuesta = conexion.getresponse()
                            return respuesta.status, dict(respuesta.getheaders()), respuesta.read()
                        finally:
                            conexion.close()

                    estado, cabeceras, contenido = pedir(entrada + "?lang=en")
                    self.assertEqual(estado, 200)
                    self.assertEqual(contenido, recursos[entrada][0])
                    self.assertEqual(cabeceras["Cache-Control"], "no-store")
                    self.assertIn("default-src 'none'", cabeceras["Content-Security-Policy"])
                    self.assertIn("form-action 'none'", cabeceras["Content-Security-Policy"])
                    self.assertEqual(cabeceras["Referrer-Policy"], "no-referrer")
                    self.assertNotIn("Set-Cookie", cabeceras)
                    self.assertNotIn("Access-Control-Allow-Origin", cabeceras)
                    self.assertEqual(pedir(entrada, "HEAD")[2], b"")
                    self.assertEqual(pedir(entrada, headers={"Host": "ajeno.invalid"})[0], 403)
                    self.assertEqual(pedir(entrada, headers={"Sec-Fetch-Site": "cross-site"})[0], 403)
                    self.assertEqual(pedir(f"/{PREFIJO}/resultado.json")[0], 404)
                    self.assertEqual(pedir("/../privado.json")[0], 404)
                    self.assertEqual(pedir(entrada, "POST")[0], 501)
                finally:
                    servidor.shutdown()
                    hilo.join(timeout=2)


if __name__ == "__main__":
    unittest.main()
