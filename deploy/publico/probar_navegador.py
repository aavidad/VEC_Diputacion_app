#!/usr/bin/env python3
"""Recorrido público real con Chrome del sistema y CA privada del ensayo."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import urllib.parse

from playwright.sync_api import sync_playwright
from runtime import load


def verify(context, page):
    overflow = page.evaluate("document.documentElement.scrollWidth > window.innerWidth")
    storage = page.evaluate("async () => ({local:localStorage.length,session:sessionStorage.length,indexed:(await indexedDB.databases()).length,cookie:document.cookie})")
    if overflow or context.cookies() or storage != {"local": 0, "session": 0, "indexed": 0, "cookie": ""}:
        raise ValueError("browser_state_rejected")


def run(config, captures):
    cfg, _ = load(config)
    settings = cfg["check"]
    origin = settings["origin"].rstrip("/")
    convocatorias_ok = True
    with tempfile.TemporaryDirectory(prefix="vec-publico-chrome-", dir="/var/tmp") as temporary:
        home = Path(temporary)
        nss = home / ".pki/nssdb"
        nss.mkdir(parents=True)
        for command in (["certutil", "-N", "-d", "sql:" + str(nss), "--empty-password"],
                        ["certutil", "-A", "-d", "sql:" + str(nss), "-n", "vec-publico-ensayo",
                         "-t", "CT,,", "-i", settings["ca_file"]]):
            subprocess.run(command, check=True, capture_output=True)
        with sync_playwright() as playwright:
            browser = playwright.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True,
                args=["--disable-dev-shm-usage", "--no-sandbox"],
                env={"PATH": os.defpath, "HOME": str(home), "TMPDIR": str(home), "LANG": "C.UTF-8"})
            try:
                for width in (1440, 390):
                    context = browser.new_context(viewport={"width": width, "height": 900}, locale="es-ES")
                    errors, failed, responses = [], [], []
                    context.on("page", lambda p: p.on("pageerror", lambda _: errors.append(True)))
                    context.on("requestfailed", lambda _: failed.append(True))
                    context.on("response", lambda r: responses.append((urllib.parse.urlsplit(r.url).path, r.status)))
                    context.route("**/*", lambda route: route.continue_() if route.request.url.startswith(origin + "/") else route.abort())
                    page = context.new_page()
                    page.set_default_timeout(10000)
                    response = page.goto(origin + "/bolsa/", wait_until="networkidle")
                    if response.status != 200:
                        raise ValueError("browser_http_rejected")
                    screenshot(page, captures / ("listado-" + str(width) + ".png"))
                    diagnostic = captures / "browser-diagnostic.json"
                    diagnostic.write_text(json.dumps({"responses": responses,
                        "body": page.locator("body").inner_text(), "js_errors": len(errors),
                        "request_failed": len(failed)}, ensure_ascii=False))
                    diagnostic.chmod(0o600)
                    cards = page.locator('.tarjeta-convocatoria[data-identificador="' + settings["convocatoria"] + '"] a')
                    if cards.count():
                        cards.click()
                        page.locator("#contenido-detalle").wait_for(state="visible")
                        page.wait_for_load_state("networkidle")
                        verify(context, page)
                        screenshot(page, captures / ("convocatoria-" + str(width) + ".png"))
                    else:
                        convocatorias_ok = False
                        schema = page.evaluate("async()=> (await (await fetch('/api/publico/bolsa/convocatorias',{cache:'no-store',credentials:'omit'})).json()).esquema")
                        print(json.dumps({"viewport": width, "convocatorias_rendered": False,
                                          "schema": schema}))
                    page.goto(origin + "/bolsa/listas.html", wait_until="networkidle")
                    page.locator("#cuerpo-tabla-bolsas tr").first.wait_for(state="visible")
                    verify(context, page)
                    screenshot(page, captures / ("bolsas-listado-" + str(width) + ".png"))
                    page.locator('button[data-bolsa-ref="' + settings["bolsa"] + '"]').click()
                    page.locator("#cuerpo-tabla-lista tr").first.wait_for(state="visible")
                    page.wait_for_load_state("networkidle")
                    verify(context, page)
                    screenshot(page, captures / ("bolsa-" + str(width) + ".png"))
                    expected = {
                        "/api/publico/bolsa/bolsas",
                        "/api/publico/bolsa/bolsas/" + settings["bolsa"] + "/lista",
                    }
                    if (errors or failed or any(s >= 400 for _, s in responses)
                            or not expected.issubset({p for p, s in responses if s == 200})):
                        raise ValueError("browser_response_rejected")
                    print(json.dumps({"viewport": width, "public_api_200": len(expected),
                                      "js_errors": 0, "cookies": 0, "storage": 0, "global_overflow": False,
                                      "bolsas_rendered": True}))
                    private = page.evaluate("async () => Promise.all(['/api/vec/contratacion-temporal/solicitudes','/api/vec/bolsa/bolsas','/portal-empleado/','/admin/','/rrhh/'].map(async p => (await fetch(p,{cache:'no-store',credentials:'omit'})).status))")
                    if private != [404] * 5:
                        raise ValueError("browser_private_route_exposed")
                    print(json.dumps({"viewport": width, "private_404": len(private)}))
                    context.close()
            finally:
                browser.close()
    return convocatorias_ok


def screenshot(page, path):
    page.screenshot(path=str(path), full_page=True)
    print(json.dumps({"capture": path.name, "sha256": hashlib.sha256(path.read_bytes()).hexdigest()}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    parser.add_argument("--captures", required=True, type=Path)
    args = parser.parse_args()
    if not args.captures.is_absolute() or not args.captures.is_dir():
        raise ValueError("capture_directory")
    sys.exit(0 if run(args.config, args.captures) else 1)
