#!/usr/bin/env python3
"""Recorre Auditoría RRHH en Chrome contra una aplicación y PG18 aislados.

La configuración privada se pasa como JSON en /dev/shm. No se guardan datos de
expedientes, certificados ni respuestas fuera de /dev/shm.
"""

from __future__ import annotations

import hashlib
from datetime import datetime
import http.client
import json
import os
from pathlib import Path
import re
import ssl
import subprocess
import sys
from urllib.parse import quote, urlsplit

from playwright.sync_api import expect, sync_playwright


RUTA = "/api/vec/auditoria/consultas"
RUTA_OPCIONES = "/api/vec/auditoria/opciones"
REFERENCIA = re.compile(r"^[A-Za-z0-9][A-Za-z0-9:_.-]{2,199}$")


def exigir(condicion: bool, mensaje: str) -> None:
    if not condicion:
        raise RuntimeError(mensaje)


def fixture(ruta: Path) -> dict:
    exigir(ruta.parent == Path("/dev/shm") and ruta.is_file() and not ruta.is_symlink()
           and ruta.stat().st_mode & 0o077 == 0, "fixture privado 0600 requerido en /dev/shm")
    datos = json.loads(ruta.read_text(encoding="utf-8"))
    url = urlsplit(datos["url"])
    exigir(url.scheme == "https" and url.hostname in {"localhost", "127.0.0.1", "::1"}
           and not (url.username or url.password or url.query or url.fragment)
           and url.path == "", "solo HTTPS loopback sin ruta base")
    exigir(datos["url"].rstrip("/") == datos["url"], "URL base sin barra final")
    for clave in ("ct_ref", "bolsa_ref"):
        exigir(bool(REFERENCIA.fullmatch(datos[clave])), f"{clave} inválida")
    for clave in ("ct_actor_ref", "bolsa_actor_ref"):
        exigir(bool(REFERENCIA.fullmatch(datos[clave])), f"{clave} inválida")
    exigir(isinstance(datos["bolsa_clicks"], list) and datos["bolsa_clicks"]
           and all(isinstance(s, str) and 0 < len(s) <= 200 for s in datos["bolsa_clicks"]),
           "falta navegación Bolsa desde ficha")
    for clave in ("ca", "ct_cert", "ct_key", "bolsa_cert", "bolsa_key", "hook"):
        exigir(Path(datos[clave]).is_file(), f"falta {clave}")
    exigir(os.access(datos["hook"], os.X_OK), "hook privado no ejecutable")
    exigir(re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d", datos["desde"])
           and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d", datos["hasta"]),
           "intervalo de formulario inválido")
    diferencia = datetime.fromisoformat(datos["hasta"]) - datetime.fromisoformat(datos["desde"])
    exigir(0 < diferencia.total_seconds() <= 31 * 86400, "intervalo fuera de 31 días")
    return datos


def hook(datos: dict, accion: str, *argumentos: str) -> dict | None:
    resultado = subprocess.run([datos["hook"], accion, *argumentos], check=True, capture_output=True,
                            text=True, timeout=90)
    if accion == "consumos_v3":
        valor = json.loads(resultado.stdout)
        exigir(isinstance(valor, dict) and set(valor) == {"ct", "bolsa"}
               and all(type(valor[k]) is int and valor[k] >= 0 for k in valor),
               "hook consumos_v3 debe devolver conteos CT/Bolsa acotados")
        return valor
    if accion == "frontera_ref":
        valor = json.loads(resultado.stdout)
        exigir(isinstance(valor, dict) and set(valor) == {"count", "ruta", "motivo", "actor_ref"}
               and valor["count"] == 1 and valor["ruta"] == RUTA
               and valor["motivo"] == "acceso_denegado" and isinstance(valor["actor_ref"], str),
               "hook frontera_ref no acreditó una denegación única")
        return valor
    exigir(not resultado.stdout.strip(), f"hook {accion} no debe imprimir datos privados")
    return None


def certificado(datos: dict, perfil: str) -> dict:
    return {"origin": datos["url"], "certPath": datos[f"{perfil}_cert"],
            "keyPath": datos[f"{perfil}_key"]}


def preflight_tls(datos: dict) -> None:
    # Chrome omite el CA privado en este ensayo local. Python verifica la
    # cadena y el nombre de host antes de cada pasada, sin redirecciones.
    for perfil in ("ct", "bolsa"):
        contexto = ssl.create_default_context(cafile=datos["ca"])
        contexto.load_cert_chain(datos[f"{perfil}_cert"], datos[f"{perfil}_key"])
        url = urlsplit(datos["url"])
        conexion = http.client.HTTPSConnection(url.hostname, url.port or 443,
                                                context=contexto, timeout=15)
        try:
            conexion.request("GET", RUTA_OPCIONES)
            respuesta = conexion.getresponse()
            exigir(respuesta.status == 200 and not respuesta.getheader("set-cookie"),
                   f"mTLS verificado pero opciones {perfil} no disponibles")
            exigir(respuesta.getheader("content-type", "").startswith("application/json"),
                   "opciones mTLS sin JSON")
            respuesta.read(8193)
        finally:
            conexion.close()


def abrir_auditoria(pagina, datos: dict, fuente: str) -> None:
    if fuente == "ct":
        url = (datos["url"] + "/portal-empleado/?expediente="
               + quote(datos["ct_ref"], safe="") + "#contratacion-temporal")
        respuesta = pagina.goto(url, wait_until="domcontentloaded")
        exigir(respuesta and respuesta.status == 200, "portal CT no disponible")
        pagina.locator("[data-ct-exp-auditoria-comun]").click(timeout=20000)
    else:
        respuesta = pagina.goto(datos["url"] + "/portal-empleado/", wait_until="domcontentloaded")
        exigir(respuesta and respuesta.status == 200, "portal Bolsa no disponible")
        for selector in datos["bolsa_clicks"]:
            pagina.locator(selector).click(timeout=20000)
    vista = pagina.locator("[data-auditoria-vista]")
    vista.wait_for(timeout=20000)
    esperado = datos["ct_ref" if fuente == "ct" else "bolsa_ref"]
    exigir(esperado in vista.locator(".auditoria-expediente").inner_text(), "ficha distinta de la autorizada")
    expect(vista.locator("[data-auditoria-filtros] button[type=submit]")).to_be_enabled(timeout=20000)


def comprobar_registros(cuerpo: dict, fuente: str, expediente: str) -> None:
    registros = cuerpo.get("registros")
    exigir(isinstance(registros, list) and registros, "lectura positiva sin historia")
    for fila in registros:
        exigir(fila.get("fuente") == fuente and fila.get("expediente_ref") == expediente,
               "fuente o expediente ajenos")
        exigir(all(isinstance(fila.get(k), str) and fila[k] for k in
                   ("id", "actor_ref", "accion", "ocurrido_en", "resultado")),
               "falta actor, instante o acción")
        exigir("motivo" in fila and "antes" in fila and "despues" in fila
               and "antes_sha256" in fila and "despues_sha256" in fila
               and "recibo_ref" in fila and "datos_disponibles" in fila,
               "falta motivo, antes/después o vínculo de expediente")
        exigir(fila["ocurrido_en"].endswith("Z") and isinstance(fila["datos_disponibles"], bool),
               "instante o disponibilidad inválidos")
    exigir(not cuerpo.get("siguiente_cursor"), "fixture con más de una página: historia incompleta")
    if fuente == "ct":
        exigir(any(fila["motivo"] and fila["recibo_ref"] and
                   ((fila["antes"] and fila["despues"]) or
                    (fila["antes_sha256"] and fila["despues_sha256"])) for fila in registros),
               "CT sin cambio con motivo, recibo y comparación")
    else:
        motivos = {fila["recibo_ref"] for fila in registros if fila["motivo"] and fila["recibo_ref"]}
        exigir(any(fila["recibo_ref"] in motivos and fila["datos_disponibles"]
                   and fila["antes"] and fila["despues"] for fila in registros),
               "Bolsa sin motivo y antes/después ligados por el mismo recibo")


def consultar_ui(pagina, datos: dict, fuente: str) -> tuple[str, dict]:
    vista = pagina.locator("[data-auditoria-vista]")
    vista.locator('input[name="desde"]').fill(datos["desde"])
    vista.locator('input[name="hasta"]').fill(datos["hasta"])
    with pagina.expect_response(lambda r: r.url.endswith(RUTA) and r.request.method == "POST", timeout=20000) as espera:
        vista.locator("[data-auditoria-filtros] button[type=submit]").click()
    respuesta = espera.value
    exigir(respuesta.status == 200 and not respuesta.header_value("set-cookie"), "POST Auditoría no fue 200 sin cookie")
    solicitud = respuesta.request.post_data_json
    expediente = datos["ct_ref" if fuente == "ct" else "bolsa_ref"]
    exigir(solicitud["fuente"] == fuente and solicitud["expediente_ref"] == expediente,
           "el cliente envió un contexto distinto")
    cuerpo = respuesta.json()
    comprobar_registros(cuerpo, fuente, expediente)
    pagina.locator('[data-auditoria-vista][data-estado="disponible"]').wait_for(timeout=10000)
    vista.locator(".auditoria-tabla tbody tr").first.locator("details summary").click()
    detalle = vista.locator(".auditoria-detalle").first.inner_text()
    exigir(expediente in detalle and ("Antes" in detalle or "antes" in detalle),
           "la ficha no muestra expediente y comparación")
    huella = hashlib.sha256(json.dumps(cuerpo, sort_keys=True, ensure_ascii=False).encode()).hexdigest()
    return huella, solicitud


def higiene(pagina, contexto, ancho: int, errores: list[str], respuestas: list[int]) -> None:
    metricas = pagina.evaluate("""async () => ({ancho: document.documentElement.scrollWidth,
      ventana: innerWidth, cookies: document.cookie, local: localStorage.length,
      sesion: sessionStorage.length, indexed: indexedDB.databases ? (await indexedDB.databases()).length : 0,
      cache: 'caches' in window ? (await caches.keys()).length : 0})""")
    exigir(metricas["ancho"] <= metricas["ventana"] + 1, f"desbordamiento global en {ancho}px")
    exigir(not metricas["cookies"] and not contexto.cookies(), "cookies persistidas")
    exigir(all(metricas[k] == 0 for k in ("local", "sesion", "indexed", "cache")),
           "almacenamiento web no vacío")
    exigir(not errores and not respuestas, "errores JS o respuestas 5xx")


def pasada(navegador, datos: dict, etiqueta: str, salida: Path) -> tuple[dict, dict]:
    pruebas = {}
    solicitudes = {}
    denegaciones = {}
    for fuente in ("ct", "bolsa"):
        for ancho in (1440, 390):
            contexto = navegador.new_context(viewport={"width": ancho, "height": 900},
                ignore_https_errors=True,
                client_certificates=[certificado(datos, fuente)], service_workers="block")
            pagina = contexto.new_page()
            errores, fallos, opciones = [], [], []
            pagina.on("pageerror", lambda error: errores.append(type(error).__name__))
            pagina.on("response", lambda r: fallos.append(r.status) if r.status >= 500 else None)
            pagina.on("response", lambda r: opciones.append(r.status)
                      if r.url.endswith(RUTA_OPCIONES) and r.request.method == "GET" else None)
            try:
                abrir_auditoria(pagina, datos, fuente)
                exigir(opciones == [200], "el navegador no leyó opciones de la ruta compuesta")
                huella, solicitud = consultar_ui(pagina, datos, fuente)
                pruebas[f"{fuente}_{ancho}"] = huella
                solicitudes[fuente] = solicitud
                higiene(pagina, contexto, ancho, errores, fallos)
                pagina.screenshot(path=str(salida / f"{etiqueta}_{fuente}_{ancho}.png"), full_page=True)
            finally:
                contexto.close()
    # La misma solicitud positiva debe recibir 403 con el certificado de la otra fuente.
    for fuente, ajeno in (("ct", "bolsa"), ("bolsa", "ct")):
        contexto = navegador.new_context(ignore_https_errors=True,
            client_certificates=[certificado(datos, ajeno)])
        try:
            respuesta = contexto.request.post(datos["url"] + RUTA,
                data=json.dumps(solicitudes[fuente]), headers={"Content-Type": "application/json"}, timeout=15000)
            exigir(respuesta.status == 403 and not respuesta.headers.get("set-cookie"),
                   f"permiso cruzado {ajeno}→{fuente}: {respuesta.status}")
            exigir(respuesta.headers.get("content-type", "").startswith("application/json")
                   and respuesta.body() == b'{"error":"consulta denegada"}\n',
                   "respuesta 403 expuso datos ajenos")
            correlacion = respuesta.headers.get("x-correlation-ref", "")
            exigir(bool(re.fullmatch(r"corr_[0-9a-f]{32}", correlacion))
                   and correlacion not in denegaciones, "403 sin correlación única")
            fila = hook(datos, "frontera_ref", correlacion)
            exigir(fila["actor_ref"] == datos[f"{ajeno}_actor_ref"],
                   "denegación CT136 sin actor del certificado ajeno")
            denegaciones[correlacion] = fila
        finally:
            contexto.close()
    return pruebas, denegaciones


def main() -> None:
    exigir(os.environ.get("VEC_AUDITORIA_WEB_SINTETICO") == "1", "declarar fixture sintético")
    exigir(len(sys.argv) == 2, "uso: recorrido.py /dev/shm/fixture-auditoria.json")
    datos = fixture(Path(sys.argv[1]))
    salida = Path("/dev/shm") / f"vec-auditoria-web-{os.getpid()}"
    salida.mkdir(mode=0o700)
    iniciada = False
    try:
        hook(datos, "start"); iniciada = True
        preflight_tls(datos)
        with sync_playwright() as playwright:
            navegador = playwright.chromium.launch(headless=True, channel="chrome")
            try:
                consumos_antes = hook(datos, "consumos_v3")
                primera, denegaciones_antes = pasada(navegador, datos, "antes", salida)
                consumos_primera = hook(datos, "consumos_v3")
                exigir(all(consumos_primera[k] == consumos_antes[k] + 2 for k in ("ct", "bolsa")),
                       "dos lecturas por fuente no dejaron consumos V3 exactos")
                hook(datos, "stop"); iniciada = False
                hook(datos, "restart_pg")
                hook(datos, "start"); iniciada = True
                preflight_tls(datos)
                exigir(hook(datos, "consumos_v3") == consumos_primera,
                       "consumos V3 cambiaron tras reiniciar")
                for correlacion, fila in denegaciones_antes.items():
                    exigir(hook(datos, "frontera_ref", correlacion) == fila,
                           "denegación CT136 cambió tras reiniciar")
                segunda, denegaciones_despues = pasada(navegador, datos, "despues", salida)
                exigir(primera == segunda, "historia cambió tras reiniciar aplicación y PostgreSQL")
                exigir(not set(denegaciones_antes).intersection(denegaciones_despues),
                       "correlaciones repetidas tras reiniciar")
                exigir(all(hook(datos, "frontera_ref", correlacion) == fila
                           for correlacion, fila in denegaciones_antes.items()),
                       "denegaciones anteriores alteradas tras nueva lectura")
                consumos_final = hook(datos, "consumos_v3")
                exigir(all(consumos_final[k] == consumos_primera[k] + 2 for k in ("ct", "bolsa")),
                       "consumos V3 posteriores al reinicio incorrectos")
            finally:
                navegador.close()
        print("OK Chrome 1440/390: CT/Bolsa, actor, instante, antes/después, motivo, expediente, "
              "403 auditable y recuperación; capturas efímeras en " + str(salida))
    finally:
        try:
            if iniciada:
                hook(datos, "stop")
        finally:
            hook(datos, "cleanup")


if __name__ == "__main__":
    try:
        main()
    except (KeyError, ValueError, OSError, RuntimeError, subprocess.SubprocessError) as error:
        print(f"FALLO recorrido Auditoría web: {error}", file=sys.stderr)
        sys.exit(1)
