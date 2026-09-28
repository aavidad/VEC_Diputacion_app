#!/usr/bin/env python3
"""Recorrido mTLS sintético de tipo configurable, publicación y descarga CT.

El servidor, el expediente y las tres identidades V3 deben existir antes de usarlo.
La fase recuperar se ejecuta después de reiniciar aplicación y PostgreSQL fuera
de este programa. No instala SQL ni modifica una base conservada.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from pathlib import Path
from urllib.parse import quote, urlparse

RUTA_CATALOGO = "/api/vec/contratacion-temporal/plantillas"
RUTA_ENTRADAS = RUTA_CATALOGO + "/entradas"
RUTA_PUBLICAR = RUTA_CATALOGO + "/publicar"
RUTA_DISPONIBLES = "/api/vec/contratacion-temporal/expedientes/borradores/disponibles"
RUTA_DESCARGA = "/api/vec/contratacion-temporal/expedientes/borradores"


class Fallo(RuntimeError):
    pass


def argumentos():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("fase", choices=("alta", "recuperar"))
    p.add_argument("--origen", required=True, help="HTTPS loopback con mTLS y V3 reales")
    for papel in ("editor", "publicador", "lector"):
        p.add_argument(f"--{papel}-cert", type=Path, required=True)
        p.add_argument(f"--{papel}-key", type=Path, required=True)
    p.add_argument("--expediente-ref", required=True)
    p.add_argument("--version", required=True, type=int)
    p.add_argument("--tipo", required=True, help="Clave nueva, sintética y única")
    p.add_argument("--etiqueta", default="Ensayo de plantilla RRHH")
    p.add_argument("--evidencia", required=True, type=Path, help="JSON sin secretos en /dev/shm")
    return p.parse_args()


def validar(cfg):
    u = urlparse(cfg.origen)
    if u.scheme != "https" or u.hostname not in ("localhost", "127.0.0.1", "::1") or u.path not in ("", "/") or u.query or u.fragment or u.username or u.password:
        raise Fallo("el origen debe ser HTTPS loopback sin ruta ni credenciales")
    if cfg.version < 1 or not cfg.expediente_ref or not cfg.tipo.startswith("ensayo_"):
        raise Fallo("se exige expediente existente y clave sintética ensayo_*")
    if not str(cfg.evidencia.resolve()).startswith("/dev/shm/"):
        raise Fallo("la evidencia debe quedar en /dev/shm")
    if cfg.fase == "alta" and cfg.evidencia.exists():
        raise Fallo("la evidencia ya existe; comprobar el estado antes de repetir cualquier POST")
    huellas = []
    for papel in ("editor", "publicador", "lector"):
        cert, key = getattr(cfg, f"{papel}_cert"), getattr(cfg, f"{papel}_key")
        if not cert.is_file() or not key.is_file() or not cert.stat().st_size or not key.stat().st_size:
            raise Fallo(f"falta certificado mTLS sintético de {papel}")
        huellas.append(hashlib.sha256(cert.read_bytes()).hexdigest())
    if len(set(huellas)) != 3:
        raise Fallo("editor, publicador y lector deben usar certificados distintos")


def contexto(navegador, cfg, papel, ancho):
    return navegador.new_context(
        client_certificates=[{"origin": cfg.origen.rstrip("/"),
                              "certPath": str(getattr(cfg, f"{papel}_cert")),
                              "keyPath": str(getattr(cfg, f"{papel}_key"))}],
        ignore_https_errors=False, service_workers="block", locale="es-ES",
        timezone_id="Europe/Madrid", viewport={"width": ancho, "height": 900 if ancho == 1440 else 844},
        accept_downloads=True,
    )


def observar(pagina):
    errores, cookies = [], []
    pagina.on("pageerror", lambda error: errores.append(str(error)))
    pagina.on("response", lambda r: cookies.append(urlparse(r.url).path) if "set-cookie" in r.headers else None)
    return errores, cookies


def comprobar_pagina(ctx, pagina, ancho, errores, cookies):
    estado = pagina.evaluate("""async () => ({ ancho: document.documentElement.clientWidth,
      contenido: document.documentElement.scrollWidth, local: localStorage.length,
      sesion: sessionStorage.length, indexed: (await indexedDB.databases()).length })""")
    if estado["ancho"] != ancho or estado["contenido"] > estado["ancho"]:
        raise Fallo(f"desbordamiento horizontal a {ancho}px: {estado}")
    if ctx.cookies() or cookies or errores or estado["local"] or estado["sesion"] or estado["indexed"]:
        raise Fallo(f"cookies, almacenamiento o error JS a {ancho}px: {errores}")


def esperar(pagina, ruta, metodo="POST"):
    return pagina.expect_response(lambda r: urlparse(r.url).path == ruta and r.request.method == metodo, timeout=30000)


def guardar_evidencia(cfg, valor):
    cfg.evidencia.write_text(json.dumps(valor, ensure_ascii=False, indent=2) + "\n")


def abrir_plantillas(pagina, cfg):
    respuesta = pagina.goto(cfg.origen.rstrip("/") + "/portal-empleado/#plantillas-rrhh", wait_until="domcontentloaded")
    if respuesta is None or respuesta.status != 200:
        raise Fallo("portal RRHH no respondió 200")
    pagina.locator(".rrhh-plantillas-panel").first.wait_for(timeout=15000)


def alta(navegador, cfg):
    evidencia = {"base": "bb67ac3ac", "tipo": cfg.tipo, "expediente_ref": cfg.expediente_ref,
                 "version": cfg.version, "viewports": [1440, 390], "fase": "alta"}
    for ancho in (1440, 390):
        ctx = contexto(navegador, cfg, "editor", ancho)
        try:
            page = ctx.new_page(); errores, cookies = observar(page)
            with esperar(page, RUTA_CATALOGO, "GET") as peticion:
                abrir_plantillas(page, cfg)
            if peticion.value.status != 200:
                raise Fallo("GET catálogo del editor no respondió 200")
            if page.locator("[data-plantillas-publicar]").count():
                raise Fallo("el editor también presenta formulario de publicación")
            if ancho == 1440:
                page.locator('[data-plantillas-accion="nueva"]').click()
                campos = {"clave": cfg.tipo, "etiqueta": cfg.etiqueta,
                          "titulo": cfg.etiqueta + " — borrador de ensayo", "orden": "99",
                          "vigente_desde": "2026-01-01", "fuente_ref": "paquete:ejemplo:vec:v1",
                          "modalidades": "*", "motivo": "Ensayo sintético RRHH 4.08"}
                for nombre, valor in campos.items():
                    page.locator(f'[data-plantillas-form] [name="{nombre}"]').fill(valor)
                page.locator('[data-plantillas-form] [name="parrafo"]').first.fill(
                    "BORRADOR DE ENSAYO SIN EFECTOS ADMINISTRATIVOS. Expediente {{numero_expediente}}.")
                with esperar(page, RUTA_ENTRADAS) as peticion:
                    page.locator('[data-plantillas-form] button[type="submit"]').click()
                r = peticion.value
                if r.status != 201:
                    raise Fallo(f"alta de tipo respondió {r.status}")
                cuerpo = r.json()
                if cuerpo.get("catalogo", {}).get("estado") != "borrador" or not any(
                    e.get("clave") == cfg.tipo for e in cuerpo["catalogo"]["entradas"]):
                    raise Fallo("el recibo de alta no conserva el tipo nuevo")
                evidencia["alta"] = {"http": r.status, "recibo": cuerpo["recibo"],
                                     "version": cuerpo["catalogo"]["version"],
                                     "revision": cuerpo["catalogo"]["revision"]}
                guardar_evidencia(cfg, evidencia)
            comprobar_pagina(ctx, page, ancho, errores, cookies)
        finally:
            ctx.close()
    ctx = contexto(navegador, cfg, "publicador", 1440)
    try:
        page = ctx.new_page(); errores, cookies = observar(page)
        with esperar(page, RUTA_CATALOGO, "GET") as peticion:
            abrir_plantillas(page, cfg)
        if peticion.value.status != 200 or cfg.tipo not in page.locator("body").inner_text():
            raise Fallo("el publicador no ve el borrador creado")
        if page.locator('[data-plantillas-accion="nueva"]').count():
            raise Fallo("el publicador también presenta alta de tipos")
        page.locator('[data-plantillas-publicar] [name="aprobacion_ref"]').fill("sin_aprobacion_rrhh:ejemplo_desarrollo")
        page.locator('[data-plantillas-publicar] [name="motivo"]').fill("Publicación sintética de ensayo RRHH 4.08")
        with esperar(page, RUTA_PUBLICAR) as peticion:
            page.locator('[data-plantillas-publicar] button[type="submit"]').click()
        r = peticion.value
        if r.status != 201:
            raise Fallo(f"publicación respondió {r.status}")
        cuerpo = r.json()
        if cuerpo.get("catalogo", {}).get("estado") != "publicado":
            raise Fallo("la respuesta no acredita versión publicada")
        evidencia["publicacion"] = {"http": r.status, "recibo": cuerpo["recibo"],
                                     "catalogo_ref": cuerpo["catalogo"]["id"],
                                     "huella": cuerpo["catalogo"]["huella_sha256"]}
        guardar_evidencia(cfg, evidencia)
        comprobar_pagina(ctx, page, 1440, errores, cookies)
    finally:
        ctx.close()
    evidencia["descarga"] = descargar(navegador, cfg)
    if any(r["catalogo_huella"] != evidencia["publicacion"]["huella"]
           or r["procedencia_ref"] != evidencia["publicacion"]["recibo"]["recibo_ref"]
           for r in evidencia["descarga"]):
        raise Fallo("la descarga no procede de la publicación registrada")
    return evidencia


def descargar(navegador, cfg):
    resultados = []
    for ancho in (1440, 390):
        ctx = contexto(navegador, cfg, "lector", ancho)
        try:
            page = ctx.new_page(); errores, cookies = observar(page)
            url = cfg.origen.rstrip("/") + "/portal-empleado/?expediente=" + quote(cfg.expediente_ref, safe="") + "#contratacion-temporal"
            with esperar(page, RUTA_DISPONIBLES) as peticion:
                inicial = page.goto(url, wait_until="domcontentloaded")
            lista = peticion.value
            if inicial is None or inicial.status != 200 or lista.status != 200:
                raise Fallo(f"detalle/lista de documentos no respondió 200 a {ancho}px")
            catalogo = lista.json()
            if not any(t.get("clave") == cfg.tipo for t in catalogo.get("tipos", [])):
                raise Fallo("la API del expediente no incluye el tipo publicado")
            if not catalogo.get("procedencia_ref"):
                raise Fallo("la lista documental carece de recibo de procedencia")
            fila = page.locator("[data-ct-borradores-publicados] tr").filter(has=page.locator(f'[data-bp-descargar="{cfg.tipo}"]'))
            fila.first.wait_for(timeout=15000)
            if cfg.etiqueta not in fila.first.inner_text():
                raise Fallo("el expediente no muestra la etiqueta del tipo nuevo")
            with esperar(page, RUTA_DESCARGA) as peticion, page.expect_download(timeout=30000) as descarga:
                fila.first.locator(f'[data-bp-descargar="{cfg.tipo}"][data-bp-formato="pdf"]').click()
            r = peticion.value
            if r.status != 200:
                raise Fallo(f"descarga respondió {r.status}")
            documento = descarga.value
            bytes_pdf = Path(documento.path()).read_bytes()
            huella = hashlib.sha256(bytes_pdf).hexdigest()
            if not bytes_pdf.startswith(b"%PDF-") or huella != r.headers.get("x-vec-documento-sha256"):
                raise Fallo("PDF o huella documental incompatible")
            if catalogo.get("catalogo_ref") != r.headers.get("x-vec-catalogo-ref") or catalogo.get("catalogo_huella_sha256") != r.headers.get("x-vec-catalogo-huella-sha256"):
                raise Fallo("la descarga no corresponde al catálogo listado")
            resultados.append({"viewport": ancho, "lista_http": lista.status,
                               "descarga_http": r.status, "pdf_sha256": huella,
                               "procedencia_ref": catalogo["procedencia_ref"],
                               "catalogo_ref": r.headers.get("x-vec-catalogo-ref"),
                               "catalogo_huella": r.headers.get("x-vec-catalogo-huella-sha256")})
            comprobar_pagina(ctx, page, ancho, errores, cookies)
        finally:
            ctx.close()
    if resultados[0]["catalogo_huella"] != resultados[1]["catalogo_huella"]:
        raise Fallo("el catálogo difiere entre vistas")
    return resultados


def recuperar(navegador, cfg, anterior):
    if anterior.get("tipo") != cfg.tipo or anterior.get("expediente_ref") != cfg.expediente_ref:
        raise Fallo("la evidencia previa no corresponde al caso solicitado")
    nuevo = descargar(navegador, cfg)
    if [r["catalogo_huella"] for r in nuevo] != [r["catalogo_huella"] for r in anterior["descarga"]]:
        raise Fallo("la versión del catálogo cambió tras reinicio")
    if [r["pdf_sha256"] for r in nuevo] != [r["pdf_sha256"] for r in anterior["descarga"]]:
        raise Fallo("el PDF cambió tras reinicio")
    anterior["recuperacion"] = nuevo
    anterior["fase"] = "recuperada_tras_reinicio_externo"
    return anterior


def main():
    try:
        cfg = argumentos(); validar(cfg)
        from playwright.sync_api import sync_playwright
        with sync_playwright() as pw:
            browser = pw.chromium.launch(headless=True)
            try:
                if cfg.fase == "alta":
                    resultado = alta(browser, cfg)
                else:
                    if not cfg.evidencia.is_file():
                        raise Fallo("falta evidencia de la fase alta")
                    resultado = recuperar(browser, cfg, json.loads(cfg.evidencia.read_text()))
            finally:
                browser.close()
        guardar_evidencia(cfg, resultado)
        print(json.dumps({"fase": resultado["fase"], "evidencia": str(cfg.evidencia)}, ensure_ascii=False))
        return 0
    except Exception as error:
        print(f"FALLO recorrido RRHH plantillas: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
