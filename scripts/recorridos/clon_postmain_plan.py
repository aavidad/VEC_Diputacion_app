#!/usr/bin/env python3
"""Plan documental H6+AD132 hacia un commit explícito de origin/main.

Request v4 recibe huellas externas de las listas, SQL y evidencia acompañante.
El orden documental es RPT#222, B#243, AD136 de A#219 y B2#226.
U17#230 queda diferida; cualquier SQL ajena, incluida E3#218, bloquea el plan.
Sólo lee objetos Git locales: nunca lee recibos, aplica SQL ni consulta servicios.
Las huellas comprueban integridad; el resultado siempre necesita aprobación.
La ejecución futura conserva H6 histórico y emite un segundo recibo postmain.
"""
from __future__ import annotations

import argparse
from dataclasses import dataclass
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

SOURCE = "73e56c106d12fdda0bd16d6fe573503c42c5495f"
LIST = "deploy/principal/lista_sql_trabajo_codexd_rpt_escritura_v3_20260930.txt"
LIST_B = "deploy/principal/lista_sql_codexb_ad133_135_20261001.txt"
LIST_A = "deploy/principal/lista_sql_codexa_e3_ad136_20260930.txt"
LIST_B2 = "deploy/principal/lista_sql_codexa_b2_post222_20261001.txt"
LIST_DEFERRED = "deploy/principal/lista_sql_trabajo_codexf_temas_sql_20260930.txt"
CAT = "deploy/postgresql/catalogos_configurables/"
AD = "deploy/postgresql/autorizacion_atestada_v3/"
AUT = "deploy/postgresql/autorizacion/migraciones/"
USERS = "deploy/postgresql/usuarios_vec/"
SQL_PATHS = (
    CAT + "roles_up.sql",
    CAT + "migraciones/000001_autoridad_categorias.up.sql",
    CAT + "migraciones/000002_lecturas_nominales.up.sql",
    AD + "migraciones/000117_lecturas_categorias_rpt.up.sql",
    CAT + "migraciones/000003_usos_nominales.up.sql",
    AD + "migraciones/000126_usos_categorias_rpt.up.sql",
)
B_SQL_PATHS = (
    AUT + "000017_perfil_usuarios_externo.up.sql",
    AUT + "000018_usuarios_externo_fechas_cero_canonicas.up.sql",
    AD + "migraciones/000118_consumo_usuarios_externo.up.sql",
    AUT + "000021_clausura_externa_tipos_temporales.up.sql",
    AD + "migraciones/000133_convergencia_post_rpt.up.sql",
    AD + "migraciones/000135_raiz_externa_post_convergencia.up.sql",
)
B_COMPANIONS = (
    AD + "pruebas_sql/000133_000135_capturar_preservacion.sql",
    AD + "pruebas_sql/000133_000135_comprobar_preservacion.sql",
    AD + "pruebas_sql/000133_000135_inventario_post_rpt.sql",
)
A_SQL_PATHS = (AD + "migraciones/000136_custodia_firmado_nucleo.up.sql",)
A_COMPANIONS = (
    AD + "migraciones/000136_custodia_firmado_nucleo.down.sql",
    AD + "pruebas_sql/ad3_136_custodia_firmado_nucleo.sql",
)
CTX = "deploy/postgresql/contexto_actor_v1/"
CT = "deploy/postgresql/contratacion_temporal/"
BOLSA = "deploy/postgresql/bolsa_llamamientos/"
PERSONAL = "deploy/postgresql/personal/"
B2_SQL_PATHS = (
    CTX + "migraciones/000018_persona_candidato_incorporacion.up.sql",
    AD + "migraciones/000127_consumidor_vinculo_categoria_rpt_ct.up.sql",
    CT + "migraciones/000154_vinculo_categoria_rpt_prospectivo.up.sql",
    AD + "migraciones/000128_consumidor_consulta_persona_aceptacion_ct_bolsa.up.sql",
    BOLSA + "migraciones/000067_persona_aceptacion_ct.up.sql",
    AD + "migraciones/000131_consumidor_consulta_anclaje_aceptacion_ct_bolsa.up.sql",
    BOLSA + "migraciones/000068_anclaje_aceptacion_incorporacion_ct.up.sql",
    AD + "migraciones/000129_consumidor_plan_incorporacion_personal_ct.up.sql",
    PERSONAL + "migraciones/000023_plan_incorporacion_ct.up.sql",
    PERSONAL + "migraciones/000024_catalogo_clases_multilingue.up.sql",
    AD + "migraciones/000130_consumidor_incorporacion_personal_b2.up.sql",
    CT + "migraciones/000155_incorporacion_personal_b2.up.sql",
    CT + "migraciones/000156_cese_incorporacion_personal_b2.up.sql",
)
# Lista cerrada: trece DOWN y nueve pruebas, también las dos de Personal.
B2_COMPANIONS = {
    CTX + "migraciones/000018_persona_candidato_incorporacion.down.sql": (B2_SQL_PATHS[0],),
    AD + "migraciones/000127_consumidor_vinculo_categoria_rpt_ct.down.sql": (B2_SQL_PATHS[1],),
    CT + "migraciones/000154_vinculo_categoria_rpt_prospectivo.down.sql": (B2_SQL_PATHS[2],),
    AD + "migraciones/000128_consumidor_consulta_persona_aceptacion_ct_bolsa.down.sql": (B2_SQL_PATHS[3],),
    BOLSA + "migraciones/000067_persona_aceptacion_ct.down.sql": (B2_SQL_PATHS[4],),
    AD + "migraciones/000131_consumidor_consulta_anclaje_aceptacion_ct_bolsa.down.sql": (B2_SQL_PATHS[5],),
    BOLSA + "migraciones/000068_anclaje_aceptacion_incorporacion_ct.down.sql": (B2_SQL_PATHS[6],),
    AD + "migraciones/000129_consumidor_plan_incorporacion_personal_ct.down.sql": (B2_SQL_PATHS[7],),
    PERSONAL + "migraciones/000023_plan_incorporacion_ct.down.sql": (B2_SQL_PATHS[8],),
    PERSONAL + "migraciones/000024_catalogo_clases_multilingue.down.sql": (B2_SQL_PATHS[9],),
    AD + "migraciones/000130_consumidor_incorporacion_personal_b2.down.sql": (B2_SQL_PATHS[10],),
    CT + "migraciones/000155_incorporacion_personal_b2.down.sql": (B2_SQL_PATHS[11],),
    CT + "migraciones/000156_cese_incorporacion_personal_b2.down.sql": (B2_SQL_PATHS[12],),
    CTX + "pruebas_sql/persona_candidato_incorporacion_000018.sql": (B2_SQL_PATHS[0],),
    AD + "pruebas_sql/ad3_127_131_b2_post136.sql": (
        B2_SQL_PATHS[1], B2_SQL_PATHS[3], B2_SQL_PATHS[5], B2_SQL_PATHS[7], B2_SQL_PATHS[10]),
    CT + "pruebas_sql/vinculo_categoria_rpt_ct154_acl.sql": (B2_SQL_PATHS[2],),
    BOLSA + "pruebas_sql/000067_persona_aceptacion_ct.sql": (B2_SQL_PATHS[4],),
    BOLSA + "pruebas_sql/000068_anclaje_aceptacion_incorporacion_ct.sql": (B2_SQL_PATHS[6],),
    "personal/pruebas_sql/plan_incorporacion_ct_000023.sql": (B2_SQL_PATHS[8],),
    "personal/pruebas_sql/catalogo_clases_multilingue_000024.sql": (B2_SQL_PATHS[9],),
    CT + "pruebas_sql/incorporacion_personal_b2_ct155.sql": (B2_SQL_PATHS[11],),
    CT + "pruebas_sql/ct156_cese_incorporacion_personal_b2.sql": (B2_SQL_PATHS[12],),
}
B2_PREREQUISITE_CANDIDATES = {
    "b_243": "c6b29fd4aa5d3ed384730796c3fc2a153b219313",
    "a_219": "74b2f4764689dd1fac3f29468be1c778fd4006c2",
}
DEFERRED_SQL = (
    USERS + "migraciones/000017_temas_preferencias_v2.up.sql",
    USERS + "pruebas_sql/temas_preferencias_v2.sql",
)
# Cada acompañante tiene un vínculo nominal cerrado con su UP. Nunca se ejecuta.
COMPANIONS = {
    CAT + "roles_down.sql": SQL_PATHS[0],
    CAT + "pruebas_sql/roles_up_acl_public.sql": SQL_PATHS[0],
    CAT + "migraciones/000001_autoridad_categorias.down.sql": SQL_PATHS[1],
    CAT + "pruebas_sql/000001_autoridad_categorias.sql": SQL_PATHS[1],
    CAT + "migraciones/000002_lecturas_nominales.down.sql": SQL_PATHS[2],
    CAT + "pruebas_sql/000002_lecturas_nominales.sql": SQL_PATHS[2],
    AD + "pruebas_sql/ad3_117_lecturas_categorias_rpt.sql": SQL_PATHS[3],
    CAT + "migraciones/000003_usos_nominales.down.sql": SQL_PATHS[4],
    CAT + "pruebas_sql/000003_usos_nominales.sql": SQL_PATHS[4],
    AD + "migraciones/000126_usos_categorias_rpt.down.sql": SQL_PATHS[5],
    AD + "pruebas_sql/000126_usos_categorias_rpt.sql": SQL_PATHS[5],
}
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
LIMIT = 4 * 1024 * 1024
GIT_ENV = {"PATH": "/usr/bin:/bin", "HOME": "/nonexistent", "LC_ALL": "C",
           "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null",
           "GIT_OPTIONAL_LOCKS": "0", "GIT_TERMINAL_PROMPT": "0",
           "GIT_NO_LAZY_FETCH": "1"}


class Refused(RuntimeError):
    """Código nominal; los errores de Git nunca se publican."""


@dataclass(frozen=True)
class Request:
    repo: Path
    target_commit: str
    expected_list_sha256: str
    expected_sql_sha256: tuple[str, ...]
    expected_b_list_sha256: str
    expected_b_sql_sha256: tuple[str, ...]
    expected_b_companion_sha256: tuple[str, ...]
    expected_a_list_sha256: str
    expected_a_sql_sha256: tuple[str, ...]
    expected_a_companion_sha256: tuple[str, ...]
    expected_b2_list_sha256: str
    expected_b2_sql_sha256: tuple[str, ...]
    expected_b2_companion_sha256: tuple[str, ...]
    expected_deferred_list_sha256: str
    expected_deferred_sql_sha256: tuple[str, ...]


def _require(condition, code):
    if not condition:
        raise Refused(code)


def _sha(data):
    return hashlib.sha256(data).hexdigest()


def canonical(value):
    return (json.dumps(value, sort_keys=True, ensure_ascii=True,
                       separators=(",", ":"), allow_nan=False) + "\n").encode("ascii")


def _path(value):
    _require(isinstance(value, str) and len(value) <= 512 and
             re.fullmatch(r"[A-Za-z0-9_./-]+", value) and
             not value.startswith("/") and
             all(p not in ("", ".", "..") for p in value.split("/")),
             "postmain_invalid_relative_path")
    return value


def _git(repo, *args):
    try:
        result = subprocess.run(
            ["/usr/bin/git", "--no-replace-objects", "-c", "core.fsmonitor=false",
             "-c", "core.hooksPath=/dev/null", "-c", "protocol.allow=never",
             "-C", str(repo), *args],
            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL, check=False, timeout=20, env=GIT_ENV)
    except (OSError, subprocess.TimeoutExpired):
        raise Refused("postmain_local_git_unavailable") from None
    _require(result.returncode == 0 and len(result.stdout) <= LIMIT,
             "postmain_git_read_failed")
    return result.stdout


def _commit(repo, value):
    _require(isinstance(value, str) and HEX40.fullmatch(value), "postmain_invalid_commit_sha")
    _require(_git(repo, "cat-file", "-t", value) == b"commit\n", "postmain_sha_not_commit")
    tree = _git(repo, "rev-parse", value + "^{tree}").decode("ascii").strip()
    _require(HEX40.fullmatch(tree), "postmain_invalid_tree_sha")
    return {"commit": value, "tree": tree}


def _blob(repo, commit, path, expected=None):
    _path(path)
    entry = _git(repo, "ls-tree", "-z", commit, "--", path)
    _require(entry.endswith(b"\0") and entry.count(b"\0") == 1, "postmain_blob_missing")
    head, actual = entry[:-1].split(b"\t", 1)
    mode, kind, oid = head.decode("ascii").split(" ")
    _require(actual.decode("ascii") == path and mode in ("100644", "100755") and
             kind == "blob" and HEX40.fullmatch(oid), "postmain_not_regular_blob")
    size = _git(repo, "cat-file", "-s", oid).strip()
    _require(size.isdigit() and 0 < int(size) <= LIMIT, "postmain_blob_size")
    # show conserva los bytes originales del commit; nunca usa el árbol de trabajo.
    data = _git(repo, "show", commit + ":" + path)
    _require(len(data) == int(size) and data == _git(repo, "cat-file", "blob", oid),
             "postmain_original_bytes_mismatch")
    digest = _sha(data)
    _require(expected is None or digest == expected, "postmain_expected_sha_mismatch")
    return data, {"blob": oid, "mode": mode, "bytes": len(data), "sha256": digest}


def _list_paths(data, expected_paths):
    try:
        lines = data.decode("utf-8").splitlines()
    except UnicodeError:
        raise Refused("postmain_invalid_causal_list") from None
    result = []
    for line in lines:
        if not line or line.startswith("#"):
            continue
        result.append(_path(line))
    _require(tuple(result) == expected_paths, "postmain_causal_order_mismatch")
    return result


def _diff(repo, target):
    raw = _git(repo, "diff", "--name-status", "-z", "--no-renames",
               "--no-ext-diff", "--no-textconv", "--ignore-submodules=none",
               SOURCE, target, "--", "*.sql")
    parts = raw.split(b"\0")
    _require(parts[-1] == b"" and len(parts[:-1]) % 2 == 0, "postmain_invalid_diff")
    changes, blockers = [], []
    for position in range(0, len(parts) - 1, 2):
        try:
            code = parts[position].decode("ascii")
            path = _path(parts[position + 1].decode("ascii"))
        except UnicodeError:
            raise Refused("postmain_invalid_diff_path") from None
        _require(code in {"A", "M", "D", "T"}, "postmain_invalid_diff_status")
        role = "causal_sql" if path in SQL_PATHS + B_SQL_PATHS + A_SQL_PATHS + B2_SQL_PATHS else (
            "companion_non_executable" if path in COMPANIONS or path in B_COMPANIONS + A_COMPANIONS or path in B2_COMPANIONS else (
                "deferred" if path in DEFERRED_SQL else "unknown_sql"))
        change = {"path": path, "status": {"A": "added", "M": "modified",
                  "D": "deleted", "T": "type_changed"}[code], "classification": role,
                  "linked_up": A_SQL_PATHS[0] if path in A_COMPANIONS else (
                      B2_COMPANIONS[path][0] if path in B2_COMPANIONS and
                      len(B2_COMPANIONS[path]) == 1 else COMPANIONS.get(path)),
                  "linked_ups": list(B_SQL_PATHS[4:]) if path in B_COMPANIONS else (
                      list(A_SQL_PATHS) if path in A_COMPANIONS else list(B2_COMPANIONS.get(path, ()))),
                  "executable": False, "before": None, "after": None}
        for key, commit, present in (("before", SOURCE, code != "A"),
                                     ("after", target, code != "D")):
            if present:
                try:
                    change[key] = _blob(repo, commit, path)[1]
                except Refused:
                    # Un tipo ajeno no puede convertirse en operación ejecutable.
                    blockers.append({"code": "non_regular_sql_blob", "path": path})
        if code in {"D", "T"}:
            blockers.append({"code": "deleted_or_type_changed_sql", "path": path})
        if role == "unknown_sql":
            blockers.append({"code": "unknown_sql_path", "path": path})
            if code == "M":
                blockers.append({"code": "previous_sql_modified", "path": path})
        changes.append(change)
    _require(len(changes) <= 4096, "postmain_diff_size")
    return changes, blockers


def _deferred_dependencies(repo, target):
    """Inventario del commit, sin inferir instalación ni promover U17."""
    raw = _git(repo, "ls-tree", "-r", "--name-only", "-z", target, "--",
               USERS + "migraciones/")
    _require(not raw or raw.endswith(b"\0"), "postmain_invalid_dependencies_tree")
    try:
        paths = [_path(item.decode("ascii")) for item in raw.split(b"\0") if item]
    except UnicodeError:
        raise Refused("postmain_invalid_dependencies_tree") from None
    return [{"dependency": "usuarios_vec:" + number,
             "absent_from_target": not any(Path(path).name.startswith(number + "_") and
                                            path.endswith(".up.sql") for path in paths),
             "installation": "not_read_not_validated"} for number in ("000015", "000016")]


def receipt_requirements(target):
    """Contrato futuro; describe evidencia requerida y no afirma haberla recibido."""
    common = ["package_sha256", "source_commit", "release_sha256", "lock_sha256",
              "plan_sha256", "approval_sha256", "pg_container_id", "cli_sha256",
              "apply_stdout_sha256", "postimage", "installed_anchors"]
    return {
        "status": "not_read_not_validated",
        "h6": {"required_fields": ["version", "kind", "commit", "source_ref",
            "approved_sql_ref", "plan_family", "file_count", "sql_instaladas",
            "run_id", "identidad_clon", "pg_container_id", "pg_image_id", "pg_volume",
            "sql_journal_sha256", "ad132_receipt_sha256", "package_sha", "lock_sha",
            "release_sha", "canary_plan_sha256", "canary_receipt_sha256"],
            "expected": {"version": 2, "kind": "h6_db_ready", "commit": SOURCE,
                "source_ref": SOURCE, "approved_sql_ref": SOURCE,
                "file_count": 62, "sql_instaladas": 62}},
        "ad132": {"required_fields": common, "expected": {"source_commit": SOURCE}},
        "postmain": {"required_fields": ["version", "kind", "plan_sha256", "approval_sha256",
            "h6_receipt_sha256", "ad132_receipt_sha256", "main_commit", "main_tree",
            "causal_list_count", "operation_count", "causal_lists", "operations",
            "preimages", "postimages", "stage_postimages", "pg_container_id", "apply_stdout_sha256"],
            "expected": {"version": 4, "kind": "clon_postmain_receipt",
                "main_commit": target["commit"], "main_tree": target["tree"],
                "causal_list_count": 4, "operation_count": 26},
            "bindings": ["causal_lists_bind_original_git_bytes_and_external_pins",
                "operations_bind_order_path_and_original_git_sha256",
                "preimages_and_postimages_bind_each_operation_and_preserved_sql",
                "same_pg_container_id_as_h6_and_ad132",
                "stage_postimages_bind_complete_causal_prefix_and_target_commit_tree"],
            "stage_postimages": [
                {"group": group, "operation_positions": list(range(first, last + 1)),
                 "completed_prefix_positions": list(range(1, last + 1)),
                 "required_fields": ["group", "operation_positions", "completed_prefix_positions",
                     "operations_sha256", "postimage_sha256", "preserved_sql_sha256",
                     "installed_anchors", "main_commit", "main_tree", "pg_container_id"],
                 "status": "not_read_not_validated"}
                for group, first, last in (("rpt_222", 1, 6), ("b_243", 7, 12),
                    ("a_219", 13, 13), ("b2_226", 14, 26))],
            "b2_prerequisite_candidates": B2_PREREQUISITE_CANDIDATES.copy()},
        "cross_equal": [["h6.pg_container_id", "ad132.pg_container_id"],
            ["h6.commit", "ad132.source_commit"],
            ["h6.package_sha", "ad132.package_sha256"],
            ["h6.lock_sha", "ad132.lock_sha256"],
            ["h6.release_sha", "ad132.release_sha256"],
            ["h6.canary_plan_sha256", "ad132.plan_sha256"]],
        "digest_binding": ["sha256(ad132_original_bytes) == h6.ad132_receipt_sha256",
                           "sha256(h6_original_bytes) == external_approval.h6_receipt_sha256"],
        "future_gates": ["external_approval_binds_plan_sha256_and_both_receipt_sha256",
            "h6_live_validation_before_transition", "ad132_live_validation_before_transition",
            "installed_sql_inventory_matches_h6_journal_and_original_package",
            "no_history_for_twenty_six_causal_sql", "second_postmain_receipt_binds_h6_and_ad132",
            "external_approval_binds_four_causal_lists_and_target_commit_tree",
            "ad136_requires_complete_rpt6_then_b6_postimage",
            "ad136_down_and_probe_never_execute_in_postmain",
            "b2_requires_complete_rpt6_b6_ad136_postimages_in_causal_order",
            "b2_ad127_requires_nucleus_post136_preimage",
            "ad136_probe_is_not_final_b2_postimage",
            "b2_down_and_probes_never_execute_in_postmain",
            "deferred_u17_never_executes_in_postmain",
            "preserve_h6_historical_receipt_and_live_validator"],
    }


def build_plan(request: Request) -> dict:
    _require(type(request) is Request and isinstance(request.repo, Path), "postmain_invalid_request")
    repo = request.repo
    _require(repo.is_absolute() and ".." not in repo.parts and repo.is_dir() and
             not any(p.is_symlink() for p in (repo, *repo.parents)), "postmain_invalid_repo_path")
    for pin in (request.expected_list_sha256, request.expected_b_list_sha256,
                request.expected_a_list_sha256, request.expected_b2_list_sha256,
                request.expected_deferred_list_sha256):
        _require(isinstance(pin, str) and HEX64.fullmatch(pin), "postmain_invalid_expected_hashes")
    for pins, size in ((request.expected_sql_sha256, 6), (request.expected_b_sql_sha256, 6),
                       (request.expected_b_companion_sha256, 3),
                       (request.expected_a_sql_sha256, 1),
                       (request.expected_a_companion_sha256, 2),
                       (request.expected_b2_sql_sha256, 13),
                       (request.expected_b2_companion_sha256, 22),
                       (request.expected_deferred_sql_sha256, 2)):
        _require(type(pins) is tuple and len(pins) == size and
                 all(isinstance(pin, str) and HEX64.fullmatch(pin) for pin in pins),
                 "postmain_invalid_expected_hashes")
    source = _commit(repo, SOURCE)
    target = _commit(repo, request.target_commit)
    origin = _git(repo, "rev-parse", "--verify", "refs/remotes/origin/main").decode("ascii").strip()
    _require(origin == target["commit"], "postmain_target_not_origin_main")
    _git(repo, "merge-base", "--is-ancestor", SOURCE, target["commit"])
    operations, lists = [], []
    for group, list_path, paths, list_pin, sql_pins in (
        ("rpt_222", LIST, SQL_PATHS, request.expected_list_sha256, request.expected_sql_sha256),
        ("b_243", LIST_B, B_SQL_PATHS, request.expected_b_list_sha256, request.expected_b_sql_sha256),
        ("a_219", LIST_A, A_SQL_PATHS, request.expected_a_list_sha256, request.expected_a_sql_sha256),
        ("b2_226", LIST_B2, B2_SQL_PATHS, request.expected_b2_list_sha256, request.expected_b2_sql_sha256)):
        data, listing = _blob(repo, target["commit"], list_path, list_pin)
        _list_paths(data, paths)
        lists.append({"group": group, "path": list_path, **listing})
        for path, expected in zip(paths, sql_pins, strict=True):
            _, record = _blob(repo, target["commit"], path, expected)
            operations.append({"position": len(operations) + 1, "group": group,
                               "path": path, "executable": False, **record})
    companions = [{"path": path, "linked_ups": list(B_SQL_PATHS[4:]),
                   "classification": "companion_non_executable", "executable": False,
                   **_blob(repo, target["commit"], path, pin)[1]}
                  for path, pin in zip(B_COMPANIONS, request.expected_b_companion_sha256, strict=True)]
    companions.extend({"path": path, "linked_up": A_SQL_PATHS[0], "linked_ups": list(A_SQL_PATHS),
        "classification": "companion_non_executable", "executable": False,
        **_blob(repo, target["commit"], path, pin)[1]}
        for path, pin in zip(A_COMPANIONS, request.expected_a_companion_sha256, strict=True))
    companions.extend({"path": path, "linked_up": ups[0] if len(ups) == 1 else None,
        "linked_ups": list(ups), "classification": "companion_non_executable", "executable": False,
        **_blob(repo, target["commit"], path, pin)[1]}
        for (path, ups), pin in zip(B2_COMPANIONS.items(), request.expected_b2_companion_sha256, strict=True))
    data, deferred_list = _blob(repo, target["commit"], LIST_DEFERRED,
                                request.expected_deferred_list_sha256)
    _list_paths(data, DEFERRED_SQL[:1])
    dependencies = _deferred_dependencies(repo, target["commit"])
    reason = ("usuarios_15_16_absent_from_target" if all(item["absent_from_target"] for item in dependencies)
              else "usuarios_15_16_not_validated_outside_postmain_scope")
    deferred = [{"path": path, "classification": "deferred", "executable": False,
                 "reason": reason,
                 "dependencies": dependencies, **_blob(repo, target["commit"], path, pin)[1]}
                for path, pin in zip(DEFERRED_SQL, request.expected_deferred_sql_sha256, strict=True)]
    changes, blockers = _diff(repo, target["commit"])
    _require(_git(repo, "rev-parse", "--verify", "refs/remotes/origin/main").decode("ascii").strip()
             == origin, "postmain_origin_main_changed")
    return {"version": 4, "kind": "clon_postmain_plan", "plan": "pending_approval",
        "executable": False, "sql_invoked": False, "source": source, "target": target,
        "origin_main_observed": origin, "causal_lists": lists,
        "operations": operations, "companions": companions, "deferred": deferred,
        "deferred_list": {"path": LIST_DEFERRED, **deferred_list},
        "sql_diff": changes, "blockers": blockers,
        "receipt_requirements": receipt_requirements(target)}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", required=True, type=Path)
    parser.add_argument("--target-commit", required=True)
    parser.add_argument("--expected-list-sha256", required=True)
    parser.add_argument("--expected-sql-sha256", required=True, action="append")
    parser.add_argument("--expected-b-list-sha256", required=True)
    parser.add_argument("--expected-b-sql-sha256", required=True, action="append")
    parser.add_argument("--expected-b-companion-sha256", required=True, action="append")
    parser.add_argument("--expected-a-list-sha256", required=True)
    parser.add_argument("--expected-a-sql-sha256", required=True, action="append")
    parser.add_argument("--expected-a-companion-sha256", required=True, action="append")
    parser.add_argument("--expected-b2-list-sha256", required=True)
    parser.add_argument("--expected-b2-sql-sha256", required=True, action="append")
    parser.add_argument("--expected-b2-companion-sha256", required=True, action="append")
    parser.add_argument("--expected-deferred-list-sha256", required=True)
    parser.add_argument("--expected-deferred-sql-sha256", required=True, action="append")
    args = parser.parse_args(argv)
    value = build_plan(Request(args.repo, args.target_commit, args.expected_list_sha256,
                               tuple(args.expected_sql_sha256), args.expected_b_list_sha256,
                               tuple(args.expected_b_sql_sha256), tuple(args.expected_b_companion_sha256),
                               args.expected_a_list_sha256, tuple(args.expected_a_sql_sha256),
                               tuple(args.expected_a_companion_sha256),
                               args.expected_b2_list_sha256, tuple(args.expected_b2_sql_sha256),
                               tuple(args.expected_b2_companion_sha256),
                               args.expected_deferred_list_sha256, tuple(args.expected_deferred_sql_sha256)))
    sys.stdout.buffer.write(canonical(value))
    return 2 if value["blockers"] else 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Refused as error:
        print(str(error), file=sys.stderr)
        raise SystemExit(1)
