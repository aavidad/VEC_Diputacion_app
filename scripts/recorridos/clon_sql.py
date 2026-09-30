#!/usr/bin/env python3
"""Instala el SQL H3, H4 y main sobre un clon H1 propiedad de Codex-M.

H1 se restaura antes; H5 solo configura la aplicación. No instala SQL de ramas
pendientes ni ejecuta DOWN. --plan valida todos los SHA sin acceder a Docker.
El journal privado v2 conserva confirmaciones observadas fuera de PostgreSQL.
Un pending obliga a retirar y reconstruir el clon en otro estado privado.
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
y añade una revisión del plan exclusivamente en el journal privado. Otros hashes exigen revisar
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
import uuid
from datetime import datetime, timezone

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
    """API pura: posiciones y bytes exactos; no acredita recibos de PostgreSQL."""
    if not isinstance(installed, list):
        raise Refused("confirmaciones incompatibles")
    if len(installed) > plan["file_count"] or (complete and len(installed) != plan["file_count"]):
        raise Refused("instalación incompleta o recibos ajenos al plan aprobado")
    for position, receipt in enumerate(installed, 1):
        row = plan["entries"][position - 1]
        if (not isinstance(receipt, dict) or receipt.get("position") != position
                or receipt.get("path") != row["path"] or receipt.get("sha256") != row["sha256"]):
            raise Refused("recibos incompatibles o incompletos; no reaplicar")


# Kit D no está aprobado ni disponible en este corte. El director integrará un
# proveedor revisado; ningún manifiesto privado contiene SQL ejecutable.
LIVE_KIT = None


def require_kit(kit, plan, context):
    if kit is None or not all(callable(getattr(kit, name, None)) for name in
                              ("validate", "identity", "confirm", "verify")):
        raise Refused("falta kit D aprobado y completo; no instalar ni publicar READY")
    if kit.validate(plan, context) is not True:
        raise Refused("kit D no corresponde a fuente, paquete y lista aprobados")


class ReadOnlyDB:
    """Frontera de callbacks del kit: PostgreSQL fuerza sólo lectura."""
    def __init__(self, db):
        self._db = db

    def query(self, text):
        if not isinstance(text, str) or not text.lstrip().upper().startswith("SELECT "):
            raise Refused("el kit sólo puede consultar anclas de lectura")
        return self._db.query("BEGIN READ ONLY;\n" + text + "\nCOMMIT;")


def verify_live(db, repo, git_repo, source_ref, state, context=None, kit=None):
    """RO; requiere kit real, journal completo y cierre AD132 independiente.

    verify(ro_db, record, context) debe cotejar anclas finales y evidencia de la
    CLI DBA. Nunca cambia phase ni confirma un pending.
    """
    plan = approved_source_plan(repo, source_ref, git_repo)
    require_installable(plan["approved_sql_ref"])
    with Journal(state) as journal:
        record = journal.load(required=True)
        context = validate_context(context or context_from_record(record))
        validate_record(record, plan, context)
        require_kit(kit or LIVE_KIT, plan, context)
        validate_receipts(record["installed"], plan)
        if record["phase"] != "ad132_confirmed":
            raise Refused("falta cierre independiente AD132; no publicar READY")
        db.check_owner()
        ro = ReadOnlyDB(db)
        provider = kit or LIVE_KIT
        if provider.identity(ro, context) != context["identidad_clon"]:
            raise Refused("identidad viva del clon distinta; reconstruir sin reaplicar")
        if provider.verify(ro, record, context) is not True:
            raise Refused("anclas vivas o evidencia AD132 divergentes; no publicar READY")
    return record


def etapas_requeridas(repo, git_repo, source_ref, state):
    """Consejo RO; journals v1 o pendientes jamás permiten otra UP."""
    target = approved_source_plan(repo, source_ref, git_repo)
    require_installable(target["approved_sql_ref"])
    path = plan_path(target["approved_sql_ref"])
    with Journal(state) as journal:
        record = journal.load()
        if record is None:
            if target["approved_sql_ref"] in RECOVERY_ONLY_REFS:
                raise Refused("plan 43 retirado; reconstruir con fuente aprobada")
            return list(path)
        current = validate_git_source(record["source_commit"], git_repo)
        validate_record(record, current, context_from_record(record))
        recognized = current["approved_sql_ref"]
        if recognized not in path:
            raise Refused("journal de otra familia o revisión posterior")
        index = path.index(recognized)
        if len(record["installed"]) < current["file_count"]:
            return list(path[index:])
        return list(path[index + 1:])


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


JOURNAL_VERSION = 2
MAX_JOURNAL = 2 * 1024 * 1024
CONTEXT_KEYS = ("identidad_clon", "estado_h1_sha", "package_sha", "list_sha", "release_sha")
REBUILD = "operación pendiente o incierta; conservar evidencia y retirar/reconstruir un clon nuevo; no reaplicar ni publicar READY"


def validate_context(context):
    if not isinstance(context, dict) or any(not isinstance(context.get(k), str)
            or not re.fullmatch(r"[a-f0-9]{64}", context[k]) for k in CONTEXT_KEYS):
        raise Refused("faltan identidad del clon o huellas H1/paquete/lista/release")
    return {k: context[k] for k in CONTEXT_KEYS}


def context_from_record(record):
    return {k: record.get(k) for k in CONTEXT_KEYS}


def record_hash(record):
    return sha(json.dumps({k: v for k, v in record.items() if k != "journal_sha"},
                          sort_keys=True, separators=(",", ":")).encode())


class Journal:
    """Journal privado, bloqueo único y reemplazo durable mediante dir_fd.

    Se conserva pending desde antes de enviar SQL hasta después del retorno
    COMMIT y la comprobación RO del kit. No hay reconciliación ni reUP.
    """
    def __init__(self, state):
        self.state = Path(state).absolute()
        self.directory = None
        self.lock = None

    def __enter__(self):
        try:
            for path in (*reversed(self.state.parents), self.state):
                if path.is_symlink():
                    raise Refused("directorio privado enlazado")
                if (path / ".git").exists():
                    raise Refused("journal debe quedar fuera de Git")
            # Recorrer por descriptores cierra carreras con enlaces en padres.
            self.directory = os.open("/", os.O_RDONLY | os.O_DIRECTORY)
            for component in self.state.parts[1:]:
                next_fd = os.open(component, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW,
                                  dir_fd=self.directory)
                os.close(self.directory)
                self.directory = next_fd
            status = os.fstat(self.directory)
            if status.st_uid != os.getuid() or status.st_mode & 0o077:
                raise Refused("estado privado exige propietario actual y modo 0700")
            self.lock = os.open("sql.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW | os.O_NONBLOCK,
                                0o600, dir_fd=self.directory)
            self._check_file(self.lock)
            fcntl.flock(self.lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            return self
        except BaseException:
            self.__exit__(None, None, None)
            raise

    def __exit__(self, *args):
        if self.lock is not None:
            os.close(self.lock)
            self.lock = None
        if self.directory is not None:
            os.close(self.directory)
            self.directory = None

    @staticmethod
    def _check_file(fd):
        status = os.fstat(fd)
        if (not stat.S_ISREG(status.st_mode) or status.st_nlink != 1
                or status.st_uid != os.getuid() or status.st_mode & 0o077
                or status.st_size > MAX_JOURNAL):
            raise Refused("journal/lock ajeno o fichero privado inválido")
        return status

    def load(self, required=False):
        try:
            fd = os.open("sql-journal.json", os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK,
                         dir_fd=self.directory)
        except FileNotFoundError:
            if required:
                raise Refused("falta journal externo v2; reconstruir clon nuevo")
            return None
        except OSError as error:
            raise Refused("journal privado inválido o enlazado") from error
        try:
            self._check_file(fd)
            with os.fdopen(fd, "rb", closefd=False) as file:
                raw = file.read(MAX_JOURNAL + 1)
            record = json.loads(raw)
            if not isinstance(record, dict) or record.get("version") != JOURNAL_VERSION:
                raise Refused("journal antiguo/ajeno; no convertir; reconstruir clon nuevo")
            if record.get("journal_sha") != record_hash(record):
                raise Refused("journal corrupto; conservar y reconstruir clon nuevo")
            uuid.UUID(record["run_id"])
            if record.get("pending") is not None:
                raise Refused(REBUILD)
            return record
        except (ValueError, KeyError, TypeError) as error:
            raise Refused("journal corrupto; conservar y reconstruir clon nuevo") from error
        finally:
            os.close(fd)

    def store(self, record):
        # Comprobar también el destino antes de reemplazarlo. No se convierte ni
        # se borra un archivo ajeno. El caller conserva su copia pending en RAM.
        try:
            fd = os.open("sql-journal.json", os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK,
                         dir_fd=self.directory)
        except FileNotFoundError:
            pass
        except OSError as error:
            raise Refused("journal privado inválido o enlazado") from error
        else:
            try:
                self._check_file(fd)
                old = json.loads(os.read(fd, MAX_JOURNAL + 1))
                if (not isinstance(old, dict) or old.get("version") != JOURNAL_VERSION
                        or old.get("run_id") != record["run_id"]
                        or old.get("journal_sha") != record_hash(old)):
                    raise Refused("journal ajeno o corrupto; conservar evidencia")
                for key in CONTEXT_KEYS:
                    if old.get(key) != record.get(key):
                        raise Refused("journal de otro clon/paquete; conservar evidencia")
            except (ValueError, KeyError, TypeError) as error:
                raise Refused("journal ajeno o corrupto; conservar evidencia") from error
            finally:
                os.close(fd)
        value = {**record, "journal_sha": record_hash(record)}
        data = (json.dumps(value, sort_keys=True, indent=2) + "\n").encode()
        if len(data) > MAX_JOURNAL:
            raise Refused("journal fuera de límites")
        temporary = ".sql-journal-" + uuid.uuid4().hex
        fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                     0o600, dir_fd=self.directory)
        try:
            with os.fdopen(fd, "wb") as file:
                file.write(data)
                file.flush()
                os.fsync(file.fileno())
            os.replace(temporary, "sql-journal.json", src_dir_fd=self.directory, dst_dir_fd=self.directory)
            os.fsync(self.directory)
        finally:
            try:
                os.unlink(temporary, dir_fd=self.directory)
            except FileNotFoundError:
                pass


def validate_record(record, plan, context):
    if record.get("version") != JOURNAL_VERSION:
        raise Refused("journal antiguo; reconstruir sin convertir")
    if record.get("pending") is not None:
        raise Refused(REBUILD)
    context = validate_context(context)
    if any(record.get(k) != context[k] for k in CONTEXT_KEYS):
        raise Refused("journal de otro clon/paquete; reconstruir sin reaplicar")
    for key, expected in (("source_commit", plan["source_ref"]),
                          ("approved_sql_ref", plan["approved_sql_ref"]),
                          ("plan_sha", plan["plan_sha"]),
                          ("inventory_sha", plan["inventory_sha"]),
                          ("entries", plan["entries"]), ("file_count", plan["file_count"])):
        if record.get(key) != expected:
            raise Refused("fuente/plan/journal divergentes; no reaplicar")
    validate_receipts(record.get("installed", []), plan, complete=False)
    try:
        for item in record["installed"]:
            if (datetime.fromisoformat(item["confirmed_at"]).tzinfo is None
                    or item["confirmation"] != "commit_returned_and_postcheck"):
                raise ValueError()
        complete = len(record["installed"]) == plan["file_count"]
        if record.get("phase") not in ({"awaiting_ad132", "ad132_confirmed"} if complete else {"installing"}):
            raise ValueError()
    except (ValueError, KeyError, TypeError) as error:
        raise Refused("confirmaciones o fase del journal inválidas") from error



def advance_record(record, plan, context):
    """Amplía sólo una arista causal; confirma el prefijo completo antes de UP."""
    validate_context(context)
    if any(record.get(k) != context[k] for k in CONTEXT_KEYS):
        raise Refused("journal de otro clon/paquete; reconstruir sin reaplicar")
    previous = record.get("approved_sql_ref")
    target = plan["approved_sql_ref"]
    if previous not in REF_COUNTS or record.get("plan_sha") != REF_PLAN_SHA[previous]:
        raise Refused("plan original del journal no aprobado")
    entries = record.get("entries")
    if not isinstance(entries, list) or plan_hash(entries) != REF_PLAN_SHA[previous]:
        raise Refused("plan del journal divergente")
    if previous != target:
        if REF_PARENT.get(target) != previous or entries != plan["entries"][:len(entries)]:
            raise Refused("journal conserva otra familia o extensión con saltos")
        validate_receipts(record["installed"], {"entries": entries, "file_count": REF_COUNTS[previous]})
        if target in RECOVERY_ONLY_REFS:
            raise Refused("plan 43 retirado; no ampliar")
        record["revisions"].append({"previous_ref": previous, "source_ref": target,
                                   "plan_sha": plan["plan_sha"], "file_count": plan["file_count"]})
        record["phase"] = "installing"
    elif record.get("inventory_sha") != plan["inventory_sha"]:
        raise Refused("inventario del journal divergente")
    record.update(source_commit=plan["source_ref"], current_source_ref=plan["source_ref"],
                  verified_source_ref=plan["source_ref"], approved_sql_ref=target,
                  plan_sha=plan["plan_sha"], current_plan_sha=plan["plan_sha"],
                  inventory_sha=plan["inventory_sha"], entries=plan["entries"],
                  file_count=plan["file_count"], plan_family=plan["plan_family"],
                  execution_manifest=plan["execution_manifest"])
    return record

def apply(db, rows, state, source_ref=MAIN_REF, source_plan=None, context=None, kit=None):
    """Instala exclusivamente bytes originales; journal externo nunca modifica BD.

    Contrato proveedor: validate(plan, context)->True; identity(ro_db, context)
    -> identidad_clon; confirm(ro_db, entry, position, context)->True después de
    COMMIT observado y postcotejo RO real; verify(ro_db, record, context)->True
    para anclas finales y evidencia AD132. Kit ausente/incompleto deniega.
    """
    if source_plan is None or (source_plan["source_ref"] != source_ref
            or source_plan["plan_sha"] != plan_hash(rows)):
        raise Refused("instalación exige procedencia Git y plan exactos")
    plan = source_plan
    approved = plan["approved_sql_ref"]
    require_installable(approved)
    if ([{k: v for k, v in row.items() if k != "sql"} for row in rows] != plan["entries"]
            or any(not isinstance(row.get("sql"), str)
                   or sha(row["sql"].encode()) != row["sha256"] for row in rows)):
        raise Refused("bytes SQL divergentes del plan; no ejecutar")
    if len(rows) != REF_COUNTS.get(approved) or plan_hash(rows) != REF_PLAN_SHA.get(approved):
        raise Refused("plan SQL no corresponde a su referencia aprobada")
    with Journal(state) as journal:
        record = journal.load()
        context = validate_context(context or (context_from_record(record) if record else None))
        if record:
            record = advance_record(record, plan, context)
            validate_record(record, plan, context)
        elif approved in RECOVERY_ONLY_REFS:
            raise Refused("plan 43 retirado; no convertir ledger antiguo")
        provider = kit or LIVE_KIT
        require_kit(provider, plan, context)
        db.check_owner()
        ro = ReadOnlyDB(db)
        if provider.identity(ro, context) != context["identidad_clon"]:
            raise Refused("identidad viva del clon distinta; reconstruir sin reaplicar")
        completed = len(record["installed"]) if record else 0
        last_entry = plan["entries"][completed - 1] if completed else None
        if provider.confirm(ro, last_entry, completed, context) is not True:
            raise Refused("preimagen viva distinta de H1/journal; no instalar")
        if record is None:
            record = {"version": JOURNAL_VERSION, "run_id": str(uuid.uuid4()), **context,
                      "source_commit": source_ref, "source_ref": source_ref,
                      "original_plan_sha": plan["plan_sha"],
                      "current_source_ref": source_ref, "verified_source_ref": source_ref,
                      "approved_sql_ref": approved, "plan_sha": plan["plan_sha"],
                      "current_plan_sha": plan["plan_sha"], "inventory_sha": plan["inventory_sha"],
                      "file_count": len(rows), "plan_family": plan["plan_family"],
                      "execution_manifest": plan["execution_manifest"],
                      "entries": plan["entries"], "installed": [], "pending": None,
                      "phase": "installing", "revisions": []}
        journal.store(record)
        skipped = len(record["installed"])
        for position, row in enumerate(rows, 1):
            if position <= skipped:
                continue
            entry = plan["entries"][position - 1]
            record["pending"] = {"position": position, **entry,
                                  "prepared_at": datetime.now(timezone.utc).isoformat()}
            journal.store(record)  # fsync del fichero Y del directorio antes de SQL.
            try:
                db.query(row["sql"])
                if provider.confirm(ro, entry, position, context) is not True:
                    raise Refused("postcotejo de lectura no confirmado")
            except Exception as error:
                # Incluso un retorno de error antes del COMMIT queda ambiguo:
                # no inventar recibo, no resolver pending, no intentar otra UP.
                raise Refused(REBUILD) from error
            record["installed"].append({"position": position, **entry,
                "confirmed_at": datetime.now(timezone.utc).isoformat(),
                "confirmation": "commit_returned_and_postcheck"})
            record["pending"] = None
            record["phase"] = "awaiting_ad132" if position == len(rows) else "installing"
            journal.store(record)
        return record


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--source-ref", default=H6_REF)
    parser.add_argument("--git-repo", type=Path, help="repositorio de control para ascendencia y objetos")
    parser.add_argument("--container")
    parser.add_argument("--state-dir", type=Path)
    parser.add_argument("--plan", action="store_true")
    parser.add_argument("--installable", action="store_true", help="rechaza un plan aún propuesto sin tocar el clon")
    parser.add_argument("--verify-live", action="store_true", help="coteja journal externo y anclas del kit; exige AD132 independiente")
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
    apply(DockerDB(args.container, state), rows, state, args.source_ref, plan)



if __name__ == "__main__":
    try:
        main()
    except (Refused, OSError, ValueError, subprocess.SubprocessError) as error:
        # No imprimir excepciones de subprocess ni JSON que podrían traer secretos.
        message = str(error) if isinstance(error, Refused) else type(error).__name__
        print(f"SQL-NO-GO {message}", file=sys.stderr)
        sys.exit(1)
