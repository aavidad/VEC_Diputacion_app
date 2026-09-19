#!/usr/bin/env python3
"""Captura de pantallas para el Manual de Usuario G18-a desde la principal."""

import pathlib
import subprocess
import time
from PIL import Image
from playwright.sync_api import sync_playwright

URL_BASE = "http://127.0.0.1:8082"
DESTINO = pathlib.Path(__file__).resolve().parents[1] / "docs" / "manual" / "capturas"


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


def ejecutar_capturas():
    DESTINO.mkdir(parents=True, exist_ok=True)
    print("Iniciando tunel temporal a 8082...")
    tunnel = subprocess.Popen(["ssh", "-N", "-L", "8082:127.0.0.1:8082", "root@cidonia.cloud"])
    try:
        time.sleep(2)
        with sync_playwright() as p:
            browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
            page = browser.new_page(viewport={"width": 1440, "height": 1000})

            # 1. B12 Cuadro de bolsas de trabajo
            print("Capturando 14_rrhh_bolsas_cuadro_b12.png...")
            page.goto(f"{URL_BASE}/portal-empleado/#resumen", wait_until="load")
            time.sleep(4)
            r14 = DESTINO / "14_rrhh_bolsas_cuadro_b12.png"
            page.screenshot(path=str(r14))
            optimizar_png(r14)

            # 2. B5 Lista de candidatos
            print("Capturando 15_rrhh_bolsas_candidatos_b5.png...")
            btn_ver = page.locator('table tbody tr button[data-accion="ver-bolsa"]').first
            btn_ver.click()
            time.sleep(3)
            r15 = DESTINO / "15_rrhh_bolsas_candidatos_b5.png"
            page.screenshot(path=str(r15))
            optimizar_png(r15)

            # 3. B5 Filtro / búsqueda
            print("Capturando 16_rrhh_bolsas_candidatos_filtrados.png...")
            input_busqueda = page.locator('#filtro-bolsa-texto')
            input_busqueda.fill("0071")
            time.sleep(2)
            r16 = DESTINO / "16_rrhh_bolsas_candidatos_filtrados.png"
            page.screenshot(path=str(r16))
            optimizar_png(r16)

            # 4. Estadísticas mensuales C18
            print("Capturando 17_contratacion_estadisticas_mensual.png...")
            page.goto(f"{URL_BASE}/portal-empleado/#contratacion-temporal", wait_until="load")
            time.sleep(3)
            btn_est = page.locator('button[data-ct-exp-vista="estadisticas"]')
            btn_est.click()
            time.sleep(3)
            r17 = DESTINO / "17_contratacion_estadisticas_mensual.png"
            page.screenshot(path=str(r17))
            optimizar_png(r17)

            # 5. Estadísticas semanales C18
            print("Capturando 18_contratacion_estadisticas_semanal.png...")
            select_periodo = page.locator('#filtro-est-periodo')
            select_periodo.select_option("semanal")
            page.locator('form[data-ct-form="filtros-estadisticas"] button[type="submit"]').click()
            time.sleep(3)
            r18 = DESTINO / "18_contratacion_estadisticas_semanal.png"
            page.screenshot(path=str(r18))
            optimizar_png(r18)

            browser.close()
    finally:
        print("Cerrando tunel temporal...")
        tunnel.terminate()
        tunnel.wait()
        print("Tunel cerrado exitosamente.")


if __name__ == "__main__":
    ejecutar_capturas()
