#!/usr/bin/env python3
"""Recorrido de autorizaciones con Chrome y cinco identidades sintéticas.

El plan, las claves y las respuestas de VEC permanecen fuera del repositorio.
"""

from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
import os
import re
import stat
import sys
from pathlib import Path
from urllib.parse import urlsplit


PERFILES = ("rrhh", "centro", "ratificador", "intervencion", "candidato_area")
CHROME = Path("/usr/bin/google-chrome")
RAIZ = Path(__file__).resolve().parents[3]
IDIOMA = "en" if os.environ.get("LANG", "").startswith("en") else "es"
MENSAJES = json.loads((Path(__file__).parent / f"mensajes.{IDIOMA}.json").read_text(encoding="utf-8"))


def mensaje(clave: str, **datos: object) -> str:
    return MENSAJES[clave].format(**datos)


class PlanInvalido(ValueError):
    def __init__(self, clave: str, **datos: object):
        super().__init__(mensaje(clave, **datos))


def _dentro_de_git(ruta: Path) -> bool:
    resuelta = ruta.resolve()
    return any((directorio / ".git").exists() or (directorio / ".git").is_symlink()
               for directorio in resuelta.parents)


def _fichero_privado(valor: object, nombre: str, *, clave: bool = False) -> Path:
    if not isinstance(valor, str) or not valor:
        raise PlanInvalido("falta_fichero", nombre=nombre)
    ruta = Path(valor)
    if not ruta.is_absolute() or not ruta.is_file() or ruta.is_symlink():
        raise PlanInvalido("fichero_externo", nombre=nombre)
    if ruta.resolve().is_relative_to(RAIZ) or _dentro_de_git(ruta):
        raise PlanInvalido("fuera_repositorio", nombre=nombre)
    if clave and stat.S_IMODE(ruta.stat().st_mode) & 0o077:
        raise PlanInvalido("fichero_privado", nombre=nombre)
    return ruta


def validar_plan(ruta_plan: Path) -> dict:
    _fichero_privado(str(ruta_plan), mensaje("nombre_plan"), clave=True)
    try:
        plan = json.loads(ruta_plan.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise PlanInvalido("plan_invalido") from exc
    if not isinstance(plan, dict):
        raise PlanInvalido("plan_invalido")
    origen = plan.get("origen")
    u = urlsplit(origen) if isinstance(origen, str) else None
    try:
        puerto = u.port if u else None
    except ValueError as exc:
        raise PlanInvalido("puerto_invalido") from exc
    if (u is None or u.scheme != "https" or u.hostname not in {"127.0.0.1", "localhost", "::1"}
            or u.username or u.password or u.path not in {"", "/"} or u.query or u.fragment or not puerto):
        raise PlanInvalido("origen_invalido")
    if not CHROME.is_file():
        raise PlanInvalido("chrome_ausente")
    evidencia = plan.get("evidencia")
    if not isinstance(evidencia, dict) or evidencia.get("clon_h3_h5") is not True or evidencia.get("binario_en_uso") is not True:
        raise PlanInvalido("clon_ausente")
    binario = _fichero_privado(evidencia.get("binario"), mensaje("nombre_binario"))
    huella = evidencia.get("sha256_binario")
    if not isinstance(huella, str) or len(huella) != 64 or not all(c in "0123456789abcdef" for c in huella):
        raise PlanInvalido("sha_ausente")
    digest = hashlib.sha256()
    with binario.open("rb") as entrada:
        for bloque in iter(lambda: entrada.read(1024 * 1024), b""):
            digest.update(bloque)
    if digest.hexdigest() != huella:
        raise PlanInvalido("sha_distinto")
    pid = evidencia.get("pid")
    if not isinstance(pid, int) or isinstance(pid, bool) or pid <= 0:
        raise PlanInvalido("pid_ausente")
    try:
        if not os.path.samefile(f"/proc/{pid}/exe", binario):
            raise PlanInvalido("pid_distinto")
    except OSError as exc:
        raise PlanInvalido("pid_no_verificable") from exc
    perfiles = plan.get("perfiles")
    if not isinstance(perfiles, dict) or set(perfiles) != set(PERFILES):
        raise PlanInvalido("perfiles_exactos")
    huellas_certificados = set()
    for nombre in PERFILES:
        p = perfiles[nombre]
        if not isinstance(p, dict):
            raise PlanInvalido("perfil_invalido", nombre=nombre)
        certificado = _fichero_privado(p.get("certificado"), mensaje("nombre_certificado", nombre=nombre))
        huella_certificado = hashlib.sha256(certificado.read_bytes()).digest()
        if huella_certificado in huellas_certificados:
            raise PlanInvalido("certificados_distintos")
        huellas_certificados.add(huella_certificado)
        _fichero_privado(p.get("clave"), mensaje("nombre_clave", nombre=nombre), clave=True)
        pruebas = p.get("pruebas")
        if not isinstance(pruebas, list) or len(pruebas) < 2:
            raise PlanInvalido("ambas_pruebas", nombre=nombre)
        clases = set()
        for i, prueba in enumerate(pruebas, 1):
            if not isinstance(prueba, dict):
                raise PlanInvalido("prueba_invalida", indice=i, nombre=nombre)
            clase, ruta, metodo, esperado = (prueba.get(k) for k in ("clase", "ruta", "metodo", "estado"))
            if clase not in {"permiso", "denegacion"} or metodo not in {"GET", "POST"}:
                raise PlanInvalido("clase_metodo", indice=i, nombre=nombre)
            if not isinstance(ruta, str) or not ruta.startswith("/api/vec/") or "//" in ruta or ".." in ruta or "?" in ruta or "#" in ruta:
                raise PlanInvalido("ruta_insegura", indice=i, nombre=nombre)
            if metodo == "POST" and ruta != "/api/vec/contratacion-temporal/cuadro/consultas":
                raise PlanInvalido("post_lectura")
            if (clase == "permiso" and esperado != 200) or (clase == "denegacion" and esperado not in {401, 403}):
                raise PlanInvalido("estado_invalido", indice=i, nombre=nombre)
            if metodo == "POST" and prueba.get("cuerpo") != {"filtros": {"texto": ""}, "paginacion": {"limite": 10}}:
                raise PlanInvalido("cuerpo_fijo")
            if clase == "permiso":
                comprobacion = prueba.get("comprobacion")
                if (not isinstance(comprobacion, dict) or not isinstance(comprobacion.get("campo"), str)
                        or not re.fullmatch(r"data(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*", comprobacion["campo"])
                        or ("igual" in comprobacion) == ("tipo" in comprobacion)
                        or comprobacion.get("tipo", "objeto") not in {"objeto", "lista", "cadena", "booleano", "numero"}):
                    raise PlanInvalido("comprobacion_positiva", indice=i, nombre=nombre)
            clases.add(clase)
        if clases != {"permiso", "denegacion"}:
            raise PlanInvalido("ambas_pruebas", nombre=nombre)
    return plan


async def _sonda(pagina, prueba: dict) -> dict:
    # Evalúa el contenido dentro de Chrome: no devuelve datos personales a Python.
    return await pagina.evaluate("""async p => {
      const opciones = {method:p.metodo, cache:'no-store', credentials:'same-origin',
        redirect:'error', headers:{Accept:'application/json'}};
      if (p.metodo === 'POST') {
        opciones.headers['Content-Type']='application/json';
        opciones.body=JSON.stringify(p.cuerpo);
      }
      const r=await fetch(p.ruta, opciones);
      let d=null;
      try { d=await r.json(); } catch (_) { return {estado:r.status, json:false, comprobado:false, sinDatos:false}; }
      const partes=p.comprobacion?.campo?.split('.') || [];
      const obtenido=partes.reduce((v,k)=>v && typeof v==='object' ? v[k] : undefined,d);
      const tipos={objeto:v=>v!==null && typeof v==='object' && !Array.isArray(v),
        lista:Array.isArray, cadena:v=>typeof v==='string',
        booleano:v=>typeof v==='boolean', numero:v=>typeof v==='number'};
      const comprobado=partes.length>0 && (Object.prototype.hasOwnProperty.call(p.comprobacion,'igual')
        ? JSON.stringify(obtenido)===JSON.stringify(p.comprobacion.igual)
        : Boolean(tipos[p.comprobacion.tipo]?.(obtenido)));
      return {estado:r.status, json:true, comprobado,
        sinDatos:!d || !Object.prototype.hasOwnProperty.call(d,'data') || d.data===null};
    }""", prueba)


async def _interceptar_local(route, origen: str) -> None:
    """Resuelve una sola respuesta y corta cualquier salto de origen o redirección."""
    esperado = urlsplit(origen)
    solicitado = urlsplit(route.request.url)
    if (solicitado.scheme, solicitado.netloc) != (esperado.scheme, esperado.netloc):
        await route.abort()
        return
    try:
        respuesta = await route.fetch(max_redirects=0)
        final = urlsplit(respuesta.url)
        if (300 <= respuesta.status < 400
                or (final.scheme, final.netloc) != (esperado.scheme, esperado.netloc)):
            await route.abort()
            return
        await route.fulfill(response=respuesta)
    except Exception:
        await route.abort()


async def _cerrar_websocket(route) -> None:
    """El recorrido de lectura no necesita canales WebSocket."""
    await route.close()


async def _recorrer(plan: dict) -> int:
    from playwright.async_api import async_playwright

    origen = plan["origen"].rstrip("/")
    async with async_playwright() as pw:
        navegador = await pw.chromium.launch(executable_path=str(CHROME), headless=True)
        try:
            for nombre in PERFILES:
                perfil = plan["perfiles"][nombre]
                contexto = await navegador.new_context(
                    client_certificates=[{"origin": origen, "certPath": perfil["certificado"], "keyPath": perfil["clave"]}],
                    ignore_https_errors=False,
                    viewport={"width": 1440, "height": 900},
                    service_workers="block",
                )
                try:
                    await contexto.route("**/*", lambda route: _interceptar_local(route, origen))
                    await contexto.route_web_socket("**/*", _cerrar_websocket)
                    # Chrome debe confiar en la CA del origen mediante el almacén
                    # configurado por el operador. No se omite la validación TLS.
                    pagina = await contexto.new_page()
                    respuesta = await pagina.goto(origen + "/portal-empleado/" if nombre != "candidato_area" else origen + "/area-personal/", wait_until="domcontentloaded", timeout=15000)
                    if (respuesta is None or respuesta.status != 200
                            or urlsplit(respuesta.url).netloc != urlsplit(origen).netloc):
                        print(mensaje("corte_entrada", nombre=nombre))
                        return 1
                    for indice, prueba in enumerate(perfil["pruebas"], 1):
                        observado = await _sonda(pagina, prueba)
                        correcto = (observado["estado"] == prueba["estado"] and observado["json"]
                                    and (observado["comprobado"] if prueba["clase"] == "permiso" else observado["sinDatos"]))
                        print(mensaje("resultado", nombre=nombre, indice=indice, clase=prueba["clase"],
                                      resultado=mensaje("ok" if correcto else "fallo"), estado=observado["estado"]))
                        if not correcto:
                            print(mensaje("corte_prueba", nombre=nombre, indice=indice, clase=prueba["clase"]))
                            return 1
                    if await contexto.cookies():
                        print(mensaje("corte_cookies", nombre=nombre))
                        return 1
                    almacenamiento = await pagina.evaluate("() => localStorage.length + sessionStorage.length")
                    if almacenamiento:
                        print(mensaje("corte_almacenamiento", nombre=nombre))
                        return 1
                finally:
                    await contexto.close()
        finally:
            await navegador.close()
    print(mensaje("corte_observado"))
    return 0


def recorrer(plan: dict) -> int:
    return asyncio.run(_recorrer(plan))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=mensaje("descripcion"))
    parser.add_argument("--plan", type=Path, required=True, help=mensaje("ayuda_plan"))
    parser.add_argument("--ejecutar", action="store_true", help=mensaje("ayuda_ejecutar"))
    args = parser.parse_args(argv)
    try:
        plan = validar_plan(args.plan)
    except PlanInvalido as exc:
        print(mensaje("no_ejecutado", detalle=exc), file=sys.stderr)
        return 2
    if not args.ejecutar:
        print(mensaje("sin_optin"))
        return 2
    try:
        return recorrer(plan)
    except Exception as exc:
        # Evita imprimir URLs, rutas privadas o cuerpos de respuesta del fallo.
        print(mensaje("corte_navegador", tipo=type(exc).__name__))
        return 1


if __name__ == "__main__":
    sys.exit(main())
