#!/usr/bin/env python3
"""Instala el SQL H3, H4 y main sobre un clon H1 propiedad de Codex-M.

H1 se restaura antes; H5 solo configura la aplicación. No instala SQL de ramas
pendientes ni ejecuta DOWN. --plan valida todos los SHA sin acceder a Docker.
El journal privado se reconstruye desde recibos transaccionales del clon.
Admite la base main@7f1ecea2f (33 SQL) y su extensión main@ff6493cfc
(CT147, posición 34). La extensión conserva los recibos y metadatos originales
y añade una revisión del plan en el esquema del clon. Otros hashes exigen revisar
de nuevo la lista causal y sus huellas, aunque el SQL parezca igual.
"""

import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import uuid

BASE_REF = "7f1ecea2fd9f8912d255a80e74da84c69e46b978"
MAIN_REF = "ff6493cfccb2da4e83c94fa7c59be24c025cb7c9"
REF_COUNTS = {BASE_REF: 33, MAIN_REF: 34}
REF_PLAN_SHA = {
    BASE_REF: "70795c1580e550e2ccc8927bf50cf7130f73ca282d6069f74ba7e697f79e6be0",
    MAIN_REF: "00d8dbaacd881a6945a33dd188e94e136b054f4e96a8ac29bdcb039637fc891c",
}
OWNER_LABEL = "vec.recorridos.owner"
OWNER = "Codex-M"
SCHEMA = "vec_recorridos_clon"
MANIFEST = Path(__file__).with_name("sql_main.txt")


class Refused(RuntimeError):
    pass


def sha(data):
    return hashlib.sha256(data).hexdigest()


def sql_literal(value):
    return "'" + value.replace("'", "''") + "'"


def load_plan(repo, manifest=MANIFEST, source_ref=MAIN_REF):
    """Congela los bytes antes de escribir; un árbol alterado falla completo."""
    rows = []
    if source_ref not in REF_COUNTS:
        raise Refused("el plan solo corresponde a los hashes main fijados")
    for line in manifest.read_text().splitlines():
        if not line.strip() or line.startswith("#"):
            continue
        if len(rows) == REF_COUNTS[source_ref]:
            break
        phase, digest, relative = line.split()
        if phase not in {"H3", "H4", "MAIN"} or not re.fullmatch(r"[a-f0-9]{64}", digest):
            raise Refused("manifiesto incompatible")
        if not re.fullmatch(r"deploy/postgresql/[a-z0-9_/]+(?:\.up\.sql|_up\.sql)", relative):
            raise Refused("ruta SQL incompatible")
        path = repo / relative
        if not path.resolve().is_relative_to(repo.resolve()) or path.is_symlink():
            raise Refused("SQL fuera del árbol fuente")
        data = path.read_bytes()
        if sha(data) != digest:
            raise Refused(f"SHA incompatible: {relative}")
        text = data.decode("utf-8")
        if len(re.findall(r"(?m)^BEGIN;\s*$", text)) != 1 or len(re.findall(r"(?m)^COMMIT;\s*$", text)) != 1:
            raise Refused(f"se exige una transacción explícita: {relative}")
        if not re.search(r"COMMIT;\s*$", text):
            raise Refused(f"COMMIT debe cerrar el fichero: {relative}")
        if any(row["path"] == relative for row in rows):
            raise Refused("SQL duplicada en el plan")
        rows.append({"phase": phase, "sha256": digest, "path": relative, "sql": text})
    if len(rows) != REF_COUNTS[source_ref]:
        raise Refused("número de SQL incompatible con el hash main fijado")
    if plan_hash(rows) != REF_PLAN_SHA[source_ref]:
        raise Refused("manifiesto incompatible con el hash main fijado")
    return rows


def plan_hash(rows):
    return sha(json.dumps([{k: v for k, v in row.items() if k != "sql"}
                           for row in rows], sort_keys=True).encode())


class DockerDB:
    def __init__(self, container, state=None):
        if not re.fullmatch(r"vec-[a-z0-9-]+", container):
            raise Refused("el nombre debe identificar un clon vec- propio")
        self.container = container
        self.state = state

    def check_owner(self):
        result = subprocess.run(["docker", "inspect", self.container],
                                capture_output=True, text=True, check=True, timeout=15)
        obj = json.loads(result.stdout)[0]
        labels = obj["Config"].get("Labels") or {}
        if labels.get(OWNER_LABEL) != OWNER or not obj["State"]["Running"]:
            raise Refused("el contenedor no es un clon activo propiedad de Codex-M")
        if self.state is not None and labels.get("vec.recorridos.state") != str(self.state):
            raise Refused("la label de estado del clon no corresponde al directorio privado")
        mounts = obj.get("Mounts", [])
        if not any(m["Type"] == "bind" and m["Destination"] == "/var/lib/postgresql"
                   and m["Source"].startswith("/dev/shm/") for m in mounts):
            raise Refused("el clon debe usar una restauración desechable en /dev/shm")
        if obj["Config"].get("Image") != "postgres:18.4":
            raise Refused("se exige postgres:18.4")
        ports = obj.get("NetworkSettings", {}).get("Ports") or {}
        if any(binding.get("HostIp") not in {"127.0.0.1", "::1"}
               for bindings in ports.values() for binding in (bindings or [])):
            raise Refused("los puertos del clon solo pueden publicarse en loopback")

    def query(self, text):
        # No shell, DSN ni entorno de autenticación. La salida SQL nunca se imprime.
        result = subprocess.run(
            ["docker", "exec", "-i", self.container, "psql", "-h", "/var/run/postgresql",
             "-p", "5432", "-U", "postgres", "-d",
             "postgres", "-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1",
             "-v", "VERBOSITY=verbose"],
            input=text, capture_output=True, text=True, timeout=180)
        if result.returncode:
            # PostgreSQL puede incluir filas o SQL literal en DETAIL/CONTEXT.
            # Conservar solo SQLSTATE; nunca volcar stderr con material privado.
            codes = re.findall(r"(?:ERROR|FATAL):\s+([0-9A-Z]{5}):", result.stderr)
            suffix = f" SQLSTATE={codes[0]}" if codes else ""
            raise Refused("PostgreSQL rechazó la operación" + suffix)
        return result.stdout.strip()


def initialize(db, rows, source_ref=MAIN_REF):
    expected = plan_hash(rows)
    exists = db.query(f"SELECT to_regnamespace('{SCHEMA}') IS NOT NULL;")
    if exists == "f":
        # Admitir solo la preimagen H1; un clon H3/H4 sin recibos no se reejecuta.
        baseline = db.query("""SELECT
          to_regprocedure('vec_contratacion_temporal.gobi_o404b_material_catalogo_v2(jsonb)') IS NULL
          AND to_regclass('vec_contratacion_temporal.numeracion_anual_asignada') IS NULL
          AND NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
            ('vec_usuarios_propietario','vec_aspirantes_propietario'))
          AND EXISTS (SELECT 1 FROM pg_proc WHERE oid = to_regprocedure(
            'vec_contratacion_temporal.gobi_o404b_material_catalogo(jsonb)')
            AND encode(sha256(convert_to(prosrc,'UTF8')),'hex') =
            'daac1fec7f04618a41337ea0d6d6bff494178da52dcbd51e84f50ab0868c15c6');""")
        if baseline != "t":
            raise Refused("preimagen H1 incompatible; no reaplicar SQL ya instalada")
        run_id = str(uuid.uuid4())
        db.query(f"""BEGIN;
          CREATE SCHEMA {SCHEMA} AUTHORIZATION postgres;
          REVOKE ALL ON SCHEMA {SCHEMA} FROM PUBLIC;
          CREATE TABLE {SCHEMA}.plan (
            singleton boolean PRIMARY KEY CHECK (singleton), run_id uuid NOT NULL,
            source_ref text NOT NULL, plan_sha text NOT NULL);
          CREATE TABLE {SCHEMA}.applied (
            position integer PRIMARY KEY, path text NOT NULL UNIQUE,
            sha256 text NOT NULL, installed_at timestamptz NOT NULL DEFAULT clock_timestamp());
          REVOKE ALL ON ALL TABLES IN SCHEMA {SCHEMA} FROM PUBLIC;
          INSERT INTO {SCHEMA}.plan VALUES (true, '{run_id}', '{source_ref}', '{expected}');
          COMMIT;""")
    raw = db.query(f"""SELECT json_build_object('run_id',run_id,'source_ref',source_ref,
                    'plan_sha',plan_sha) FROM {SCHEMA}.plan WHERE singleton;""")
    meta = json.loads(raw)
    return acknowledge_plan(db, rows, source_ref, meta)


def acknowledge_plan(db, rows, source_ref, meta):
    """Preserva plan original; la única ampliación admitida es su prefijo exacto."""
    original_count = REF_COUNTS.get(meta["source_ref"])
    if original_count is None or original_count > len(rows):
        raise Refused("el clon conserva otro plan; requiere revisión del inventario")
    if meta["plan_sha"] != plan_hash(rows[:original_count]):
        raise Refused("el prefijo del plan original ha cambiado; no instalar")
    exists = db.query(f"SELECT to_regclass('{SCHEMA}.plan_revisions') IS NOT NULL;")
    revisions = []
    if exists == "t":
        revisions = json.loads(db.query(f"""SELECT coalesce(json_agg(x ORDER BY revision), '[]'::json)
          FROM (SELECT revision,source_ref,plan_sha,file_count,acknowledged_at
                FROM {SCHEMA}.plan_revisions) x;"""))
        if len(revisions) != 1 or revisions[0]["revision"] != 2 or (
                revisions[0]["source_ref"] != MAIN_REF or len(rows) != 34
                or revisions[0]["plan_sha"] != plan_hash(rows)
                or revisions[0]["file_count"] != 34 or meta["source_ref"] != BASE_REF):
            raise Refused("revisiones de plan incompatibles; no instalar ni volver a la base")
    if meta["source_ref"] != source_ref and not revisions:
        if meta["source_ref"] != BASE_REF or source_ref != MAIN_REF or len(rows) != 34:
            raise Refused("extensión de plan no autorizada")
        if len(receipts(db, rows)) != original_count:
            raise Refused("la extensión requiere las 33 SQL originales completas")
        db.query(f"""BEGIN;
          SELECT pg_advisory_xact_lock(hashtextextended('vec_recorridos_clon:sql',0));
          CREATE TABLE {SCHEMA}.plan_revisions (
            revision integer PRIMARY KEY CHECK (revision=2), source_ref text NOT NULL,
            plan_sha text NOT NULL, file_count integer NOT NULL CHECK (file_count=34),
            acknowledged_at timestamptz NOT NULL DEFAULT clock_timestamp());
          REVOKE ALL ON {SCHEMA}.plan_revisions FROM PUBLIC;
          INSERT INTO {SCHEMA}.plan_revisions(revision,source_ref,plan_sha,file_count)
            VALUES (2,'{source_ref}','{plan_hash(rows)}',34);
          COMMIT;""")
        return acknowledge_plan(db, rows, source_ref, meta)
    # current_* acredita el plan reconocido; solo los recibos acreditan instalación.
    return {**meta, "current_source_ref": source_ref, "current_plan_sha": plan_hash(rows),
            "revisions": revisions}


def receipts(db, rows):
    raw = db.query(f"""SELECT coalesce(json_agg(x ORDER BY position), '[]'::json)
                    FROM (SELECT position,path,sha256,installed_at FROM {SCHEMA}.applied) x;""")
    installed = json.loads(raw)
    if len(installed) > len(rows):
        raise Refused("hay recibos ajenos al plan")
    for position, receipt in enumerate(installed, 1):
        expected = rows[position - 1]
        if (receipt["position"] != position or receipt["path"] != expected["path"]
                or receipt["sha256"] != expected["sha256"]):
            raise Refused("recibos incompatibles o incompletos; no reaplicar")
    return installed


def instrument(row, position):
    """Añade exclusivamente el recibo del clon dentro del COMMIT original."""
    guard = f"""BEGIN;
SELECT pg_advisory_xact_lock(hashtextextended('vec_recorridos_clon:sql',0));
DO $clon_guard$ BEGIN
 IF (SELECT count(*) FROM {SCHEMA}.applied) <> {position - 1}
 THEN RAISE EXCEPTION 'orden SQL del clon incompatible'; END IF;
END $clon_guard$;
"""
    text = re.sub(r"(?m)^BEGIN;\s*$", lambda _: guard, row["sql"], count=1)
    receipt = f"""RESET ROLE;
INSERT INTO {SCHEMA}.applied(position,path,sha256)
VALUES ({position},{sql_literal(row['path'])},{sql_literal(row['sha256'])});
COMMIT;
"""
    return re.sub(r"COMMIT;\s*$", lambda _: receipt, text)


def write_journal(state, meta, installed):
    journal = state / "sql-journal.json"
    if journal.is_symlink():
        raise Refused("el journal privado no puede ser un enlace")
    if journal.exists():
        old = json.loads(journal.read_text())
        if old["run_id"] != meta["run_id"] or old["plan_sha"] != meta["plan_sha"]:
            raise Refused("journal de otro clon; usar un directorio propio")
        old_revisions = old.get("revisions", [])
        if old_revisions != meta.get("revisions", [])[:len(old_revisions)]:
            raise Refused("revisión del journal ausente de PostgreSQL")
        if old["installed"] != installed[:len(old["installed"])]:
            raise Refused("el journal contiene una instalación ausente de PostgreSQL")
    descriptor, temporary = tempfile.mkstemp(prefix=".sql-journal-", dir=state)
    try:
        with os.fdopen(descriptor, "w") as file:
            json.dump({**meta, "installed": installed}, file, indent=2)
            file.write("\n")
            file.flush()
            os.fsync(file.fileno())
        os.replace(temporary, journal)
        descriptor = os.open(state, os.O_DIRECTORY)
        try:
            os.fsync(descriptor)
        finally:
            os.close(descriptor)
    finally:
        Path(temporary).unlink(missing_ok=True)


def apply(db, rows, state, source_ref=MAIN_REF):
    db.check_owner()
    meta = initialize(db, rows, source_ref)
    installed = receipts(db, rows)
    write_journal(state, meta, installed)
    skipped = len(installed)
    for position, row in enumerate(rows, 1):
        if position <= skipped:
            continue
        try:
            db.query(instrument(row, position))
        except Refused as error:
            raise Refused(f"FALLO {position} {row['path']}: {error}") from error
        installed = receipts(db, rows)
        if len(installed) != position:
            raise Refused("falta el recibo transaccional; detener sin reaplicar")
        write_journal(state, meta, installed)
        print(f"OK {position} {row['phase']} {row['path']}", flush=True)
    print(f"SQL-OK instaladas={len(installed)} nuevas={len(installed) - skipped} recuperadas={skipped}")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--source-ref", default=MAIN_REF)
    parser.add_argument("--container")
    parser.add_argument("--state-dir", type=Path)
    parser.add_argument("--plan", action="store_true")
    args = parser.parse_args(argv)
    rows = load_plan(args.repo, source_ref=args.source_ref)
    if args.plan:
        for position, row in enumerate(rows, 1):
            print(position, row["phase"], row["sha256"], row["path"])
        return
    if not args.container or not args.state_dir:
        raise Refused("la instalación exige --container y --state-dir")
    state = args.state_dir.resolve()
    if state.is_relative_to(args.repo.resolve()):
        raise Refused("el journal debe estar fuera del árbol Git")
    if any((parent / ".git").exists() for parent in [state, *state.parents]):
        raise Refused("el journal debe estar fuera de cualquier repositorio Git")
    if args.state_dir.is_symlink():
        raise Refused("el directorio privado no puede ser un enlace")
    state.mkdir(mode=0o700, parents=True, exist_ok=True)
    if state.stat().st_uid != os.getuid() or state.stat().st_mode & 0o077:
        raise Refused("el directorio de estado exige propietario actual y modo 0700")
    descriptor = os.open(state / "sql.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        fcntl.flock(descriptor, fcntl.LOCK_EX | fcntl.LOCK_NB)
        apply(DockerDB(args.container, state), rows, state, args.source_ref)
    finally:
        os.close(descriptor)


if __name__ == "__main__":
    try:
        main()
    except (Refused, OSError, ValueError, subprocess.SubprocessError) as error:
        # No imprimir excepciones de subprocess ni JSON que podrían traer secretos.
        message = str(error) if isinstance(error, Refused) else type(error).__name__
        print(f"SQL-NO-GO {message}", file=sys.stderr)
        sys.exit(1)
