"""Recorre el Portal del Empleado en modo presentacion y captura cada vista."""
import os
import pathlib
import time

from playwright.sync_api import sync_playwright

DIRECTORIO_MANUAL = pathlib.Path(__file__).resolve().parent
BASE = os.environ.get(
    "VEC_PORTAL_BASE_URL",
    "http://127.0.0.1:18084/portal-empleado/?presentacion=rrhh",
)
DESTINO = pathlib.Path(
    os.environ.get("VEC_CAPTURAS_DESTINO", DIRECTORIO_MANUAL / "capturas")
).resolve()
DESTINO.mkdir(parents=True, exist_ok=True)

VISTAS = [
    ("portal", "01_portada"),
    ("bolsa/resumen", "02_cuadro_mando"),
    ("bolsa/elaboracion", "03_elaboracion"),
    ("bolsa/contratos", "05_contratos"),
    ("bolsa/reglas", "06_reglas"),
    ("bolsa/consulta", "07_consulta"),
    ("bolsa/estadisticas", "08_estadisticas"),
    ("bolsa/documentos", "09_documentos"),
    ("bolsa/comunicaciones", "10_comunicaciones"),
    ("bolsa/auditoria", "11_auditoria"),
]


def esperar(pagina):
    pagina.wait_for_load_state("networkidle")
    time.sleep(0.6)


def capturar(pagina, nombre, completa=True):
    ruta = DESTINO / f"{nombre}.png"
    pagina.screenshot(path=str(ruta), full_page=completa)
    print(f"capturada {ruta.name}")


def abrir_dialogo_y_capturar(pagina, accion, nombre):
    pagina.locator(f'[data-accion="{accion}"]').first.click()
    pagina.locator("#dialogo-detalle[open]").wait_for(state="visible")
    capturar(pagina, nombre, completa=False)
    pagina.locator("#dialogo-detalle .boton-cerrar").click()
    pagina.locator("#dialogo-detalle").wait_for(state="hidden")


with sync_playwright() as p:
    navegador = p.chromium.launch(channel="chrome")
    pagina = navegador.new_page(viewport={"width": 1440, "height": 900}, device_scale_factor=2)
    pagina.goto(BASE + "#portal")
    esperar(pagina)

    for hash_vista, nombre in VISTAS:
        pagina.goto(f"{BASE}#{hash_vista}")
        esperar(pagina)
        capturar(pagina, nombre)

    # Detalle de elaboración y configuración de bases.
    pagina.goto(f"{BASE}#bolsa/elaboracion")
    esperar(pagina)
    pagina.locator('[data-accion="seleccionar-elaboracion"]').first.click()
    esperar(pagina)
    capturar(pagina, "03b_detalle_expediente")
    abrir_dialogo_y_capturar(
        pagina,
        "configurar-bases",
        "03c_configurar_bases",
    )

    # Asistente de llamamientos paso a paso
    pagina.goto(f"{BASE}#bolsa/llamamientos")
    esperar(pagina)
    pagina.locator('[data-accion="seleccionar-necesidad"]').first.click()
    esperar(pagina)
    capturar(pagina, "04a_llamamiento_paso1")

    # La propuesta la calcula el servidor. La presentación usa su contrato
    # sintético aislado; el navegador nunca marca ni ordena personas.
    pagina.locator('[data-accion="solicitar-propuesta"]').click()
    pagina.locator('[data-accion="siguiente-paso"]').wait_for(state="visible")
    esperar(pagina)
    capturar(pagina, "04b_llamamiento_paso2")
    pagina.locator('[data-accion="siguiente-paso"]').click()
    esperar(pagina)
    capturar(pagina, "04c_llamamiento_paso3")
    pagina.locator('[data-accion="siguiente-paso"]').click()
    esperar(pagina)
    capturar(pagina, "04d_llamamiento_paso4")

    # Diálogos y preferencias accesibles.
    pagina.goto(f"{BASE}#bolsa/reglas")
    esperar(pagina)
    abrir_dialogo_y_capturar(pagina, "detalle-regla", "06b_detalle_regla")
    abrir_dialogo_y_capturar(pagina, "avisos", "12_avisos")
    abrir_dialogo_y_capturar(pagina, "ayuda", "13_ayuda")
    pagina.locator("#boton-contraste").click()
    esperar(pagina)
    capturar(pagina, "14_alto_contraste")

    # El PDF usa el HTML versionado y las capturas recién generadas. A4 es el
    # formato administrativo esperado; no se incrustan rutas del puesto local.
    manual_html = DIRECTORIO_MANUAL / "manual_portal_bolsas.html"
    manual_pdf = DIRECTORIO_MANUAL / "manual_portal_bolsas.pdf"
    pagina_pdf = navegador.new_page()
    pagina_pdf.goto(manual_html.as_uri(), wait_until="networkidle")
    pagina_pdf.pdf(
        path=str(manual_pdf),
        format="A4",
        print_background=True,
        margin={"top": "12mm", "right": "12mm", "bottom": "12mm", "left": "12mm"},
    )
    print(f"generado {manual_pdf.name}")

    navegador.close()
    print("RECORRIDO COMPLETO")
