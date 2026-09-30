#!/usr/bin/env python3
"""Instala el SQL H3, H4 y main sobre un clon H1 propiedad de Codex-M.

H1 se restaura antes; H5 solo configura la aplicación. No instala SQL de ramas
pendientes ni ejecuta DOWN. --plan valida todos los SHA sin acceder a Docker.
El journal privado se reconstruye desde recibos transaccionales del clon.
Admite la base main@7f1ecea2f (33 SQL), la extensión main@ff6493cfc
(CT147, posición 34), main@e78687528 (AD3-114/CT148, posiciones 35/36)
y main@a7d9df2b3 (AD3-113/Documentos9, posiciones 37/38).
Cada extensión conserva los recibos y metadatos originales
y añade una revisión del plan en el esquema del clon. Otros hashes exigen revisar
de nuevo la lista causal y sus huellas. Descendientes del último plan aprobado
se admiten con ascendencia y TODO el inventario SQL idéntico, cotejado con el
archivo extraído. Esta prueba no aprueba los contratos de otros componentes.
"""

import argparse
import fcntl
import hashlib
import io
import json
import os
from pathlib import Path
import re
import posixpath
import stat
import subprocess
import sys
import tempfile
import uuid

BASE_REF = "7f1ecea2fd9f8912d255a80e74da84c69e46b978"
PREVIOUS_REF = "ff6493cfccb2da4e83c94fa7c59be24c025cb7c9"
THIRD_REF = "e78687528d5725efd74e95c858d389f4437099ca"
MAIN_REF = "a7d9df2b3285b0df6be6bba0bae09331463f0a3d"
REF_COUNTS = {BASE_REF: 33, PREVIOUS_REF: 34, THIRD_REF: 36, MAIN_REF: 38}
REF_ORDER = tuple(REF_COUNTS)
REF_PLAN_SHA = {
    BASE_REF: "70795c1580e550e2ccc8927bf50cf7130f73ca282d6069f74ba7e697f79e6be0",
    PREVIOUS_REF: "00d8dbaacd881a6945a33dd188e94e136b054f4e96a8ac29bdcb039637fc891c",
    THIRD_REF: "ad57f371c901f7418af156f4e651c7a36c2bda125f7bad380653272c2b320e14",
    MAIN_REF: "b92cea3eb1cb5497bab027eac561a168b3572a117417acf45597a6eac39403f5",
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


def load_plan(repo, manifest=MANIFEST, source_ref=MAIN_REF, contents=None):
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
        if contents is None:
            path = repo / relative
            if not path.resolve().is_relative_to(repo.resolve()) or path.is_symlink():
                raise Refused("SQL fuera del árbol fuente")
            data = path.read_bytes()
        else:
            data = contents[relative]
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


class GitSource:
    """Lectura de objetos del repositorio de control, sin entorno Git heredado."""
    def __init__(self, repo=None):
        self.repo = Path(repo or Path(__file__).resolve().parents[2]).resolve()
        top = self.run("rev-parse", "--show-toplevel").decode().strip()
        if Path(top).resolve() != self.repo:
            raise Refused("Git exige la raíz del repositorio de control")
        grafts = Path(self.run("rev-parse", "--git-path", "info/grafts").decode().strip())
        if not grafts.is_absolute():
            grafts = self.repo / grafts
        if grafts.exists() and grafts.stat().st_size:
            raise Refused("la historia Git contiene grafts; no verificar ascendencia")

    def run(self, *args, input=None, ancestor=False):
        result = subprocess.run(
            ["/usr/bin/git", "--no-replace-objects", "-C", str(self.repo),
             "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", *args],
            input=input, capture_output=True, timeout=60,
            env={"PATH": "/usr/bin:/bin", "HOME": "/nonexistent", "LC_ALL": "C",
                 "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null",
                 "GIT_NO_REPLACE_OBJECTS": "1", "GIT_TERMINAL_PROMPT": "0"})
        if ancestor and result.returncode in (0, 1):
            return result.returncode == 0
        if result.returncode:
            raise Refused("Git no acredita la fuente solicitada")
        return result.stdout

    def inventory(self, commit):
        raw = self.run("ls-tree", "-r", "-z", "-l", "--full-tree", commit)
        files = []
        for entry in raw.split(b"\0"):
            if not entry:
                continue
            header, name = entry.split(b"\t", 1)
            mode, kind, oid, size = header.decode("ascii").split()
            path = name.decode("utf-8")
            if path.lower().endswith((".up.sql", "_up.sql")) and not path.startswith("deploy/postgresql/"):
                raise Refused("UP fuera del inventario SQL aprobado")
            if not path.startswith("deploy/postgresql/"):
                continue
            if kind != "blob" or mode not in {"100644", "100755"}:
                raise Refused("el namespace SQL contiene enlaces o submódulos")
            if path.lower().endswith(".sql"):
                files.append({"path": path, "mode": mode, "blob_oid": oid, "size": int(size)})
        if not files or len(files) > 4096 or sum(f["size"] for f in files) > 128 * 1024 * 1024:
            raise Refused("inventario SQL ausente o fuera de límites")
        oids = list(dict.fromkeys(f["blob_oid"] for f in files))
        stream = io.BytesIO(self.run("cat-file", "--batch", input=("\n".join(oids) + "\n").encode()))
        blobs = {}
        for oid in oids:
            header = stream.readline().decode("ascii").split()
            if len(header) != 3 or header[:2] != [oid, "blob"]:
                raise Refused("objeto Git SQL incompatible")
            data = stream.read(int(header[2]))
            if stream.read(1) != b"\n":
                raise Refused("objeto Git SQL incompleto")
            blobs[oid] = data
        contents = {f["path"]: blobs[f["blob_oid"]] for f in files}
        records = [{k: v for k, v in f.items() if k != "size"} | {"sha256": sha(contents[f["path"]])}
                   for f in sorted(files, key=lambda x: x["path"])]
        validate_includes(contents)
        return records, contents


def validate_includes(contents):
    """El inventario cierra también los componentes incluidos por psql."""
    for path, data in contents.items():
        for line in data.decode("utf-8").splitlines():
            if not re.match(r"^\s*\\i(?:r)?(?:\s|$)", line):
                continue
            match = re.fullmatch(r"\s*\\ir\s+([A-Za-z0-9_./-]+)\s*", line)
            if not match:
                raise Refused("include SQL dinámico o sin directorio relativo cerrado")
            target = posixpath.normpath(posixpath.join(posixpath.dirname(path), match[1]))
            if not target.startswith("deploy/postgresql/") or target not in contents:
                raise Refused("include SQL fuera del inventario aprobado")


def validate_git_source(source_ref, git_repo=None):
    """Preflight RO: verifica commit y plan SQL; aún no acredita un archive."""
    if not re.fullmatch(r"[0-9a-f]{40}", source_ref):
        raise Refused("la fuente requiere un SHA de commit completo")
    git = GitSource(git_repo)
    if git.run("cat-file", "-t", source_ref).strip() != b"commit":
        raise Refused("la fuente debe ser un commit")
    main = git.run("rev-parse", "--verify", "refs/remotes/origin/main^{commit}").decode().strip()
    approved = source_ref if source_ref in REF_COUNTS else MAIN_REF
    if (not git.run("merge-base", "--is-ancestor", approved, source_ref, ancestor=True)
            or not git.run("merge-base", "--is-ancestor", source_ref, main, ancestor=True)):
        raise Refused("la fuente no pertenece a la historia aprobada de origin/main")
    expected, _ = git.inventory(approved)
    actual, contents = git.inventory(source_ref)
    if actual != expected:
        raise Refused("SQL distinto del plan aprobado; requiere revisión de un plan nuevo")
    rows = load_plan(None, source_ref=approved, contents=contents)
    return {"source_ref": source_ref, "approved_sql_ref": approved,
            "plan_sha": plan_hash(rows), "inventory_sha": sha(json.dumps(actual, sort_keys=True).encode()),
            "file_count": len(rows), "entries": [{k: v for k, v in r.items() if k != "sql"} for r in rows],
            "verified_main_ref": main, "sql_inventory": actual}


def approved_source_plan(repo, source_ref, git_repo=None):
    """API RO para material: prueba Git y TODO el SQL del archive, sin tocar BD."""
    plan = validate_git_source(source_ref, git_repo)
    root = Path(repo)
    if root.is_symlink() or not root.is_dir():
        raise Refused("la fuente extraída debe ser un directorio regular")
    namespace = root / "deploy/postgresql"
    if (root / "deploy").is_symlink() or namespace.is_symlink():
        raise Refused("el namespace SQL extraído no admite enlaces")
    actual = {}
    for directory, dirs, files in os.walk(namespace, followlinks=False):
        if any((Path(directory) / d).is_symlink() for d in dirs):
            raise Refused("el archive SQL contiene directorios enlazados")
        for name in files:
            if not name.lower().endswith(".sql"):
                continue
            path = Path(directory) / name
            status = path.lstat()
            if not stat.S_ISREG(status.st_mode):
                raise Refused("el archive SQL contiene enlaces o ficheros especiales")
            actual[path.relative_to(root).as_posix()] = path
    expected = {r["path"]: r for r in plan["sql_inventory"]}
    if actual.keys() != expected.keys():
        raise Refused("el inventario SQL del archive no coincide con Git")
    for relative, path in actual.items():
        record = expected[relative]
        descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
        with os.fdopen(descriptor, "rb") as file:
            status = os.fstat(file.fileno())
            if not stat.S_ISREG(status.st_mode) or status.st_size > 128 * 1024 * 1024:
                raise Refused("tipo o tamaño SQL extraído incompatible")
            mode = "100755" if status.st_mode & 0o111 else "100644"
            data = file.read(128 * 1024 * 1024 + 1)
        if mode != record["mode"] or sha(data) != record["sha256"]:
            raise Refused("los bytes o el modo SQL del archive no coinciden con Git")
    for directory, _, files in os.walk(root, followlinks=False):
        for name in files:
            path = Path(directory) / name
            if name.lower().endswith((".up.sql", "_up.sql")) and not path.is_relative_to(namespace):
                raise Refused("el archive contiene UP fuera del namespace SQL")
    return plan


def validate_receipts(installed, plan, complete=True):
    """API pura: recibos exactos del plan; material exige instalación completa."""
    if len(installed) > plan["file_count"] or (complete and len(installed) != plan["file_count"]):
        raise Refused("instalación incompleta o recibos ajenos al plan aprobado")
    for position, receipt in enumerate(installed, 1):
        row = plan["entries"][position - 1]
        if (receipt["position"] != position or receipt["path"] != row["path"]
                or receipt["sha256"] != row["sha256"]):
            raise Refused("recibos incompatibles o incompletos; no reaplicar")


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
        original_index = REF_ORDER.index(meta["source_ref"])
        expected_refs = REF_ORDER[original_index + 1:original_index + 1 + len(revisions)]
        if not revisions or len(expected_refs) != len(revisions):
            raise Refused("revisiones de plan incompatibles; no instalar ni volver a la base")
        for revision, ref in zip(revisions, expected_refs):
            count = REF_COUNTS[ref]
            if (revision["revision"] != REF_ORDER.index(ref) + 1
                    or revision["source_ref"] != ref or count > len(rows)
                    or revision["plan_sha"] != plan_hash(rows[:count])
                    or revision["file_count"] != count):
                raise Refused("revisiones de plan incompatibles; no instalar ni volver a la base")
    recognized_ref = revisions[-1]["source_ref"] if revisions else meta["source_ref"]
    if recognized_ref != source_ref:
        index = REF_ORDER.index(recognized_ref)
        if index + 1 >= len(REF_ORDER) or REF_ORDER[index + 1] != source_ref:
            raise Refused("extensión no autorizada: completar primero la revisión intermedia")
        required = REF_COUNTS[recognized_ref]
        if len(receipts(db, rows)) != required:
            raise Refused(f"la extensión requiere las {required} SQL anteriores completas")
        revision = REF_ORDER.index(source_ref) + 1
        count = REF_COUNTS[source_ref]
        if exists == "t":
            # V2 restringía esta infraestructura a la revisión 2 y 34 ficheros.
            # Ampliar solo sus CHECK; no actualizar filas ni recibos originales.
            ddl = f"""ALTER TABLE {SCHEMA}.plan_revisions
              DROP CONSTRAINT IF EXISTS plan_revisions_revision_check,
              DROP CONSTRAINT IF EXISTS plan_revisions_file_count_check,
              DROP CONSTRAINT IF EXISTS plan_revisions_supported,
              ADD CONSTRAINT plan_revisions_supported CHECK (
                (revision=2 AND file_count=34) OR (revision=3 AND file_count=36)
                OR (revision=4 AND file_count=38));"""
        else:
            ddl = f"""CREATE TABLE {SCHEMA}.plan_revisions (
              revision integer PRIMARY KEY, source_ref text NOT NULL,
              plan_sha text NOT NULL, file_count integer NOT NULL,
              acknowledged_at timestamptz NOT NULL DEFAULT clock_timestamp(),
              CONSTRAINT plan_revisions_supported CHECK (
                (revision=2 AND file_count=34) OR (revision=3 AND file_count=36)
                OR (revision=4 AND file_count=38)));
              REVOKE ALL ON {SCHEMA}.plan_revisions FROM PUBLIC;"""
        db.query(f"""BEGIN;
          SELECT pg_advisory_xact_lock(hashtextextended('vec_recorridos_clon:sql',0));
          {ddl}
          INSERT INTO {SCHEMA}.plan_revisions(revision,source_ref,plan_sha,file_count)
            VALUES ({revision},'{source_ref}','{plan_hash(rows)}',{count});
          COMMIT;""")
        return acknowledge_plan(db, rows, source_ref, meta)
    # Plan SQL reconocido; apply añade la procedencia de código verificada.
    return {**meta, "current_source_ref": source_ref, "current_plan_sha": plan_hash(rows),
            "revisions": revisions}


def receipts(db, rows):
    raw = db.query(f"""SELECT coalesce(json_agg(x ORDER BY position), '[]'::json)
                    FROM (SELECT position,path,sha256,installed_at FROM {SCHEMA}.applied) x;""")
    installed = json.loads(raw)
    validate_receipts(installed, {"file_count": len(rows), "entries": rows}, complete=False)
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


def apply(db, rows, state, source_ref=MAIN_REF, source_plan=None):
    if source_plan and (source_plan["source_ref"] != source_ref
                        or source_plan["plan_sha"] != plan_hash(rows)):
        raise Refused("la procedencia y el plan SQL no coinciden")
    db.check_owner()
    approved_ref = source_plan["approved_sql_ref"] if source_plan else source_ref
    meta = initialize(db, rows, approved_ref)
    meta["approved_sql_ref"] = approved_ref
    if source_plan:
        # Esta procedencia no aprueba los contratos Go, material ni DB_READY.
        meta.update(current_source_ref=source_ref, verified_source_ref=source_ref,
                    verified_main_ref=source_plan["verified_main_ref"],
                    inventory_sha=source_plan["inventory_sha"])
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
    parser.add_argument("--git-repo", type=Path, help="repositorio de control para ascendencia y objetos")
    parser.add_argument("--container")
    parser.add_argument("--state-dir", type=Path)
    parser.add_argument("--plan", action="store_true")
    args = parser.parse_args(argv)
    if args.plan:
        # Plan desde el COMMIT, incluso si el checkout de control contiene WIP.
        plan = validate_git_source(args.source_ref, args.git_repo)
        for position, row in enumerate(plan["entries"], 1):
            print(position, row["phase"], row["sha256"], row["path"])
        return
    plan = approved_source_plan(args.repo, args.source_ref, args.git_repo)
    rows = load_plan(args.repo, source_ref=plan["approved_sql_ref"])
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
        apply(DockerDB(args.container, state), rows, state, args.source_ref, plan)
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
