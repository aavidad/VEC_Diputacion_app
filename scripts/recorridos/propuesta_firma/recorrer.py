#!/usr/bin/env python3
"""Recorre propuesta, seis PDF y una firma de prueba en un clon local de VEC."""

from __future__ import annotations

import argparse
from asyncio import CancelledError
import errno
import hashlib
import json
import os
import re
import shutil
import stat
import sys
from pathlib import Path
from urllib.parse import urlsplit


DOCUMENTOS = (
    ("informe-definitivo", "informe-definitivo-borrador.pdf"),
    ("resolucion", "resolucion-borrador.pdf"),
    ("diligencia", "diligencia-borrador.pdf"),
    ("toma-posesion", "toma-posesion-borrador.pdf"),
    ("notificacion", "notificacion-borrador.pdf"),
    ("comunicacion-centro", "comunicacion-centro-borrador.pdf"),
)
RUTA_DETALLE = "/api/vec/contratacion-temporal/expedientes/consultas"
RUTA_FIRMAS = "/api/vec/contratacion-temporal/firmas-documento"
RUTA_CONSULTA_FIRMAS = RUTA_FIRMAS + "/consultas"
RUTA_CIRCUITO = "/api/vec/contratacion-temporal/circuito-firma"
RUTA_SEGUIMIENTO = "/api/vec/contratacion-temporal/seguimiento-cese"
SHA256 = re.compile(r"[0-9a-f]{64}\Z")
REFERENCIA = re.compile(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}\Z")
INSTANTE_UTC = re.compile(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z\Z")


class Corte(RuntimeError):
    def __init__(self, paso: str, motivo: str):
        super().__init__(motivo)
        self.paso = paso
        self.motivo = motivo


def argumentos(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--entorno", type=Path, help="JSON externo de precondiciones del clon")
    p.add_argument("--binario", type=Path, help="binario externo instalado en el clon")
    p.add_argument("--origen", help="origen HTTPS local del clon")
    p.add_argument("--certificado", type=Path, help="certificado mTLS sintético externo")
    p.add_argument("--clave", type=Path, help="clave mTLS sintética externa")
    p.add_argument("--expediente-ref", help="referencia opaca de un expediente sintético")
    p.add_argument("--version-propuesta", type=int, help="versión de la propuesta ya registrada")
    p.add_argument("--firmar", action="store_true", help="abre AutoFirma del puesto con intervención humana")
    p.add_argument("--comparar", type=Path, help="informe anterior, para comprobar recuperación tras reinicio externo")
    p.add_argument("--salida", type=Path, help="informe JSON sin documentos ni secretos")
    p.add_argument("--captura-movil", type=Path, help="PNG sintético de 390 px, en ruta privada externa")
    p.add_argument("--captura-escritorio", type=Path, help="PNG sintético de 1440 px, en ruta privada externa")
    return p.parse_args(argv)


def validar_entrada(a):
    for ruta in (a.salida, a.captura_movil, a.captura_escritorio):
        if ruta:
            try:
                descriptor = abrir_directorio_privado(ruta)
                os.close(descriptor)
            except OSError:
                raise Corte("precondiciones", "la salida requiere un directorio propio 0700, externo a Git y sin enlaces") from None
    if not a.entorno or not a.entorno.is_file():
        raise Corte("precondiciones", "falta acreditación externa del clon H3-H5 y binario")
    try:
        entorno = json.loads(a.entorno.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        raise Corte("precondiciones", "el inventario del clon no es JSON legible") from None
    if not isinstance(entorno, dict) or entorno.get("clon") != "local" or entorno.get("datos") != "sinteticos" \
            or entorno.get("hitos") != ["H3", "H4", "H5"] \
            or not SHA256.fullmatch(str(entorno.get("binario_sha256", ""))):
        raise Corte("precondiciones", "el clon H3-H5, los datos sintéticos o el binario no están acreditados")
    try:
        u = urlsplit(a.origen or "")
        puerto = u.port
    except ValueError:
        raise Corte("precondiciones", "origen HTTPS local no válido") from None
    if u.scheme != "https" or u.hostname not in {"localhost", "127.0.0.1", "::1"} \
            or not puerto or u.username or u.password or u.path or u.query or u.fragment \
            or entorno.get("origen") != a.origen:
        raise Corte("precondiciones", "el origen debe ser HTTPS local y coincidir con el inventario")
    if not a.expediente_ref or not REFERENCIA.fullmatch(a.expediente_ref) \
            or not a.version_propuesta or a.version_propuesta < 7:
        raise Corte("precondiciones", "falta expediente sintético con versión de propuesta válida")
    if not a.certificado or not a.clave or any(not x.is_file() or x.stat().st_size == 0 for x in (a.certificado, a.clave)):
        raise Corte("precondiciones", "falta material mTLS sintético externo")
    if not a.binario or not a.binario.is_file() or not os.access(a.binario, os.X_OK):
        raise Corte("precondiciones", "falta binario externo ejecutable del clon")
    try:
        huella = hashlib.sha256()
        with a.binario.open("rb") as fichero:
            for bloque in iter(lambda: fichero.read(1024 * 1024), b""):
                huella.update(bloque)
    except OSError:
        raise Corte("precondiciones", "no se puede cotejar el binario externo") from None
    if huella.hexdigest() != entorno["binario_sha256"]:
        raise Corte("precondiciones", "la huella del binario externo no coincide con el inventario")
    if a.firmar and a.comparar:
        raise Corte("precondiciones", "la recuperación es de solo lectura; no se puede pedir otra firma")
    chrome = next((ruta for ruta in ("/usr/bin/google-chrome", "/usr/bin/chromium", "/usr/bin/chromium-browser")
                   if Path(ruta).is_file()), None)
    if not chrome:
        raise Corte("precondiciones", "falta Chrome del sistema")
    if a.firmar and not shutil.which("xdg-open"):
        raise Corte("firma_admitida", "AutoFirma del puesto no está preparada para abrir enlaces")
    return chrome, entorno


def datos_respuesta(respuesta):
    try:
        cuerpo = respuesta.json()
    except ValueError:
        raise Corte("contrato_http", "respuesta JSON ilegible") from None
    return datos_respuesta_json(cuerpo)


def datos_respuesta_json(cuerpo):
    if not isinstance(cuerpo, dict) or not isinstance(cuerpo.get("data"), dict):
        raise Corte("contrato_http", "falta envoltorio de datos")
    return cuerpo["data"]


def comprobar_propuesta(detalle, ref, version):
    resumen = detalle.get("resumen")
    if not isinstance(resumen, dict) or resumen.get("expediente_ref") != ref:
        raise Corte("propuesta", "la consulta no devolvió el expediente solicitado")
    if not isinstance(resumen.get("version"), int) or resumen["version"] < version:
        raise Corte("propuesta", "la versión de la propuesta aún no figura en el expediente")
    historia = detalle.get("hitos")
    if not isinstance(historia, list) or len(historia) < version \
            or not isinstance(historia[version - 1], dict) \
            or historia[version - 1].get("version_expediente") != version \
            or historia[version - 1].get("accion_clave") != "registrar_propuesta_formalizacion":
        raise Corte("propuesta", "la propuesta no está en la versión indicada del historial")
    if resumen.get("fase_clave") != "nombramiento" or resumen.get("estado_clave") != "en_curso":
        raise Corte("propuesta", "el expediente no está en nombramiento en curso")
    return {"version_propuesta": version, "version_actual": resumen["version"], "historia": len(historia)}


def comprobar_recibo_propuesta(datos, ref, version):
    estado = datos.get("estado")
    if not isinstance(estado, dict) or estado.get("expediente_ref") != ref:
        raise Corte("propuesta", "el seguimiento no corresponde al expediente")
    propuestas = estado.get("propuestas")
    if not isinstance(propuestas, list):
        raise Corte("propuesta", "el seguimiento no devuelve las propuestas conservadas")
    coincidencias = [p for p in propuestas if isinstance(p, dict) and p.get("version_resultante") == version]
    if len(coincidencias) != 1 or not REFERENCIA.fullmatch(str(coincidencias[0].get("recibo_ref", ""))):
        raise Corte("propuesta", "falta un recibo único para la versión de propuesta")
    return {"recibo_ref": coincidencias[0]["recibo_ref"], "confirmada_en": coincidencias[0].get("confirmada_en")}


def resumen_firma(data, ref, documento):
    if data.get("expediente_ref") != ref or data.get("documento") != documento \
            or data.get("resultado") != "firmado" or data.get("firma_eficaz") is not False \
            or data.get("firma_verificada") is not True:
        raise Corte("verificacion", "el recibo no confirma la firma de prueba pedida")
    verificacion = data.get("verificacion")
    if not isinstance(verificacion, dict) or verificacion.get("estado") != "valida" \
            or verificacion.get("motivo") != "verificada" \
            or not SHA256.fullmatch(str(verificacion.get("firmado_sha256", ""))):
        raise Corte("verificacion", "el recibo no acredita verificación positiva de GrxFirma")
    if not REFERENCIA.fullmatch(str(data.get("recibo_ref", ""))) \
            or not INSTANTE_UTC.fullmatch(str(data.get("registrada_en", ""))) \
            or not isinstance(data.get("paso_orden"), int) or data["paso_orden"] < 1:
        raise Corte("verificacion", "faltan recibo, fecha u orden de la firma")
    return {"recibo_ref": data["recibo_ref"], "firma_ref": data.get("firma_ref"),
            "firmado_sha256": verificacion["firmado_sha256"], "registrada_en": data["registrada_en"],
            "estado": "firmado", "paso_orden": data["paso_orden"], "firma_eficaz": False}


def firma_recuperada(estado, recibo_ref):
    """Extrae solo los campos que devuelve la consulta CT118; no revalida firma_ref ni SHA."""
    if estado.get("firma_eficaz") is not False or not REFERENCIA.fullmatch(str(recibo_ref or "")):
        raise Corte("recuperacion", "estado de firma no comparable")
    documentos = estado.get("documentos")
    if not isinstance(documentos, list):
        raise Corte("recuperacion", "consulta de firmas sin documentos")
    encontrados = [(p, p.get("orden")) for d in documentos if isinstance(d, dict)
                   and d.get("documento") == "informe_definitivo" for p in d.get("pasos", [])
                   if isinstance(p, dict) and p.get("recibo_ref") == recibo_ref]
    if len(encontrados) != 1:
        raise Corte("recuperacion", "el recibo no figura una sola vez en el informe")
    paso, orden = encontrados[0]
    if paso.get("estado") != "firmado" or not INSTANTE_UTC.fullmatch(str(paso.get("registrada_en", ""))) \
            or not isinstance(orden, int) or orden < 1:
        raise Corte("recuperacion", "el paso recuperado no conserva firma, fecha u orden")
    return {"recibo_ref": recibo_ref, "registrada_en": paso["registrada_en"],
            "estado": paso["estado"], "paso_orden": orden}


def comparar_campos_firma(original, recuperada):
    if not isinstance(original, dict) or not isinstance(recuperada, dict) \
            or not REFERENCIA.fullmatch(str(original.get("recibo_ref", ""))) \
            or not INSTANTE_UTC.fullmatch(str(original.get("registrada_en", ""))) \
            or original.get("estado") != "firmado" \
            or not isinstance(original.get("paso_orden"), int) or original["paso_orden"] < 1 \
            or any(original.get(campo) != recuperada.get(campo)
                   for campo in ("recibo_ref", "registrada_en", "estado", "paso_orden")):
        raise Corte("recuperacion", "recibo, fecha, estado u orden de firma cambiaron")


def comparar_recuperacion(informe, ruta):
    try:
        previo = json.loads(ruta.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        raise Corte("recuperacion", "falta informe anterior legible") from None
    if previo.get("expediente_ref") != informe["expediente_ref"] \
            or previo.get("propuesta") != informe.get("propuesta") \
            or previo.get("pdf") != informe.get("pdf"):
        raise Corte("recuperacion", "propuesta o huellas PDF cambiaron tras el reinicio")
    if previo.get("firma"):
        comparar_campos_firma(previo["firma"], informe.get("firma_recuperada"))
    return True


def limitar_peticion_cdp(cdp, evento, origen):
    """Mantiene Chrome nativo y corta otros orígenes y respuestas 3xx."""
    if evento.get("responseErrorReason"):
        raise Corte("guardia_red", "la guardia recibió un fallo de transporte")
    identificador = evento["requestId"]
    esperado = urlsplit(origen)
    solicitado = urlsplit(evento["request"]["url"])
    if solicitado.username or solicitado.password or \
            (solicitado.scheme, solicitado.hostname, solicitado.port) != \
            (esperado.scheme, esperado.hostname, esperado.port) \
            or 300 <= evento.get("responseStatusCode", 0) < 400:
        cdp.send("Fetch.failRequest", {"requestId": identificador, "errorReason": "BlockedByClient"})
    else:
        # route.fetch añade Connection: keep-alive con otro cliente HTTP.
        # CDP conserva la petición y el transporte originales del navegador.
        cdp.send("Fetch.continueRequest", {"requestId": identificador})


class GuardiaNavegador:
    """Instala una única guardia global antes de crear contextos o páginas."""

    def __init__(self, browser, origen):
        self.browser = browser
        self.errores = []
        self.cerrando = False
        try:
            # Canal de cierre independiente: no instala ningún interceptor.
            self.control = browser.new_browser_cdp_session()
            self.cdp = browser.new_browser_cdp_session()
            self.cdp.on("Fetch.requestPaused", self.interceptar)
            self.cdp.on("close", self.sesion_perdida)
            browser.on("disconnected", self.desconectado)
            self.origen = origen
            self.cdp.send("Fetch.enable", {"patterns": [
                {"urlPattern": "*", "requestStage": "Request"},
                {"urlPattern": "*", "requestStage": "Response"},
            ]})
        except Exception:
            self.cerrar()
            raise Corte("guardia_red", "no se pudo instalar la guardia global del navegador") from None

    def interceptar(self, evento):
        if self.cerrando:
            return
        try:
            limitar_peticion_cdp(self.cdp, evento, self.origen)
        except Exception as e:
            if not self.cerrando:
                self.solicitar_cierre(type(e).__name__)

    def desconectado(self):
        if not self.cerrando:
            self.errores.append("navegador_desconectado")
            self.cerrando = True

    def sesion_perdida(self, *_):
        if not self.cerrando:
            self.solicitar_cierre("sesion_guardia_desconectada")

    def solicitar_cierre(self, motivo):
        self.errores.append(motivo)
        self.cerrando = True
        # Browser.close por CDP no reentra en el cierre síncrono de Playwright;
        # el canal de control conserva esta orden si se pierde la sesión Fetch.
        try:
            self.control.send("Browser.close")
        except (Exception, CancelledError):
            # El finally vuelve a cerrar el proceso propio y confirma el cierre.
            pass

    def cerrar(self):
        # Mantener Fetch activo hasta cerrar todo Chrome, incluidos sus targets.
        self.cerrando = True
        self.browser.close()


def limitar_websocket(route, permitir_autofirma):
    destino = urlsplit(route.url)
    if permitir_autofirma and (destino.scheme, destino.hostname, destino.port, destino.path) == \
            ("wss", "127.0.0.1", 63117, "") and not destino.query and not destino.fragment:
        route.connect_to_server()
    else:
        route.close()


def abrir_directorio_privado(ruta):
    """Abre el padre sin seguir enlaces ni aceptar repositorios o worktrees."""
    ruta = Path(ruta)
    if ".." in ruta.parts or not ruta.name:
        raise OSError(errno.EPERM, "ruta privada no válida")
    absoluta = ruta if ruta.is_absolute() else Path.cwd() / ruta
    descriptor = os.open(absoluta.anchor, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        for componente in (*absoluta.parent.parts[1:], None):
            try:
                os.stat(".git", dir_fd=descriptor, follow_symlinks=False)
            except FileNotFoundError:
                pass
            else:
                raise OSError(errno.EPERM, "no se guardan artefactos dentro de Git")
            try:
                cabecera = os.stat("HEAD", dir_fd=descriptor, follow_symlinks=False)
                objetos = os.stat("objects", dir_fd=descriptor, follow_symlinks=False)
                configuracion = os.stat("config", dir_fd=descriptor, follow_symlinks=False)
            except FileNotFoundError:
                pass
            else:
                if stat.S_ISREG(cabecera.st_mode) and stat.S_ISDIR(objetos.st_mode) \
                        and stat.S_ISREG(configuracion.st_mode):
                    raise OSError(errno.EPERM, "no se guardan artefactos dentro de Git bare")
            if componente is not None:
                siguiente = os.open(componente, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW,
                                    dir_fd=descriptor)
                os.close(descriptor)
                descriptor = siguiente
        padre = os.fstat(descriptor)
        if padre.st_uid != os.getuid() or stat.S_IMODE(padre.st_mode) != 0o700:
            raise OSError(errno.EPERM, "el directorio de salida debe ser propio y 0700")
        return descriptor
    except BaseException:
        os.close(descriptor)
        raise


def guardar_privado(ruta, contenido):
    directorio = abrir_directorio_privado(ruta)
    try:
        descriptor = os.open(Path(ruta).name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                             0o600, dir_fd=directorio)
        with os.fdopen(descriptor, "wb") as fichero:
            fichero.write(contenido)
    finally:
        os.close(directorio)


def capturar_corte(page, a, informe):
    """Conserva la pantalla alcanzada, también si falta una propuesta o un PDF."""
    for nombre, ancho, alto, ruta in (
        ("escritorio_1440", 1440, 900, a.captura_escritorio),
        ("movil_390", 390, 844, a.captura_movil),
    ):
        if not ruta or (nombre == "movil_390" and informe.get(nombre, {}).get("captura_guardada")):
            continue
        try:
            page.set_viewport_size({"width": ancho, "height": alto})
            captura = page.screenshot(full_page=True)
            guardar_privado(ruta, captura)
            dimensiones = page.evaluate("""() => ({ancho: document.documentElement.clientWidth,
              contenido: document.documentElement.scrollWidth})""")
            informe[nombre] = {"ancho": ancho, "captura_guardada": True,
                               "captura_sha256": hashlib.sha256(captura).hexdigest(),
                               "sin_desbordamiento": dimensiones["contenido"] <= dimensiones["ancho"]}
        except Exception as e:
            informe[nombre] = {"ancho": ancho, "captura_guardada": False,
                               "error": type(e).__name__}


def comprobar_movil(page, a, pdf_escritorio):
    """Repite solo lecturas y descargas a 390 px; no firma ni registra efectos."""
    page.set_viewport_size({"width": 390, "height": 844})
    try:
        for accion, nombre in DOCUMENTOS:
            boton = page.locator(f'[data-ct-exp-accion="descargar-{accion}"]').first
            if not boton.is_visible() or not boton.is_enabled():
                raise Corte("movil_390", f"no se puede descargar {accion} a 390 px")
            with page.expect_download(timeout=30_000) as espera_descarga:
                with page.expect_response(lambda r: urlsplit(r.url).path == RUTA_DETALLE and r.request.method == "POST", timeout=30_000) as espera_pdf:
                    boton.click()
            respuesta = espera_pdf.value
            descarga = espera_descarga.value
            if respuesta.status != 200 or descarga.suggested_filename != nombre:
                raise Corte("movil_390", f"{accion}: respuesta o descarga móvil distinta")
            bytes_pdf = Path(descarga.path()).read_bytes()
            if len(bytes_pdf) != pdf_escritorio[accion]["bytes"] \
                    or hashlib.sha256(bytes_pdf).hexdigest() != pdf_escritorio[accion]["sha256"]:
                raise Corte("movil_390", f"{accion}: bytes distintos del escritorio")
        dimensiones = page.evaluate("""() => ({ancho: document.documentElement.clientWidth,
          contenido: document.documentElement.scrollWidth})""")
        if dimensiones["ancho"] != 390 or dimensiones["contenido"] > dimensiones["ancho"]:
            raise Corte("movil_390", "la página desborda horizontalmente a 390 px")
        captura = page.screenshot(full_page=True)
        if a.captura_movil:
            try:
                guardar_privado(a.captura_movil, captura)
            except OSError:
                raise Corte("movil_390", "no se pudo crear la captura privada") from None
        return {"ancho": 390, "pdf_identicos": len(DOCUMENTOS),
                "sin_desbordamiento": True, "captura_sha256": hashlib.sha256(captura).hexdigest(),
                "captura_guardada": bool(a.captura_movil)}
    finally:
        page.set_viewport_size({"width": 1440, "height": 900})


def recorrer(a, chrome, entorno):
    try:
        from playwright.sync_api import sync_playwright, TimeoutError as PlaywrightTimeout
    except ImportError:
        raise Corte("precondiciones", "Playwright de Python no está instalado") from None
    informe = {"estado": "CORTE", "entorno": "clon_local_h3_h5", "binario_sha256": entorno["binario_sha256"],
               "expediente_ref": a.expediente_ref, "pdf": {}, "firma": None, "http": []}
    with sync_playwright() as pw:
        browser = pw.chromium.launch(executable_path=chrome, headless=not a.firmar)
        guardia = GuardiaNavegador(browser, a.origen)
        context = browser.new_context(
            client_certificates=[{"origin": a.origen, "certPath": str(a.certificado), "keyPath": str(a.clave)}],
            ignore_https_errors=False, service_workers="block", accept_downloads=True,
            locale="es-ES", timezone_id="Europe/Madrid", viewport={"width": 1440, "height": 900},
        )
        page = context.new_page()
        errores_js = []
        cookies_http = []
        page.on("pageerror", lambda e: errores_js.append(type(e).__name__))
        page.on("response", lambda r: cookies_http.append(urlsplit(r.url).path) if "set-cookie" in r.headers else None)
        rutas_observadas = {"/portal-empleado/", RUTA_DETALLE, RUTA_FIRMAS,
                            RUTA_CONSULTA_FIRMAS, RUTA_CIRCUITO, RUTA_SEGUIMIENTO,
                            "/api/vec/contratacion-temporal/cuadro/consultas",
                            "/api/vec/contratacion-temporal/catalogos-alta"}
        page.on("response", lambda r: informe["http"].append({"metodo": r.request.method,
                "ruta": urlsplit(r.url).path, "estado": r.status})
                if urlsplit(r.url).path in rutas_observadas else None)
        try:
            # Nunca se navega al servicio GrxFirma desde el navegador: VEC lo invoca en servidor.
            context.route_web_socket("**/*", lambda route: limitar_websocket(route, a.firmar))
            pagina = page.goto(a.origen + "/portal-empleado/#contratacion-temporal", wait_until="domcontentloaded", timeout=30_000)
            if not pagina or pagina.status != 200:
                raise Corte("portal", "el portal RRHH local no respondió 200")
            abrir = page.locator(f'[data-ct-exp-abrir="{a.expediente_ref}"]')
            try:
                abrir.wait_for(timeout=20_000)
            except PlaywrightTimeout:
                raise Corte("propuesta", "el expediente no aparece en la bandeja autorizada") from None
            with page.expect_response(lambda r: urlsplit(r.url).path == RUTA_DETALLE and r.request.method == "POST", timeout=20_000) as espera:
                abrir.click()
            respuesta = espera.value
            if respuesta.status != 200:
                raise Corte("propuesta", f"consulta del expediente: HTTP {respuesta.status}")
            informe["propuesta"] = comprobar_propuesta(datos_respuesta(respuesta), a.expediente_ref, a.version_propuesta)
            seguimiento = page.evaluate("""async ([ref, ruta]) => {
              const r = await fetch(ruta, {
                method: 'POST', headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
                body: JSON.stringify({expediente_ref: ref}), credentials: 'same-origin', cache: 'no-store',
                redirect: 'error', referrerPolicy: 'no-referrer'
              });
              const texto = await r.text();
              let datos = null;
              try { if (texto.length <= 131072) datos = JSON.parse(texto); } catch {}
              return {status: r.status, datos};
            }""", [a.expediente_ref, RUTA_SEGUIMIENTO])
            if seguimiento["status"] != 200 or not isinstance(seguimiento.get("datos"), dict):
                raise Corte("propuesta", f"seguimiento de propuesta: HTTP {seguimiento['status']}")
            informe["propuesta"].update(comprobar_recibo_propuesta(datos_respuesta_json(seguimiento["datos"]),
                                                                      a.expediente_ref, a.version_propuesta))
            for accion, nombre in DOCUMENTOS:
                boton = page.locator(f'[data-ct-exp-accion="descargar-{accion}"]').first
                if not boton.is_visible() or not boton.is_enabled():
                    raise Corte("borradores", f"no está disponible {accion}")
                with page.expect_download(timeout=30_000) as espera_descarga:
                    with page.expect_response(lambda r: urlsplit(r.url).path == RUTA_DETALLE and r.request.method == "POST", timeout=30_000) as espera_pdf:
                        boton.click()
                r = espera_pdf.value
                if r.status != 200:
                    raise Corte("borradores", f"{accion}: HTTP {r.status}")
                descarga = espera_descarga.value
                if descarga.suggested_filename != nombre:
                    raise Corte("borradores", f"nombre de {accion} inesperado")
                bytes_pdf = Path(descarga.path()).read_bytes()
                if not bytes_pdf.startswith(b"%PDF-") or len(bytes_pdf) > 2 * 1024 * 1024:
                    raise Corte("borradores", f"{accion} no es un PDF acotado")
                informe["pdf"][accion] = {"nombre": nombre, "bytes": len(bytes_pdf),
                                           "sha256": hashlib.sha256(bytes_pdf).hexdigest(), "http": 200}
            informe["movil_390"] = comprobar_movil(page, a, informe["pdf"])
            # El circuito se monta al abrir la ficha; se vuelve a abrir sin repetir la propuesta.
            page.reload(wait_until="domcontentloaded")
            abrir.wait_for(timeout=20_000)
            with page.expect_response(lambda r: urlsplit(r.url).path == RUTA_CIRCUITO and r.request.method == "GET", timeout=20_000) as espera_circuito:
                abrir.click()
            circuito = espera_circuito.value
            if circuito.status != 200:
                raise Corte("firma_admitida", f"catálogo de firma: HTTP {circuito.status}")
            catalogo = datos_respuesta(circuito)
            if catalogo.get("firma_eficaz") is not False:
                raise Corte("firma_admitida", "el catálogo no declara el límite de eficacia")
            if a.comparar:
                try:
                    anterior = json.loads(a.comparar.read_text(encoding="utf-8"))
                except (OSError, ValueError):
                    raise Corte("recuperacion", "falta informe anterior legible") from None
                firma_anterior = anterior.get("firma")
                if firma_anterior:
                    page.reload(wait_until="domcontentloaded")
                    abrir = page.locator(f'[data-ct-exp-abrir="{a.expediente_ref}"]')
                    abrir.wait_for(timeout=20_000)
                    with page.expect_response(lambda x: urlsplit(x.url).path == RUTA_CONSULTA_FIRMAS and x.request.method == "POST", timeout=20_000) as espera_consulta:
                        abrir.click()
                    consulta = espera_consulta.value
                    if consulta.status != 200:
                        raise Corte("recuperacion", f"consulta de firmas: HTTP {consulta.status}")
                    informe["firma_recuperada"] = firma_recuperada(datos_respuesta(consulta), firma_anterior.get("recibo_ref"))
                    informe["campos_firma_no_revalidados"] = ["firma_ref", "firmado_sha256"]
                informe["comparacion_con_informe_anterior"] = comparar_recuperacion(informe, a.comparar)
                if not firma_anterior:
                    raise Corte("firma_admitida", "PDF y propuesta recuperados; todavía falta firma admitida")
                raise Corte("custodia", "recibo de firma recuperado; falta recibo verificable de custodia")
            if not a.firmar:
                raise Corte("firma_admitida", "falta firma admitida; no se abrió AutoFirma")
            detalles = page.locator("[data-ct-firma-detalles]")
            detalles.wait_for(timeout=20_000)
            detalles.evaluate("e => e.open = true")
            boton = page.locator('[data-ct-firma-accion="firmar"][data-ct-firma-documento="informe_definitivo"]').first
            if not boton.is_visible() or not boton.is_enabled():
                raise Corte("firma_admitida", "no hay paso de firma habilitado para el informe")
            try:
                with page.expect_response(lambda r: urlsplit(r.url).path == RUTA_FIRMAS and r.request.method == "POST", timeout=180_000) as espera_firma:
                    boton.click()
            except PlaywrightTimeout:
                raise Corte("firma_admitida", "AutoFirma no devolvió una firma al portal dentro del plazo") from None
            r = espera_firma.value
            if r.status not in (200, 201):
                raise Corte("verificacion", f"registro de firma: HTTP {r.status}")
            informe["firma"] = resumen_firma(datos_respuesta(r), a.expediente_ref, "informe_definitivo")
            page.reload(wait_until="domcontentloaded")
            abrir = page.locator(f'[data-ct-exp-abrir="{a.expediente_ref}"]')
            abrir.wait_for(timeout=20_000)
            with page.expect_response(lambda x: urlsplit(x.url).path == RUTA_CONSULTA_FIRMAS and x.request.method == "POST", timeout=20_000) as espera_consulta:
                abrir.click()
            consulta = espera_consulta.value
            if consulta.status != 200:
                raise Corte("recuperacion", f"consulta de firmas: HTTP {consulta.status}")
            informe["firma_recuperada"] = firma_recuperada(datos_respuesta(consulta), informe["firma"]["recibo_ref"])
            comparar_campos_firma(informe["firma"], informe["firma_recuperada"])
            informe["campos_firma_no_revalidados"] = ["firma_ref", "firmado_sha256"]
            # CT118 conserva huellas y recibo. No hay recibo de custodia del original y firmado.
            raise Corte("custodia", "falta recibo verificable de custodia del original y el firmado")
        except Corte as e:
            informe["estado"] = "CORTE"
            informe["corte"] = e.paso
            informe["motivo"] = e.motivo
            return informe
        except Exception as e:
            informe["estado"] = "CORTE"
            informe["corte"] = "error_no_clasificado"
            informe["motivo"] = type(e).__name__
            return informe
        finally:
            capturar_corte(page, a, informe)
            informe["sin_errores_js"] = not errores_js
            informe["sin_errores_intercepcion"] = not guardia.errores
            try:
                informe["sin_cookies_http"] = not cookies_http and not context.cookies()
            except Exception:
                informe["sin_cookies_http"] = False
            try:
                informe["sin_almacenamiento_web"] = page.evaluate("""async () =>
                  localStorage.length === 0 && sessionStorage.length === 0 &&
                  (await indexedDB.databases()).length === 0 && (await caches.keys()).length === 0""")
            except Exception:
                informe["sin_almacenamiento_web"] = False
            if not all(informe[k] for k in ("sin_errores_js", "sin_errores_intercepcion",
                                           "sin_cookies_http", "sin_almacenamiento_web")):
                informe["estado"] = "CORTE"
                informe["corte"] = "controles_navegador"
                informe["motivo"] = "fallo de la guardia de red, errores JavaScript, cookies o almacenamiento web detectados"
            if guardia.errores:
                informe["corte"] = "guardia_red"
                informe["motivo"] = "la guardia global falló; se cerró el navegador"
            guardia.cerrar()
    return informe


def main(argv=None):
    a = argumentos(argv)
    informe = {"estado": "NO EJECUTADO", "corte": "precondiciones"}
    try:
        chrome, entorno = validar_entrada(a)
        informe = recorrer(a, chrome, entorno)
    except Corte as e:
        informe["estado"] = "NO EJECUTADO" if e.paso == "precondiciones" else "CORTE"
        informe["corte"] = e.paso
        informe["motivo"] = e.motivo
    except Exception as e:
        informe["estado"] = "CORTE"
        informe["corte"] = "error_no_clasificado"
        informe["motivo"] = type(e).__name__
    salida = json.dumps(informe, ensure_ascii=False, sort_keys=True)
    if a.salida:
        try:
            guardar_privado(a.salida, (salida + "\n").encode("utf-8"))
        except OSError:
            print("No se pudo crear el informe privado sin sobrescribir un archivo", file=sys.stderr)
            return 2
    print(salida)
    return 0 if informe["estado"] == "COMPLETO" else 2


if __name__ == "__main__":
    sys.exit(main())
