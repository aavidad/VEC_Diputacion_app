#!/usr/bin/env python3
"""Captura de pantallas de diseño de Portal RRHH para antes/después (G19)."""

import argparse
import pathlib
import time
from PIL import Image
from playwright.sync_api import sync_playwright

URL_BASE = "http://127.0.0.1:8082/portal-empleado/"
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


def esperar_carga(pagina, segundos=4):
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

        # 01 Inicio
        print(f"Capturando 01_inicio_{fase}...")
        page.goto(f"{URL_BASE}#portal", wait_until="load")
        esperar_carga(page, 4)
        ruta = DESTINO / f"01_inicio_{fase}.png"
        page.screenshot(path=str(ruta))
        optimizar_png(ruta)

        # 02 Cuadro de mando
        print(f"Capturando 02_cuadro_{fase}...")
        btn_cuadro = page.locator('button[data-vista="contratacion-temporal"][data-ct-exp-vista="cuadro"]').first
        if btn_cuadro.count() > 0:
            btn_cuadro.click()
        else:
            page.goto(f"{URL_BASE}#contratacion-temporal", wait_until="load")
        esperar_carga(page, 4)
        ruta = DESTINO / f"02_cuadro_{fase}.png"
        page.screenshot(path=str(ruta))
        optimizar_png(ruta)

        # 04 Alta
        print(f"Capturando 04_alta_{fase}...")
        btn_alta = page.locator('button[data-ct-exp-vista="alta"]').first
        if btn_alta.count() > 0:
            btn_alta.click()
            time.sleep(2)
        ruta = DESTINO / f"04_alta_{fase}.png"
        page.screenshot(path=str(ruta))
        optimizar_png(ruta)

        # 03 Expediente (detalle con campos y raíl)
        print(f"Capturando 03_expediente_{fase}...")
        btn_volver_cuadro = page.locator('button[data-ct-exp-vista="cuadro"]').first
        if btn_volver_cuadro.count() > 0:
            btn_volver_cuadro.click()
            time.sleep(2)
        btn_exp = page.locator('tr:has-text("000010") button, tr:has-text("000009") button, .ct-exp-listado table tbody tr button').first
        if btn_exp.count() > 0:
            btn_exp.click()
            time.sleep(3)
        ruta = DESTINO / f"03_expediente_{fase}.png"
        page.screenshot(path=str(ruta))
        optimizar_png(ruta)

        # 05 Documentos
        print(f"Capturando 05_documentos_{fase}...")
        btn_doc = page.locator('button[data-ct-exp-vista="documentos"]').first
        if btn_doc.count() > 0:
            btn_doc.click()
            time.sleep(2)
        else:
            page.goto(f"{URL_BASE}#documentos", wait_until="load")
            esperar_carga(page, 3)
        ruta = DESTINO / f"05_documentos_{fase}.png"
        page.screenshot(path=str(ruta))
        optimizar_png(ruta)

        # 06 Auditoría
        print(f"Capturando 06_auditoria_{fase}...")
        btn_aud = page.locator('button[data-ct-exp-vista="auditoria"]').first
        if btn_aud.count() > 0:
            btn_aud.click()
            time.sleep(2)
        else:
            page.goto(f"{URL_BASE}#auditoria", wait_until="load")
            esperar_carga(page, 3)
        ruta = DESTINO / f"06_auditoria_{fase}.png"
        page.screenshot(path=str(ruta))
        optimizar_png(ruta)

        browser.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--fase", choices=["antes", "despues"], default="despues")
    args = parser.parse_args()
    capturar(args.fase)
