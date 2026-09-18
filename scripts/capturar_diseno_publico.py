#!/usr/bin/env python3
"""Captura de pantallas de diseño de Portal Público (/bolsa/ y /bolsa/listas.html) para antes/después (G19)."""

import argparse
import json
import pathlib
import time
import urllib.request
from urllib.parse import unquote
from PIL import Image
from playwright.sync_api import sync_playwright

URL_BOLSA = "http://127.0.0.1:8082/bolsa/"
URL_LISTAS = "http://127.0.0.1:8082/bolsa/listas.html"
DESTINO = pathlib.Path(__file__).resolve().parents[1] / "docs" / "manual" / "capturas" / "diseno"
RAIZ_REPO = pathlib.Path(__file__).resolve().parents[1]
RAIZ_WEB = RAIZ_REPO / "web" / "static"


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


def mapear_estado_bolsa(estado_origen):
    if estado_origen in ("trabajando", "pendiente_incorporacion"):
        return "ocupado"
    if estado_origen == "disponible_desde":
        return "no_disponible"
    if estado_origen == "renuncia":
        return "renuncia_pendiente"
    if estado_origen in ("disponible", "no_disponible", "excluido"):
        return estado_origen
    return "disponible"


def cargar_dataset_bolsas():
    ruta = RAIZ_REPO / "data" / "demo" / "bolsa" / "v1.bolsas-demo.json"
    if ruta.is_file():
        return json.loads(ruta.read_text(encoding="utf-8"))
    return {"bolsas": [], "candidaturas": []}


def generar_fixture_bolsas(dataset):
    bolsas = []
    for b in dataset.get("bolsas", []):
        bolsas.append({
            "bolsa_ref": b.get("bolsa_ref", "").replace(":demo:", ":sintetico:"),
            "categoria": b.get("categoria", ""),
            "grupos": b.get("grupos", ["C1"]),
            "tipo_lista": b.get("tipo_lista", "ordinaria"),
            "vigente_desde": f"{b.get('vigente_desde', '2025-01-01')}T00:00:00Z",
            "vigente_hasta": f"{b.get('vigente_hasta')}T00:00:00Z" if b.get("vigente_hasta") else None,
            "total": b.get("candidaturas", 0),
        })
    return {
        "data": {
            "esquema": "vec.bolsa.publico.bolsas.v1",
            "generado_en": "2026-09-17T20:00:00Z",
            "bolsas": bolsas,
        }
    }


def generar_fixture_lista(dataset, bolsa_ref):
    bolsas = dataset.get("bolsas", [])
    bolsa_orig = next((b for b in bolsas if b.get("bolsa_ref", "").replace(":demo:", ":sintetico:") == bolsa_ref), None)
    if not bolsa_orig and bolsas:
        bolsa_orig = bolsas[0]

    bolsa = {
        "bolsa_ref": (bolsa_orig.get("bolsa_ref", "") if bolsa_orig else bolsa_ref).replace(":demo:", ":sintetico:"),
        "categoria": bolsa_orig.get("categoria", "Administrativo/a") if bolsa_orig else "Administrativo/a",
        "grupos": bolsa_orig.get("grupos", ["C1"]) if bolsa_orig else ["C1"],
        "tipo_lista": bolsa_orig.get("tipo_lista", "ordinaria") if bolsa_orig else "ordinaria",
        "vigente_desde": f"{bolsa_orig.get('vigente_desde', '2025-01-01')}T00:00:00Z" if bolsa_orig else "2025-01-01T00:00:00Z",
        "vigente_hasta": None,
        "total": bolsa_orig.get("candidaturas", 15) if bolsa_orig else 15,
    }

    candidaturas = [c for c in dataset.get("candidaturas", []) if c.get("bolsa_ref") == (bolsa_orig.get("bolsa_ref") if bolsa_orig else "")]
    if not candidaturas:
        estados = ["disponible", "ocupado", "no_disponible", "renuncia_pendiente", "excluido"]
        candidaturas = [{"documento_enmascarado": f"***{1000 + i * 111}**", "estado_clave": estados[i % len(estados)]} for i in range(12)]

    posiciones = []
    for idx, c in enumerate(candidaturas[:15]):
        posiciones.append({
            "orden": idx + 1,
            "documento_enmascarado": c.get("documento_enmascarado", f"***{1000 + idx}**"),
            "estado_clave": mapear_estado_bolsa(c.get("estado_clave", "disponible")),
        })

    return {
        "data": {
            "esquema": "vec.bolsa.publico.lista.v1",
            "generado_en": "2026-09-17T20:00:00Z",
            "bolsa": bolsa,
            "posiciones": posiciones,
            "hay_mas": False,
            "cursor_siguiente": None,
        }
    }


def convertir_convocatorias_v2(v1_bytes):
    v1 = json.loads(v1_bytes.decode("utf-8"))
    HUELLA_A = "a" * 64
    HUELLA_PROY = "b" * 64
    snapshot = {
        "catalogo_id": "categorias-profesionales",
        "version": 1,
        "huella_sha256": HUELLA_A,
        "huella_proyeccion_sha256": HUELLA_PROY,
    }

    dicc_map = {}
    convocatorias_v2 = []

    for c in v1.get("convocatorias", []):
        cats = []
        for cat_item in c.get("categorias", []):
            clave = cat_item if isinstance(cat_item, str) else cat_item.get("clave", "auxiliar-administrativo")
            etiqueta = clave.replace("-", " ").capitalize()
            if clave not in dicc_map:
                dicc_map[clave] = {
                    "clave": clave,
                    "version": 1,
                    "etiqueta": etiqueta,
                    "semantica": "informacion",
                    "catalogo_categorias": snapshot,
                }
            cats.append({"clave": clave, "version": 1})

        c_copia = dict(c)
        c_copia["catalogo_categorias"] = snapshot
        c_copia["categorias"] = cats
        convocatorias_v2.append(c_copia)

    diccionario = list(dicc_map.values())

    v2 = {
        "esquema": "vec.bolsa.publico.convocatorias.v2",
        "fuente": v1.get("fuente", {"revision": "2026-07", "actualizada_en": "2026-07-16T00:00:00Z", "demostracion": True, "aviso": "Demo"}),
        "facetas": {
            "tipos": v1.get("facetas", {}).get("tipos", []),
            "categorias": [{"clave": d["clave"], "version": 1, "etiqueta": d["etiqueta"], "semantica": "informacion", "numero_resultados": 1} for d in diccionario],
            "estados": v1.get("facetas", {}).get("estados", []),
        },
        "diccionario_categorias": diccionario,
        "paginacion": v1.get("paginacion", {"pagina": 1, "tamano": 12, "total": len(convocatorias_v2), "paginas": 1}),
        "convocatorias": convocatorias_v2,
    }
    return json.dumps(v2).encode("utf-8")


def esperar_carga(pagina, segundos=3):
    pagina.wait_for_load_state("load")
    time.sleep(segundos)


def capturar(fase: str):
    DESTINO.mkdir(parents=True, exist_ok=True)
    dataset = cargar_dataset_bolsas()
    fixture_bolsas = generar_fixture_bolsas(dataset)

    # Cache convocatorias V2
    try:
        req = urllib.request.urlopen("http://127.0.0.1:8082/api/publico/bolsa/convocatorias?pagina=1&tamano=12")
        convocatorias_v2_bytes = convertir_convocatorias_v2(req.read())
    except Exception as e:
        print("Aviso al obtener convocatorias remotas:", e)
        convocatorias_v2_bytes = None

    with sync_playwright() as p:
        browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
        context = browser.new_context(viewport={"width": 1440, "height": 1000})
        page = context.new_page()

        def interceptar(route):
            url = route.request.url

            # Servir V2 para convocatorias
            if "/api/publico/bolsa/convocatorias" in url and convocatorias_v2_bytes:
                route.fulfill(
                    status=200,
                    content_type="application/json; charset=utf-8",
                    body=convocatorias_v2_bytes,
                )
                return

            # Interceptar API de bolsas públicas
            if "/api/publico/bolsa/bolsas" in url:
                if "/lista" in url:
                    partes = url.split("/api/publico/bolsa/bolsas/")[1].split("/lista")[0]
                    b_ref = unquote(partes)
                    route.fulfill(
                        status=200,
                        content_type="application/json; charset=utf-8",
                        body=json.dumps(generar_fixture_lista(dataset, b_ref)).encode("utf-8"),
                    )
                    return
                route.fulfill(
                    status=200,
                    content_type="application/json; charset=utf-8",
                    body=json.dumps(fixture_bolsas).encode("utf-8"),
                )
                return

            # Si es fase antes y pide CSS, usar el CSS del servidor (remoto)
            if fase == "antes":
                if "/bolsa/listas.css" in url or "/bolsa/bolsa.css" in url:
                    route.continue_()
                    return

            # Si es fase después, servir siempre CSS y JS locales
            if fase == "despues":
                if "/bolsa/" in url:
                    rel = url.split("/bolsa/")[1].split("?")[0]
                    local = RAIZ_WEB / "bolsa" / rel
                    if local.is_file():
                        if rel.endswith(".html"):
                            ct = "text/html; charset=utf-8"
                        elif rel.endswith(".css"):
                            ct = "text/css; charset=utf-8"
                        elif rel.endswith(".js"):
                            ct = "text/javascript; charset=utf-8"
                        else:
                            ct = "application/octet-stream"
                        route.fulfill(status=200, content_type=ct, body=local.read_bytes())
                        return

            route.continue_()

        page.route("**/*", interceptar)

        # 11 Convocatorias públicas (/bolsa/)
        print(f"Capturando 11_publico_convocatorias_{fase}...")
        page.goto(URL_BOLSA, wait_until="load")
        esperar_carga(page, 4)
        ruta_conv = DESTINO / f"11_publico_convocatorias_{fase}.png"
        page.screenshot(path=str(ruta_conv), full_page=False)
        optimizar_png(ruta_conv)

        # 12 Listas de aspirantes (/bolsa/listas.html)
        print(f"Capturando 12_publico_listas_{fase}...")
        page.goto(URL_LISTAS, wait_until="load")
        esperar_carga(page, 3)
        btn_primera = page.locator("button[data-accion='ver-bolsa']").first
        if btn_primera.count() > 0:
            btn_primera.click()
            esperar_carga(page, 2)
        ruta_listas = DESTINO / f"12_publico_listas_{fase}.png"
        page.screenshot(path=str(ruta_listas), full_page=False)
        optimizar_png(ruta_listas)

        browser.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--fase", choices=["antes", "despues"], default="despues")
    args = parser.parse_args()
    capturar(args.fase)
