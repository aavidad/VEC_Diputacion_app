#!/usr/bin/env python3
"""Captura de pantallas de diseño de Área Personal para antes/después (G19)."""

import argparse
import pathlib
import time
from PIL import Image
from playwright.sync_api import sync_playwright

URL_BASE = "http://127.0.0.1:8082/area-personal/"
DESTINO = pathlib.Path(__file__).resolve().parents[1] / "docs" / "manual" / "capturas" / "diseno"
RAIZ_WEB = pathlib.Path(__file__).resolve().parents[1] / "web" / "static"


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


def esperar_vista(pagina):
    pagina.wait_for_load_state("networkidle")
    for _ in range(20):
        texto = pagina.inner_text("#titulo-vista")
        if "Cargando" not in texto:
            break
        time.sleep(0.5)
    time.sleep(1)


def capturar(fase: str):
    DESTINO.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as p:
        browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
        context = browser.new_context(viewport={"width": 1440, "height": 1000})
        page = context.new_page()

        def interceptar_archivos(route):
            url = route.request.url
            if "/area-personal/" in url:
                rel = url.split("/area-personal/")[1].split("?")[0]
                if rel == "" or rel == "index.html":
                    route.fulfill(status=200, content_type="text/html; charset=utf-8", body=(RAIZ_WEB / "area-personal" / "index.html").read_bytes())
                    return
                if fase == "antes" and rel == "area-personal.css":
                    # Servir el CSS desplegado en el servidor
                    route.continue_()
                    return
                local = RAIZ_WEB / "area-personal" / rel
                if local.is_file():
                    content_type = "text/javascript" if rel.endswith(".js") else ("text/css" if rel.endswith(".css") else "application/octet-stream")
                    route.fulfill(status=200, content_type=content_type, body=local.read_bytes())
                    return
            route.continue_()

        page.route("**/*", interceptar_archivos)

        # 09 Área Personal - Inicio
        print(f"Capturando 09_area_personal_inicio_{fase}...")
        page.goto(f"{URL_BASE}?vista=inicio", wait_until="networkidle")
        esperar_vista(page)
        ruta_ini = DESTINO / f"09_area_personal_inicio_{fase}.png"
        page.screenshot(path=str(ruta_ini))
        optimizar_png(ruta_ini)

        # 10 Área Personal - Perfil
        print(f"Capturando 10_area_personal_perfil_{fase}...")
        page.goto(f"{URL_BASE}?vista=perfil", wait_until="networkidle")
        esperar_vista(page)
        ruta_per = DESTINO / f"10_area_personal_perfil_{fase}.png"
        page.screenshot(path=str(ruta_per))
        optimizar_png(ruta_per)

        browser.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--fase", choices=["antes", "despues"], default="despues")
    args = parser.parse_args()
    capturar(args.fase)
