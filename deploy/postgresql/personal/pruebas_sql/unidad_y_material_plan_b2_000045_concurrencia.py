#!/usr/bin/env python3
"""Dos sesiones PostgreSQL reales en un clon propio desechable, sin V3 simulado."""
import os
import re
import selectors
import subprocess
import sys
import time

CONTAINER = sys.argv[1] if len(sys.argv) == 2 else "vec-codexh-p45-r2-pg18"
if not re.fullmatch(r"vec-codexh-p45-[a-z0-9-]+pg18", CONTAINER):
    raise SystemExit("Se requiere el nombre del clon propio desechable P45")
CMD = ["docker", "exec", "-i", CONTAINER, "psql", "-X", "-qAt", "-v",
       "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate", "-U", "postgres", "-d", "postgres"]
ENV = {"PATH": "/usr/bin:/bin", "LANG": "C.UTF-8"}


def query(sql):
    result = subprocess.run(CMD, input=sql.encode(), stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, timeout=20, env=ENV)
    if result.returncode:
        raise RuntimeError(result.stderr.decode())
    return result.stdout.decode().strip()


class Session:
    def __init__(self, name):
        self.proc = subprocess.Popen(CMD, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                     stderr=subprocess.STDOUT, env=ENV)
        self.output = b""
        self.send("SET application_name='" + name + "';")

    def send(self, sql):
        self.proc.stdin.write((sql + "\n").encode())
        self.proc.stdin.flush()

    def until(self, marker, seconds=5):
        deadline = time.monotonic() + seconds
        with selectors.DefaultSelector() as sel:
            sel.register(self.proc.stdout, selectors.EVENT_READ)
            while marker.encode() not in self.output:
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise RuntimeError("Tiempo agotado: " + self.output.decode())
                if not sel.select(remaining):
                    continue
                chunk = os.read(self.proc.stdout.fileno(), 4096)
                if not chunk:
                    raise RuntimeError("Sesión terminada: " + self.output.decode())
                self.output += chunk
        return self.output.decode()

    def close(self):
        if self.proc.poll() is None:
            self.proc.stdin.close()
            try:
                self.proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.proc.terminate()
                self.proc.wait(timeout=5)


expected = "9ba1f3c54a9140a2dfe6dca5ee6a55637f6a0deafb4e8dbbff88a4e82be2d258"
function = "vec_personal.registrar_acto_empleado_b2_interna(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
actual = query("SELECT encode(sha256(convert_to(pg_get_functiondef('" + function + "'::regprocedure),'UTF8')),'hex');")
if actual != expected:
    raise SystemExit("Postimagen P45 distinta")
if query("SELECT count(*) FROM vec_personal.org_nodo_historia WHERE acto_ref='acto:p45:concurrencia';") != "0":
    raise SystemExit("El clon ya conserva este ensayo; usar otra copia anterior al ensayo")
source = query("SELECT prosrc FROM pg_proc WHERE oid='" + function + "'::regprocedure;")
start = source.index("  PERFORM c.generacion FROM vec_personal.control_unidad_bootstrap_admin_v1 c")
end = source.index("  -- Personal45: solo un nodo publicado actual", start)
# Ejecuta exactamente el segmento instalado. No sustituye funciones, consumidores
# ni atestación V3; esta prueba acredita la coordinación de la barrera, no un alta.
lock = "DO $bloqueo$ BEGIN\n" + source[start:end] + "END $bloqueo$;"
settings = "SET LOCAL ROLE vec_personal_propietario; SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='15s';"
insert = """INSERT INTO vec_personal.org_nodo_historia
(nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,
catalogo_revision,catalogo_entrada_clave,denominacion,retirado,vigente_desde,
conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
VALUES(gen_random_uuid(),1,'org:p45:concurrencia','uni:p45:concurrencia','centro',
'estructura-organizativa-dipgra',1,1,'entrada:p45:__N__','Unidad sintética P45',false,
'2026-01-01',clock_timestamp(),'fuente:p45:concurrencia','acto:p45:concurrencia',repeat('a',64));"""
sessions = []
try:
    reader = Session("p45:lectorA"); sessions.append(reader)
    reader.send("BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE; " + settings +
                " SELECT generacion FROM vec_personal.control_unidad_bootstrap_admin_v1 WHERE singleton;\n\\echo SNAPSHOT_A")
    reader.until("SNAPSHOT_A")
    query("BEGIN; " + settings + insert.replace("__N__", "a") + " COMMIT;")
    reader.send(lock)
    error = reader.until("ERROR:")
    if "40001" not in error:
        raise RuntimeError("La publicación posterior a la instantánea no dio 40001: " + error)
    print("P45_CONCURRENCIA_A_OK: publicación confirmada después de snapshot -> 40001")
    reader.close()

    reader = Session("p45:lectorB"); sessions.append(reader)
    reader.send("BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE; " + settings + lock + "\n\\echo SHARE_B")
    reader.until("SHARE_B")
    publisher = Session("p45:publicadorB"); sessions.append(publisher)
    publisher.send("BEGIN; " + settings + insert.replace("__N__", "b") + " COMMIT;\n\\echo PUBLICADO_B")
    deadline = time.monotonic() + 1.5
    while True:
        blocked = query("SELECT count(*) FROM pg_stat_activity p JOIN pg_stat_activity r ON r.application_name='p45:lectorB' "
                        "WHERE p.application_name='p45:publicadorB' AND p.wait_event_type='Lock' AND r.pid=ANY(pg_blocking_pids(p.pid));")
        if blocked == "1":
            break
        if time.monotonic() >= deadline:
            raise RuntimeError("El publicador no esperó al lector con FOR SHARE")
        time.sleep(0.02)
    reader.send("COMMIT;\n\\echo COMMIT_B")
    reader.until("COMMIT_B")
    publisher.until("PUBLICADO_B")
    if query("SELECT count(*) FROM vec_personal.org_nodo_historia WHERE acto_ref='acto:p45:concurrencia';") != "2":
        raise RuntimeError("No se conservaron las dos publicaciones sintéticas")
    print("P45_CONCURRENCIA_B_OK: publicador bloqueado por lector; publica después de COMMIT")
    print("P45_CONCURRENCIA_OK; dos publicaciones sintéticas conservadas solo en este clon")
finally:
    for session in sessions:
        session.close()
