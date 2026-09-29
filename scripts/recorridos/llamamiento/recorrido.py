#!/usr/bin/env python3
"""Guion supervisado para un llamamiento sintético en Chrome del sistema.

El operador realiza cada acción en la interfaz. Este proceso observa la respuesta
HTTP y comprueba recibos; nunca construye peticiones de efecto por su cuenta.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
from pathlib import Path
from urllib.parse import urlsplit


RUTAS = {
    "seleccion": ("rrhh", "/api/vec/contratacion-temporal/llamamientos/seleccion"),
    "comunicacion": ("rrhh", "/api/vec/contratacion-temporal/llamamientos/comunicaciones"),
    "candidato_respuesta": ("candidato", "/api/vec/bolsa/mi-bolsa/respuestas"),
    "declaracion_rrhh": ("rrhh", "/api/vec/contratacion-temporal/llamamientos/respuestas/registro"),
    "resolucion_rrhh": ("rrhh", "/api/vec/contratacion-temporal/llamamientos/resoluciones"),
    "siguiente": ("rrhh", "/api/vec/contratacion-temporal/llamamientos/siguientes"),
}
BASE = tuple(RUTAS)[:-1]
FECHA_ETAPA = {
    "seleccion": "confirmada_en", "comunicacion": "registrada_en",
    "candidato_respuesta": "registrada_en", "declaracion_rrhh": "registrada_en",
    "resolucion_rrhh": "resuelta_en", "siguiente": "confirmada_en",
}
ESTADO_HTTP_INICIAL = {
    "seleccion": 200,  # Selección responde 200 tanto al abrir como al recuperar.
    "comunicacion": 201, "candidato_respuesta": 201,
    "declaracion_rrhh": 201, "resolucion_rrhh": 201, "siguiente": 201,
}


class NoEjecutado(ValueError):
    """Falta una condición previa; no se abre el navegador."""


class FalloRecorrido(RuntimeError):
    """El recorrido empezó y una comprobación falló."""


def leer_json(ruta: Path) -> dict:
    try:
        valor = json.loads(ruta.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        raise NoEjecutado("falta un escenario o manifiesto local válido") from error
    if not isinstance(valor, dict):
        raise NoEjecutado("el escenario o manifiesto no es un objeto")
    return valor


def sha256(ruta: Path) -> str:
    resumen = hashlib.sha256()
    with ruta.open("rb") as entrada:
        for bloque in iter(lambda: entrada.read(1024 * 1024), b""):
            resumen.update(bloque)
    return resumen.hexdigest()


def fichero_externo(ruta: str, raiz: Path) -> Path:
    if not isinstance(ruta, str):
        raise NoEjecutado("falta la ruta de un archivo local")
    valor = Path(ruta)
    if not valor.is_absolute() or not valor.is_file() or valor.stat().st_size == 0:
        raise NoEjecutado("falta un archivo local absoluto y no vacío")
    repositorio_compartido = raiz.parent.parent if raiz.parent.name == ".worktrees" else raiz
    if valor.resolve().is_relative_to(repositorio_compartido):
        raise NoEjecutado("el material del escenario debe estar fuera de Git")
    return valor


def comprobar_entrada(escenario: dict, raiz: Path, *, observar: bool = False) -> dict:
    origen = escenario.get("origen", "")
    if not isinstance(origen, str):
        raise NoEjecutado("falta el origen HTTPS local")
    try:
        u = urlsplit(origen)
        puerto = u.port
    except ValueError as error:
        raise NoEjecutado("puerto local no válido") from error
    if (u.scheme != "https" or u.hostname not in {"localhost", "127.0.0.1", "::1"}
            or not puerto or u.username or u.password or u.path or u.query or u.fragment):
        raise NoEjecutado("el origen debe ser HTTPS loopback, sin ruta ni credenciales")
    if escenario.get("clon") != "H3-H5":
        raise NoEjecutado("falta el clon aislado H3-H5")
    manifiesto = leer_json(fichero_externo(escenario.get("manifiesto_clon", ""), raiz))
    if (manifiesto.get("origen") != origen or manifiesto.get("clon") != "H3-H5"
            or manifiesto.get("hitos_instalados") != ["H3", "H4", "H5"]
            or manifiesto.get("datos") != "sinteticos" or manifiesto.get("listo") is not True):
        raise NoEjecutado("el manifiesto externo no acredita H3-H5 para este origen")
    binario = fichero_externo(escenario.get("binario", ""), raiz)
    huella = escenario.get("binario_sha256", "")
    if (not isinstance(huella, str) or len(huella) != 64 or sha256(binario) != huella
            or manifiesto.get("binario_sha256") != huella):
        raise NoEjecutado("el binario no coincide con el manifiesto del clon")
    if not os.access(binario, os.X_OK):
        raise NoEjecutado("el binario no es ejecutable")
    identidades = escenario.get("identidades", {})
    if not isinstance(identidades, dict) or set(identidades) != {"rrhh", "candidato"}:
        raise NoEjecutado("faltan las dos identidades independientes")
    certificados = []
    for actor in ("rrhh", "candidato"):
        identidad = identidades[actor]
        if not isinstance(identidad, dict):
            raise NoEjecutado("identidad incompleta")
        certificados.append(fichero_externo(identidad.get("certificado", ""), raiz))
        fichero_externo(identidad.get("clave", ""), raiz)
        ruta = identidad.get("ruta", "")
        if not isinstance(ruta, str) or not ruta.startswith(
            "/portal-empleado/" if actor == "rrhh" else "/area-personal/"
        ) or "//" in ruta or "#" in ruta:
            raise NoEjecutado("la ruta de una identidad no pertenece a su portal")
    if sha256(certificados[0]) == sha256(certificados[1]):
        raise NoEjecutado("RRHH y candidato usan el mismo certificado")
    # La apertura inicial no produce recibos: su ausencia no tapa un fallo HTTP.
    if observar:
        return escenario
    respuesta = escenario.get("respuesta")
    if respuesta not in {"aceptacion", "renuncia"}:
        raise NoEjecutado("la respuesta debe ser aceptación o renuncia")
    etapas = escenario.get("etapas")
    esperadas = list(BASE) + (["siguiente"] if respuesta == "renuncia" else [])
    if not isinstance(etapas, list) or any(not isinstance(e, dict) for e in etapas) or [e.get("nombre") for e in etapas] != esperadas:
        raise NoEjecutado("la secuencia no coincide con el circuito esperado")
    for etapa in etapas:
        campos = etapa.get("campos_recibo")
        if (not isinstance(campos, dict) or not campos
                or any(not isinstance(k, str) or not isinstance(v, (str, int)) or v == ""
                       for k, v in campos.items())
                or FECHA_ETAPA[etapa["nombre"]] not in campos
                or not any(k in campos for k in ("recibo", "recibo_ref", "recibo_local_ref", "justificante_ref"))):
            raise NoEjecutado("falta la referencia, fecha o versión esperada de un recibo")
    if not isinstance(escenario.get("bolsa_ref"), str) or not escenario["bolsa_ref"]:
        raise NoEjecutado("falta la referencia de Bolsa sintética")
    return escenario


def comprobar_recibo(cuerpo: dict, esperados: dict) -> None:
    datos = cuerpo.get("data") if isinstance(cuerpo, dict) else None
    if not isinstance(datos, dict):
        raise FalloRecorrido("la respuesta no contiene un recibo de datos")
    for campo, valor in esperados.items():
        if datos.get(campo) != valor:
            raise FalloRecorrido("el recibo no coincide con la evidencia local esperada")


def estado_http_esperado(nombre: str, recuperar: bool) -> int:
    return 200 if recuperar else ESTADO_HTTP_INICIAL[nombre]


def comprobar_navegador(contexto, pagina) -> None:
    estado = pagina.evaluate("""() => ({
        cliente: document.documentElement.clientWidth,
        contenido: document.documentElement.scrollWidth,
        local: localStorage.length, sesion: sessionStorage.length
    })""")
    if contexto.cookies() or estado["local"] or estado["sesion"]:
        raise FalloRecorrido("hay cookies o almacenamiento web")
    if estado["contenido"] > estado["cliente"]:
        raise FalloRecorrido("hay desbordamiento horizontal")


def mismo_origen(url: str, origen: str) -> bool:
    """Exige esquema, host y puerto exactos, sin usuario ni contraseña."""
    try:
        destino, autorizado = urlsplit(url), urlsplit(origen)
        return (destino.scheme, destino.hostname, destino.port) == (
            autorizado.scheme, autorizado.hostname, autorizado.port
        ) and not destino.username and not destino.password
    except ValueError:
        return False


def servir_solo_origen(route, origen: str) -> None:
    """Cierra la red antes del efecto, incluidos saltos HTTP del servidor local."""
    if not mismo_origen(route.request.url, origen):
        route.abort("blockedbyclient")
        return
    try:
        respuesta = route.fetch(max_redirects=0)
    except Exception:
        route.abort("failed")
        return
    if not mismo_origen(respuesta.url, origen) or 300 <= respuesta.status < 400:
        route.abort("blockedbyclient")
        return
    route.fulfill(response=respuesta)


def ejecutar(escenario: dict, recuperar: bool = False) -> None:
    try:
        from playwright.sync_api import sync_playwright
    except ImportError as error:
        raise NoEjecutado("falta Playwright Python local") from error
    origen = escenario["origen"]
    with sync_playwright() as playwright:
        navegador = playwright.chromium.launch(headless=False, executable_path="/usr/bin/google-chrome")
        try:
            contextos = {}
            paginas = {}
            errores = {"rrhh": [], "candidato": []}
            observadas = {"rrhh": [], "candidato": []}
            cookies_set = {"rrhh": [], "candidato": []}
            externas = {"rrhh": [], "candidato": []}
            for actor, identidad in escenario["identidades"].items():
                contexto = navegador.new_context(
                    client_certificates=[{"origin": origen, "certPath": identidad["certificado"],
                                          "keyPath": identidad["clave"]}],
                    ignore_https_errors=False, service_workers="block", locale="es-ES",
                    timezone_id="Europe/Madrid", viewport={"width": 1440, "height": 900},
                )
                contexto.route("**/*", lambda route: servir_solo_origen(route, origen))
                contexto.route_web_socket("**/*", lambda websocket: websocket.close())
                pagina = contexto.new_page()
                pagina.on("pageerror", lambda error, actor=actor: errores[actor].append(str(error)))
                pagina.on("request", lambda r, actor=actor: externas[actor].append(1)
                          if urlsplit(r.url).scheme in {"http", "https"}
                          and urlsplit(r.url).netloc != urlsplit(origen).netloc else None)
                def observar(r, actor=actor):
                    if urlsplit(r.url).netloc != urlsplit(origen).netloc:
                        return
                    if "set-cookie" in r.headers:
                        cookies_set[actor].append(1)
                    if r.request.method in {"POST", "GET"}:
                        observadas[actor].append(r)
                pagina.on("response", observar)
                contextos[actor], paginas[actor] = contexto, pagina
                r = pagina.goto(origen + identidad["ruta"], wait_until="networkidle")
                if r is None or r.status != 200:
                    raise FalloRecorrido("uno de los portales no abrió con su identidad")
                comprobar_navegador(contexto, pagina)
                if cookies_set[actor] or externas[actor]:
                    raise FalloRecorrido("el servidor emitió Set-Cookie o la página llamó a otro origen")
            for etapa in escenario["etapas"]:
                nombre = etapa["nombre"]
                actor, ruta = RUTAS[nombre]
                if recuperar and nombre == "candidato_respuesta":
                    lecturas = [r for r in observadas["candidato"] if
                                r.request.method == "GET" and urlsplit(r.url).path == "/api/vec/bolsa/mi-bolsa"]
                    if not lecturas or lecturas[-1].status != 200:
                        raise FalloRecorrido("la lectura propia de Bolsa no se recuperó")
                    cuerpo = lecturas[-1].json()
                    portal = cuerpo.get("data", {}).get("portal", [])
                    propios = [p for p in portal if isinstance(p, dict)
                               and p.get("bolsa") == escenario["bolsa_ref"]]
                    if len(propios) != 1 or not isinstance(propios[0].get("ultima_respuesta"), dict):
                        raise FalloRecorrido("la respuesta propia no aparece tras reiniciar")
                    ultimo = propios[0]["ultima_respuesta"]
                    esperados = etapa["campos_recibo"]
                    if (ultimo.get("recibo") != esperados.get("recibo")
                            or ultimo.get("respondida_en") != esperados.get("registrada_en")):
                        raise FalloRecorrido("el recibo o fecha propios cambiaron tras reiniciar")
                    print("candidato_respuesta: GET propio conservó recibo y fecha.", flush=True)
                    continue
                observadas[actor].clear()
                accion = "recupere con la misma clave" if recuperar else "complete una sola vez"
                print(f"{nombre}: {accion} la acción en la pestaña {actor}; pulse Intro al ver su recibo.", flush=True)
                input()
                coincidentes = [r for r in observadas[actor] if
                                r.request.method == "POST" and urlsplit(r.url).path == ruta]
                esperado_http = estado_http_esperado(nombre, recuperar)
                if len(coincidentes) != 1 or coincidentes[0].status != esperado_http:
                    raise FalloRecorrido("la etapa no produjo un único POST con el estado esperado")
                comprobar_recibo(coincidentes[0].json(), etapa["campos_recibo"])
                if errores[actor] or cookies_set[actor] or externas[actor]:
                    raise FalloRecorrido("el navegador registró JavaScript erróneo, Set-Cookie o tráfico externo")
                comprobar_navegador(contextos[actor], paginas[actor])
                print(f"{nombre}: recibo comprobado; no se guardan sus datos.", flush=True)
            for actor, pagina in paginas.items():
                pagina.set_viewport_size({"width": 390, "height": 844})
                comprobar_navegador(contextos[actor], pagina)
                if errores[actor] or cookies_set[actor] or externas[actor]:
                    raise FalloRecorrido("la vista móvil registró un error o tráfico externo")
            print("RECUPERACIÓN OBSERVADA tras reinicio." if recuperar else
                  "RECORRIDO OBSERVADO. Reinicie aplicación y PostgreSQL; ejecute --recuperar.")
        finally:
            navegador.close()


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--escenario", required=True, type=Path, help="JSON absoluto, fuera de Git")
    modo = p.add_mutually_exclusive_group()
    modo.add_argument("--comprobar", action="store_true", help="valida entradas, sin abrir Chrome")
    modo.add_argument("--ejecutar", action="store_true", help="abre Chrome y observa las acciones del operador")
    modo.add_argument("--recuperar", action="store_true", help="tras reinicio: comprueba GET propio y replays RRHH")
    modo.add_argument("--observar", action="store_true")
    p.add_argument("--evidencias", type=Path)
    a = p.parse_args()
    raiz = Path(__file__).resolve().parents[3]
    try:
        escenario = comprobar_entrada(leer_json(fichero_externo(str(a.escenario), raiz)), raiz,
                                     observar=a.observar)
        if a.observar:
            from observar import observar_apertura
            return observar_apertura(escenario, a.evidencias)
        if a.comprobar:
            print("PREPARADO: entradas locales verificadas; NO EJECUTADO en navegador")
            return 0
        if not a.ejecutar and not a.recuperar:
            raise NoEjecutado("indique --comprobar, --ejecutar o --recuperar")
        ejecutar(escenario, recuperar=a.recuperar)
        return 0
    except NoEjecutado as error:
        print(f"NO EJECUTADO: {error}", file=sys.stderr)
        return 3
    except FalloRecorrido as error:
        print(f"FALLO DEL RECORRIDO: {error}", file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print("FALLO DEL RECORRIDO: interrumpido", file=sys.stderr)
        return 1
    except Exception as error:
        print(f"FALLO DEL RECORRIDO: {error.__class__.__name__}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
