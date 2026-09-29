#!/usr/bin/env python3
"""Recorrido sintético centro → ratificación → alta RRHH en un clon aislado.

No prepara ni instala bases. El archivo de acreditación y los certificados se
proporcionan fuera de Git. La salida deliberadamente no incluye datos de la
petición, certificados, cuerpos HTTP ni recibos completos.
"""

from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
import subprocess
import sys
from datetime import date, timedelta
from pathlib import Path
from urllib.parse import urlparse


RUTA_CENTRO = "/portal-empleado/peticiones-centro/"
RUTA_RRHH = "/api/vec/contratacion-temporal/peticiones-centro/rrhh"
RUTA_OPERACIONES = "/api/vec/contratacion-temporal/peticiones-centro/operaciones"
RUTA_CONTEXTO = "/api/vec/contratacion-temporal/peticiones-centro/contexto"
RUTA_BANDEJA = "/api/vec/contratacion-temporal/peticiones-centro/bandeja"
RUTA_PREFERENCIAS = "/api/vec/usuarios/mis-preferencias"


class NoEjecutado(Exception):
    """Falta una condición previa; no se abrió el navegador ni hubo escrituras."""


def origen_local(valor: str) -> str:
    try:
        url = urlparse(valor)
        valido = (url.scheme == "https" and url.hostname in {"127.0.0.1", "localhost", "::1"}
                  and bool(url.port) and not url.username and not url.password
                  and url.path in {"", "/"} and not url.query and not url.fragment)
    except ValueError:
        valido = False
    if not valido:
        raise NoEjecutado("--origen debe ser un origen HTTPS de loopback con puerto, sin credenciales ni ruta")
    return valor.rstrip("/")


def sha256_archivo(ruta: Path) -> str:
    h = hashlib.sha256()
    with ruta.open("rb") as archivo:
        for bloque in iter(lambda: archivo.read(1024 * 1024), b""):
            h.update(bloque)
    return h.hexdigest()


def preflight(args: argparse.Namespace) -> str:
    origen = origen_local(args.origen or "")
    if not args.acreditacion:
        raise NoEjecutado("falta --acreditacion: JSON externo que identifique clon aislado con H3, H4 y H5 instalados")
    try:
        prueba = json.loads(Path(args.acreditacion).read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        raise NoEjecutado("--acreditacion no es un JSON local legible") from exc
    if (prueba.get("tipo") != "clon_aislado_sintetico" or prueba.get("origen") != origen
            or not isinstance(prueba.get("clon_id"), str) or not prueba["clon_id"]
            or any(prueba.get(hito) is not True for hito in ("h3_instalado", "h4_instalado", "h5_instalado"))):
        raise NoEjecutado("la acreditación no confirma origen local, clon sintético aislado y H3–H5 instalados")
    if not args.binario or not Path(args.binario).is_file() or not os.access(args.binario, os.X_OK):
        raise NoEjecutado("falta --binario del servidor que escucha el origen local")
    if prueba.get("binario_sha256") != sha256_archivo(Path(args.binario)):
        raise NoEjecutado("el SHA256 del binario no coincide con la acreditación H3–H5")
    if not args.reinicio or not Path(args.reinicio).is_file() or not os.access(args.reinicio, os.X_OK):
        raise NoEjecutado("falta --reinicio: ejecutable externo que reinicie solo aplicación y PostgreSQL del clon")
    if not Path("/usr/bin/google-chrome").is_file():
        raise NoEjecutado("falta /usr/bin/google-chrome del sistema")
    if importlib.util.find_spec("playwright") is None:
        raise NoEjecutado("falta Playwright para Python en este entorno")
    for rol in ("solicitante", "ratificador", "rrhh"):
        cert, clave = getattr(args, f"cert_{rol}"), getattr(args, f"clave_{rol}")
        if not cert or not clave or not Path(cert).is_file() or not Path(clave).is_file():
            raise NoEjecutado(f"faltan certificado y clave mTLS externos para {rol}")
    huellas_certificados(*(getattr(args, f"cert_{rol}") for rol in ("solicitante", "ratificador", "rrhh")))
    return origen


def huellas_certificados(*rutas: str) -> tuple[str, ...]:
    if len(rutas) != 3 or any(not ruta or not Path(ruta).is_file() for ruta in rutas):
        raise NoEjecutado("faltan los tres certificados mTLS externos")
    huellas = tuple(sha256_archivo(Path(ruta)) for ruta in rutas)
    if len(set(huellas)) != 3:
        raise NoEjecutado("los tres certificados mTLS deben tener contenido distinto")
    return huellas


def exigir_tres_actores(solicitante: str, ratificador: str, rrhh: str) -> None:
    if not all(isinstance(actor, str) and actor for actor in (solicitante, ratificador, rrhh)) \
            or len({solicitante, ratificador, rrhh}) != 3:
        raise AssertionError("solicitante, ratificador y RRHH no son tres identidades autenticadas distintas")


def datos(respuesta, ruta: str, metodo: str = "GET") -> dict:
    if respuesta.request.method != metodo or urlparse(respuesta.url).path != ruta or respuesta.status not in (200, 201):
        raise AssertionError(f"{metodo} {ruta}: respuesta inesperada")
    cuerpo = respuesta.json()
    if not isinstance(cuerpo, dict) or not isinstance(cuerpo.get("data"), dict):
        raise AssertionError(f"{metodo} {ruta}: falta data JSON")
    return cuerpo["data"]


def verificar_recibo_centro(recibo: dict, peticion: str, version: int, estado: str, actor: str) -> tuple:
    if (recibo.get("peticion_ref") != peticion or recibo.get("version") != version
            or recibo.get("estado") != estado or recibo.get("actor_ref") != actor
            or not recibo.get("recibo_ref") or not recibo.get("registrado_en")
            or recibo.get("estado_local") not in {"registrado", "replay_confirmado"}):
        raise AssertionError("recibo de centro incompatible con petición, actor, estado o versión")
    return recibo["recibo_ref"], recibo["registrado_en"]


def verificar_entrega(entrega: dict, peticion: str) -> tuple:
    recibo = entrega.get("recibo_alta") or {}
    if (entrega.get("peticion", {}).get("referencia") != peticion
            or entrega.get("estado_entrega") != "confirmada"
            or not recibo.get("expediente_ref") or not recibo.get("numero_visible")
            or recibo.get("version") != 1 or not recibo.get("recibo_ref")
            or not recibo.get("auditoria_ref") or not recibo.get("evento_ref")
            or not recibo.get("confirmada_en")):
        raise AssertionError("el alta RRHH no está confirmada con recibo completo")
    return tuple(recibo[campo] for campo in (
        "expediente_ref", "numero_visible", "version", "recibo_ref",
        "auditoria_ref", "evento_ref", "confirmada_en"))


def datos_api(respuesta, ruta: str, metodo: str = "GET") -> dict:
    if urlparse(respuesta.url).path != ruta or respuesta.status not in (200, 201):
        raise AssertionError(f"{metodo} {ruta}: respuesta inesperada")
    cuerpo = respuesta.json()
    if not isinstance(cuerpo, dict) or not isinstance(cuerpo.get("data"), dict):
        raise AssertionError(f"{metodo} {ruta}: falta data JSON")
    return cuerpo["data"]


def mismo_origen(url: str, origen: str) -> bool:
    try:
        destino, permitido = urlparse(url), urlparse(origen)
        return (destino.scheme == permitido.scheme and destino.hostname == permitido.hostname
                and destino.port == permitido.port and not destino.username and not destino.password)
    except ValueError:
        return False


def interceptar_ruta(ruta, origen: str) -> None:
    if not mismo_origen(ruta.request.url, origen):
        ruta.abort()
        return
    try:
        respuesta = ruta.fetch(max_redirects=0)
        if not mismo_origen(respuesta.url, origen) or 300 <= respuesta.status < 400:
            ruta.abort()
            return
        ruta.fulfill(response=respuesta)
    except Exception:
        ruta.abort()


def bloquear_websocket(_ruta) -> None:
    # El socket interceptado queda sin conectar: no se llama connect_to_server.
    # close() dentro del callback síncrono de Playwright bloquea su despachador.
    return


def contexto(browser, origen: str, cert: str, clave: str):
    # Playwright usa Chrome del sistema. El certificado se limita al origen local.
    ctx = browser.new_context(
        viewport={"width": 1440, "height": 900},
        client_certificates=[{"origin": origen, "certPath": cert, "keyPath": clave}],
        # Chrome confía en la CA instalada en el sistema. No se desactiva TLS.
        ignore_https_errors=False,
        service_workers="block",
    )
    ctx.route("**/*", lambda ruta: interceptar_ruta(ruta, origen))
    ctx.route_web_socket("**/*", bloquear_websocket)
    return ctx


def respuesta_de(page, ruta: str, metodo: str, accion):
    with page.expect_response(lambda r: urlparse(r.url).path == ruta and r.request.method == metodo, timeout=20_000) as espera:
        accion()
    return espera.value


def actor_y_bandeja(page, origen: str) -> tuple[dict, dict]:
    respuestas = {}
    def guardar(r):
        ruta = urlparse(r.url).path
        if ruta in {RUTA_CONTEXTO, RUTA_BANDEJA} and r.request.method == "GET":
            respuestas[ruta] = r
    page.on("response", guardar)
    page.goto(origen + RUTA_CENTRO, wait_until="domcontentloaded")
    page.locator("#aplicacion [data-accion]").first.wait_for(timeout=20_000)
    page.remove_listener("response", guardar)
    if RUTA_CONTEXTO not in respuestas or RUTA_BANDEJA not in respuestas:
        raise AssertionError("la interfaz no cargó contexto y bandeja autorizados")
    return datos(respuestas[RUTA_CONTEXTO], RUTA_CONTEXTO), datos(respuestas[RUTA_BANDEJA], RUTA_BANDEJA)


def primer_valor(page, nombre: str):
    opciones = page.locator(f'[name="{nombre}"] option').evaluate_all(
        "items => items.map(x => x.value).filter(Boolean)")
    if not opciones:
        raise AssertionError(f"catálogo sin opción válida: {nombre}")
    page.locator(f'[name="{nombre}"]').select_option(opciones[0])


def recorrer(args: argparse.Namespace, origen: str) -> None:
    from playwright.sync_api import sync_playwright

    with sync_playwright() as pw:
        browser = pw.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
        try:
            contextos = {rol: contexto(browser, origen, getattr(args, f"cert_{rol}"),
                                      getattr(args, f"clave_{rol}"))
                        for rol in ("solicitante", "ratificador", "rrhh")}
            paginas = {rol: ctx.new_page() for rol, ctx in contextos.items()}
            errores_js = []
            for pagina in paginas.values():
                pagina.on("pageerror", lambda err: errores_js.append(type(err).__name__))

            # La preferencia propia identifica al principal RRHH autenticado.
            # Se consulta antes de cualquier escritura del centro.
            identidad = contextos["rrhh"].request.get(origen + RUTA_PREFERENCIAS, max_redirects=0)
            vista_rrhh = datos_api(identidad, RUTA_PREFERENCIAS)
            estado_rrhh = vista_rrhh.get("estado")
            actor_rrhh = estado_rrhh.get("persona_ref") if isinstance(estado_rrhh, dict) else None
            if not actor_rrhh:
                raise AssertionError("la consulta propia de RRHH no acredita un principal")

            sol, _ = actor_y_bandeja(paginas["solicitante"], origen)
            actor_sol = sol.get("actor", {})
            if actor_sol.get("puede_presentar") is not True or actor_sol.get("puede_ratificar") is not False:
                raise AssertionError("certificado solicitante sin perfil exclusivo")
            previo_rat = contextos["ratificador"].request.get(origen + RUTA_CONTEXTO, max_redirects=0)
            actor_rat_previo = datos_api(previo_rat, RUTA_CONTEXTO).get("actor", {})
            if actor_rat_previo.get("puede_ratificar") is not True or actor_rat_previo.get("puede_presentar") is not False:
                raise AssertionError("certificado ratificador sin perfil exclusivo")
            exigir_tres_actores(actor_sol["referencia"], actor_rat_previo.get("referencia"), actor_rrhh)
            p = paginas["solicitante"]
            p.locator('[data-accion="nueva"]').click()
            for campo in ("contacto_ref", "categoria_ref", "grupo_subgrupo", "motivo_clave"):
                primer_valor(p, campo)
            p.locator('[name="detalle"]').fill("Necesidad sintética para comprobar el puente centro a RRHH.")
            inicio = date.today() + timedelta(days=30)
            p.locator('[name="inicio"]').fill(inicio.isoformat())
            p.locator('[name="fin"]').fill((inicio + timedelta(days=30)).isoformat())
            p.locator('[name="rc_existe"][value="no"]').check()
            p.locator('[data-ct-form] button[type="submit"]').click()
            p.locator('[data-ct-accion="confirmar"]').wait_for()
            r = respuesta_de(p, RUTA_OPERACIONES, "POST", lambda: p.locator('[data-ct-accion="confirmar"]').click())
            comando_sol = r.request.post_data_json
            alta_centro = datos(r, RUTA_OPERACIONES, "POST")
            peticion = alta_centro.get("peticion_ref")
            recibo_sol = verificar_recibo_centro(alta_centro, peticion, 1, "pendiente_ratificacion", actor_sol["referencia"])
            if not peticion or not peticion.startswith("peticion:centro:"):
                raise AssertionError("referencia de petición inválida")

            rat, bandeja_rat = actor_y_bandeja(paginas["ratificador"], origen)
            actor_rat = rat.get("actor", {})
            if (actor_rat.get("puede_ratificar") is not True or actor_rat.get("puede_presentar") is not False
                    or actor_rat.get("referencia") != actor_rat_previo["referencia"]):
                raise AssertionError("ratificador no es una identidad separada")
            if not any(x.get("referencia") == peticion and x.get("version") == 1 for x in bandeja_rat.get("peticiones", [])):
                raise AssertionError("petición v1 ausente de la bandeja del ratificador")
            p = paginas["ratificador"]
            p.locator(f'[data-seleccionar="{peticion}"]').click()
            p.locator('[data-accion="abrir-ratificacion"]').click()
            p.locator('[name="motivo_ratificacion"]').fill("Ratificación sintética tras revisar la necesidad.")
            p.locator('[name="confirmacion_ratificacion"]').check()
            r = respuesta_de(p, RUTA_OPERACIONES, "POST", lambda: p.locator('[data-accion="confirmar-ratificar"]').click())
            comando_rat = r.request.post_data_json
            ratificacion = datos(r, RUTA_OPERACIONES, "POST")
            recibo_rat = verificar_recibo_centro(ratificacion, peticion, 2, "ratificada", actor_rat["referencia"])

            p = paginas["rrhh"]
            r = respuesta_de(p, RUTA_RRHH, "GET", lambda: p.goto(origen + RUTA_CENTRO + "?vista=rrhh", wait_until="domcontentloaded"))
            filas = datos(r, RUTA_RRHH)["peticiones"]
            fila = next((x for x in filas if x.get("peticion", {}).get("referencia") == peticion), None)
            if not fila or fila["peticion"].get("version") != 2 or fila.get("estado_entrega") != "pendiente":
                raise AssertionError("RRHH no ve la petición ratificada pendiente")
            p.locator(f'[data-seleccionar-rrhh="{peticion}"]').click()
            p.locator('[data-accion="abrir-alta-rrhh"]').click()
            p.locator('[name="confirmacion-alta-rrhh"]').check()
            r = respuesta_de(p, RUTA_RRHH, "POST", lambda: p.locator('[data-accion="confirmar-alta-rrhh"]').click())
            if r.status == 503:
                p.locator('[data-accion="reintentar-alta-rrhh"]').wait_for()
                r = respuesta_de(p, RUTA_RRHH, "POST", lambda: p.locator('[data-accion="reintentar-alta-rrhh"]').click())
            alta_rrhh = datos(r, RUTA_RRHH, "POST")
            recibo_inicial = verificar_entrega(alta_rrhh, peticion)
            cuerpo_replay = {"peticion_ref": peticion, "version_esperada": 2}
            replay = contextos["rrhh"].request.post(origen + RUTA_RRHH, data=cuerpo_replay, max_redirects=0)
            recibo_replay = verificar_entrega(datos_api(replay, RUTA_RRHH, "POST"), peticion)
            if recibo_replay != recibo_inicial:
                raise AssertionError("replay con recibo distinto o expediente duplicado")

            # El hook es externo y solo debe reiniciar los dos procesos del clon.
            subprocess.run([args.reinicio], check=True, timeout=120,
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            lectura = contextos["rrhh"].request.get(origen + RUTA_RRHH, max_redirects=0)
            filas = datos_api(lectura, RUTA_RRHH)["peticiones"]
            coincidencias = [x for x in filas if x.get("peticion", {}).get("referencia") == peticion]
            if len(coincidencias) != 1 or verificar_entrega(coincidencias[0], peticion) != recibo_inicial:
                raise AssertionError("la recuperación tras reinicio no conserva una entrega y su recibo")
            replay = contextos["rrhh"].request.post(origen + RUTA_RRHH, data=cuerpo_replay, max_redirects=0)
            if verificar_entrega(datos_api(replay, RUTA_RRHH, "POST"), peticion) != recibo_inicial:
                raise AssertionError("replay tras reinicio distinto del recibo original")
            for rol, comando, esperado, version, estado, actor in (
                ("solicitante", comando_sol, recibo_sol, 1, "pendiente_ratificacion", actor_sol["referencia"]),
                ("ratificador", comando_rat, recibo_rat, 2, "ratificada", actor_rat["referencia"]),
            ):
                respuesta = contextos[rol].request.post(origen + RUTA_OPERACIONES, data=comando, max_redirects=0)
                recibido = datos_api(respuesta, RUTA_OPERACIONES, "POST")
                if (verificar_recibo_centro(recibido, peticion, version, estado, actor) != esperado
                        or recibido.get("estado_local") != "replay_confirmado"):
                    raise AssertionError("replay de centro tras reinicio no conserva el recibo original")
            if errores_js:
                raise AssertionError("se observaron errores JavaScript")
            if any(ctx.cookies() for ctx in contextos.values()):
                raise AssertionError("el navegador recibió cookies")
            if any(p.evaluate("() => localStorage.length || sessionStorage.length") for p in paginas.values()):
                raise AssertionError("el navegador escribió almacenamiento web")
            if any(p.evaluate("async () => (await indexedDB.databases()).length") for p in paginas.values()):
                raise AssertionError("el navegador creó bases IndexedDB")
            print("EJECUTADO: petición v1, ratificación v2, alta RRHH, replay y recuperación idénticos")
            print("LÍMITE: circuito sintético; no acredita firma, entrega externa ni cierre CT")
        finally:
            browser.close()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    for nombre in ("origen", "acreditacion", "binario", "reinicio"):
        parser.add_argument(f"--{nombre}")
    for rol in ("solicitante", "ratificador", "rrhh"):
        parser.add_argument(f"--cert-{rol}", dest=f"cert_{rol}")
        parser.add_argument(f"--clave-{rol}", dest=f"clave_{rol}")
    args = parser.parse_args()
    try:
        origen = preflight(args)
    except NoEjecutado as exc:
        print(f"NO EJECUTADO: {exc}")
        return 2
    try:
        recorrer(args, origen)
    except Exception as exc:
        print(f"FALLÓ: {type(exc).__name__}; revisar evidencia privada del clon sin repetir una escritura incierta")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
