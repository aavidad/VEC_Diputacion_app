#!/usr/bin/env python3
"""Ensayo aislado de Dietas: navegador, API, PostgreSQL, recibo y recuperación.

No arranca ni reinicia servicios. La fase ``antes`` crea una comisión sintética
desde el navegador. Después de reiniciar *externamente* la instancia aislada,
``despues`` comprueba detalle, replay y cardinalidad sin crear otra comisión.

Preparación manual (sólo una instancia desechable PG18/Go distinta de principal):
  1. Crear ~/.local/state/vec-dietas-ensayo/instancia.json (modo 0600):
     {"purpose":"vec-dietas-ensayo-aislado-v1",
      "origin":"https://localhost:18443",
      "pg_service":"vec_dietas_ensayo_1",
      "database":"vec_dietas_ensayo_1",
      "pg_container":"vec-dietas-ensayo-pg18-20260924",
      "pg_host_port":15565,
      "portal_path":"/portal-empleado/",
      "app_pid_file":"/home/alberto/.local/state/vec-dietas-ensayo/vec-server.pid"}
  2. Configurar ese servicio libpq en PGSERVICEFILE privado; el DSN y las
     credenciales nunca se pasan por argumentos ni se guardan en el informe.
  3. Exportar VEC_DIETAS_ENSAYO_CA, _CERT, _KEY, _CT_CERT, _CT_KEY y
     _RELACION_REF, apuntando exclusivamente a identidades sintéticas de la
     instancia (empleado y RRHH separados). Opcionalmente
     VEC_DIETAS_ENSAYO_FECHA (AAAA-MM-DD, fecha habilitada en el fixture).
  4. Ejecutar --fase preflight, --fase papeles, --fase antes, --fase documento,
     --fase enviar; reiniciar sólo la instancia aislada mediante su
     procedimiento externo; ejecutar --fase despues. ``papeles`` precisa
     certificados sintéticos separados para administrativo, responsable e
     Intervención. ``enviar`` depende del circuito D6 instalado.

El marcador es una barrera operativa, no una autoridad de identidad. El servidor
debe aplicar su mTLS, autorización V3, roles y asignación de Personal reales.
"""

from __future__ import annotations

import argparse
import configparser
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import socket
import ssl
import subprocess
import sys
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlsplit
from urllib.request import Request, urlopen

try:
    from playwright.sync_api import Error as PlaywrightError
except ImportError:
    class PlaywrightError(Exception):
        """Permite informar la ausencia de Playwright en la fase de navegador."""


PRIVADO = Path.home() / ".local/state/vec-dietas-ensayo"
MARCADOR = PRIVADO / "instancia.json"
ESTADO = PRIVADO / "resultado.json"
SERVICIO_PG = PRIVADO / "pg_service.conf"
RUTA = "/api/vec/dietas/comisiones"
REF = re.compile(r"^dco_[A-Za-z0-9_-]{22,128}$")
REL = re.compile(r"^rel_[A-Za-z0-9_-]{22,128}$")
KEY = re.compile(r"^[A-Za-z0-9_-]{16,128}$")
DB = re.compile(r"^vec_dietas_ensayo_[a-z0-9_]{1,40}$")
FECHA = re.compile(r"^\d{4}-\d{2}-\d{2}$")


class EnsayoError(Exception):
    pass


def exigir(condicion: bool, mensaje: str) -> None:
    if not condicion:
        raise EnsayoError(mensaje)


def leer_json_privado(ruta: Path) -> dict:
    exigir(ruta.is_file() and not ruta.is_symlink(), f"falta fichero privado {ruta.name}")
    exigir(ruta.stat().st_mode & 0o077 == 0, f"permisos inseguros de {ruta.name}; usar chmod 600")
    dato = json.loads(ruta.read_text(encoding="utf-8"))
    exigir(isinstance(dato, dict), f"{ruta.name} no es objeto JSON")
    return dato


def configurar() -> dict:
    exigir(PRIVADO.is_dir() and not PRIVADO.is_symlink(), f"crear directorio privado {PRIVADO} con modo 700")
    exigir(PRIVADO.stat().st_mode & 0o077 == 0, "directorio de ensayo con permisos inseguros")
    cfg = leer_json_privado(MARCADOR)
    exigir(cfg.get("purpose") == "vec-dietas-ensayo-aislado-v1", "marcador de instancia aislada inválido")
    u = urlsplit(cfg.get("origin", ""))
    exigir(u.scheme == "https" and u.hostname in ("localhost", "127.0.0.1", "::1") and
           u.port not in (None, 443, 8443, 9443) and not u.path and not u.query and not u.fragment,
           "origin debe ser HTTPS local en puerto exclusivo; principal está excluida")
    direcciones = socket.getaddrinfo(u.hostname, u.port, type=socket.SOCK_STREAM)
    exigir(direcciones and all(d[4][0] in ("127.0.0.1", "::1") for d in direcciones),
           "origin no resuelve exclusivamente a loopback")
    exigir(DB.fullmatch(cfg.get("pg_service", "")) and DB.fullmatch(cfg.get("database", "")),
           "servicio y base PG deben llamarse vec_dietas_ensayo_…")
    exigir(cfg.get("pg_container") == "vec-dietas-ensayo-pg18-20260924" and
           cfg.get("pg_host_port") == 15565,
           "marcador PG no corresponde al contenedor y puerto dedicados")
    portal = cfg.get("portal_path", "")
    exigir(isinstance(portal, str) and portal.startswith("/") and portal.endswith("/") and
           ".." not in portal and "?" not in portal and "#" not in portal,
           "portal_path inválido")
    cfg["portal_path"] = portal
    pid_file = Path(cfg.get("app_pid_file", ""))
    exigir(pid_file.parent == PRIVADO and pid_file.name.endswith(".pid"),
           "app_pid_file debe ser un PID de Go dentro del directorio aislado")
    cfg["app_pid_file"] = str(pid_file)
    return cfg


def material_tls(ct: bool = False) -> tuple[Path, Path, Path]:
    rutas = []
    sufijo = "_CT" if ct else ""
    for nombre in ("VEC_DIETAS_ENSAYO_CA", "VEC_DIETAS_ENSAYO" + sufijo + "_CERT",
                   "VEC_DIETAS_ENSAYO" + sufijo + "_KEY"):
        valor = os.environ.get(nombre, "")
        exigir(valor != "", f"falta {nombre}")
        ruta = Path(valor).resolve(strict=True)
        exigir(ruta.is_file() and ruta.is_relative_to(PRIVADO / "material"),
               f"{nombre} debe ser material sintético aislado")
        if nombre.endswith("_KEY"):
            exigir(ruta.stat().st_mode & 0o077 == 0, f"clave {nombre} con permisos inseguros")
        rutas.append(ruta)
    return tuple(rutas)


def relacion() -> str:
    valor = os.environ.get("VEC_DIETAS_ENSAYO_RELACION_REF", "")
    exigir(bool(REL.fullmatch(valor)), "falta relación sintética rel_… de Personal")
    return valor


def fecha_ensayo() -> str:
    valor = os.environ.get("VEC_DIETAS_ENSAYO_FECHA", "2026-09-24")
    exigir(bool(FECHA.fullmatch(valor)), "VEC_DIETAS_ENSAYO_FECHA no válida")
    return valor


def psql_disponible() -> str:
    ruta = shutil.which("psql") or "/opt/metasploit-framework/embedded/bin/psql"
    exigir(Path(ruta).is_file(), "falta cliente psql; instalar cliente PostgreSQL 18 o indicar PATH")
    return ruta


def pg(cfg: dict, consulta: str) -> str:
    entorno = os.environ.copy()
    exigir(SERVICIO_PG.is_file() and not SERVICIO_PG.is_symlink() and
           SERVICIO_PG.stat().st_mode & 0o077 == 0,
           "falta servicio libpq privado de la instancia")
    for nombre in ("PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGSERVICE"):
        entorno.pop(nombre, None)
    entorno["PGSERVICEFILE"] = str(SERVICIO_PG)
    entorno["PGSERVICE"] = cfg["pg_service"]
    entorno["PGCONNECT_TIMEOUT"] = "5"
    # El servicio libpq y su contraseña permanecen fuera del argumento y Git.
    proc = subprocess.run(
        [psql_disponible(), "-X", "-w", "-A", "-t", "-v", "ON_ERROR_STOP=1", "-c", consulta],
        env=entorno, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
        text=True, timeout=12, check=False,
    )
    exigir(proc.returncode == 0, "PostgreSQL aislado no accesible o SELECT no autorizado")
    return proc.stdout.strip()


def pg_identidad(cfg: dict) -> dict:
    servicio = configparser.ConfigParser(interpolation=None)
    exigir(SERVICIO_PG.is_file() and SERVICIO_PG.stat().st_mode & 0o077 == 0,
           "servicio libpq privado ausente o inseguro")
    servicio.read(SERVICIO_PG)
    opciones = servicio[cfg["pg_service"]] if servicio.has_section(cfg["pg_service"]) else {}
    exigir(opciones.get("host") == "127.0.0.1" and
           opciones.get("port") == str(cfg["pg_host_port"]) and
           opciones.get("dbname") == cfg["database"] and
           opciones.get("sslmode") == "verify-full" and
           opciones.get("sslrootcert") == str(PRIVADO / "pg-ca/ca.crt"),
           "servicio libpq no apunta exactamente al PG18 aislado con TLS verificado")
    proc = subprocess.run(["docker", "inspect", cfg["pg_container"]],
                          stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                          text=True, timeout=5, check=False)
    exigir(proc.returncode == 0, "contenedor PG18 dedicado no existe")
    contenedor = json.loads(proc.stdout)[0]
    puertos = contenedor.get("NetworkSettings", {}).get("Ports", {}).get("5432/tcp")
    exigir(contenedor.get("Name") == "/" + cfg["pg_container"] and
           contenedor.get("State", {}).get("Running") is True and
           contenedor.get("Config", {}).get("Image") == "postgres:18.4-alpine" and
           contenedor.get("Config", {}).get("Labels", {}).get("vec.dietas.ensayo") == "20260924" and
           puertos == [{"HostIp": "127.0.0.1", "HostPort": str(cfg["pg_host_port"])}] and
           any(m.get("Name") == "vec-dietas-ensayo-pg18-data-20260924"
               for m in contenedor.get("Mounts", [])),
           "contenedor PG no coincide con imagen, volumen o puerto aislados")
    dato = json.loads(pg(cfg, "SELECT json_build_object('base',current_database(),"
                          "'version',current_setting('server_version_num')::int,"
                          "'inicio',pg_postmaster_start_time(),"
                          "'puerto',inet_server_port(),"
                          "'direccion',host(inet_server_addr()),"
                          "'tls',(SELECT ssl FROM pg_stat_ssl WHERE pid=pg_backend_pid()))::text;"))
    exigir(dato["base"] == cfg["database"] and 180000 <= dato["version"] < 190000,
           "PG no es la base aislada marcada o no es PostgreSQL 18")
    exigir(dato["tls"] is True and dato["puerto"] == 5432 and isinstance(dato["direccion"], str) and
           dato["direccion"] != "", "PG no responde dentro del contenedor dedicado")
    return dato


def app_identidad(cfg: dict) -> str:
    ruta = Path(cfg["app_pid_file"])
    exigir(ruta.is_file() and not ruta.is_symlink() and ruta.stat().st_mode & 0o077 == 0,
           "falta PID privado de Go aislado (modo 600)")
    pid_texto = ruta.read_text(encoding="ascii").strip()
    exigir(pid_texto.isdecimal(), "PID de Go inválido")
    pid = int(pid_texto)
    proc = Path("/proc") / str(pid)
    try:
        cmd = (proc / "cmdline").read_bytes().split(b"\0")[0]
        stat = (proc / "stat").read_text(encoding="ascii")
    except OSError as error:
        raise EnsayoError("PID de Go aislado no está vivo") from error
    exigir(b"vec-server" in cmd, "PID señalado no ejecuta vec-server")
    # Campo 22 de /proc/PID/stat; el nombre de proceso entre paréntesis puede
    # contener espacios, por eso se separa después del último paréntesis.
    inicio = stat.rsplit(") ", 1)[1].split()[19]
    return f"{pid}:{inicio}"


def pg_estado(cfg: dict, referencia: str) -> dict:
    exigir(bool(REF.fullmatch(referencia)), "referencia de comisión inválida")
    # La referencia ya está limitada a un alfabeto sin comillas; se consulta
    # sólo la base aislada y únicamente después de verificar su identidad.
    q = f"""SELECT json_build_object(
      'alta',(SELECT count(*) FROM vec_dietas.borrador_comision WHERE referencia='{referencia}'),
      'recibo_alta',(SELECT count(*) FROM vec_dietas.recibo_borrador_comision WHERE comision_ref='{referencia}'),
      'historia_alta',(SELECT count(*) FROM vec_dietas.historia_borrador_comision WHERE comision_ref='{referencia}'),
      'versiones',(SELECT coalesce(json_agg(version ORDER BY version),'[]'::json) FROM vec_dietas.comision_revision WHERE comision_ref='{referencia}'),
      'recibos',(SELECT count(*) FROM vec_dietas.recibo_operacion_comision WHERE comision_ref='{referencia}'),
      'historia',(SELECT count(*) FROM vec_dietas.historia_operacion_comision WHERE comision_ref='{referencia}'),
      'outbox',(SELECT count(*) FROM vec_dietas.outbox_comision WHERE comision_ref='{referencia}')
    )::text;"""
    dato = json.loads(pg(cfg, q))
    exigir(dato["alta"] == dato["recibo_alta"] == dato["historia_alta"] == 1,
           "alta v1, recibo o historia no tienen cardinalidad unitaria")
    versiones = dato["versiones"]
    exigir(isinstance(versiones, list) and versiones == list(range(2, 2 + len(versiones))),
           "versiones de comisión no son consecutivas")
    exigir(dato["recibos"] == dato["historia"] == len(versiones),
           "recibos/historia de operaciones no coinciden con versiones")
    return dato


def contexto_tls(papel: str = "") -> ssl.SSLContext:
    if papel:
        ca = material_tls()[0]
        prefijo = "VEC_DIETAS_ENSAYO_" + papel.upper()
        cert_valor, key_valor = os.environ.get(prefijo + "_CERT", ""), os.environ.get(prefijo + "_KEY", "")
        exigir(cert_valor != "" and key_valor != "", f"falta certificado sintético de {papel}")
        cert, key = Path(cert_valor).resolve(strict=True), Path(key_valor).resolve(strict=True)
        exigir(cert.is_file() and key.is_file() and
               cert.is_relative_to(PRIVADO / "material") and key.is_relative_to(PRIVADO / "material") and
               key.stat().st_mode & 0o077 == 0,
               f"certificado de {papel} fuera de material sintético aislado")
    else:
        ca, cert, key = material_tls()
    contexto = ssl.create_default_context(cafile=str(ca))
    contexto.load_cert_chain(str(cert), str(key))
    return contexto


def pedir(cfg: dict, metodo: str, ruta: str, cuerpo: dict | None = None) -> tuple[int, dict]:
    datos = None if cuerpo is None else json.dumps(cuerpo, ensure_ascii=False).encode()
    cabeceras = {"Accept": "application/json"}
    if datos is not None:
        cabeceras["Content-Type"] = "application/json; charset=utf-8"
    try:
        with urlopen(Request(cfg["origin"] + ruta, data=datos, headers=cabeceras, method=metodo),
                     context=contexto_tls(), timeout=20) as respuesta:
            exigir(respuesta.headers.get("Set-Cookie") is None, "API emitió Set-Cookie")
            bruto = respuesta.read(131073)
            exigir(len(bruto) <= 131072, "respuesta API excesiva")
            return respuesta.status, json.loads(bruto)
    except HTTPError as error:
        raise EnsayoError(f"{metodo} {ruta}: HTTP {error.code}") from error
    except URLError as error:
        raise EnsayoError(f"{metodo} {ruta}: transporte TLS no disponible") from error


def estado_lectura_papel(cfg: dict, papel: str, ruta: str) -> int:
    try:
        with urlopen(Request(cfg["origin"] + ruta, headers={"Accept": "application/json"}, method="GET"),
                     context=contexto_tls(papel), timeout=20) as respuesta:
            exigir(respuesta.headers.get("Set-Cookie") is None, "API emitió Set-Cookie")
            respuesta.read(131073)
            return respuesta.status
    except HTTPError as error:
        error.close()
        return error.code
    except URLError as error:
        raise EnsayoError(f"bandeja {papel}: transporte TLS no disponible") from error


def comprobar_papeles(cfg: dict) -> None:
    # Las cuatro bandejas se consultan con certificados distintos; el nombre
    # del papel no confiere permiso. Cada 200 debe provenir de V3 y su ámbito.
    pruebas = (
        ("ADMIN", "revision"), ("RESPONSABLE", "autorizacion"),
        ("CT", "liquidacion"), ("INTERVENCION", "fiscalizacion"),
    )
    for papel, etapa in pruebas:
        ruta = "/api/vec/dietas/comisiones/circuito?etapa=" + etapa
        estado = estado_lectura_papel(cfg, papel, ruta)
        exigir(estado == 200, f"bandeja {etapa} para {papel}: HTTP {estado}, falta ruta o concesión exacta")
    denegado = estado_lectura_papel(cfg, "", "/api/vec/dietas/comisiones/circuito?etapa=revision")
    exigir(denegado in (401, 403),
           f"empleado accedió a bandeja administrativa o ruta ausente: HTTP {denegado}")


def comprobar_item(dato: dict, *, referencia: str | None = None) -> tuple[dict, dict]:
    exigir(isinstance(dato, dict) and isinstance(dato.get("comision"), dict) and
           isinstance(dato.get("recibo"), dict), "respuesta sin comisión o recibo")
    comision, recibo = dato["comision"], dato["recibo"]
    exigir(bool(REF.fullmatch(comision.get("referencia", ""))) and
           (referencia is None or comision["referencia"] == referencia),
           "referencia de comisión inconsistente")
    exigir(isinstance(recibo.get("version"), int) and recibo["version"] >= 1 and
           isinstance(recibo.get("registrado_en"), str) and
           isinstance(recibo.get("referencia"), str) and recibo["referencia"].startswith("rcd_"),
           "recibo incompleto")
    exigir(comision.get("relacion_ref") == relacion() and
           comision.get("codigos_ruta") == ["18087", "18175"] and
           comision.get("calculo", {}).get("procedencia") == "osrm_interno" and
           str(comision.get("calculo", {}).get("version_tarifa", "")).startswith("provisional:"),
           "relación, ruta OSRM o tarifa provisional incompatibles")
    return comision, recibo


def comprobar_asignacion(cfg: dict) -> dict:
    estado, relaciones = pedir(cfg, "GET", "/api/vec/personal/relaciones-dietas")
    exigir(estado == 200 and isinstance(relaciones.get("relaciones_autorizadas"), list),
           "Personal no devuelve relaciones autorizadas")
    candidatas = [r for r in relaciones["relaciones_autorizadas"]
                  if r.get("relacion_ref") == relacion()]
    exigir(len(candidatas) == 1 and isinstance(candidatas[0].get("unidad_ref"), str),
           "relación sintética no autorizada o ambigua en Personal")
    unidad = candidatas[0]["unidad_ref"]
    ruta = ("/api/vec/personal/asignaciones-dietas/" + quote(relacion()) +
            "?fecha_referencia=" + quote(fecha_ensayo()) + "&unidad_ref=" + quote(unidad))
    estado, dato = pedir(cfg, "GET", ruta)
    a = dato.get("asignacion", {})
    exigir(estado == 200 and a.get("relacion_ref") == relacion() and
           a.get("unidad_ref") == unidad and a.get("grupo_dieta") in (1, 2, 3) and
           a.get("administrativo_persona_ref") and a.get("responsable_persona_ref") and
           len({a.get("persona_ref"), a.get("administrativo_persona_ref"),
                a.get("responsable_persona_ref")}) == 3,
           "asignación Personal no vigente o papeles no segregados")
    return a


def resumen_immutable(item: dict) -> dict:
    copia = json.loads(json.dumps(item))
    copia["recibo"].pop("repeticion", None)
    return copia


def ruta_detalle(ref: str) -> str:
    exigir(bool(REF.fullmatch(ref)), "referencia de comisión inválida")
    return RUTA + "/" + quote(ref) + "?relacion_ref=" + quote(relacion())


def verificar_operacion(cfg: dict, metodo: str, ruta: str, cuerpo: dict,
                        version: int, estado_esperado: str) -> dict:
    estado, item = pedir(cfg, metodo, ruta, cuerpo)
    comision, recibo = comprobar_item(item)
    exigir(estado in (200, 201) and recibo["version"] == version and
           comision.get("version") == version and comision.get("estado") == estado_esperado,
           f"{metodo} no devolvió versión {version} y estado {estado_esperado}")
    estado_detalle, detalle = pedir(cfg, "GET", ruta_detalle(comision["referencia"]))
    exigir(estado_detalle == 200 and resumen_immutable(detalle) == resumen_immutable(item),
           "detalle no conserva operación y recibo")
    estado_replay, replay = pedir(cfg, metodo, ruta, cuerpo)
    exigir(estado_replay == 200 and resumen_immutable(replay) == resumen_immutable(item) and
           replay["recibo"].get("repeticion") is True,
           "replay de operación no conserva versión y recibo")
    return item


def navegador(cfg: dict) -> tuple[dict, dict]:
    try:
        from playwright.sync_api import sync_playwright
    except ImportError as error:
        raise EnsayoError("falta Playwright Python para el recorrido de navegador") from error
    ca, cert, key = material_tls()
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        try:
            ctx = browser.new_context(
                client_certificates=[{"origin": cfg["origin"], "certPath": str(cert), "keyPath": str(key)}],
                ignore_https_errors=False, service_workers="block", locale="es-ES",
                viewport={"width": 1440, "height": 900}, accept_downloads=False,
            )
            # Playwright usa la confianza del sistema; CA propia debe estar
            # instalada en el perfil aislado. Un error TLS detiene el ensayo.
            page = ctx.new_page()
            errores = []
            page.on("pageerror", lambda error: errores.append(str(error)))
            # CT/RRHH se comprueba antes de escribir y con identidad propia.
            # El empleado no hereda permisos por tener Dietas visible.
            _, cert_ct, key_ct = material_tls(ct=True)
            ctx_ct = browser.new_context(
                client_certificates=[{"origin": cfg["origin"], "certPath": str(cert_ct),
                                      "keyPath": str(key_ct)}],
                ignore_https_errors=False, service_workers="block", locale="es-ES",
                viewport={"width": 1440, "height": 900}, accept_downloads=False,
            )
            pagina_ct = ctx_ct.new_page()
            errores_ct = []
            pagina_ct.on("pageerror", lambda error: errores_ct.append(str(error)))
            pagina_ct.goto(cfg["origin"] + cfg["portal_path"] + "#contratacion-temporal",
                           wait_until="domcontentloaded", timeout=20000)
            pagina_ct.locator('[data-modulo="contratacion-temporal"]').wait_for(timeout=15000)
            exigir(not ctx_ct.cookies() and not errores_ct, "CT tiene cookies o errores JS")
            ctx_ct.close()
            page.goto(cfg["origin"] + cfg["portal_path"] + "#dietas", wait_until="domcontentloaded", timeout=20000)
            page.locator("[data-dietas-recorridos]").wait_for(timeout=15000)
            page.locator("[data-dietas-abrir-nueva-comision]").click()
            form = page.locator("[data-dietas-borrador-form]")
            form.wait_for(timeout=10000)
            fecha = fecha_ensayo()
            for nombre, valor in (("fecha_inicio", fecha), ("fecha_fin", fecha),
                                  ("hora_inicio", "08:00"), ("hora_fin", "18:00"),
                                  ("motivo", "Ensayo sintético de comisión completa")):
                form.locator(f'[name="{nombre}"]').fill(valor)
            form.locator('[name="origen_codigo"]').select_option("18087")
            form.locator('[name="destino_codigo"]').select_option("18175")
            selector_relacion = form.locator('[name="relacion_ref"]')
            if selector_relacion.count():
                selector_relacion.select_option(relacion())
            with page.expect_response(lambda r: r.request.method == "POST" and
                                      urlsplit(r.url).path == RUTA, timeout=30000) as pendiente:
                form.locator("[data-dietas-borrador-guardar]").click()
            respuesta = pendiente.value
            exigir(respuesta.status == 201, f"POST navegador respondió HTTP {respuesta.status}")
            exigir("set-cookie" not in respuesta.headers, "navegador recibió Set-Cookie")
            solicitud = respuesta.request.post_data_json
            exigir(isinstance(solicitud, dict) and KEY.fullmatch(solicitud.get("clave_idempotencia", "")) and
                   solicitud.get("relacion_ref") == relacion(), "POST navegador sin clave o relación esperada")
            bruto = respuesta.body()
            exigir(len(bruto) <= 131072, "respuesta navegador excesiva")
            item = json.loads(bruto)
            comision, _ = comprobar_item(item)
            page.locator("[data-dietas-borrador-recibo]").wait_for(timeout=10000)
            exigir(not ctx.cookies() and not errores, "cookies o errores JS en navegador")
            return item, solicitud
        finally:
            browser.close()


def guardar(estado: dict) -> None:
    temporal = ESTADO.with_suffix(".tmp")
    descriptor = os.open(temporal, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8") as fichero:
            json.dump(estado, fichero, ensure_ascii=False, indent=2)
            fichero.write("\n")
            fichero.flush()
            os.fsync(fichero.fileno())
        os.replace(temporal, ESTADO)
    finally:
        temporal.unlink(missing_ok=True)


def antes(cfg: dict) -> None:
    exigir(not ESTADO.exists(), "resultado.json ya existe; usar fase despues o preservar y retirar manualmente en la instancia aislada")
    identidad = pg_identidad(cfg)
    app = app_identidad(cfg)
    asignacion = comprobar_asignacion(cfg)
    item, solicitud = navegador(cfg)
    comision, recibo = comprobar_item(item)
    estado, detalle = pedir(cfg, "GET", ruta_detalle(comision["referencia"]))
    exigir(estado == 200 and resumen_immutable(detalle) == resumen_immutable(item),
           "GET no conserva la comisión o recibo del navegador")
    sql = pg_estado(cfg, comision["referencia"])
    exigir(sql["versiones"] == [], "el alta del navegador añadió revisiones inesperadas")
    guardar({"origin": cfg["origin"], "database": cfg["database"],
             "pg_start_before": identidad["inicio"], "app_before": app,
             "item": item, "ultimo_item": item, "solicitud": solicitud, "sql": sql,
             "asignacion": asignacion})
    print("ANTES: asignación Personal segregada, navegador POST 201, API GET 200, PG18 alta/recibo/historia 1/1/1, CT visible")


def documento(cfg: dict) -> None:
    guardado = leer_json_privado(ESTADO)
    exigir("enviar" not in guardado, "la comisión ya fue enviada")
    original = guardado["item"]
    ref = original["comision"]["referencia"]
    version = original["recibo"]["version"] + 1
    if "documento_intent" not in guardado:
        base = guardado["solicitud"]
        guardado["documento_intent"] = {
            "clave_idempotencia": secrets.token_urlsafe(24),
            "version_esperada": original["recibo"]["version"],
            "relacion_ref": relacion(), "fecha_inicio": base["fecha_inicio"],
            "fecha_fin": base["fecha_fin"], "hora_inicio": base["hora_inicio"],
            "hora_fin": base["hora_fin"], "motivo": base["motivo"],
            "codigos_ruta": base["codigos_ruta"], "otros": [],
        }
        guardar(guardado)  # La intención sobrevive a un resultado incierto.
    item = verificar_operacion(cfg, "PUT", RUTA + "/" + quote(ref),
                              guardado["documento_intent"], version, "borrador")
    exigir(isinstance(item["comision"].get("documento"), dict),
           "v2 carece de documento orientativo")
    sql = pg_estado(cfg, ref)
    exigir(sql["versiones"] == [2] and sql["recibos"] == sql["historia"] == 1,
           "v2 no tiene una revisión, recibo e historia únicos")
    guardado["documento"] = item
    guardado["ultimo_item"] = item
    guardado["sql"] = sql
    guardar(guardado)
    print("DOCUMENTO: PUT/GET/replay v2, documento orientativo y una revisión/recibo/historia PG18")


def enviar(cfg: dict) -> None:
    guardado = leer_json_privado(ESTADO)
    exigir("documento" in guardado, "falta fase documento v2")
    previo = guardado["documento"]
    ref = previo["comision"]["referencia"]
    version = previo["recibo"]["version"] + 1
    if "enviar_intent" not in guardado:
        guardado["enviar_intent"] = {
            "clave_idempotencia": secrets.token_urlsafe(24),
            "version_esperada": previo["recibo"]["version"],
            "relacion_ref": relacion(),
        }
        guardar(guardado)
    item = verificar_operacion(cfg, "POST", RUTA + "/" + quote(ref) + "/enviar",
                              guardado["enviar_intent"], version,
                              "enviado_pendiente_revision")
    sql = pg_estado(cfg, ref)
    exigir(sql["versiones"] == [2, 3] and sql["recibos"] == sql["historia"] == 2 and
           sql["outbox"] >= 1,
           "envío v3 no tiene revisiones/recibos/historia/outbox esperados")
    guardado["enviar"] = item
    guardado["ultimo_item"] = item
    guardado["sql"] = sql
    guardar(guardado)
    print("ENVIAR: POST/GET/replay v3, asignación Personal, recibo e historia/outbox PG18")


def despues(cfg: dict) -> None:
    guardado = leer_json_privado(ESTADO)
    exigir("documento_intent" not in guardado or "documento" in guardado,
           "documento quedó incierto; repetir --fase documento con la misma clave")
    exigir("enviar_intent" not in guardado or "enviar" in guardado,
           "envío quedó incierto; repetir --fase enviar con la misma clave")
    exigir(guardado.get("origin") == cfg["origin"] and guardado.get("database") == cfg["database"],
           "resultado pertenece a otra instancia")
    identidad = pg_identidad(cfg)
    exigir(identidad["inicio"] != guardado.get("pg_start_before"),
           "PostgreSQL no acredita reinicio desde la fase antes")
    exigir(app_identidad(cfg) != guardado.get("app_before"),
           "Go no acredita reinicio desde la fase antes")
    exigir(comprobar_asignacion(cfg) == guardado["asignacion"],
           "asignación Personal cambió tras reinicio")
    original = guardado["item"]
    ultimo = guardado["ultimo_item"]
    ref = original["comision"]["referencia"]
    estado, detalle = pedir(cfg, "GET", ruta_detalle(ref))
    exigir(estado == 200 and resumen_immutable(detalle) == resumen_immutable(ultimo),
           "GET tras reinicio cambió comisión o recibo")
    estado, replay = pedir(cfg, "POST", RUTA, guardado["solicitud"])
    exigir(estado == 200 and resumen_immutable(replay) == resumen_immutable(original) and
           replay["recibo"].get("repeticion") is True,
           "replay del alta tras reinicio no conserva recibo")
    for fase, metodo, sufijo in (("documento", "PUT", ""), ("enviar", "POST", "/enviar")):
        if fase in guardado:
            estado_op, item = pedir(cfg, metodo, RUTA + "/" + quote(ref) + sufijo,
                                    guardado[fase + "_intent"])
            exigir(estado_op == 200 and
                   resumen_immutable(item) == resumen_immutable(guardado[fase]) and
                   item["recibo"].get("repeticion") is True,
                   f"replay {fase} tras reinicio cambió recibo o historia")
    sql = pg_estado(cfg, ref)
    exigir(sql == guardado["sql"], "replay o reinicio duplicó filas/historia")
    print("DESPUÉS: PG18 y Go reiniciados, asignación estable, GET/replays 200, mismos recibos e historia, sin duplicados")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fase", required=True,
                        choices=("preflight", "papeles", "antes", "documento", "enviar", "despues"))
    args = parser.parse_args()
    cfg = configurar()
    psql_disponible()
    if args.fase == "preflight":
        material_tls()
        material_tls(ct=True)
        relacion()
        pg_identidad(cfg)
        print("PRE-FLIGHT: marcador, ficheros TLS y PG18 aislado confirmados; identidades, Go/API/navegador pendientes")
    elif args.fase == "papeles":
        pg_identidad(cfg)
        app_identidad(cfg)
        comprobar_papeles(cfg)
        print("PAPELES: cuatro bandejas accesibles por concesión nominal; empleado denegado en revisión")
    elif args.fase == "antes":
        antes(cfg)
    elif args.fase == "documento":
        documento(cfg)
    elif args.fase == "enviar":
        enviar(cfg)
    else:
        despues(cfg)


if __name__ == "__main__":
    try:
        main()
    except (EnsayoError, PlaywrightError, OSError, ValueError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        print(f"DIETAS ENSAYO: {error}", file=sys.stderr)
        sys.exit(1)
