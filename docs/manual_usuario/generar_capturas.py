"""Captura la presentación o exporta el manual canónico sin visitar el portal."""
import argparse
import html
import os
import pathlib
import time

from playwright.sync_api import sync_playwright

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--solo-exportar", action="store_true",
                    help="Generar HTML y PDF desde Markdown, sin capturas ni portal.")
parser.add_argument("--directorio-manual", type=pathlib.Path,
                    default=pathlib.Path(__file__).resolve().parent)
parser.add_argument("--base-enlaces", default=(
    "https://github.com/aavidad/VEC_Diputacion_app/blob/"
    "integracion/ct-producto-ligero-20260821/docs/manual_usuario/"
), help="Base pública para los enlaces del PDF; no se visita al exportar.")
argumentos = parser.parse_args()
DIRECTORIO_MANUAL = argumentos.directorio_manual.resolve()
BASE = os.environ.get(
    "VEC_PORTAL_BASE_URL",
    "http://127.0.0.1:18084/portal-empleado/?presentacion=rrhh&perfil=administrador",
)
DESTINO = pathlib.Path(
    os.environ.get("VEC_CAPTURAS_DESTINO", DIRECTORIO_MANUAL / "capturas")
).resolve()
if not argumentos.solo_exportar:
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


def exportar_manual(navegador):
    # Conversor instalado: no se descarga ni se mantiene otro texto del manual.
    try:
        from markdown_it import MarkdownIt
    except ImportError as error:
        raise SystemExit("Falta markdown-it: exportación no realizada.") from error

    markdown = (DIRECTORIO_MANUAL / "manual_portal_bolsas.md").read_text(encoding="utf-8")
    titulo = next(linea[2:] for linea in markdown.splitlines() if linea.startswith("# "))
    cuerpo = MarkdownIt("commonmark", {"html": False}).enable("table").render(markdown)
    estilo = """
body { font-family: "DejaVu Sans", Arial, sans-serif; color: #1a2733;
  max-width: 210mm; margin: 0 auto; padding: 12mm; box-sizing: border-box;
  line-height: 1.45; font-size: 10pt; overflow-wrap: anywhere; }
h1 { color: #0d2d52; font-size: 21pt; border-bottom: 3px solid #0d2d52; }
h2 { color: #0d2d52; margin-top: 22px; border-bottom: 1px solid #c9d6e4; }
h3 { color: #17466e; }
h1, h2, h3 { break-after: avoid; }
table { border-collapse: collapse; width: 100%; font-size: 9pt; margin: 12px 0; }
th, td { border: 1px solid #c9d6e4; padding: 6px; text-align: left; vertical-align: top; }
th { background: #eef3f8; }
thead { display: table-header-group; }
tr { break-inside: avoid; }
code { font-size: .95em; background: #eef3f8; overflow-wrap: anywhere; }
a { color: #17466e; }
li { margin: 4px 0; }
@media print { body { padding: 0; } }
"""
    documento = ("<!doctype html>\n<html lang=\"es\"><head>\n"
                 "<meta charset=\"utf-8\"><meta name=\"viewport\" "
                 "content=\"width=device-width, initial-scale=1\">\n"
                 f"<title>{html.escape(titulo)}</title>\n<style>{estilo}</style>\n"
                 "</head><body><main>\n" + cuerpo + "</main></body></html>\n")
    manual_html = DIRECTORIO_MANUAL / "manual_portal_bolsas.html"
    manual_pdf = DIRECTORIO_MANUAL / "manual_portal_bolsas.pdf"
    manual_html.write_text(documento, encoding="utf-8")

    pagina_pdf = navegador.new_page()
    # El documento se carga en memoria: no visita enlaces, portal ni imágenes.
    pagina_pdf.route("**/*", lambda ruta: ruta.abort())
    base = html.escape(argumentos.base_enlaces, quote=True)
    pagina_pdf.set_content(documento.replace("<head>", f'<head><base href="{base}">', 1))
    pagina_pdf.pdf(
        path=str(manual_pdf), format="A4", print_background=True,
        tagged=True, outline=True,
        margin={"top": "12mm", "right": "12mm", "bottom": "12mm", "left": "12mm"},
    )
    pagina_pdf.close()
    print(f"generados {manual_html.name} y {manual_pdf.name} desde Markdown")


with sync_playwright() as p:
    entorno_navegador = dict(os.environ)
    entorno_navegador.pop("SSLKEYLOGFILE", None)
    navegador = p.chromium.launch(channel="chrome", env=entorno_navegador)
    if argumentos.solo_exportar:
        exportar_manual(navegador)
        navegador.close()
        raise SystemExit(0)
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

    # Las exportaciones siempre derivan del Markdown, no de un HTML mantenido aparte.
    exportar_manual(navegador)

    navegador.close()
    print("RECORRIDO COMPLETO")
