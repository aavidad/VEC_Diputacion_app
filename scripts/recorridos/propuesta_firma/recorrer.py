#!/usr/bin/env python3
"""Recorre propuesta, PDF y firmas V2 de prueba en un clon local de VEC.

Primera firma: --firmar --paso 1 --salida <privado/primera.json>.
Segunda identidad: --firmar --paso 2 --continuar <privado/primera.json>
                  --salida <privado/segunda.json>, con otro certificado mTLS.
Tras reinicio externo: --comparar <privado/segunda.json>
                      --reinicio <privado/reinicio.json> --salida <privado/recuperada.json>.
Cada ejecución conserva sus argumentos habituales de clon, binario y expediente.
No sustituye autoridades ausentes ni acredita eficacia administrativa.
"""

from __future__ import annotations

import argparse
import base64
from asyncio import CancelledError
import errno
import hashlib
from importlib import import_module, metadata
import json
import os
import re
import shutil
import stat
import sys
import ssl
from pathlib import Path
from urllib.parse import urlsplit

import recuperacion_nominal


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
RUTA_REGISTRO_V2 = RUTA_FIRMAS + "/registro-vec"
RUTA_ORIGINAL = RUTA_FIRMAS + "/original"
RUTA_PREFLIGHT = "/api/vec/contratacion-temporal/firma/preflight"
RUTA_CONSULTA_FIRMAS = RUTA_FIRMAS + "/consultas"
RUTA_CONSULTA_TECNICA_V2 = RUTA_FIRMAS + "/consultas-v2"
RUTA_CIRCUITO = "/api/vec/contratacion-temporal/circuito-firma"
RUTA_SEGUIMIENTO = "/api/vec/contratacion-temporal/seguimiento-cese"
SHA256 = re.compile(r"[0-9a-f]{64}\Z")
REFERENCIA = re.compile(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}\Z")
INSTANTE_UTC = re.compile(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z\Z")
CAMPOS_RECIBO_V2 = {"esquema", "recibo_ref", "firma_ref", "ya_registrada", "expediente_ref", "version_expediente",
    "documento", "paso_orden", "paso_ref", "secuencia", "registrada_en", "documento_custodiado",
    "verificacion_tecnica", "firma_eficaz", "material_root_sha256", "revision_pdf"}
CAMPOS_RECIBO_ANIDADOS = {
    "documento_custodiado": {"expediente_ref", "documento_ref", "version", "huella_sha256"},
    "verificacion_tecnica": {"estado", "motivo", "politica", "revocacion", "sello_tiempo", "original_sha256", "firmado_sha256"},
    "revision_pdf": {"orden_firma", "entrada_sha256", "revision_sha256", "evidencia_sha256"},
}


def proyectar_recibo_v2(data):
    """Solo datos técnicos del contrato; nunca propagar campos de entrada extra."""
    salida = {k:data[k] for k in CAMPOS_RECIBO_V2 if k in data}
    for k,permitidos in CAMPOS_RECIBO_ANIDADOS.items():
        if isinstance(salida.get(k),dict):
            salida[k] = {campo:salida[k][campo] for campo in permitidos if campo in salida[k]}
    return salida


def informe_publico(informe):
    campos = {"estado", "corte", "motivo", "entorno", "binario_sha256", "expediente_ref", "documento", "pdf", "firma",
        "http", "registro_incierto", "firmas_v2", "propuesta", "movil_390", "escritorio_1440", "reinicio",
        "canal_certificado_sha256", "preflight_v2", "pdf_primera_revision", "pdf_firmado", "pendiente", "e2e",
        "campos_no_revalidados_por_consulta", "verificacion_criptografica_repetida", "firma_eficaz",
        "sin_errores_js", "sin_errores_intercepcion", "sin_cookies_http", "sin_almacenamiento_web", "consulta_tecnica_v2"}
    salida = {k:informe[k] for k in campos if k in informe}
    if isinstance(salida.get("firmas_v2"),list):
        salida["firmas_v2"] = [proyectar_recibo_v2(rec) for rec in salida["firmas_v2"] if isinstance(rec,dict)]
    return salida


class Corte(RuntimeError):
    def __init__(self, paso: str, motivo: str):
        super().__init__(motivo)
        self.paso = paso
        self.motivo = motivo


def comprobar_sdk_playwright():
    """Lee el pin local y verifica el SDK antes de exponer el lector importado."""
    try:
        lineas = [linea.strip() for linea in Path(__file__).with_name("requirements.txt")
                  .read_text(encoding="utf-8").splitlines()
                  if linea.strip() and not linea.lstrip().startswith("#")]
        pin = re.fullmatch(r"playwright==([0-9]+\.[0-9]+\.[0-9]+)", lineas[0]) if len(lineas) == 1 else None
        if not pin:
            raise ValueError
        requerida = pin.group(1)
    except (OSError, ValueError):
        raise Corte("precondiciones", "falta un catálogo válido de la dependencia Playwright") from None
    try:
        distribucion = metadata.distribution("playwright")
        if distribucion.version != requerida:
            raise Corte("precondiciones", "Playwright debe coincidir con la versión exacta del catálogo")
        # La metadata puede proceder de otra instalación presente en sys.path.
        # Contrastar también los módulos realmente importados, sin abrir Chrome.
        for nombre, relativo in (("playwright", "playwright/__init__.py"),
                                 ("playwright.sync_api", "playwright/sync_api/__init__.py"),
                                 ("playwright._impl._cdp_session", "playwright/_impl/_cdp_session.py")):
            modulo = import_module(nombre)
            actual = Path(modulo.__file__).resolve(strict=True)
            declarado = Path(distribucion.locate_file(relativo)).resolve(strict=True)
            if actual != declarado:
                raise Corte("precondiciones", "el SDK importado no corresponde a la dependencia verificada")
    except (metadata.PackageNotFoundError, ImportError, OSError, TypeError):
        raise Corte("precondiciones", "no se pudo verificar la dependencia Playwright instalada") from None
    return requerida


def argumentos(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--entorno", type=Path, help="JSON externo de precondiciones del clon")
    p.add_argument("--binario", type=Path, help="binario externo instalado en el clon")
    p.add_argument("--origen", help="origen HTTPS local del clon")
    p.add_argument("--certificado", type=Path, help="certificado mTLS sintético externo")
    p.add_argument("--clave", type=Path, help="clave mTLS sintética externa")
    p.add_argument("--expediente-ref", help="referencia opaca de un expediente sintético")
    p.add_argument("--version-propuesta", type=int, help="versión de la propuesta ya registrada")
    p.add_argument("--firmar", action="store_true", help="abre AutoFirma para un paso V2 con intervención humana")
    p.add_argument("--paso", type=int, choices=(1, 2), default=1, help="paso V2 a registrar; el servidor decide el pendiente")
    p.add_argument("--documento", default="informe_definitivo", choices=tuple(x[0].replace("-", "_") for x in DOCUMENTOS))
    p.add_argument("--continuar", type=Path, help="informe privado confirmado de la primera firma, para la segunda identidad")
    p.add_argument("--reinicio", type=Path, help="acta privada del reinicio externo para --comparar")
    p.add_argument("--comparar", type=Path, help="informe anterior, para comprobar recuperación tras reinicio externo")
    p.add_argument("--recuperacion-nominal", action="store_true", help="recuperacion_nominal_v2")
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
    if a.firmar and (not a.salida or (a.paso == 2 and not a.continuar) or (a.paso == 1 and a.continuar)):
        raise Corte("precondiciones", "cada firma requiere salida privada; el paso 2 requiere el informe del paso 1")
    if a.comparar and (not a.reinicio or a.continuar):
        raise Corte("precondiciones", "la recuperación requiere acta de reinicio y no acepta continuación")
    if a.continuar and not a.firmar:
        raise Corte("precondiciones", "continuar se usa únicamente para registrar el segundo paso")
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


def leer_informe_privado(ruta):
    directorio = abrir_directorio_privado(ruta)
    try:
        fd = os.open(Path(ruta).name, os.O_RDONLY | os.O_NOFOLLOW, dir_fd=directorio)
        with os.fdopen(fd, "rb") as f:
            st = os.fstat(f.fileno())
            if not stat.S_ISREG(st.st_mode) or st.st_uid != os.getuid() or st.st_nlink != 1 \
                    or stat.S_IMODE(st.st_mode) != 0o600 or st.st_size > 1024 * 1024:
                raise Corte("precondiciones", "el informe debe ser privado, propio y acotado")
            valor = json.loads(f.read())
        if not isinstance(valor, dict):
            raise Corte("precondiciones", "el informe privado no es un objeto")
        return valor
    except (OSError, ValueError):
        raise Corte("precondiciones", "no se pudo leer el informe privado") from None
    finally:
        os.close(directorio)


def huella_certificado_canal(ruta):
    try:
        pem = Path(ruta).read_text(encoding="ascii")
        if len(pem) > 65536 or pem.count("-----BEGIN CERTIFICATE-----") != 1 or pem.count("-----END CERTIFICATE-----") != 1:
            raise ValueError
        der = ssl.PEM_cert_to_DER_cert(pem)
        return hashlib.sha256(der).hexdigest()
    except (OSError, ValueError, UnicodeError):
        raise Corte("precondiciones", "el certificado del canal debe ser PEM de una sola hoja") from None


def preflight_v2(datos, solicitud):
    campos = {"esquema", "version_expediente", "documento", "catalogo_ref", "catalogo_huella",
              "paso_pendiente", "original_ref", "original_version", "vias_disponibles",
              "entrada_documento_ref", "entrada_documento_version", "entrada_documento_sha256"}
    if not isinstance(datos,dict) or set(datos) != campos or datos.get("esquema") != "vec.contratacion-temporal.preflight-firma.v2" \
            or datos.get("version_expediente") != solicitud["version"] \
            or datos.get("documento") != solicitud["documento"] \
            or not REFERENCIA.fullmatch(str(datos.get("original_ref", ""))) \
            or datos.get("original_ref") != solicitud["original_ref"] \
            or datos.get("original_version") != solicitud["original_version"] \
            or datos.get("catalogo_ref") != solicitud["catalogo_ref"] \
            or datos.get("catalogo_huella") != solicitud["catalogo_huella"] \
            or type(datos.get("paso_pendiente")) is not int or datos["paso_pendiente"] not in (0, 1, 2) \
            or not isinstance(datos.get("vias_disponibles"), list) \
            or any(v not in ("certificado_vec", "portafirmas_registro_rrhh") for v in datos["vias_disponibles"]) \
            or len(set(datos["vias_disponibles"])) != len(datos["vias_disponibles"]):
        raise Corte("contrato_http", "el preflight no cumple el contrato V2 del documento")
    if datos["paso_pendiente"] == 0:
        if datos["vias_disponibles"] or datos["entrada_documento_ref"] != "" \
                or datos["entrada_documento_version"] != 0 or datos["entrada_documento_sha256"] != "":
            raise Corte("contrato_http", "el preflight completo conserva una entrada o vía activa")
    elif not REFERENCIA.fullmatch(str(datos["entrada_documento_ref"])) \
            or type(datos["entrada_documento_version"]) is not int or datos["entrada_documento_version"] < 1 \
            or not SHA256.fullmatch(str(datos["entrada_documento_sha256"])) \
            or (datos["paso_pendiente"] == 1 and (datos["entrada_documento_ref"] != datos["original_ref"]
                or datos["entrada_documento_version"] != datos["original_version"])) \
            or (datos["paso_pendiente"] == 2 and datos["entrada_documento_ref"] == datos["original_ref"]):
        raise Corte("contrato_http", "el preflight no acredita la revisión de entrada")
    return datos


def recibo_v2(data, solicitud, enviado_sha256, estado_http):
    claves = CAMPOS_RECIBO_V2
    if not isinstance(data,dict):
        raise Corte("verificacion","el recibo V2 no es un objeto")
    c = data.get("documento_custodiado", {})
    v = data.get("verificacion_tecnica", {})
    r = data.get("revision_pdf", {})
    if set(data) != claves or data.get("esquema") != "vec.contratacion-temporal.registro-firma-vec.v2" \
            or data.get("firma_eficaz") is not False or estado_http not in (200, 201) \
            or type(data.get("ya_registrada")) is not bool or data["ya_registrada"] != (estado_http == 200) \
            or any(data.get(k) != solicitud[k] for k in ("expediente_ref", "version_expediente", "documento", "paso_orden")) \
            or any(not REFERENCIA.fullmatch(str(data.get(k, ""))) for k in ("recibo_ref", "firma_ref", "paso_ref")) \
            or type(data.get("secuencia")) is not int or data["secuencia"] < 1 \
            or not INSTANTE_UTC.fullmatch(str(data.get("registrada_en", ""))) \
            or not SHA256.fullmatch(str(data.get("material_root_sha256", ""))) \
            or not SHA256.fullmatch(str(enviado_sha256)) or not SHA256.fullmatch(str(solicitud["original_sha256"])) \
            or not SHA256.fullmatch(str(solicitud["entrada_sha256"])) \
            or not isinstance(c, dict) or set(c) != {"expediente_ref", "documento_ref", "version", "huella_sha256"} \
            or any(not re.fullmatch(r"ref:[0-9a-f]{64}", str(c.get(k, ""))) for k in ("expediente_ref", "documento_ref")) \
            or type(c.get("version")) is not int or c["version"] < 1 or c.get("huella_sha256") != enviado_sha256 \
            or not isinstance(v, dict) or set(v) != {"estado", "motivo", "politica", "revocacion", "sello_tiempo", "original_sha256", "firmado_sha256"} \
            or v.get("estado") != "valida" or v.get("motivo") != "verificada" or v.get("revocacion") != "vigente" \
            or v.get("sello_tiempo") not in ("no_presente", "valido", "no_comprobado") \
            or not isinstance(v.get("politica"), str) or not 1 <= len(v["politica"]) <= 256 \
            or v.get("original_sha256") != solicitud["original_sha256"] or v.get("firmado_sha256") != enviado_sha256 \
            or not isinstance(r, dict) or set(r) != {"orden_firma", "entrada_sha256", "revision_sha256", "evidencia_sha256"} \
            or r.get("orden_firma") != solicitud["paso_orden"] or r.get("entrada_sha256") != solicitud["entrada_sha256"] \
            or r.get("revision_sha256") != enviado_sha256 or not SHA256.fullmatch(str(r.get("evidencia_sha256", ""))):
        raise Corte("verificacion", "el recibo no acredita firma V2, revisión y custodia de los bytes enviados")
    return proyectar_recibo_v2(data)


def recibos_guardados_v2(previo, cantidad):
    firmas = previo.get("firmas_v2")
    propuesta = previo.get("propuesta")
    version = propuesta.get("version_actual") if isinstance(propuesta,dict) else None
    if not isinstance(firmas,list) or len(firmas) != cantidad or type(version) is not int or version < 1:
        raise Corte("contrato_http","el informe anterior no contiene los recibos V2 esperados")
    validados = []
    for orden,rec in enumerate(firmas,1):
        if not isinstance(rec,dict):
            raise Corte("contrato_http","el recibo guardado no es un objeto V2")
        v = rec.get("verificacion_tecnica")
        c = rec.get("documento_custodiado")
        if not isinstance(v,dict) or not isinstance(c,dict):
            raise Corte("contrato_http","el recibo guardado no contiene verificación y custodia")
        original = validados[0]["verificacion_tecnica"]["original_sha256"] if validados else v.get("original_sha256")
        entrada = validados[-1]["documento_custodiado"]["huella_sha256"] if validados else original
        solicitud = {"expediente_ref":previo.get("expediente_ref"),"version_expediente":version,
                     "documento":previo.get("documento"),"paso_orden":orden,
                     "original_sha256":original,"entrada_sha256":entrada}
        validado = recibo_v2(rec,solicitud,c.get("huella_sha256"),200 if rec.get("ya_registrada") is True else 201)
        if validados and (validado["secuencia"] != validados[-1]["secuencia"]+1
                or validado["recibo_ref"] == validados[-1]["recibo_ref"]
                or validado["firma_ref"] == validados[-1]["firma_ref"]):
            raise Corte("cadena","los recibos guardados no forman la secuencia V2")
        validados.append(validado)
    return validados


def validar_continuacion(previo, informe, canal):
    firmas = previo.get("firmas_v2")
    if previo.get("registro_incierto") is not False or previo.get("estado") != "PRIMERA_FIRMA_CONFIRMADA" \
            or not isinstance(firmas, list) or len(firmas) != 1 \
            or firmas[0].get("paso_orden") != 1 or firmas[0].get("firma_eficaz") is not False \
            or not SHA256.fullmatch(str(previo.get("canal_certificado_sha256", ""))) \
            or previo["canal_certificado_sha256"] == canal \
            or any(previo.get(k) != informe.get(k) for k in ("expediente_ref", "documento", "binario_sha256", "propuesta", "pdf")):
        raise Corte("continuacion", "se requiere el primer recibo confirmado y otro certificado de canal, sin operación incierta")
    return recibos_guardados_v2(previo,1)


def comparar_estado_v2(estado, firmas, documento):
    documentos = estado.get("documentos")
    if estado.get("firma_eficaz") is not False or not isinstance(documentos, list):
        raise Corte("recuperacion", "la consulta no acredita el estado de firmas")
    docs = [d for d in documentos if isinstance(d, dict) and d.get("documento") == documento]
    if len(docs) != 1:
        raise Corte("recuperacion", "el documento firmado no figura una sola vez")
    for recibo in firmas:
        pasos = [p for p in docs[0].get("pasos", []) if p.get("recibo_ref") == recibo["recibo_ref"]]
        if len(pasos) != 1 or pasos[0].get("estado") != "firmado" \
                or pasos[0].get("orden") != recibo["paso_orden"] \
                or pasos[0].get("registrada_en") != recibo["registrada_en"] \
                or pasos[0].get("documento_custodiado") != recibo["documento_custodiado"]:
            raise Corte("recuperacion", "la consulta cambió recibo, fecha, orden o custodia")
    if docs[0].get("paso_pendiente") != (2 if len(firmas) == 1 else 0):
        raise Corte("recuperacion", "el paso pendiente no coincide con los recibos confirmados")


def comprobar_reinicio(acta, informe):
    if acta.get("expediente_ref") != informe["expediente_ref"] \
            or acta.get("aplicacion_reiniciada") is not True or acta.get("postgresql_reiniciado") is not True \
            or not INSTANTE_UTC.fullmatch(str(acta.get("instante_utc", ""))):
        raise Corte("recuperacion", "falta el acta del reinicio externo del clon")
    return {"instante_utc": acta["instante_utc"], "origen_evidencia": "declaracion_externa"}


def consultar_estado_v2(page, a):
    respuesta = page.evaluate("""async ([ref, ruta]) => {
      const r = await fetch(ruta, {method:'POST', headers:{'Content-Type':'application/json',Accept:'application/json'},
        body:JSON.stringify({expediente_ref:ref}), credentials:'same-origin',cache:'no-store',redirect:'error',referrerPolicy:'no-referrer'});
      let data=null; const texto=await r.text(); try {if(texto.length<=131072)data=JSON.parse(texto);}catch{}
      return {status:r.status,data};
    }""", [a.expediente_ref, RUTA_CONSULTA_FIRMAS])
    if respuesta["status"] != 200:
        raise Corte("autoridad_pendiente", f"consulta de firmas no disponible: HTTP {respuesta['status']}")
    return datos_respuesta_json(respuesta["data"])


def solicitud_consulta_tecnica_v2(informe, firmas):
    intento = informe.get("intento_v2")
    campos = {"expediente_ref","version_expediente","documento","paso_orden","original_ref",
              "original_version","clave_idempotencia","firmado_sha256"}
    preflight = informe.get("preflight_v2")
    if not firmas or not isinstance(intento,dict) or set(intento) != campos or not isinstance(preflight,dict):
        raise Corte("consulta_v2_pendiente","falta el intento real conservado o su catálogo de firma")
    ultimo = firmas[-1]
    if any(intento.get(k) != ultimo[k] for k in ("expediente_ref","version_expediente","documento","paso_orden")) \
            or intento["firmado_sha256"] != ultimo["documento_custodiado"]["huella_sha256"] \
            or intento["original_ref"] != preflight.get("original_ref") \
            or intento["original_version"] != preflight.get("original_version") \
            or not REFERENCIA.fullmatch(str(intento["original_ref"])) \
            or type(intento["original_version"]) is not int or intento["original_version"] < 1 \
            or not isinstance(intento["clave_idempotencia"],str) \
            or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{15,63}",intento["clave_idempotencia"]) \
            or not SHA256.fullmatch(str(preflight.get("catalogo_huella",""))):
        raise Corte("consulta_v2_pendiente","el intento guardado no corresponde a los recibos de firma")
    return {k:intento[k] for k in ("expediente_ref","version_expediente","documento","paso_orden","clave_idempotencia")} | {
        "catalogo_huella":preflight["catalogo_huella"],"via":"certificado_vec"}


def validar_consulta_tecnica_v2(datos, informe, firmas):
    """Contrato técnico cerrado; no reconstruye identidad, canon ni eficacia."""
    campos = {"esquema","expediente_ref","version_expediente","documento","historia_revision","historia_sha256",
              "firmas","recuperacion","campos_no_disponibles","firma_eficaz"}
    campos_firma = {"firma_ref","recibo_ref","registrada_en","secuencia","paso_orden","paso_ref","version_expediente",
                    "via","resultado","catalogo_ref","catalogo_huella","original","documento_custodiado","revision_pdf"}
    campos_revision = {"orden_firma","firma_anterior_ref","recibo_anterior_ref","entrada_documento","entrada_longitud",
                       "byte_range","revision_sha256","contenido_firmado_sha256","revision_longitud","evidencia_firmas_sha256"}
    exacto = lambda d,c: isinstance(d,dict) and set(d)==c
    entero = lambda n: type(n) is int and 0<n<=9007199254740991
    ref = lambda r: isinstance(r,str) and bool(REFERENCIA.fullmatch(r))
    sha = lambda h: isinstance(h,str) and bool(SHA256.fullmatch(h))
    solicitud = solicitud_consulta_tecnica_v2(informe,firmas)
    if not exacto(datos,campos) or datos["esquema"] != "vec.contratacion-temporal.consulta-firmas-r5.v2" \
            or any(datos.get(k)!=solicitud[k] for k in ("expediente_ref","version_expediente","documento")) \
            or datos["firma_eficaz"] is not False or datos["recuperacion"] != "parcial" \
            or datos["campos_no_disponibles"] != ["material_root_sha256","canon_nominal"] \
            or type(datos["historia_revision"]) is not int or not 0<=datos["historia_revision"]<=9007199254740991 \
            or not sha(datos["historia_sha256"]) or not isinstance(datos["firmas"],list) or len(datos["firmas"])>128:
        raise Corte("contrato_http","la consulta técnica no cumple el contrato V2 o sus límites")
    por_firma = {}; por_recibo = set()
    for f in datos["firmas"]:
        if not exacto(f,campos_firma) or any(not ref(f[k]) for k in ("firma_ref","recibo_ref","paso_ref","catalogo_ref")) \
                or f["firma_ref"] in por_firma or f["recibo_ref"] in por_recibo \
                or not INSTANTE_UTC.fullmatch(str(f["registrada_en"])) or not entero(f["secuencia"]) \
                or not entero(f["paso_orden"]) or f["paso_orden"]>16 or not entero(f["version_expediente"]) \
                or f["version_expediente"]>solicitud["version_expediente"] or not sha(f["catalogo_huella"]) \
                or f["via"] not in ("","certificado_vec","portafirmas_registro_rrhh") or f["resultado"] not in ("firmado","devuelto"):
            raise Corte("contrato_http","metadatos de firma V2 no válidos o duplicados")
        original=f["original"];custodia=f["documento_custodiado"];revision=f["revision_pdf"]
        if not exacto(original,{"documento_ref","version","huella_sha256"}) \
                or (original["documento_ref"]!="" and not ref(original["documento_ref"])) \
                or type(original["version"]) is not int or not 0<=original["version"]<=9007199254740991 \
                or (original["huella_sha256"]!="" and not sha(original["huella_sha256"])) \
                or (custodia is not None and (not exacto(custodia,CAMPOS_RECIBO_ANIDADOS["documento_custodiado"])
                    or not ref(custodia["expediente_ref"]) or not ref(custodia["documento_ref"])
                    or not entero(custodia["version"]) or not sha(custodia["huella_sha256"]))):
            raise Corte("contrato_http","el original o la custodia no cumplen la proyección técnica")
        if revision is not None:
            entrada=revision.get("entrada_documento") if isinstance(revision,dict) else None
            br=revision.get("byte_range") if isinstance(revision,dict) else None
            if not exacto(revision,campos_revision) or not exacto(entrada,{"documento_ref","version","huella_sha256"}) \
                    or not ref(entrada["documento_ref"]) or not entero(entrada["version"]) or not sha(entrada["huella_sha256"]) \
                    or revision["orden_firma"] not in (1,2) or type(revision["orden_firma"]) is not int \
                    or not entero(revision["entrada_longitud"]) or not entero(revision["revision_longitud"]) \
                    or not revision["entrada_longitud"]<revision["revision_longitud"]<=1024*1024 \
                    or not isinstance(br,list) or len(br)!=4 or any(type(n) is not int or n<0 for n in br) \
                    or br[0]!=0 or br[1]<revision["entrada_longitud"] or br[2]<=br[1] \
                    or br[2]>revision["revision_longitud"] or br[3]<=0 or br[2]+br[3]!=revision["revision_longitud"] \
                    or any(not sha(revision[k]) for k in ("revision_sha256","contenido_firmado_sha256","evidencia_firmas_sha256")) \
                    or any(revision[k]!="" and not ref(revision[k]) for k in ("firma_anterior_ref","recibo_anterior_ref")):
                raise Corte("contrato_http","la revisión PDF no cumple el contrato técnico V2")
        por_firma[f["firma_ref"]]=f;por_recibo.add(f["recibo_ref"])
    for indice,recibo in enumerate(firmas):
        f=por_firma.get(recibo["firma_ref"]);rev=f.get("revision_pdf") if f else None
        if not f or not rev:
            raise Corte("consulta_v2_pendiente","la consulta no devuelve la revisión persistida de cada recibo")
        if any(f[k]!=recibo[k] for k in ("firma_ref","recibo_ref","registrada_en","secuencia","paso_orden","paso_ref","version_expediente")) \
                or f["documento_custodiado"]!=recibo["documento_custodiado"] or f["via"]!="certificado_vec" or f["resultado"]!="firmado" \
                or f["catalogo_huella"]!=solicitud["catalogo_huella"] or f["catalogo_ref"]!=informe["preflight_v2"]["catalogo_ref"] \
                or f["original"]!={"documento_ref":informe["intento_v2"]["original_ref"],"version":informe["intento_v2"]["original_version"],
                                    "huella_sha256":recibo["verificacion_tecnica"]["original_sha256"]} \
                or rev["orden_firma"]!=recibo["paso_orden"] or rev["entrada_documento"]["huella_sha256"]!=recibo["revision_pdf"]["entrada_sha256"] \
                or rev["revision_sha256"]!=recibo["revision_pdf"]["revision_sha256"] \
                or rev["evidencia_firmas_sha256"]!=recibo["revision_pdf"]["evidencia_sha256"]:
            raise Corte("recuperacion","la consulta técnica cambió referencias, recibo, fecha, custodia o revisión")
        previo=firmas[indice-1] if indice else None
        esperado=f["original"] if previo is None else {k:previo["documento_custodiado"][k] for k in ("documento_ref","version","huella_sha256")}
        if rev["entrada_documento"]!=esperado or rev["firma_anterior_ref"]!=(previo["firma_ref"] if previo else "") \
                or rev["recibo_anterior_ref"]!=(previo["recibo_ref"] if previo else ""):
            raise Corte("recuperacion","la revisión PDF no continúa el antecedente confirmado")
    return json.loads(json.dumps(datos))


def consultar_json_firmas_v2(page, ruta, solicitud, limite):
    respuesta=page.evaluate(r"""async ([ruta, solicitud, limite]) => {
      const control=new AbortController();
      const plazo=setTimeout(()=>control.abort(),30000);
      try {
      const r=await fetch(ruta,{method:'POST',headers:{'Content-Type':'application/json',Accept:'application/json'},
        body:JSON.stringify(solicitud),mode:'same-origin',credentials:'same-origin',cache:'no-store',redirect:'error',
        referrerPolicy:'no-referrer',signal:control.signal});
      if(r.status!==200)return {status:r.status,data:null};
      if(!/^application\/json(?:;\s*charset=utf-8)?$/i.test(r.headers.get('Content-Type')||''))return {status:200,data:null};
      const lector=r.body?.getReader();if(!lector)return {status:200,data:null};
      const partes=[];let total=0;
      try {for(;;){const {done,value}=await lector.read();if(done)break;total+=value.byteLength;
        if(total>limite)return {status:200,data:null};partes.push(value);}}
      finally {void lector.cancel().catch(()=>{});}
      const bytes=new Uint8Array(total);let offset=0;for(const parte of partes){bytes.set(parte,offset);offset+=parte.byteLength;}
      try {return {status:200,data:JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes))};}
      catch{return {status:200,data:null};}
      } catch {return {status:0,data:null};}
      finally {clearTimeout(plazo);control.abort();}
    }""",[ruta,solicitud,limite])
    if respuesta["status"]!=200:
        raise Corte("consulta_v2_pendiente",f"consulta técnica V2 no disponible o denegada: HTTP {respuesta['status']}")
    envoltorio=respuesta.get("data")
    if not isinstance(envoltorio,dict) or set(envoltorio)!={"data"}:
        raise Corte("contrato_http","la consulta técnica V2 no devuelve datos acotados")
    return envoltorio["data"]


def consultar_metadatos_v2(page,informe,firmas):
    solicitud=solicitud_consulta_tecnica_v2(informe,firmas)
    datos=consultar_json_firmas_v2(page,RUTA_CONSULTA_TECNICA_V2,solicitud,262144)
    return validar_consulta_tecnica_v2(datos,informe,firmas)


def consultar_recuperacion_nominal_v2(page,informe,firmas):
    solicitud=solicitud_consulta_tecnica_v2(informe,firmas)
    datos=consultar_json_firmas_v2(page,recuperacion_nominal.RUTA,solicitud,recuperacion_nominal.MAX_RESPUESTA)
    return recuperacion_nominal.validar(datos,informe,firmas,validar_consulta_tecnica_v2,Corte)


def descargar_revision_v2(page, a, recibo):
    c = recibo["documento_custodiado"]
    selector = f'[data-ct-fase-firma-documento="{a.documento}"] [data-ct-descargar-firmado]'
    boton = page.locator(selector)
    esperados = {"data-ct-firmado-expediente":c["expediente_ref"], "data-ct-firmado-documento":c["documento_ref"],
                 "data-ct-firmado-version":str(c["version"]), "data-ct-firmado-huella":c["huella_sha256"]}
    try:
        # La consulta que repinta el botón puede terminar después del recibo POST.
        page.wait_for_function("""([selector, esperados]) => {
          const b=document.querySelector(selector);
          return b && !b.disabled && Object.entries(esperados).every(([k,v])=>b.getAttribute(k)===v);
        }""", arg=[selector,esperados],timeout=20_000)
    except Exception:
        raise Corte("custodia_pendiente", "no está disponible la descarga del PDF custodiado del recibo") from None
    if any(boton.get_attribute(k) != v for k,v in esperados.items()) or not boton.is_enabled():
        raise Corte("custodia", "el botón de descarga no corresponde al recibo firmado")
    with page.expect_download(timeout=30_000) as espera:
        boton.click()
    contenido = Path(espera.value.path()).read_bytes()
    if not contenido.startswith(b"%PDF-") or len(contenido) > 1024 * 1024 \
            or hashlib.sha256(contenido).hexdigest() != c["huella_sha256"]:
        raise Corte("custodia", "el PDF descargado no conserva la huella del recibo")
    return {"bytes":len(contenido), "sha256":c["huella_sha256"], "documento_custodiado":c}


def recorrer_firmas_v2(page, a, informe, catalogo, timeout_error):
    if a.comparar:
        previo = leer_informe_privado(a.comparar)
        firmas = previo.get("firmas_v2")
        if previo.get("registro_incierto") is not False or previo.get("estado") != "COMPLETO" \
                or not isinstance(firmas, list) or len(firmas) != 2 \
                or any(previo.get(k) != informe.get(k) for k in ("expediente_ref","documento","binario_sha256","propuesta","pdf")):
            raise Corte("recuperacion", "se requieren los dos recibos V2 confirmados del mismo clon")
        firmas = recibos_guardados_v2(previo,2)
        if a.recuperacion_nominal is True and not previo.get("recuperacion_nominal_v2"):
            raise Corte("recuperacion_nominal", "baseline_recuperacion_nominal_ausente_o_distinta")
        baseline = previo.get("consulta_tecnica_v2")
        validar_consulta_tecnica_v2(baseline,previo,firmas)
        informe["reinicio"] = comprobar_reinicio(leer_informe_privado(a.reinicio), informe)
        comparar_estado_v2(consultar_estado_v2(page,a),firmas,a.documento)
        pdf = descargar_revision_v2(page,a,firmas[-1])
        if pdf != previo.get("pdf_firmado"):
            raise Corte("recuperacion", "el PDF custodiado cambió después del reinicio")
        tecnico = consultar_metadatos_v2(page,previo,firmas)
        if tecnico != baseline:
            raise Corte("recuperacion","los metadatos V2 o la historia cambiaron después del reinicio")
        informe.update(firmas_v2=firmas,pdf_firmado=pdf,consulta_tecnica_v2=tecnico,estado="RECUPERACION_PARCIAL",
                       pendiente="material_root_sha256_y_canon_nominal_no_expuestos", e2e=False,
                       campos_no_revalidados_por_consulta=["material_root_sha256","canon_nominal"],
                       verificacion_criptografica_repetida=False)
        if a.recuperacion_nominal is True:
            nominal = consultar_recuperacion_nominal_v2(page,previo,firmas)
            recuperacion_nominal.comparar(previo.get("recuperacion_nominal_v2"),nominal,Corte)
            informe.update(recuperacion_nominal_v2=nominal,estado="RECUPERACION_NOMINAL_CONFIRMADA",
                           campos_no_revalidados_por_consulta=[],
                           pendiente="recorrido_real_y_auditoria_descarga_pendientes")
        return
    if not a.firmar:
        raise Corte("autoridad_pendiente", "pendiente firma nominal V2; no se abrió AutoFirma")
    canal = huella_certificado_canal(a.certificado)
    informe["canal_certificado_sha256"] = canal
    previo = leer_informe_privado(a.continuar) if a.continuar else None
    anteriores = validar_continuacion(previo,informe,canal) if previo else []
    informe["firmas_v2"] = anteriores[:]
    nominal_antecedente = None
    if anteriores:
        if a.recuperacion_nominal is True and not previo.get("recuperacion_nominal_v2"):
            raise Corte("recuperacion_nominal", "baseline_recuperacion_nominal_ausente_o_distinta")
        comparar_estado_v2(consultar_estado_v2(page,a),anteriores,a.documento)
        informe["pdf_primera_revision"] = descargar_revision_v2(page,a,anteriores[0])
        baseline = previo.get("consulta_tecnica_v2")
        validar_consulta_tecnica_v2(baseline,previo,anteriores)
        if consultar_metadatos_v2(page,previo,anteriores) != baseline:
            raise Corte("recuperacion","los metadatos de la primera firma cambiaron antes del segundo paso")
        if a.recuperacion_nominal is True:
            nominal = consultar_recuperacion_nominal_v2(page,previo,anteriores)
            recuperacion_nominal.comparar(previo.get("recuperacion_nominal_v2"),nominal,Corte)
            nominal_antecedente = nominal
    detalles = page.locator("[data-ct-firma-detalles]")
    detalles.wait_for(timeout=20_000)
    detalles.evaluate("e => e.open = true")
    bloque = page.locator(f'[data-ct-circuito-documento="{a.documento}"]')
    comprobar = bloque.locator('[data-ct-firma-accion="comprobar"]')
    if comprobar.count() != 1 or not comprobar.is_enabled():
        raise Corte("autoridad_pendiente", "el montaje no ofrece el preflight nominal V2")
    try:
        with page.expect_response(lambda r:urlsplit(r.url).path==RUTA_PREFLIGHT and r.request.method=="POST",timeout=30_000) as espera:
            comprobar.click()
    except timeout_error:
        raise Corte("autoridad_pendiente", "el original o el preflight nominal no están disponibles") from None
    respuesta = espera.value
    if respuesta.status != 200:
        raise Corte("autoridad_pendiente",f"preflight nominal: HTTP {respuesta.status}")
    solicitud_preflight = json.loads(respuesta.request.post_data)
    solicitud = {"version":informe["propuesta"]["version_actual"],"documento":a.documento,
                 "original_ref":solicitud_preflight.get("original_ref"),"original_version":solicitud_preflight.get("original_version"),
                 "catalogo_ref":catalogo["catalogo_ref"],"catalogo_huella":catalogo["huella_sha256"]}
    preflight = preflight_v2(datos_respuesta(respuesta),solicitud)
    if preflight["paso_pendiente"] != a.paso or "certificado_vec" not in preflight["vias_disponibles"]:
        raise Corte("autoridad_pendiente", "el servidor no autoriza la vía certificado para el paso solicitado")
    if anteriores and any(preflight.get(k) != anteriores[0]["documento_custodiado"].get(v) for k,v in (
            ("entrada_documento_sha256","huella_sha256"),("entrada_documento_ref","documento_ref"),
            ("entrada_documento_version","version"))):
        raise Corte("cadena", "el segundo paso no parte del PDF custodiado de la primera firma")
    informe["preflight_v2"] = preflight
    intento = {}
    def registrar_intento(request):
        if urlsplit(request.url).path != RUTA_REGISTRO_V2 or request.method != "POST":
            return
        try:
            cuerpo = json.loads(request.post_data)
            campos = {"expediente_ref","version_expediente","documento","paso_orden","original_ref",
                      "original_version","firmado_base64","clave_idempotencia"}
            if not isinstance(cuerpo,dict) or set(cuerpo) != campos \
                    or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{15,63}",str(cuerpo.get("clave_idempotencia",""))) \
                    or any(cuerpo.get(k) != v for k,v in {
                        "expediente_ref":a.expediente_ref,"version_expediente":solicitud["version"],
                        "documento":a.documento,"paso_orden":a.paso,"original_ref":preflight["original_ref"],
                        "original_version":preflight["original_version"]}.items()):
                raise ValueError
            bytes_pdf = base64.b64decode(cuerpo["firmado_base64"], validate=True)
            intento.update({k:cuerpo[k] for k in campos if k != "firmado_base64"})
            intento["firmado_sha256"] = hashlib.sha256(bytes_pdf).hexdigest()
            informe["registro_incierto"] = True
            informe["intento_v2"] = intento.copy()
            a.guardar_progreso(informe)
        except Exception:
            informe["registro_incierto"] = True
            a.guardar_progreso(informe)
            raise Corte("registro_incierto", "no se pudo conservar el intento V2 antes de confirmar") from None
    page.on("request",registrar_intento)
    try:
        boton = bloque.locator('[data-ct-firma-accion="certificado_vec"]')
        try:
            boton.wait_for(timeout=20_000)
        except timeout_error:
            raise Corte("autoridad_pendiente", "la identidad no dispone de firma nominal V2") from None
        # Registrar la incertidumbre antes de que una pulsación pueda producir un efecto.
        informe["registro_incierto"] = True
        a.guardar_progreso(informe)
        with page.expect_response(lambda r:urlsplit(r.url).path==RUTA_REGISTRO_V2 and r.request.method=="POST",timeout=180_000) as espera:
            boton.click()
        respuesta=espera.value
        if respuesta.status not in (200,201):
            raise Corte("registro_incierto",f"registro V2 sin recibo confirmado: HTTP {respuesta.status}")
        if not intento or any(intento.get(k) != v for k,v in {
                "expediente_ref":a.expediente_ref,"version_expediente":solicitud["version"],
                "documento":a.documento,"paso_orden":a.paso,"original_ref":preflight["original_ref"],
                "original_version":preflight["original_version"]}.items()):
            raise Corte("registro_incierto","el intento enviado no corresponde al preflight")
        solicitud_recibo={"expediente_ref":a.expediente_ref,"version_expediente":solicitud["version"],
                          "documento":a.documento,"paso_orden":a.paso,
                          "original_sha256":anteriores[0]["verificacion_tecnica"]["original_sha256"] if anteriores else preflight["entrada_documento_sha256"],
                          "entrada_sha256":preflight["entrada_documento_sha256"]}
        recibo=recibo_v2(datos_respuesta(respuesta),solicitud_recibo,intento["firmado_sha256"],respuesta.status)
        if anteriores and (recibo["secuencia"] != anteriores[0]["secuencia"]+1
                or recibo["recibo_ref"] == anteriores[0]["recibo_ref"]
                or recibo["firma_ref"] == anteriores[0]["firma_ref"]):
            raise Corte("cadena","el segundo recibo no continúa la primera firma")
        informe["firmas_v2"].append(recibo)
        informe["registro_incierto"]=False
        a.guardar_progreso(informe)
    except timeout_error:
        raise Corte("registro_incierto" if informe["registro_incierto"] else "firma_admitida",
                    "AutoFirma no devolvió un recibo V2; no se repetirá la operación") from None
    finally:
        page.remove_listener("request",registrar_intento)
    comparar_estado_v2(consultar_estado_v2(page,a),informe["firmas_v2"],a.documento)
    # La interfaz vuelve a consultar y repinta la descarga tras el registro real.
    informe["pdf_firmado"]=descargar_revision_v2(page,a,recibo)
    informe["consulta_tecnica_v2"]=consultar_metadatos_v2(page,informe,informe["firmas_v2"])
    if a.recuperacion_nominal is True:
        nominal=consultar_recuperacion_nominal_v2(page,informe,informe["firmas_v2"])
        if nominal_antecedente is not None:
            referencias={f["firma_ref"] for f in nominal_antecedente}
            recuperacion_nominal.comparar(nominal_antecedente,
                [f for f in nominal if f["firma_ref"] in referencias],Corte)
        informe["recuperacion_nominal_v2"]=nominal
    if informe["consulta_tecnica_v2"]["firmas"]:
        actual = next(f for f in informe["consulta_tecnica_v2"]["firmas"] if f["firma_ref"]==recibo["firma_ref"])
        if actual["revision_pdf"]["revision_longitud"]!=informe["pdf_firmado"]["bytes"]:
            raise Corte("recuperacion","la longitud de la revisión V2 no coincide con el PDF descargado")
    informe["estado"]="PRIMERA_FIRMA_CONFIRMADA" if a.paso==1 else "COMPLETO"
    informe["firma_eficaz"]=False
    informe["e2e"]=False
    informe["pendiente"]="recuperacion_tras_reinicio_y_canon_nominal_no_expuesto"


def recorrer(a, chrome, entorno):
    comprobar_sdk_playwright()
    try:
        from playwright.sync_api import sync_playwright, TimeoutError as PlaywrightTimeout
    except ImportError:
        raise Corte("precondiciones", "Playwright de Python no está instalado") from None
    informe = {"estado": "CORTE", "entorno": "clon_local_h3_h5", "binario_sha256": entorno["binario_sha256"],
               "expediente_ref": a.expediente_ref, "documento": a.documento, "pdf": {}, "firma": None,
               "firmas_v2": [], "http": [], "registro_incierto": False}
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
        rutas_observadas = {"/portal-empleado/", RUTA_DETALLE, RUTA_FIRMAS, RUTA_REGISTRO_V2, RUTA_ORIGINAL, RUTA_PREFLIGHT,
                            RUTA_CONSULTA_FIRMAS, RUTA_CONSULTA_TECNICA_V2, RUTA_CIRCUITO, RUTA_SEGUIMIENTO,
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
            recorrer_firmas_v2(page, a, informe, catalogo, PlaywrightTimeout)
            return informe
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
    salida_fd = None
    ultimo_progreso = None
    def conservar(valor):
        nonlocal ultimo_progreso
        ultimo_progreso = json.loads(json.dumps(valor))
        if salida_fd is not None:
            contenido = (json.dumps(valor,ensure_ascii=False,sort_keys=True)+"\n").encode("utf-8")
            os.lseek(salida_fd,0,os.SEEK_SET)
            os.ftruncate(salida_fd,0)
            with os.fdopen(os.dup(salida_fd),"wb") as f:
                f.write(contenido)
                f.flush()
                os.fsync(f.fileno())
    a.guardar_progreso = conservar
    try:
        chrome, entorno = validar_entrada(a)
        if a.salida:
            directorio = abrir_directorio_privado(a.salida)
            try:
                salida_fd = os.open(a.salida.name,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600,dir_fd=directorio)
            finally:
                os.close(directorio)
            conservar(informe)
        informe = recorrer(a, chrome, entorno)
    except Corte as e:
        informe = ultimo_progreso or informe
        informe["estado"] = "NO EJECUTADO" if e.paso == "precondiciones" else "CORTE"
        informe["corte"] = e.paso
        informe["motivo"] = e.motivo
    except Exception as e:
        informe = ultimo_progreso or informe
        informe["estado"] = "CORTE"
        informe["corte"] = "error_no_clasificado"
        informe["motivo"] = type(e).__name__
    try:
        if salida_fd is not None:
            conservar(informe)
        elif a.salida:
            guardar_privado(a.salida,(json.dumps(informe,ensure_ascii=False,sort_keys=True)+"\n").encode("utf-8"))
    except OSError:
        print("No se pudo conservar el informe privado",file=sys.stderr)
        return 2
    finally:
        if salida_fd is not None:
            os.close(salida_fd)
    # Las claves idempotentes se conservan solo en el informe privado para reconciliar.
    publico = informe_publico(informe)
    print(json.dumps(publico,ensure_ascii=False,sort_keys=True))
    return 0 if informe["estado"] in ("COMPLETO","PRIMERA_FIRMA_CONFIRMADA","RECUPERACION_NOMINAL_CONFIRMADA") else 2


# Se aplica también a quienes importan únicamente GuardiaNavegador: el módulo
# no se entrega al llamador con un SDK que carezca del evento close verificado.
try:
    comprobar_sdk_playwright()
except Corte as e:
    if __name__ != "__main__":
        raise
    print(json.dumps({"estado": "NO EJECUTADO", "corte": e.paso, "motivo": e.motivo}, ensure_ascii=False))
    sys.exit(2)

if __name__ == "__main__":
    sys.exit(main())
