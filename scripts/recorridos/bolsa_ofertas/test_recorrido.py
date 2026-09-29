"""Puerta sintética del guion: nunca toca servicios ni inicia Chrome."""

import hashlib
import os
import stat
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from recorrido import (FalloRecorrido, NoEjecutado, cantidad_seleccionada, dentro_de_git,
                       guardar_captura, instante_obligatorio, instalar_filtro_red, referencia_obligatoria,
                       validar_configuracion, validar_origen, version_obligatoria)


class PuertaRecorrido(unittest.TestCase):
    def test_captura_privada_exclusiva_sin_seguir_enlaces(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            destino = raiz / "captura.png"
            mascara = os.umask(0)
            try:
                guardar_captura(destino, b"png sintetico")
            finally:
                os.umask(mascara)
            self.assertEqual(stat.S_IMODE(destino.stat().st_mode), 0o600)
            self.assertEqual(destino.read_bytes(), b"png sintetico")
            with self.assertRaises(FileExistsError):
                guardar_captura(destino, b"otra captura")
            enlace = raiz / "enlace.png"
            enlace.symlink_to(destino)
            with self.assertRaises(FileExistsError):
                guardar_captura(enlace, b"sobrescritura")
            self.assertEqual(destino.read_bytes(), b"png sintetico")

    def test_rechaza_destinos_remotos_y_credenciales_en_url(self):
        for origen in ("https://vec.cidonia.cloud:443", "http://127.0.0.1:8443",
                       "https://usuario:clave@127.0.0.1:8443", "https://127.0.0.1:8443/api"):
            with self.subTest(origen=origen), self.assertRaises(NoEjecutado):
                validar_origen(origen)

    def test_sin_clon_h3_h5_o_binario_exacto_no_se_abre_navegador(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            binario = raiz / "vec-server"
            binario.write_bytes(b"binario sintetico")
            cert = raiz / "rrhh.crt"
            clave = raiz / "rrhh.key"
            cert_candidato = raiz / "candidato.crt"
            clave_candidato = raiz / "candidato.key"
            cert.write_text("certificado rrhh")
            clave.write_text("clave rrhh")
            cert_candidato.write_text("certificado candidato")
            clave_candidato.write_text("clave candidato")
            clave.chmod(0o600)
            clave_candidato.chmod(0o600)
            datos = {
                "origen": "https://127.0.0.1:8443", "clon": "aislado_h3_h4_h5",
                "hitos": ["H3", "H4", "H5"], "binario": str(binario),
                "binario_sha256": hashlib.sha256(binario.read_bytes()).hexdigest(),
                "rrhh": {"certificado": str(cert), "clave": str(clave)},
                "candidato": {"certificado": str(cert_candidato), "clave": str(clave_candidato)},
                "bolsa_ref": "bolsa:sintetica", "bolsa_sintetica_reservada": True,
                "oferta": {"categoria": "Auxiliar", "centro": "Centro sintético",
                           "fecha_inicio": "2026-10-15", "descripcion": "Ensayo"},
            }
            sin_clon = {**datos, "hitos": ["H3", "H4"]}
            with self.assertRaisesRegex(NoEjecutado, "clon aislado"):
                validar_configuracion(sin_clon, "alta")
            sin_binario = {**datos, "binario_sha256": "0" * 64}
            with self.assertRaisesRegex(NoEjecutado, "binario VEC exacto"):
                validar_configuracion(sin_binario, "alta")
            self.assertEqual(validar_configuracion(datos, "alta")["bolsa_ref"], "bolsa:sintetica")
            with self.assertRaisesRegex(NoEjecutado, "reinicio"):
                validar_configuracion(datos, "recuperar")

            misma_ruta = {**datos, "candidato": {"certificado": str(cert), "clave": str(clave_candidato)}}
            with self.assertRaisesRegex(NoEjecutado, "certificados mTLS distintos"):
                validar_configuracion(misma_ruta, "alta")
            cert_duplicado = raiz / "otro-candidato.crt"
            cert_duplicado.write_bytes(cert.read_bytes())
            misma_huella = {**datos, "candidato": {"certificado": str(cert_duplicado), "clave": str(clave_candidato)}}
            with self.assertRaisesRegex(NoEjecutado, "certificados mTLS distintos"):
                validar_configuracion(misma_huella, "alta")
            clave_duplicada = raiz / "otra-candidato.key"
            clave_duplicada.write_bytes(clave.read_bytes())
            clave_duplicada.chmod(0o600)
            misma_clave = {**datos, "candidato": {"certificado": str(cert_candidato), "clave": str(clave_duplicada)}}
            with self.assertRaisesRegex(NoEjecutado, "claves mTLS distintos"):
                validar_configuracion(misma_clave, "alta")

    def test_material_en_otro_worktree_se_rechaza(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            (raiz / ".git").mkdir()
            otro = raiz / ".worktrees" / "otro"
            otro.mkdir(parents=True)
            secreto = otro / "clave.key"
            secreto.write_text("sintetico")
            self.assertTrue(dentro_de_git(secreto))
            enlace = raiz.parent / (raiz.name + "-enlace")
            enlace.symlink_to(secreto)
            try:
                self.assertTrue(dentro_de_git(enlace))
            finally:
                enlace.unlink()

    def test_campos_durables_omitidos_nunca_comparan_none_con_none(self):
        for comprobar, campo in ((version_obligatoria, "version"),
                                 (instante_obligatorio, "publicada_en"),
                                 (instante_obligatorio, "manifestada_en"),
                                 (referencia_obligatoria, "recibo_ref")):
            with self.subTest(campo=campo), self.assertRaises(FalloRecorrido):
                comprobar({}, campo)
        with self.assertRaises(FalloRecorrido):
            version_obligatoria({"version": True}, "version")
        with self.assertRaises(FalloRecorrido):
            instante_obligatorio({"publicada_en": "2026-02-30T10:00:00Z"}, "publicada_en")

    def test_total_b7_sale_del_estado_global_mas_alla_de_una_pagina(self):
        self.assertEqual(cantidad_seleccionada("12 seleccionadas de 15 candidaturas que cumplen el filtro."), 12)
        self.assertEqual(cantidad_seleccionada("100 seleccionadas."), 100)

    def test_redirect_a_otro_puerto_no_contacta_destino(self):
        contactos = {"origen": 0, "destino": 0}

        class Destino(BaseHTTPRequestHandler):
            def do_GET(self):
                contactos["destino"] += 1
                self.send_response(200)
                self.end_headers()

            def log_message(self, *_):
                pass

        destino = ThreadingHTTPServer(("127.0.0.1", 0), Destino)

        class Origen(BaseHTTPRequestHandler):
            def do_GET(self):
                contactos["origen"] += 1
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
            from playwright.sync_api import Error, sync_playwright

            with sync_playwright() as playwright:
                navegador = playwright.chromium.launch(headless=True, executable_path="/usr/bin/google-chrome",
                    args=["--disable-background-networking", "--disable-extensions", "--disable-sync"])
                try:
                    contexto = navegador.new_context(service_workers="block")
                    fallos = []
                    base = f"http://127.0.0.1:{origen.server_port}"
                    instalar_filtro_red(contexto, base, fallos)
                    try:
                        contexto.new_page().goto(base + "/", wait_until="domcontentloaded")
                    except Error:
                        pass
                    self.assertEqual(contactos, {"origen": 1, "destino": 0})
                    self.assertIn("redirección rechazada sin seguirla", fallos)
                finally:
                    navegador.close()
        finally:
            origen.shutdown()
            destino.shutdown()
            origen.server_close()
            destino.server_close()


if __name__ == "__main__":
    unittest.main()
