#!/usr/bin/env python3
"""Recorre el área personal en un clon local sintético ya preparado.

La ausencia de una precondición termina con NO EJECUTADO (código 2), antes de
importar Playwright, abrir Chrome o conectar con el servidor.
"""

from __future__ import annotations

import argparse
import errno
import hashlib
import json
import os
import shutil
import stat
import subprocess
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


def raices_git() -> tuple[Path, ...]:
    try:
        comun = subprocess.run(
            ["git", "-C", str(RAIZ_REPO), "rev-parse", "--git-common-dir"],
            check=True, capture_output=True, text=True,
        ).stdout.strip()
        lista = subprocess.run(
            ["git", "-C", str(RAIZ_REPO), "worktree", "list", "--porcelain"],
            check=True, capture_output=True, text=True,
        ).stdout
        ruta_comun = Path(comun)
        if not ruta_comun.is_absolute():
            ruta_comun = RAIZ_REPO / ruta_comun
        compartida = ruta_comun.resolve().parent
        raices = set()
        for linea in lista.splitlines():
            if linea.startswith("worktree "):
                ruta = Path(linea[9:])
                raices.add((ruta if ruta.is_absolute() else RAIZ_REPO / ruta).resolve())
        if not comun or RAIZ_REPO not in raices or compartida not in raices:
            raise ValueError("inventario incompleto")
        return tuple(sorted(raices | {compartida}))
    except (OSError, subprocess.CalledProcessError, ValueError):
        raise NoEjecutado("Git: no se puede comprobar la raíz compartida y todos los worktrees") from None


def dentro_git(ruta: Path, raices: tuple[Path, ...]) -> bool:
    try:
        destino = ruta.resolve()
        return any(destino.is_relative_to(raiz) for raiz in raices)
    except OSError:
        raise NoEjecutado("material: no se puede comprobar su ubicación") from None


def sha256_archivo(ruta: Path) -> str:
    resumen = hashlib.sha256()
    with ruta.open("rb") as fichero:
        for bloque in iter(lambda: fichero.read(1024 * 1024), b""):
            resumen.update(bloque)
    return resumen.hexdigest()


def abrir_directorio_privado(ruta: Path) -> int:
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


def preparar_evidencias(ruta: Path) -> Path:
    directorio = None
    try:
        directorio = abrir_directorio_privado(ruta)
        os.mkdir(ruta.name, mode=0o700, dir_fd=directorio)
        absoluta = ruta if ruta.is_absolute() else Path.cwd() / ruta
        validar_capturas(absoluta)
        return absoluta
    except OSError:
        raise NoEjecutado("evidencias: se exige una carpeta nueva fuera de Git, sin enlaces y con padre propio 0700") from None
    finally:
        if directorio is not None:
            os.close(directorio)


def validar_capturas(evidencias: Path) -> None:
    for ancho in (1440, 390):
        directorio = None
        try:
            ruta = evidencias / f"area-personal-{ancho}.png"
            directorio = abrir_directorio_privado(ruta)
            try:
                os.stat(ruta.name, dir_fd=directorio, follow_symlinks=False)
            except FileNotFoundError:
                pass
            else:
                raise OSError(errno.EEXIST, "no se sobrescribe una captura existente")
        except OSError:
            raise NoEjecutado("evidencias: capturas ocupadas o directorio sin garantías privadas") from None
        finally:
            if directorio is not None:
                os.close(directorio)


def guardar_captura(ruta: Path, contenido: bytes) -> None:
    directorio = abrir_directorio_privado(ruta)
    try:
        descriptor = os.open(ruta.name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                             0o600, dir_fd=directorio)
        with os.fdopen(descriptor, "wb") as fichero:
            fichero.write(contenido)
    finally:
        os.close(directorio)


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
    raices = raices_git()
    if not acta.is_file() or dentro_git(acta, raices):
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
        binario_valido = (binario.is_file() and not dentro_git(binario, raices)
                          and isinstance(huella, str) and len(huella) == 64
                          and sha256_archivo(binario) == huella.lower())
    except OSError:
        binario_valido = False
    if not binario_valido:
        raise NoEjecutado("binario: falta o no coincide con el acta")
    for nombre, ruta in (("certificado", certificado), ("clave", clave)):
        try:
            valido = ruta.is_file() and ruta.stat().st_size > 0 and not dentro_git(ruta, raices)
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


def bloquear_websocket(ruta, incidencias: list[str]) -> None:
    incidencias.append("websocket_bloqueado")
    # Sin connect_to_server Playwright crea un socket simulado y no abre red.
    # Se descartan los mensajes para que tampoco se simule una respuesta.
    ruta.on_message(lambda _mensaje: None)


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


def ejecutar(origen: str, chrome: Path, certificado: Path, clave: Path,
             evidencias: Path | None = None) -> dict:
    if evidencias is not None:
        validar_capturas(evidencias)
    from playwright.sync_api import sync_playwright

    resultado = {"estado": "CORTE", "pasos": [], "primer_corte": None, "http": []}
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
                contexto.route_web_socket("**/*", lambda ruta: bloquear_websocket(ruta, respuestas_externas))
                pagina = contexto.new_page()
                pagina.on("pageerror", lambda error: errores_js.append(type(error).__name__))
                pagina.on("request", lambda peticion: respuestas_externas.append("externa")
                          if not mismo_origen(peticion.url, origen) else None)
                pagina.on("response", lambda respuesta: cookies_set.append("set-cookie")
                          if "set-cookie" in respuesta.headers else None)
                pagina.on("response", lambda respuesta: resultado["http"].append({
                    "metodo": respuesta.request.method,
                    "ruta": urlparse(respuesta.url).path,
                    "estado": respuesta.status,
                }) if urlparse(respuesta.url).path in RUTAS.values() else None)
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
                    pagina.wait_for_function("document.querySelector('#titulo-vista')?.textContent.trim() === 'Disponibilidad y llamamientos'")
                    if not bolsa or pagina.locator("#titulo-vista").inner_text().strip() != "Disponibilidad y llamamientos":
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
                    resultado["viewports_comprobados"] = []
                    resultado["capturas"] = []
                    for ancho, alto in ((1440, 900), (390, 844)):
                        try:
                            pagina.set_viewport_size({"width": ancho, "height": alto})
                            # Las vistas estabilizan su distribución antes de capturar.
                            pagina.wait_for_timeout(250)
                            if evidencias is not None:
                                nombre = f"area-personal-{ancho}.png"
                                guardar_captura(evidencias / nombre, pagina.screenshot(full_page=True))
                                resultado["capturas"].append(nombre)
                            comprobar_pagina(pagina, contexto, errores_js, respuestas_externas, cookies_set)
                            resultado["viewports_comprobados"].append(ancho)
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
    parser.add_argument("--evidencias", type=Path,
                        help="Carpeta nueva externa a Git para capturas sintéticas")
    args = parser.parse_args()
    try:
        origen, chrome = preparar(args.origen, args.acta, args.certificado, args.clave)
        evidencias = preparar_evidencias(args.evidencias) if args.evidencias is not None else None
    except NoEjecutado as error:
        print(json.dumps({"estado": "NO EJECUTADO", "motivo": str(error)}, ensure_ascii=False))
        return 2
    try:
        resultado = ejecutar(origen, chrome, args.certificado, args.clave, evidencias)
        print(json.dumps(resultado, ensure_ascii=False, sort_keys=True))
        return 0 if resultado["estado"] == "COMPLETO_CON_LIMITES" else 1
    except NoEjecutado as error:
        print(json.dumps({"estado": "NO EJECUTADO", "motivo": str(error)}, ensure_ascii=False))
        return 2
    except Exception:
        # No incluir excepciones de Playwright: pueden contener URL, cabeceras o datos.
        print(json.dumps({"estado": "CORTE", "primer_corte": {"paso": "runtime", "motivo": "fallo no clasificado; revisar en entorno aislado"}}, ensure_ascii=False))
        return 1


if __name__ == "__main__":
    sys.exit(main())
