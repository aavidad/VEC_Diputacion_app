#!/usr/bin/env python3
"""Capturas de revisión visual del portal VEC contra un despliegue real.

Abre cada pantalla en Chrome y guarda la captura en los anchos indicados. Con
--local sirve los ficheros estáticos (CSS, JS, HTML, i18n) desde un árbol de
trabajo en lugar del desplegado, de modo que se puede revisar un cambio visual
con datos reales antes de publicarlo. Informa de errores JS, respuestas /api con
estado >= 400 y desbordamiento horizontal de la página.

Destino y credencial llegan por entorno; nunca se escriben en el repositorio:
  VEC_URL      base del despliegue, p. ej. https://vec.example/portal-empleado/
  VEC_USUARIO  usuario de la autenticación básica del proxy (opcional)
  VEC_CLAVE    clave de esa autenticación (opcional)

Uso:
  capturar_pantallas.py SALIDA [--local web/static] [--anchos 1440,390]
                        [--pantalla nombre=#ruta[@texto_a_pulsar]] ...
Sin --pantalla se recorren las pantallas principales de RRHH.
"""

import argparse
import mimetypes
import os
import pathlib
import sys
from urllib.parse import urlsplit

from playwright.sync_api import sync_playwright

PANTALLAS_RRHH = [
    "inicio=#inicio",
    "cuadro_bolsa=#bolsa/resumen",
    "candidatos=#bolsa/resumen@ADMINISTRATIVO",
    "estadisticas=#bolsa/estadisticas",
    "bandeja_ct=#contratacion-temporal",
    "detalle_ct=#contratacion-temporal@fila",
]
ESPERA_MS = 5000


def interceptor(raiz: pathlib.Path):
    """Sirve desde `raiz` todo recurso estático que exista en el árbol local."""

    def servir(ruta):
        camino = urlsplit(ruta.request.url).path
        if camino.startswith("/api/"):
            return ruta.continue_()
        local = raiz / camino.lstrip("/")
        if camino.endswith("/"):
            local = local / "index.html"
        if not local.is_file():
            return ruta.continue_()
        tipo = mimetypes.guess_type(local.name)[0] or "application/octet-stream"
        if local.suffix == ".mjs":
            tipo = "text/javascript"
        return ruta.fulfill(status=200, content_type=tipo, body=local.read_bytes())

    return servir


def pulsar(pagina, texto):
    if texto == "fila":
        destino = pagina.locator("[data-ct-expediente], tbody tr").first
    else:
        destino = pagina.get_by_text(texto, exact=True).first
    if destino.count():
        destino.click()
        pagina.wait_for_timeout(3000)


def main():
    arg = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    arg.add_argument("salida", type=pathlib.Path)
    arg.add_argument("--local", type=pathlib.Path)
    arg.add_argument("--anchos", default="1440,390")
    arg.add_argument("--pantalla", action="append")
    arg.add_argument("--pagina-completa", action="store_true")
    opciones = arg.parse_args()

    base = os.environ.get("VEC_URL")
    if not base:
        sys.exit("Falta VEC_URL")
    credencial = None
    if os.environ.get("VEC_USUARIO"):
        credencial = {"username": os.environ["VEC_USUARIO"], "password": os.environ.get("VEC_CLAVE", "")}
    opciones.salida.mkdir(parents=True, exist_ok=True)
    fallos = 0

    with sync_playwright() as p:
        navegador = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
        for ancho in (int(a) for a in opciones.anchos.split(",")):
            alto = 900 if ancho >= 1000 else 844
            contexto = navegador.new_context(
                viewport={"width": ancho, "height": alto}, http_credentials=credencial, locale="es-ES"
            )
            if opciones.local:
                contexto.route("**/*", interceptor(opciones.local.resolve()))
            pagina = contexto.new_page()
            errores, red = [], []
            pagina.on("pageerror", lambda e: errores.append(str(e)[:160]))
            pagina.on(
                "response",
                lambda r: red.append(f"{r.status} {urlsplit(r.url).path[:70]}")
                if "/api/" in r.url and r.status >= 400
                else None,
            )
            for entrada in opciones.pantalla or PANTALLAS_RRHH:
                nombre, ruta = entrada.split("=", 1)
                ruta, _, accion = ruta.partition("@")
                errores.clear()
                red.clear()
                pagina.goto(base + ruta, wait_until="load")
                pagina.wait_for_timeout(ESPERA_MS)
                if accion:
                    pulsar(pagina, accion)
                desborde = pagina.evaluate(
                    "() => document.documentElement.scrollWidth - document.documentElement.clientWidth"
                )
                pagina.screenshot(
                    path=str(opciones.salida / f"{nombre}_{ancho}.png"), full_page=opciones.pagina_completa
                )
                problemas = errores + red + ([f"desborde horizontal {desborde}px"] if desborde > 0 else [])
                fallos += bool(problemas)
                print(f"{'FALLO' if problemas else 'ok   '} {nombre}_{ancho}", *problemas, sep="\n  ")
            contexto.close()
        navegador.close()
    sys.exit(1 if fallos else 0)


if __name__ == "__main__":
    main()
