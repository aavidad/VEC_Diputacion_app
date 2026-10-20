#!/usr/bin/env python3
"""Prepara LOGIN y configuración del gobierno ADMIN; no aplica ningún acto."""
import argparse
import datetime as dt
import json
import os
import re
import secrets
from pathlib import Path

from preparar_runtime import validar_copia, validar_entrada, validar_gobierno


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
    ap.add_argument("--fase", required=True, choices=("entrada", "configurar"))
    ap.add_argument("--run", required=True, type=Path)
    ap.add_argument("--dia", required=True)
    ap.add_argument("--sid-copia", required=True)
    a = ap.parse_args()
    if not re.fullmatch(r"[0-9]{8}[a-z0-9]{1,12}", a.dia):
        ap.error("día inválido")
    e = json.loads((a.run / "entrada-vec-admin.json").read_text())
    if e.get("dia_gobierno") != a.dia:
        ap.error("día de gobierno ajeno a la entrada")
    kit = Path(e["kit_dir"])
    try:
        validar_copia(a.run, kit, a.sid_copia)
        validar_entrada(e, a.run, a.sid_copia)
    except (ValueError, OSError, KeyError, json.JSONDecodeError) as exc:
        ap.error(str(exc))
    cfg = json.loads((a.run / "config.json").read_text())
    contrato = json.loads((kit / "contrato-copia-local-admin-v1.json").read_text())
    gov = a.run / "gobierno" / a.dia
    sufijo = e["sufijo_login"]
    if not re.fullmatch(r"[a-z0-9_]{3,18}", sufijo):
        ap.error("sufijo LOGIN inválido")
    read_login = "vec_adm_gobread_" + sufijo
    op_login = "vec_adm_gobcap_" + sufijo
    if a.fase == "entrada":
        gov.mkdir(parents=True, mode=0o700)
        roles = [(read_login, "vec_autorizacion_atestada_v3_lector_gobierno"),
                 (op_login, "vec_gobierno_capacidades_admin_operador")]
        lines = ["BEGIN;"]
        dsns = {}
        for login, group in roles:
            password = secrets.token_urlsafe(32)
            lines += [f"CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '{password}' VALID UNTIL '{(dt.datetime.now(dt.timezone.utc)+dt.timedelta(days=90)).isoformat()}';",
                      f"GRANT {group} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;",
                      f"GRANT CONNECT ON DATABASE {e['pg_base']} TO {group};"]
            dsns[login] = f"host={e['pg_host']} port={e['pg_port']} dbname={e['pg_base']} user={login} password={password} sslmode=verify-full sslrootcert={e['pg_ca']}"
        lines.append("COMMIT;")
        escribir(a.run / "dba" / ("roles-gobierno-" + a.dia + ".sql"), "\n".join(lines) + "\n")
        g = {"directorio_material": cfg["proveedor"]["directorio_material"],
             "ruta_configuracion_hmac": cfg["proveedor"]["ruta_configuracion_hmac"],
             "archivo_semilla_raiz": e["semilla_raiz"], "dsn_lectura": dsns[read_login], "dsn_operador": dsns[op_login],
             "salida": str(gov), "horas_validez_claves": contrato["vigencia_dias"] * 24, "conjunto_capacidades": 7}
        escribir(a.run / ("config-gobierno-" + a.dia + ".json"), g)
        print("Gobierno preparado: instalar roles y ejecutar fase preparar del CLI")
        return
    try:
        candidate, _, _ = validar_gobierno(a.run, a.dia)
    except (ValueError, OSError, KeyError, json.JSONDecodeError) as exc:
        ap.error(str(exc))
    if any(not re.fullmatch(r"[0-9a-f]{64}", candidate.get(k, "")) for k in ("plan_sha256", "preimagen_sha256", "material_sha256")) or not re.fullmatch(r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+Z", candidate.get("caduca_en", "")):
        ap.error("plan no cotejable")
    sql = ("BEGIN;\n" + f"DO $g$ BEGIN IF current_database()<>'{e['pg_base']}' OR (SELECT system_identifier FROM pg_control_system())<>{a.sid_copia} THEN RAISE EXCEPTION 'destino distinto'; END IF; END $g$;\n"
           "SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;\n"
           "INSERT INTO vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1(identidad_login,conjunto_version,plan_sha256,preimagen_sha256,material_sha256,vigente_desde,vigente_hasta,entorno)\n"
           f"VALUES('{op_login}',7,'{candidate['plan_sha256']}','{candidate['preimagen_sha256']}','{candidate['material_sha256']}',clock_timestamp()-interval '1 minute','{candidate['caduca_en']}'::timestamptz,'{e['entorno']}');\nCOMMIT;\n")
    escribir(a.run / "dba" / ("config-gobierno-" + a.dia + ".sql"), sql)
    print("Configuración del gobierno preparada para el plan exacto")


if __name__ == "__main__":
    main()
