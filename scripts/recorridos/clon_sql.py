#!/usr/bin/env python3
"""Instala el SQL H3, H4 y main sobre un clon H1 propiedad de Codex-M.

H1 se restaura antes; H5 solo configura la aplicación. No instala SQL de ramas
pendientes ni ejecuta DOWN. --plan valida todos los SHA sin acceder a Docker.
El journal privado se reconstruye desde recibos transaccionales del clon.
Admite la base main@7f1ecea2f (33 SQL), la extensión main@ff6493cfc
(CT147, posición 34), main@e78687528 (AD3-114/CT148, posiciones 35/36)
main@a7d9df2b3 (AD3-113/Documentos9, posiciones 37/38), main@1e443463d
(Aspirantes000002, posición39) y main@890b3fe0e (roles, categorías, lecturas
nominales y AD3-117, posiciones40..43).
H6 main@5694d2da1 instala 41 SQL: el prefijo39 y CT150/CT151. El plan interno
main@ebac67de4 añade CT145/AD125/CT152 como revisión7 de esa familia, hasta44.
main@73e56c106 propone CT153 en posición45, revisión8. AD3-132 queda fuera
del instalador y sus recibos: requiere la CLI DBA tras el kit y antes de arrancar.
Retiene cuatro SQL RPT y 19 del portal exterior, cotejadas con todo el inventario.
Las familias 41/44/45 y 43 parten de39; no existe transición de43 a41/44/45.
El plan 43 de 890b3fe0e está retirado para nuevas instalaciones: se conserva
solo para lectura y recuperación de los 43 recibos exactos ya instalados.
Cada extensión conserva los recibos y metadatos originales
y añade una revisión del plan en el esquema del clon. Otros hashes exigen revisar
de nuevo la lista causal y sus huellas. Descendientes de un plan aprobado
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
from datetime import datetime

BASE_REF = "7f1ecea2fd9f8912d255a80e74da84c69e46b978"
PREVIOUS_REF = "ff6493cfccb2da4e83c94fa7c59be24c025cb7c9"
THIRD_REF = "e78687528d5725efd74e95c858d389f4437099ca"
FOURTH_REF = "a7d9df2b3285b0df6be6bba0bae09331463f0a3d"
FIFTH_REF = "1e443463df69dffeaac239f9b7000f48dd1b7bb7"
MAIN_REF = "890b3fe0e9f9e30e249b9dc2d3778971121a8cc2"
H6_REF = "5694d2da15e19fa97afecae51e1a30ce21d5fca5"
H6_FIRMA_REF = "ebac67de4e43fc49add3d82a011b2b0c9f6a6b21"
H6_FIRMA_FINAL_REF = "73e56c106d12fdda0bd16d6fe573503c42c5495f"
REF_COUNTS = {BASE_REF: 33, PREVIOUS_REF: 34, THIRD_REF: 36, FOURTH_REF: 38,
              FIFTH_REF: 39, MAIN_REF: 43, H6_REF: 41, H6_FIRMA_REF: 44,
              H6_FIRMA_FINAL_REF: 45}
# Índice de aprobaciones SQL en Git; no determina el orden de instalación.
REF_ORDER = tuple(REF_COUNTS)
REF_PARENT = {BASE_REF: None, PREVIOUS_REF: BASE_REF, THIRD_REF: PREVIOUS_REF,
              FOURTH_REF: THIRD_REF, FIFTH_REF: FOURTH_REF,
              MAIN_REF: FIFTH_REF, H6_REF: FIFTH_REF, H6_FIRMA_REF: H6_REF,
              H6_FIRMA_FINAL_REF: H6_FIRMA_REF}
RECOVERY_ONLY_REFS = {MAIN_REF}
PROPOSED_REFS = {H6_FIRMA_REF, H6_FIRMA_FINAL_REF}
REF_PLAN_SHA = {
    BASE_REF: "70795c1580e550e2ccc8927bf50cf7130f73ca282d6069f74ba7e697f79e6be0",
    PREVIOUS_REF: "00d8dbaacd881a6945a33dd188e94e136b054f4e96a8ac29bdcb039637fc891c",
    THIRD_REF: "ad57f371c901f7418af156f4e651c7a36c2bda125f7bad380653272c2b320e14",
    FOURTH_REF: "b92cea3eb1cb5497bab027eac561a168b3572a117417acf45597a6eac39403f5",
    FIFTH_REF: "af888b95d532a0b698212adbebb381d5f2f126e7393396af90000fddc9ca3427",
    MAIN_REF: "5999af8fc61d25a6d63c4f2664012edfc59f4a38b8fb5ddea5daea2b818f4877",
    H6_REF: "95c3feff3cbd5b95cf0286af74c576d2c551d92b3cb7787b337754d3aeed87fb",
    H6_FIRMA_REF: "248e8771af8a8d59fb9df94d3400cabecb9421dbb56b8b62c479432f2e7bab92",
    H6_FIRMA_FINAL_REF: "52240f99125ce4b83872bc41c75bfda233e7957b0b2978aa361c26fba708ac48",
}
OWNER_LABEL = "vec.recorridos.owner"
OWNER = "Codex-M"
SCHEMA = "vec_recorridos_clon"
MANIFEST = Path(__file__).with_name("sql_main.txt")
H6_MANIFEST = Path(__file__).with_name("sql_main_h6.txt")
H6_FIRMA_MANIFEST = Path(__file__).with_name("sql_main_h6_firma.txt")
H6_CT153 = {
    "phase": "MAIN",
    "sha256": "6e45fbb7b1338a9fcda1b67551bd2c485d763fe64c19f80d3c37d7467ef7f3d6",
    "path": "deploy/postgresql/contratacion_temporal/migraciones/000153_perfil_reincorporacion_titular.up.sql",
}
H6_DBA_EXCLUDED = {
    "deploy/postgresql/autorizacion_atestada_v3/migraciones/000132_cierre_temp_public_base_vec.up.sql":
        "06dfefe50be8029fcee36a794d65e8d4c73d486d4e1ea2ce23823b9865c4bbea",
}
H6_WITHHELD = {
    "deploy/postgresql/catalogos_configurables/roles_up.sql":
        "55b5b8aa2102ce45fc56f37eb165f3cd5119e343a345d238e26a5e3023ff2547",
    "deploy/postgresql/catalogos_configurables/migraciones/000001_autoridad_categorias.up.sql":
        "1d940cc3be8f000bc10fc7ba97ed6e9105b0bf4681491fa0776dd2e2e81e13b7",
    "deploy/postgresql/catalogos_configurables/migraciones/000002_lecturas_nominales.up.sql":
        "97e22af3b344427360a9541c6807f9d9a63a260142edb2dc128959924fd3fbda",
    "deploy/postgresql/autorizacion_atestada_v3/migraciones/000117_lecturas_categorias_rpt.up.sql":
        "eaceb3b03b6db8a5bed6be4b42ff9e939525234308cdda52b899716f98658cca",
}
H6_FIRMA_WITHHELD = H6_WITHHELD | {
    "deploy/postgresql/identidad_sesiones_v1/roles_externos_up.sql": "210fec093139607c3d4a42a35705e085eb9f8d7e631d62d9867f6dd62bc0adaf",
    "deploy/postgresql/identidad_sesiones_v1/migraciones/000007_identidad_externa_v1.up.sql": "74a9feaeab395c756d9a7db6d39bafca3fb2af54e12039ba7d5907851997bb1c",
    "deploy/postgresql/identidad_sesiones_v1/migraciones/000008_fachada_autorizacion_externa_v1.up.sql": "49584659f913bf203a9e420502e03b7f3cae02c228b1c603f87832103d3542e5",
    "deploy/postgresql/contexto_actor_v1/roles_candidato_externo_up.sql": "1dab1abe1405ffff850ee1e6b688462ac5b05d8ff6a682e7e07331334e339c76",
    "deploy/postgresql/contexto_actor_v1/migraciones/000012_contexto_candidato_externo.up.sql": "d7146f04da8f95eb6d564a5ea8cdfb1c0d237df957042f285fe43bb4b0a6fe31",
    "deploy/postgresql/contexto_actor_v1/migraciones/000013_acreditacion_candidato_externo_exacto.up.sql": "0e05fd70fd443f6c029df933b679dddf0198c1dbeefcd7683f95014009885a85",
    "deploy/postgresql/contexto_actor_v1/migraciones/000014_perfil_usuarios_externo.up.sql": "89476b2e2d131ee78804878bae9fb9ab1517ae288973adb42c1f7d603af67bee",
    "deploy/postgresql/contexto_actor_v1/migraciones/000015_contexto_externo_segregado.up.sql": "23654ff30e8fc72e04245dd9ba27f69a62746c83250cbafb05d025fbdfa2b2f0",
    "deploy/postgresql/contexto_actor_v1/migraciones/000017_contexto_externo_tipos_temporales.up.sql": "cdc46530f3360651739e444f051e5a1ef35d27d81d09f68daaeef3ab34702c37",
    "deploy/postgresql/bolsa_llamamientos/migraciones/000060_portal_externo_candidato.up.sql": "2a98410ec05feda76aef3ceb01bf4ca173c9f0da6c27f7d22876c536e14e3214",
    "deploy/postgresql/bolsa_llamamientos/migraciones/000061_denegaciones_portal_externo.up.sql": "449094f319a198936effeedaff2a766966f721953e8d5ce4931b2cd0c66cdff2",
    "deploy/postgresql/autorizacion_atestada_v3/migraciones/000115_consumidor_bolsa_portal_externo.up.sql": "89b15f44b43593c74eb0889d692ab958c60b8aa12e93d58b4fa30a325ca27b46",
    "deploy/postgresql/autorizacion/migraciones/000015_candidato_externo.up.sql": "98f28821769bfc68ae3f95dd971e6694bb6c8a033639cee3c781778cbce72aec",
    "deploy/postgresql/autorizacion/migraciones/000016_asignacion_perfil_candidato_externo.up.sql": "7cd4b35819e2731cd0281f60b59aaa7b214b2c2e109284dd792347c2fc323e18",
    "deploy/postgresql/autorizacion/migraciones/000019_candidato_externo_fechas_cero_canonicas.up.sql": "da485ff14aee904c6d48e6b544804a3ceb82a0c038c3fc90992f43a75e4c3f05",
    "deploy/postgresql/autorizacion_atestada_v3/migraciones/000116_consumo_candidato_externo.up.sql": "dc15f6e9d86c92725e7e63970cc02a06f4b31310a1cc7f4d857d865db7e5778c",
    "deploy/postgresql/autorizacion/migraciones/000020_clausura_externa_tipos_temporales.up.sql": "180200f6e42c09272a439652df7b02faf64ca8e009715b0356519553deffeb30",
    "deploy/postgresql/autorizacion_atestada_v3/migraciones/000121_clausura_externa_tipos_temporales.up.sql": "c0a6387ac3c75cfed9bede92128756f7c9cb692839f2d5ee8461ea169ae2043c",
    "deploy/postgresql/bolsa_llamamientos/migraciones/000065_clausura_externa_tipos_temporales.up.sql": "07c00288269d91dbd4331c4e28ed5f363288df62926fc1c08dcf7571525c8fd6",
}


def execution_manifest(source_ref):
    return {H6_REF: H6_MANIFEST, H6_FIRMA_REF: H6_FIRMA_MANIFEST,
            H6_FIRMA_FINAL_REF: H6_FIRMA_MANIFEST}.get(source_ref, MANIFEST)


def withheld_sql(source_ref):
    return {H6_REF: H6_WITHHELD, H6_FIRMA_REF: H6_FIRMA_WITHHELD,
            H6_FIRMA_FINAL_REF: H6_FIRMA_WITHHELD}.get(source_ref, {})


def dba_excluded_sql(source_ref):
    return H6_DBA_EXCLUDED if source_ref == H6_FIRMA_FINAL_REF else {}


class Refused(RuntimeError):
    pass


def sha(data):
    return hashlib.sha256(data).hexdigest()


def sql_literal(value):
    return "'" + value.replace("'", "''") + "'"


def plan_path(source_ref):
    """Camino de ejecución causal; 43 y H6 son ramas hermanas."""
    if source_ref not in REF_PARENT:
        raise Refused("referencia sin camino SQL aprobado")
    parent = REF_PARENT[source_ref]
    return (*plan_path(parent), source_ref) if parent else (source_ref,)


def plan_family(source_ref):
    return {H6_REF: "h6_41", H6_FIRMA_REF: "h6_44", H6_FIRMA_FINAL_REF: "h6_45",
            MAIN_REF: "retained_43"}.get(source_ref, "common_prefix")


def require_installable(approved_ref):
    if approved_ref in PROPOSED_REFS:
        raise Refused("plan SQL propuesto: falta revisión y ensayo antes de instalar")


def validate_history(original, revisions):
    """Reconoce revisiones completas contiguas dentro de una única familia."""
    recognized = original
    for revision in revisions:
        ref = revision["source_ref"]
        if (ref not in REF_PARENT or REF_PARENT[ref] != recognized
                or revision["revision"] != len(plan_path(ref))
                or revision["plan_sha"] != REF_PLAN_SHA[ref]
                or revision["file_count"] != REF_COUNTS[ref]):
            raise Refused("revisiones de plan incompatibles o con saltos; no instalar ni volver a la base")
        recognized = ref
    return recognized


def load_plan(repo, manifest=None, source_ref=MAIN_REF, contents=None):
    """Congela los bytes antes de escribir; un árbol alterado falla completo."""
    rows = []
    if source_ref not in REF_COUNTS:
        raise Refused("el plan solo corresponde a los hashes main fijados")
    if manifest is None:
        manifest = execution_manifest(source_ref)
    historical_tail = False
    for line in manifest.read_text().splitlines():
        if not line.strip() or line.startswith("#"):
            continue
        phase, digest, relative = line.split()
        if relative in H6_DBA_EXCLUDED:
            raise Refused("AD3-132 excluida del plan funcional; requiere la CLI DBA separada")
        if len(rows) == REF_COUNTS[source_ref]:
            # La revisión7 conserva sus44 entradas al leer el manifiesto nuevo.
            # Solo admite la cola exacta CT153; no lee bytes ajenos a su commit.
            if (source_ref == H6_FIRMA_REF and not historical_tail
                    and {"phase": phase, "sha256": digest, "path": relative} == H6_CT153):
                historical_tail = True
                continue
            if source_ref in (H6_REF, H6_FIRMA_REF, H6_FIRMA_FINAL_REF):
                raise Refused(f"manifiesto H6 contiene SQL fuera de sus {REF_COUNTS[source_ref]} entradas")
            break
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
    if source_ref in (H6_REF, H6_FIRMA_REF, H6_FIRMA_FINAL_REF):
        for relative, digest in withheld_sql(source_ref).items():
            if contents is None:
                path = repo / relative
                if not path.resolve().is_relative_to(repo.resolve()) or path.is_symlink():
                    raise Refused("SQL retenida fuera del árbol fuente")
                data = path.read_bytes()
            else:
                data = contents[relative]
            if sha(data) != digest or any(row["path"] == relative for row in rows):
                raise Refused("SQL retenida distinta de la aprobación H6; requiere nueva revisión")
        for relative, digest in dba_excluded_sql(source_ref).items():
            if contents is None:
                path = repo / relative
                if not path.resolve().is_relative_to(repo.resolve()) or path.is_symlink():
                    raise Refused("SQL DBA fuera del árbol fuente")
                data = path.read_bytes()
            else:
                data = contents[relative]
            if sha(data) != digest:
                raise Refused("SQL DBA excluida distinta de la propuesta H6; requiere nueva revisión")
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
    if not git.run("merge-base", "--is-ancestor", source_ref, main, ancestor=True):
        raise Refused("la fuente no pertenece a la historia aprobada de origin/main")
    approved = next((ref for ref in reversed(REF_COUNTS)
                     if git.run("merge-base", "--is-ancestor", ref, source_ref, ancestor=True)), None)
    if approved is None:
        raise Refused("la fuente no pertenece a la historia aprobada de origin/main")
    expected, _ = git.inventory(approved)
    actual, contents = git.inventory(source_ref)
    if actual != expected:
        raise Refused("SQL distinto del plan aprobado; requiere revisión de un plan nuevo")
    rows = load_plan(None, source_ref=approved, contents=contents)
    return {"source_ref": source_ref, "approved_sql_ref": approved,
            "status": ("proposed" if approved in PROPOSED_REFS else
                       "recovery_only" if approved in RECOVERY_ONLY_REFS else "installable"),
            "plan_family": plan_family(approved),
            "execution_manifest": execution_manifest(approved).name,
            "plan_sha": plan_hash(rows), "inventory_sha": sha(json.dumps(actual, sort_keys=True).encode()),
            "file_count": len(rows), "entries": [{k: v for k, v in r.items() if k != "sql"} for r in rows],
            "dba_excluded": [{"path": p, "sha256": digest}
                             for p, digest in dba_excluded_sql(approved).items()],
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


def verify_live(db, repo, git_repo, source_ref, state):
    """Acredita recibos y ACL actuales con consultas de solo lectura antes de READY."""
    plan = approved_source_plan(repo, source_ref, git_repo)
    require_installable(plan["approved_sql_ref"])
    rows = load_plan(repo, source_ref=plan["approved_sql_ref"])
    db.check_owner()
    installed = receipts(db, rows)
    validate_receipts(installed, plan)
    journal = state / "sql-journal.json"
    if journal.is_symlink() or not journal.is_file() or journal.stat().st_size > 2 * 1024 * 1024:
        raise Refused("journal SQL ausente o inválido para READY")
    record = json.loads(journal.read_text())
    if not isinstance(record, dict):
        raise Refused("journal SQL inválido para READY")
    if (record.get("installed") != installed or record.get("current_source_ref") != source_ref
            or record.get("approved_sql_ref") != plan["approved_sql_ref"]
            or record.get("current_plan_sha") != plan["plan_sha"]
            or record.get("inventory_sha") != plan["inventory_sha"]):
        raise Refused("journal y base divergen; no publicar READY")
    actual = json.loads(db.query(f"""SELECT json_build_object(
      'base_ref',p.source_ref,'base_sha',p.plan_sha,
      'last_ref',coalesce((SELECT r.source_ref FROM {SCHEMA}.plan_revisions r ORDER BY r.revision DESC LIMIT 1),p.source_ref),
      'last_sha',coalesce((SELECT r.plan_sha FROM {SCHEMA}.plan_revisions r ORDER BY r.revision DESC LIMIT 1),p.plan_sha)
      ) FROM {SCHEMA}.plan p WHERE p.singleton;"""))
    if (actual.get("base_ref") != record.get("source_ref")
            or actual.get("base_sha") != record.get("plan_sha")
            or actual.get("last_ref") != plan["approved_sql_ref"]
            or actual.get("last_sha") != plan["plan_sha"]):
        raise Refused("plan SQL vivo distinto del journal; no publicar READY")
    acl_safe = db.query("""SELECT NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_database d
      CROSS JOIN LATERAL pg_catalog.aclexplode(
        coalesce(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
      WHERE d.datname=current_database() AND a.grantee=0
    ) AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolcanlogin
        AND left(r.rolname,4)='vec_'
        AND (pg_catalog.has_database_privilege(r.oid,current_database(),'TEMP')
             OR pg_catalog.has_database_privilege(r.oid,current_database(),'CREATE'))
    );""")
    if acl_safe != "t":
        raise Refused("ACL de la base cambió; no publicar READY")


def etapas_requeridas(repo, git_repo, source_ref, state):
    """Consejo RO para el orquestador; nunca sustituye las comprobaciones de BD.

    Sin journal devuelve todos los prefijos aprobados hasta el destino. Un
    journal debe acreditar un prefijo completo y conocido; el instalador real
    coteja su run_id y sus recibos con PostgreSQL antes de cada UP.
    """
    target = approved_source_plan(repo, source_ref, git_repo)
    require_installable(target["approved_sql_ref"])
    target_path = plan_path(target["approved_sql_ref"])
    journal = Path(state) / "sql-journal.json"
    if not journal.exists() and not journal.is_symlink():
        if target["approved_sql_ref"] in RECOVERY_ONLY_REFS:
            raise Refused("plan 43 retirado para nuevas instalaciones; requiere una fuente corregida aprobada")
        return list(target_path)
    status = journal.lstat()
    if not stat.S_ISREG(status.st_mode) or status.st_size > 2 * 1024 * 1024:
        raise Refused("el journal no es un fichero regular válido")
    try:
        record = json.loads(journal.read_text())
        uuid.UUID(record["run_id"])
        original = record["source_ref"]
        if original not in REF_COUNTS or record["plan_sha"] != REF_PLAN_SHA[original]:
            raise Refused("el journal pertenece a un plan no aprobado")
        revisions = record.get("revisions", [])
        recognized = validate_history(original, revisions)
        if record.get("approved_sql_ref", recognized) != recognized:
            raise Refused("la aprobación del journal no coincide con su historia")
        if record.get("current_plan_sha", REF_PLAN_SHA[recognized]) != REF_PLAN_SHA[recognized]:
            raise Refused("la huella actual del journal no coincide con su aprobación")
        current = record.get("current_source_ref", recognized)
        if record.get("verified_source_ref", current) != current:
            raise Refused("la procedencia del journal no coincide con su fuente")
        current_plan = validate_git_source(current, git_repo)
        if current_plan["approved_sql_ref"] != recognized:
            raise Refused("la fuente del journal no conserva su plan SQL aprobado")
        if "inventory_sha" in record and record["inventory_sha"] != current_plan["inventory_sha"]:
            raise Refused("el inventario del journal no coincide con Git")
        for key in ("plan_family", "execution_manifest"):
            if key in record and record[key] != current_plan[key]:
                raise Refused("la familia o manifiesto del journal no coincide con Git")
        if "file_count" in record and record["file_count"] != current_plan["file_count"]:
            raise Refused("el número físico de SQL del journal no coincide con su plan")
        # Una caída puede ocurrir tras reconocer la revisión y antes de su
        # última UP. El instalador coteja además estos recibos con PostgreSQL.
        validate_receipts(record["installed"], current_plan, complete=False)
        if any(datetime.fromisoformat(r["installed_at"]).tzinfo is None
               for r in record["installed"]):
            raise Refused("el journal conserva recibos sin fecha válida")
        if recognized not in target_path:
            raise Refused("el journal conserva otra familia o una revisión posterior al destino")
        completed = target_path.index(recognized)
        if target["approved_sql_ref"] in RECOVERY_ONLY_REFS and recognized != target["approved_sql_ref"]:
            raise Refused("plan 43 retirado para nuevas instalaciones; requiere una fuente corregida aprobada")
        if len(record["installed"]) < current_plan["file_count"]:
            return list(target_path[completed:])
        return list(target_path[completed + 1:])
    except (KeyError, TypeError, ValueError, AttributeError) as error:
        raise Refused("el journal tiene metadatos inválidos o incompletos") from error


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
    """Preserva el plan original y amplía por una arista causal aprobada."""
    target_path = plan_path(source_ref)
    original_count = REF_COUNTS.get(meta["source_ref"])
    if original_count is None or meta["source_ref"] not in target_path:
        raise Refused("el clon conserva otro plan; requiere revisión del inventario")
    if meta["plan_sha"] != plan_hash(rows[:original_count]):
        raise Refused("el prefijo del plan original ha cambiado; no instalar")
    exists = db.query(f"SELECT to_regclass('{SCHEMA}.plan_revisions') IS NOT NULL;")
    revisions = []
    if exists == "t":
        revisions = json.loads(db.query(f"""SELECT coalesce(json_agg(x ORDER BY revision), '[]'::json)
          FROM (SELECT revision,source_ref,plan_sha,file_count,acknowledged_at
                FROM {SCHEMA}.plan_revisions) x;"""))
        if not revisions:
            raise Refused("revisiones de plan incompatibles; no instalar ni volver a la base")
        validate_history(meta["source_ref"], revisions)
        for revision in revisions:
            ref = revision["source_ref"]
            count = REF_COUNTS[ref]
            if (ref not in target_path or count > len(rows)
                    or revision["plan_sha"] != plan_hash(rows[:count])
                    or revision["file_count"] != count):
                raise Refused("revisiones de plan incompatibles; no instalar ni volver a la base")
    recognized_ref = revisions[-1]["source_ref"] if revisions else meta["source_ref"]
    if recognized_ref != source_ref:
        if source_ref in RECOVERY_ONLY_REFS:
            raise Refused("plan 43 retirado para nuevas revisiones; conservar solo recuperación exacta")
        if REF_PARENT[source_ref] != recognized_ref:
            raise Refused("extensión no autorizada: completar primero la revisión intermedia")
        required = REF_COUNTS[recognized_ref]
        if len(receipts(db, rows)) != required:
            raise Refused(f"la extensión requiere las {required} SQL anteriores completas")
        revision = len(target_path)
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
                OR (revision=4 AND file_count=38) OR (revision=5 AND file_count=39)
                OR (revision=6 AND file_count IN (41,43))
                OR (revision=7 AND file_count=44)
                OR (revision=8 AND file_count=45));"""
        else:
            ddl = f"""CREATE TABLE {SCHEMA}.plan_revisions (
              revision integer PRIMARY KEY, source_ref text NOT NULL,
              plan_sha text NOT NULL, file_count integer NOT NULL,
              acknowledged_at timestamptz NOT NULL DEFAULT clock_timestamp(),
              CONSTRAINT plan_revisions_supported CHECK (
                (revision=2 AND file_count=34) OR (revision=3 AND file_count=36)
                OR (revision=4 AND file_count=38) OR (revision=5 AND file_count=39)
                OR (revision=6 AND file_count IN (41,43))
                OR (revision=7 AND file_count=44)
                OR (revision=8 AND file_count=45)));
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
            "plan_family": plan_family(source_ref),
            "execution_manifest": execution_manifest(source_ref).name,
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
    approved_ref = source_plan["approved_sql_ref"] if source_plan else source_ref
    require_installable(approved_ref)
    db.check_owner()
    if len(rows) != REF_COUNTS.get(approved_ref) or plan_hash(rows) != REF_PLAN_SHA.get(approved_ref):
        raise Refused("el plan SQL no corresponde a su referencia aprobada")
    if approved_ref in RECOVERY_ONLY_REFS:
        # Antes de initialize: ni provisión, ni revisión nueva, ni primera UP.
        exists = db.query(f"SELECT to_regclass('{SCHEMA}.applied') IS NOT NULL;")
        if exists != "t":
            raise Refused("plan 43 retirado para nuevas instalaciones; no hay ledger 43 recuperable")
        installed = receipts(db, rows)
        validate_receipts(installed, {"file_count": len(rows), "entries": rows})
    meta = initialize(db, rows, approved_ref)
    meta["approved_sql_ref"] = approved_ref
    meta.update(plan_family=plan_family(approved_ref),
                execution_manifest=execution_manifest(approved_ref).name,
                file_count=len(rows))
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
    parser.add_argument("--source-ref", default=H6_REF)
    parser.add_argument("--git-repo", type=Path, help="repositorio de control para ascendencia y objetos")
    parser.add_argument("--container")
    parser.add_argument("--state-dir", type=Path)
    parser.add_argument("--plan", action="store_true")
    parser.add_argument("--installable", action="store_true", help="rechaza un plan aún propuesto sin tocar el clon")
    parser.add_argument("--verify-live", action="store_true", help="coteja recibos y ACL reales antes de READY")
    parser.add_argument("--steps", action="store_true", help="etapas pendientes, consejo JSON sin Docker/BD")
    args = parser.parse_args(argv)
    if args.installable:
        require_installable(validate_git_source(args.source_ref, args.git_repo)["approved_sql_ref"])
        return
    if args.verify_live:
        if not args.container or not args.state_dir:
            raise Refused("la verificación viva exige clon y estado")
        verify_live(DockerDB(args.container, args.state_dir.resolve()), args.repo,
                    args.git_repo, args.source_ref, args.state_dir.resolve())
        return
    if args.steps:
        if not args.state_dir:
            raise Refused("--steps exige --state-dir")
        print(json.dumps(etapas_requeridas(args.repo, args.git_repo, args.source_ref, args.state_dir)))
        return
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
