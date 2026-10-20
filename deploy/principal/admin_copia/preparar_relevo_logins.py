#!/usr/bin/env python3
"""Prepara el relevo de cinco LOGIN ADMIN de la preimagen declarada."""
import argparse
import json
import os
import re
from pathlib import Path
from preparar_runtime import validar_copia, validar_entrada

GRUPOS = (
    ("vec_identidad_sesiones_v1_admin_perfiles_runtime", "cuentas_admin"),
    ("vec_contexto_actor_v1_admin_contexto", "contexto"),
    ("vec_identidad_sesiones_v1_admin_preperfil", "selector"),
    ("vec_admin_frontera_tecnica_ejecutor", "frontera_tecnica"),
    ("vec_admin_usuarios_lector", "lector"),
)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--run", required=True, type=Path)
    ap.add_argument("--sid-copia", required=True)
    ap.add_argument("--anteriores", required=True, type=Path, help="JSON privado con los cinco LOGIN de la preimagen")
    a = ap.parse_args()
    if not a.run.is_absolute():
        ap.error("destino inválido")
    logins = json.loads((a.run / "vec-admin/logins.json").read_text())
    e = json.loads((a.run / "entrada-vec-admin.json").read_text())
    try:
        validar_copia(a.run, Path(e["kit_dir"]), a.sid_copia)
        validar_entrada(e, a.run, a.sid_copia)
    except (ValueError, OSError, KeyError, json.JSONDecodeError) as exc:
        ap.error(str(exc))
    antiguos = json.loads(a.anteriores.read_text())
    if (set(antiguos) != {key for _, key in GRUPOS}
            or any(not re.fullmatch(r"[a-z][a-z0-9_]{5,63}", logins.get(key, ""))
                   or not re.fullmatch(r"[a-z][a-z0-9_]{5,63}", antiguos.get(key, ""))
                   or logins[key] == antiguos[key] for _, key in GRUPOS)):
        ap.error("preimagen o LOGIN nuevo no cotejable")
    sql = ["BEGIN;", f"DO $g$ BEGIN IF current_database()<>'{e['pg_base']}' OR (SELECT system_identifier FROM pg_control_system())<>{a.sid_copia} THEN RAISE EXCEPTION 'destino distinto'; END IF;"]
    for group, key in GRUPOS:
        old = antiguos[key]
        new = logins[key]
        sql.append(f"IF (SELECT count(*) FROM pg_auth_members a JOIN pg_roles g ON g.oid=a.roleid JOIN pg_roles m ON m.oid=a.member WHERE g.rolname='{group}' AND m.rolname IN ('{old}','{new}'))<>2 THEN RAISE EXCEPTION 'membresías ADMIN divergentes: {key}'; END IF;")
    sql.append("END $g$;")
    for group, key in GRUPOS:
        old = antiguos[key]
        sql += [f"REVOKE {group} FROM {old};", f"ALTER ROLE {old} NOLOGIN;"]
    sql.append("COMMIT;")
    out = a.run / "dba/relevo-logins-admin.sql"
    fd = os.open(out, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, "w") as f:
        f.write("\n".join(sql) + "\n")
        f.flush()
        os.fsync(f.fileno())
    print("Relevo ADMIN preparado: respaldar roles antes de ejecutar")


if __name__ == "__main__":
    main()
