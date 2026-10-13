#!/usr/bin/env python3
"""Prepara un único expediente sintético B2 por las APIs nominales de VEC."""
from __future__ import annotations

import argparse
import fcntl
import hashlib
import json
import os
import re
import shlex
import subprocess
import sys
import tempfile
import uuid
from datetime import date, datetime, timedelta, timezone
from pathlib import Path
from urllib.parse import urlsplit, quote

ROOT = Path(__file__).resolve().parents[3]
EJEMPLOS = ROOT / "docs/evidencias/codexh-b2-recorrido-20261011/comandos"
BASE = "/api/vec/contratacion-temporal"
RUTAS = {
    "presentar": ("centro-solicitante", f"{BASE}/peticiones-centro/operaciones"),
    "ratificar": ("centro-ratificador", f"{BASE}/peticiones-centro/operaciones"),
    "alta": ("RRHH", f"{BASE}/peticiones-centro/rrhh"),
    "analisis": ("RRHH", f"{BASE}/analisis/registros"),
    "cobertura": ("RRHH", f"{BASE}/cobertura/decisiones"),
    "asignacion": ("RRHH", f"{BASE}/asignaciones"),
    "informe": ("RRHH", f"{BASE}/informes-juridicos/preparaciones"),
    "fiscalizacion": ("intervencion", f"{BASE}/fiscalizaciones/resultados"),
    "vinculo_rpt": ("RRHH", f"{BASE}/incorporacion-personal-b2/vinculo-categoria-rpt/v1"),
    "seleccion": ("RRHH", f"{BASE}/llamamientos/seleccion"),
    "vinculo_bolsa": ("RRHH", f"{BASE}/expedientes/vinculos-bolsa"),
    "respuesta_portal": ("candidato", "/api/vec/bolsa/mi-bolsa/respuestas"),
    "comunicacion": ("RRHH", f"{BASE}/llamamientos/comunicaciones"),
    "respuesta": ("RRHH", f"{BASE}/llamamientos/respuestas/registro"),
    "aceptacion": ("RRHH", f"{BASE}/llamamientos/resoluciones"),
    "propuesta": ("RRHH", f"{BASE}/formalizacion/propuestas"),
    "resolucion": ("RRHH", f"{BASE}/resoluciones-formalizacion"),
}
CANON_GO = {
    "seleccion": ("expediente_ref", "version_esperada", "clave_idempotencia"),
    "vinculo_bolsa": ("expediente_ref", "version_esperada", "bolsa_ref",
                     "llamamiento_ref", "recibo_emision_ref", "clave_idempotencia"),
    "comunicacion": ("clave_idempotencia", "organizacion_ref", "expediente_ref", "llamamiento_ref",
                     "version_esperada", "prueba_entrega_ref", "tipo_antecedente"),
    "respuesta": ("clave_idempotencia", "organizacion_ref", "expediente_ref", "llamamiento_ref",
                  "comunicacion_ref", "version_comunicacion_esperada", "respuesta", "correo_ref",
                  "correo_sha256", "recibida_en"),
    "aceptacion": ("clave_idempotencia", "organizacion_ref", "expediente_ref", "llamamiento_ref",
                   "comunicacion_ref", "version_esperada", "respuesta", "prueba_respuesta_ref",
                   "revision_respuesta_rrhh", "revision_plazo_rrhh", "criterio_validacion_ref"),
    "propuesta": ("clave_idempotencia", "expediente_ref", "llamamiento_ref",
                  "resolucion_llamamiento_aceptada_ref", "recibo_resolucion_aceptada_ref",
                  "version_esperada", "tipo_formalizacion", "plantilla", "anexos",
                  "politica_firma", "plan_firma"),
}
PLAN = f"{BASE}/incorporacion-personal-b2/plan/v1"
RPT_PUBLICA = ROOT / "data/catalogos/rpt/v1.rpt-2026.json"


class Corte(Exception):
    pass


def exigir_origen(value):
    u = urlsplit(value)
    if (u.scheme != "https" or u.hostname not in ("127.0.0.1", "localhost", "::1")
            or not u.port or u.username or u.password or u.path not in ("", "/")
            or u.query or u.fragment):
        raise Corte("FALTA: --origen debe ser HTTPS loopback con puerto y sin ruta")
    return value.rstrip("/")


def guardar(path, value):
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    if path.is_symlink() or path.parent.is_symlink():
        raise Corte("FALTA: estado privado no admite enlaces simbólicos")
    fd, temp = tempfile.mkstemp(prefix=".vec-b2-", dir=path.parent)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as out:
            json.dump(value, out, ensure_ascii=False, sort_keys=True, indent=2)
            out.write("\n")
            out.flush()
            os.fsync(out.fileno())
        os.replace(temp, path)
        dfd = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(dfd)
        finally:
            os.close(dfd)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)


def cargar(path):
    if not path.exists():
        return {"esquema": "vec.preparar-alta-b2.v1", "operaciones": {}, "valores": {}}
    if path.is_symlink() or path.stat().st_mode & 0o077:
        raise Corte("FALTA: estado privado inseguro; exige fichero 0600 sin symlink")
    value = json.loads(path.read_text(encoding="utf-8"))
    if value.get("esquema") != "vec.preparar-alta-b2.v1":
        raise Corte("FALTA: formato de estado privado ajeno")
    return value


def mTLS(args, actor):
    directory = args.material / "mtls"
    aliases = {
        "RRHH": ["cliente", "cliente-RRHH", "cliente-rrhh", "rrhh"],
        "centro-solicitante": ["cliente-centro-solicitante", "centro-solicitante"],
        "centro-ratificador": ["cliente-centro-ratificador", "centro-ratificador"],
        "intervencion": ["cliente-intervencion", "intervencion"],
        "candidato": ["candidato"],
    }
    for base in aliases[actor]:
        cert, key = directory / (base + ".crt"), directory / (base + ".key")
        if cert.is_file() and key.is_file():
            return cert, key
    raise Corte(f"FALTA: certificado y clave mTLS de {actor} en {directory}")


def http(args, actor, method, route, body=None):
    if not route.startswith("/") or route.startswith("//"):
        raise Corte("FALTA: ruta HTTP inválida")
    cert, key = mTLS(args, actor)
    ca = args.material / "ca/ca.crt"
    if not ca.is_file():
        raise Corte(f"FALTA: CA privada {ca}")
    command = ["curl", "-q", "--noproxy", "*", "--silent", "--show-error", "--proto", "=https",
               "--max-redirs", "0", "--connect-timeout", "5", "--max-time", "25",
               "--cacert", str(ca), "--cert", str(cert), "--key", str(key),
               "-H", "Accept: application/json", "-H", "Cache-Control: no-store"]
    if method == "POST":
        command += ["-H", "Content-Type: application/json", "--request", "POST",
                    "--data-binary", "@-"]
    command += ["--write-out", "\n%{http_code}", args.origen + route]
    raw = serializar_cuerpo(body) if body is not None else None
    try:
        result = subprocess.run(command, input=raw, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, timeout=32, check=False)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise Corte(f"HTTP {method} {route}: sin respuesta local verificable; conservar la clave") from exc
    if result.returncode:
        raise Corte(f"HTTP {method} {route}: error de canal local; no se cambia la clave de la operación")
    try:
        payload, status = result.stdout.rsplit(b"\n", 1)
        code = int(status)
        data = json.loads(payload)
    except (ValueError, json.JSONDecodeError) as exc:
        raise Corte(f"HTTP {method} {route}: respuesta no verificable") from exc
    if 300 <= code < 400:
        raise Corte(f"HTTP {method} {route}: redirección rechazada")
    return code, data


def data_http(args, actor, method, route, body=None, allowed=(200,)):
    code, content = http(args, actor, method, route, body)
    if code not in allowed or not isinstance(content, dict) or not isinstance(content.get("data"), dict):
        error = content.get("error") if isinstance(content, dict) else None
        codigo = error.get("codigo", "respuesta_invalida") if isinstance(error, dict) else "respuesta_invalida"
        raise Corte(f"HTTP {method} {route}: {code} {codigo}; conservar estado privado y recuperar con la misma clave")
    return content["data"]


def serializar_cuerpo(body):
    return json.dumps(body, ensure_ascii=False, separators=(",", ":")).encode("utf-8")


def cuerpo_canonico(name, stored):
    """Reconstruye orden Go de transporte tras leer JSON privado ordenado por claves."""
    if name not in CANON_GO:
        return stored
    if set(stored) != set(CANON_GO[name]) - ({"tipo_antecedente"} if name == "comunicacion" else set()):
        raise Corte(f"FALTA: cuerpo privado {name} incompleto")
    body = {key: stored[key] for key in CANON_GO[name] if key in stored}
    if name == "propuesta":
        for key in ("tipo_formalizacion", "plantilla", "politica_firma", "plan_firma"):
            snap = body[key]
            if set(snap) != {"referencia", "version", "huella_sha256"}:
                raise Corte("FALTA: snapshot privado de propuesta incompleto")
            body[key] = {sub: snap[sub] for sub in ("referencia", "version", "huella_sha256")}
    return body


def efecto(args, state, name, make_body):
    actor, route = RUTAS[name]
    old = state["operaciones"].get(name)
    if old is None:
        body = make_body()
        if name == "respuesta_portal":
            body["clave"] = str(uuid.uuid4())
        elif name != "alta":
            body["clave_idempotencia"] = str(uuid.uuid4())
        if name in CANON_GO:
            if set(body) != set(CANON_GO[name]) - ({"tipo_antecedente"} if name == "comunicacion" else set()):
                raise Corte(f"FALTA: cuerpo canónico {name} incompleto")
            body = {key: body[key] for key in CANON_GO[name] if key in body}
        old = {"actor": actor, "ruta": route, "cuerpo": body, "estado": "pendiente"}
        state["operaciones"][name] = old
        guardar(args.estado, state)  # durable ANTES de emitir cualquier POST
    if old["actor"] != actor or old["ruta"] != route:
        raise Corte(f"FALTA: operación privada {name} incompatible")
    if old.get("estado") == "confirmada":
        return old["respuesta"]
    out = data_http(args, actor, "POST", route, cuerpo_canonico(name, old["cuerpo"]), allowed=(200, 201))
    old["respuesta"] = out
    old["estado"] = "confirmada"
    guardar(args.estado, state)
    return out


def ejemplo(number, name):
    value = json.loads((EJEMPLOS / f"{number:02d}-{name}.json").read_text(encoding="utf-8"))
    value.pop("clave_idempotencia", None)
    return value


def buscar(obj, key):
    if isinstance(obj, dict):
        if key in obj and obj[key] not in (None, ""):
            return obj[key]
        for value in obj.values():
            found = buscar(value, key)
            if found is not None:
                return found
    elif isinstance(obj, list):
        for value in obj:
            found = buscar(value, key)
            if found is not None:
                return found
    return None


def campo(obj, name):
    result = buscar(obj, name)
    if result is None:
        raise Corte(f"FALTA: el recibo/consulta no trae {name}; revisar JSON privado")
    return result


def lectura_db(args, query):
    base = comando_psql(args)
    try:
        process = subprocess.run(base, input=("BEGIN READ ONLY; SET LOCAL statement_timeout='10s'; " + query + "; ROLLBACK;\n").encode(),
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=20, check=False,
                                 env=entorno_contenedor())
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise Corte("FALTA: PostgreSQL de solo lectura no respondió") from exc
    if process.returncode:
        raise Corte("FALTA: consulta PostgreSQL de solo lectura no disponible en contenedor configurado")
    return [line for line in process.stdout.decode().splitlines() if line not in ("BEGIN", "SET", "ROLLBACK", "")]


def ejecutar_fixture(args, filename, variables):
    for key, value in variables.items():
        if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}", value):
            raise Corte(f"FALTA: parámetro SQL {key} inválido")
    sql = (Path(__file__).parent / filename).read_bytes()
    command = comando_psql(args)
    for key, value in variables.items():
        command += ["-v", f"{key}={value}"]
    try:
        result = subprocess.run(command, input=sql, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, timeout=45, check=False, env=entorno_contenedor())
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise Corte(f"FALTA: fixture {filename} sin confirmación SQL; conservar estado privado") from exc
    if result.returncode:
        raise Corte(f"FALTA: fixture SQL {filename} falló; conservar estado privado y revisar sólo el clon sintético")


def comando_psql(args):
    local = ["podman", "--remote=false"] if args.motor == "podman" else ["docker", "--host", "unix:///var/run/docker.sock"]
    return local + ["exec", "-i", args.contenedor, "psql", "-X", "-A", "-t", "-v", "ON_ERROR_STOP=1",
                    "-h", "/var/run/postgresql", "-U", "postgres", "-d", "postgres"]


def entorno_contenedor():
    env = os.environ.copy()
    for key in ("DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH"):
        env.pop(key, None)
    return env


def firma_fixture(args, state, expediente):
    if state["valores"].get("firma_fixture") == "confirmada":
        return
    circuito = data_http(args, "RRHH", "GET", f"{BASE}/circuito-firma")
    if circuito.get("catalogo_ref") != "vec.contratacion_temporal.circuito_firma:1":
        raise Corte("FALTA: catálogo de firma distinto; revisar los pasos del fixture")
    documentos = circuito.get("documentos", [])
    informe = next((d for d in documentos if d.get("documento") == "informe_definitivo"), None)
    if not informe or [x.get("referencia") for x in informe.get("pasos", [])] != [
        "vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1",
        "vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p2",
    ]:
        raise Corte("FALTA: pasos de firma de informe distintos; no se insertó ficción")
    variables = {"expediente": expediente, "catalogo_ref": circuito["catalogo_ref"],
                 "catalogo_sha": circuito["huella_sha256"]}
    state["valores"]["firma_fixture"] = "pendiente"
    guardar(args.estado, state)
    ejecutar_fixture(args, "fixture_firma.sql", variables)
    state["valores"]["firma_fixture"] = "confirmada"
    guardar(args.estado, state)


def vincular_emision_nativa(args, state, expediente, seleccionado):
    """CT201 enlaza el recibo B7 exacto; no crea Bolsa ni participación."""
    if state["operaciones"].get("vinculo_bolsa", {}).get("estado") == "confirmada":
        return state["operaciones"]["vinculo_bolsa"]["respuesta"]
    llamamiento = campo(seleccionado, "llamamiento_ref")
    recibo = campo(seleccionado, "recibo_ref")
    clave = state["operaciones"]["seleccion"]["cuerpo"]["clave_idempotencia"]
    for ref in (expediente, llamamiento, recibo, clave):
        if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}", ref):
            raise Corte("FALTA: referencias de selección B7 no verificables")
    query = ("SELECT o.bolsa_ref,o.recibo_ref,a.version FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 o "
             "JOIN vec_contratacion_temporal.expediente_integral_actual a ON a.expediente_ref=o.expediente_ref "
             f"WHERE o.expediente_ref='{expediente}' AND o.clave_ct='{clave}' "
             f"AND o.llamamiento_ref='{llamamiento}' AND o.recibo_ref='{recibo}'")
    filas = lectura_db(args, query)
    if len(filas) != 1 or len(filas[0].split("|")) != 3:
        raise Corte("FALTA: origen B110 único y recibo B7 coincidente para CT201")
    bolsa, recibo_real, version = filas[0].split("|")
    if (recibo_real != recibo or not version.isdigit() or
            not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}", bolsa)):
        raise Corte("FALTA: versión CT u origen B7 no verificables")
    return efecto(args, state, "vinculo_bolsa", lambda: {
        "expediente_ref": expediente, "version_esperada": int(version),
        "bolsa_ref": bolsa, "llamamiento_ref": llamamiento,
        "recibo_emision_ref": recibo})


def continuar_llamamiento(args, state, expediente):
    seleccionado = efecto(args, state, "seleccion", lambda: {
        "expediente_ref": expediente, "version_esperada": 6})
    llamamiento = campo(seleccionado, "llamamiento_ref")
    vinculo_bolsa = vincular_emision_nativa(args, state, expediente, seleccionado)

    def comunicacion():
        return {"organizacion_ref": "organizacion:desarrollo:dipgra",
                "expediente_ref": expediente, "llamamiento_ref": llamamiento,
                "version_esperada": 1,
                "prueba_entrega_ref": campo(seleccionado, "recibo_ref")}
    comunicada = efecto(args, state, "comunicacion", comunicacion)
    comunicacion_ref = campo(comunicada, "comunicacion_ref")

    def respuesta():
        correo_ref = f"correo:rrhh:b2:{uuid.uuid4().hex}"
        huella = hashlib.sha256((correo_ref + ":contenido-ficticio").encode()).hexdigest()
        return {"organizacion_ref": "organizacion:desarrollo:dipgra", "expediente_ref": expediente,
                "llamamiento_ref": llamamiento, "comunicacion_ref": comunicacion_ref,
                "version_comunicacion_esperada": 2, "respuesta": "aceptacion",
                "correo_ref": correo_ref, "correo_sha256": huella,
                "recibida_en": (datetime.now(timezone.utc) - timedelta(minutes=1)).strftime("%Y-%m-%dT%H:%M:%SZ")}
    respondida = efecto(args, state, "respuesta", respuesta)

    # El recibo RRHH no decide por la persona. La aceptación firme se registra
    # con su propio certificado y clave en Bolsa B30 antes de resolver en CT.
    efecto(args, state, "respuesta_portal", lambda: {
        "bolsa": campo(vinculo_bolsa, "bolsa_ref"), "respuesta": "acepta",
        "causa": "", "justificante_ref": "", "justificante_sha256": ""})

    def aceptacion():
        return {"organizacion_ref": "organizacion:desarrollo:dipgra", "expediente_ref": expediente,
                "llamamiento_ref": llamamiento, "comunicacion_ref": comunicacion_ref,
                "version_esperada": 2, "respuesta": "aceptacion",
                "prueba_respuesta_ref": campo(respondida, "justificante_ref"),
                "revision_respuesta_rrhh": True, "revision_plazo_rrhh": True,
                "criterio_validacion_ref": "politica:ct:revision-manual-sintetica:20260906"}
    aceptada = efecto(args, state, "aceptacion", aceptacion)

    def propuesta():
        ruta = "/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json"
        code, publicacion = http(args, "RRHH", "GET", ruta)
        if code != 200 or publicacion.get("esquema") != "vec.ct.propuesta.publicaciones-desarrollo.v1":
            raise Corte("FALTA: publicaciones de formalización vigentes del servidor")
        entradas = publicacion.get("publicaciones", {})
        def snapshot(nombre):
            x = entradas[nombre]
            return {"referencia": x["referencia"], "version": x["version"],
                    "huella_sha256": hashlib.sha256(x["contenido"].encode()).hexdigest()}
        b = {"expediente_ref": expediente, "llamamiento_ref": llamamiento,
             "resolucion_llamamiento_aceptada_ref": campo(aceptada, "resolucion_ref"),
             "recibo_resolucion_aceptada_ref": campo(aceptada, "recibo_local_ref"),
             "version_esperada": 6, "anexos": []}
        for name in ("tipo_formalizacion", "plantilla", "politica_firma", "plan_firma"):
            b[name] = snapshot(name)
        return b
    propuesta_out = efecto(args, state, "propuesta", propuesta)
    prep = data_http(args, "RRHH", "GET", f"{BASE}/resoluciones-formalizacion?expediente_ref={quote(expediente, safe='')}")
    propuesta_ref = campo(prep, "propuesta_ref")
    if propuesta_ref != campo(propuesta_out, "propuesta_ref"):
        raise Corte("FALTA: preparación de resolución no corresponde a propuesta propia")

    def resolucion():
        return {"expediente_ref": expediente, "version_esperada": campo(prep, "version_esperada"),
                "propuesta_ref": propuesta_ref, "numero_resolucion": f"RRHH/2026/{uuid.uuid4().hex[:8].upper()}",
                "fecha_resolucion": date.today().isoformat(),
                "motivo": "Cobertura temporal de la vacante de Administrativo de Transformación Digital.",
                "confirma_revision_propuesta": True, "confirma_ejercicio_manual": True}
    efecto(args, state, "resolucion", resolucion)
    final = data_http(args, "RRHH", "GET", f"{PLAN}?expediente_ref={quote(expediente, safe='')}")
    vacantes = final.get("opciones", {}).get("vacantes", [])
    compatibles = [v for v in vacantes if v.get("unidad_ref") == "unidad:centro:520"
                   and v.get("puesto_ref") and v.get("plaza_ref")
                   and v.get("version_plantilla_ref") and v.get("version_rpt_ref")]
    if not compatibles:
        raise Corte("FALTA: GET plan 200 sin vacante RPT válida del centro 520; comprobar publicación histórica y disponibilidad")
    state["valores"]["vacante_plan"] = compatibles[0]
    state["valores"]["plan_get_200"] = True
    guardar(args.estado, state)
    print(f"PREPARADO expediente_ref={expediente} numero={state['valores']['numero']} url={args.origen}/portal-empleado/#contratacion-temporal")
    print(f"RECIBOS_PRIVADOS={args.estado}")


def config_privada(args):
    if not args.config.is_file():
        return None
    try:
        value = json.loads(args.config.read_text(encoding="utf-8"))
    except (ValueError, OSError):
        raise Corte("FALTA: JSON privado B2 ilegible")
    return value.get("personal_b2") if isinstance(value, dict) else None


def rpt_compatible(args):
    cfg = config_privada(args)
    org = cfg.get("organismo_ref") if isinstance(cfg, dict) else None
    if not isinstance(org, str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}", org):
        raise Corte("FALTA: organismo B2 no está fijado en JSON privado")
    query = ("SELECT count(*) FROM vec_personal.puesto_rpt_historia p "
             "JOIN vec_personal.puesto_tipo_historia t ON t.tipo_ref=p.tipo_ref AND t.revision=p.tipo_revision AND t.organismo_ref=p.organismo_ref "
             "JOIN vec_personal.version_rpt_historia r ON r.version_ref=t.rpt_version_ref AND r.revision=t.rpt_revision AND r.organismo_ref=t.organismo_ref "
             "JOIN vec_personal.vinculo_plaza_puesto_historia v "
             "ON v.puesto_ref=p.puesto_ref AND v.organismo_ref=p.organismo_ref "
             "JOIN vec_personal.plaza_plantilla_historia l "
             "ON l.plaza_ref=v.plaza_ref AND l.organismo_ref=p.organismo_ref "
             "JOIN vec_personal.version_plantilla_historia pl ON pl.version_ref=l.plantilla_version_ref AND pl.revision=l.plantilla_revision AND pl.organismo_ref=l.organismo_ref "
             f"WHERE p.organismo_ref='{org}' AND p.unidad_ref='unidad:centro:520' "
             "AND p.codigo_puesto_fuente='9/01' AND l.codigo_plaza_fuente='2026/0033' AND l.unidad_ref=p.unidad_ref AND v.unidad_ref=p.unidad_ref "
             "AND r.estado='publicada' AND pl.estado='publicada' AND r.publicada_en IS NOT NULL AND pl.publicada_en IS NOT NULL "
             "AND NOT p.retirado AND NOT t.retirado AND NOT v.retirado AND NOT l.retirado AND NOT r.retirado AND NOT pl.retirado "
             "AND p.fuente_ref NOT LIKE 'fuente:ensayo:%' AND l.fuente_ref NOT LIKE 'fuente:ensayo:%' AND v.fuente_ref NOT LIKE 'fuente:ensayo:%'")
    count = lectura_db(args, query)
    if len(count) != 1 or not count[0].isdigit():
        raise Corte("FALTA: consulta de puesto/plaza RPT no verificable")
    return int(count[0]), org


def necesidad_rpt_publicada():
    try:
        raw = RPT_PUBLICA.read_bytes()
        catalogo = json.loads(raw)
    except (OSError, ValueError) as exc:
        raise Corte("FALTA: catálogo RPT público usado por el servidor") from exc
    if not isinstance(catalogo, dict) or not isinstance(catalogo.get("puestos"), list):
        raise Corte("FALTA: catálogo RPT público con estructura verificable")
    candidatos = [p for p in catalogo["puestos"] if isinstance(p, dict) and p.get("codigo") == "9"
                  and p.get("centro_codigo") == "520" and p.get("categoria_clave") == "administrativo"]
    fuente = catalogo.get("fuente", {})
    if len(candidatos) != 1 or not isinstance(fuente.get("importacion"), str) or not fuente["importacion"]:
        raise Corte("FALTA: puesto Administrativo 9/01 del centro 520 en RPT pública")
    return {"puesto_codigo": "9", "rpt_catalogo_ref": fuente["importacion"],
            "rpt_catalogo_huella_sha256": hashlib.sha256(raw).hexdigest(),
            "denominacion": candidatos[0]["denominacion"]}


def preparar_rpt(args, state):
    count, _ = rpt_compatible(args)
    if count != 1:
        raise Corte("FALTA: RPT publicada compatible con centro 520, puesto 9 y plaza 2026/0033")
    state["valores"]["rpt_necesidad"] = necesidad_rpt_publicada()
    state["valores"]["rpt_puesto9"] = "existente"


def comprobar_fuente_bolsa_ct(args):
    """Coteja el montaje SQL y una constitución vigente sin consumir su lectura nominal."""
    query = ("SELECT "
             "(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace "
             "WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='leer_origen_llamamiento_ct_v1'),"
             "(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace "
             "WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='leer_origen_bolsa_ct_auditado_v1'),"
             "(SELECT count(*) FROM vec_bolsa_llamamientos.constitucion c "
             "JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=c.bolsa_ref "
             "AND b.version=c.version_bolsa AND b.huella_bolsa_sha256=c.huella_bolsa_sha256 "
             "WHERE c.categoria_ref='categoria:rpt:administrativo' AND b.estado='vigente' "
             "AND b.vigente_desde<=now() AND (b.vigente_hasta IS NULL OR now()<b.vigente_hasta) "
             "AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s "
             "WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa "
             "AND s.huella_sustituida_sha256=c.huella_bolsa_sha256))")
    if lectura_db(args, query) != ["1|1|1"]:
        raise Corte("FALTA: B110/AD261 o Bolsa Administrativo constituida vigente única")
    m10 = args.config.parent.parent / "cliente_m10_ct.json"
    try:
        cfg = json.loads(m10.read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        raise Corte(f"FALTA: configuración privada M10 en {m10}") from exc
    organismo = config_privada(args)
    if not isinstance(cfg, dict) or (cfg.get("version") != 1 or cfg.get("origen") != args.origen or
            cfg.get("organismo_ref") != (organismo or {}).get("organismo_ref") or
            not all(cfg.get(k) for k in ("directorio", "autoridad_ca", "certificado_cliente", "clave_cliente"))):
        raise Corte("FALTA: M10 no coincide con origen, organismo y material privado B2")
    try:
        launcher = args.launcher.read_text(encoding="utf-8")
    except OSError as exc:
        raise Corte("FALTA: lanzador privado para verificar M10") from exc
    if not re.search(r"(?m)^\s*(?:export\s+)?VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH\s*=\s*\S+", launcher):
        raise Corte("FALTA: VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH en lanzador privado")


def comando_catalogo_faltante(args, referencia):
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
    cert, key = mTLS(args, "RRHH")
    privado = args.estado.parent / ("catalogo-" + referencia.replace(":", "-") + ".uuid")
    cuerpo = EJEMPLOS / "catalogos-personal" / archivos[referencia]
    endpoint = args.origen + f"{BASE}/incorporacion-personal-b2/catalogos-registro-empleado/v1"
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


def comprobar(args):
    faltas = []
    ajustar = shlex.quote(str(Path(__file__).with_name("ajustar_privado.py")))
    try:
        lectura_db(args, "SELECT 1")
        print("EXISTE PostgreSQL de solo lectura")
    except Corte:
        faltas.append("FALTA PostgreSQL: --motor podman --contenedor vec-postgresql-20260906")
    for actor in ("RRHH", "centro-solicitante", "centro-ratificador", "intervencion", "candidato"):
        try:
            mTLS(args, actor)
            print(f"EXISTE mTLS {actor}")
        except Corte as exc:
            faltas.append(str(exc))
    if not (args.material / "ca/ca.crt").is_file():
        faltas.append("FALTA CA privada: --material <directorio-con-ca-y-mtls>")
    try:
        comprobar_fuente_bolsa_ct(args)
        print("EXISTE fuente B110/AD261, Bolsa Administrativo única y M10 configurado")
    except Corte as exc:
        faltas.append(str(exc))
    try:
        launcher = args.launcher.read_text(encoding="utf-8")
    except OSError:
        launcher = ""
    if re.search(r"(?m)^\s*(?:export\s+)?VEC_PERSONAL_B2_GOBIERNO_ENABLED\s*=\s*['\"]?true['\"]?(?:\s|$)", launcher):
        print("EXISTE VEC_PERSONAL_B2_GOBIERNO_ENABLED=true en lanzador privado")
    else:
        faltas.append(f"FALTA flag VEC_PERSONAL_B2_GOBIERNO_ENABLED=true; comando manual: python3 {ajustar} flag {shlex.quote(str(args.launcher))}")
    cfg = config_privada(args)
    if isinstance(cfg, dict) and cfg.get("cese_fecha_efecto") in ("primer_dia_sin_relacion", "ultimo_dia_con_relacion"):
        print("EXISTE personal_b2.cese_fecha_efecto en configuración privada")
    else:
        faltas.append(f"FALTA personal_b2.cese_fecha_efecto; comando manual: python3 {ajustar} cese {shlex.quote(str(args.config))}")
    try:
        rpt_count, _ = rpt_compatible(args)
        necesidad_rpt_publicada()
        if rpt_count == 1:
            print("EXISTE RPT puesto 9/01, plaza 2026/0033, unidad centro 520")
        else:
            faltas.append("FALTA: RPT publicada compatible con centro 520, puesto 9 y plaza 2026/0033")
    except Corte as exc:
        faltas.append(str(exc))
    anterior = None
    if args.estado.exists():
        try:
            anterior = cargar(args.estado)
            if anterior.get("destino", {}).get("origen") == args.origen:
                if anterior.get("valores", {}).get("plan_get_200") is True:
                    print("EXISTE expediente B2 preparado según recibos privados; --aplicar recupera su lectura")
                else:
                    print("EXISTE preparación B2 parcial; --aplicar recupera las claves originales")
            else:
                faltas.append("FALTA: el estado B2 pertenece a otro origen")
        except (Corte, OSError, ValueError):
            faltas.append("FALTA: estado B2 privado ilegible o inseguro")
    else:
        print("FALTA expediente B2 preparado en este estado; no hay recibo previo")
    if not isinstance(anterior, dict) or not isinstance(anterior.get("operaciones", {}).get("alta"), dict):
        faltas.append("FALTA: la entrega ratificada centro→RRHH no transporta necesidad RPT versionada; "
                      "pendiente contrato de presentación/ratificación antes del primer POST")
    try:
        # El GET nominal sólo devuelve catálogo; se consulta con la identidad RRHH.
        route = "/api/vec/personal/catalogos-registro-empleado?tipo=regimen&estado=publicada&limite=100"
        regimes = data_http(args, "RRHH", "GET", route)
        route = "/api/vec/personal/catalogos-registro-empleado?tipo=modalidad&estado=publicada&limite=100"
        modalities = data_http(args, "RRHH", "GET", route)
        entries = list(regimes.get("entradas", [])) + list(modalities.get("entradas", []))
        required = {"regimen:funcionario_carrera", "regimen:funcionario_interino",
                    "regimen:laboral_fijo", "regimen:laboral_temporal", "modalidad:vacante",
                    "modalidad:sustitucion", "modalidad:programa", "modalidad:acumulacion_tareas"}
        have = {item.get("ref") for item in entries if item.get("estado") == "publicada"}
        for missing in sorted(required - have):
            faltas.append(f"FALTA catálogo HTTP {missing}; comando para publicar con clave privada durable:\n{comando_catalogo_faltante(args, missing)}")
        if required <= have:
            print("EXISTEN ocho catálogos régimen/modalidad publicados")
    except Corte as exc:
        faltas.append(str(exc))
    for missing in faltas:
        print(missing)
    return 0 if not faltas else 2


def nuevo_numero(args, state, pedido):
    valores = state["valores"]
    if "numero" not in valores:
        catalogo = data_http(args, "RRHH", "GET", f"{BASE}/catalogos-alta")
        politica = catalogo.get("numero_expediente_moad")
        if not isinstance(politica, dict) or not isinstance(politica.get("patron"), str) or not isinstance(politica.get("ejemplo"), str):
            raise Corte("FALTA: política MOAD publicada en catálogo de alta")
        muestra = re.fullmatch(r"([0-9]{4}/[A-Za-z0-9._-]*?)([0-9]{1,12})", politica["ejemplo"])
        if not muestra or muestra.group(1)[:4] != str(date.today().year):
            raise Corte("FALTA: ejemplo MOAD anual con sufijo numérico utilizable")
        prefijo, cifras = muestra.groups()
        q = ("SELECT coalesce(max(substring(numero_visible from " + str(len(prefijo) + 1) + ")::numeric),-1) FROM ("
             "SELECT numero_visible FROM vec_contratacion_temporal.expediente_alta UNION ALL "
             "SELECT numero_visible FROM vec_contratacion_temporal.identidad_reserva_alta UNION ALL "
             "SELECT numero_visible FROM vec_contratacion_temporal.numeracion_anual_asignada UNION ALL "
             "SELECT numero_visible FROM vec_contratacion_temporal.candidatura_alta_tecnica) n "
             f"WHERE left(numero_visible,{len(prefijo)})='{prefijo}' "
             f"AND substring(numero_visible from {len(prefijo) + 1}) ~ '^[0-9]+$'")
        usado = lectura_db(args, q)
        if len(usado) != 1 or not re.fullmatch(r"-?[0-9]+", usado[0]):
            raise Corte("FALTA: numeración CT y reservas no verificables")
        numero = pedido or prefijo + str(max(int(cifras), int(usado[0]) + 1)).zfill(len(cifras))
        try:
            valido = len(politica["patron"]) <= 512 and re.fullmatch(politica["patron"], numero)
        except re.error:
            valido = None
        if not valido or not numero.startswith(prefijo) or not numero[len(prefijo):].isdigit() or int(numero[len(prefijo):]) <= int(usado[0]):
            raise Corte("FALTA: número MOAD fuera de política publicada o ya reservado")
        valores["numero"] = numero
    if pedido and valores["numero"] != pedido:
        raise Corte("FALTA: --numero no coincide con el número privado del primer intento")
    return valores["numero"]


def nueva_peticion(args, state):
    now = date.today()
    rpt = state["valores"].get("rpt_necesidad")
    if not isinstance(rpt, dict) or rpt.get("puesto_codigo") != "9":
        raise Corte("FALTA: publicación RPT del puesto solicitado")
    # Después del cese sintético conservado el 20-11-2026. El dato es de ensayo.
    start = max(now + timedelta(days=21), date(2026, 12, 1))
    end = date(2027, 1, 31) if start == date(2026, 12, 1) else start + timedelta(days=61)
    def body():
        x = ejemplo(1, "presentar")
        x["solicitud"]["periodo"] = {"inicio": start.isoformat() + "T00:00:00Z",
                                      "fin": end.isoformat() + "T00:00:00Z"}
        x["solicitud"]["detalle"] = "Cobertura de la vacante de Administrativo para tramitar solicitudes y documentación de Transformación Digital."
        x["solicitud"]["puesto_solicitado"] = "Puesto 9 — " + rpt["denominacion"]
        return x
    return efecto(args, state, "presentar", body)


def aplicar(args, state):
    esperado = {"origen": args.origen, "material": str(args.material), "contenedor": args.contenedor}
    if "destino" in state and state["destino"] != esperado:
        raise Corte("FALTA: estado privado pertenece a otro origen, material o contenedor")
    state["destino"] = esperado
    if (state["valores"].get("plan_get_200") is True
            and state["operaciones"].get("resolucion", {}).get("estado") == "confirmada"):
        expediente = state["valores"]["expediente_ref"]
        route = f"{PLAN}?expediente_ref={quote(expediente, safe='')}"
        code, body = http(args, "RRHH", "GET", route)
        if code == 200 and isinstance(body.get("data"), dict):
            opciones = body["data"].get("opciones")
            if body["data"].get("recibo"):
                print(f"PREPARADO_HISTORICO expediente_ref={expediente} numero={state['valores']['numero']} incorporación posterior recuperada")
            elif isinstance(opciones, dict) and opciones.get("vacantes"):
                print(f"PREPARADO expediente_ref={expediente} numero={state['valores']['numero']} url={args.origen}/portal-empleado/#contratacion-temporal")
            else:
                print(f"PREPARADO_HISTORICO expediente_ref={expediente} numero={state['valores']['numero']} incorporación/cese posteriores recuperados")
            print(f"RECIBOS_PRIVADOS={args.estado}")
            return
        raise Corte(f"FALTA: lectura GET del plan existente respondió HTTP {code}; no se repitió ningún POST")
    preparar_rpt(args, state)
    # Cada cuerpo queda congelado en disco antes del POST. Una interrupción obliga
    # a reintentar esa misma intención; nunca se inventa otra clave en recuperación.
    numero = nuevo_numero(args, state, args.numero)
    guardar(args.estado, state)
    presentado = nueva_peticion(args, state)
    peticion = campo(presentado, "peticion_ref")
    if not isinstance(peticion, str) or not peticion.startswith("peticion:centro:"):
        raise Corte("FALTA: petición de centro sin referencia verificable")

    def ratificar():
        b = ejemplo(2, "ratificar")
        b["peticion_ref"] = peticion
        b["motivo"] = "Revisadas la vacante de Administrativo, la categoría C1 y las fechas solicitadas por Transformación Digital."
        return b
    efecto(args, state, "ratificar", ratificar)

    alta = efecto(args, state, "alta", lambda: {
        "peticion_ref": peticion, "version_esperada": 2,
        "numero_expediente_moad": numero,
    })
    expediente = campo(alta, "expediente_ref")
    if not isinstance(expediente, str) or not expediente.startswith("expediente:ct:"):
        raise Corte("FALTA: alta CT sin expediente_ref verificable")
    state["valores"].update(expediente_ref=expediente, numero=numero)
    guardar(args.estado, state)
    if args.hasta == "alta":
        print(f"TRAMO_ALTA expediente_ref={expediente} numero={numero} estado={args.estado}")
        return

    def analisis():
        b = ejemplo(4, "analisis")
        b["expediente_ref"] = expediente
        if (b["analisis"]["categoria_ref"] != state["operaciones"]["presentar"]["cuerpo"]["solicitud"]["categoria_ref"]
                or state["valores"]["rpt_necesidad"]["puesto_codigo"] != "9"):
            raise Corte("FALTA: análisis no coincide con la categoría y el puesto RPT solicitados")
        b["analisis"]["periodo"] = state["operaciones"]["presentar"]["cuerpo"]["solicitud"]["periodo"]
        b["analisis"]["observaciones"] = "La vacante de Administrativo permitirá tramitar las solicitudes y la documentación de Transformación Digital durante el periodo solicitado."
        # Son publicaciones técnicas gobernadas por la preparación RRHH.
        # El ejemplo del contrato trae las tres referencias instaladas.
        return b
    efecto(args, state, "analisis", analisis)
    propuesta = None
    if "cobertura" not in state["operaciones"]:
        propuesta = data_http(args, "RRHH", "POST", f"{BASE}/cobertura/propuesta",
                             {"expediente_ref": expediente, "version_esperada": 2})

    def cobertura():
        b = ejemplo(5, "cobertura")
        b["expediente_ref"] = expediente
        b["identidad_semantica"] = campo(propuesta, "identidad_semantica")
        if b["via_elegida"] not in str(propuesta):
            raise Corte("FALTA: propuesta gobernada no ofrece bolsa_vigente")
        return b
    efecto(args, state, "cobertura", cobertura)
    efecto(args, state, "asignacion", lambda: {**ejemplo(6, "asignacion"), "expediente_ref": expediente})
    efecto(args, state, "informe", lambda: {**ejemplo(7, "informe"), "expediente_ref": expediente})
    if args.hasta == "informe":
        print(f"TRAMO_INFORME expediente_ref={expediente} numero={numero} estado={args.estado}")
        return

    vinculo = data_http(args, "RRHH", "GET", f"{BASE}/incorporacion-personal-b2/vinculo-categoria-rpt/v1?expediente_ref={quote(expediente, safe='')}")
    categoria = data_http(args, "RRHH", "GET", f"{BASE}/incorporacion-personal-b2/categorias-rpt/v1")
    entrada = next((x for x in categoria.get("categorias", []) if x.get("categoria_id") == "categoria:rpt:administrativo"), None)
    if not entrada:
        raise Corte("FALTA: categoría RPT administrativo publicada; revisar GET categorías-rpt/v1 y cargar la fuente existente de forma autorizada")
    anclaje = campo(vinculo, "analisis")

    def registrar_vinculo():
        a = anclaje
        if not isinstance(a, dict):
            raise Corte("FALTA: anclaje de análisis RPT inválido")
        return {"expediente_ref": expediente, "version_expediente_esperada": a["version_expediente"],
                "analisis_version": a["analisis_version"], "analisis_recibo_ref": a["analisis_recibo_ref"],
                "analisis_huella_sha256": a["analisis_huella_sha256"], "categoria_ref": a["categoria_ref"],
                "catalogo_version": entrada["catalogo_version"], "catalogo_huella_sha256": entrada["catalogo_huella_sha256"],
                "categoria_id": entrada["categoria_id"], "fuente_ref": entrada["fuente_ref"],
                "motivo_ref": a["analisis_recibo_ref"], "aprobacion_ref": entrada["aprobacion_ref"],
                "revision_esperada": 0, "anterior_recibo_ref": ""}
    efecto(args, state, "vinculo_rpt", registrar_vinculo)

    # El informe tiene versión 5. El circuito del ensayo exige dos precondiciones
    # ficticias de remisión antes de la fiscalización; se insertan por SQL acotado.
    firma_fixture(args, state, expediente)
    efecto(args, state, "fiscalizacion", lambda: {**ejemplo(8, "fiscalizacion"), "expediente_ref": expediente})
    if args.hasta == "fiscalizacion":
        print(f"TRAMO_FISCALIZACION expediente_ref={expediente} numero={numero} estado={args.estado}")
        return
    continuar_llamamiento(args, state, expediente)


def argumentos(argv):
    p = argparse.ArgumentParser(description=__doc__)
    group = p.add_mutually_exclusive_group(required=True)
    group.add_argument("--comprobar", action="store_true")
    group.add_argument("--aplicar", metavar="PREPARA-ALTA-B2")
    p.add_argument("--origen", default=os.environ.get("VEC_B2_ORIGEN", "https://127.0.0.1:8443"))
    p.add_argument("--material", type=Path, default=Path(os.environ.get("VEC_B2_MATERIAL", "/home/openclaw/.local/state/vec-desarrollo-20260906/material/desarrollo-fiscalizacion-20260904")))
    p.add_argument("--launcher", type=Path, help="lanzador privado del servidor; sólo lectura")
    p.add_argument("--config", type=Path, default=Path(os.environ["VEC_B2_CONFIG"]) if "VEC_B2_CONFIG" in os.environ else None)
    p.add_argument("--estado", type=Path, default=Path(os.environ.get("VEC_B2_ESTADO", "/home/openclaw/.local/state/vec-desarrollo-20260906/preparar-alta-b2/estado.json")))
    p.add_argument("--motor", choices=("podman", "docker"), default=os.environ.get("VEC_B2_MOTOR", "podman"))
    p.add_argument("--contenedor", default=os.environ.get("VEC_B2_CONTENEDOR", "vec-postgresql-20260906"))
    p.add_argument("--numero", default=os.environ.get("VEC_B2_NUMERO", ""))
    p.add_argument("--hasta", choices=("alta", "informe", "fiscalizacion", "resolucion"), default="resolucion")
    args = p.parse_args(argv)
    if args.aplicar is not None and args.aplicar != "PREPARA-ALTA-B2":
        raise Corte("FALTA: --aplicar exige el argumento exacto PREPARA-ALTA-B2")
    args.origen = exigir_origen(args.origen)
    args.material = args.material.resolve()
    args.launcher = (args.launcher or args.material.parents[1] / "arrancar_app.sh").resolve()
    if args.config is None:
        estado_dir = args.material.parents[1]
        preferida = estado_dir / "incorporacion-servidor-20260910/config/personal-b2/servidor.json"
        candidatas = sorted(estado_dir.glob("incorporacion-servidor-*/config/personal-b2*/servidor.json"))
        args.config = preferida if preferida.is_file() else (candidatas[0] if len(candidatas) == 1 else preferida)
    args.config = args.config.resolve()
    args.estado = args.estado.resolve()
    if ROOT == args.estado or ROOT in args.estado.parents:
        raise Corte("FALTA: --estado debe estar fuera de Git")
    if not re.fullmatch(r"[a-zA-Z0-9_-]{3,100}", args.contenedor):
        raise Corte("FALTA: nombre de contenedor inválido")
    return args


def main(argv):
    os.umask(0o077)
    args = argumentos(argv)
    if args.comprobar:
        return comprobar(args)
    if comprobar(args):
        raise Corte("FALTA: precondiciones; no se generó ni emitió efecto")
    args.estado.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
    lock = args.estado.with_suffix(args.estado.suffix + ".lock")
    if lock.is_symlink():
        raise Corte("FALTA: bloqueo privado inseguro")
    fd = os.open(lock, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        state = cargar(args.estado)
        aplicar(args, state)
    except BlockingIOError as exc:
        raise Corte("FALTA: ya hay otra preparación usando este estado privado") from exc
    finally:
        os.close(fd)
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv[1:]))
    except Corte as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(2)
