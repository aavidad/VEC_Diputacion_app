#!/usr/bin/env python3
"""Comprueba y prepara un expediente sintético B2 hasta la firma real."""
from __future__ import annotations

import argparse
import fcntl
import hashlib
import json
import os
import re
import shlex
import stat
import subprocess
import sys
import tempfile
import uuid
from datetime import date, datetime, timedelta, timezone
from pathlib import Path
from urllib.parse import urlsplit, quote

from preparar_configuracion import (comando_catalogo_faltante as comando_catalogo_privado,
                                    comando_datos_ejercicio, comprobar_relevo_cese as relevo_cese_configurado,
                                    ruta_datos_ejercicio, comando_psql, entorno_contenedor,
                                    leer_conexiones_privadas)

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
    "vinculo_rpt": ("RRHH", f"{BASE}/incorporacion-personal-b2/vinculo-categoria-rpt/v1"),
}
RPT_PUBLICA = ROOT / "data/catalogos/rpt/v1.rpt-2026.json"
RUTA_CONTEXTO_CENTRO = f"{BASE}/peticiones-centro/contexto"
ESQUEMA_NECESIDAD = "vec.ct.alta_necesidad.v1"
DATOS_EJERCICIO = ("plaza_codigo", "organica_codigo", "funcional_codigo",
                   "proyecto_gasto_codigo", "porcentaje_financiacion")
BOLSA_VIGENTE = "bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk"
CT215_PROSRC = {
    "registrar_peticion_centro_v1": "f414c20fa088b3a86ab8987fe58f69fe3796eda75af2ef6a4d1c8b2e8d140016",
    "gestionar_entrega_peticion_centro_v1": "954e275051e54b259a690820208ab75cab3d9a668cf520741803c0b6a995cbc9",
}


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
    if (not isinstance(value, dict) or value.get("esquema") != "vec.preparar-alta-b2.v1" or
            not isinstance(value.get("operaciones"), dict) or not isinstance(value.get("valores", {}), dict) or
            ("destino" in value and not isinstance(value["destino"], dict))):
        raise Corte("FALTA: formato de estado privado ajeno")
    value.setdefault("valores", {})
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


def efecto(args, state, name, make_body):
    actor, route = RUTAS[name]
    old = state["operaciones"].get(name)
    if old is None:
        body = make_body()
        if name != "alta":
            body["clave_idempotencia"] = str(uuid.uuid4())
        old = {"actor": actor, "ruta": route, "cuerpo": body, "estado": "pendiente"}
        state["operaciones"][name] = old
        guardar(args.estado, state)  # durable ANTES de emitir cualquier POST
    if old["actor"] != actor or old["ruta"] != route:
        raise Corte(f"FALTA: operación privada {name} incompatible")
    if old.get("estado") == "confirmada":
        return old["respuesta"]
    out = data_http(args, actor, "POST", route, old["cuerpo"], allowed=(200, 201))
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
    plaza = datos_ejercicio(args)["plaza_codigo"]
    query = ("SELECT count(*),min(p.puesto_ref::text),min(l.plaza_ref::text) FROM vec_personal.puesto_rpt_historia p "
             "JOIN vec_personal.puesto_tipo_historia t ON t.tipo_ref=p.tipo_ref AND t.revision=p.tipo_revision AND t.organismo_ref=p.organismo_ref "
             "JOIN vec_personal.version_rpt_historia r ON r.version_ref=t.rpt_version_ref AND r.revision=t.rpt_revision AND r.organismo_ref=t.organismo_ref "
             "JOIN vec_personal.vinculo_plaza_puesto_historia v "
             "ON v.puesto_ref=p.puesto_ref AND v.organismo_ref=p.organismo_ref "
             "JOIN vec_personal.plaza_plantilla_historia l "
             "ON l.plaza_ref=v.plaza_ref AND l.organismo_ref=p.organismo_ref "
             "JOIN vec_personal.version_plantilla_historia pl ON pl.version_ref=l.plantilla_version_ref AND pl.revision=l.plantilla_revision AND pl.organismo_ref=l.organismo_ref "
             f"WHERE p.organismo_ref='{org}' AND p.unidad_ref='unidad:centro:520' "
             f"AND p.codigo_puesto_fuente='9/01' AND (l.codigo_plaza_fuente='{plaza}' "
             f"OR split_part(l.codigo_plaza_fuente,'/',2)='{plaza}') "
             "AND l.unidad_ref=p.unidad_ref AND v.unidad_ref=p.unidad_ref "
             "AND r.estado='publicada' AND pl.estado='publicada' AND r.publicada_en IS NOT NULL AND pl.publicada_en IS NOT NULL "
             "AND NOT p.retirado AND NOT t.retirado AND NOT v.retirado AND NOT l.retirado AND NOT r.retirado AND NOT pl.retirado "
             "AND p.fuente_ref NOT LIKE 'fuente:ensayo:%' AND l.fuente_ref NOT LIKE 'fuente:ensayo:%' AND v.fuente_ref NOT LIKE 'fuente:ensayo:%'")
    count = lectura_db(args, query)
    columnas = count[0].split("|") if len(count) == 1 else []
    if len(columnas) != 3 or not columnas[0].isdigit():
        raise Corte("FALTA: consulta de puesto/plaza RPT no verificable")
    return int(columnas[0]), org, (columnas[1], columnas[2])


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
    if not isinstance(fuente, dict):
        raise Corte("FALTA: procedencia del catálogo RPT público")
    if len(candidatos) != 1 or not isinstance(fuente.get("importacion"), str) or not fuente["importacion"]:
        raise Corte("FALTA: puesto Administrativo 9/01 del centro 520 en RPT pública")
    return {"puesto_codigo": "9", "rpt_catalogo_ref": fuente["importacion"],
            "rpt_catalogo_huella_sha256": hashlib.sha256(raw).hexdigest(),
            "denominacion": candidatos[0]["denominacion"]}


def datos_ejercicio(args):
    ruta = ruta_datos_ejercicio(args)
    if not ruta.is_absolute() or ROOT == ruta or ROOT in ruta.parents or ruta.parent.is_symlink():
        raise Corte("FALTA: JSON de necesidad exige ruta privada absoluta fuera de Git")
    try:
        info = ruta.stat()
        if ruta.is_symlink() or not stat.S_ISREG(info.st_mode) or info.st_size > 4096 or stat.S_IMODE(info.st_mode) != 0o600:
            raise OSError("modo privado")
        datos = json.loads(ruta.read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        raise Corte(f"FALTA: cinco datos de la necesidad del ejercicio en JSON privado 0600: {ruta}") from exc
    if (not isinstance(datos, dict) or set(datos) != set(DATOS_EJERCICIO) or
            any(not isinstance(datos[k], str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._/-]{0,79}", datos[k])
                for k in DATOS_EJERCICIO) or
            not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9-]{0,63}", datos["plaza_codigo"]) or
            not re.fullmatch(r"(?:[1-9]|[1-9][0-9]|100)", datos["porcentaje_financiacion"])):
        raise Corte("FALTA: JSON privado de necesidad exige cinco códigos válidos, sin otros campos")
    return datos


def necesidad_contexto_centro(contexto, solicitud, rpt, datos):
    catalogos = contexto.get("catalogos", {}) if isinstance(contexto, dict) else {}
    actor = contexto.get("actor", {}) if isinstance(contexto, dict) else {}
    fuentes = catalogos.get("necesidades", {}) if isinstance(catalogos, dict) else {}
    vinculos = contexto.get("vinculos_rpt", {}) if isinstance(contexto, dict) else {}
    causa = next((c for c in fuentes.get("causas", []) if isinstance(c, dict) and c.get("clave") == "vacante"), None) if isinstance(fuentes, dict) else None
    obligatorios = {"numero_personas", "plaza_codigo", "puesto_codigo", "rpt_catalogo_ref", "rpt_catalogo_huella_sha256",
                    "organica_codigo", "funcional_codigo", "proyecto_gasto_codigo", "porcentaje_financiacion"}
    if (not isinstance(catalogos, dict) or catalogos.get("esquema") != "vec.contratacion_temporal.catalogos_alta.v2" or
            not isinstance(actor, dict) or actor.get("puede_presentar") is not True or
            not isinstance(causa, dict) or not obligatorios.issubset(set(causa.get("campos_obligatorios", []))) or
            not isinstance(vinculos, dict) or
            not isinstance(vinculos.get("categorias"), list) or not isinstance(vinculos.get("centros"), list) or
            {"categoria_ref": solicitud["categoria_ref"], "rpt_categoria_clave": "administrativo"} not in vinculos.get("categorias", []) or
            {"centro_ref": solicitud["centro_ref"], "rpt_centro_codigo": "520"} not in vinculos.get("centros", []) or
            solicitud["jornada_minutos"] != fuentes.get("jornada_referencia_minutos") or
            solicitud["motivo_clave"] != causa.get("clave") or
            not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}", str(fuentes.get("referencia", ""))) or
            not isinstance(fuentes.get("version"), int) or fuentes["version"] < 1 or
            not re.fullmatch(r"[0-9a-f]{64}", str(fuentes.get("huella_sha256", "")))):
        raise Corte("FALTA: contexto-centro v3 no publica necesidad vacante y vínculo RPT compatibles")
    campos = {"numero_personas": str(solicitud["numero_personas"]), **datos,
              "puesto_codigo": rpt["puesto_codigo"], "rpt_catalogo_ref": rpt["rpt_catalogo_ref"],
              "rpt_catalogo_huella_sha256": rpt["rpt_catalogo_huella_sha256"]}
    return {"esquema": "vec.ct.necesidad_alta.v1", "catalogo_ref": fuentes["referencia"],
            "catalogo_version": fuentes["version"], "catalogo_huella_sha256": fuentes["huella_sha256"],
            "causa_clave": causa["clave"], "jornada_minutos": solicitud["jornada_minutos"], "campos": campos}


def periodo_ejercicio():
    inicio = max(date.today() + timedelta(days=21), date(2026, 12, 1))
    fin = date(2027, 1, 31) if inicio == date(2026, 12, 1) else inicio + timedelta(days=61)
    return inicio, fin


def vacante_personal_compatible(args, plaza, inicio):
    conocido = (datetime.now(timezone.utc) - timedelta(minutes=1)).strftime("%Y-%m-%dT%H:%M:%S.%fZ")
    cursor, vistos, halladas = "", set(), []
    for _ in range(20):
        ruta = f"/api/vec/personal/vacantes?vigente_en={inicio.isoformat()}&conocido_en={quote(conocido, safe='')}&limite=100"
        if cursor:
            ruta += "&cursor=" + quote(cursor, safe="")
        pagina = data_http(args, "RRHH", "GET", ruta).get("pagina", {})
        if not isinstance(pagina, dict) or pagina.get("cobertura") != "completa":
            raise Corte("FALTA: consulta nominal de vacantes Personal con cobertura completa")
        vacantes = pagina.get("vacantes", [])
        if not isinstance(vacantes, list):
            raise Corte("FALTA: lista de vacantes Personal no verificable")
        for fila in vacantes:
            if (isinstance(fila, dict) and isinstance(fila.get("codigo_plaza_fuente"), str) and
                    fila["codigo_plaza_fuente"].rsplit("/", 1)[-1] == plaza and
                    fila.get("unidad_ref") == "unidad:centro:520" and fila.get("puesto_ref") and
                    fila.get("version_plantilla_ref") and fila.get("version_rpt_ref") and
                    fila.get("estado_cobertura") == "vacante_sin_ocupacion"):
                halladas.append(fila)
        siguiente = pagina.get("cursor_siguiente", "")
        if not siguiente:
            break
        if not isinstance(siguiente, str) or siguiente in vistos:
            raise Corte("FALTA: cursor de vacantes Personal ambiguo")
        vistos.add(siguiente)
        cursor = siguiente
    else:
        raise Corte("FALTA: paginación de vacantes Personal excede el límite verificable")
    if len(halladas) != 1:
        raise Corte("FALTA: plaza del ejercicio no es vacante Personal única y compatible en centro 520")
    return halladas[0]


def preparar_rpt(args, state):
    if state["valores"].get("rpt_necesidad") and state["valores"].get("datos_ejercicio"):
        return
    if state["operaciones"].get("presentar"):
        raise Corte("FALTA: estado parcial sin procedencia RPT congelada")
    count, _, refs = rpt_compatible(args)
    if count != 1 or not all(refs):
        raise Corte("FALTA: plaza del JSON privado vinculada al puesto 9/01 publicado del centro 520")
    state["valores"]["rpt_necesidad"] = necesidad_rpt_publicada()
    datos = datos_ejercicio(args)
    plaza = datos["plaza_codigo"]
    state["valores"]["rpt_necesidad"]["plaza_codigo"] = plaza
    state["valores"]["datos_ejercicio"] = datos
    vacante = vacante_personal_compatible(args, plaza, periodo_ejercicio()[0])
    if (vacante["puesto_ref"], vacante["plaza_ref"]) != refs:
        raise Corte("FALTA: vacante Personal no coincide con el puesto y la plaza RPT históricos")
    state["valores"]["vacante_preflight"] = vacante
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
             f"WHERE c.categoria_ref='categoria:rpt:administrativo' AND c.bolsa_ref='{BOLSA_VIGENTE}' AND b.estado='vigente' "
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


def comprobar_ct215(args):
    query = ("SELECT p.proname,encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),p.pronargs,"
             "p.prosecdef,pg_get_userbyid(p.proowner),"
             "coalesce(array_to_string(p.proconfig,','),'') LIKE '%search_path=pg_catalog, pg_temp%',"
             "has_function_privilege('vec_contratacion_temporal_ejecutor',p.oid,'EXECUTE') "
             "FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace "
             "WHERE n.nspname='vec_contratacion_temporal' AND p.proname IN "
             "('registrar_peticion_centro_v1','gestionar_entrega_peticion_centro_v1') ORDER BY p.proname")
    filas = lectura_db(args, query)
    vistas = {}
    for fila in filas:
        columnas = fila.split("|")
        if len(columnas) != 7 or columnas[0] in vistas:
            raise Corte("FALTA: catálogo instalado CT215 no verificable")
        vistas[columnas[0]] = columnas[1:]
    for nombre, esperado in CT215_PROSRC.items():
        actual = vistas.get(nombre, [""])[0]
        if actual != esperado:
            raise Corte(f"FALTA: CT215 {nombre} huella esperada={esperado} actual={actual or 'ausente'}")
        huella, argumentos, definidora, propietario, ruta, ejecutor = vistas[nombre]
        if (argumentos != "11" or definidora != "t" or propietario != "vec_contratacion_temporal_propietario" or
                ruta != "t" or ejecutor != "t"):
            raise Corte(f"FALTA: CT215 {nombre} firma o ACL incompatible")


def comando_catalogo_faltante(args, referencia):
    return comando_catalogo_privado(args, referencia, *mTLS(args, "RRHH"), EJEMPLOS, BASE)


def comprobar_relevo_cese(launcher, conexiones=""):
    try:
        relevo_cese_configurado(launcher, conexiones)
    except ValueError as exc:
        raise Corte(str(exc)) from exc


def comprobar(args):
    faltas = []
    ajustar = shlex.quote(str(Path(__file__).with_name("ajustar_privado.py")))
    anterior = None
    if args.estado.exists():
        try:
            anterior = cargar(args.estado)
            if any(anterior.get("valores", {}).get(x) for x in ("firma_fixture", "anclaje_fixture")):
                raise Corte("FALTA: estado de ensayo con fixture de firma o anclaje; revisar sus actos antes de continuar")
            if anterior.get("destino", {}).get("origen") == args.origen:
                if anterior.get("valores", {}).get("preparacion_lista") is True:
                    print("EXISTE expediente B2 preparado; continúe el recorrido en la ficha web")
                else:
                    print("EXISTE preparación B2 parcial; --aplicar recupera las claves originales")
            else:
                faltas.append("FALTA: el estado B2 pertenece a otro origen")
        except (Corte, OSError, ValueError) as exc:
            faltas.append(f"FALTA: estado B2 privado ilegible o inseguro: {exc}")
    else:
        print("FALTA expediente B2 preparado en este estado; no hay recibo previo")
    nueva = (not isinstance(anterior, dict) or
             ("presentar" not in anterior.get("operaciones", {}) and
              "datos_ejercicio" not in anterior.get("valores", {})))
    finalizada = isinstance(anterior, dict) and anterior.get("valores", {}).get("preparacion_lista") is True
    if finalizada:
        try:
            mTLS(args, "RRHH")
            if not (args.material / "ca/ca.crt").is_file():
                raise Corte("FALTA: CA privada para recuperar el expediente")
        except Corte as exc:
            faltas.append(str(exc))
        for missing in faltas:
            print(missing)
        return 0 if not faltas else 2
    try:
        lectura_db(args, "SELECT 1")
        print("EXISTE PostgreSQL de solo lectura")
    except Corte:
        faltas.append(f"FALTA PostgreSQL: revisar --motor {args.motor} --contenedor {args.contenedor}")
    actores = ("RRHH", "centro-solicitante", "centro-ratificador", "intervencion")
    for actor in actores:
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
    flags_personal = re.findall(r"(?m)^[ \t]*(?:export[ \t]+)?VEC_PERSONAL_B2_GOBIERNO_ENABLED[ \t]*=[ \t]*['\"]?([^'\"\s#]+)", launcher)
    if flags_personal == ["true"]:
        print("EXISTE VEC_PERSONAL_B2_GOBIERNO_ENABLED=true en lanzador privado")
    else:
        faltas.append(f"FALTA flag VEC_PERSONAL_B2_GOBIERNO_ENABLED=true; comando manual: python3 {ajustar} flag {shlex.quote(str(args.launcher))}")
    try:
        conexiones = leer_conexiones_privadas(args.conexiones) if args.conexiones else ""
        comprobar_relevo_cese(launcher, conexiones)
        print("EXISTEN selector CT, selector Bolsa, relevo activo y DSN nominal de cese")
    except (Corte, ValueError) as exc:
        faltas.append(str(exc))
    cfg = config_privada(args)
    if isinstance(cfg, dict) and cfg.get("cese_fecha_efecto") in ("primer_dia_sin_relacion", "ultimo_dia_trabajado"):
        print("EXISTE personal_b2.cese_fecha_efecto en configuración privada")
    else:
        faltas.append(f"FALTA personal_b2.cese_fecha_efecto; comando manual: python3 {ajustar} cese {shlex.quote(str(args.config))}")
    if nueva:
        try:
            comprobar_ct215(args)
            print("EXISTE CT215 con dos funciones, huellas y permisos esperados")
        except Corte as exc:
            faltas.append(str(exc))
        try:
            datos = datos_ejercicio(args)
            rpt_count, _, refs = rpt_compatible(args)
            rpt = necesidad_rpt_publicada()
            if rpt_count != 1:
                raise Corte("FALTA: plaza privada vinculada al puesto 9/01 publicado del centro 520")
            print(f"EXISTE RPT puesto 9/01 y plaza {datos['plaza_codigo']} publicada en centro 520")
            vacante = vacante_personal_compatible(args, datos["plaza_codigo"], periodo_ejercicio()[0])
            if (vacante["puesto_ref"], vacante["plaza_ref"]) != refs:
                raise Corte("FALTA: vacante Personal no coincide con el puesto y la plaza RPT históricos")
            print("EXISTE vacante Personal única del puesto y plaza para el periodo solicitado")
            contexto = data_http(args, "centro-solicitante", "GET", RUTA_CONTEXTO_CENTRO)
            necesidad_contexto_centro(contexto, ejemplo(1, "presentar")["solicitud"], rpt, datos)
            print("EXISTE contexto-centro v3 con necesidad y vínculo RPT publicados")
        except Corte as exc:
            faltas.append(str(exc))
            if not ruta_datos_ejercicio(args).exists():
                faltas.append("Comando para crear el ejemplo privado; confirme una plaza compatible antes de aplicar:\n" + comando_datos_ejercicio(args))
    else:
        if isinstance(anterior, dict) and "presentar" in anterior.get("operaciones", {}):
            print("EXISTE cuerpo de presentación congelado; se conserva la instantánea de origen al reintentar")
        else:
            print("EXISTEN cinco datos RPT congelados; el primer POST conservará su propia clave y catálogo")
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
    # Después del cese sintético conservado el 20-11-2026. El dato es de ensayo.
    start, end = periodo_ejercicio()
    def body():
        rpt = state["valores"].get("rpt_necesidad")
        if not isinstance(rpt, dict) or rpt.get("puesto_codigo") != "9":
            raise Corte("FALTA: publicación RPT del puesto solicitado")
        x = ejemplo(1, "presentar")
        x["solicitud"]["periodo"] = {"inicio": start.isoformat() + "T00:00:00Z",
                                      "fin": end.isoformat() + "T00:00:00Z"}
        x["solicitud"]["detalle"] = "Cobertura de la vacante de Administrativo para tramitar solicitudes y documentación de Transformación Digital."
        x["solicitud"]["puesto_solicitado"] = "Puesto 9 — " + rpt["denominacion"]
        contexto = data_http(args, "centro-solicitante", "GET", RUTA_CONTEXTO_CENTRO)
        x["esquema"] = ESQUEMA_NECESIDAD
        datos = state["valores"].get("datos_ejercicio")
        if not isinstance(datos, dict) or set(datos) != set(DATOS_EJERCICIO):
            raise Corte("FALTA: cinco datos de la necesidad no congelados en estado privado")
        x["necesidad"] = necesidad_contexto_centro(contexto, x["solicitud"], rpt, datos)
        return x
    return efecto(args, state, "presentar", body)


def aplicar(args, state):
    if any(state["valores"].get(x) for x in ("firma_fixture", "anclaje_fixture")):
        raise Corte("FALTA: estado de ensayo con firma o anclaje ficticios; revisión administrativa requerida")
    esperado = {"origen": args.origen, "material": str(args.material), "contenedor": args.contenedor}
    if "destino" in state and state["destino"] != esperado:
        raise Corte("FALTA: estado privado pertenece a otro origen, material o contenedor")
    state["destino"] = esperado
    if state["valores"].get("preparacion_lista") is True:
        print(f"PREPARADO expediente_ref={state['valores']['expediente_ref']} numero={state['valores']['numero']} estado={args.estado}")
        return
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
                or state["valores"].get("rpt_necesidad", {}).get("puesto_codigo", "9") != "9"):
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
    state["valores"]["preparacion_lista"] = True
    guardar(args.estado, state)
    print(f"PREPARADO expediente_ref={expediente} numero={numero} estado={args.estado}")
    print("Continúe en la ficha web con la firma real del informe y el caso de aceptación de FLUJO_APP.md §4.0")


def argumentos(argv):
    p = argparse.ArgumentParser(description=__doc__)
    group = p.add_mutually_exclusive_group(required=True)
    group.add_argument("--comprobar", action="store_true")
    group.add_argument("--aplicar", metavar="PREPARA-ALTA-B2")
    p.add_argument("--origen", default=os.environ.get("VEC_B2_ORIGEN", "https://127.0.0.1:8443"))
    p.add_argument("--material", type=Path, default=Path(os.environ["VEC_B2_MATERIAL"]) if "VEC_B2_MATERIAL" in os.environ else None)
    p.add_argument("--launcher", type=Path, help="lanzador privado del servidor; sólo lectura")
    p.add_argument("--conexiones", type=Path, help="exportaciones privadas 0600 importadas por el lanzador; sólo lectura")
    p.add_argument("--config", type=Path, default=Path(os.environ["VEC_B2_CONFIG"]) if "VEC_B2_CONFIG" in os.environ else None)
    p.add_argument("--estado", type=Path, default=Path(os.environ["VEC_B2_ESTADO"]) if "VEC_B2_ESTADO" in os.environ else None)
    p.add_argument("--motor", choices=("podman", "docker"), default=os.environ.get("VEC_B2_MOTOR", "podman"))
    p.add_argument("--contenedor", default=os.environ.get("VEC_B2_CONTENEDOR", "vec-postgresql-20260906"))
    p.add_argument("--numero", default=os.environ.get("VEC_B2_NUMERO", ""))
    p.add_argument("--hasta", choices=("alta", "informe"), default="informe")
    args = p.parse_args(argv)
    if args.aplicar is not None and args.aplicar != "PREPARA-ALTA-B2":
        raise Corte("FALTA: --aplicar exige el argumento exacto PREPARA-ALTA-B2")
    if args.material is None or args.estado is None:
        raise Corte("FALTA: indique --material y --estado privados, o VEC_B2_MATERIAL y VEC_B2_ESTADO")
    args.origen = exigir_origen(args.origen)
    args.material = args.material.resolve()
    if args.conexiones is not None:
        args.conexiones = args.conexiones.absolute()
        if ROOT == args.conexiones or ROOT in args.conexiones.parents:
            raise Corte("FALTA: --conexiones debe estar fuera de Git")
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
