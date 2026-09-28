#!/usr/bin/env python3
"""Sonda de navegador mTLS para un VEC sintético y aislado.

La salida JSON distingue observaciones reales de escenarios no preparados. No
guarda certificados ni respuestas de negocio en el repositorio.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from urllib.parse import urlparse


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--material", type=Path, required=True)
    parser.add_argument("--salida", type=Path, required=True)
    parser.add_argument("--bolsa-ref", default="")
    args = parser.parse_args()
    parsed = urlparse(args.base_url)
    if parsed.scheme != "https" or parsed.hostname != "localhost" or not parsed.port:
        parser.error("la URL debe ser https://localhost:PUERTO")
    material = args.material.resolve(strict=True)
    for relative in ("mtls/cliente.crt", "mtls/cliente.key", "ca/ca.crt"):
        if not (material / relative).is_file():
            parser.error(f"falta material mTLS: {relative}")

    try:
        from playwright.sync_api import sync_playwright
    except ImportError as exc:
        raise SystemExit("Falta Playwright Python para Chrome mTLS") from exc

    resultado = {"esquema": "vec.rrhh.e2e-aislado.navegador.v1", "origen": args.base_url, "vistas": []}
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True, executable_path="/usr/bin/google-chrome")
        try:
            for width in (1440, 390):
                context = browser.new_context(
                    viewport={"width": width, "height": 900},
                    locale="es-ES", timezone_id="Europe/Madrid", service_workers="block",
                    ignore_https_errors=True,
                    client_certificates=[{
                        "origin": args.base_url,
                        "certPath": str(material / "mtls/cliente.crt"),
                        "keyPath": str(material / "mtls/cliente.key"),
                    }],
                )
                page = context.new_page()
                errors: list[str] = []
                page.on("pageerror", lambda error: errors.append(str(error)[:300]))
                page.on("console", lambda message: errors.append(message.text[:300]) if message.type == "error" else None)
                response = page.goto(f"{args.base_url}/portal-empleado/", wait_until="domcontentloaded", timeout=30000)
                page.wait_for_timeout(500)
                status = response.status if response else 0
                overflow = page.evaluate("document.documentElement.scrollWidth > window.innerWidth")
                storage = page.evaluate("({ cookies: document.cookie, local: localStorage.length, session: sessionStorage.length })")
                view = {"ancho": width, "portal_http": status, "overflow_global": bool(overflow),
                        "errores_js": errors, "cookies": storage["cookies"],
                        "local_storage": storage["local"], "session_storage": storage["session"]}
                if args.bolsa_ref:
                    view["politica_ofertas"] = page.evaluate("""async (bolsa) => {
                        const r = await fetch('/api/vec/bolsa/politica-ofertas?'
                            + new URLSearchParams({bolsa_ref: bolsa}),
                            {headers: {Accept: 'application/json'}, credentials: 'same-origin'});
                        const body = await r.json().catch(() => null);
                        const data = body?.data ?? body;
                        return {status: r.status, version: data?.version ?? null,
                            configurada: data?.configurada ?? null,
                            plazo: data?.politica?.plazo ?? null,
                            no_cubierta: data?.politica?.no_cubierta ?? null,
                            recibo_ref: data?.recibo_ref ?? null};
                    }""", args.bolsa_ref)
                resultado["vistas"].append(view)
                context.close()
        finally:
            browser.close()
    args.salida.parent.mkdir(parents=True, exist_ok=True)
    args.salida.write_text(json.dumps(resultado, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    if any(v["portal_http"] != 200 or v["overflow_global"] or v["errores_js"]
           or v["cookies"] or v["local_storage"] or v["session_storage"] for v in resultado["vistas"]):
        return 1
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:
        print(f"navegador mTLS: {type(exc).__name__}: {exc}", file=sys.stderr)
        sys.exit(2)
