"""Prueba focal sintética del cierre previo y de la comparación de recibos."""

import hashlib
import json
import shutil
import stat
import subprocess
import sys
import tempfile
import threading
import unittest
from unittest.mock import MagicMock, patch
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from recorrido import (FECHA_ETAPA, FalloRecorrido, NoEjecutado, comprobar_entrada,
                       comprobar_recibo, estado_http_esperado, fichero_externo,
                       mismo_origen, servir_solo_origen)


class RecorridoPrueba(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporal.cleanup)
        self.raiz = Path(self.temporal.name) / "repositorio"
        self.raiz.mkdir()
        externo = Path(self.temporal.name) / "material"
        externo.mkdir()
        self.binario = externo / "vec-server"
        self.binario.write_bytes(b"binario sintetico sin servidor")
        self.binario.chmod(self.binario.stat().st_mode | stat.S_IXUSR)
        huella = hashlib.sha256(self.binario.read_bytes()).hexdigest()
        rrhh = externo / "rrhh.crt"
        candidato = externo / "candidato.crt"
        clave_rrhh = externo / "rrhh.key"
        clave_candidato = externo / "candidato.key"
        for ruta, contenido in ((rrhh, b"rrhh"), (candidato, b"candidato"),
                                (clave_rrhh, b"clave rrhh"), (clave_candidato, b"clave candidato")):
            ruta.write_bytes(contenido)
        manifiesto = externo / "clon.json"
        manifiesto.write_text(json.dumps({"origen": "https://127.0.0.1:18443", "clon": "H3-H5",
                                         "hitos_instalados": ["H3", "H4", "H5"], "datos": "sinteticos",
                                         "listo": True, "binario_sha256": huella}), encoding="utf-8")
        self.escenario = {"origen": "https://127.0.0.1:18443", "clon": "H3-H5",
                          "manifiesto_clon": str(manifiesto), "binario": str(self.binario),
                          "binario_sha256": huella, "bolsa_ref": "bolsa:sintetica",
                          "identidades": {
                              "rrhh": {"certificado": str(rrhh), "clave": str(clave_rrhh),
                                       "ruta": "/portal-empleado/"},
                              "candidato": {"certificado": str(candidato), "clave": str(clave_candidato),
                                            "ruta": "/area-personal/?vista=llamamientos"}},
                          "respuesta": "renuncia",
                          "etapas": [{"nombre": nombre, "campos_recibo": {
                              "recibo" if nombre == "candidato_respuesta" else "recibo_ref": "recibo:sintetico",
                              FECHA_ETAPA[nombre]: "2026-09-29T00:00:00.000000Z"}}
                                     for nombre in ("seleccion", "comunicacion", "candidato_respuesta",
                                                    "declaracion_rrhh", "resolucion_rrhh", "siguiente")],
                          }

    def test_escenario_valido_solo_prepara(self):
        self.assertIs(comprobar_entrada(self.escenario, self.raiz), self.escenario)

    def test_observacion_no_exige_recibos_futuros_y_conserva_cierre_previo(self):
        escenario = {k: v for k, v in self.escenario.items()
                     if k not in {"etapas", "bolsa_ref", "respuesta"}}
        self.assertIs(comprobar_entrada(escenario, self.raiz, observar=True), escenario)
        with self.assertRaises(NoEjecutado):
            comprobar_entrada(escenario, self.raiz)
        with self.assertRaises(NoEjecutado):
            comprobar_entrada({**escenario, "binario_sha256": "0" * 64}, self.raiz,
                              observar=True)

    def test_evidencias_no_sobrescriben_carpeta_ni_entran_en_git(self):
        from observar import preparar_destino
        existente = Path(self.temporal.name) / "material"
        with self.assertRaises(FileExistsError):
            preparar_destino(existente)
        (self.raiz / ".git").mkdir()
        with self.assertRaises(NoEjecutado):
            preparar_destino(self.raiz / "capturas")

    def test_cli_observar_sin_destino_sigue_siendo_no_ejecutado(self):
        escenario = Path(self.temporal.name) / "material" / "apertura.json"
        escenario.write_text(json.dumps(self.escenario), encoding="utf-8")
        resultado = subprocess.run(
            [sys.executable, str(Path(__file__).with_name("recorrido.py")),
             "--escenario", str(escenario), "--observar"],
            capture_output=True, text=True, timeout=5, check=False)
        self.assertEqual(resultado.returncode, 3)
        self.assertIn("NO EJECUTADO", resultado.stderr)

    def test_timeout_de_apertura_conserva_capturas_en_ambas_vistas(self):
        from observar import observar_apertura
        from playwright.sync_api import TimeoutError
        destino = Path(self.temporal.name) / "capturas"
        playwright = MagicMock()
        navegador = playwright.chromium.launch.return_value
        contexto = navegador.new_context.return_value
        contexto.cookies.return_value = []
        pagina = contexto.new_page.return_value
        pagina.goto.side_effect = TimeoutError("red sin reposo")
        pagina.evaluate.return_value = {"ancho": 390, "contenido": 390, "local": 0,
                                       "sesion": 0, "indexeddb": 0}
        pagina.screenshot.side_effect = lambda **kwargs: Path(kwargs["path"]).write_bytes(b"png sintetico")
        with patch("playwright.sync_api.sync_playwright") as iniciar:
            iniciar.return_value.__enter__.return_value = playwright
            self.assertEqual(observar_apertura(self.escenario, destino), 1)
        informe = json.loads((destino / "resultado.json").read_text())
        self.assertEqual(informe["estado"], "CORTE_APLICACION")
        self.assertEqual(informe["operaciones_completadas"], [])
        self.assertEqual(informe["flujo_posterior"], "NO_EJECUTADO")
        for actor in ("rrhh", "candidato"):
            self.assertEqual(informe["perfiles"][actor]["fallo"], "TimeoutError")
            self.assertEqual(len(informe["perfiles"][actor]["vistas"]), 2)
            for ancho in (1440, 390):
                self.assertTrue((destino / f"llamamiento-{actor}-{ancho}.png").is_file())

    def test_faltan_clon_o_binario_cierra(self):
        for campo, valor in (("clon", "HITO1"), ("binario_sha256", "0" * 64)):
            with self.subTest(campo=campo):
                escenario = {**self.escenario, campo: valor}
                with self.assertRaises(NoEjecutado):
                    comprobar_entrada(escenario, self.raiz)

    def test_identidad_separada_y_secuencia_obligatorias(self):
        identidades = dict(self.escenario["identidades"])
        identidades["candidato"] = {**identidades["candidato"],
                                     "certificado": identidades["rrhh"]["certificado"]}
        with self.assertRaises(NoEjecutado):
            comprobar_entrada({**self.escenario, "identidades": identidades}, self.raiz)
        with self.assertRaises(NoEjecutado):
            comprobar_entrada({**self.escenario, "etapas": self.escenario["etapas"][:-1]}, self.raiz)

    def test_recibo_inmutable(self):
        comprobar_recibo({"data": {"recibo_ref": "recibo:1", "version_resultante": 3}},
                        {"recibo_ref": "recibo:1", "version_resultante": 3})
        with self.assertRaises(FalloRecorrido):
            comprobar_recibo({"data": {"recibo_ref": "recibo:2"}}, {"recibo_ref": "recibo:1"})

    def test_seleccion_inicial_y_replay_responden_200(self):
        self.assertEqual(estado_http_esperado("seleccion", False), 200)
        self.assertEqual(estado_http_esperado("seleccion", True), 200)
        self.assertEqual(estado_http_esperado("comunicacion", False), 201)
        self.assertEqual(estado_http_esperado("comunicacion", True), 200)

    def test_cli_sin_modo_no_declara_exito(self):
        escenario = Path(self.temporal.name) / "material" / "escenario.json"
        escenario.write_text(json.dumps(self.escenario), encoding="utf-8")
        comando = [sys.executable, str(Path(__file__).with_name("recorrido.py")),
                   "--escenario", str(escenario)]
        sin_modo = subprocess.run(comando, capture_output=True, text=True, check=False, timeout=5)
        self.assertEqual(sin_modo.returncode, 3)
        self.assertIn("NO EJECUTADO", sin_modo.stderr)
        comprobar = subprocess.run(comando + ["--comprobar"], capture_output=True,
                                  text=True, check=False, timeout=5)
        self.assertEqual(comprobar.returncode, 0)
        self.assertIn("PREPARADO", comprobar.stdout)

    def test_material_en_raiz_compartida_no_se_acepta(self):
        worktree = self.raiz / ".worktrees" / "otro"
        worktree.mkdir(parents=True)
        material = self.raiz / "clave.key"
        material.write_bytes(b"sintetico")
        with self.assertRaises(NoEjecutado):
            fichero_externo(str(material), worktree)

    def test_url_de_otro_puerto_se_denegaria(self):
        self.assertTrue(mismo_origen("https://127.0.0.1:18443/ruta", "https://127.0.0.1:18443"))
        self.assertFalse(mismo_origen("https://127.0.0.1:18444/ruta", "https://127.0.0.1:18443"))
        self.assertFalse(mismo_origen("http://127.0.0.1:18443/ruta", "https://127.0.0.1:18443"))

    def test_chrome_no_contacta_destino_de_302_en_otro_puerto(self):
        if not shutil.which("google-chrome"):
            self.skipTest("Chrome del sistema no disponible")
        try:
            from playwright.sync_api import Error, sync_playwright
        except ImportError:
            self.skipTest("Playwright Python no disponible")

        contactos = []
        peticiones_origen = []

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                contactos.append(self.path)
                self.send_response(200)
                self.end_headers()

            def log_message(self, *_):
                pass

        destino = ThreadingHTTPServer(("127.0.0.1", 0), Destino)

        class Origen(BaseHTTPRequestHandler):
            def do_GET(self):
                peticiones_origen.append(self.path)
                self.send_response(302)
                self.send_header("Location", f"http://127.0.0.1:{destino.server_port}/fuera")
                self.end_headers()

            def log_message(self, *_):
                pass

        origen = ThreadingHTTPServer(("127.0.0.1", 0), Origen)
        hilos = [threading.Thread(target=servidor.serve_forever, daemon=True)
                 for servidor in (origen, destino)]
        for hilo in hilos:
            hilo.start()
        try:
            with sync_playwright() as playwright:
                navegador = playwright.chromium.launch(headless=True,
                                                       executable_path=shutil.which("google-chrome"))
                try:
                    contexto = navegador.new_context()
                    url_origen = f"http://127.0.0.1:{origen.server_port}"
                    contexto.route("**/*", lambda route: servir_solo_origen(route, url_origen))
                    with self.assertRaises(Error):
                        contexto.new_page().goto(url_origen + "/salto", timeout=5000)
                    self.assertIn("/salto", peticiones_origen, "el origen no sirvió el 302")
                    self.assertEqual(contactos, [], "Chrome alcanzó el puerto de destino")
                finally:
                    navegador.close()
        finally:
            origen.shutdown()
            destino.shutdown()
            origen.server_close()
            destino.server_close()
            for hilo in hilos:
                hilo.join(timeout=1)


if __name__ == "__main__":
    unittest.main()
