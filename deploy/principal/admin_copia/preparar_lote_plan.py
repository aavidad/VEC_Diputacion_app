#!/usr/bin/env python3
"""Prepara pools privados del lote y del plan del gobierno ADMIN conjunto 7."""
import argparse
import json
import os
import re
import secrets
from pathlib import Path
from preparar_runtime import validar_copia, validar_entrada, validar_gobierno


def leer(path):
    return json.loads(Path(path).read_text())


def escribir(path, value):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "w") as f:
        if isinstance(value, str):
            f.write(value)
        else:
            json.dump(value, f, ensure_ascii=False, indent=1)
            f.write("\n")
        f.flush()
        os.fsync(f.fileno())


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--run", required=True, type=Path)
    ap.add_argument("--dia", required=True)
    ap.add_argument("--sid-copia", required=True)
    a = ap.parse_args()
    if not a.run.is_absolute() or not re.fullmatch(r"[0-9]{8}[a-z0-9]{1,12}", a.dia):
        ap.error("parámetros inválidos")
    e = leer(a.run / "entrada-vec-admin.json")
    if e.get("dia_gobierno") != a.dia:
        ap.error("día de gobierno ajeno a la entrada")
    v = a.run / "vec-admin"
    c = v / "config"
    overlay = leer(c / ("usuarios-" + a.dia + ".json"))
    try:
        validar_copia(a.run, Path(e["kit_dir"]), a.sid_copia)
        validar_entrada(e, a.run, a.sid_copia)
        _, mat, _ = validar_gobierno(a.run, a.dia, aplicado=True)
    except (ValueError, OSError, KeyError, json.JSONDecodeError) as exc:
        ap.error(str(exc))
    if len(overlay["confianza"]["entradas_capacidad"]) != 2:
        ap.error("gobierno conjunto 7 u overlay de usuarios incompletos")
    sid = a.sid_copia
    suffix = e["sufijo_login"]
    if not re.fullmatch(r"[a-z0-9_]{3,18}", suffix):
        ap.error("sufijo LOGIN inválido")
    users = [("lote", "vec_adm_lote_" + suffix, "vec_admin_perfiles_lote_ejecutor"),
             ("plan_firma", "vec_adm_plan_firma_" + suffix, "vec_plan_firma_gobierno_ejecutor")]
    sql = ["BEGIN;", f"DO $g$ BEGIN IF current_database()<>'{e['pg_base']}' OR (SELECT system_identifier FROM pg_control_system())<>{sid} THEN RAISE EXCEPTION 'destino distinto'; END IF; END $g$;"]
    for name, login, group in users:
        pw = secrets.token_urlsafe(32)
        dsn = f"host={e['pg_host']} port={e['pg_port']} dbname={e['pg_base']} user={login} password={pw} sslmode=verify-full sslrootcert={e['pg_ca']}"
        escribir(v / "pools" / (name + ".json"), {"dsn": dsn})
        sql += [f"CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '{pw}';",
                f"GRANT {group} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;",
                f"GRANT CONNECT ON DATABASE {e['pg_base']} TO {group};"]
    sql.append("SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;")
    audience_lote = "vec_autorizacion.administracion_perfiles.lote_ordinario.v1"
    audience_plan = "vec_catalogos_configurables.plan_nominal_firma.gobierno.v1"
    if not re.fullmatch(r"[a-z][a-z0-9._-]{1,79}", e["procesos"]["usuarios"]):
        ap.error("proceso de usuarios inválido")
    sql.append(f"INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) VALUES('{users[0][1]}','{audience_lote}','administracion.perfiles.aplicar_lote_ordinario','{e['procesos']['usuarios']}','administracion_privilegiada');")
    for op in ("crear", "actualizar", "publicar", "retirar"):
        sql.append(f"INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) VALUES('{users[1][1]}','{audience_plan}','vec.catalogos.{op}','vec-admin','administracion_privilegiada');")
    sql.append("COMMIT;")
    escribir(a.run / "dba" / ("lote-plan-" + a.dia + ".sql"), "\n".join(sql) + "\n")

    def entrada(aud):
        x = next((x for x in mat["claves"] if x["audiencia"] == aud), None)
        if x is None:
            ap.error("audiencia ausente del gobierno")
        return {"audiencia": aud, "clave_id": x["clave_id"], "emisor_id": x["emisor_id"],
                "huella_gobierno": x["huella_gobierno_sha256"], "version": x["version"],
                "revision_gobierno": x["revision_gobierno"],
                "material_archivo": str(v / "claves" / a.dia / (aud + ".bin")), "estado": "emision",
                "valida_desde": x["valida_desde"], "valida_hasta": x["valida_hasta"], "revocada_en": "0001-01-01T00:00:00Z"}

    meta = overlay["confianza"]
    base = {k: meta[k] for k in ("cabecera", "raiz", "gobierno")}
    plan = leer(a.run / "bootstrap/plan-v3.json")["plan"]["personas"][0]
    amb = plan["ambitos"]
    if len(amb) != 2 or amb[0]["dimension"] != "organizacion_ref" or amb[1]["dimension"] != "unidad_ref":
        ap.error("ámbito del bootstrap no cotejable")
    motivo = e["motivo"]
    unit = {"unidad_ref": amb[1]["valores"][0], "fuente_organizacion": amb[0]["fuente"], "fuente_unidad": amb[1]["fuente"]}
    escribir(c / "lote.json", {"modo": "lote_v1", "pool_lote": str(v / "pools/lote.json"),
           "confianza": {**base, "entradas_capacidad": [entrada(audience_lote)]}, "motivo_lote": motivo,
           "unidades": [unit], "motivos_cambio": [{"motivo": motivo, "clave_i18n": "alta_funciones"}]})
    escribir(c / "plan-firma.json", {"pool": str(v / "pools/plan_firma.json"),
           "confianza": {**base, "entradas_capacidad": [entrada(audience_plan)]}, "motivo": motivo})
    overlay["destinos"]["preparar_lote_ordinario"] = {"accion": "administracion.perfiles.frontera.preparar_lote_ordinario",
          "recurso_ref": "administracion:perfiles:preparar_lote_ordinario", "finalidad_ref": "gestion_perfiles", "tipo_recurso": "fijo"}
    usuarios = c / ("usuarios-" + a.dia + "-lote.json")
    escribir(usuarios, overlay)
    env = (c / ("entorno-" + a.dia + ".env")).read_text().replace(str(c / ("usuarios-" + a.dia + ".json")), str(usuarios))
    env += "VEC_ADMIN_LOTE_CONFIG_FILE=" + str(c / "lote.json") + "\n"
    env += "VEC_ADMIN_PLAN_FIRMA_CONFIG_FILE=" + str(c / "plan-firma.json") + "\n"
    escribir(c / ("entorno-" + a.dia + "-lote-plan.env"), env)
    print("Pools y configuración escalonada preparados; aplicar SQL y ensayar arranque aparte")


if __name__ == "__main__":
    main()
