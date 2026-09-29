#!/usr/bin/env python3
"""Recorrido local de RRHH: análisis, cobertura, asignación e informe jurídico.

Solo opera sobre un clon sintético H3-H5 acreditado por un manifiesto externo.
No arranca servicios, instala migraciones ni usa identidades de producción.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from pathlib import Path
from urllib.parse import urlparse


RUTAS = {
    "cuadro": "/api/vec/contratacion-temporal/cuadro/consultas",
    "detalle": "/api/vec/contratacion-temporal/expedientes/consultas",
    "analisis": "/api/vec/contratacion-temporal/analisis/registros",
    "cobertura": "/api/vec/contratacion-temporal/cobertura/decisiones",
    "asignacion": "/api/vec/contratacion-temporal/asignaciones",
    "informe": "/api/vec/contratacion-temporal/informes-juridicos/preparaciones",
}
REFERENCIA = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$")


class NoEjecutado(Exception):
    """Falta una precondición antes de abrir Chrome o de enviar un efecto."""


class Corte(Exception):
    """El recorrido se detuvo en una puerta concreta."""


def origen_local(valor: str) -> str:
    url = urlparse(valor)
    try:
        puerto = url.port
    except ValueError as error:
        raise NoEjecutado("origen: puerto inválido") from error
    if (url.scheme != "https" or url.hostname not in {"localhost", "127.0.0.1", "::1"}
            or not puerto or url.username or url.password or url.path not in {"", "/"}
            or url.query or url.fragment):
        raise NoEjecutado("origen: se exige HTTPS loopback con puerto y sin ruta, usuario ni consulta")
    return valor.rstrip("/")


def solicitud_permitida(url_solicitada: str, origen: str) -> bool:
    try:
        url = urlparse(url_solicitada)
        base = urlparse(origen)
        return (url.scheme, url.hostname, url.port) == (base.scheme, base.hostname, base.port) \
            and not url.username and not url.password
    except ValueError:
        return False


def instalar_filtro_red(contexto, origen: str, fallos: list[str]) -> None:
    """Interceta sin seguir redirecciones; ningún salto sale al otro puerto."""
    def filtrar(ruta) -> None:
        if not solicitud_permitida(ruta.request.url, origen):
            fallos.append("red: petición fuera del origen HTTPS loopback acreditado")
            ruta.abort()
            return
        try:
            respuesta = ruta.fetch(max_redirects=0, timeout=20_000)
            if not solicitud_permitida(respuesta.url, origen) or 300 <= respuesta.status < 400:
                fallos.append("red: redirección o respuesta fuera del origen acreditado")
                ruta.abort()
                return
            if "set-cookie" in respuesta.headers:
                fallos.append("red: Set-Cookie prohibido")
                ruta.abort()
                return
            ruta.fulfill(response=respuesta)
        except Exception:
            fallos.append("red: petición local sin respuesta verificable")
            ruta.abort()

    contexto.route("**/*", filtrar)


def leer_json(ruta: Path) -> dict:
    try:
        dato = json.loads(ruta.read_text(encoding="utf-8"))
    except (OSError, ValueError) as error:
        raise NoEjecutado(f"entrada: falta JSON válido en {ruta.name}") from error
    if not isinstance(dato, dict):
        raise NoEjecutado(f"entrada: {ruta.name} debe contener un objeto JSON")
    return dato


def comprobar_precondiciones(cfg: argparse.Namespace) -> tuple[str, dict]:
    if not all((cfg.origen, cfg.expediente_ref, cfg.certificado, cfg.clave,
                cfg.certificado_denegado, cfg.clave_denegada, cfg.clon, cfg.binario)):
        raise NoEjecutado("entrada: faltan URL, referencia, identidades mTLS, clon H3-H5 o binario VEC")
    origen = origen_local(cfg.origen)
    if not REFERENCIA.fullmatch(cfg.expediente_ref):
        raise NoEjecutado("entrada: referencia opaca de expediente inválida")
    manifiesto = leer_json(cfg.clon)
    hitos = manifiesto.get("hitos")
    if (manifiesto.get("tipo") != "clon_local_h3_h5"
            or manifiesto.get("sintetico") is not True
            or manifiesto.get("origen") != origen
            or manifiesto.get("expediente_ref") != cfg.expediente_ref
            or not isinstance(hitos, list) or len(hitos) != 3
            or not all(isinstance(hito, str) for hito in hitos)
            or set(hitos) != {"H3", "H4", "H5"}
            or not re.fullmatch(r"[0-9a-f]{40}", str(manifiesto.get("commit_binario", "")))
            or not re.fullmatch(r"[0-9a-f]{64}", str(manifiesto.get("sha256_binario", "")))):
        raise NoEjecutado("clon: falta acreditación externa sintética H3-H5 para este origen, expediente y binario")
    if not cfg.binario.is_file():
        raise NoEjecutado("binario: falta el ejecutable VEC declarado para el clon")
    huella = hashlib.sha256()
    try:
        with cfg.binario.open("rb") as fichero:
            for bloque in iter(lambda: fichero.read(1024 * 1024), b""):
                huella.update(bloque)
    except OSError as error:
        raise NoEjecutado("binario: no se pudo leer para comprobar SHA256") from error
    if huella.hexdigest() != manifiesto["sha256_binario"]:
        raise NoEjecutado("binario: SHA256 distinto del manifiesto del clon")
    for fichero in (cfg.certificado, cfg.clave, cfg.certificado_denegado, cfg.clave_denegada):
        if not fichero.is_file() or fichero.stat().st_size == 0:
            raise NoEjecutado("mTLS: falta certificado o clave sintética legible")
    if cfg.modo == "registrar" and not cfg.efectos:
        raise NoEjecutado("efectos: --registrar exige --efectos tras revisar expediente y formulario")
    if cfg.modo == "registrar" and not cfg.datos:
        raise NoEjecutado("datos: falta JSON externo con análisis y vía gobernada")
    if cfg.modo == "recuperar" and not cfg.esperado:
        raise NoEjecutado("recuperación: falta resultado anterior para comparar versión e historia")
    return origen, manifiesto


def extraer_data(respuesta) -> dict:
    try:
        cuerpo = respuesta.json()
        data = cuerpo["data"]
    except (ValueError, KeyError, TypeError) as error:
        raise Corte("respuesta: JSON sin envoltorio data") from error
    if not isinstance(data, dict):
        raise Corte("respuesta: data no es un objeto")
    return data


def observar(pagina, ruta: str, accion, estados=(200,)) -> dict:
    with pagina.expect_response(
        lambda r: urlparse(r.url).path == ruta and r.request.method == "POST",
        timeout=20_000,
    ) as pendiente:
        accion()
    respuesta = pendiente.value
    if respuesta.status not in estados:
        raise Corte(f"{ruta}: HTTP {respuesta.status}; no se reenvía el efecto")
    if "set-cookie" in respuesta.headers:
        raise Corte(f"{ruta}: Set-Cookie prohibido")
    return extraer_data(respuesta)


def abrir_expediente(pagina, origen: str, expediente_ref: str) -> dict:
    respuesta = pagina.goto(origen + "/portal-empleado/#contratacion-temporal",
                            wait_until="domcontentloaded", timeout=20_000)
    if respuesta is None or respuesta.status != 200:
        raise Corte("portal: no respondió HTTP 200")
    # El cuadro puede paginar. Filtrar por la referencia opaca en el servidor
    # no está admitido; el operador debe aportar un expediente visible en la lista.
    pagina.locator("[data-modulo='contratacion-temporal']").wait_for(timeout=15_000)
    selector = f'[data-ct-exp-abrir="{expediente_ref}"]'
    if pagina.locator(selector).count() != 1:
        raise Corte("cuadro: expediente no visible en la página autorizada; no se crea otro")
    return observar(pagina, RUTAS["detalle"], lambda: pagina.locator(selector).click())


def comprobar_denegacion(navegador, cfg: argparse.Namespace, origen: str) -> None:
    contexto = navegador.new_context(
        client_certificates=[{"origin": origen, "certPath": str(cfg.certificado_denegado),
                              "keyPath": str(cfg.clave_denegada)}],
        ignore_https_errors=False, service_workers="block",
    )
    try:
        fallos: list[str] = []
        instalar_filtro_red(contexto, origen, fallos)
        pagina = contexto.new_page()
        respuesta = pagina.goto(origen + "/portal-empleado/#contratacion-temporal",
                                wait_until="domcontentloaded", timeout=20_000)
        if respuesta is None:
            raise Corte("permiso: el portal no respondió para la identidad denegada")
        if fallos:
            raise Corte(fallos[0])
        if respuesta.status in (401, 403):
            if contexto.cookies():
                raise Corte("permiso: la identidad denegada recibió cookie")
            return
        if respuesta.status != 200:
            raise Corte(f"permiso: portal HTTP {respuesta.status} impide comprobar denegación")
        estado = pagina.evaluate("""async ({ruta, referencia}) => {
          const respuesta = await fetch(ruta, {
            method: 'POST', credentials: 'omit', cache: 'no-store', redirect: 'error',
            headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
            body: JSON.stringify({expediente_ref: referencia, version_observada: 0})
          });
          return {estado: respuesta.status, cookie: respuesta.headers.has('set-cookie')};
        }""", {"ruta": RUTAS["detalle"], "referencia": cfg.expediente_ref})
        if estado["estado"] not in (401, 403, 404) or estado["cookie"] or contexto.cookies():
            raise Corte("permiso: la identidad sin concesión obtuvo detalle o cookie")
        if fallos:
            raise Corte(fallos[0])
    finally:
        contexto.close()


def comprobar_detalle(detalle: dict, referencia: str, version_minima=1) -> dict:
    resumen = detalle.get("resumen")
    hitos = detalle.get("hitos")
    if (not isinstance(resumen, dict) or resumen.get("expediente_ref") != referencia
            or not isinstance(resumen.get("version"), int)
            or resumen["version"] < version_minima or not isinstance(hitos, list)):
        raise Corte("detalle: referencia, versión o historia no acreditadas")
    return {"version": resumen["version"], "fase": resumen.get("fase_clave"),
            "estado": resumen.get("estado_clave"), "hitos": len(hitos)}


def comprobar_recibo(recibo: dict, referencia: str, version_anterior: int, etapa: str) -> dict:
    if (recibo.get("expediente_ref") != referencia
            or recibo.get("version_resultante") != version_anterior + 1
            or not isinstance(recibo.get("recibo_ref"), str)
            or not REFERENCIA.fullmatch(recibo["recibo_ref"])
            or not re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z",
                                str(recibo.get("confirmada_en") or recibo.get("registrada_en") or ""))):
        raise Corte(f"{etapa}: recibo sin vínculo, versión o fecha acreditados")
    if etapa == "cobertura" and recibo.get("estado") != "aplicada":
        raise Corte("cobertura: decisión sin estado aplicada")
    return {"etapa": etapa, "recibo_ref": recibo["recibo_ref"],
            "version": recibo["version_resultante"],
            "fecha": recibo.get("confirmada_en") or recibo.get("registrada_en")}


def controles_privacidad(contexto, pagina, errores: list[str], fallos_red: list[str]) -> None:
    estado = pagina.evaluate("""() => ({
      local: localStorage.length, sesion: sessionStorage.length,
      ancho: document.documentElement.scrollWidth - document.documentElement.clientWidth
    })""")
    if fallos_red:
        raise Corte(fallos_red[0])
    if contexto.cookies() or errores or estado["local"] or estado["sesion"] or estado["ancho"] > 0:
        raise Corte("navegador: cookies, almacenamiento, error JS o desbordamiento horizontal")


def formulario(pagina, etapa: str, ruta: str, completar) -> dict:
    selector = f"[data-ct-{etapa}-form]"
    if pagina.locator(selector).count() != 1:
        raise Corte(f"{etapa}: formulario no disponible; comprobar fase, versión y concesión")
    completar(pagina.locator(selector))
    return observar(pagina, ruta, lambda: pagina.locator(selector).locator("button[type='submit']").click(), (201,))


def completar_analisis(form, datos: dict) -> None:
    permitidos = {"modalidad_clave", "categoria_ref", "grupo_subgrupo", "causa_clave", "inicio",
                 "fin", "jornada_horas", "jornada_minutos", "entrada_rc_referencia", "observaciones"}
    if set(datos) != permitidos:
        raise Corte("análisis: JSON debe aportar exactamente los diez campos del formulario")
    for campo in ("modalidad_clave", "categoria_ref", "grupo_subgrupo", "causa_clave",
                  "entrada_rc_referencia"):
        form.locator(f"[name='{campo}']").select_option(str(datos[campo]))
    for campo in ("inicio", "fin", "jornada_horas", "jornada_minutos", "observaciones"):
        form.locator(f"[name='{campo}']").fill(str(datos[campo]))


def ejecutar(cfg: argparse.Namespace) -> dict:
    origen, manifiesto = comprobar_precondiciones(cfg)
    try:
        from playwright.sync_api import sync_playwright
    except ImportError as error:
        raise NoEjecutado("Playwright Python no está instalado") from error
    chrome = cfg.chrome or Path("/snap/bin/chromium")
    if not chrome.is_file():
        raise NoEjecutado("falta el Chrome/Chromium del sistema")
    resultado = {"estado": "EN CURSO", "expediente_ref": cfg.expediente_ref,
                 "binario_declarado": manifiesto["commit_binario"],
                 "sha256_binario": manifiesto["sha256_binario"],
                 "etapas": [], "corte": ""}
    with sync_playwright() as pw:
        navegador = pw.chromium.launch(executable_path=str(chrome), headless=True)
        try:
            comprobar_denegacion(navegador, cfg, origen)
            resultado["permiso_denegado"] = "HTTP 401/403/404 sin datos"
            for ancho, alto in ((1440, 900), (390, 844)):
                contexto = navegador.new_context(
                    client_certificates=[{"origin": origen, "certPath": str(cfg.certificado),
                                          "keyPath": str(cfg.clave)}],
                    ignore_https_errors=False, service_workers="block", locale="es-ES",
                    timezone_id="Europe/Madrid", viewport={"width": ancho, "height": alto},
                )
                try:
                    fallos_red: list[str] = []
                    instalar_filtro_red(contexto, origen, fallos_red)
                    pagina = contexto.new_page()
                    errores: list[str] = []
                    pagina.on("pageerror", lambda error: errores.append(str(error)))
                    pagina.on("dialog", lambda dialog: dialog.accept())
                    detalle = abrir_expediente(pagina, origen, cfg.expediente_ref)
                    if fallos_red:
                        raise Corte(fallos_red[0])
                    observado = comprobar_detalle(detalle, cfg.expediente_ref)
                    resultado["etapas"].append({"ancho": ancho, "lectura": observado})
                    if cfg.modo == "recuperar":
                        esperado = leer_json(cfg.esperado)
                        if (esperado.get("estado") != "RECORRIDO"
                                or esperado.get("expediente_ref") != cfg.expediente_ref
                                or esperado.get("binario_declarado") != manifiesto["commit_binario"]
                                or esperado.get("sha256_binario") != manifiesto["sha256_binario"]
                                or observado["version"] != esperado.get("version_final")
                                or observado["hitos"] != esperado.get("hitos_final")):
                            raise Corte("recuperación: versión o número de hitos distintos tras reinicio")
                        controles_privacidad(contexto, pagina, errores, fallos_red)
                        continue
                    if ancho == 390:
                        controles_privacidad(contexto, pagina, errores, fallos_red)
                        continue
                    datos = leer_json(cfg.datos)
                    if set(datos) != {"analisis", "via_cobertura", "motivo_cobertura"}:
                        raise Corte("datos: análisis, vía y motivo deben venir del catálogo gobernado")
                    for clave in ("via_cobertura", "motivo_cobertura"):
                        valor = datos[clave]
                        if (not isinstance(valor, str)
                                or (valor != "" and not re.fullmatch(r"[a-z][a-z0-9._-]{1,79}", valor))
                                or (clave == "via_cobertura" and valor == "")):
                            raise Corte(f"datos: {clave} no es una clave válida del catálogo")
                    etapas = (("analisis", "analisis", lambda f: completar_analisis(f, datos["analisis"])),
                              ("cobertura", "cobertura", lambda f: (
                                  f.locator(f"[name='via_elegida'][value='{datos['via_cobertura']}']").check(),
                                  f.locator("[name='motivo_clave']").select_option(datos["motivo_cobertura"]))),
                              ("asignacion", "asignacion", lambda f: f.locator("[name='confirmacion']").check()),
                              ("informe", "informe", lambda f: f.locator("[name='confirmacion']").check()))
                    for etapa, ruta, completar in etapas:
                        if fallos_red:
                            raise Corte(fallos_red[0])
                        if pagina.locator(f"[data-ct-{etapa}-form]").count() == 0:
                            # Fase ya realizada en el expediente existente: solo avanzar
                            # si la proyección autorizada acredita su resultado.
                            if etapa == "analisis" and detalle.get("analisis"):
                                continue
                            if etapa == "cobertura" and detalle.get("cobertura"):
                                continue
                            if etapa == "asignacion" and detalle.get("asignacion"):
                                continue
                            raise Corte(f"{etapa}: formulario ausente y sin resultado proyectado")
                        recibo = formulario(pagina, etapa, RUTAS[ruta], completar)
                        evidencia = comprobar_recibo(recibo, cfg.expediente_ref, observado["version"], etapa)
                        resultado["etapas"].append(evidencia)
                        observado["version"] = evidencia["version"]
                        # Tras la operación, el formulario puede desaparecer. La lectura
                        # posterior se hace de nuevo desde la lista, con otro POST de detalle.
                        detalle = abrir_expediente(pagina, origen, cfg.expediente_ref)
                        actualizado = comprobar_detalle(detalle, cfg.expediente_ref, observado["version"])
                        if actualizado["hitos"] != observado["hitos"] + 1:
                            raise Corte(f"{etapa}: número de hitos distinto del único efecto esperado")
                        observado = actualizado
                    if not any("recibo_ref" in etapa for etapa in resultado["etapas"]):
                        raise Corte("efectos: el expediente ya estaba tramitado; no se atribuye un recorrido nuevo")
                    resultado["version_final"] = observado["version"]
                    resultado["hitos_final"] = observado["hitos"]
                    controles_privacidad(contexto, pagina, errores, fallos_red)
                finally:
                    contexto.close()
        finally:
            navegador.close()
    resultado["estado"] = "RECORRIDO" if cfg.modo == "registrar" else "LECTURA RECUPERADA"
    resultado["corte"] = "informe jurídico de desarrollo; firma, envío y fiscalización sin acreditar"
    return resultado


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--origen")
    parser.add_argument("--expediente-ref")
    parser.add_argument("--certificado", type=Path)
    parser.add_argument("--clave", type=Path)
    parser.add_argument("--certificado-denegado", type=Path)
    parser.add_argument("--clave-denegada", type=Path)
    parser.add_argument("--clon", type=Path, help="manifiesto externo del clon sintético H3-H5")
    parser.add_argument("--binario", type=Path, help="ejecutable VEC del clon para verificar SHA256")
    parser.add_argument("--datos", type=Path, help="datos sintéticos externos para formularios")
    parser.add_argument("--esperado", type=Path, help="JSON de la primera ejecución, tras reiniciar app y PostgreSQL")
    parser.add_argument("--chrome", type=Path)
    parser.add_argument("--modo", choices=("registrar", "recuperar"), default="recuperar")
    parser.add_argument("--efectos", action="store_true", help="autoriza los POST de escritura en el clon")
    cfg = parser.parse_args()
    try:
        salida = ejecutar(cfg)
        print(json.dumps(salida, ensure_ascii=False, sort_keys=True))
        return 0
    except NoEjecutado as error:
        print(json.dumps({"estado": "NO EJECUTADO", "corte": str(error)}, ensure_ascii=False))
        return 2
    except Corte as error:
        print(json.dumps({"estado": "CORTADO", "corte": str(error)}, ensure_ascii=False))
        return 1
    except Exception as error:
        print(json.dumps({"estado": "CORTADO", "corte": f"navegador: {type(error).__name__}; no se reintenta ninguna escritura"}, ensure_ascii=False))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
