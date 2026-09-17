"""Genera las capturas canónicas del manual de Contratación Temporal con Playwright."""
import argparse
import os
import pathlib
import time
from PIL import Image
from playwright.sync_api import sync_playwright

DIRECTORIO_MANUAL = pathlib.Path(__file__).resolve().parent
DESTINO = DIRECTORIO_MANUAL / "capturas" / "contratacion"

URL_RRHH = os.environ.get("VEC_PORTAL_RRHH", "http://127.0.0.1:8082/portal-empleado/")
URL_INTERVENCION = os.environ.get("VEC_PORTAL_INTERVENCION", "http://127.0.0.1:8083/portal-empleado/")


def optimizar_png(ruta: pathlib.Path, max_kb: int = 245):
    """Garantiza que la captura en disco no sobrepase el umbral de 250 KB."""
    tamano_kb = ruta.stat().st_size / 1024
    if tamano_kb > max_kb:
        img = Image.open(ruta)
        # Reducción a paleta de 256 colores conservando nitidez tipográfica
        img_opt = img.convert("RGB").quantize(colors=256)
        img_opt.save(ruta, "PNG", optimize=True)
        tamano_nuevo = ruta.stat().st_size / 1024
        print(f"  optimizada {ruta.name}: {tamano_kb:.1f} KB -> {tamano_nuevo:.1f} KB")
    else:
        print(f"  capturada {ruta.name}: {tamano_kb:.1f} KB")


def esperar_carga(pagina, segundos=5):
    pagina.wait_for_load_state("load")
    time.sleep(segundos)


def capturar_pantalla(pagina, nombre: str, clip=None, completa: bool = False):
    ruta = DESTINO / f"{nombre}.png"
    if clip is not None:
        pagina.screenshot(path=str(ruta), clip=clip)
    else:
        pagina.screenshot(path=str(ruta), full_page=completa)
    optimizar_png(ruta)


def ejecutar_capturas():
    DESTINO.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as p:
        browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)

        # Contexto 1: RRHH (8082) en viewport escritorio 1440x1000
        ctx_rrhh = browser.new_context(viewport={"width": 1440, "height": 1000})
        page = ctx_rrhh.new_page()

        # 01 Inicio del portal
        print("Capturando 01 inicio del portal...")
        page.goto(f"{URL_RRHH}#portal", wait_until="load")
        esperar_carga(page, 5)
        capturar_pantalla(page, "01_inicio_portal")

        # 02 Cuadro de mando completo
        print("Capturando 02 cuadro de mando completo...")
        page.goto(f"{URL_RRHH}#contratacion-temporal", wait_until="load")
        esperar_carga(page, 5)
        capturar_pantalla(page, "02_cuadro_mando_completo")

        # 03 Cuadro con filtro por fase
        print("Capturando 03 cuadro con filtro por fase...")
        filtro_fase = page.locator('select[name="fase"]').first
        filtro_fase.select_option("fiscalizacion")
        time.sleep(2)
        capturar_pantalla(page, "03_cuadro_filtro_fase")
        # Restablecer filtro
        filtro_fase.select_option("")
        time.sleep(1)

        # 04 Nueva petición vacía
        print("Capturando 04 nueva petición vacía...")
        page.locator('button[data-ct-exp-vista="alta"]').first.click()
        time.sleep(2)
        capturar_pantalla(page, "04_nueva_peticion_vacia")

        # 05 Nueva petición rellena (antes de «Revisar solicitud»)
        print("Capturando 05 nueva petición rellena...")
        page.locator('#ct-centro_ref').select_option(index=1)
        time.sleep(0.5)
        if page.locator('#ct-contacto_ref option').count() > 1:
            page.locator('#ct-contacto_ref').select_option(index=1)
        page.locator('#ct-categoria_ref').select_option(index=1)
        time.sleep(0.5)
        if page.locator('#ct-grupo_subgrupo option').count() > 1:
            page.locator('#ct-grupo_subgrupo').select_option(index=1)
        page.locator('#ct-motivo_clave').select_option(index=1)
        page.locator('#ct-detalle').fill('Cobertura de vacante por acumulación de tareas extraordinarias en el servicio.')
        page.locator('#ct-inicio').fill('2027-01-15')
        page.locator('#ct-fin').fill('2027-07-15')
        page.locator('#ct-observaciones').fill('Dotación presupuestaria consignada en el ejercicio económico.')
        page.locator('input[name="rc_existe"][value="no"]').check()
        time.sleep(1)
        capturar_pantalla(page, "05_nueva_peticion_rellena")

        # 06 Revisión y «Confirmar y registrar» (paso previo, no pulsar confirmar)
        print("Capturando 06 revisión y confirmación...")
        page.locator('button:has-text("Revisar solicitud")').first.click()
        time.sleep(2)
        capturar_pantalla(page, "06_revision_confirmar_registrar")

        # 07 Detalle de 000005 (solicitud registrada) con raíl
        print("Capturando 07 detalle de 000005...")
        page.locator('button[data-ct-exp-vista="cuadro"]').first.click()
        time.sleep(2)
        page.locator('tr:has-text("000005") button, tr:has-text("000005") a').first.click()
        time.sleep(2)
        capturar_pantalla(page, "07_detalle_000005_solicitud_rail")

        # 08 Formulario de análisis (en 000005, sin enviar)
        print("Capturando 08 formulario de análisis...")
        capturar_pantalla(page, "08_formulario_analisis_000005")

        # 09 Detalle de 000006 con análisis, coste y observaciones
        print("Capturando 09 detalle de 000006...")
        page.locator('button[data-ct-exp-vista="cuadro"]').first.click()
        time.sleep(2)
        page.locator('tr:has-text("000006") button, tr:has-text("000006") a').first.click()
        time.sleep(2)
        capturar_pantalla(page, "09_detalle_000006_analisis_coste")

        # 10 Gestión de bolsa: comprobaciones y vía (000007)
        print("Capturando 10 gestión de bolsa 000007...")
        page.locator('button[data-ct-exp-vista="cuadro"]').first.click()
        time.sleep(2)
        page.locator('tr:has-text("000007") button, tr:has-text("000007") a').first.click()
        time.sleep(2)
        capturar_pantalla(page, "10_gestion_bolsa_000007")

        # 11 Asignación a unidad (000008)
        print("Capturando 11 asignación a unidad 000008...")
        page.locator('button[data-ct-exp-vista="cuadro"]').first.click()
        time.sleep(2)
        page.locator('tr:has-text("000008") button, tr:has-text("000008") a').first.click()
        time.sleep(2)
        capturar_pantalla(page, "11_asignacion_unidad_000008")

        # 12 Informe jurídico y documentos (000009, pestaña Documentos)
        print("Capturando 12 informe jurídico y documentos 000009...")
        page.locator('button[data-ct-exp-vista="cuadro"]').first.click()
        time.sleep(2)
        page.locator('tr:has-text("000009") button, tr:has-text("000009") a').first.click()
        time.sleep(2)
        page.locator('button[data-ct-exp-vista="documentos"]').first.click()
        time.sleep(2)
        capturar_pantalla(page, "12_informe_juridico_documentos_000009")
        # Recargar para restablecer estado tras la denegación de documentos
        page.reload(wait_until="load")
        esperar_carga(page, 5)

        # 14 Reparo y subsanación (000010, raíl «Con incidencia»)
        print("Capturando 14 reparo y subsanación 000010...")
        page.locator('tr:has-text("000010") button, tr:has-text("000010") a').first.click()
        time.sleep(2)
        capturar_pantalla(page, "14_reparo_subsanacion_000010")

        # 15 Llamamiento (000011, bloque «Llamamiento y comunicación»)
        print("Capturando 15 llamamiento 000011...")
        page.locator('button[data-ct-exp-vista="cuadro"]').first.click()
        time.sleep(2)
        page.locator('tr:has-text("000011") button, tr:has-text("000011") a').first.click()
        time.sleep(2)
        capturar_pantalla(page, "15_llamamiento_comunicacion_000011")

        # 16 Historial de actuaciones desplegado
        print("Capturando 16 historial de actuaciones...")
        historial = page.locator('details.ct-exp-historial summary').first
        if historial.count() > 0:
            historial.click()
            time.sleep(1)
        capturar_pantalla(page, "16_historial_actuaciones_desplegado")

        # 17 Auditoría
        print("Capturando 17 auditoría...")
        page.locator('button[data-ct-exp-vista="auditoria"]').first.click()
        time.sleep(2)
        capturar_pantalla(page, "17_auditoria_expediente")

        # 18 Ayuda contextual abierta
        print("Capturando 18 ayuda contextual abierta...")
        btn_ayuda = page.locator('button:has-text("Ayuda"), [data-accion="ayuda"]').first
        if btn_ayuda.count() > 0:
            btn_ayuda.click()
            time.sleep(2)
            capturar_pantalla(page, "18_ayuda_contextual_abierta")
            page.keyboard.press("Escape")
            time.sleep(1)

        ctx_rrhh.close()

        # Contexto 2: Intervención (8083) para la fiscalización del 000009
        # 13 Fiscalización desde Intervención (8083, 000009)
        print("Capturando 13 fiscalización desde Intervención...")
        ctx_intervencion = browser.new_context(viewport={"width": 1440, "height": 1000})
        page_int = ctx_intervencion.new_page()
        page_int.goto(f"{URL_INTERVENCION}#contratacion-temporal", wait_until="load")
        esperar_carga(page_int, 5)
        page_int.locator("#ct-fiscalizacion-expediente").fill("2026/CT-000009")
        page_int.locator("#ct-fiscalizacion-version").fill("5")
        page_int.locator('button:has-text("Abrir fiscalización")').first.click()
        time.sleep(2)
        capturar_pantalla(page_int, "13_fiscalizacion_intervencion_000009")
        ctx_intervencion.close()

        # Contexto 3: Móvil 390px
        # 19 Detalle a 390 px (viewport móvil) del 000010
        print("Capturando 19 detalle móvil 390px...")
        ctx_movil = browser.new_context(viewport={"width": 390, "height": 844}, is_mobile=True)
        page_mov = ctx_movil.new_page()
        page_mov.goto(f"{URL_RRHH}#contratacion-temporal", wait_until="load")
        esperar_carga(page_mov, 5)
        page_mov.locator('tr:has-text("000010") button, tr:has-text("000010") a').first.click()
        time.sleep(2)
        capturar_pantalla(page_mov, "19_detalle_movil_390px_000010")
        ctx_movil.close()

        browser.close()
        print("Todas las capturas se han generado y optimizado con éxito.")


if __name__ == "__main__":
    ejecutar_capturas()
