#!/usr/bin/env python3
"""Prepara SQL de DBA para AUT53 desde plan y aprobación reales de la copia."""
import argparse
import hashlib
import json
import os
import re
from pathlib import Path


def source(value):
    p = Path(value)
    if not p.is_absolute() or not p.is_file() or p.is_symlink() or p.stat().st_mode & 0o077 or p.stat().st_uid != os.getuid():
        raise ValueError("archivo privado requerido")
    return p.read_bytes()


def main():
    ap = argparse.ArgumentParser()
    for key in ("plan", "aprobacion", "aprobacion-ref", "login", "salida"):
        ap.add_argument("--" + key, required=True)
    a = ap.parse_args()
    os.umask(0o077)
    if not re.fullmatch(r"vec_[a-z0-9_]{3,55}", a.login) or not re.fullmatch(r"aprobacion:[A-Za-z0-9:._-]{8,120}", a.aprobacion_ref):
        raise ValueError("LOGIN o referencia de aprobación inválidos")
    plan_raw, approval_raw = source(a.plan), source(a.aprobacion)
    plan, approval = json.loads(plan_raw), json.loads(approval_raw)
    plan_sha = hashlib.sha256(plan_raw).hexdigest()
    if approval.get("huella_plan_sha256") != plan_sha or plan.get("esquema") != "vec.admin.cargos-firma.plan.v2" or len(plan.get("cargos", [])) != 16:
        raise ValueError("aprobación o plan AUT53 divergente")
    if not approval.get("alcance") or not approval.get("orden"):
        raise ValueError("aprobación sin alcance y orden")
    output = Path(a.salida)
    if not output.is_absolute() or output.exists() or output.is_symlink():
        raise ValueError("salida nueva absoluta requerida")
    approval_sha = hashlib.sha256(approval_raw).hexdigest()
    sql = ("BEGIN;\n" +
           f"CREATE ROLE {a.login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;\n" +
           f"GRANT vec_admin_cargos_firma_ejecutor TO {a.login} WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;\n" +
           "GRANT CONNECT ON DATABASE postgres TO vec_admin_cargos_firma_ejecutor;\n" +
           "INSERT INTO vec_autorizacion.config_cargos_firma_admin_v1 VALUES(" +
           f"'{a.login}','{plan_sha}','{a.aprobacion_ref}','{approval_sha}','desarrollo'," +
           "clock_timestamp()-interval '1 minute',clock_timestamp()+interval '2 hours');\nCOMMIT;\n")
    with output.open("xb") as f:
        f.write(sql.encode())
        f.flush()
        os.fsync(f.fileno())
    output.chmod(0o600)
    print("Configuración AUT53 preparada; DBA debe revisar LOGIN y aplicarla en P")


if __name__ == "__main__":
    main()
