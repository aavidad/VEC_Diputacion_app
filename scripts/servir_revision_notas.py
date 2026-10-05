#!/usr/bin/env python3
"""Sirve los recursos públicos del visor de revisión de notas en loopback."""

import argparse
import json
import mimetypes
from http.server import ThreadingHTTPServer
from pathlib import Path

if __package__:
    from .servir_preparacion_rrhh import COMUNES, MAX_RECURSO, handler_para
else:
    from servir_preparacion_rrhh import COMUNES, MAX_RECURSO, handler_para


PREFIJO = "portal-empleado/modulos/seleccion/preparacion-notas"
PROPIOS = (
    *(f"{PREFIJO}/{nombre}" for nombre in (
        "index.html", "entrada.js", "modelo.js", "vista.js", "preparacion-notas.css",
    )),
    "portal-empleado/modulos/seleccion/preparacion-bases/modelo.js",
    "portal-empleado/modulos/seleccion/preparacion-bases/contrato-http.js",
    "portal-empleado/modulos/seleccion/configuracion.js",
    "portal-empleado/modulos/seleccion/dom.js",
)


def cargar_recursos(raiz):
    """Carga una lista cerrada de recursos; los archivos preparados quedan fuera."""
    raiz = raiz.resolve(strict=True)

    def leer(ruta):
        archivo = (raiz / ruta).resolve(strict=True)
        if not archivo.is_relative_to(raiz) or not archivo.is_file():
            raise ValueError("recurso_invalido")
        if archivo.stat().st_size > MAX_RECURSO:
            raise ValueError("recurso_demasiado_grande")
        contenido = archivo.read_bytes()
        if not contenido or len(contenido) > MAX_RECURSO:
            raise ValueError("recurso_invalido")
        return contenido

    indice = json.loads(leer("textos/idiomas.json"))
    rutas = [*COMUNES, *PROPIOS]
    for idioma in indice["idiomas"]:
        codigo = idioma["codigo"]
        if not isinstance(codigo, str) or not codigo.replace("-", "").isalpha():
            raise ValueError("indice_idiomas_invalido")
        rutas.append(f"textos/{codigo}/seleccion.json")
    recursos = {}
    for ruta in rutas:
        tipo = mimetypes.guess_type(ruta)[0] or "application/octet-stream"
        recursos[f"/{ruta}"] = (leer(ruta), tipo)
    entrada = f"/{PREFIJO}/"
    recursos[entrada] = recursos[f"/{PREFIJO}/index.html"]
    return recursos, entrada


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--web-dir", type=Path, default=Path(__file__).resolve().parents[1] / "web/static")
    args = parser.parse_args()
    recursos, entrada = cargar_recursos(args.web_dir)
    with ThreadingHTTPServer(("127.0.0.1", 0), handler_para(recursos)) as servidor:
        servidor.daemon_threads = True
        servidor.timeout = 5
        print(f"http://127.0.0.1:{servidor.server_port}{entrada}", flush=True)
        try:
            servidor.serve_forever()
        except KeyboardInterrupt:
            pass


if __name__ == "__main__":
    main()
