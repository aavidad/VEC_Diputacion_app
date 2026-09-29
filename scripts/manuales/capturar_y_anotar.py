#!/usr/bin/env python3
"""Capturas locales anotadas para manuales, con una fuente sintética comprobable."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from io import BytesIO
from pathlib import Path
from threading import Thread
from urllib.parse import parse_qsl, urlsplit

from PIL import Image, ImageDraw, ImageFont


CHROME = Path("/usr/bin/google-chrome")
RAIZ_REPOSITORIO = Path(__file__).resolve().parents[2]
TAMANOS = (("escritorio", 1440, 900), ("movil", 390, 844))
CLAVE = re.compile(r"[a-z0-9]+(?:-[a-z0-9]+)*\Z")
SENSIBLE = re.compile(
    r"\b[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}\b|"
    r"\b\d{8}[A-Za-z]\b|"
    r"\b(?:[6789]\d{8}|ES\d{22})\b|"
    r"\b(?:Bearer\s+\S+|password\s*[:=]\s*\S+|token\s*[:=]\s*\S+)\b",
    re.IGNORECASE,
)
OCULTAR_BASE = "input, textarea, [contenteditable], [data-private], [data-sensitive]"
ESTILO_ESTABLE = """*, *::before, *::after {
  animation-duration: 0s !important; transition-duration: 0s !important;
  caret-color: transparent !important;
}"""


def validar_url_local(valor: str, *, origen: bool = False) -> str:
    """Admite HTTP(S) de loopback sin credenciales ni parámetros en la URL."""
    if any(c.isspace() for c in valor) or "\\" in valor:
        raise ValueError("URL inválida")
    try:
        url = urlsplit(valor)
        puerto = url.port
    except ValueError as exc:
        raise ValueError("puerto de URL inválido") from exc
    if (url.scheme not in {"http", "https"} or
            url.hostname not in {"localhost", "127.0.0.1", "::1"} or
            url.username is not None or url.password is not None or
            url.query or url.fragment or not url.netloc):
        raise ValueError("use una URL HTTP(S) de loopback sin credenciales ni parámetros")
    if origen and url.path not in {"", "/"}:
        raise ValueError("la URL base debe contener solo el origen")
    if puerto is not None and puerto < 1:
        raise ValueError("puerto de URL inválido")
    return valor.rstrip("/") if origen else valor


def validar_escenario(datos: object) -> dict:
    if not isinstance(datos, dict) or set(datos) != {"pantallas"}:
        raise ValueError("el escenario debe contener solo 'pantallas'")
    pantallas = datos["pantallas"]
    if not isinstance(pantallas, list) or not pantallas:
        raise ValueError("incluya al menos una pantalla")
    claves = set()
    for pantalla in pantallas:
        if not isinstance(pantalla, dict) or set(pantalla) != {"clave", "ruta", "pasos", "marcas", "ocultar"}:
            raise ValueError("cada pantalla requiere clave, ruta, pasos, marcas y ocultar")
        clave, ruta = pantalla["clave"], pantalla["ruta"]
        if not isinstance(clave, str) or not CLAVE.fullmatch(clave) or clave in claves:
            raise ValueError("clave de pantalla inválida o repetida")
        claves.add(clave)
        if (not isinstance(ruta, str) or not ruta.startswith("/") or
                ruta.startswith("//") or "?" in ruta or "#" in ruta or "\\" in ruta or
                SENSIBLE.search(ruta)):
            raise ValueError("la ruta debe ser relativa, sin parámetros ni datos sensibles")
        if not isinstance(pantalla["pasos"], list):
            raise ValueError("pasos debe ser una lista")
        for paso in pantalla["pasos"]:
            if not isinstance(paso, dict) or paso.get("accion") not in {
                    "clic", "esperar", "rellenar", "seleccionar"}:
                raise ValueError("paso no admitido")
            requiere_valor = paso["accion"] in {"rellenar", "seleccionar"}
            claves_esperadas = {"accion", "selector", "valor"} if requiere_valor else {"accion", "selector"}
            if set(paso) != claves_esperadas:
                raise ValueError("campos de paso inválidos")
            _validar_selector(paso["selector"])
            if requiere_valor:
                valor = paso["valor"]
                if (not isinstance(valor, str) or not valor or len(valor) > 200 or
                        SENSIBLE.search(valor)):
                    raise ValueError("el valor de formulario debe ser sintético y no sensible")
        if not isinstance(pantalla["ocultar"], list):
            raise ValueError("ocultar debe ser una lista de selectores")
        for selector in pantalla["ocultar"]:
            _validar_selector(selector)
        marcas = pantalla["marcas"]
        if not isinstance(marcas, list) or not marcas:
            raise ValueError("cada pantalla requiere marcas")
        numeros = set()
        for marca in marcas:
            if (not isinstance(marca, dict) or
                    set(marca) != {"numero", "selector", "texto", "tipo"} or
                    type(marca["numero"]) is not int or marca["numero"] < 1 or
                    marca["numero"] in numeros or marca["tipo"] not in {"recuadro", "flecha"}):
                raise ValueError("marca inválida o número repetido")
            numeros.add(marca["numero"])
            _validar_selector(marca["selector"])
            texto = marca["texto"]
            if not isinstance(texto, str) or not texto.strip() or len(texto) > 180 or SENSIBLE.search(texto):
                raise ValueError("texto de marca vacío, extenso o sensible")
    return datos


def _validar_selector(valor: object) -> None:
    if (not isinstance(valor, str) or not valor.strip() or len(valor) > 240 or
            SENSIBLE.search(valor)):
        raise ValueError("selector vacío, extenso o con datos sensibles")


def _detectar_datos_sensibles(pagina) -> None:
    texto = pagina.locator("body").inner_text(timeout=5000)
    if SENSIBLE.search(texto):
        raise ValueError("la pantalla contiene un correo, DNI o secreto visible; prepare datos sintéticos")


def _certificados_externos(base_url: str, certificado: Path | None,
                           clave: Path | None, frase_env: str | None) -> list[dict]:
    if certificado is None and clave is None and frase_env is None:
        return []
    if certificado is None or clave is None:
        raise ValueError("mTLS requiere certificado y clave juntos")
    if urlsplit(base_url).scheme != "https":
        raise ValueError("mTLS requiere un origen HTTPS local")
    archivos = []
    for ruta in (certificado, clave):
        real = ruta.resolve()
        if real.is_relative_to(RAIZ_REPOSITORIO) or not real.is_file():
            raise ValueError("los archivos mTLS deben existir fuera del repositorio")
        archivos.append(real)
    datos = {"origin": base_url, "certPath": str(archivos[0]), "keyPath": str(archivos[1])}
    if frase_env:
        if not re.fullmatch(r"[A-Z][A-Z0-9_]*", frase_env) or frase_env not in os.environ:
            raise ValueError("variable de frase mTLS ausente o inválida")
        datos["passphrase"] = os.environ[frase_env]
    return [datos]


def _anotar(png: bytes, marcas: list[dict], cajas: list[dict]) -> bytes:
    imagen = Image.open(BytesIO(png)).convert("RGB")
    dibujo = ImageDraw.Draw(imagen)
    fuente = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 17)
    for marca, caja in zip(marcas, cajas, strict=True):
        x, y, ancho, alto = (round(caja[k]) for k in ("x", "y", "width", "height"))
        x2, y2 = x + ancho, y + alto
        if x < 0 or y < 0 or x2 > imagen.width or y2 > imagen.height:
            raise ValueError(f"la marca {marca['numero']} queda fuera del área visible")
        color = "#126B99"
        radio = 15
        cx = max(radio + 2, min(imagen.width - radio - 2, x + 12))
        cy = max(radio + 2, min(imagen.height - radio - 2, y - 18))
        if marca["tipo"] == "recuadro":
            dibujo.rounded_rectangle((x, y, x2, y2), radius=5, outline=color, width=4)
        else:
            tx, ty = x + ancho // 2, y + alto // 2
            dibujo.line((cx, cy, tx, ty), fill=color, width=4)
            dibujo.polygon(((tx, ty), (tx - 9, ty - 4), (tx - 4, ty - 9)), fill=color)
        dibujo.ellipse((cx - radio, cy - radio, cx + radio, cy + radio), fill=color)
        etiqueta = str(marca["numero"])
        bbox = dibujo.textbbox((0, 0), etiqueta, font=fuente)
        dibujo.text((cx - (bbox[2] - bbox[0]) / 2, cy - (bbox[3] - bbox[1]) / 2 - bbox[1]),
                    etiqueta, fill="white", font=fuente)
    buffer = BytesIO()
    imagen.save(buffer, format="PNG", optimize=True)
    return buffer.getvalue()


def capturar(base_url: str, escenario: dict, salida: Path, *, ensayo: bool = False,
             confirmar_sinteticos: bool = False, certificado: Path | None = None,
             clave: Path | None = None, frase_env: str | None = None) -> Path:
    """Navega cada pantalla con Chrome limpio y escribe PNG más manifiesto."""
    base_url = validar_url_local(base_url, origen=True)
    validar_escenario(escenario)
    if not ensayo and not confirmar_sinteticos:
        raise ValueError("confirme que el servidor contiene exclusivamente datos sintéticos")
    certificados = _certificados_externos(base_url, certificado, clave, frase_env)
    if not CHROME.is_file():
        raise RuntimeError(f"Chrome del sistema no existe: {CHROME}")
    try:
        from playwright.sync_api import sync_playwright
    except ImportError as exc:
        raise RuntimeError("falta Python Playwright") from exc

    if salida.exists() and any(salida.iterdir()):
        raise ValueError("el directorio de salida debe estar vacío para evitar mezclar capturas")
    salida.mkdir(parents=True, exist_ok=True)
    registros = []
    with sync_playwright() as playwright:
        navegador = playwright.chromium.launch(executable_path=str(CHROME), headless=True)
        try:
            for pantalla in escenario["pantallas"]:
                for etiqueta, ancho, alto in TAMANOS:
                    contexto = navegador.new_context(viewport={"width": ancho, "height": alto},
                                                    device_scale_factor=1, service_workers="block",
                                                    accept_downloads=False,
                                                    client_certificates=certificados)
                    try:
                        contexto.route("**/*", lambda ruta: (
                            ruta.continue_() if _peticion_local(ruta.request.url, base_url) else ruta.abort()
                        ))
                        contexto.route_web_socket("**/*", lambda conexion: conexion.close())
                        pagina = contexto.new_page()
                        respuesta = pagina.goto(base_url + pantalla["ruta"], wait_until="domcontentloaded", timeout=15000)
                        if respuesta is None or not respuesta.ok:
                            raise RuntimeError(f"no se pudo abrir {pantalla['clave']} ({etiqueta})")
                        pagina.add_style_tag(content=ESTILO_ESTABLE)
                        for paso in pantalla["pasos"]:
                            localizador = pagina.locator(paso["selector"])
                            if paso["accion"] == "clic":
                                localizador.click(timeout=10000)
                            elif paso["accion"] == "esperar":
                                localizador.wait_for(state="visible", timeout=10000)
                            elif paso["accion"] == "rellenar":
                                localizador.fill(paso["valor"], timeout=10000)
                            else:
                                localizador.select_option(value=paso["valor"], timeout=10000)
                        pagina.evaluate("() => document.fonts.ready")
                        _detectar_datos_sensibles(pagina)
                        mascaras = [pagina.locator(OCULTAR_BASE)]
                        mascaras.extend(pagina.locator(selector) for selector in pantalla["ocultar"])
                        cajas = []
                        for marca in pantalla["marcas"]:
                            caja = pagina.locator(marca["selector"]).bounding_box(timeout=10000)
                            if caja is None:
                                raise ValueError(f"selector de marca no visible: {marca['numero']}")
                            cajas.append(caja)
                        png = pagina.screenshot(full_page=False, animations="disabled",
                                                mask=mascaras, mask_color="#273746")
                        anotado = _anotar(png, pantalla["marcas"], cajas)
                        nombre = f"{pantalla['clave']}-{etiqueta}.png"
                        (salida / nombre).write_bytes(anotado)
                        registros.append({"pantalla": pantalla["clave"], "tamano": etiqueta,
                                         "ancho": ancho, "alto": alto, "archivo": nombre,
                                         "estado_http": respuesta.status,
                                         "sha256": hashlib.sha256(anotado).hexdigest(),
                                         "marcas": [{"numero": m["numero"], "texto": m["texto"],
                                                     "tipo": m["tipo"]} for m in pantalla["marcas"]]})
                    finally:
                        contexto.close()
        finally:
            navegador.close()
    manifiesto = salida / "manifiesto.json"
    manifiesto.write_text(json.dumps({"tipo": "ensayo-sintetico" if ensayo else "navegacion-local-sintetica",
                                    "fecha_utc": datetime.now(timezone.utc).isoformat(),
                                    "chrome": str(CHROME), "capturas": registros},
                                   ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return manifiesto


def _peticion_local(url: str, base_url: str) -> bool:
    try:
        if any(c.isspace() for c in url) or "\\" in url:
            return False
        destino = urlsplit(url)
        origen = urlsplit(base_url)
        parametros = parse_qsl(destino.query, keep_blank_values=True)
        if any(re.search(r"token|password|secret|auth|session|credential|clave", nombre, re.I)
               or SENSIBLE.search(valor) for nombre, valor in parametros):
            return False
        return (destino.scheme, destino.hostname, destino.port) == (
            origen.scheme, origen.hostname, origen.port) and not (
                destino.username or destino.password or destino.fragment)
    except ValueError:
        return False


class _PaginaEnsayo(BaseHTTPRequestHandler):
    def do_GET(self):
        contenido = b'<!doctype html><html lang="es"><meta charset="utf-8"><title>Ensayo</title><main><h1>Pagina de ensayo</h1><button id="abrir" onclick="document.getElementById(\'detalle\').hidden=false">Abrir</button><section id="detalle" hidden><h2>Detalle sintetico</h2><p>Contenido de prueba</p></section></main></html>'
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(contenido)))
        self.end_headers()
        self.wfile.write(contenido)

    def log_message(self, *_args):
        pass


def ejecutar_ensayo(salida: Path) -> Path:
    servidor = ThreadingHTTPServer(("127.0.0.1", 0), _PaginaEnsayo)
    hilo = Thread(target=servidor.serve_forever, daemon=True)
    hilo.start()
    escenario = {"pantallas": [{"clave": "ensayo", "ruta": "/", "pasos": [
        {"accion": "clic", "selector": "#abrir"},
        {"accion": "esperar", "selector": "#detalle:not([hidden])"}],
        "marcas": [{"numero": 1, "selector": "h1", "texto": "Título de la página", "tipo": "recuadro"},
                   {"numero": 2, "selector": "#detalle", "texto": "Detalle abierto", "tipo": "flecha"}],
        "ocultar": []}]}
    try:
        return capturar(f"http://127.0.0.1:{servidor.server_port}", escenario, salida, ensayo=True)
    finally:
        servidor.shutdown()
        servidor.server_close()
        hilo.join(timeout=2)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ensayo", action="store_true", help="usa una página sintética local")
    parser.add_argument("--base-url", help="origen HTTP(S) local")
    parser.add_argument("--escenario", type=Path, help="JSON con pantallas y marcas")
    parser.add_argument("--salida", type=Path, required=True)
    parser.add_argument("--datos-sinteticos-confirmados", action="store_true")
    parser.add_argument("--mtls-certificado", type=Path, help="certificado de cliente fuera del repositorio")
    parser.add_argument("--mtls-clave", type=Path, help="clave de cliente fuera del repositorio")
    parser.add_argument("--mtls-frase-env", help="nombre de variable de entorno con la frase de la clave")
    args = parser.parse_args(argv)
    try:
        if args.ensayo:
            if args.base_url or args.escenario or args.mtls_certificado or args.mtls_clave or args.mtls_frase_env:
                parser.error("el ensayo no acepta servidor, escenario ni certificado")
            manifiesto = ejecutar_ensayo(args.salida)
        else:
            if not args.base_url or not args.escenario:
                parser.error("indique --base-url y --escenario")
            escenario = json.loads(args.escenario.read_text(encoding="utf-8"))
            manifiesto = capturar(args.base_url, escenario, args.salida,
                                  confirmar_sinteticos=args.datos_sinteticos_confirmados,
                                  certificado=args.mtls_certificado, clave=args.mtls_clave,
                                  frase_env=args.mtls_frase_env)
    except (OSError, ValueError, RuntimeError, json.JSONDecodeError) as exc:
        print(f"Error: {exc}", file=sys.stderr)
        return 1
    except Exception:
        print("Error: falló la navegación o captura local; revise el escenario y el servidor", file=sys.stderr)
        return 1
    print(manifiesto)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
