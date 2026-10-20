#!/usr/bin/env python3
"""Guarda antes del ajuste la preimagen de las tablas ADMIN ficticias de HZ37."""
import argparse
import os
import re
import subprocess
from pathlib import Path

TABLAS = (
    "vec_autorizacion.asignacion_perfil_actual",
    "vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1",
    "vec_identidad_sesiones_v1.vinculo_certificado_admin_v1",
    "vec_identidad_sesiones_v1.politica_certificado_admin_v1",
    "vec_identidad_sesiones_v1.titularidad_cuenta_persona_v1",
    "vec_contexto_actor_v1.titularidad_cuenta_persona_v1",
    "vec_autorizacion.fuentes_iniciales_admin_v1",
    "vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1",
    "vec_contexto_actor_v1.fuentes_iniciales_admin_v1",
    "vec_autorizacion.outbox_bootstrap_central_admin_v3",
    "vec_autorizacion.bootstrap_central_admin_v3",
    "vec_autorizacion.control_continuidad_admin",
    "vec_autorizacion.rol_administrable_exacto_v1",
    "vec_personal.unidad_inicial_admin_v1",
)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--contenedor", required=True)
    ap.add_argument("--sid-copia", required=True, help="SID PostgreSQL cotejado por el conductor")
    ap.add_argument("--salida", required=True, type=Path)
    a = ap.parse_args()
    if (not re.fullmatch(r"[a-z0-9-]{6,80}", a.contenedor) or
        not re.fullmatch(r"[0-9]{15,20}", a.sid_copia) or
        not a.salida.is_absolute() or a.salida.exists() or
        a.salida.parent.is_symlink() or a.salida.parent.stat().st_mode & 0o077):
        ap.error("destino o salida no privada")
    env = {"PATH": "/usr/bin:/bin", "LANG": "C",
           "HOME": str(a.salida.parent),
           "DOCKER_CONFIG": str(a.salida.parent / ".docker-sin-contexto"),
           "DOCKER_HOST": "unix:///var/run/docker.sock"}
    sid = subprocess.run(["/usr/bin/docker", "exec", "-i", a.contenedor,
                          "psql", "-X", "-At", "-v", "ON_ERROR_STOP=1",
                          "-U", "postgres", "-d", "postgres", "-c",
                          "SELECT system_identifier FROM pg_control_system()"],
                         capture_output=True, text=True, check=True,
                         timeout=30, env=env).stdout.strip()
    if sid != a.sid_copia:
        ap.error("SID de la copia divergente")
    cmd = ["/usr/bin/docker", "exec", "-i", a.contenedor, "pg_dump", "-U", "postgres", "-d", "postgres", "-Fc", "--data-only"]
    for table in TABLAS:
        cmd += ["-t", table]
    fd = os.open(a.salida, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
    try:
        with os.fdopen(fd, "wb") as f:
            result = subprocess.run(cmd, stdout=f, stderr=subprocess.PIPE,
                                    timeout=120, env=env)
    except Exception:
        a.salida.unlink(missing_ok=True)
        raise
    if result.returncode != 0 or a.salida.stat().st_size < 100:
        a.salida.unlink()
        ap.error("pg_dump no confirmó la preimagen")
    print("Preimagen ADMIN guardada en archivo privado")


if __name__ == "__main__":
    main()
