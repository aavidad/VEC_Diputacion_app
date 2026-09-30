import json
import tempfile
import unittest
import hashlib
import os
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import recorrer


class RecorridoFirmaTest(unittest.TestCase):
    def test_salida_rechaza_git_worktree_permisos_y_enlaces(self):
        with tempfile.TemporaryDirectory() as tmp:
            raiz = Path(tmp)
            privado = raiz / "privado"
            privado.mkdir(mode=0o700)
            recorrer.guardar_privado(privado / "valido.json", b"sintetico")
            self.assertEqual((privado / "valido.json").read_bytes(), b"sintetico")
            for marcador in ("directorio", "fichero"):
                repo = raiz / marcador
                repo.mkdir(mode=0o700)
                if marcador == "directorio":
                    (repo / ".git").mkdir()
                else:
                    (repo / ".git").write_text("gitdir: otro", encoding="utf-8")
                hijo = repo / "capturas"
                hijo.mkdir(mode=0o700)
                with self.assertRaises(OSError):
                    recorrer.guardar_privado(hijo / "rechazada.png", b"sintetico")
                self.assertFalse((hijo / "rechazada.png").exists())
            bare = raiz / "bare"
            bare.mkdir(mode=0o700)
            (bare / "HEAD").write_text("ref: refs/heads/main", encoding="utf-8")
            (bare / "config").write_text("[core]\nbare = true", encoding="utf-8")
            (bare / "objects").mkdir()
            with self.assertRaises(OSError):
                recorrer.guardar_privado(bare / "rechazada.png", b"sintetico")
            compartido = raiz / "compartido"
            compartido.mkdir(mode=0o755)
            with self.assertRaises(OSError):
                recorrer.guardar_privado(compartido / "rechazada.png", b"sintetico")
            enlace = raiz / "enlace"
            enlace.symlink_to(privado, target_is_directory=True)
            with self.assertRaises(OSError):
                recorrer.guardar_privado(enlace / "rechazada.png", b"sintetico")
            with self.assertRaises(OSError):
                recorrer.guardar_privado(privado / ".." / "rechazada.png", b"sintetico")

    def test_capturas_del_primer_corte_son_privadas_y_no_sobrescriben(self):
        class Pagina:
            def set_viewport_size(self, dimensiones):
                self.ancho = dimensiones["width"]

            def screenshot(self, *, full_page):
                return b"captura sintetica"

            def evaluate(self, codigo):
                return {"ancho": self.ancho, "contenido": self.ancho}

        with tempfile.TemporaryDirectory() as tmp:
            carpeta = Path(tmp)
            escritorio = carpeta / "1440.png"
            movil = carpeta / "390.png"
            args = recorrer.argumentos(["--captura-escritorio", str(escritorio),
                                       "--captura-movil", str(movil)])
            informe = {"corte": "propuesta"}
            recorrer.capturar_corte(Pagina(), args, informe)
            for ruta in (escritorio, movil):
                self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
                self.assertEqual(ruta.read_bytes(), b"captura sintetica")
            self.assertTrue(informe["escritorio_1440"]["captura_guardada"])
            self.assertTrue(informe["movil_390"]["sin_desbordamiento"])
            segundo = {"corte": "propuesta"}
            recorrer.capturar_corte(Pagina(), args, segundo)
            self.assertFalse(segundo["escritorio_1440"]["captura_guardada"])
            self.assertEqual(segundo["escritorio_1440"]["error"], "FileExistsError")
            self.assertEqual(escritorio.read_bytes(), b"captura sintetica")

    def test_sin_clon_no_ejecuta_navegador(self):
        with self.assertRaises(recorrer.Corte) as error:
            recorrer.validar_entrada(recorrer.argumentos([]))
        self.assertEqual(error.exception.paso, "precondiciones")

    def test_binario_que_no_coincide_con_inventario_corta_antes_de_chrome(self):
        with tempfile.TemporaryDirectory() as tmp:
            carpeta = Path(tmp)
            binario = carpeta / "vec-server"
            binario.write_bytes(b"binario sintetico")
            binario.chmod(0o700)
            certificado = carpeta / "cliente.crt"
            clave = carpeta / "cliente.key"
            certificado.write_bytes(b"certificado sintetico")
            clave.write_bytes(b"clave sintetica")
            inventario = carpeta / "inventario.json"
            inventario.write_text(json.dumps({"clon": "local", "datos": "sinteticos",
                "hitos": ["H3", "H4", "H5"], "origen": "https://localhost:8443",
                "binario_sha256": hashlib.sha256(b"otro binario").hexdigest()}), encoding="utf-8")
            a = recorrer.argumentos(["--entorno", str(inventario), "--binario", str(binario),
                "--origen", "https://localhost:8443", "--certificado", str(certificado),
                "--clave", str(clave), "--expediente-ref", "expediente:sintetico",
                "--version-propuesta", "7"])
            with self.assertRaises(recorrer.Corte) as error:
                recorrer.validar_entrada(a)
            self.assertEqual(error.exception.paso, "precondiciones")
            self.assertIn("huella", error.exception.motivo)

    def test_solo_acepta_propuesta_en_version_indicada(self):
        expediente = {"resumen": {"expediente_ref": "expediente:sintetico", "version": 8,
                                  "fase_clave": "nombramiento", "estado_clave": "en_curso"},
                      "hitos": [{"accion_clave": "otro"}] * 6 +
                      [{"accion_clave": "registrar_propuesta_formalizacion", "version_expediente": 7},
                       {"accion_clave": "otro", "version_expediente": 8}]}
        self.assertEqual(recorrer.comprobar_propuesta(expediente, "expediente:sintetico", 7)["version_propuesta"], 7)
        with self.assertRaises(recorrer.Corte):
            recorrer.comprobar_propuesta(expediente, "expediente:ajeno", 7)
        with self.assertRaises(recorrer.Corte):
            recorrer.comprobar_propuesta(expediente, "expediente:sintetico", 8)

    def test_recibo_de_firma_exige_verificacion_y_no_eficacia(self):
        recibo = {"expediente_ref": "expediente:sintetico", "documento": "informe_definitivo",
                  "resultado": "firmado", "firma_eficaz": False, "firma_verificada": True,
                  "registrada_en": "2026-09-29T20:00:00Z", "paso_orden": 1,
                  "recibo_ref": "recibo:prueba", "verificacion": {"estado": "valida", "motivo": "verificada",
                  "firmado_sha256": "a" * 64}}
        self.assertEqual(recorrer.resumen_firma(recibo, "expediente:sintetico", "informe_definitivo")["recibo_ref"], "recibo:prueba")
        with self.assertRaises(recorrer.Corte):
            recorrer.resumen_firma({**recibo, "firma_eficaz": True}, "expediente:sintetico", "informe_definitivo")
        with self.assertRaises(recorrer.Corte):
            recorrer.resumen_firma({**recibo, "verificacion": {**recibo["verificacion"], "estado": "indeterminada"}},
                                   "expediente:sintetico", "informe_definitivo")

    def test_recibo_de_propuesta_sale_de_consulta_separada(self):
        datos = {"estado": {"expediente_ref": "expediente:sintetico", "propuestas": [
            {"version_resultante": 7, "recibo_ref": "recibo:propuesta", "confirmada_en": "2026-09-06T00:00:00Z"}]}}
        self.assertEqual(recorrer.comprobar_recibo_propuesta(datos, "expediente:sintetico", 7)["recibo_ref"], "recibo:propuesta")
        with self.assertRaises(recorrer.Corte):
            recorrer.comprobar_recibo_propuesta(datos, "expediente:sintetico", 8)
        with self.assertRaises(recorrer.Corte):
            recorrer.comprobar_recibo_propuesta({"estado": {**datos["estado"], "propuestas": datos["estado"]["propuestas"] * 2}},
                                               "expediente:sintetico", 7)

    def test_recuperacion_exige_huellas_y_recibo_iguales(self):
        firma = {"recibo_ref": "recibo:prueba", "registrada_en": "2026-09-29T20:00:00Z",
                 "estado": "firmado", "paso_orden": 1, "firma_ref": "firma:origen", "firmado_sha256": "b" * 64}
        recuperada = {"recibo_ref": "recibo:prueba", "registrada_en": "2026-09-29T20:00:00Z",
                      "estado": "firmado", "paso_orden": 1}
        previo = {"expediente_ref": "expediente:sintetico", "propuesta": {"version_propuesta": 7},
                  "pdf": {"informe-definitivo": {"sha256": "a" * 64}},
                  "firma": firma}
        actual = {**previo, "firma": None, "firma_recuperada": recuperada}
        with tempfile.TemporaryDirectory() as tmp:
            ruta = Path(tmp) / "anterior.json"
            ruta.write_text(json.dumps(previo), encoding="utf-8")
            self.assertTrue(recorrer.comparar_recuperacion(actual, ruta))
            with self.assertRaises(recorrer.Corte):
                recorrer.comparar_recuperacion({**actual, "pdf": {}}, ruta)
            with self.assertRaises(recorrer.Corte):
                recorrer.comparar_recuperacion({**actual, "firma_recuperada": None}, ruta)
            with self.assertRaises(recorrer.Corte):
                recorrer.comparar_recuperacion({**actual, "firma_recuperada": {
                    **recuperada, "registrada_en": "2026-09-29T20:00:01Z"}}, ruta)
            self.assertEqual(recorrer.firma_recuperada({"firma_eficaz": False, "documentos": [
                {"documento": "informe_definitivo", "pasos": [{"orden": 1, "estado": "firmado",
                "recibo_ref": "recibo:prueba", "registrada_en": "2026-09-29T20:00:00Z"}]}]},
                "recibo:prueba"), recuperada)

    def test_cdp_conserva_transporte_y_corta_otros_origenes(self):
        class CDP:
            def send(self, metodo, parametros):
                self.llamada = (metodo, parametros)

        for url, permitido in (("https://localhost:8443/portal-empleado/", True),
                               ("https://localhost:8444/otro", False),
                               ("http://localhost:8443/otro", False),
                               ("https://example.invalid/otro", False)):
            cdp = CDP()
            recorrer.limitar_peticion_cdp(cdp, {"requestId": "prueba", "request": {"url": url}},
                                        "https://localhost:8443")
            self.assertEqual(cdp.llamada[0], "Fetch.continueRequest" if permitido else "Fetch.failRequest")

    def test_respuesta_redirect_se_corta_antes_de_seguir(self):
        class CDP:
            def send(self, metodo, parametros):
                self.llamada = (metodo, parametros)

        for estado in (301, 302, 303, 307, 308):
            cdp = CDP()
            recorrer.limitar_peticion_cdp(cdp, {"requestId": "prueba", "responseStatusCode": estado,
                                               "request": {"url": "https://localhost:8443/entrada"}},
                                        "https://localhost:8443")
            self.assertEqual(cdp.llamada, ("Fetch.failRequest",
                {"requestId": "prueba", "errorReason": "BlockedByClient"}))
        cdp = CDP()
        recorrer.limitar_peticion_cdp(cdp, {"requestId": "prueba", "responseStatusCode": 200,
                                           "request": {"url": "https://localhost:8443/entrada"}},
                                    "https://localhost:8443")
        self.assertEqual(cdp.llamada, ("Fetch.continueRequest", {"requestId": "prueba"}))

    def test_fallo_transporte_no_continua_peticion(self):
        class CDP:
            def send(self, *args):
                raise AssertionError("no debe continuar un transporte fallido")

        with self.assertRaises(recorrer.Corte) as corte:
            recorrer.limitar_peticion_cdp(CDP(), {"requestId": "prueba",
                "responseErrorReason": "ConnectionFailed"}, "https://localhost:8443")
        self.assertEqual(corte.exception.paso, "guardia_red")

    def test_guardia_global_cierra_chrome_si_instalacion_o_intercepcion_fallan(self):
        class CDP:
            fallo = False

            def on(self, evento, callback):
                if evento == "Fetch.requestPaused":
                    self.callback = callback
                else:
                    self.close_callback = callback

            def send(self, metodo, parametros):
                if self.fallo:
                    raise RuntimeError("fallo sintético")

        class Browser:
            cerrado = False

            def __init__(self):
                self.cdp = CDP()

            def new_browser_cdp_session(self):
                return self.cdp

            def on(self, evento, callback):
                self.desconectado = callback

            def close(self):
                self.cerrado = True

        browser = Browser()
        browser.cdp.fallo = True
        with self.assertRaises(recorrer.Corte):
            recorrer.GuardiaNavegador(browser, "https://localhost:8443")
        self.assertTrue(browser.cerrado)

        for fallo in ("intercepcion", "transporte", "desconexion", "sesion"):
            browser = Browser()
            guardia = recorrer.GuardiaNavegador(browser, "https://localhost:8443")
            evento = {"requestId": "prueba", "request": {"url": "https://localhost:8443/"}}
            if fallo == "intercepcion":
                browser.cdp.fallo = True
                browser.cdp.callback(evento)
            elif fallo == "transporte":
                browser.cdp.callback({**evento, "responseErrorReason": "ConnectionFailed"})
            elif fallo == "desconexion":
                browser.desconectado()
            else:
                browser.cdp.close_callback(None)
            self.assertTrue(guardia.errores)
            guardia.cerrar()
            self.assertTrue(browser.cerrado)

    def test_websocket_solo_autofirma_explicita(self):
        class Ruta:
            def __init__(self, url):
                self.url = url
                self.conectada = False
                self.cerrada = False

            def connect_to_server(self):
                self.conectada = True

            def close(self):
                self.cerrada = True

        permitida = Ruta("wss://127.0.0.1:63117")
        recorrer.limitar_websocket(permitida, True)
        self.assertTrue(permitida.conectada)
        denegada = Ruta("wss://127.0.0.1:63117")
        recorrer.limitar_websocket(denegada, False)
        self.assertTrue(denegada.cerrada)
        otro_puerto = Ruta("wss://127.0.0.1:63118")
        recorrer.limitar_websocket(otro_puerto, True)
        self.assertTrue(otro_puerto.cerrada)


@unittest.skipUnless(os.environ.get("VEC_PRUEBA_CHROME") == "1",
                     "Chrome focal requiere namespace local aislado")
class GuardiaChromeTest(unittest.TestCase):
    def setUp(self):
        self.hits = []
        prueba = self

        class Servidor(BaseHTTPRequestHandler):
            def do_GET(self):
                prueba.hits.append((self.server.label, self.path))
                if self.path == "/redirect":
                    self.send_response(302)
                    self.send_header("Location", prueba.sentinel + "/popup-redirect")
                    self.end_headers()
                    return
                if self.path == "/disconnect":
                    self.close_connection = True
                    return
                self.send_response(200)
                if self.path == "/pdf":
                    self.send_header("Content-Type", "application/pdf")
                    self.send_header("Content-Disposition", 'attachment; filename="fixture.pdf"')
                    cuerpo = prueba.pdf
                elif self.path == "/worker.js":
                    self.send_header("Content-Type", "application/javascript")
                    cuerpo = (f'fetch("{prueba.origin}/worker-positive");'
                              f'fetch("{prueba.sentinel}/worker").catch(()=>0);'
                              'postMessage("started")').encode()
                else:
                    self.send_header("Access-Control-Allow-Origin", "*")
                    cuerpo = b'<h1>fixture</h1><a href="/pdf" download>PDF</a>'
                self.end_headers()
                self.wfile.write(cuerpo)

            def log_message(self, *args):
                pass

        self.servers = []
        for label, host in (("origin", "127.0.0.1"), ("sentinel", "127.0.0.2")):
            servidor = ThreadingHTTPServer((host, 0), Servidor)
            servidor.label = label
            threading.Thread(target=servidor.serve_forever, daemon=True).start()
            self.servers.append(servidor)
        self.origin = f"http://127.0.0.1:{self.servers[0].server_port}"
        self.sentinel = f"http://127.0.0.2:{self.servers[1].server_port}"
        self.pdf = b"%PDF-1.4\nfixture-nativo\n%%EOF\n"

    def tearDown(self):
        for servidor in self.servers:
            servidor.shutdown()
            servidor.server_close()

    def test_browser_fetch_popup_worker_oopif_y_pdf(self):
        from playwright.sync_api import sync_playwright

        with sync_playwright() as pw:
            browser = pw.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True,
                args=["--site-per-process", "--enable-features=IsolateSandboxedIframes"])
            guardia = recorrer.GuardiaNavegador(browser, self.origin)
            eventos = []
            guardia.cdp.on("Fetch.requestPaused", lambda e: eventos.append(e))
            try:
                context = browser.new_context(service_workers="block", accept_downloads=True)
                page = context.new_page()
                self.assertEqual(page.goto(self.origin + "/main", wait_until="domcontentloaded").status, 200)
                with page.expect_download(timeout=5_000) as descarga:
                    page.get_by_text("PDF", exact=True).click()
                self.assertEqual(Path(descarga.value.path()).read_bytes(), self.pdf)
                page.evaluate('(url)=>window.open(url,"_blank")', self.sentinel + "/popup-direct")
                page.evaluate('(url)=>window.open(url,"_blank")', self.origin + "/redirect")
                page.evaluate('''()=>new Promise(resolve=>{
                    window.worker=new Worker('/worker.js');worker.onmessage=()=>resolve(true)
                })''')
                page.evaluate('''([origin,sentinel])=>{
                    let frame=document.createElement('iframe');frame.sandbox='allow-scripts';
                    frame.srcdoc='<script>fetch("'+origin+'/oopif-positive");fetch("'+sentinel+
                        '/oopif").catch(()=>0)</script>';document.body.append(frame)
                }''', [self.origin, self.sentinel])
                page.wait_for_timeout(700)
                tipos = {t["type"] for t in guardia.cdp.send("Target.getTargets")["targetInfos"]}
                self.assertIn("worker", tipos)
                self.assertIn("iframe", tipos)
                intentadas = {e["request"]["url"] for e in eventos}
                for ruta in ("/popup-direct", "/worker", "/oopif"):
                    self.assertIn(self.sentinel + ruta, intentadas)
                self.assertNotIn(self.sentinel + "/popup-redirect", intentadas)
                self.assertIn(("origin", "/redirect"), self.hits)
                self.assertIn(("origin", "/worker-positive"), self.hits)
                self.assertIn(("origin", "/oopif-positive"), self.hits)
                self.assertFalse(any(label == "sentinel" for label, ruta in self.hits))
                self.assertFalse(guardia.errores)
                print("Chrome global:", guardia.cdp.send("Browser.getVersion")["product"],
                      "targets=", sorted(tipos), "sentinel_hits=0; PDF original idéntico")
            finally:
                guardia.cerrar()

    def test_response_error_real_cierra_browser(self):
        from playwright.sync_api import sync_playwright, Error

        with sync_playwright() as pw:
            browser = pw.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
            guardia = recorrer.GuardiaNavegador(browser, self.origin)
            try:
                page = browser.new_context(service_workers="block").new_page()
                with self.assertRaises(Error):
                    page.goto(self.origin + "/disconnect", wait_until="domcontentloaded")
                self.assertTrue(guardia.errores)
            finally:
                guardia.cerrar()
            self.assertFalse(browser.is_connected())

    def test_perdida_session_real_cierra_browser(self):
        from playwright.sync_api import sync_playwright, Error

        with sync_playwright() as pw:
            browser = pw.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
            guardia = recorrer.GuardiaNavegador(browser, self.origin)
            try:
                browser.new_context(service_workers="block").new_page()
                try:
                    guardia.cdp.detach()
                except Error:
                    pass
                self.assertIn("sesion_guardia_desconectada", guardia.errores)
            finally:
                guardia.cerrar()
            self.assertFalse(browser.is_connected())


if __name__ == "__main__":
    unittest.main()
