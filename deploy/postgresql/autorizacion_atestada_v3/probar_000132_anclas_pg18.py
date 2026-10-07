#!/usr/bin/env python3
"""Comprueba en PostgreSQL 18 que un OID de rol no altera el ancla CT145."""
import json
from pathlib import Path
import secrets
import subprocess
import time


QUERY = Path(__file__).with_name("000132_anclas_h6.sql").read_bytes()
DOCKER = ["docker", "--host", "unix:///var/run/docker.sock"]
FIXTURE = b"""
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN;
CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
CREATE TABLE vec_contratacion_temporal.firma_documento_custodia_v1(id integer PRIMARY KEY);
ALTER TABLE vec_contratacion_temporal.firma_documento_custodia_v1 ENABLE ROW LEVEL SECURITY;
CREATE POLICY ejercicio ON vec_contratacion_temporal.firma_documento_custodia_v1
  TO vec_contratacion_temporal_propietario USING (id>0) WITH CHECK (id>0);
CREATE POLICY publico ON vec_contratacion_temporal.firma_documento_custodia_v1
  TO PUBLIC USING (false);
"""


def command(*args, data=None):
    result = subprocess.run([*DOCKER, *args], input=data, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, timeout=30, check=False)
    if result.returncode:
        raise RuntimeError(f"Docker/PostgreSQL falló: código {result.returncode}")
    return result.stdout.strip()


def psql(name, sql):
    return command("exec", "-i", "--user", "postgres", name,
                   "psql", "-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1",
                   "-U", "postgres", "-d", "postgres", data=sql)


def start(name):
    command("run", "-d", "--rm", "--network", "none", "--shm-size", "64m",
            "--memory", "512m", "--cpus", "1", "--pids-limit", "64",
            "--name", name, "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "postgres:18.4")
    for _ in range(100):
        ready = subprocess.run([*DOCKER, "exec", name, "pg_isready", "-U", "postgres",
                                "-d", "postgres"], stdout=subprocess.DEVNULL,
                               stderr=subprocess.DEVNULL, timeout=5, check=False)
        if ready.returncode == 0:
            return
        time.sleep(0.1)
    raise RuntimeError("PostgreSQL 18 no arrancó a tiempo")


def anchor(name):
    return json.loads(psql(name, QUERY).splitlines()[-1])["ct145_table"]


def main():
    nonce = secrets.token_hex(4)
    first, second = (f"vec-ad132-oid-{nonce}-{suffix}" for suffix in ("a", "b"))
    started = []
    try:
        for name in (first, second):
            start(name)
            started.append(name)
        psql(first, FIXTURE)
        psql(second, b"CREATE ROLE relleno_antes NOLOGIN;\n" + FIXTURE)
        oid_sql = b"SELECT oid FROM pg_catalog.pg_roles WHERE rolname='vec_contratacion_temporal_propietario';"
        if psql(first, oid_sql) == psql(second, oid_sql):
            raise RuntimeError("fixture no desplazó el OID del mismo rol")
        before = anchor(first)
        if before != anchor(second):
            raise RuntimeError("ancla depende del OID del rol")
        psql(second, b"ALTER POLICY ejercicio ON vec_contratacion_temporal.firma_documento_custodia_v1 TO relleno_antes;")
        if before == anchor(second):
            raise RuntimeError("ancla ignoró un cambio de privilegio de política")
        print("ANCLA-CT145-OID-OK: mismo nombre con OID distinto conserva huella; otro rol la cambia")
    finally:
        for name in started:
            subprocess.run([*DOCKER, "stop", name], stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL, timeout=30, check=False)


if __name__ == "__main__":
    main()
