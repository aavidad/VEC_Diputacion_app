#!/usr/bin/env python3
"""Recorrido de navegador local para la lectura propia de Bolsa.

No prepara datos, no inicia servicios y no acepta orígenes que no sean
loopback. El servidor y los certificados sintéticos se entregan explícitamente
por quien haya montado el entorno aislado.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from urllib.parse import urlparse


class FalloRecorrido(RuntimeError):
    """Un contrato visible o de transporte no se cumplió."""


def argumentos() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--origen", required=True, help="https://localhost:PUERTO")
    parser.add_argument("--certificado", required=True, type=Path)
    parser.add_argument("--clave", required=True, type=Path)
    parser.add_argument("--ruta", default="/area-personal/?vista=llamamientos")
    return parser.parse_args()


def comprobar_entrada(cfg: argparse.Namespace) -> str:
    origen = cfg.origen.rstrip("/")
    url = urlparse(origen)
    if url.scheme != "https" or url.hostname not in {"localhost", "127.0.0.1", "::1"}:
        raise FalloRecorrido("el recorrido solo admite HTTPS en loopback")
    if url.username or url.password or url.path or url.query or url.fragment:
        raise FalloRecorrido("el origen debe ser solo esquema, host y puerto")
    if not cfg.ruta.startswith("/area-personal/"):
        raise FalloRecorrido("la ruta debe pertenecer al área personal")
    for fichero in (cfg.certificado, cfg.clave):
        if not fichero.is_file() or fichero.stat().st_size == 0:
            raise FalloRecorrido("falta material mTLS sintético legible")
    return origen


def comprobar_contexto(contexto, pagina, origen: str, ancho: int) -> dict[str, object]:
    errores_js: list[str] = []
    respuestas: dict[str, int] = {}
    cookies_set: list[str] = []

    pagina.on("pageerror", lambda error: errores_js.append(str(error)))

    def observar(respuesta) -> None:
        ruta = urlparse(respuesta.url).path
        if ruta in {"/api/vec/bolsa/mi-bolsa", "/api/vec/bolsa/mi-bolsa/historial"}:
            respuestas[ruta] = respuesta.status
            if "set-cookie" in respuesta.headers:
                cookies_set.append(ruta)

    pagina.on("response", observar)
    respuesta = pagina.goto(origen + "/area-personal/?vista=llamamientos", wait_until="networkidle", timeout=30_000)
    if respuesta is None or respuesta.status != 200:
        raise FalloRecorrido(f"el área personal respondió {getattr(respuesta, 'status', 'sin respuesta')}")
    pagina.locator("#titulo-vista").wait_for(timeout=10_000)
    pagina.locator("#historial-mi-bolsa").wait_for(timeout=10_000)
    pagina.wait_for_function(
        """() => document.querySelector('#historial-mi-bolsa')?.textContent.includes('Histórico de mi bolsa')""",
        timeout=10_000,
    )
    if pagina.locator("#titulo-vista").inner_text().strip() != "Mi bolsa":
        raise FalloRecorrido("la navegación no mostró Mi bolsa")
    if respuestas.get("/api/vec/bolsa/mi-bolsa") != 200:
        raise FalloRecorrido("GET Mi bolsa no respondió 200 en el navegador")
    if respuestas.get("/api/vec/bolsa/mi-bolsa/historial") != 200:
        raise FalloRecorrido("GET del histórico propio no respondió 200 en el navegador")
    estado = pagina.evaluate(
        """() => ({
          anchoCliente: document.documentElement.clientWidth,
          anchoContenido: document.documentElement.scrollWidth,
          local: localStorage.length,
          sesion: sessionStorage.length
        })"""
    )
    if estado["anchoCliente"] != ancho or estado["anchoContenido"] > estado["anchoCliente"]:
        raise FalloRecorrido("la vista presenta desbordamiento horizontal")
    if contexto.cookies() or cookies_set or errores_js or estado["local"] or estado["sesion"]:
        raise FalloRecorrido("cookies, almacenamiento web o errores JavaScript detectados")
    return {"viewport": ancho, "mi_bolsa": 200, "historial": 200}


def main() -> int:
    cfg = argumentos()
    try:
        origen = comprobar_entrada(cfg)
        from playwright.sync_api import sync_playwright

        resultados: list[dict[str, object]] = []
        with sync_playwright() as playwright:
            navegador = playwright.chromium.launch(headless=True)
            try:
                for ancho, alto in ((1440, 900), (390, 844)):
                    contexto = navegador.new_context(
                        client_certificates=[{"origin": origen, "certPath": str(cfg.certificado), "keyPath": str(cfg.clave)}],
                        ignore_https_errors=False,
                        service_workers="block",
                        locale="es-ES",
                        timezone_id="Europe/Madrid",
                        viewport={"width": ancho, "height": alto},
                    )
                    try:
                        resultados.append(comprobar_contexto(contexto, contexto.new_page(), origen, ancho))
                    finally:
                        contexto.close()
            finally:
                navegador.close()
        print(json.dumps({"recorrido": "mi_bolsa_lectura", "resultados": resultados}, ensure_ascii=False))
        return 0
    except Exception as error:
        print(f"FALLO recorrido Mi Bolsa: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
