"""Recorre el Portal del Empleado en modo presentacion y captura cada vista."""
import pathlib
import sys
import time

from playwright.sync_api import sync_playwright

BASE = "http://127.0.0.1:18084/portal-empleado/?presentacion=rrhh"
DESTINO = pathlib.Path("/home/alberto/Trabajo/VEC_Diputacion_app/docs/manual_usuario/capturas")
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


with sync_playwright() as p:
    navegador = p.chromium.launch(channel="chrome")
    pagina = navegador.new_page(viewport={"width": 1440, "height": 900}, device_scale_factor=2)
    pagina.goto(BASE + "#portal")
    esperar(pagina)

    for hash_vista, nombre in VISTAS:
        pagina.goto(f"{BASE}#{hash_vista}")
        esperar(pagina)
        capturar(pagina, nombre)

    # Asistente de llamamientos paso a paso
    pagina.goto(f"{BASE}#bolsa/llamamientos")
    esperar(pagina)
    capturar(pagina, "04a_llamamiento_paso1")

    # Enumerar botones visibles para avanzar por el asistente
    for paso in range(2, 6):
        botones = pagina.locator("button:visible").all_inner_texts()
        avance = None
        for texto in botones:
            limpio = texto.strip().lower()
            if any(clave in limpio for clave in ("siguiente", "continuar", "revisar", "confirmar")):
                avance = texto.strip()
                break
        if not avance:
            print(f"paso {paso}: sin boton de avance; botones={botones[:12]}")
            break
        # En el paso de candidatos, marcar algunos antes de avanzar
        casillas = pagina.locator("input[type=checkbox]:visible")
        if casillas.count() > 0 and paso == 2:
            for i in range(min(3, casillas.count())):
                casillas.nth(i).check()
            time.sleep(0.3)
            capturar(pagina, "04a_llamamiento_paso1_seleccion")
        pagina.get_by_role("button", name=avance).first.click()
        time.sleep(0.8)
        capturar(pagina, f"04{chr(96 + paso)}_llamamiento_paso{paso}")

    navegador.close()
    print("RECORRIDO COMPLETO")
