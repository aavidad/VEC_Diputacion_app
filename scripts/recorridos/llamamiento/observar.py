"""Primer corte de los portales; no crea peticiones de efecto."""

import json
import os
from pathlib import Path
from urllib.parse import urlsplit

from errores import NoEjecutado
from recorrido import mismo_origen, servir_solo_origen


IDIOMA = "en" if os.environ.get("LANG", "").startswith("en") else "es"
MENSAJES = json.loads((Path(__file__).parent / f"mensajes.{IDIOMA}.json").read_text())


def preparar_destino(destino):
    if not isinstance(destino, Path) or not destino.is_absolute():
        raise NoEjecutado(MENSAJES["destino"])
    if any((p / ".git").exists() for p in (destino, *destino.parents)):
        raise NoEjecutado(MENSAJES["destino"])
    if any(p.is_symlink() for p in (destino, *destino.parents)):
        raise NoEjecutado(MENSAJES["destino"])
    destino.mkdir(mode=0o700, parents=True, exist_ok=False)
    return destino


def observar_apertura(escenario, destino):
    try:
        from playwright.sync_api import sync_playwright
    except ImportError as error:
        raise NoEjecutado(MENSAJES["navegador"]) from error
    if not Path("/usr/bin/google-chrome").is_file():
        raise NoEjecutado(MENSAJES["navegador"])

    destino = preparar_destino(destino)
    informe = {"estado": "APERTURA_OBSERVADA", "operaciones_completadas": [],
               "perfiles": {}, "reinicio": False}
    origen = escenario["origen"]
    with sync_playwright() as playwright:
        navegador = playwright.chromium.launch(headless=True, executable_path="/usr/bin/google-chrome")
        try:
            for actor, identidad in escenario["identidades"].items():
                resultado = {"http": [], "errores_js": 0, "set_cookie": 0,
                             "externas": 0, "fallos_red": 0, "vistas": []}
                informe["perfiles"][actor] = resultado
                contexto = navegador.new_context(
                    client_certificates=[{"origin": origen, "certPath": identidad["certificado"],
                                          "keyPath": identidad["clave"]}],
                    service_workers="block", ignore_https_errors=False, locale="es-ES",
                    timezone_id="Europe/Madrid", viewport={"width": 1440, "height": 900})
                contexto.route("**/*", lambda route: servir_solo_origen(route, origen))
                contexto.route_web_socket("**/*", lambda websocket: websocket.close())
                pagina = contexto.new_page()

                def contar(campo):
                    resultado[campo] += 1

                def respuesta(r):
                    resultado["set_cookie"] += int("set-cookie" in r.headers)
                    if urlsplit(r.url).path.startswith("/api/vec/") or r.request.is_navigation_request():
                        resultado["http"].append({"metodo": r.request.method,
                                                  "ruta": urlsplit(r.url).path, "estado": r.status})

                pagina.on("pageerror", lambda _: contar("errores_js"))
                pagina.on("response", respuesta)
                pagina.on("requestfailed", lambda _: contar("fallos_red"))
                pagina.on("request", lambda r: contar("externas")
                          if not mismo_origen(r.url, origen) else None)
                try:
                    pagina.goto(origen + identidad["ruta"], wait_until="networkidle", timeout=30000)
                except Exception as error:
                    resultado["fallo"] = type(error).__name__
                finally:
                    for ancho, alto in ((1440, 900), (390, 844)):
                        estado = {"ancho": ancho, "captura": False}
                        try:
                            pagina.set_viewport_size({"width": ancho, "height": alto})
                            pagina.wait_for_timeout(250)
                            captura = destino / f"llamamiento-{actor}-{ancho}.png"
                            pagina.screenshot(path=str(captura), full_page=True, timeout=5000)
                            captura.chmod(0o600)
                            estado["captura"] = True
                            estado.update(pagina.evaluate("""async () => ({
                                ancho: document.documentElement.clientWidth,
                                contenido: document.documentElement.scrollWidth,
                                local: localStorage.length, sesion: sessionStorage.length,
                                indexeddb: typeof indexedDB.databases === 'function'
                                  ? (await indexedDB.databases()).length : null
                            })"""))
                            estado["cookies"] = len(contexto.cookies())
                        except Exception as error:
                            estado["fallo"] = type(error).__name__
                        resultado["vistas"].append(estado)
                    contexto.close()
        finally:
            navegador.close()
    fallos_http = [h for p in informe["perfiles"].values() for h in p["http"] if h["estado"] >= 400]
    informe["primer_corte_http"] = fallos_http[0] if fallos_http else None
    fallo = bool(fallos_http) or any(
        p.get("fallo") or any(p[k] for k in ("errores_js", "set_cookie", "externas", "fallos_red"))
        or any(v.get("fallo") or v.get("contenido", 0) > v["ancho"] or v.get("local")
               or v.get("sesion") or v.get("cookies") or v.get("indexeddb") for v in p["vistas"])
        for p in informe["perfiles"].values())
    if fallo:
        informe["estado"] = "CORTE_APLICACION"
    informe["flujo_posterior"] = "NO_EJECUTADO"
    ruta = destino / "resultado.json"
    ruta.write_text(json.dumps(informe, ensure_ascii=False, indent=2) + "\n")
    ruta.chmod(0o600)
    print(MENSAJES["corte" if fallo else "apertura"])
    return 1 if fallo else 0
