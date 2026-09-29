#!/usr/bin/env python3
"""Recorrido mTLS de Bolsa en un clon local aislado, con Chrome del sistema."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
from datetime import datetime
from pathlib import Path
from urllib.parse import urlsplit


RUTA_POLITICA = "/api/vec/bolsa/politica-ofertas"
RUTA_OFERTAS = "/api/vec/bolsa/ofertas"
RUTA_MI_BOLSA = "/api/vec/bolsa/mi-bolsa"
RUTA_DISPOSICIONES = RUTA_MI_BOLSA + "/disposiciones"
RUTA_AVISOS = "/api/vec/bolsa/avisos"
RUTA_EMISIONES = "/api/vec/bolsa/llamamientos/emisiones"
REFERENCIA = re.compile(r"^[A-Za-z0-9][A-Za-z0-9:._/-]{0,255}$")
INSTANTE_UTC = re.compile(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$")


class NoEjecutado(Exception):
    """Falta una precondición que impide tocar el clon."""


class FalloRecorrido(Exception):
    """Una respuesta o un estado visible incumple el contrato esperado."""


def sha256(ruta: Path) -> str:
    suma = hashlib.sha256()
    with ruta.open("rb") as origen:
        for bloque in iter(lambda: origen.read(1024 * 1024), b""):
            suma.update(bloque)
    return suma.hexdigest()


def dentro_de_git(ruta: Path) -> bool:
    """Incluye el checkout principal y cualquier worktree, también por symlink."""
    destino = ruta.resolve()
    return any((ancestro / ".git").exists() for ancestro in (destino.parent, *destino.parents))


def validar_origen(valor: str) -> str:
    try:
        url = urlsplit(valor)
        puerto = url.port
    except ValueError as error:
        raise NoEjecutado("origen local mal formado") from error
    if (url.scheme != "https" or url.hostname not in {"localhost", "127.0.0.1", "::1"}
            or puerto is None or url.username or url.password or url.path not in {"", "/"}
            or url.query or url.fragment):
        raise NoEjecutado("se exige un origen HTTPS de loopback sin ruta ni credenciales")
    return valor.rstrip("/")


def validar_configuracion(datos: dict, fase: str) -> dict:
    """El director acredita el clon H3-H5; el guion comprueba binario y entradas."""
    if not isinstance(datos, dict) or datos.get("clon") != "aislado_h3_h4_h5" or datos.get("hitos") != ["H3", "H4", "H5"]:
        raise NoEjecutado("falta acreditación explícita del clon aislado H3-H5")
    datos["origen"] = validar_origen(datos.get("origen", ""))
    binario = Path(datos.get("binario", ""))
    huella = datos.get("binario_sha256", "")
    if dentro_de_git(binario):
        raise NoEjecutado("el binario del clon debe estar fuera de Git y sus worktrees")
    if not binario.is_file() or not re.fullmatch(r"[0-9a-f]{64}", huella) or sha256(binario) != huella:
        raise NoEjecutado("falta el binario VEC exacto del clon")
    chrome = Path(datos.get("chrome", "/usr/bin/google-chrome"))
    if not chrome.is_file() or chrome.resolve().name not in {"chrome", "google-chrome", "chromium"}:
        raise NoEjecutado("falta Chrome del sistema")
    datos["chrome"] = str(chrome)
    rutas_material = {}
    for rol in ("rrhh", "candidato"):
        material = datos.get(rol, {})
        for campo in ("certificado", "clave"):
            ruta = Path(material.get(campo, ""))
            if dentro_de_git(ruta):
                raise NoEjecutado(f"material mTLS de {rol} dentro de Git o un worktree")
            if not ruta.is_file() or ruta.stat().st_size == 0:
                raise NoEjecutado(f"falta material mTLS sintético de {rol}")
            if campo == "clave" and ruta.stat().st_mode & 0o077:
                raise NoEjecutado(f"la clave mTLS de {rol} debe tener permisos 0600")
            rutas_material[(rol, campo)] = ruta.resolve()
    for campo in ("certificado", "clave"):
        ruta_rrhh = rutas_material[("rrhh", campo)]
        ruta_candidato = rutas_material[("candidato", campo)]
        if ruta_rrhh == ruta_candidato or sha256(ruta_rrhh) == sha256(ruta_candidato):
            raise NoEjecutado(f"RRHH y candidato deben usar {campo}s mTLS distintos")
    bolsa = datos.get("bolsa_ref", "")
    if not REFERENCIA.fullmatch(bolsa):
        raise NoEjecutado("falta referencia opaca de una bolsa sintética")
    if fase == "alta":
        oferta = datos.get("oferta", {})
        if not all(isinstance(oferta.get(c), str) and oferta[c].strip() for c in ("categoria", "centro", "fecha_inicio", "descripcion")):
            raise NoEjecutado("faltan datos sintéticos de la oferta")
        if not re.fullmatch(r"\d{4}-\d{2}-\d{2}", oferta["fecha_inicio"]):
            raise NoEjecutado("fecha_inicio debe tener formato AAAA-MM-DD")
        if datos.get("bolsa_sintetica_reservada") is not True:
            raise NoEjecutado("la bolsa de ensayo debe estar reservada para este recorrido")
        if datos.get("emitir_b7") is True:
            emision = datos.get("emision_b7", {})
            if not all(isinstance(emision.get(c), str) and emision[c].strip()
                       for c in ("referencia", "centro", "fecha_inicio", "descripcion")):
                raise NoEjecutado("faltan datos sintéticos para la emisión B7")
    elif datos.get("reinicio_confirmado") is not True:
        raise NoEjecutado("falta confirmación externa del reinicio de VEC y PostgreSQL del clon")
    return datos


def respuesta_json(respuesta, ruta: str, codigos: set[int]) -> dict:
    if respuesta.status not in codigos:
        raise FalloRecorrido(f"{ruta} respondió HTTP {respuesta.status}")
    if "set-cookie" in respuesta.headers:
        raise FalloRecorrido(f"{ruta} emitió Set-Cookie")
    try:
        cuerpo = respuesta.json()
    except ValueError as error:
        raise FalloRecorrido(f"{ruta} no devolvió JSON") from error
    if not isinstance(cuerpo, dict) or not isinstance(cuerpo.get("data"), dict):
        raise FalloRecorrido(f"{ruta} devolvió un sobre inesperado")
    return cuerpo["data"]


def version_obligatoria(datos: dict, campo: str) -> int:
    valor = datos.get(campo)
    if type(valor) is not int or valor < 1:
        raise FalloRecorrido(f"falta {campo} válida en recibo o recuperación")
    return valor


def referencia_obligatoria(datos: dict, campo: str) -> str:
    valor = datos.get(campo)
    if not isinstance(valor, str) or not REFERENCIA.fullmatch(valor):
        raise FalloRecorrido(f"falta {campo} válida en recibo o recuperación")
    return valor


def instante_obligatorio(datos: dict, campo: str) -> str:
    valor = datos.get(campo)
    if not isinstance(valor, str) or not INSTANTE_UTC.fullmatch(valor):
        raise FalloRecorrido(f"falta {campo} UTC en recibo o recuperación")
    try:
        datetime.fromisoformat(valor.replace("Z", "+00:00"))
    except ValueError as error:
        raise FalloRecorrido(f"{campo} no es una fecha válida") from error
    return valor


def cantidad_seleccionada(texto: str) -> int:
    """Lee el total global del estado B7, no los seis checks de la página."""
    coincidencia = re.match(r"^\s*([\d.,\u00a0]+)\s+seleccionadas\b", texto)
    if not coincidencia:
        raise FalloRecorrido("B7 no muestra el total de personas seleccionadas")
    return int("".join(c for c in coincidencia.group(1) if c.isdigit()))


def instalar_filtro_red(contexto_actual, origen: str, fallos: list[str]) -> None:
    """Una redirección se inspecciona como respuesta y nunca se sigue."""
    def filtrar(ruta):
        url = urlsplit(ruta.request.url)
        if f"{url.scheme}://{url.netloc}" != origen:
            fallos.append("solicitud fuera del origen local")
            ruta.abort("blockedbyclient")
            return
        try:
            respuesta = ruta.fetch(max_redirects=0, timeout=15000)
            if 300 <= respuesta.status < 400:
                fallos.append("redirección rechazada sin seguirla")
                ruta.abort("blockedbyclient")
            else:
                ruta.fulfill(response=respuesta)
        except Exception:
            fallos.append("error de transporte local")
            ruta.abort("failed")

    contexto_actual.route("**/*", filtrar)
    contexto_actual.route_web_socket("**/*", lambda socket: (fallos.append("WebSocket rechazado"), socket.close()))


def contexto(navegador, origen: str, rol: dict, fallos: list[str]):
    creado = navegador.new_context(
        client_certificates=[{"origin": origen, "certPath": rol["certificado"], "keyPath": rol["clave"]}],
        viewport={"width": 1440, "height": 900}, locale="es-ES", timezone_id="Europe/Madrid",
        ignore_https_errors=False, service_workers="block", accept_downloads=False,
    )
    instalar_filtro_red(creado, origen, fallos)
    return creado


def observar_pagina(pagina, fallos: list[str], cfg: dict | None = None) -> None:
    pagina.on("pageerror", lambda error: fallos.append(type(error).__name__))
    if cfg is not None:
        def registrar(respuesta):
            url = urlsplit(respuesta.url)
            if url.path.startswith("/api/vec/"):
                cfg.setdefault("_http", []).append({"metodo": respuesta.request.method,
                    "ruta": url.path, "estado": respuesta.status,
                    "set_cookie": "set-cookie" in respuesta.headers})
        pagina.on("response", registrar)


def registrar_paso(cfg: dict, nombre: str, datos: dict) -> None:
    """Conserva cada recibo antes de avanzar; no autoriza repetir un alta parcial."""
    cfg.setdefault("_pasos", []).append({"paso": nombre, **datos})
    if cfg.get("_evidencia"):
        Path(cfg["_evidencia"]).write_text(json.dumps({"estado": "INICIADO",
            "pasos": cfg["_pasos"], "http": cfg.get("_http", [])},
            ensure_ascii=False, indent=2) + "\n")


def guardar_captura(ruta: Path, datos: bytes) -> None:
    """El PNG se crea privado, sin seguir enlaces ni sobrescribir otro archivo."""
    descriptor = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, "wb") as salida:
        salida.write(datos)


def capturar(cfg: dict, pagina, nombre: str, comprobar=None) -> None:
    if not cfg.get("_capturas"):
        if comprobar:
            comprobar()
        return
    for ancho, alto in ((1440, 900), (390, 844)):
        pagina.set_viewport_size({"width": ancho, "height": alto})
        pagina.wait_for_timeout(250)
        guardar_captura(Path(cfg["_capturas"]) / f"{nombre}-{ancho}.png",
                        pagina.screenshot(full_page=True))
        estado = pagina.evaluate("""async () => ({ ancho: document.documentElement.clientWidth,
            contenido: document.documentElement.scrollWidth, local: localStorage.length,
            sesion: sessionStorage.length,
            indexeddb: (await indexedDB.databases()).length })""")
        cfg.setdefault("_navegador", []).append({"pantalla": nombre, "viewport": ancho, **estado})
        if comprobar:
            comprobar()
    pagina.set_viewport_size({"width": 1440, "height": 900})


def capturar_corte(cfg: dict, contextos: dict) -> None:
    for rol, actual in contextos.items():
        for indice, pagina in enumerate(actual.pages):
            try:
                capturar(cfg, pagina, f"corte-{rol}-{indice}")
            except Exception:
                pass


def comprobar_navegador(contexto_actual, pagina, fallos: list[str]) -> None:
    estado = pagina.evaluate("""async () => ({ ancho: document.documentElement.clientWidth,
        contenido: document.documentElement.scrollWidth, local: localStorage.length,
        sesion: sessionStorage.length, indexeddb: (await indexedDB.databases()).length })""")
    if (fallos or contexto_actual.cookies() or estado["local"] or estado["sesion"]
            or estado["indexeddb"] or estado["contenido"] > estado["ancho"]):
        raise FalloRecorrido("errores JS, cookies, almacenamiento web o desbordamiento horizontal")


def abrir_bolsa_rrhh(pagina, origen: str, bolsa: str) -> None:
    pagina.goto(origen + "/portal-empleado/#bolsa/resumen", wait_until="domcontentloaded")
    fila = pagina.locator(f'tr[data-bolsa-ref="{bolsa}"] button[data-accion="ver-bolsa"]').first
    fila.wait_for(timeout=20000)
    fila.click()
    pagina.locator('[data-bolsa-accion="iniciar-b7"]').wait_for(timeout=20000)


def abrir_llamamiento_rrhh(pagina, origen: str, bolsa: str) -> None:
    abrir_bolsa_rrhh(pagina, origen, bolsa)
    pagina.evaluate("location.hash = '#bolsa/llamamientos'")
    pagina.locator('[data-rrhh-plazos-form="politica"]').wait_for(timeout=20000)
    pagina.locator('[data-ofertas-form="publicar"]').wait_for(timeout=20000)


def ensayar_envio_b7(pagina, cfg: dict) -> dict:
    """Emite solo con habilitación explícita en el clon y dos personas elegibles."""
    if cfg.get("emitir_b7") is not True:
        return {"estado": "NO EJECUTADO", "motivo": "emisión B7 no habilitada en la configuración privada"}
    origen = cfg["origen"]
    abrir_bolsa_rrhh(pagina, origen, cfg["bolsa_ref"])
    pagina.locator('[data-bolsa-accion="iniciar-b7"]').click()
    pagina.locator('[data-bolsa-form="b7-paso1"] button[type="submit"]').click()
    paso2 = pagina.locator('[data-bolsa-form="b7-paso2"]')
    paso2.wait_for(timeout=15000)
    paso2.locator('[data-bolsa-accion="b7-seleccionar-todas"]').click()
    pagina.wait_for_function("""() => {
      const campo = document.querySelector('[data-bolsa-form="b7-paso2"] [data-b7-seleccion-status]');
      return campo && !campo.textContent.includes('Consultando');
    }""", timeout=15000)
    elegidos = cantidad_seleccionada(paso2.locator('[data-b7-seleccion-status]').inner_text())
    if elegidos < 2:
        return {"estado": "PUNTO_DE_CORTE", "motivo": "menos de dos personas elegibles en la bolsa sintética"}
    paso2.locator('button[type="submit"]').click()
    paso3 = pagina.locator('[data-bolsa-form="b7-paso3"]')
    paso3.wait_for(timeout=15000)
    datos = cfg.get("emision_b7", {})
    for campo in ("referencia", "centro", "fecha_inicio", "descripcion"):
        valor = datos.get(campo, "")
        if not isinstance(valor, str) or not valor.strip():
            raise NoEjecutado(f"falta dato sintético B7: {campo}")
        paso3.locator(f'[name="{campo}"]').fill(valor)
    if datos.get("plazo"):
        paso3.locator('[name="plazo"]').fill(datos["plazo"])
    paso3.locator('button[type="submit"]').click()
    paso4 = pagina.locator('[data-bolsa-form="b7-paso4"]')
    paso4.wait_for(timeout=15000)
    if paso4.locator('button[type="submit"]').is_disabled():
        return {"estado": "PUNTO_DE_CORTE", "motivo": "revisión independiente pendiente antes de emitir"}
    paso4.locator('[name="confirmacion"]').check()
    with pagina.expect_response(lambda r: r.url.split("?")[0] == origen + RUTA_EMISIONES and r.request.method == "POST") as espera:
        paso4.locator('button[type="submit"]').click()
    emision = respuesta_json(espera.value, RUTA_EMISIONES, {200, 201})
    recibo = referencia_obligatoria(emision, "recibo_ref")
    pagina.locator('[data-b7-recibo]').wait_for(timeout=15000)
    return {"estado": "EMISION_B7_REGISTRADA_API", "recibo": recibo,
            "destinatarios_seleccionados": elegidos, "entrega_corporativa": "NO ACREDITADA"}


def alta(cfg: dict, navegador) -> dict:
    origen, bolsa = cfg["origen"], cfg["bolsa_ref"]
    errores_rrhh: list[str] = []
    errores_candidato: list[str] = []
    rrhh = contexto(navegador, origen, cfg["rrhh"], errores_rrhh)
    candidato = contexto(navegador, origen, cfg["candidato"], errores_candidato)
    try:
        pagina = rrhh.new_page()
        observar_pagina(pagina, errores_rrhh, cfg)
        avisos_respuestas = []
        pagina.on("response", lambda r: avisos_respuestas.append(r)
                  if r.url.split("?")[0] == origen + RUTA_AVISOS and r.request.method == "GET" else None)
        abrir_llamamiento_rrhh(pagina, origen, bolsa)
        politica = pagina.locator('[data-rrhh-plazos-form="politica"]')
        politica.locator('[name="cantidad"]').fill("48")
        politica.locator('[name="unidad"]').select_option("horas_naturales")
        politica.locator('[name="municipio_sede"]').fill(cfg.get("municipio_sintetico", "18001"))
        politica.locator('[name="plazas_llamada"]').select_option("simultanea")
        politica.locator('[name="plazas_respuesta_horas"]').fill("24")
        politica.locator('[name="plazas_tras_renuncia"]').select_option("siguiente_en_orden")
        with pagina.expect_response(lambda r: r.url.split("?")[0] == origen + RUTA_POLITICA and r.request.method == "POST") as espera:
            politica.locator('button[type="submit"]').click()
        politica_recibo = respuesta_json(espera.value, RUTA_POLITICA, {200, 201})
        if politica_recibo.get("bolsa_ref") != bolsa:
            raise FalloRecorrido("la política no devolvió recibo de la bolsa elegida")
        recibo_politica = referencia_obligatoria(politica_recibo, "recibo_ref")
        version_politica = version_obligatoria(politica_recibo, "version")
        registrar_paso(cfg, "politica", politica_recibo)

        formulario = pagina.locator('[data-ofertas-form="publicar"]')
        for campo in ("categoria", "centro", "fecha_inicio", "descripcion"):
            formulario.locator(f'[name="{campo}"]').fill(cfg["oferta"][campo])
        formulario.locator('[name="numero_plazas"]').fill(str(cfg["oferta"].get("numero_plazas", 1)))
        if cfg["oferta"].get("fecha_fin"):
            formulario.locator('[name="fecha_fin"]').fill(cfg["oferta"]["fecha_fin"])
        with pagina.expect_response(lambda r: r.url.split("?")[0] == origen + RUTA_OFERTAS and r.request.method == "POST") as espera:
            formulario.locator('button[type="submit"]').click()
        oferta = respuesta_json(espera.value, RUTA_OFERTAS, {200, 201})
        referencia = oferta.get("oferta_ref", "")
        if not REFERENCIA.fullmatch(referencia) or oferta.get("bolsa_ref") != bolsa or oferta.get("estado") != "abierta":
            raise FalloRecorrido("la oferta no quedó abierta en la bolsa elegida")
        publicada_en = instante_obligatorio(oferta, "publicada_en")
        registrar_paso(cfg, "oferta", oferta)
        pagina.locator(f'[data-oferta-ref="{referencia}"]').first.wait_for(timeout=15000)
        capturar(cfg, pagina, "rrhh-oferta", lambda: comprobar_navegador(rrhh, pagina, errores_rrhh))

        propia = candidato.new_page()
        observar_pagina(propia, errores_candidato, cfg)
        with propia.expect_response(lambda r: r.url.split("?")[0] == origen + RUTA_MI_BOLSA and r.request.method == "GET") as espera:
            propia.goto(origen + "/area-personal/?vista=llamamientos", wait_until="domcontentloaded")
        datos_mi_bolsa = respuesta_json(espera.value, RUTA_MI_BOLSA, {200})
        visibles = datos_mi_bolsa.get("ofertas", [])
        if not any(o.get("oferta") == referencia and o.get("estado") == "abierta" for o in visibles):
            raise FalloRecorrido("el candidato no ve la oferta abierta en Mi Bolsa")
        ficha = propia.locator(f'[data-oferta-mi-bolsa="{referencia}"]')
        ficha.wait_for(timeout=15000)
        with propia.expect_response(lambda r: r.url.split("?")[0] == origen + RUTA_DISPOSICIONES and r.request.method == "POST") as espera:
            ficha.locator('form[data-portal-mi-bolsa="disposicion"] button[type="submit"]').click()
        disposicion = respuesta_json(espera.value, RUTA_DISPOSICIONES, {200, 201})
        recibo_disposicion = referencia_obligatoria(disposicion, "recibo")
        disposicion_fecha = instante_obligatorio(disposicion, "manifestada_en")
        registrar_paso(cfg, "disposicion", disposicion)
        capturar(cfg, propia, "candidato-disposicion", lambda: comprobar_navegador(candidato, propia, errores_candidato))

        # Los avisos son internos. El correo B7 se comprueba por separado: una
        # oferta publicada y su disposición no acreditan SMTP ni entrega.
        pagina.evaluate("location.hash = '#bolsa/resumen'")
        if not avisos_respuestas:
            raise FalloRecorrido("la lectura de avisos internos no apareció en el navegador")
        avisos = respuesta_json(avisos_respuestas[-1], RUTA_AVISOS, {200})
        b7 = ensayar_envio_b7(pagina, cfg)
        return {"estado": "ALTA_ACREDITADA", "politica_recibo": recibo_politica,
                "politica_version": version_politica, "oferta_ref": referencia,
                "oferta_publicada_en": publicada_en, "oferta_estado": oferta["estado"],
                "disposicion_recibo": recibo_disposicion, "disposicion_fecha": disposicion_fecha,
                "avisos": "consulta_interna_200", "avisos_total": len(avisos.get("items", [])),
                "correo_b7": b7, "mailpit": "NO COMPROBADO; relay no equivale a entrega",
                "entrega_corporativa": "NO ACREDITADA"}
    except Exception:
        capturar_corte(cfg, {"rrhh": rrhh, "candidato": candidato})
        raise
    finally:
        rrhh.close()
        candidato.close()


def recuperar(cfg: dict, navegador, anterior: dict) -> dict:
    if anterior.get("estado") != "ALTA_ACREDITADA" or not REFERENCIA.fullmatch(anterior.get("oferta_ref", "")):
        raise NoEjecutado("falta evidencia de alta para recuperar")
    recibo_politica_previo = referencia_obligatoria(anterior, "politica_recibo")
    version_politica_previa = version_obligatoria(anterior, "politica_version")
    fecha_oferta_previa = instante_obligatorio(anterior, "oferta_publicada_en")
    recibo_disposicion_previo = referencia_obligatoria(anterior, "disposicion_recibo")
    fecha_disposicion_previa = instante_obligatorio(anterior, "disposicion_fecha")
    origen, bolsa, referencia = cfg["origen"], cfg["bolsa_ref"], anterior["oferta_ref"]
    errores_rrhh: list[str] = []
    errores_candidato: list[str] = []
    rrhh = contexto(navegador, origen, cfg["rrhh"], errores_rrhh)
    candidato = contexto(navegador, origen, cfg["candidato"], errores_candidato)
    try:
        p = rrhh.new_page()
        observar_pagina(p, errores_rrhh, cfg)
        politicas = []
        p.on("response", lambda r: politicas.append(r)
             if r.url.split("?")[0] == origen + RUTA_POLITICA and r.request.method == "GET" else None)
        with p.expect_response(lambda r: r.url.split("?")[0] == origen + RUTA_OFERTAS and r.request.method == "GET") as espera:
            abrir_llamamiento_rrhh(p, origen, bolsa)
        if not politicas:
            raise FalloRecorrido("la política no se consultó tras el reinicio")
        politica = respuesta_json(politicas[-1], RUTA_POLITICA, {200})
        if (referencia_obligatoria(politica, "recibo_ref") != recibo_politica_previo
                or version_obligatoria(politica, "version") != version_politica_previa):
            raise FalloRecorrido("el recibo o la versión de la política cambió tras el reinicio")
        ofertas = respuesta_json(espera.value, RUTA_OFERTAS, {200}).get("ofertas", [])
        coincidentes = [o for o in ofertas if o.get("oferta_ref") == referencia]
        if (len(coincidentes) != 1 or instante_obligatorio(coincidentes[0], "publicada_en") != fecha_oferta_previa
                or coincidentes[0].get("estado") != anterior["oferta_estado"]):
            raise FalloRecorrido("la oferta cambió o está duplicada tras el reinicio")
        p.locator(f'[data-oferta-ref="{referencia}"]').first.wait_for(timeout=15000)
        capturar(cfg, p, "recuperacion-rrhh", lambda: comprobar_navegador(rrhh, p, errores_rrhh))
        q = candidato.new_page()
        observar_pagina(q, errores_candidato, cfg)
        with q.expect_response(lambda r: r.url.split("?")[0] == origen + RUTA_MI_BOLSA and r.request.method == "GET") as espera:
            q.goto(origen + "/area-personal/?vista=llamamientos", wait_until="domcontentloaded")
        mi_bolsa = respuesta_json(espera.value, RUTA_MI_BOLSA, {200})
        propias = [o for o in mi_bolsa.get("ofertas", []) if o.get("oferta") == referencia]
        if len(propias) != 1 or not isinstance(propias[0].get("disposicion"), dict):
            raise FalloRecorrido("la disposición falta o está duplicada tras el reinicio")
        if (referencia_obligatoria(propias[0]["disposicion"], "recibo") != recibo_disposicion_previo
                or instante_obligatorio(propias[0]["disposicion"], "manifestada_en") != fecha_disposicion_previa):
            raise FalloRecorrido("la disposición cambió o está duplicada tras el reinicio")
        q.locator(f'[data-oferta-mi-bolsa="{referencia}"]').first.wait_for(timeout=15000)
        capturar(cfg, q, "recuperacion-candidato", lambda: comprobar_navegador(candidato, q, errores_candidato))
        return {"estado": "RECUPERACION_ACREDITADA", "oferta_ref": referencia,
                "disposicion_recibo": recibo_disposicion_previo, "post_repetidos": 0,
                "entrega_corporativa": "NO ACREDITADA"}
    except Exception:
        capturar_corte(cfg, {"rrhh": rrhh, "candidato": candidato})
        raise
    finally:
        rrhh.close()
        candidato.close()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", type=Path, required=True, help="JSON privado fuera de Git")
    parser.add_argument("--fase", choices=("alta", "recuperar"), required=True)
    parser.add_argument("--evidencia", type=Path, required=True, help="JSON privado fuera de Git")
    args = parser.parse_args()
    try:
        if dentro_de_git(args.config) or dentro_de_git(args.evidencia):
            raise NoEjecutado("configuración y evidencia privadas deben quedar fuera de Git")
        if not args.evidencia.parent.is_dir() or args.evidencia.parent.stat().st_mode & 0o077:
            raise NoEjecutado("el directorio de evidencia privada debe existir con permisos 0700")
        if not args.config.is_file():
            raise NoEjecutado("falta configuración privada del clon")
        if args.config.stat().st_mode & 0o077:
            raise NoEjecutado("la configuración privada debe tener permisos 0600")
        cfg = validar_configuracion(json.loads(args.config.read_text()), args.fase)
        if args.fase == "alta" and args.evidencia.exists():
            raise NoEjecutado("ya existe evidencia: no se repite el alta")
        if args.fase == "recuperar" and not args.evidencia.is_file():
            raise NoEjecutado("falta evidencia original para recuperar")
        if args.fase == "alta":
            descriptor = os.open(args.evidencia, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(descriptor, "w", encoding="utf-8") as salida:
                salida.write('{"estado":"INICIADO","nota":"No repetir escrituras si falla; reconciliar el clon"}\n')
        capturas = args.evidencia.parent / (args.evidencia.stem + "-capturas")
        capturas.mkdir(mode=0o700, exist_ok=True)
        if dentro_de_git(capturas) or capturas.stat().st_mode & 0o077:
            raise NoEjecutado("las capturas deben quedar en un directorio privado fuera de Git")
        cfg["_capturas"] = str(capturas)
        cfg["_evidencia"] = str(args.evidencia) if args.fase == "alta" else None
        from playwright.sync_api import sync_playwright
        with sync_playwright() as playwright:
            navegador = playwright.chromium.launch(headless=True, executable_path=cfg["chrome"],
                args=["--disable-background-networking", "--disable-extensions", "--disable-sync",
                      "--no-default-browser-check"])
            try:
                resultado = alta(cfg, navegador) if args.fase == "alta" else recuperar(cfg, navegador, json.loads(args.evidencia.read_text()))
            finally:
                navegador.close()
        resultado["http"] = cfg.get("_http", [])
        resultado["pasos"] = cfg.get("_pasos", [])
        resultado["navegador"] = cfg.get("_navegador", [])
        resultado["capturas"] = sorted(p.name for p in capturas.glob("*.png"))
        if args.fase == "alta":
            args.evidencia.write_text(json.dumps(resultado, ensure_ascii=False, indent=2) + "\n")
        print(json.dumps(resultado, ensure_ascii=False))
        return 0
    except NoEjecutado as error:
        print(json.dumps({"estado": "NO EJECUTADO", "motivo": str(error)}, ensure_ascii=False))
        return 2
    except Exception as error:
        # El error se reduce al tipo: Playwright puede incluir URL o datos personales.
        mensaje = str(error) if isinstance(error, FalloRecorrido) else type(error).__name__
        resultado = {"estado": "FALLÓ", "motivo": mensaje}
        if "cfg" in locals():
            resultado["pasos"] = cfg.get("_pasos", [])
            resultado["http"] = cfg.get("_http", [])
            resultado["navegador"] = cfg.get("_navegador", [])
            if "capturas" in locals():
                resultado["capturas"] = sorted(p.name for p in capturas.glob("*.png"))
            if args.fase == "alta" and cfg.get("_evidencia"):
                args.evidencia.write_text(json.dumps(resultado, ensure_ascii=False, indent=2) + "\n")
        print(json.dumps(resultado, ensure_ascii=False))
        return 1


if __name__ == "__main__":
    sys.exit(main())
