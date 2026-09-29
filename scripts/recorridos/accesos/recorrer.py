#!/usr/bin/env python3
"""Recorrido de autorizaciones con Chrome y cinco identidades sintéticas.

El plan, las claves y las respuestas de VEC permanecen fuera del repositorio.
"""

from __future__ import annotations

import argparse
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


class PlanInvalido(ValueError):
    pass


def _fichero_privado(valor: object, nombre: str, *, clave: bool = False) -> Path:
    if not isinstance(valor, str) or not valor:
        raise PlanInvalido(f"falta {nombre}")
    ruta = Path(valor)
    if not ruta.is_absolute() or not ruta.is_file() or ruta.is_symlink():
        raise PlanInvalido(f"{nombre} debe ser un fichero regular externo absoluto")
    if ruta.resolve().is_relative_to(RAIZ):
        raise PlanInvalido(f"{nombre} no puede estar dentro del repositorio")
    if clave and stat.S_IMODE(ruta.stat().st_mode) & 0o077:
        raise PlanInvalido(f"{nombre} debe ser privado (modo 0600 o más estricto)")
    return ruta


def validar_plan(ruta_plan: Path) -> dict:
    _fichero_privado(str(ruta_plan), "plan", clave=True)
    try:
        plan = json.loads(ruta_plan.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise PlanInvalido("plan JSON inválido") from exc
    if not isinstance(plan, dict):
        raise PlanInvalido("plan JSON inválido")
    origen = plan.get("origen")
    u = urlsplit(origen) if isinstance(origen, str) else None
    try:
        puerto = u.port if u else None
    except ValueError as exc:
        raise PlanInvalido("puerto local inválido") from exc
    if (u is None or u.scheme != "https" or u.hostname not in {"127.0.0.1", "localhost", "::1"}
            or u.username or u.password or u.path not in {"", "/"} or u.query or u.fragment or not puerto):
        raise PlanInvalido("origen debe ser HTTPS local con puerto, sin ruta ni credenciales")
    if not CHROME.is_file():
        raise PlanInvalido("falta /usr/bin/google-chrome")
    evidencia = plan.get("evidencia")
    if not isinstance(evidencia, dict) or evidencia.get("clon_h3_h5") is not True or evidencia.get("binario_en_uso") is not True:
        raise PlanInvalido("falta acreditación privada del clon H3–H5 y binario en uso")
    binario = _fichero_privado(evidencia.get("binario"), "binario")
    huella = evidencia.get("sha256_binario")
    if not isinstance(huella, str) or len(huella) != 64 or not all(c in "0123456789abcdef" for c in huella):
        raise PlanInvalido("falta SHA256 del binario")
    digest = hashlib.sha256()
    with binario.open("rb") as entrada:
        for bloque in iter(lambda: entrada.read(1024 * 1024), b""):
            digest.update(bloque)
    if digest.hexdigest() != huella:
        raise PlanInvalido("SHA256 del binario no coincide")
    pid = evidencia.get("pid")
    if not isinstance(pid, int) or isinstance(pid, bool) or pid <= 0:
        raise PlanInvalido("falta PID del binario local en uso")
    try:
        if not os.path.samefile(f"/proc/{pid}/exe", binario):
            raise PlanInvalido("PID no ejecuta el binario declarado")
    except OSError as exc:
        raise PlanInvalido("PID/binario local no verificable") from exc
    perfiles = plan.get("perfiles")
    if not isinstance(perfiles, dict) or set(perfiles) != set(PERFILES):
        raise PlanInvalido("el plan exige los cinco perfiles exactos")
    huellas_certificados = set()
    for nombre in PERFILES:
        p = perfiles[nombre]
        if not isinstance(p, dict):
            raise PlanInvalido(f"perfil {nombre} inválido")
        certificado = _fichero_privado(p.get("certificado"), f"certificado de {nombre}")
        huella_certificado = hashlib.sha256(certificado.read_bytes()).digest()
        if huella_certificado in huellas_certificados:
            raise PlanInvalido("los cinco perfiles deben usar certificados distintos")
        huellas_certificados.add(huella_certificado)
        _fichero_privado(p.get("clave"), f"clave de {nombre}", clave=True)
        pruebas = p.get("pruebas")
        if not isinstance(pruebas, list) or len(pruebas) < 2:
            raise PlanInvalido(f"{nombre} requiere permiso positivo y denegación")
        clases = set()
        for i, prueba in enumerate(pruebas, 1):
            if not isinstance(prueba, dict):
                raise PlanInvalido(f"prueba {i} de {nombre} inválida")
            clase, ruta, metodo, esperado = (prueba.get(k) for k in ("clase", "ruta", "metodo", "estado"))
            if clase not in {"permiso", "denegacion"} or metodo not in {"GET", "POST"}:
                raise PlanInvalido(f"prueba {i} de {nombre}: clase o método inválido")
            if not isinstance(ruta, str) or not ruta.startswith("/api/vec/") or "//" in ruta or ".." in ruta or "?" in ruta or "#" in ruta:
                raise PlanInvalido(f"prueba {i} de {nombre}: ruta insegura")
            if metodo == "POST" and ruta != "/api/vec/contratacion-temporal/cuadro/consultas":
                raise PlanInvalido("POST solo admite la consulta sin escritura del cuadro CT")
            if (clase == "permiso" and esperado != 200) or (clase == "denegacion" and esperado not in {401, 403}):
                raise PlanInvalido(f"prueba {i} de {nombre}: estado esperado inválido")
            if metodo == "POST" and prueba.get("cuerpo") != {"filtros": {"texto": ""}, "paginacion": {"limite": 10}}:
                raise PlanInvalido("la consulta del cuadro exige el cuerpo fijo sintético")
            if clase == "permiso":
                comprobacion = prueba.get("comprobacion")
                if (not isinstance(comprobacion, dict) or not isinstance(comprobacion.get("campo"), str)
                        or not re.fullmatch(r"data(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*", comprobacion["campo"])
                        or ("igual" in comprobacion) == ("tipo" in comprobacion)
                        or comprobacion.get("tipo", "objeto") not in {"objeto", "lista", "cadena", "booleano", "numero"}):
                    raise PlanInvalido(f"prueba {i} de {nombre}: falta comprobación de contenido positivo")
            clases.add(clase)
        if clases != {"permiso", "denegacion"}:
            raise PlanInvalido(f"{nombre} requiere ambas clases de prueba")
    return plan


def _sonda(pagina, prueba: dict) -> dict:
    # Evalúa el contenido dentro de Chrome: no devuelve datos personales a Python.
    return pagina.evaluate("""async p => {
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


def recorrer(plan: dict) -> int:
    from playwright.sync_api import sync_playwright

    origen = plan["origen"].rstrip("/")
    with sync_playwright() as pw:
        navegador = pw.chromium.launch(executable_path=str(CHROME), headless=True)
        try:
            for nombre in PERFILES:
                perfil = plan["perfiles"][nombre]
                contexto = navegador.new_context(
                    client_certificates=[{"origin": origen, "certPath": perfil["certificado"], "keyPath": perfil["clave"]}],
                    ignore_https_errors=False,
                    viewport={"width": 1440, "height": 900},
                    service_workers="block",
                )
                try:
                    contexto.route("**/*", lambda route: route.continue_() if
                                   urlsplit(route.request.url).netloc == urlsplit(origen).netloc
                                   and urlsplit(route.request.url).scheme == "https" else route.abort())
                    # Chrome debe confiar en la CA del origen mediante el almacén
                    # configurado por el operador. No se omite la validación TLS.
                    pagina = contexto.new_page()
                    respuesta = pagina.goto(origen + "/portal-empleado/" if nombre != "candidato_area" else origen + "/area-personal/", wait_until="domcontentloaded", timeout=15000)
                    if (respuesta is None or respuesta.status != 200
                            or urlsplit(respuesta.url).netloc != urlsplit(origen).netloc):
                        print(f"PRIMER CORTE: {nombre} / entrada HTTP inesperada")
                        return 1
                    for indice, prueba in enumerate(perfil["pruebas"], 1):
                        observado = _sonda(pagina, prueba)
                        correcto = (observado["estado"] == prueba["estado"] and observado["json"]
                                    and (observado["comprobado"] if prueba["clase"] == "permiso" else observado["sinDatos"]))
                        print(f"{nombre} {indice} {prueba['clase']}: {'OK' if correcto else 'FALLO'} HTTP {observado['estado']}")
                        if not correcto:
                            print(f"PRIMER CORTE: {nombre} prueba {indice} ({prueba['clase']}); sin respuesta ni datos en la salida")
                            return 1
                    if contexto.cookies():
                        print(f"PRIMER CORTE: {nombre} emitió cookies")
                        return 1
                    almacenamiento = pagina.evaluate("() => localStorage.length + sessionStorage.length")
                    if almacenamiento:
                        print(f"PRIMER CORTE: {nombre} usó almacenamiento web")
                        return 1
                finally:
                    contexto.close()
        finally:
            navegador.close()
    print("CORTE OBSERVADO: cinco perfiles, permisos y denegaciones configurados; sin operaciones de escritura")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Recorrido local de accesos VEC con certificados por perfil")
    parser.add_argument("--plan", type=Path, required=True, help="JSON privado externo al repositorio")
    parser.add_argument("--ejecutar", action="store_true", help="habilita el navegador contra el origen local")
    args = parser.parse_args(argv)
    try:
        plan = validar_plan(args.plan)
    except PlanInvalido as exc:
        print(f"NO EJECUTADO: {exc}", file=sys.stderr)
        return 2
    if not args.ejecutar:
        print("NO EJECUTADO: plan válido; falta --ejecutar para abrir Chrome")
        return 2
    try:
        return recorrer(plan)
    except Exception as exc:
        # Evita imprimir URLs, rutas privadas o cuerpos de respuesta del fallo.
        print(f"PRIMER CORTE: navegador o conexión ({type(exc).__name__}); consulte el entorno privado")
        return 1


if __name__ == "__main__":
    sys.exit(main())
