"""Puerta local de seguridad: no requiere navegador ni servidor."""

from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest.mock import patch

from recorrer import NoEjecutado, RAIZ_REPO, bloquear_websocket, dentro_git, ejecutar, filtrar_red, guardar_captura, preparar, preparar_evidencias, raices_git, validar_capturas


class Precondiciones(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.raiz = Path(self.temp.name)
        self.binario = self.raiz / "vec-server"
        self.binario.write_bytes(b"binario sintetico de prueba")
        self.certificado = self.raiz / "cert.pem"
        self.clave = self.raiz / "clave.pem"
        self.certificado.write_bytes(b"material ficticio")
        self.clave.write_bytes(b"material ficticio")
        self.acta = self.raiz / "acta.json"
        self.origen = "https://127.0.0.1:18443"
        self.datos = {
            "origen": self.origen,
            "clon_sintetico": True,
            "clon_ref": "clon:prueba",
            "hitos_verificados": ["H3", "H4", "H5"],
            "binario": str(self.binario),
            "binario_sha256": hashlib.sha256(self.binario.read_bytes()).hexdigest(),
        }

    def guardar(self):
        self.acta.write_text(json.dumps(self.datos), encoding="utf-8")

    def test_sin_acta_no_ejecuta(self):
        with self.assertRaisesRegex(NoEjecutado, "acta"):
            preparar(self.origen, self.acta, self.certificado, self.clave)

    def test_hito_o_huella_incompleta_no_ejecuta(self):
        self.datos["hitos_verificados"] = ["H3", "H4"]
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "H3, H4 y H5"):
            preparar(self.origen, self.acta, self.certificado, self.clave)
        self.datos["hitos_verificados"].append("H5")
        self.datos["binario_sha256"] = "0" * 64
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "binario"):
            preparar(self.origen, self.acta, self.certificado, self.clave)

    def test_acta_malformada_falla_cerrada(self):
        self.datos["hitos_verificados"] = 5
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "H3, H4 y H5"):
            preparar(self.origen, self.acta, self.certificado, self.clave)
        self.datos["hitos_verificados"] = ["H3", "H4", "H5"]
        self.datos["binario"] = None
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "binario"):
            preparar(self.origen, self.acta, self.certificado, self.clave)
        with self.assertRaisesRegex(NoEjecutado, "puerto"):
            preparar("https://127.0.0.1:invalido", self.acta, self.certificado, self.clave)

    def test_origen_externo_o_material_ausente_no_ejecuta(self):
        self.guardar()
        with self.assertRaisesRegex(NoEjecutado, "origen"):
            preparar("https://cidonia.example", self.acta, self.certificado, self.clave)
        self.clave.unlink()
        with self.assertRaisesRegex(NoEjecutado, "clave"):
            preparar(self.origen, self.acta, self.certificado, self.clave)

    def test_acta_completa_permita_pasar_solo_la_puerta(self):
        self.guardar()
        with patch("recorrer.shutil.which", return_value="/usr/bin/google-chrome"):
            origen, chrome = preparar(self.origen, self.acta, self.certificado, self.clave)
        self.assertEqual(origen, self.origen)
        self.assertEqual(chrome, Path("/usr/bin/google-chrome"))

    def test_material_privado_fuera_de_todas_las_raices_git(self):
        raices = raices_git()
        self.assertIn(RAIZ_REPO, raices)
        comun = subprocess.run(["git", "-C", str(RAIZ_REPO), "rev-parse", "--git-common-dir"],
                               check=True, capture_output=True, text=True).stdout.strip()
        compartida = Path(comun).resolve().parent
        self.assertIn(compartida, raices)
        self.assertTrue(dentro_git(compartida / "acta.json", raices))
        self.assertTrue(dentro_git(RAIZ_REPO / "clave.pem", raices))
        self.assertTrue(dentro_git(Path("/dev/shm/vec-otra-rama/clave.pem"),
                                   (Path("/dev/shm/vec-otra-rama"),)))
        self.assertFalse(dentro_git(self.acta, raices))
        with self.assertRaisesRegex(NoEjecutado, "acta"):
            preparar(self.origen, compartida / "AGENTS.md", self.certificado, self.clave)

    def test_titulo_de_bolsa_coincide_con_catalogo(self):
        catalogo = json.loads((RAIZ_REPO / "web/static/textos/es/area-personal.json").read_text())
        self.assertEqual(catalogo["rutas"]["llamamientos"], "Disponibilidad y llamamientos")

    def test_capturas_privadas_no_sobrescriben_evidencia_ni_entran_en_git(self):
        with self.assertRaisesRegex(NoEjecutado, "evidencias"):
            preparar_evidencias(RAIZ_REPO / "capturas-prueba")
        destino = self.raiz / "capturas"
        self.assertEqual(preparar_evidencias(destino), destino.resolve())
        self.assertEqual(destino.stat().st_mode & 0o777, 0o700)
        with self.assertRaisesRegex(NoEjecutado, "evidencias"):
            preparar_evidencias(destino)

    def test_archivo_de_captura_es_privado_y_no_sobrescribe(self):
        ruta = self.raiz / "captura.png"
        guardar_captura(ruta, b"captura sintetica")
        self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
        with self.assertRaises(FileExistsError):
            guardar_captura(ruta, b"otro contenido")
        self.assertEqual(ruta.read_bytes(), b"captura sintetica")


class SalidasPrivadas(unittest.TestCase):
    """Fixtures de sistema de archivos; no importan Playwright ni abren red."""

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.raiz = Path(self.temp.name)

    def directorio(self, nombre):
        ruta = self.raiz / nombre
        ruta.mkdir(mode=0o700)
        return ruta

    def test_otro_repositorio_y_worktree_se_rechazan_antes_de_chrome(self):
        for clase in ("repositorio", "worktree"):
            with self.subTest(clase=clase):
                raiz = self.directorio(clase)
                if clase == "repositorio":
                    (raiz / ".git").mkdir()
                else:
                    (raiz / ".git").write_text("gitdir: /referencia-sintetica/worktrees/otro\n")
                padre = raiz / "privado"
                padre.mkdir(mode=0o700)
                with self.assertRaises(NoEjecutado):
                    preparar_evidencias(padre / "capturas")
                self.assertFalse((padre / "capturas").exists())
                with patch("builtins.__import__") as importar:
                    with self.assertRaises(NoEjecutado):
                        ejecutar("https://127.0.0.1:18531", Path("chrome"),
                                 Path("cert"), Path("clave"), padre)
                    importar.assert_not_called()

    def test_bare_y_enlace_git_se_rechazan(self):
        bare = self.directorio("bare")
        (bare / "HEAD").write_text("ref: refs/heads/main\n")
        (bare / "objects").mkdir()
        (bare / "config").write_text("[core]\n bare = true\n")
        with self.assertRaises(NoEjecutado):
            preparar_evidencias(bare / "capturas")
        repo = self.directorio("git-enlace")
        (repo / ".git").symlink_to(self.raiz / "git-ausente")
        with self.assertRaises(NoEjecutado):
            preparar_evidencias(repo / "capturas")

    def test_ancestro_enlazado_y_ruta_con_subida_se_rechazan(self):
        padre = self.directorio("privado")
        enlace = self.raiz / "enlace"
        enlace.symlink_to(padre, target_is_directory=True)
        with self.assertRaises(NoEjecutado):
            preparar_evidencias(enlace / "capturas")
        with self.assertRaises(NoEjecutado):
            preparar_evidencias(padre / ".." / "capturas")
        with self.assertRaises(OSError):
            guardar_captura(enlace / "archivo.png", b"sintetico")
        self.assertFalse((padre / "archivo.png").exists())

    def test_padre_abierto_o_ajeno_se_rechaza(self):
        padre = self.directorio("abierto")
        padre.chmod(0o755)
        with self.assertRaises(NoEjecutado):
            preparar_evidencias(padre / "capturas")
        with self.assertRaises(OSError):
            guardar_captura(padre / "archivo.png", b"sintetico")
        padre.chmod(0o700)
        with patch("recorrer.os.getuid", return_value=os.getuid() + 1):
            with self.assertRaises(NoEjecutado):
                preparar_evidencias(padre / "capturas")

    def test_enlace_y_hardlink_de_captura_no_se_sobrescriben(self):
        contenido = self.raiz / "original.png"
        contenido.write_bytes(b"original sintetico")
        for clase in ("symlink", "hardlink"):
            with self.subTest(clase=clase):
                padre = self.directorio(clase)
                ruta = padre / "area-personal-1440.png"
                if clase == "symlink":
                    ruta.symlink_to(contenido)
                else:
                    os.link(contenido, ruta)
                with self.assertRaises(NoEjecutado):
                    validar_capturas(padre)
                with self.assertRaises(FileExistsError):
                    guardar_captura(ruta, b"otro contenido")
                self.assertEqual(contenido.read_bytes(), b"original sintetico")

    def test_exterior_privado_admite_dos_capturas_y_no_sobrescribe(self):
        carpeta = preparar_evidencias(self.raiz / "capturas")
        self.assertEqual(carpeta.stat().st_mode & 0o777, 0o700)
        validar_capturas(carpeta)
        for ancho in (1440, 390):
            ruta = carpeta / f"area-personal-{ancho}.png"
            guardar_captura(ruta, b"captura sintetica")
            self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
            self.assertEqual(ruta.stat().st_nlink, 1)
            with self.assertRaises(FileExistsError):
                guardar_captura(ruta, b"otro contenido")
            self.assertEqual(ruta.read_bytes(), b"captura sintetica")


class Redirecciones(unittest.TestCase):
    @unittest.skipUnless(shutil.which("google-chrome"), "falta Chrome del sistema")
    def test_302_local_a_otro_puerto_no_alcanza_destino(self):
        from playwright.sync_api import Error, sync_playwright

        alcanzadas = []

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                alcanzadas.append(self.path)
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"destino")

            def log_message(self, *_):
                pass

        destino = ThreadingHTTPServer(("127.0.0.1", 0), Destino)
        hilo_destino = threading.Thread(target=destino.serve_forever, daemon=True)
        hilo_destino.start()
        self.addCleanup(hilo_destino.join, 2)
        self.addCleanup(destino.server_close)
        self.addCleanup(destino.shutdown)

        class Origen(BaseHTTPRequestHandler):
            def do_GET(self):
                if self.path == "/salir":
                    self.send_response(302)
                    self.send_header("Location", f"http://127.0.0.1:{destino.server_port}/destino")
                    self.end_headers()
                else:
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(b"origen")

            def log_message(self, *_):
                pass

        servidor = ThreadingHTTPServer(("127.0.0.1", 0), Origen)
        hilo_origen = threading.Thread(target=servidor.serve_forever, daemon=True)
        hilo_origen.start()
        self.addCleanup(hilo_origen.join, 2)
        self.addCleanup(servidor.server_close)
        self.addCleanup(servidor.shutdown)

        origen = f"http://127.0.0.1:{servidor.server_port}"
        incidencias = []
        with sync_playwright() as playwright:
            navegador = playwright.chromium.launch(executable_path=shutil.which("google-chrome"), headless=True)
            try:
                contexto = navegador.new_context(service_workers="block")
                contexto.route("**/*", lambda ruta: filtrar_red(ruta, origen, incidencias))
                pagina = contexto.new_page()
                self.assertEqual(pagina.goto(origen + "/bien").status, 200)
                with self.assertRaises(Error):
                    pagina.goto(origen + "/salir", timeout=5000)
                self.assertEqual(alcanzadas, [])
                self.assertIn("redireccion_bloqueada", incidencias)
                contexto.close()
            finally:
                navegador.close()

    @unittest.skipUnless(shutil.which("google-chrome"), "falta Chrome del sistema")
    def test_websocket_no_abre_handshake_externo(self):
        from playwright.sync_api import sync_playwright

        alcanzadas = []

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                alcanzadas.append(self.path)
                self.send_response(400)
                self.end_headers()

            def log_message(self, *_):
                pass

        servidor = ThreadingHTTPServer(("127.0.0.1", 0), Destino)
        hilo = threading.Thread(target=servidor.serve_forever, daemon=True)
        hilo.start()
        self.addCleanup(hilo.join, 2)
        self.addCleanup(servidor.server_close)
        self.addCleanup(servidor.shutdown)

        incidencias = []
        with sync_playwright() as playwright:
            navegador = playwright.chromium.launch(executable_path=shutil.which("google-chrome"), headless=True)
            try:
                contexto = navegador.new_context(service_workers="block")
                contexto.route_web_socket("**/*", lambda ruta: bloquear_websocket(ruta, incidencias))
                pagina = contexto.new_page()
                pagina.goto("about:blank")
                pagina.evaluate("url => { new WebSocket(url); return true; }",
                                f"ws://127.0.0.1:{servidor.server_port}/socket")
                pagina.wait_for_timeout(150)
                self.assertEqual(alcanzadas, [])
                self.assertIn("websocket_bloqueado", incidencias)
                contexto.close()
            finally:
                navegador.close()


if __name__ == "__main__":
    unittest.main()
