#!/usr/bin/env python3
"""Prueba la presentación completa desde un contenedor sin dependencias externas."""

from __future__ import annotations

import ipaddress
import json
import os
import struct
import sys
from urllib.error import HTTPError
from urllib.parse import urlparse
from urllib.request import Request, urlopen


TIEMPO_ESPERA = 25


def base_autorizada() -> str:
    valor = os.environ.get("VEC_PRESENTACION_BASE_URL", "").strip().rstrip("/")
    analizada = urlparse(valor)
    if analizada.scheme != "http" or not analizada.hostname or analizada.path not in {"", "/"}:
        raise RuntimeError("VEC_PRESENTACION_BASE_URL debe ser una URL HTTP interna canónica")
    if analizada.username or analizada.password or analizada.query or analizada.fragment:
        raise RuntimeError("la URL de presentación no admite estado ambiental")
    try:
        direccion = ipaddress.ip_address(analizada.hostname)
        puerto = analizada.port
    except ValueError as error:
        raise RuntimeError("la URL de presentación exige una IP literal y puerto válido") from error
    if puerto is None or not (direccion.is_private or direccion.is_loopback):
        raise RuntimeError("la prueba solo admite una IP privada o de loopback")
    return valor


def solicitar(
    base: str,
    ruta: str,
    *,
    metodo: str = "GET",
    cuerpo: bytes | None = None,
    cabeceras_extra: dict[str, str] | None = None,
) -> tuple[int, object, bytes]:
    cabeceras = {"Accept": "application/json"}
    if cuerpo is not None:
        cabeceras["Content-Type"] = "application/json; charset=utf-8"
    cabeceras.update(cabeceras_extra or {})
    peticion = Request(base + ruta, data=cuerpo, headers=cabeceras, method=metodo)
    try:
        with urlopen(peticion, timeout=TIEMPO_ESPERA) as respuesta:
            return respuesta.status, respuesta.headers, respuesta.read(9 * 1024 * 1024)
    except HTTPError as error:
        return error.code, error.headers, error.read(64 * 1024)


def exigir(condicion: bool, mensaje: str) -> None:
    if not condicion:
        raise RuntimeError(mensaje)


def main() -> int:
    base = base_autorizada()
    estado, cabeceras, portada = solicitar(
        base, "/portal-empleado/?presentacion=rrhh&perfil=administrador",
    )
    exigir(estado == 200, f"portal HTTP {estado}")
    exigir(cabeceras.get("X-Vec-Modo-Presentacion") == "aislada-sintetica-v1", "falta marca de presentación")
    exigir(b"Presentaci" in portada and b"portal" in portada.lower(), "portada RRHH inesperada")

    estado, _, convocatorias = solicitar(base, "/api/publico/bolsa/convocatorias")
    exigir(estado == 200, f"consulta pública HTTP {estado}")
    exigir(b"vec.bolsa.publico.convocatorias.v1" in convocatorias, "contrato público inesperado")

    estado, cabeceras_pdf, bases_pdf = solicitar(
        base, "/bolsa/documentos/bases-operario-demo.pdf",
        cabeceras_extra={"Accept": "application/pdf"},
    )
    exigir(estado == 200, f"bases PDF HTTP {estado}")
    exigir(cabeceras_pdf.get_content_type() == "application/pdf", "bases sin tipo PDF")
    exigir(bases_pdf.startswith(b"%PDF-"), "bases sin firma PDF")

    cuerpo = json.dumps({
        "coordinates": [
            {"lat": 37.17428891, "lon": -3.59869101, "name": "Granada"},
            {"lat": 36.74535308, "lon": -3.52045559, "name": "Motril"},
        ],
        "alternatives": 1,
    }, separators=(",", ":")).encode("utf-8")
    estado, cabeceras, contenido = solicitar(
        base, "/api/presentacion/cartografia/rutas", metodo="POST", cuerpo=cuerpo,
    )
    exigir(estado == 200, f"mediador HTTP {estado}: {contenido[:200]!r}")
    exigir(not cabeceras.get("Set-Cookie"), "el mediador emitió una cookie")
    exigir(not cabeceras.get("Access-Control-Allow-Origin"), "el mediador emitió CORS")
    respuesta = json.loads(contenido.decode("utf-8", errors="strict"))
    rutas = respuesta.get("routes")
    exigir(respuesta.get("code") == "Ok", "OSRM no confirmó la ruta")
    exigir(respuesta.get("engine") == "osrm_on_premise", "motor cartográfico inesperado")
    exigir(respuesta.get("route_scope") == "Granada provincia + 15 km", "ámbito inesperado")
    version = respuesta.get("data_version")
    exigir(isinstance(version, str) and 8 <= len(version) <= 100, "versión de grafo ausente")
    exigir(isinstance(rutas, list) and len(rutas) == 1, "alternativas inesperadas")
    ruta = rutas[0]
    exigir(60_000 < ruta.get("distance", 0) < 90_000, "distancia Granada-Motril inesperada")
    exigir(len(ruta.get("legs", [])) == 1, "tramos inesperados")
    geometria = ruta.get("geometry", {})
    exigir(geometria.get("type") == "LineString", "geometría no lineal")
    exigir(len(geometria.get("coordinates", [])) > 100, "geometría real insuficiente")

    estado, cabeceras, png = solicitar(base, "/tiles/osm/14/8028/6367.png")
    exigir(estado == 200, f"tesela HTTP {estado}")
    exigir(cabeceras.get_content_type() == "image/png", "la tesela no es PNG")
    exigir(len(png) > 8 and png[:8] == struct.pack(">Q", 0x89504E470D0A1A0A), "firma PNG inválida")
    exigir("no-store" in cabeceras.get("Cache-Control", "").lower(), "la tesela permite caché de localización")
    exigir(not cabeceras.get("ETag") and not cabeceras.get("Last-Modified"), "la tesela conserva validadores de caché")
    exigir(cabeceras.get("Cross-Origin-Resource-Policy") == "same-origin", "la tesela no limita su origen")
    exigir(cabeceras.get("X-Content-Type-Options") == "nosniff", "la tesela permite inferencia de tipo")

    estado_get, _, _ = solicitar(base, "/api/presentacion/cartografia/rutas")
    estado_zoom, _, _ = solicitar(base, "/tiles/osm/15/16056/12734.png")
    estado_privado, _, _ = solicitar(base, "/api/vec/session")
    estado_lanzador, cabeceras_lanzador, cuerpo_lanzador = solicitar(
        base, "/presentacion/",
    )
    estado_cookie, cabeceras_cookie, cuerpo_cookie = solicitar(
        base, "/presentacion/", cabeceras_extra={"Cookie": "sesion=no-admitida"},
    )
    exigir(estado_get == 403, f"GET cartográfico inesperado: {estado_get}")
    exigir(estado_zoom == 404, f"zoom no autorizado inesperado: {estado_zoom}")
    exigir(estado_privado == 404, f"API privada inesperada: {estado_privado}")
    exigir(estado_lanzador == 200, f"lanzador inesperado: {estado_lanzador}")
    exigir(estado_cookie == 200, f"cookie ambiental no neutralizada: {estado_cookie}")
    exigir(cuerpo_cookie == cuerpo_lanzador, "una cookie ambiental alteró el lanzador")
    exigir(not cabeceras_lanzador.get("Set-Cookie"), "el lanzador emitió una cookie")
    exigir(not cabeceras_cookie.get("Set-Cookie"), "el borde respondió con una cookie")

    print("Smoke de cartografía real de la presentación superado.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (RuntimeError, json.JSONDecodeError, UnicodeError, OSError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        raise SystemExit(1)
