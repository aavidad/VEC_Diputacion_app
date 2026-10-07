"""Comprueba límites y evita convertir un cierre esperado en éxito funcional."""

from pathlib import Path
import argparse
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
import json
import os
import io
from contextlib import redirect_stdout
from types import SimpleNamespace
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch

from convoca_integrado import (API_BOLSA, API_PREFERENCIAS, CAPACIDADES, CONSULTAR_JS, cargar_expectativas,
                              clasificar, consola_inesperada, crear_contexto, main, registrar_consola,
                              registrar_fallo_red, registrar_http, url_local, validar_material_externo)


class LimitesRecorrido(unittest.TestCase):
    def test_destinos_fuera_de_loopback_y_credenciales_no_se_admiten(self):
        for url in ("https://example.com:443", "http://127.0.0.1:8080@ejemplo.invalid",
                    "http://usuario:clave@localhost:8080", "http://localhost:8080/?token=valor",
                    "http://localhost:8080/area-personal/", "http://localhost"):
            with self.subTest(url=url), self.assertRaises(ValueError):
                url_local(url)
        self.assertEqual(url_local("http://127.0.0.1:8080/"), "http://127.0.0.1:8080")
        self.assertEqual(url_local("http://[::1]:8080"), "http://[::1]:8080")

    def test_cierre_no_completa_flujo_y_solo_codigo_exacto_identifica_h6(self):
        expectativa = {"estado": "cerrada", "http": [403, 404, 503], "codigos_h6": ["h6_sin_fuente"]}
        for http in (403, 404, 503):
            resultado = clasificar("externo.mi_bolsa", {"http": http}, expectativa)
            self.assertTrue(resultado["coincide"])
            self.assertFalse(resultado["flujo_funcional_completado"])
            self.assertFalse(resultado["dependencia_h6_identificada"])
        confirmado = clasificar("externo.mi_bolsa", {"http": 503, "codigo": "h6_sin_fuente"}, expectativa)
        self.assertTrue(confirmado["dependencia_h6_identificada"])
        inesperado = clasificar("externo.mi_bolsa", {"http": 401}, expectativa)
        self.assertFalse(inesperado["coincide"])

    def test_http_200_invalido_o_inesperado_no_es_conforme(self):
        disponible = {"estado": "disponible"}
        invalido = clasificar("publico.listado", {"http": 200, "contrato_valido": False}, disponible)
        self.assertFalse(invalido["coincide"])
        cerrado = {"estado": "cerrada", "http": [403]}
        inesperado = clasificar("externo.mi_bolsa", {"http": 200, "contrato_valido": True}, cerrado)
        self.assertFalse(inesperado["coincide"])
        self.assertFalse(inesperado["flujo_funcional_completado"])

    def test_ejemplo_solo_declara_dependencias_auxiliares_concretas(self):
        capacidades, auxiliares = cargar_expectativas(Path(__file__).with_name("convoca_expectativas.json"))
        self.assertEqual(len(capacidades), 4)
        self.assertEqual(set(auxiliares), {"/api/vec/usuarios/area-personal/mi-imagen",
                                        "/api/vec/usuarios/area-personal/mis-correos"})

    def test_set_cookie_se_detecta_sin_retener_el_valor_sensible(self):
        class Respuesta:
            url = "http://127.0.0.1:8080/api/prueba"
            status = 200
            headers = {}

            def all_headers(self):
                return {"set-cookie": "fixture=sentinel; Max-Age=0; Path=/"}

        registro = registrar_http(Respuesta())
        self.assertTrue(registro["set_cookie"])
        self.assertNotIn("sentinel", json.dumps(registro))


class FiabilidadEvidencia(unittest.TestCase):
    def agregado(self, directorio, vista, cerrado=False):
        config = {"esquema": "vec.recorrido.convoca.v1", "datos": "sinteticos",
                  "capacidades": {c: {"estado": "disponible"} for c in CAPACIDADES}}
        if cerrado:
            config["capacidades"]["externo.mi_bolsa"] = {"estado": "cerrada", "http": [403]}
        ruta = directorio / "expectativas.json"
        ruta.write_text(json.dumps(config))

        def recorrer_stub(superficie, *_):
            capacidades = [{"capacidad": c, "coincide": True,
                            "estado": "dependencia_cerrada" if cerrado and c == "externo.mi_bolsa" else "consulta_disponible"}
                           for c in CAPACIDADES if c.startswith(superficie + ".") for _ in (1440, 390)]
            return {"superficie": superficie, "capacidades": capacidades, "vistas": [vista] if superficie == "externo" else []}

        with patch("convoca_integrado.recorrer", recorrer_stub), redirect_stdout(io.StringIO()):
            codigo = main(["--url-publico", "http://127.0.0.1:18091", "--url-externo", "http://127.0.0.1:18092",
                           "--datos-sinteticos", "--chrome", "/bin/true", "--expectativas", str(ruta),
                           "--salida", str(directorio / "resultado")])
        return codigo, json.loads((directorio / "resultado/resultado.json").read_text())

    def test_fallo_sin_http_impide_exito_con_ocho_capacidades_conformes(self):
        fallo = registrar_fallo_red(SimpleNamespace(url="http://127.0.0.1:18092/recurso?secreto=sentinel",
                                                   method="GET", failure="net::ERR_CONNECTION_RESET"))
        self.assertNotIn("sentinel", json.dumps(fallo))
        with tempfile.TemporaryDirectory() as temporal:
            codigo, informe = self.agregado(Path(temporal), {"fallos_red": [fallo]})
        self.assertEqual(codigo, 1)
        self.assertFalse(informe["consultas_conformes"])

    def test_error_consola_inesperado_impide_exito_con_ocho_capacidades_conformes(self):
        with tempfile.TemporaryDirectory() as temporal:
            codigo, informe = self.agregado(Path(temporal), {"errores_consola": 1})
        self.assertEqual(codigo, 1)
        self.assertFalse(informe["consultas_conformes"])

    def test_cierre_http_exacto_conserva_codigo_dos_y_no_exime_errores_de_aplicacion(self):
        mensaje = SimpleNamespace(text="Failed to load resource: the server responded with a status of 403 (Forbidden)",
                                  location={"url": "http://127.0.0.1:18092" + API_BOLSA, "lineNumber": 0, "columnNumber": 0})
        registro = registrar_consola(mensaje)
        vista = {"errores_consola": 1, "consola_error": [registro],
                 "red": [{"ruta": API_BOLSA, "http": 403, "metodo": "GET", "set_cookie": False}]}
        with tempfile.TemporaryDirectory() as temporal:
            codigo, informe = self.agregado(Path(temporal), vista, cerrado=True)
        self.assertEqual(codigo, 2)
        self.assertFalse(informe["flujo_funcional_completado"])
        mensaje.text = "sentinel"
        vista["consola_error"] = [registrar_consola(mensaje)]
        self.assertEqual(consola_inesperada(vista, {API_BOLSA: [403]}), 1)
        vista["consola_error"] = [registro, registro]
        self.assertEqual(consola_inesperada(vista, {API_BOLSA: [403]}), 1)
        vista["red"][0]["ruta"] = "/otra"
        self.assertEqual(consola_inesperada(vista, {API_BOLSA: [403]}), 2)

    def test_material_pareado_https_privado_y_solo_contexto_externo(self):
        with tempfile.TemporaryDirectory() as temporal:
            directorio = Path(temporal)
            cert, clave = directorio / "cert.pem", directorio / "key.pem"
            for p in (cert, clave):
                p.write_bytes(b"fixture_metadata_only")
                p.chmod(0o600)
            for base, primero, segundo in [("https://127.0.0.1:18443", cert, None),
                                          ("https://127.0.0.1:18443", None, clave),
                                          ("http://127.0.0.1:18092", cert, clave),
                                          ("https://example.com:443", cert, clave)]:
                with self.assertRaises(ValueError):
                    validar_material_externo(base, primero, segundo)
            material = validar_material_externo("https://127.0.0.1:18443", cert, clave)
            configuraciones = []
            browser = SimpleNamespace(new_context=lambda **kw: configuraciones.append(kw))
            args = SimpleNamespace(preparacion_local=False, material_externo=material)
            crear_contexto(browser, "publico", 1440, 900, args)
            crear_contexto(browser, "externo", 1440, 900, args)
            self.assertNotIn("client_certificates", configuraciones[0])
            self.assertEqual(configuraciones[1]["client_certificates"], material)
            self.assertFalse(configuraciones[1]["ignore_https_errors"])
            clave.chmod(0o644)
            with self.assertRaises(ValueError):
                validar_material_externo("https://127.0.0.1:18443", cert, clave)
            clave.chmod(0o600)
            enlace = directorio / "link.pem"
            enlace.symlink_to(cert)
            with self.assertRaises(ValueError):
                validar_material_externo("https://127.0.0.1:18443", enlace, clave)
            enlace_duro = directorio / "hardlink.pem"
            os.link(cert, enlace_duro)
            with self.assertRaises(ValueError):
                validar_material_externo("https://127.0.0.1:18443", cert, clave)
            enlace_duro.unlink()
            with patch("convoca_integrado.os.getuid", return_value=cert.stat().st_uid + 1), self.assertRaises(ValueError):
                validar_material_externo("https://127.0.0.1:18443", cert, clave)
            directorio.chmod(0o755)
            with self.assertRaises(ValueError):
                validar_material_externo("https://127.0.0.1:18443", cert, clave)
            directorio.chmod(0o700)
            (directorio / ".git").write_text("fixture")
            with self.assertRaises(ValueError):
                validar_material_externo("https://127.0.0.1:18443", cert, clave)


def comprobar_imports_200(servidor, salida, chrome):
    from playwright.sync_api import sync_playwright

    salida.mkdir(parents=True, mode=0o700)
    base = f"http://127.0.0.1:{servidor.server_port}"
    casos = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(executable_path=chrome, headless=True, timeout=30_000)
        try:
            context = browser.new_context(service_workers="block", viewport={"width": 1440, "height": 900})
            try:
                red = []
                def limitar(route):
                    if route.request.method in ("GET", "HEAD") and route.request.url.startswith(base + "/"):
                        route.continue_()
                    else:
                        route.abort()
                context.route("**/*", limitar)
                page = context.new_page()
                page.set_default_timeout(30_000)
                page.on("response", lambda res: red.append(registrar_http(res)))
                page.goto(base + "/area-personal/?vista=llamamientos&lang=es", wait_until="networkidle")
                observados = page.evaluate("""() => performance.getEntriesByType('resource')
                  .map(r => new URL(r.name)).filter(u => u.origin === location.origin &&
                    ['/area-personal/contrato.js','/area-personal/cliente-http.js'].includes(u.pathname))
                  .map(u => ({ruta:u.pathname, version:u.searchParams.get('v')}))""")
                rutas = {"externo.mi_bolsa": API_BOLSA, "externo.preferencias": API_PREFERENCIAS}
                for capacidad, ruta in rutas.items():
                    observado = page.evaluate(CONSULTAR_JS, {"ruta": ruta, "capacidad": capacidad})
                    modulo = observado.get("modulo", {})
                    if (observado.get("http") != 200 or not observado.get("contrato_valido")
                        or modulo.get("tipo") != "function"
                        or {k: modulo.get(k) for k in ("ruta", "version")} not in observados):
                        raise AssertionError("import_200_invalido")
                    casos.append({"caso": "contrato_200_valido", "capacidad": capacidad, "observado": observado})
                servidor.invalidar = True
                for capacidad, ruta in rutas.items():
                    observado = page.evaluate(CONSULTAR_JS, {"ruta": ruta, "capacidad": capacidad})
                    if observado.get("http") != 200 or observado.get("contrato_valido") or observado.get("fallo") != "consulta_o_contrato":
                        raise AssertionError("contrato_malformado_admitido")
                    casos.append({"caso": "contrato_200_rechazado", "capacidad": capacidad, "observado": observado})
                servidor.invalidar = False
                # Ocultar sólo la observación del módulo debe impedir incluso el GET.
                page.evaluate("() => { performance.getEntriesByType = () => []; }")
                previas = servidor.consultas
                for capacidad, ruta in rutas.items():
                    observado = page.evaluate(CONSULTAR_JS, {"ruta": ruta, "capacidad": capacidad})
                    if observado.get("http") != 0 or observado.get("contrato_valido") or servidor.consultas != previas:
                        raise AssertionError("consulta_sin_modulo_montado")
                    casos.append({"caso": "modulo_ausente_sin_red", "capacidad": capacidad, "observado": observado})
                # Una cookie expirada no permanece en el contexto, pero su emisión se denuncia.
                servidor.emitir_cookie = True
                with page.expect_response(lambda r: r.url == base + API_BOLSA) as respuesta_cookie:
                    page.evaluate("""async ruta => {const r = await fetch(ruta,
                      {method:'GET', cache:'no-store', credentials:'same-origin'}); await r.text();}""", API_BOLSA)
                header_emitido = registrar_http(respuesta_cookie.value)["set_cookie"]
                retenidas = context.cookies()
                emitidas = [r for r in red if r["set_cookie"]]
                if not header_emitido or not emitidas or retenidas:
                    raise AssertionError("cookie_expirada_no_detectada")
                casos.append({"caso": "set_cookie_expirada_detectada", "emisiones": len(emitidas), "cookies_retenidas": len(retenidas)})
            finally:
                context.close()
        finally:
            browser.close()
    informe = {"esquema": "vec.recorrido.convoca.imports.v1", "tipo_ejecucion": "prueba_guion",
               "codigo_salida": 0, "flujo_funcional_completado": False, "registro_acreditado": False,
               "persistencia_acreditada": False, "modulos_observados": observados, "casos": casos}
    (salida / "imports-200.json").write_text(json.dumps(informe, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"codigo_salida": 0, "tipo_ejecucion": "prueba_guion", "casos": len(casos)}))
    return 0


def ensayar_chrome() -> int:
    """Servidor de prueba privado y efímero; nunca es una fuente de producto."""
    parser = argparse.ArgumentParser()
    parser.add_argument("--ensayar-chrome", action="store_true", required=True)
    parser.add_argument("--salida", type=Path, required=True)
    parser.add_argument("--chrome", default="/usr/bin/google-chrome")
    parser.add_argument("--escenario", choices=("cerrado", "externo-200"), default="cerrado")
    args = parser.parse_args()
    raiz = Path(__file__).resolve().parents[2] / "web" / "static"
    if not (raiz / "bolsa" / "preparacion" / "index.html").is_file():
        parser.error("consumidor_preparacion_no_integrado")
    fixture = json.loads(Path(__file__).with_name("convoca_fixture.json").read_text())

    class Handler(SimpleHTTPRequestHandler):
        def __init__(self, *a, **kw):
            super().__init__(*a, directory=str(raiz), **kw)

        def log_message(self, *a):
            pass

        def do_GET(self):
            ruta = self.path.split("?")[0]
            if not ruta.startswith("/api/"):
                return super().do_GET()
            if self.server.externo:
                campo = {API_BOLSA: "mi_bolsa", API_PREFERENCIAS: "preferencias"}.get(ruta)
                if self.server.positivo and campo:
                    self.server.consultas += 1
                    estado, datos = 200, fixture["externo_200"][campo]
                    if self.server.invalidar:
                        datos = {"data": {"esquema": "fixture_invalida"}}
                else:
                    estado, datos = 403, {"error": {"codigo": "dependencia_sintetica_cerrada"}}
            elif ruta == "/api/publico/bolsa/categorias":
                estado, datos = 200, fixture["categorias"]
            elif ruta == "/api/publico/bolsa/convocatorias/sintetica-2026":
                estado, datos = 200, fixture["detalle"]
            elif ruta == "/api/publico/bolsa/convocatorias":
                estado, datos = 200, fixture["listado"]
            else:
                estado, datos = 404, {"error": {"codigo": "ruta_sintetica_desconocida"}}
            cuerpo = json.dumps(datos).encode()
            self.send_response(estado)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(cuerpo)))
            if self.server.emitir_cookie:
                self.send_header("Set-Cookie", "fixture=sentinel; Max-Age=0; Path=/")
            self.end_headers()
            self.wfile.write(cuerpo)

    servidores = []
    try:
        for externo in ((True,) if args.escenario == "externo-200" else (False, True)):
            servidor = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
            servidor.externo = externo
            servidor.positivo = args.escenario == "externo-200"
            servidor.invalidar = False
            servidor.emitir_cookie = False
            servidor.consultas = 0
            servidores.append(servidor)
            threading.Thread(target=servidor.serve_forever, daemon=True).start()
        if args.escenario == "externo-200":
            return comprobar_imports_200(servidores[0], args.salida, args.chrome)
        codigo = main(["--url-publico", f"http://127.0.0.1:{servidores[0].server_port}",
                       "--url-externo", f"http://127.0.0.1:{servidores[1].server_port}",
                       "--datos-sinteticos", "--tipo-ejecucion", "prueba_guion", "--preparacion-local",
                       "--chrome", args.chrome, "--salida", str(args.salida), "--timeout-ms", "30000"])
        informe = json.loads((args.salida / "resultado.json").read_text())
        if codigo != 2 or informe["flujo_funcional_completado"] or informe["registro_acreditado"] or informe["persistencia_acreditada"]:
            return 1
        return 0
    finally:
        for servidor in servidores:
            servidor.shutdown()
            servidor.server_close()


if __name__ == "__main__":
    if "--ensayar-chrome" in sys.argv:
        raise SystemExit(ensayar_chrome())
    unittest.main()
