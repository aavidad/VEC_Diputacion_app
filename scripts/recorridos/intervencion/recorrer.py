#!/usr/bin/env python3
"""Recorrido de Intervención y RRHH sobre dos expedientes sintéticos locales.

El operador prepara el clon y reinicia aplicación y PostgreSQL entre las fases
``registrar`` y ``recuperar``. La fase de recuperación usa las mismas peticiones
capturadas en el navegador; nunca crea una clave nueva.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import uuid
from pathlib import Path
from urllib.parse import quote, urlsplit


RUTA_FISCAL = "/api/vec/contratacion-temporal/fiscalizaciones/resultados"
RUTA_SUBSANACION = "/api/vec/contratacion-temporal/subsanacion-reparos"
RUTA_DETALLE = "/api/vec/contratacion-temporal/expedientes/consultas"
REF = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$")
REPO = Path(__file__).resolve().parents[3]


def raiz_git_estable(repo: Path) -> Path:
    """Un worktree anidado y la raíz integrada comparten el mismo árbol Git."""
    return repo.parent.parent if repo.parent.name == ".worktrees" else repo


ARBOL_COMPARTIDO = raiz_git_estable(REPO)


class NoEjecutado(Exception):
    """Falta una precondición antes de abrir el navegador."""


class FalloRecorrido(Exception):
    """Una respuesta o un estado observado incumple el contrato."""


def cargar_json(ruta: Path) -> dict:
    if not ruta.is_file() or ruta.stat().st_size > 32_768:
        raise NoEjecutado("falta un fichero de configuración acotado")
    try:
        datos = json.loads(ruta.read_text(encoding="utf-8"))
    except (UnicodeError, json.JSONDecodeError) as error:
        raise NoEjecutado("JSON de configuración o evidencia no válido") from error
    if not isinstance(datos, dict):
        raise NoEjecutado("la configuración debe ser un objeto JSON")
    return datos


def exterior(ruta: Path) -> Path:
    real = ruta.expanduser().resolve(strict=True)
    if real == ARBOL_COMPARTIDO or ARBOL_COMPARTIDO in real.parents:
        raise NoEjecutado("certificados, claves y evidencia deben permanecer fuera de Git")
    return real


def validar_configuracion(datos: dict, salida: Path, fase: str) -> dict:
    esperados = {"origen", "chrome", "binario", "binario_sha256", "hitos_clon",
                 "uso_sintetico",
                 "intervencion_cert", "intervencion_key", "rrhh_cert", "rrhh_key",
                 "favorable", "reparo"}
    if set(datos) != esperados:
        raise NoEjecutado("faltan entradas locales o sobran campos de configuración")
    origen = urlsplit(datos["origen"])
    if (origen.scheme != "https" or origen.hostname not in {"localhost", "127.0.0.1", "::1"}
            or not origen.port or origen.username or origen.password or origen.path
            or origen.query or origen.fragment):
        raise NoEjecutado("el origen debe ser HTTPS loopback sin ruta ni credenciales")
    if datos["hitos_clon"] != ["H3", "H4", "H5"]:
        raise NoEjecutado("clon H3–H5 no acreditado en la entrada")
    if datos["uso_sintetico"] is not True:
        raise NoEjecutado("datos e identidades sintéticos no acreditados")
    chrome, binario = (Path(datos[clave]).expanduser().resolve(strict=True)
                       for clave in ("chrome", "binario"))
    if not chrome.is_file() or not binario.is_file():
        raise NoEjecutado("falta Chrome del sistema o el binario VEC")
    if not re.fullmatch(r"[0-9a-f]{64}", datos["binario_sha256"]):
        raise NoEjecutado("falta la huella exacta del binario")
    huella = hashlib.sha256()
    with binario.open("rb") as archivo:
        for bloque in iter(lambda: archivo.read(1024 * 1024), b""):
            huella.update(bloque)
    if huella.hexdigest() != datos["binario_sha256"]:
        raise NoEjecutado("la huella del binario VEC no coincide")
    for clave in ("intervencion_cert", "intervencion_key", "rrhh_cert", "rrhh_key"):
        ruta = exterior(Path(datos[clave]))
        if not ruta.is_file() or ruta.stat().st_size == 0:
            raise NoEjecutado("falta material mTLS sintético externo")
        datos[clave] = str(ruta)
    if Path(datos["intervencion_cert"]).read_bytes() == Path(datos["rrhh_cert"]).read_bytes():
        raise NoEjecutado("Intervención y RRHH deben usar certificados distintos")
    if Path(datos["intervencion_key"]).read_bytes() == Path(datos["rrhh_key"]).read_bytes():
        raise NoEjecutado("Intervención y RRHH deben usar claves distintas")
    refs = []
    for nombre in ("favorable", "reparo"):
        caso = datos[nombre]
        if (not isinstance(caso, dict) or set(caso) != {"expediente_ref", "version_esperada"}
                or not isinstance(caso["expediente_ref"], str)
                or not REF.fullmatch(caso["expediente_ref"])
                or type(caso["version_esperada"]) is not int
                or caso["version_esperada"] < 1):
            raise NoEjecutado("cada caso exige referencia opaca y versión inicial exacta")
        refs.append(caso["expediente_ref"])
    if refs[0] == refs[1]:
        raise NoEjecutado("los dos recorridos exigen expedientes distintos")
    salida = salida.expanduser().resolve(strict=False)
    if salida == ARBOL_COMPARTIDO or ARBOL_COMPARTIDO in salida.parents:
        raise NoEjecutado("la evidencia debe escribirse fuera de Git")
    if fase == "registrar" and salida.exists():
        raise NoEjecutado("ya existe evidencia: no repetir operaciones con claves nuevas")
    if fase == "recuperar" and not salida.is_file():
        raise NoEjecutado("falta la evidencia del registro anterior al reinicio")
    datos["origen"] = datos["origen"].rstrip("/")
    datos["chrome"] = str(chrome)
    return datos


def envoltorio(respuesta, ruta: str, estados: tuple[int, ...]) -> dict:
    if respuesta.status not in estados or urlsplit(respuesta.url).path != ruta:
        raise FalloRecorrido(f"{ruta}: estado HTTP inesperado")
    cuerpo = respuesta.json()
    if not isinstance(cuerpo, dict) or not isinstance(cuerpo.get("data"), dict):
        raise FalloRecorrido(f"{ruta}: falta el envoltorio de respuesta")
    if "set-cookie" in respuesta.headers:
        raise FalloRecorrido(f"{ruta}: Set-Cookie inesperado")
    return cuerpo["data"]


def validar_recibo(recibo: dict, solicitud: dict, operacion: str) -> dict:
    fase = "fiscalizacion" if operacion == "favorable" else "subsanacion_unidad"
    estado = "en_curso" if operacion == "favorable" else "incidencia"
    if (recibo.get("expediente_ref") != solicitud["expediente_ref"]
            or recibo.get("version_resultante") != solicitud["version_esperada"] + 1
            or recibo.get("fase_resultante") != fase
            or recibo.get("estado_resultante") != estado
            or not isinstance(recibo.get("recibo_ref"), str)
            or not isinstance(recibo.get("registrada_en"), str)
            or not isinstance(recibo.get("auditoria_ref"), str)
            or not isinstance(recibo.get("evento_ref"), str)
            or not isinstance(recibo.get("actor_ref"), str)
            or (operacion != "subsanacion" and recibo.get("resultado") != solicitud["resultado"])):
        raise FalloRecorrido(f"recibo {operacion} no ligado a la operación")
    return {campo: recibo[campo] for campo in (
        "expediente_ref", "version_resultante", "fase_resultante", "estado_resultante",
        "recibo_ref", "registrada_en", "auditoria_ref", "evento_ref", "actor_ref")}


def persistir(salida: Path, evidencia: dict, nuevo: bool = False):
    salida.parent.mkdir(parents=True, exist_ok=True)
    temporal = salida.with_name(salida.name + "." + uuid.uuid4().hex + ".tmp")
    descriptor = os.open(temporal, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8") as archivo:
            json.dump(evidencia, archivo, ensure_ascii=False, indent=2)
            archivo.flush()
            os.fsync(archivo.fileno())
        if nuevo:
            try:
                os.link(temporal, salida, follow_symlinks=False)
            except FileExistsError as error:
                raise NoEjecutado("la evidencia ya existe") from error
        else:
            os.replace(temporal, salida)
    finally:
        temporal.unlink(missing_ok=True)


def responder_sin_redireccion(interceptada, origen: str):
    destino = urlsplit(interceptada.request.url)
    if f"{destino.scheme}://{destino.netloc}" != origen:
        interceptada.abort()
        return
    try:
        respuesta = interceptada.fetch(max_redirects=0, timeout=30_000)
        final = urlsplit(respuesta.url)
        if (300 <= respuesta.status < 400
                or f"{final.scheme}://{final.netloc}" != origen):
            interceptada.abort()
            return
        interceptada.fulfill(response=respuesta)
    except Exception:
        interceptada.abort()


def ligar_peticion(pagina, ruta: str, expediente: str, evidencia: dict,
                   salida: Path, nombre: str, estado_ejecucion: dict):
    """Conserva la clave antes de permitir el POST del navegador."""
    def guardar_y_continuar(interceptada):
        solicitud = interceptada.request.post_data_json
        if (interceptada.request.method != "POST" or not isinstance(solicitud, dict)
                or solicitud.get("expediente_ref") != expediente
                or not isinstance(solicitud.get("clave_idempotencia"), str)):
            interceptada.abort()
            return
        try:
            evidencia["intencion_pendiente"] = {"operacion": nombre, "solicitud": solicitud}
            persistir(salida, evidencia)
        except Exception:
            interceptada.abort()
            return
        estado_ejecucion["post_posible"] = True
        responder_sin_redireccion(interceptada, evidencia["origen"])
    pagina.route("**" + ruta, guardar_y_continuar)
    return guardar_y_continuar


def contexto(navegador, datos: dict, actor: str):
    actor_contexto = navegador.new_context(
        client_certificates=[{"origin": datos["origen"],
                              "certPath": datos[f"{actor}_cert"],
                              "keyPath": datos[f"{actor}_key"]}],
        ignore_https_errors=False, service_workers="block", locale="es-ES",
        timezone_id="Europe/Madrid", viewport={"width": 1440, "height": 900},
        accept_downloads=True,
    )
    cerrar_destinos_externos(actor_contexto, datos["origen"])
    actor_contexto.route_web_socket("**/*", lambda canal: canal.close(code=1008))
    return actor_contexto


def cerrar_destinos_externos(actor_contexto, origen: str):
    def filtrar(ruta):
        destino = urlsplit(ruta.request.url)
        if destino.scheme == "https" and f"{destino.scheme}://{destino.netloc}" == origen:
            responder_sin_redireccion(ruta, origen)
        else:
            ruta.abort()
    actor_contexto.route("**/*", filtrar)


def abrir(pagina, datos: dict, expediente: str | None = None):
    ruta = "/portal-empleado/"
    if expediente is not None:
        ruta += "?expediente=" + quote(expediente, safe="")
    respuesta = pagina.goto(datos["origen"] + ruta + "#contratacion-temporal",
                            wait_until="networkidle", timeout=30_000)
    if respuesta is None or respuesta.status != 200:
        raise FalloRecorrido("el portal no respondió 200")


def medir_pagina(contexto_actor, pagina):
    estado = pagina.evaluate("""async () => ({ancho: document.documentElement.clientWidth,
      contenido: document.documentElement.scrollWidth,
      local: localStorage.length, sesion: sessionStorage.length,
      indexeddb: (await indexedDB.databases()).length})""")
    estado["cookies"] = len(contexto_actor.cookies())
    return estado


def comprobar_pagina(contexto_actor, pagina):
    estado = medir_pagina(contexto_actor, pagina)
    if (estado["contenido"] > estado["ancho"] or estado["local"] or estado["sesion"]
            or estado["indexeddb"] or estado["cookies"]):
        raise FalloRecorrido("desbordamiento, almacenamiento web o cookies")
    return estado


def capturar_pagina(contexto_actor, pagina, salida: Path, nombre: str) -> list[dict]:
    """Captura los dos anchos sin enviar otra operación ni guardar certificados."""
    if not re.fullmatch(r"[a-z_]+", nombre):
        raise ValueError("nombre de captura no válido")
    carpeta = salida.parent / (salida.stem + "-capturas")
    carpeta.mkdir(mode=0o700, exist_ok=True)
    if carpeta.is_symlink():
        raise NoEjecutado("la carpeta de capturas no puede ser un enlace")
    os.chmod(carpeta, 0o700)
    capturas = []
    try:
        for ancho, alto in ((1440, 900), (390, 844)):
            pagina.set_viewport_size({"width": ancho, "height": alto})
            pagina.wait_for_timeout(250)
            ruta = carpeta / f"{nombre}-{ancho}.png"
            descriptor = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(descriptor, "wb") as archivo:
                imagen = pagina.screenshot(full_page=True, animations="disabled")
                archivo.write(imagen)
            estado = medir_pagina(contexto_actor, pagina)
            capturas.append({"archivo": ruta.name, "sha256": hashlib.sha256(imagen).hexdigest(),
                             "viewport": {"width": ancho, "height": alto}, "estado": estado})
    finally:
        pagina.set_viewport_size({"width": 1440, "height": 900})
    return capturas


def observar_http(respuesta, evidencia: dict):
    """Guarda ruta y estado; excluye cuerpos, cabeceras y parámetros."""
    ruta = urlsplit(respuesta.url).path
    if ruta.startswith("/api/"):
        evidencia.setdefault("http_observado", []).append({
            "metodo": respuesta.request.method, "ruta": ruta, "estado": respuesta.status})


def registrar_fiscalizacion(pagina, datos: dict, caso: dict, resultado: str,
                            evidencia: dict, salida: Path, estado_ejecucion: dict):
    abrir(pagina, datos)
    pagina.locator("[data-ct-fiscalizacion-acceso] [name=expediente_ref]").fill(caso["expediente_ref"])
    pagina.locator("[data-ct-fiscalizacion-acceso] [name=version_esperada]").fill(str(caso["version_esperada"]))
    pagina.locator("[data-ct-fiscalizacion-acceso] button[type=submit]").click()
    form = pagina.locator("[data-ct-fiscalizacion-form]")
    form.wait_for(timeout=10_000)
    form.locator(f'[name=resultado][value="{resultado}"]').check()
    if resultado == "desfavorable":
        form.locator("[name=observaciones]").fill("Reparo sintético para subsanación de unidad.")
    etiqueta = "favorable" if resultado == "favorable" else "reparo"
    interceptador = ligar_peticion(pagina, RUTA_FISCAL, caso["expediente_ref"],
                                   evidencia, salida, etiqueta, estado_ejecucion)
    with pagina.expect_response(lambda r: urlsplit(r.url).path == RUTA_FISCAL
                                and r.request.method == "POST", timeout=30_000) as observado:
        form.locator("button[type=submit]").click()
    pagina.unroute("**" + RUTA_FISCAL, interceptador)
    respuesta = observado.value
    solicitud = respuesta.request.post_data_json
    if not isinstance(solicitud, dict) or solicitud.get("resultado") != resultado:
        raise FalloRecorrido("petición de fiscalización no ligada al formulario")
    recibo = envoltorio(respuesta, RUTA_FISCAL, (201,))
    resumen = validar_recibo(recibo, solicitud, etiqueta)
    pagina.locator("[data-ct-fiscalizacion-recibo]").wait_for(timeout=10_000)
    return {"solicitud": solicitud, "recibo": resumen, "http_inicial": 201}


def registrar_subsanacion(pagina, datos: dict, fiscal: dict,
                          evidencia: dict, salida: Path, estado_ejecucion: dict):
    expediente = fiscal["recibo"]["expediente_ref"]
    abrir(pagina, datos, expediente)
    form = pagina.locator("[data-ct-subsanacion-form]")
    form.wait_for(timeout=15_000)
    form.locator("[name=observaciones]").fill("Subsanación sintética del reparo para nueva revisión.")
    form.locator("button[type=submit]").click()
    with pagina.expect_download(timeout=10_000):
        pagina.locator("[data-ct-subsanacion-guardar]").click()
    interceptador = ligar_peticion(pagina, RUTA_SUBSANACION, expediente,
                                   evidencia, salida, "subsanacion", estado_ejecucion)
    with pagina.expect_response(lambda r: urlsplit(r.url).path == RUTA_SUBSANACION
                                and r.request.method == "POST", timeout=30_000) as observado:
        pagina.locator("[data-ct-subsanacion-enviar]").click()
    pagina.unroute("**" + RUTA_SUBSANACION, interceptador)
    respuesta = observado.value
    solicitud = respuesta.request.post_data_json
    if not isinstance(solicitud, dict):
        raise FalloRecorrido("petición de subsanación ausente")
    resumen = validar_recibo(envoltorio(respuesta, RUTA_SUBSANACION, (201,)),
                             solicitud, "subsanacion")
    pagina.locator("[data-ct-subsanacion-recibo]").wait_for(timeout=10_000)
    return {"solicitud": solicitud, "recibo": resumen, "http_inicial": 201}


def repetir(contexto_actor, datos: dict, ruta: str, registro: dict, estado_ejecucion: dict):
    estado_ejecucion["post_posible"] = True
    respuesta = contexto_actor.request.post(datos["origen"] + ruta,
                                            data=registro["solicitud"], timeout=30_000,
                                            max_redirects=0)
    estado_esperado = 201 if ruta == RUTA_FISCAL else 200
    recibido = envoltorio(respuesta, ruta, (estado_esperado,))
    operacion = ("subsanacion" if ruta == RUTA_SUBSANACION else
                 "favorable" if registro["solicitud"]["resultado"] == "favorable" else "reparo")
    if validar_recibo(recibido, registro["solicitud"], operacion) != registro["recibo"]:
        raise FalloRecorrido("el replay cambió recibo, fecha, auditoría o evento")
    return respuesta.status


def consultar_detalle(contexto_actor, datos: dict, expediente: str, version: int,
                      acciones: list[str]):
    respuesta = contexto_actor.request.post(datos["origen"] + RUTA_DETALLE,
                                           data={"expediente_ref": expediente,
                                                 "version_observada": version}, timeout=30_000,
                                           max_redirects=0)
    detalle = envoltorio(respuesta, RUTA_DETALLE, (200,))
    resumen = detalle.get("resumen", {})
    hitos = detalle.get("hitos", [])
    if (resumen.get("expediente_ref") != expediente or resumen.get("version") != version
            or not isinstance(hitos, list) or len(hitos) != version
            or [h.get("accion_clave") for h in hitos[-len(acciones):]] != acciones):
        raise FalloRecorrido("detalle o historia cambiaron tras el reinicio")


def ejecutar(fase: str, datos: dict, salida: Path, estado_ejecucion: dict) -> dict:
    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        navegador = p.chromium.launch(executable_path=datos["chrome"], headless=True)
        estado_ejecucion["navegador"] = True
        try:
            intervencion = contexto(navegador, datos, "intervencion")
            rrhh = contexto(navegador, datos, "rrhh")
            try:
                if fase == "registrar":
                    evidencia = {"esquema": "vec.recorrido-intervencion.v1",
                                 "origen": datos["origen"], "binario_sha256": datos["binario_sha256"]}
                    persistir(salida, evidencia, nuevo=True)
                    pagina_int = intervencion.new_page()
                    pagina_rrhh = rrhh.new_page()
                    pagina_int.on("dialog", lambda dialogo: dialogo.accept())
                    pagina_rrhh.on("dialog", lambda dialogo: dialogo.accept())
                    js = []
                    cookies_respuesta = []
                    pagina_int.on("pageerror", lambda error: js.append(str(error)))
                    pagina_rrhh.on("pageerror", lambda error: js.append(str(error)))
                    pagina_int.on("response", lambda r: cookies_respuesta.append(urlsplit(r.url).path)
                                  if "set-cookie" in r.headers else None)
                    pagina_rrhh.on("response", lambda r: cookies_respuesta.append(urlsplit(r.url).path)
                                   if "set-cookie" in r.headers else None)
                    pagina_int.on("response", lambda r: observar_http(r, evidencia))
                    pagina_rrhh.on("response", lambda r: observar_http(r, evidencia))
                    try:
                        favorable = registrar_fiscalizacion(pagina_int, datos, datos["favorable"],
                                                         "favorable", evidencia, salida, estado_ejecucion)
                        evidencia["favorable"] = favorable
                        evidencia.pop("intencion_pendiente", None)
                        evidencia["capturas_favorable"] = capturar_pagina(intervencion, pagina_int,
                                                                          salida, "favorable")
                        persistir(salida, evidencia)
                        reparo = registrar_fiscalizacion(pagina_int, datos, datos["reparo"],
                                                      "desfavorable", evidencia, salida, estado_ejecucion)
                        evidencia["reparo"] = reparo
                        evidencia.pop("intencion_pendiente", None)
                        evidencia["capturas_reparo"] = capturar_pagina(intervencion, pagina_int,
                                                                       salida, "reparo")
                        persistir(salida, evidencia)
                        subsanacion = registrar_subsanacion(pagina_rrhh, datos, reparo,
                                                        evidencia, salida, estado_ejecucion)
                        evidencia["subsanacion"] = subsanacion
                        evidencia.pop("intencion_pendiente", None)
                        evidencia["capturas_subsanacion"] = capturar_pagina(rrhh, pagina_rrhh,
                                                                            salida, "subsanacion")
                    except Exception:
                        evidencia["corte"] = "primer_fallo"
                        try:
                            evidencia["capturas_corte"] = capturar_pagina(intervencion, pagina_int,
                                                                          salida, "corte_intervencion")
                            if pagina_rrhh.url != "about:blank":
                                evidencia["capturas_corte_rrhh"] = capturar_pagina(rrhh, pagina_rrhh,
                                                                               salida, "corte_rrhh")
                        except Exception:
                            evidencia["captura_corte_incompleta"] = True
                        raise
                    finally:
                        evidencia["errores_js"] = len(js)
                        evidencia["set_cookie"] = len(cookies_respuesta)
                        persistir(salida, evidencia)
                    if (favorable["recibo"]["actor_ref"] != reparo["recibo"]["actor_ref"]
                            or favorable["recibo"]["actor_ref"] == subsanacion["recibo"]["actor_ref"]):
                        raise FalloRecorrido("los recibos no conservan separación Intervención y RRHH")
                    if js or cookies_respuesta:
                        raise FalloRecorrido("errores JavaScript o Set-Cookie en el navegador")
                    comprobar_pagina(intervencion, pagina_int)
                    comprobar_pagina(rrhh, pagina_rrhh)
                    pagina_int.set_viewport_size({"width": 390, "height": 844})
                    pagina_rrhh.set_viewport_size({"width": 390, "height": 844})
                    comprobar_pagina(intervencion, pagina_int)
                    comprobar_pagina(rrhh, pagina_rrhh)
                    if js or cookies_respuesta:
                        raise FalloRecorrido("errores JavaScript o Set-Cookie tras cambiar de tamaño")
                    return {"fase": fase, "estado": "REGISTRADO", "evidencia": str(salida)}
                evidencia = cargar_json(salida)
                if (evidencia.get("esquema") != "vec.recorrido-intervencion.v1"
                        or evidencia.get("origen") != datos["origen"]
                        or evidencia.get("binario_sha256") != datos["binario_sha256"]
                        or "intencion_pendiente" in evidencia
                        or any(nombre not in evidencia for nombre in
                               ("favorable", "reparo", "subsanacion"))):
                    raise NoEjecutado("evidencia y entorno no corresponden al registro")
                resultados = {}
                for nombre in ("favorable", "reparo"):
                    resultados[nombre] = repetir(intervencion, datos, RUTA_FISCAL,
                                                 evidencia[nombre], estado_ejecucion)
                resultados["subsanacion"] = repetir(rrhh, datos, RUTA_SUBSANACION,
                                                     evidencia["subsanacion"], estado_ejecucion)
                consultar_detalle(rrhh, datos, evidencia["favorable"]["recibo"]["expediente_ref"],
                                  evidencia["favorable"]["recibo"]["version_resultante"],
                                  ["contratacion_temporal.fiscalizacion.registrar"])
                consultar_detalle(rrhh, datos, evidencia["subsanacion"]["recibo"]["expediente_ref"],
                                  evidencia["subsanacion"]["recibo"]["version_resultante"],
                                  ["contratacion_temporal.fiscalizacion.registrar",
                                   "contratacion_temporal.subsanacion_reparos.registrar"])
                return {"fase": fase, "estado": "RECUPERADO", "http": resultados,
                        "reinicio": "acreditado_por_operador"}
            finally:
                intervencion.close()
                rrhh.close()
        finally:
            navegador.close()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("fase", choices=("preparar", "registrar", "recuperar"))
    parser.add_argument("--config", required=True, type=Path)
    parser.add_argument("--evidencia", required=True, type=Path)
    parser.add_argument("--reinicio-acreditado", action="store_true")
    args = parser.parse_args()
    estado_ejecucion = {"navegador": False, "post_posible": False}
    try:
        if args.fase == "recuperar" and not args.reinicio_acreditado:
            raise NoEjecutado("dirección debe acreditar el reinicio de aplicación y PostgreSQL")
        datos = validar_configuracion(cargar_json(args.config), args.evidencia,
                                     "registrar" if args.fase == "preparar" else args.fase)
        if args.fase == "preparar":
            print(json.dumps({"estado": "PREPARADO", "ejecutado": False}, ensure_ascii=False))
        else:
            print(json.dumps(ejecutar(args.fase, datos, args.evidencia, estado_ejecucion),
                             ensure_ascii=False))
        return 0
    except Exception as error:
        if estado_ejecucion["post_posible"]:
            estado = "FALLO_CON_EFECTO_POSIBLE"
        elif estado_ejecucion["navegador"] or isinstance(error, FalloRecorrido):
            estado = "FALLO"
        elif isinstance(error, (NoEjecutado, FileNotFoundError, OSError,
                                ValueError, ImportError)):
            estado = "NO EJECUTADO"
        else:
            estado = "FALLO"
        motivo = (str(error) if isinstance(error, (NoEjecutado, FalloRecorrido))
                  else "fallo local; conserve la evidencia y revise el entorno")
        print(json.dumps({"estado": estado, "motivo": motivo}, ensure_ascii=False),
              file=sys.stderr)
        return 2 if estado == "NO EJECUTADO" else 1


if __name__ == "__main__":
    raise SystemExit(main())
