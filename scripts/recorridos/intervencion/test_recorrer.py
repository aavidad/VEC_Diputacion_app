"""Prueba focal del cierre previo y del vínculo entre solicitud y recibo."""

from __future__ import annotations

import hashlib
import http.server
import io
import json
import os
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
    def datos_configuracion(self, raiz):
        for nombre in ("chrome", "vec-server", "int.crt", "int.key", "rrhh.crt", "rrhh.key"):
            (raiz / nombre).write_text(nombre, encoding="utf-8")
        return {"origen": "https://127.0.0.1:18531", "chrome": str(raiz / "chrome"),
                "binario": str(raiz / "vec-server"),
                "binario_sha256": hashlib.sha256(b"vec-server").hexdigest(),
                "hitos_clon": ["H3", "H4", "H5"], "uso_sintetico": True,
                "intervencion_cert": str(raiz / "int.crt"), "intervencion_key": str(raiz / "int.key"),
                "rrhh_cert": str(raiz / "rrhh.crt"), "rrhh_key": str(raiz / "rrhh.key"),
                "favorable": {"expediente_ref": "expediente:sintetico:a", "version_esperada": 5},
                "reparo": {"expediente_ref": "expediente:sintetico:b", "version_esperada": 5}}

    def test_otro_git_worktree_y_bare_se_rechazan_antes_de_chrome(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            datos = self.datos_configuracion(raiz)
            for tipo in ("repositorio", "worktree", "bare"):
                with self.subTest(tipo=tipo):
                    repo = raiz / tipo
                    repo.mkdir(mode=0o700)
                    privada = repo / "privada"
                    privada.mkdir(mode=0o700)
                    if tipo == "repositorio":
                        (repo / ".git").mkdir()
                    elif tipo == "worktree":
                        (repo / ".git").write_text("gitdir: /otro/repositorio", encoding="utf-8")
                    else:
                        (repo / "HEAD").write_text("ref: refs/heads/main", encoding="utf-8")
                        (repo / "objects").mkdir()
                        (repo / "config").write_text("[core]\n bare = true", encoding="utf-8")
                    salida = privada / "evidencia.json"
                    with patch.object(recorrer, "cargar_json", return_value=datos.copy()), \
                         patch.object(recorrer, "ejecutar") as navegador, \
                         patch("sys.argv", ["recorrer.py", "registrar", "--config", str(raiz / "config.json"),
                                            "--evidencia", str(salida)]), \
                         patch("sys.stderr", io.StringIO()):
                        self.assertEqual(recorrer.main(), 2)
                        navegador.assert_not_called()
                    self.assertEqual(list(privada.iterdir()), [])

    def test_padres_propios_0700_y_sin_symlinks(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            privada = raiz / "privada"
            privada.mkdir(mode=0o700)
            salida = privada / "evidencia.json"
            for modo in (0o755, 0o770):
                privada.chmod(modo)
                with self.assertRaises(OSError):
                    recorrer.preparar_destinos_privados(salida, "registrar")
                self.assertEqual(privada.stat().st_mode & 0o777, modo)
            privada.chmod(0o700)
            with patch.object(recorrer.os, "getuid", return_value=os.getuid() + 1):
                with self.assertRaises(OSError):
                    recorrer.abrir_directorio_privado(salida)
            enlace = raiz / "enlace"
            enlace.symlink_to(privada, target_is_directory=True)
            with self.assertRaises(OSError):
                recorrer.preparar_destinos_privados(enlace / salida.name, "registrar")
            capturas = privada / "evidencia-capturas"
            capturas.mkdir(mode=0o755)
            with self.assertRaises(OSError):
                recorrer.preparar_destinos_privados(salida, "registrar")
            capturas.rmdir()
            capturas.symlink_to(raiz, target_is_directory=True)
            with self.assertRaises(OSError):
                recorrer.preparar_destinos_privados(salida, "registrar")

    def test_legacy_y_ultima_instantanea_completa_sin_reescribir_al_recuperar(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            legacy = raiz / "legacy.json"
            original = {"esquema": "vec.recorrido-intervencion.v1",
                        "favorable": {"solicitud": {"clave_idempotencia": "clave:original"},
                                      "recibo": {"recibo_ref": "recibo:original"}}}
            legacy.write_text(json.dumps(original, indent=2), encoding="utf-8")
            legacy.chmod(0o600)
            previo = legacy.read_bytes()
            recorrer.preparar_destinos_privados(legacy, "recuperar")
            self.assertEqual(recorrer.cargar_evidencia(legacy), original)
            self.assertEqual(legacy.read_bytes(), previo)
            self.assertFalse((raiz / "legacy-capturas").exists())

            salida = raiz / "registro.json"
            recorrer.preparar_destinos_privados(salida, "registrar")
            recorrer.persistir(salida, {"esquema": original["esquema"]}, nuevo=True)
            primera = salida.read_bytes()
            inode = salida.stat().st_ino
            pendiente = {**original, "intencion_pendiente": {
                "operacion": "subsanacion", "solicitud": {"clave_idempotencia": "clave:pendiente"}}}
            recorrer.persistir(salida, pendiente)
            self.assertEqual(salida.stat().st_ino, inode)
            self.assertTrue(salida.read_bytes().startswith(primera))
            with salida.open("ab") as archivo:
                archivo.write(b'{"intencion_pendiente":')
            antes = salida.read_bytes()
            recorrer.preparar_destinos_privados(salida, "recuperar")
            self.assertEqual(recorrer.cargar_evidencia(salida), pendiente)
            self.assertEqual(salida.read_bytes(), antes)
            self.assertEqual(salida.stat().st_mode & 0o777, 0o600)
            with self.assertRaisesRegex(NoEjecutado, "identidad"):
                recorrer.persistir(salida, {"cambio": "no añadir tras interrupción"})
            with self.assertRaises(FileExistsError):
                recorrer.persistir(salida, {}, nuevo=True)
            self.assertEqual(salida.read_bytes(), antes)

            utf8 = raiz / "utf8-interrumpido.jsonl"
            utf8.write_bytes(json.dumps(pendiente).encode("utf-8") + b'\n{"observaciones":"\xc3')
            utf8.chmod(0o600)
            self.assertEqual(recorrer.cargar_evidencia(utf8), pendiente)

    def test_lectura_recovery_rechaza_sustitucion_desde_preflight(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            salida = raiz / "legacy.json"
            salida.write_text('{"clave_idempotencia":"original"}', encoding="utf-8")
            salida.chmod(0o600)
            recorrer.preparar_destinos_privados(salida, "recuperar")
            salida.rename(raiz / "conservada.json")
            salida.write_text('{"clave_idempotencia":"sustituida"}', encoding="utf-8")
            salida.chmod(0o600)
            with self.assertRaisesRegex(NoEjecutado, "validación previa"):
                recorrer.cargar_evidencia(salida)
            self.assertEqual((raiz / "conservada.json").read_text(encoding="utf-8"),
                             '{"clave_idempotencia":"original"}')

    def test_reapertura_niega_symlink_hardlink_modo_y_cambio_de_inode(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            salida = raiz / "evidencia.json"
            recorrer.persistir(salida, {"clave_idempotencia": "original"}, nuevo=True)
            original = salida.read_bytes()
            enlace = raiz / "hardlink.json"
            os.link(salida, enlace)
            with self.assertRaises(OSError):
                recorrer.cargar_evidencia(salida)
            with self.assertRaises(OSError):
                recorrer.persistir(salida, {"cambio": "rechazado"})
            self.assertEqual(salida.read_bytes(), original)
            enlace.unlink()
            salida.chmod(0o644)
            with self.assertRaises(OSError):
                recorrer.cargar_evidencia(salida)
            with self.assertRaises(OSError):
                recorrer.persistir(salida, {"cambio": "rechazado"})
            salida.chmod(0o600)
            conservada = raiz / "original.json"
            salida.rename(conservada)
            salida.symlink_to(conservada)
            with self.assertRaises(OSError):
                recorrer.preparar_destinos_privados(salida, "recuperar")
            with self.assertRaises(OSError):
                recorrer.persistir(salida, {"cambio": "rechazado"})
            salida.unlink()
            salida.write_text("{}", encoding="utf-8")
            salida.chmod(0o600)
            with self.assertRaisesRegex(NoEjecutado, "identidad"):
                recorrer.persistir(salida, {"cambio": "rechazado"})
            self.assertEqual(salida.read_text(encoding="utf-8"), "{}")
            self.assertEqual(conservada.read_bytes(), original)

    def test_captura_existente_no_se_sobrescribe_ni_se_aceptan_enlaces_al_recuperar(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            salida = raiz / "evidencia.json"
            recorrer.preparar_destinos_privados(salida, "registrar")
            captura = raiz / "evidencia-capturas" / "favorable-1440.png"
            victima = raiz / "conservar.png"
            recorrer.guardar_privado(victima, b"conservar")
            captura.symlink_to(victima)
            with self.assertRaises(NoEjecutado):
                recorrer.preparar_destinos_privados(salida, "registrar")
            recorrer.persistir(salida, {"clave_idempotencia": "original"}, nuevo=True)
            with self.assertRaises(OSError):
                recorrer.preparar_destinos_privados(salida, "recuperar")
            captura.unlink()
            os.link(victima, captura)
            with self.assertRaises(OSError):
                recorrer.preparar_destinos_privados(salida, "recuperar")
            self.assertEqual(victima.read_bytes(), b"conservar")

    def test_replay_fiscalizacion_201_conserva_peticion_y_recibo(self):
        origen = "https://127.0.0.1:18531"
        solicitud = {"expediente_ref": "expediente:sintetico:a", "version_esperada": 5,
                     "resultado": "favorable", "observaciones": "",
                     "clave_idempotencia": "clave:sintetica:original"}
        recibo = {"expediente_ref": solicitud["expediente_ref"], "version_resultante": 6,
                  "fase_resultante": "fiscalizacion", "estado_resultante": "en_curso",
                  "recibo_ref": "recibo:sintetico:original",
                  "registrada_en": "2026-09-29T00:00:00Z",
                  "auditoria_ref": "auditoria:sintetica:original",
                  "evento_ref": "evento:sintetico:original",
                  "actor_ref": "actor:intervencion:sintetico"}
        registro = {"solicitud": solicitud.copy(), "recibo": recibo.copy()}
        cuerpo = {**recibo, "resultado": "favorable"}
        respuesta = SimpleNamespace(status=201, url=origen + recorrer.RUTA_FISCAL,
                                    headers={}, json=lambda: {"data": cuerpo})
        peticiones = []

        def post(url, **opciones):
            peticiones.append({"url": url, **opciones})
            return respuesta

        contexto = SimpleNamespace(request=SimpleNamespace(post=post))
        estado = {"post_posible": False}
        self.assertEqual(recorrer.repetir(contexto, {"origen": origen}, recorrer.RUTA_FISCAL,
                                         registro, estado), 201)
        self.assertTrue(estado["post_posible"])
        self.assertEqual(peticiones, [{"url": origen + recorrer.RUTA_FISCAL,
                                      "data": solicitud, "timeout": 30_000, "max_redirects": 0}])
        self.assertEqual(registro, {"solicitud": solicitud, "recibo": recibo})

        for campo in ("recibo_ref", "registrada_en", "auditoria_ref", "evento_ref", "actor_ref"):
            with self.subTest(campo=campo):
                respuesta.json = lambda campo=campo: {"data": {**cuerpo, campo: "cambiado"}}
                with self.assertRaisesRegex(FalloRecorrido, "el replay cambió"):
                    recorrer.repetir(contexto, {"origen": origen}, recorrer.RUTA_FISCAL,
                                     registro, estado)

        respuesta.json = lambda: {"data": cuerpo}
        respuesta.status = 200
        with self.assertRaisesRegex(FalloRecorrido, "estado HTTP inesperado"):
            recorrer.repetir(contexto, {"origen": origen}, recorrer.RUTA_FISCAL, registro, estado)
        respuesta.status = 201
        respuesta.url = origen + recorrer.RUTA_SUBSANACION
        with self.assertRaisesRegex(FalloRecorrido, "estado HTTP inesperado"):
            recorrer.repetir(contexto, {"origen": origen}, recorrer.RUTA_FISCAL, registro, estado)

    def test_replay_subsanacion_201_conserva_peticion_y_recibo(self):
        origen = "https://127.0.0.1:18531"
        solicitud = {"expediente_ref": "expediente:sintetico:b", "version_esperada": 6,
                     "observaciones": "Subsanación sintética original.",
                     "clave_idempotencia": "clave:subsanacion:original"}
        recibo = {"expediente_ref": solicitud["expediente_ref"], "version_resultante": 7,
                  "fase_resultante": "subsanacion_unidad", "estado_resultante": "incidencia",
                  "recibo_ref": "recibo:subsanacion:original",
                  "registrada_en": "2026-09-29T00:01:00Z",
                  "auditoria_ref": "auditoria:subsanacion:original",
                  "evento_ref": "evento:subsanacion:original", "actor_ref": "actor:rrhh:sintetico"}
        registro = {"solicitud": solicitud.copy(), "recibo": recibo.copy()}
        respuesta = SimpleNamespace(status=201, url=origen + recorrer.RUTA_SUBSANACION,
                                    headers={}, json=lambda: {"data": recibo.copy()})
        peticiones = []

        def post(url, **opciones):
            peticiones.append({"url": url, **opciones})
            return respuesta

        contexto = SimpleNamespace(request=SimpleNamespace(post=post))
        estado = {"post_posible": False}
        self.assertEqual(recorrer.repetir(contexto, {"origen": origen}, recorrer.RUTA_SUBSANACION,
                                         registro, estado), 201)
        self.assertEqual(peticiones, [{"url": origen + recorrer.RUTA_SUBSANACION,
                                      "data": solicitud, "timeout": 30_000, "max_redirects": 0}])
        self.assertTrue(estado["post_posible"])
        for campo in ("recibo_ref", "registrada_en", "auditoria_ref", "evento_ref", "actor_ref"):
            with self.subTest(campo=campo):
                respuesta.json = lambda campo=campo: {"data": {**recibo, campo: "cambiado"}}
                with self.assertRaisesRegex(FalloRecorrido, "el replay cambió"):
                    recorrer.repetir(contexto, {"origen": origen}, recorrer.RUTA_SUBSANACION,
                                     registro, estado)
        respuesta.json = lambda: {"data": recibo.copy()}
        respuesta.status = 200
        with self.assertRaisesRegex(FalloRecorrido, "estado HTTP inesperado"):
            recorrer.repetir(contexto, {"origen": origen}, recorrer.RUTA_SUBSANACION, registro, estado)
        self.assertEqual(registro, {"solicitud": solicitud, "recibo": recibo})

        cantidad = len(peticiones)
        estado = {"post_posible": False}
        with self.assertRaisesRegex(FalloRecorrido, "ruta de recuperación no prevista"):
            recorrer.repetir(contexto, {"origen": origen}, "/api/otro", registro, estado)
        self.assertEqual(len(peticiones), cantidad)
        self.assertFalse(estado["post_posible"])

    def test_http_excluye_query_cabeceras_y_respuesta(self):
        respuesta = SimpleNamespace(
            url="https://127.0.0.1:18531/api/vec/recurso?secreto=no_guardar",
            status=503, request=SimpleNamespace(method="POST"),
            headers={"Authorization": "no_guardar"})
        evidencia = {}
        recorrer.observar_http(respuesta, evidencia)
        self.assertEqual(evidencia, {"http_observado": [
            {"metodo": "POST", "ruta": "/api/vec/recurso", "estado": 503}]})

    def test_capturas_privadas_conservan_geometria_aun_si_desborda(self):
        class Pagina:
            viewport = None

            def set_viewport_size(self, viewport):
                self.viewport = viewport

            def wait_for_timeout(self, _milisegundos):
                pass

            def screenshot(self, **_opciones):
                return b"imagen-sintetica"

            def evaluate(self, _expresion):
                return {"ancho": self.viewport["width"],
                        "contenido": self.viewport["width"] + 1,
                        "local": 0, "sesion": 0, "indexeddb": 0}

        with tempfile.TemporaryDirectory() as temporal:
            salida = Path(temporal) / "evidencia.json"
            recorrer.preparar_destinos_privados(salida, "registrar")
            pagina = Pagina()
            contexto = SimpleNamespace(cookies=lambda: [])
            capturas = recorrer.capturar_pagina(contexto, pagina, salida, "corte_intervencion")
            carpeta = salida.parent / "evidencia-capturas"
            self.assertEqual(carpeta.stat().st_mode & 0o777, 0o700)
            self.assertEqual(len(capturas), 2)
            for captura in capturas:
                self.assertEqual((carpeta / captura["archivo"]).stat().st_mode & 0o777, 0o600)
                self.assertEqual(captura["estado"]["contenido"], captura["estado"]["ancho"] + 1)
            self.assertEqual(pagina.viewport, {"width": 1440, "height": 900})
            with self.assertRaises(FileExistsError):
                recorrer.capturar_pagina(contexto, pagina, salida, "corte_intervencion")

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
