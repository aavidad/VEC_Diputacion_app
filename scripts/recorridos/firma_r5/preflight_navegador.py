#!/usr/bin/env python3
"""Lectura previa en Chrome: no envía firmas ni presenta un recibo sintético."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--certificado', type=Path, required=True)
    parser.add_argument('--clave', type=Path, required=True)
    parser.add_argument('--salida', type=Path, required=True)
    args = parser.parse_args()
    origen = 'https://localhost:18443'
    # Se importa el lector completo: su pin Playwright se comprueba antes de Chrome.
    archivo = args.source / 'scripts/recorridos/propuesta_firma/recorrer.py'
    spec = importlib.util.spec_from_file_location('vec_guardia_firma_original', archivo)
    reader = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = reader
    spec.loader.exec_module(reader)
    reader.comprobar_sdk_playwright()
    for credential in [args.certificado, args.clave]:
        st = credential.stat()
        if credential.resolve() != credential.absolute() or st.st_uid != os.getuid() or st.st_mode & 0o077:
            raise ValueError('material_sintetico_no_privado')
    parent = args.salida.parent
    if parent.resolve() != parent.absolute() or parent.stat().st_mode & 0o077 or parent.stat().st_uid != os.getuid():
        raise ValueError('salida_no_privada')
    if args.salida.exists() or any((p / '.git').exists() for p in [parent, *parent.parents]):
        raise ValueError('salida_existente_o_git')
    chrome = shutil.which('google-chrome')
    if not chrome:
        raise ValueError('chrome_sistema_ausente')
    from playwright.sync_api import sync_playwright
    observations = []
    errors = []
    with sync_playwright() as pw:
        browser = pw.chromium.launch(executable_path=chrome, headless=True)
        guardia = reader.GuardiaNavegador(browser, origen)
        context = browser.new_context(viewport={'width': 1440, 'height': 900},
            ignore_https_errors=False, service_workers='block', locale='es-ES',
            client_certificates=[{'origin': origen, 'certPath': str(args.certificado), 'keyPath': str(args.clave)}])
        context.route_web_socket('**/*', lambda route: reader.limitar_websocket(route, False))
        context.on('response', lambda response: observations.append({'metodo': response.request.method,
            'ruta': response.url[len(origen):].split('?')[0], 'estado': response.status}) if response.url.startswith(origen + '/') and len(observations) < 256 else None)
        page = context.new_page()
        page.on('pageerror', lambda _: errors.append('error_js') if len(errors) < 16 else None)
        live = page.goto(origen + '/livez', wait_until='domcontentloaded', timeout=15000)
        if live is None or live.status != 200:
            raise ValueError('livez_no_disponible')
        portal = page.goto(origen + '/portal-empleado/#contratacion-temporal', wait_until='domcontentloaded', timeout=30000)
        page.wait_for_timeout(1000)
        measurements = []
        for width in [1440, 390]:
            page.set_viewport_size({'width': width, 'height': 900})
            measurements.append({'ancho': width, 'desbordamiento': page.evaluate('document.documentElement.scrollWidth > innerWidth'),
                'local_storage': page.evaluate('localStorage.length'), 'session_storage': page.evaluate('sessionStorage.length')})
        result = {'corte': 'preflight_navegador', 'firma_ejecutada': False, 'recibo_acreditado': False,
            'chrome_sistema': True, 'portal_http': portal.status if portal else None, 'http': observations,
            'errores_js': errors, 'guardia_red': list(guardia.errores), 'cookies': len(context.cookies()), 'pantallas': measurements}
        context.close()
        guardia.cerrar()
    raw = (json.dumps(result, indent=2) + '\n').encode()
    reader.guardar_privado(args.salida, raw)
    print(json.dumps({'corte': result['corte'], 'portal_http': result['portal_http'],
        'firma_ejecutada': False, 'salida_sha256': hashlib.sha256(raw).hexdigest()}))


if __name__ == '__main__':
    try:
        main()
    except Exception:
        print(json.dumps({'corte': 'precondiciones', 'firma_ejecutada': False, 'codigo': 'lectura_no_acreditada'}))
        raise SystemExit(2)
