#!/usr/bin/env python3
"""Captura de pantallas de diseño de Peticiones del Centro y Organización para antes/después (G19)."""

import argparse
import pathlib
import time
from PIL import Image
from playwright.sync_api import sync_playwright

URL_PETICIONES = "http://127.0.0.1:8082/portal-empleado/peticiones-centro/?vista=rrhh"
URL_ORGANIZACION = "http://127.0.0.1:8082/portal-empleado/organizacion/"
DESTINO = pathlib.Path(__file__).resolve().parents[1] / "docs" / "manual" / "capturas" / "diseno"
RAIZ_WEB = pathlib.Path(__file__).resolve().parents[1] / "web" / "static" / "portal-empleado"


def optimizar_png(ruta: pathlib.Path, max_kb: int = 245):
    tamano_kb = ruta.stat().st_size / 1024
    if tamano_kb > max_kb:
        img = Image.open(ruta)
        img_opt = img.convert("RGB").quantize(colors=256)
        img_opt.save(ruta, "PNG", optimize=True)
        tamano_nuevo = ruta.stat().st_size / 1024
        print(f"  optimizada {ruta.name}: {tamano_kb:.1f} KB -> {tamano_nuevo:.1f} KB")
    else:
        print(f"  capturada {ruta.name}: {tamano_kb:.1f} KB")


def esperar_carga(pagina, segundos=3):
    pagina.wait_for_load_state("load")
    time.sleep(segundos)


def capturar(fase: str):
    DESTINO.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as p:
        browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
        context = browser.new_context(viewport={"width": 1440, "height": 1000})
        page = context.new_page()

        if fase == "despues":
            def interceptar_css(route):
                url = route.request.url
                if "/portal-empleado/" in url and ".css" in url:
                    ruta_relativa = url.split("/portal-empleado/")[1].split("?")[0]
                    ruta_local = RAIZ_WEB / ruta_relativa
                    if ruta_local.is_file():
                        route.fulfill(
                            status=200,
                            content_type="text/css",
                            body=ruta_local.read_bytes(),
                        )
                        return
                route.continue_()

            page.route("**/*.css*", interceptar_css)

        # 07 Peticiones Centro (vista RRHH)
        print(f"Capturando 07_peticiones_centro_{fase}...")
        page.goto(URL_PETICIONES, wait_until="load")
        esperar_carga(page, 3)
        ruta_pet = DESTINO / f"07_peticiones_centro_{fase}.png"
        page.screenshot(path=str(ruta_pet))
        optimizar_png(ruta_pet)

        # 08 Organización
        print(f"Capturando 08_organizacion_{fase}...")
        page.goto(URL_ORGANIZACION, wait_until="load")
        esperar_carga(page, 3)
        ruta_org = DESTINO / f"08_organizacion_{fase}.png"
        page.screenshot(path=str(ruta_org))
        optimizar_png(ruta_org)

        browser.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--fase", choices=["antes", "despues"], default="despues")
    args = parser.parse_args()
    capturar(args.fase)
