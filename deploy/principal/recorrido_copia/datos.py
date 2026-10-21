"""Datos sintéticos de la copia, tras material y conexiones nuevos."""

import hashlib
import json
import os
from pathlib import Path
import re
from urllib.parse import quote, urlencode, urlsplit

from .bolsa import preparar_bolsa
from .intentos import preparar_intentos


HUELLA_HZ37 = "7ae0613fa174aebc8aa36c079a6c4b268b771df41c9310129ee9dfae9650cc45"


def _privado(ruta: Path, valor) -> None:
    temporal = ruta.with_name("." + ruta.name + ".nuevo")
    if temporal.exists() or temporal.is_symlink():
        raise ValueError("Publicación privada anterior incompleta")
    with open(temporal, "x", encoding="utf-8", opener=lambda p, f: os.open(p, f, 0o600)) as salida:
        json.dump(valor, salida, ensure_ascii=False, indent=2)
        salida.write("\n")
    os.replace(temporal, ruta)


def _huella(ruta: Path) -> str:
    with ruta.open("rb") as entrada:
        return hashlib.file_digest(entrada, "sha256").hexdigest()


def _dsn(login: str, password: str, copia, estado) -> str:
    return ("postgresql://" + quote(login) + ":" + quote(password) + "@localhost:" +
            str(estado["puerto_pg"]) + "/postgres?" + urlencode({
                "sslmode": "verify-full", "sslrootcert": str(copia.material / "ca/ca.crt")}))


def preparar_datos(copia) -> None:
    """Ajusta datos HZ37 y publica conexiones propias; falla sobre una copia nueva."""
    estado = copia.leer()
    if estado.get("material_modo") != "nuevo":
        raise ValueError("La fábrica requiere material nuevo de esta copia")
    sid = estado.get("sid_copia")
    if not isinstance(sid, str) or not re.fullmatch(r"[0-9]{15,20}", sid):
        raise ValueError("Falta el SID del conductor de esta copia")
    if estado.get("preparada") or not copia.existe():
        raise ValueError("La copia ya está preparada o no está activa")
    if json.loads((copia.raiz / "fuentes.json").read_text()).get(
            "principal-hz37-20261014.dump") != HUELLA_HZ37:
        raise ValueError("La copia no procede del volcado HZ37 fijado")
    repo = Path(estado["fuente"])
    if repo != Path(__file__).resolve().parents[3]:
        raise ValueError("Fuentes de Git ajenas a esta copia")
    conexiones = json.loads((copia.raiz / "conexiones.json").read_text())
    previos = json.loads((copia.raiz / "roles-previos.json").read_text())
    if set(previos) != {"VEC_CT_DATABASE_URL", "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL"}:
        raise ValueError("Faltan los dos orígenes AD278")
    retirar = []
    for variable, dato in previos.items():
        viejo = urlsplit(conexiones[variable]).username
        if not viejo or not re.fullmatch(r"vcl_[a-z0-9_]+", viejo):
            raise ValueError("LOGIN temporal inesperado")
        if not re.fullmatch(r"vec_[a-z0-9_]+", dato["login"]):
            raise ValueError("LOGIN de origen inválido")
        retirar.append("DROP ROLE " + viejo + ";")
        conexiones[variable] = _dsn(dato["login"], dato["password"], copia, estado)
    logins = {clave: urlsplit(url).username for clave, url in conexiones.items()}
    if any(not valor or not re.fullmatch("[a-z0-9_]+", valor) for valor in logins.values()):
        raise ValueError("LOGIN inválido en conexiones")
    sql = ["BEGIN;", "SET LOCAL search_path=pg_catalog,pg_temp;",
           "SET LOCAL lock_timeout='5s';", "SET LOCAL statement_timeout='30s';",
           "DO $pre$ BEGIN IF current_user<>'postgres' OR current_database()<>'postgres' "
           "OR (SELECT system_identifier::text FROM pg_catalog.pg_control_system())<> '" + sid +
           "' THEN RAISE EXCEPTION 'Copia de datos ajena al conductor'; END IF; END $pre$;", *retirar]
    sql.append("DO $roles$ DECLARE r record; BEGIN FOR r IN SELECT rolname FROM pg_catalog.pg_roles "
               "WHERE NOT rolcanlogin AND rolname NOT LIKE 'pg_%' AND rolname NOT IN "
               "('vec_autorizacion_fuente','vec_autorizacion_motivos_evaluador') LOOP "
               "EXECUTE pg_catalog.format('ALTER ROLE %I NOINHERIT',r.rolname); "
               "END LOOP; END $roles$;")
    for grupo in ("vec_autorizacion_motivos_rrhh_resolutor", "vec_identidad_sesiones_v1_registrador",
                  "vec_identidad_sesiones_v1_revalidador", "vec_contexto_actor_v1_runtime",
                  "vec_contratacion_temporal_consultor_rrhh", "vec_autorizacion_registro"):
        sql.append("ALTER ROLE " + grupo + " NOINHERIT;")
    atributos = {}
    for archivo in sorted((repo / "deploy/postgresql").rglob("*up.sql")):
        if "pruebas_sql" in archivo.parts:
            continue
        for match in re.finditer(r"CREATE ROLE\s+(vec_[a-z0-9_]+)\s+(.*?);", archivo.read_text(), re.S):
            opciones = re.findall(r"\b(?:NOINHERIT|INHERIT)\b", match[2])
            if opciones:
                if match[1] in atributos and atributos[match[1]] != opciones[0]:
                    raise ValueError("Declaraciones de rol incompatibles")
                atributos[match[1]] = opciones[0]
    for rol, atributo in atributos.items():
        sql.append("DO $r$ BEGIN IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='" +
                   rol + "' AND NOT rolcanlogin) THEN ALTER ROLE " + rol + " " + atributo +
                   "; END IF; END $r$;")
    sql.append("COMMENT ON ROLE vec_autorizacion_motivos_rrhh_resolutor IS "
               "'vec_autorizacion:rol-motivos-rrhh-resolutor:v1';")
    # Excepción heredada: el contrato instalado del resolutor exige NULL.
    # Dirección la retirará cuando ese contrato admita una fecha de caducidad.
    sql.append("UPDATE pg_catalog.pg_authid SET rolvaliduntil=NULL WHERE rolname='" +
               logins["VEC_CT_MOTIVOS_RRHH_DATABASE_URL"] + "';")
    gov = logins["VEC_CT_GOBIERNO_DATABASE_URL"]
    sql.append("GRANT vec_contratacion_temporal_gobernador TO " + gov +
               " WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;")
    for grupo in ("vec_identidad_sesiones_v1_propietario", "vec_contexto_actor_v1_propietario",
                  "vec_autorizacion_propietario", "vec_autorizacion_motivos_proyector"):
        sql.append("GRANT " + grupo + " TO " + gov + " WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;")
    for tabla in ("puntero_clave_emision", "puntero_configuracion_actual"):
        sql.append("ALTER TABLE vec_autorizacion_atestada_v3." + tabla + " DISABLE TRIGGER ALL;")
    sql.append("TRUNCATE vec_autorizacion_atestada_v3.puntero_clave_emision,"
               "vec_autorizacion_atestada_v3.puntero_configuracion_actual;")
    for tabla in ("puntero_clave_emision", "puntero_configuracion_actual"):
        sql.append("ALTER TABLE vec_autorizacion_atestada_v3." + tabla + " ENABLE TRIGGER ALL;")
    sql.append("COMMIT;")
    copia.sql_bytes(("\n".join(sql) + "\n").encode(), "datos-copia-sql.log")
    _privado(copia.raiz / "conexiones.json", conexiones)
    _privado(copia.raiz / "entorno_modulos.json", conexiones)
    estado["entorno_modulos_sha256"] = _huella(copia.raiz / "entorno_modulos.json")
    _privado(copia.estado, estado)
    for archivo, nombre in (("identidad.json", "Carmen Molina Ortega"),
                            ("intervencion.json", "Luis Serrano Vega")):
        ruta = copia.material / "identidad" / archivo
        identidad = json.loads(ruta.read_text())
        identidad["display_name"] = nombre
        _privado(ruta, identidad)
    preparar_intentos(copia)
    preparar_bolsa(copia)
    from .limpieza_bolsa import preparar as limpiar_lote_versionado
    limpiar_lote_versionado(copia)
