#!/usr/bin/env python3
"""Recorre el área personal en un clon local sintético ya preparado.

La ausencia de una precondición termina con NO EJECUTADO (código 2), antes de
importar Playwright, abrir Chrome o conectar con el servidor.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import sys
from pathlib import Path
from urllib.parse import urlparse


RUTAS = {
    "preferencias": "/api/vec/usuarios/area-personal/mis-preferencias",
    "correos": "/api/vec/usuarios/area-personal/mis-correos",
    "imagen": "/api/vec/usuarios/area-personal/mi-imagen",
    "bolsa": "/api/vec/bolsa/mi-bolsa",
    "historial": "/api/vec/bolsa/mi-bolsa/historial",
}
HITOS = frozenset({"H3", "H4", "H5"})
RAIZ_REPO = Path(__file__).resolve().parents[3]


class NoEjecutado(Exception):
    pass


class Corte(Exception):
    def __init__(self, paso: str, motivo: str):
        super().__init__(motivo)
        self.paso = paso
        self.motivo = motivo


def dentro_repo(ruta: Path) -> bool:
    return ruta.resolve().is_relative_to(RAIZ_REPO)


def sha256_archivo(ruta: Path) -> str:
    resumen = hashlib.sha256()
    with ruta.open("rb") as fichero:
        for bloque in iter(lambda: fichero.read(1024 * 1024), b""):
            resumen.update(bloque)
    return resumen.hexdigest()


def preparar(origen: str, acta: Path, certificado: Path, clave: Path) -> tuple[str, Path]:
    try:
        url = urlparse(origen)
        puerto = url.port
    except ValueError:
        raise NoEjecutado("origen: URL o puerto inválido") from None
    if (url.scheme != "https" or url.hostname not in {"localhost", "127.0.0.1", "::1"}
            or not puerto or url.username or url.password or url.path not in {"", "/"}
            or url.query or url.fragment):
        raise NoEjecutado("origen: se exige HTTPS loopback sin credenciales ni ruta")
    if not acta.is_file() or dentro_repo(acta):
        raise NoEjecutado("acta: falta un acta externa al repositorio")
    try:
        datos = json.loads(acta.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        raise NoEjecutado("acta: JSON ilegible") from None
    hitos = datos.get("hitos_verificados") if isinstance(datos, dict) else None
    if (not isinstance(datos, dict) or datos.get("origen") != origen.rstrip("/")
            or datos.get("clon_sintetico") is not True
            or not isinstance(datos.get("clon_ref"), str) or not datos["clon_ref"]
            or not isinstance(hitos, list) or not all(isinstance(h, str) for h in hitos)
            or set(hitos) != HITOS):
        raise NoEjecutado("acta: clon sintético H3, H4 y H5 sin acreditar")
    if not isinstance(datos.get("binario"), str) or not datos["binario"]:
        raise NoEjecutado("binario: falta la ruta externa")
    binario = Path(datos["binario"])
    huella = datos.get("binario_sha256")
    try:
        binario_valido = (binario.is_file() and not dentro_repo(binario)
                          and isinstance(huella, str) and len(huella) == 64
                          and sha256_archivo(binario) == huella.lower())
    except OSError:
        binario_valido = False
    if not binario_valido:
        raise NoEjecutado("binario: falta o no coincide con el acta")
    for nombre, ruta in (("certificado", certificado), ("clave", clave)):
        try:
            valido = ruta.is_file() and ruta.stat().st_size > 0 and not dentro_repo(ruta)
        except OSError:
            valido = False
        if not valido:
            raise NoEjecutado(f"{nombre}: falta material sintético externo a Git")
    chrome = next((Path(p) for nombre in ("google-chrome", "google-chrome-stable", "chromium")
                   if (p := shutil.which(nombre))), None)
    if chrome is None:
        raise NoEjecutado("Chrome: no está instalado el navegador del sistema")
    return origen.rstrip("/"), chrome


def dato(respuesta) -> dict:
    try:
        objeto = respuesta.json()
    except ValueError:
        raise Corte("respuesta", "JSON inválido") from None
    if not isinstance(objeto, dict) or not isinstance(objeto.get("data"), dict):
        raise Corte("respuesta", "contrato de datos inválido")
    return objeto["data"]


def esperar_respuesta(pagina, ruta: str, metodo: str, accion):
    with pagina.expect_response(
        lambda r: urlparse(r.url).path == ruta and r.request.method == metodo,
        timeout=15000,
    ) as espera:
        accion()
    respuesta = espera.value
    estados = {200, 201} if metodo != "GET" else {200}
    if respuesta.status not in estados:
        raise Corte(ruta, f"{metodo} respondió {respuesta.status}")
    return dato(respuesta)


def huella_recibo(datos: dict, paso: str) -> dict:
    ref = datos.get("recibo_ref")
    version = datos.get("version")
    if not isinstance(ref, str) or not ref or not isinstance(version, int) or version < 1:
        raise Corte(paso, "respuesta sin recibo o versión válidos")
    return {"recibo_sha256": hashlib.sha256(ref.encode()).hexdigest(), "version": version}


def mismo_origen(url: str, origen: str) -> bool:
    try:
        destino = urlparse(url)
        esperado = urlparse(origen)
        return (destino.scheme, destino.hostname, destino.port) == (
            esperado.scheme, esperado.hostname, esperado.port)
    except ValueError:
        return False


def filtrar_red(ruta, origen: str, incidencias: list[str]) -> None:
    if not mismo_origen(ruta.request.url, origen):
        incidencias.append("origen_bloqueado")
        ruta.abort()
        return
    try:
        respuesta = ruta.fetch(max_redirects=0)
        if 300 <= respuesta.status < 400 or not mismo_origen(respuesta.url, origen):
            incidencias.append("redireccion_bloqueada")
            ruta.abort()
            return
        ruta.fulfill(response=respuesta)
    except Exception:
        incidencias.append("red_no_disponible")
        ruta.abort()


def comprobar_pagina(pagina, contexto, errores_js: list[str], respuestas_externas: list[str], cookies_set: list[str]) -> None:
    estado = pagina.evaluate("""async () => ({
      ancho: document.documentElement.clientWidth,
      contenido: document.documentElement.scrollWidth,
      local: localStorage.length,
      sesion: sessionStorage.length,
      indexed: indexedDB.databases ? await indexedDB.databases() : []
    })""")
    if (estado["contenido"] > estado["ancho"] or estado["local"] or estado["sesion"]
            or estado["indexed"] or contexto.cookies() or cookies_set or errores_js or respuestas_externas):
        raise Corte("controles", "desbordamiento, almacenamiento, cookies, JS o red externa")


def ejecutar(origen: str, chrome: Path, certificado: Path, clave: Path) -> dict:
    from playwright.sync_api import sync_playwright

    resultado = {"estado": "CORTE", "pasos": [], "primer_corte": None}
    with sync_playwright() as playwright:
        navegador = playwright.chromium.launch(executable_path=str(chrome), headless=True)
        try:
            contexto = navegador.new_context(
                client_certificates=[{"origin": origen, "certPath": str(certificado), "keyPath": str(clave)}],
                ignore_https_errors=False, service_workers="block", locale="es-ES",
                timezone_id="Europe/Madrid", viewport={"width": 1440, "height": 900},
            )
            try:
                errores_js = []
                respuestas_externas = []
                cookies_set = []
                contexto.route("**/*", lambda ruta: filtrar_red(ruta, origen, respuestas_externas))
                pagina = contexto.new_page()
                pagina.on("pageerror", lambda error: errores_js.append(type(error).__name__))
                pagina.on("request", lambda peticion: respuestas_externas.append("externa")
                          if not mismo_origen(peticion.url, origen) else None)
                pagina.on("response", lambda respuesta: cookies_set.append("set-cookie")
                          if "set-cookie" in respuesta.headers else None)
                paso_actual = "preferencias"
                try:
                    # 1. Preferencias: un valor diferente, recibo y recuperación por GET.
                    esperar_respuesta(pagina, RUTAS["preferencias"], "GET",
                                     lambda: pagina.goto(origen + "/area-personal/?vista=preferencias", wait_until="domcontentloaded"))
                    pagina.locator("#formulario-preferencias").wait_for()
                    opciones = pagina.locator("#preferencia-filas option").evaluate_all(
                        "els => els.map(el => el.value)")
                    elegido = pagina.locator("#preferencia-filas").input_value()
                    nuevo = next((valor for valor in opciones if valor != elegido), None)
                    if nuevo is None:
                        raise Corte("preferencias", "catálogo sin una alternativa de filas")
                    pagina.locator("#preferencia-idioma").select_option("es")
                    pagina.locator("#preferencia-filas").select_option(nuevo)
                    recibo = esperar_respuesta(pagina, RUTAS["preferencias"], "PUT",
                                               lambda: pagina.locator("#formulario-preferencias button[type=submit]").click())
                    evidencia = huella_recibo(recibo, "preferencias")
                    pagina.locator(".preferencias-exito").wait_for()
                    recuperado = esperar_respuesta(pagina, RUTAS["preferencias"], "GET", pagina.reload)
                    if (recuperado.get("estado", {}).get("valores", {}).get("filas") != int(nuevo)
                            or recuperado.get("estado", {}).get("valores", {}).get("idioma") != "es"
                            or recuperado.get("estado", {}).get("version") != evidencia["version"]):
                        raise Corte("preferencias", "GET no recupera el valor guardado")
                    resultado["pasos"].append({"paso": "preferencias", "estado": "RECUPERADO", **evidencia})

                    # 2. Correos: la verificación exige código recibido por la persona.
                    # El guion no lee buzones ni genera códigos de prueba.
                    paso_actual = "correos_verificados"
                    correos = esperar_respuesta(pagina, RUTAS["correos"], "GET", pagina.reload)
                    lista = correos.get("correos")
                    if not isinstance(lista, list) or not any(
                            isinstance(c, dict) and c.get("estado") == "verificado" for c in lista):
                        raise Corte("correos_verificados", "no consta un correo verificado autorizado")
                    pagina.locator("[data-correos-raiz] .correos-estado--confirmado, [data-correos-raiz] .correos-estado--activo").first.wait_for()
                    resultado["pasos"].append({"paso": "correos_verificados", "estado": "CONSULTADO"})

                    # 3. Imagen: se elige otra paleta sin subir ficheros personales.
                    paso_actual = "imagen"
                    imagen = esperar_respuesta(pagina, RUTAS["imagen"], "GET", pagina.reload)
                    pagina.locator("[data-imagen-form]").wait_for()
                    paletas = pagina.locator("[name=imagen-paleta]").evaluate_all("els => els.map(el => el.value)")
                    actual = imagen.get("estado", {}).get("eleccion", {}).get("paleta")
                    paleta = next((p for p in paletas if p != actual), None)
                    if paleta is None:
                        raise Corte("imagen", "catálogo sin otra paleta")
                    pagina.locator(f"[name=imagen-paleta][value='{paleta}']").check()
                    recibo = esperar_respuesta(pagina, RUTAS["imagen"], "POST",
                                               lambda: pagina.locator("[data-imagen-form] button[type=submit]").click())
                    evidencia = huella_recibo(recibo, "imagen")
                    pagina.locator(".imagen-mensaje--exito").wait_for()
                    recuperado = esperar_respuesta(pagina, RUTAS["imagen"], "GET", pagina.reload)
                    if (recuperado.get("estado", {}).get("eleccion", {}).get("paleta") != paleta
                            or recuperado.get("estado", {}).get("version") != evidencia["version"]):
                        raise Corte("imagen", "GET no recupera la paleta guardada")
                    resultado["pasos"].append({"paso": "imagen", "estado": "RECUPERADO", **evidencia})

                    # 4. Mi Bolsa conserva su autoridad y su histórico propio.
                    paso_actual = "mi_bolsa"
                    bolsa = esperar_respuesta(pagina, RUTAS["bolsa"], "GET",
                                             lambda: pagina.goto(origen + "/area-personal/?vista=llamamientos", wait_until="domcontentloaded"))
                    esperar_respuesta(pagina, RUTAS["historial"], "GET", pagina.reload)
                    pagina.locator("#historial-mi-bolsa").wait_for()
                    pagina.wait_for_function("document.querySelector('#titulo-vista')?.textContent.trim() === 'Mi bolsa'")
                    if not bolsa or pagina.locator("#titulo-vista").inner_text().strip() != "Mi bolsa":
                        raise Corte("mi_bolsa", "consulta o vista propia incompleta")
                    resultado["pasos"].append({"paso": "mi_bolsa", "estado": "CONSULTADO"})

                    # 5. La ficha de Aspirantes #160 sigue fuera de esta base.
                    # Nunca se acepta el perfil visual genérico como sustituto.
                    paso_actual = "mi_ficha_aspirantes"
                    ficha = pagina.get_by_role("link", name="Mi ficha", exact=True)
                    if ficha.count() != 1:
                        raise Corte("mi_ficha_aspirantes", "no hay acceso a Mi ficha Aspirantes en este runtime")
                    ficha.click()
                    if "ficha" not in pagina.locator("#titulo-vista").inner_text().lower():
                        raise Corte("mi_ficha_aspirantes", "la ficha no confirma su vista propia")
                    resultado["pasos"].append({"paso": "mi_ficha_aspirantes", "estado": "VISTA; SIN EFECTO ACREDITADO"})
                    comprobar_pagina(pagina, contexto, errores_js, respuestas_externas, cookies_set)
                    resultado["estado"] = "COMPLETO_CON_LIMITES"
                except Corte as error:
                    resultado["primer_corte"] = {"paso": error.paso, "motivo": error.motivo}
                except Exception:
                    resultado["primer_corte"] = {"paso": paso_actual, "motivo": "interacción o tiempo de espera fallido"}
                finally:
                    try:
                        comprobar_pagina(pagina, contexto, errores_js, respuestas_externas, cookies_set)
                        resultado["viewports_comprobados"] = [1440]
                        pagina.set_viewport_size({"width": 390, "height": 844})
                        comprobar_pagina(pagina, contexto, errores_js, respuestas_externas, cookies_set)
                        resultado["viewports_comprobados"].append(390)
                    except Exception:
                        if resultado["primer_corte"] is None:
                            resultado["primer_corte"] = {"paso": "controles", "motivo": "controles de navegador fallidos"}
                            resultado["estado"] = "CORTE"
            finally:
                contexto.close()
        finally:
            navegador.close()
    return resultado


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--origen", required=True)
    parser.add_argument("--acta", required=True, type=Path)
    parser.add_argument("--certificado", required=True, type=Path)
    parser.add_argument("--clave", required=True, type=Path)
    args = parser.parse_args()
    try:
        origen, chrome = preparar(args.origen, args.acta, args.certificado, args.clave)
    except NoEjecutado as error:
        print(json.dumps({"estado": "NO EJECUTADO", "motivo": str(error)}, ensure_ascii=False))
        return 2
    try:
        resultado = ejecutar(origen, chrome, args.certificado, args.clave)
        print(json.dumps(resultado, ensure_ascii=False, sort_keys=True))
        return 0 if resultado["estado"] == "COMPLETO_CON_LIMITES" else 1
    except Exception:
        # No incluir excepciones de Playwright: pueden contener URL, cabeceras o datos.
        print(json.dumps({"estado": "CORTE", "primer_corte": {"paso": "runtime", "motivo": "fallo no clasificado; revisar en entorno aislado"}}, ensure_ascii=False))
        return 1


if __name__ == "__main__":
    sys.exit(main())
