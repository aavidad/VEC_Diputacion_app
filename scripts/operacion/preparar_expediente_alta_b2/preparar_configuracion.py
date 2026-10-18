#!/usr/bin/env python3
"""Comandos privados y comprobación estática del lanzador B2."""
from __future__ import annotations

import json
import os
import re
import shlex
import stat
from pathlib import Path
from urllib.parse import parse_qsl, unquote, urlsplit


def comando_psql(args):
    local = ["podman", "--remote=false"] if args.motor == "podman" else ["docker", "--host", "unix:///var/run/docker.sock"]
    return local + ["exec", "-i", args.contenedor, "psql", "-X", "-A", "-t", "-v", "ON_ERROR_STOP=1",
                    "-h", "/var/run/postgresql", "-U", "postgres", "-d", "postgres"]


def entorno_contenedor():
    env = os.environ.copy()
    for key in ("DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH"):
        env.pop(key, None)
    return env


def ruta_datos_ejercicio(args):
    return Path(os.environ.get("VEC_B2_NECESIDAD_EJEMPLO", str(args.config.parent.parent / "necesidad_ejemplo_b2.json")))


def comando_datos_ejercicio(args):
    ruta = ruta_datos_ejercicio(args)
    contenido = {"plaza_codigo": "0033", "organica_codigo": "520", "funcional_codigo": "920",
                 "proyecto_gasto_codigo": "2026-520", "porcentaje_financiacion": "100"}
    q = shlex.quote
    return (f"set -eu; umask 077; mkdir -p -m 700 {q(str(ruta.parent))}; test ! -e {q(str(ruta))}; "
            f"cat > {q(str(ruta))} <<'JSON'\n{json.dumps(contenido, ensure_ascii=False, indent=2)}\nJSON\n"
            f"chmod 600 {q(str(ruta))}")


def leer_conexiones_privadas(ruta):
    try:
        fd = os.open(ruta, os.O_RDONLY | os.O_NOFOLLOW)
        with os.fdopen(fd, "r", encoding="utf-8") as origen:
            meta = os.fstat(origen.fileno())
            if (not stat.S_ISREG(meta.st_mode) or stat.S_IMODE(meta.st_mode) != 0o600 or
                    meta.st_nlink != 1 or meta.st_size > 1024 * 1024):
                raise ValueError("FALTA: --conexiones exige fichero privado regular 0600")
            return origen.read()
    except (OSError, UnicodeError) as exc:
        raise ValueError("FALTA: --conexiones no se puede leer con seguridad") from exc


def comprobar_relevo_cese(launcher, conexiones=""):
    fuentes = launcher + "\n" + conexiones

    def valor(nombre):
        patron = rf"(?m)^[ \t]*(?:export[ \t]+)?{nombre}[ \t]*=[ \t]*(.*)$"
        halladas = list(re.finditer(patron, fuentes))
        if len(halladas) > 1:
            raise ValueError(f"FALTA: {nombre} tiene varias asignaciones privadas")
        if not halladas:
            return ""
        bruto = halladas[0].group(1).strip()
        forma = re.fullmatch(r"(?:'([^']*)'|\"([^\"$`\\]*)\"|([A-Za-z0-9:/?%=._~+@,!-]+))(?:[ \t]+#.*)?", bruto)
        literal = next((x for x in forma.groups() if x is not None), "") if forma else ""
        if not literal or any(c in literal for c in ("$", "`", "\\", "\n", "\r")):
            raise ValueError(f"FALTA: {nombre} no tiene valor literal verificable")
        return literal

    for selector in ("VEC_CT_SEGUIMIENTO_CESE_ENABLED", "VEC_BOLSA_CESE_CT_ENABLED"):
        if valor(selector) != "true":
            raise ValueError(f"FALTA: {selector}=true en lanzador activo")
    intervalo = valor("VEC_BOLSA_CONTRATOS_CT_INTERVALO")
    if intervalo:
        patron = r"(?:[0-9]+(?:\.[0-9]+)?(?:ns|us|µs|ms|s|m|h))+"
        if not re.fullmatch(patron, intervalo):
            raise ValueError("FALTA: intervalo del relevo de cese no verificable")
        factores = {"ns": 1e-9, "us": 1e-6, "µs": 1e-6, "ms": 1e-3, "s": 1, "m": 60, "h": 3600}
        trozos = re.findall(r"([0-9]+(?:\.[0-9]+)?)(ns|us|µs|ms|s|m|h)", intervalo)
        segundos = sum(float(numero) * factores[unidad] for numero, unidad in trozos)
        if not 1 <= segundos <= 3600:
            raise ValueError("FALTA: intervalo del relevo de cese debe estar entre 1s y 1h")
    dsn = valor("VEC_BOLSA_CESE_CT_DATABASE_URL")
    if not dsn:
        raise ValueError("FALTA: DSN nominal VEC_BOLSA_CESE_CT_DATABASE_URL en lanzador activo")
    if dsn.startswith(("postgres://", "postgresql://")):
        try:
            partes = urlsplit(dsn)
            parametros = parse_qsl(partes.query, keep_blank_values=True)
            modos = [v for k, v in parametros if k == "sslmode"]
            seguro = (bool(partes.hostname) and "/" not in unquote(partes.hostname) and
                      partes.port is not None and 1 <= partes.port <= 65535 and
                      modos == ["verify-full"] and
                      not any(k in ("host", "hostaddr", "service") for k, _ in parametros))
        except ValueError:
            seguro = False
    else:
        partes = shlex.split(dsn)
        pares = [parte.split("=", 1) for parte in partes]
        claves = [par[0] for par in pares if len(par) == 2]
        valores = dict(par for par in pares if len(par) == 2)
        puerto = valores.get("port", "")
        host = valores.get("host", "")
        seguro = (len(pares) == len(claves) == len(set(claves)) and
                  valores.get("sslmode") == "verify-full" and bool(host) and
                  "/" not in host and "hostaddr" not in valores and "service" not in valores and
                  puerto.isdigit() and 1 <= int(puerto) <= 65535)
    if not seguro:
        raise ValueError("FALTA: DSN de cese sin TLS verify-full verificable")


def comando_catalogo_faltante(args, referencia, cert, key, ejemplos, base):
    archivos = {
        "regimen:funcionario_carrera": "01-funcionario_carrera.json",
        "regimen:funcionario_interino": "02-funcionario_interino.json",
        "regimen:laboral_fijo": "03-laboral_fijo.json",
        "regimen:laboral_temporal": "04-laboral_temporal.json",
        "modalidad:vacante": "05-vacante.json",
        "modalidad:sustitucion": "06-sustitucion.json",
        "modalidad:programa": "07-programa.json",
        "modalidad:acumulacion_tareas": "08-acumulacion_tareas.json",
    }
    privado = args.estado.parent / ("catalogo-" + referencia.replace(":", "-") + ".uuid")
    cuerpo = ejemplos / "catalogos-personal" / archivos[referencia]
    endpoint = args.origen + f"{base}/incorporacion-personal-b2/catalogos-registro-empleado/v1"
    q = shlex.quote
    crear_clave = ("import os,sys,uuid; p=sys.argv[1]; "
                   "fd=os.open(p,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600); "
                   "f=os.fdopen(fd,'w'); f.write(str(uuid.uuid4())+'\\n'); "
                   "f.flush(); os.fsync(f.fileno()); f.close()")
    return (f"set -eu\numask 077\nmkdir -p -m 700 {q(str(args.estado.parent))}\n"
            f"test -s {q(str(privado))} || python3 -c {q(crear_clave)} {q(str(privado))}\n"
            f"curl -q --noproxy '*' --silent --show-error --max-time 20 --max-redirs 0 --proto '=https' "
            f"--cacert {q(str(args.material / 'ca/ca.crt'))} --cert {q(str(cert))} --key {q(str(key))} "
            "-H 'Accept: application/json' -H 'Content-Type: application/json' "
            f"-H \"Idempotency-Key: $(cat {q(str(privado))})\" "
            f"--data-binary {q('@' + str(cuerpo))} {q(endpoint)}")
