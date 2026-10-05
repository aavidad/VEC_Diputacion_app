#!/usr/bin/env python3
"""Sirve una preparación sintética en loopback, sin escrituras ni datos personales."""

import argparse
import json
import mimetypes
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlsplit


MAX_RECURSO = 1024 * 1024
COMUNES = (
    "portal-empleado/portal.css", "portal-empleado/portal-componentes.css",
    "portal-empleado/portal-patrones.css", "portal-empleado/portal-flujos.css",
    "portal-empleado/portal-modulos.css", "comun/tema-vec.css",
    "comun/textos.js", "comun/idioma.js", "textos/idiomas.json",
)


def cargar_recursos(raiz, modulo):
    """La lista cerrada no expone directorios, fuentes Go ni archivos del operador."""
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

    rutas = list(COMUNES)
    indice = json.loads(leer("textos/idiomas.json"))
    for idioma in indice["idiomas"]:
        codigo = idioma["codigo"]
        if not isinstance(codigo, str) or not codigo.replace("-", "").isalpha():
            raise ValueError("indice_idiomas_invalido")
        rutas.append(f"textos/{codigo}/{modulo}.json")
        error = f"textos/{codigo}/{modulo}-error.json"
        if (raiz / error).is_file():
            rutas.append(error)
    visor_bases = modulo == "seleccion-bases-preparacion"
    visor_admision = modulo == "selectivos-admision-visor"
    visor_acta = modulo == "selectivos-acta-visor"
    visor_lista = modulo == "selectivos-lista-admision-visor"
    visor_archivo = visor_bases or visor_admision or visor_acta or visor_lista
    prefijo = "portal-empleado/modulos/seleccion/preparacion-bases" if visor_bases else (
        "portal-empleado/modulos/seleccion/preparacion-admision" if visor_admision else (
            "portal-empleado/modulos/seleccion/preparacion-acta" if visor_acta else (
                "portal-empleado/modulos/seleccion/preparacion-lista-admision" if visor_lista else f"portal-empleado/modulos/{modulo}"
            )
        )
    )
    # Archivos de la vista, nunca pruebas ni datos aportados por una persona.
    propios = ("preparacion-bases.css", "cliente-http.js", "contrato-http.js") if visor_bases else (
        ("preparacion-admision.css", "controlador.js") if visor_admision else (
            ("preparacion-acta.css",) if visor_acta else (
                ("preparacion-lista-admision.css",) if visor_lista else (f"{modulo}.css", "escenario.json")
            )
        )
    )
    if visor_admision or visor_acta or visor_lista:
        rutas.extend((
            "portal-empleado/modulos/seleccion/preparacion-bases/modelo.js",
            "portal-empleado/modulos/seleccion/preparacion-bases/contrato-http.js",
        ))
    if visor_acta:
        rutas.append("portal-empleado/modulos/seleccion/dom.js")
    if visor_lista:
        rutas.append("portal-empleado/modulos/seleccion/preparacion-admision/controlador.js")
        # Textos de motivos de cada catálogo de admisión, por idioma; sólo JSON regulares.
        for idioma in indice["idiomas"]:
            carpeta = raiz / "textos" / idioma["codigo"]
            rutas.extend(f"textos/{idioma['codigo']}/{a.name}" for a in sorted(carpeta.glob("motivos-*.json")) if a.is_file())
    for nombre in ("index.html", "entrada.js", "cliente.js", "vista.js", "modelo.js", *propios):
        ruta = f"{prefijo}/{nombre}"
        if (raiz / ruta).exists():
            rutas.append(ruta)
    if f"{prefijo}/index.html" not in rutas or not visor_archivo and f"{prefijo}/escenario.json" not in rutas:
        raise ValueError("preparacion_ausente")
    if not visor_archivo:
        escenario = json.loads(leer(f"{prefijo}/escenario.json"))
        if escenario.get("alcance") != "preparacion_sintetica":
            raise ValueError("alcance_invalido")
    recursos = {}
    for ruta in rutas:
        contenido = leer(ruta)
        tipo = mimetypes.guess_type(ruta)[0] or "application/octet-stream"
        recursos[f"/{ruta}"] = (contenido, tipo)
    recursos[f"/{prefijo}/"] = recursos[f"/{prefijo}/index.html"]
    defecto = indice.get("por_defecto", indice["idiomas"][0]["codigo"])
    error = recursos.get(f"/textos/{defecto}/{modulo}-error.json")
    if error is not None:
        recursos[f"/{prefijo}/error-catalogo.json"] = error
    return recursos, f"/{prefijo}/"


def handler_para(recursos):
    class Handler(BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.0"

        def setup(self):
            super().setup()
            self.connection.settimeout(5)

        def log_message(self, *_args):
            pass

        def do_GET(self):
            self.responder(False)

        def do_HEAD(self):
            self.responder(True)

        def responder(self, solo_cabeceras):
            esperado = f"127.0.0.1:{self.server.server_port}"
            if self.headers.get("Host") != esperado or self.headers.get("Sec-Fetch-Site") == "cross-site":
                self.send_error(403)
                return
            recurso = recursos.get(urlsplit(self.path).path)
            if recurso is None:
                self.send_error(404)
                return
            contenido, tipo = recurso
            self.send_response(200)
            self.send_header("Content-Type", tipo + ("; charset=utf-8" if tipo.startswith("text/") or tipo == "application/javascript" else ""))
            self.send_header("Content-Length", str(len(contenido)))
            self.send_header("Cache-Control", "no-store")
            self.send_header("X-Content-Type-Options", "nosniff")
            self.send_header("Referrer-Policy", "no-referrer")
            self.send_header("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; font-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
            self.send_header("Cross-Origin-Resource-Policy", "same-origin")
            self.end_headers()
            if not solo_cabeceras:
                self.wfile.write(contenido)

    return Handler


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--modulo", choices=("formacion", "carrera", "seleccion-bases-preparacion", "selectivos-admision-visor", "selectivos-acta-visor", "selectivos-lista-admision-visor"), required=True)
    parser.add_argument("--web-dir", type=Path, default=Path(__file__).resolve().parents[1] / "web/static")
    args = parser.parse_args()
    recursos, entrada = cargar_recursos(args.web_dir, args.modulo)
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
