#!/usr/bin/env python3
"""Regresiones de la aprobación y del resultado incierto de AD3-132."""
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest import mock


CLI = Path(__file__).with_name("aplicar_000132_temp_public.py")
spec = importlib.util.spec_from_file_location("ad132_cli", CLI)
ad132 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ad132)


def inventory(app_login="vec_app_ejercicio", extra=None, public_temp=True):
    names = ["postgres", app_login] + ([extra] if extra else [])
    return {
        "database_name": "postgres", "database_oid": "5", "owner_name": "postgres",
        "owner_oid": "10", "allowconn": True, "acl_sha256": "a" * 64,
        "role_graph_sha256": "b" * 64, "public_temp": public_temp,
        "public_connect": True, "vec_anchors": ad132.ANCHORS,
        "non_system_schemas": ad132.SCHEMAS,
        "public_relations": ad132.PUBLIC_RELATIONS,
        "public_nonextension_functions": ad132.PUBLIC_FUNCTIONS,
        "extensions": [["pgcrypto", "public"], ["plpgsql", "pg_catalog"]],
        "functions": {"ad3_core": True, "identity_read": True},
        "login_roles": sorted(names), "connect_roles": sorted(names),
        "logins": [
            {"role": role, "connect": True, "superuser": role == "postgres",
             "memberships": ["vec_contratacion_temporal_ejecutor"] if role != "postgres" else [],
             "vec_schema_usage": ["vec_contratacion_temporal"] if role != "postgres" else [],
             "active_sessions": 0, "temp": public_temp}
            for role in sorted(names)
        ],
    }


class AD132Regression(unittest.TestCase):
    def test_plan_libre_no_sustituye_recibo_del_canario(self):
        with tempfile.TemporaryDirectory(prefix="vec-ad132-plan-") as folder:
            root = Path(folder)
            plan = {"huellas": {"incorporación/servidor.json": "a" * 64}}
            plan_path = root / "plan-conexiones.json"
            plan_path.write_bytes(ad132.canonical_json(plan))
            plan_path.chmod(0o600)
            plan_sha = ad132.sha_bytes(plan_path.read_bytes())
            receipt_path = root / "plan-canonico-clon.json"
            receipt = {"version": 1, "kind": "clon", "package_sha256": "b" * 64,
                       "source_commit": "c" * 40, "pg_container_id": "d" * 64,
                       "plan_sha256": plan_sha,
                       "material_inventory_sha256": ad132.sha_bytes(ad132.canonical_json(plan["huellas"])),
                       "canary_image_id": "sha256:" + "e" * 64,
                       "arranque_sha256": "f" * 64,
                       "canary_output_sha256": plan_sha}
            receipt_path.write_bytes(ad132.canonical_json(receipt))
            receipt_path.chmod(0o600)
            args = SimpleNamespace(plan=plan_path, plan_receipt=receipt_path,
                                   container="d" * 64)
            release = {"package_sha256": "b" * 64, "source_commit": "c" * 40}
            ad132.load_plan_receipt(args, release, plan_sha, plan)
            with self.assertRaisesRegex(ad132.Stop, "no liga plan"):
                ad132.load_plan_receipt(args, release, "0" * 64, plan)
            with self.assertRaisesRegex(ad132.Stop, "no liga plan"):
                ad132.load_plan_receipt(SimpleNamespace(plan=plan_path,
                    plan_receipt=receipt_path, container="1" * 64), release, plan_sha, plan)
            principal_path = root / "plan-canonico-principal.json"
            principal = dict(receipt, kind="principal", pg_container_id="1" * 64,
                             origin_receipt_sha256=ad132.sha_bytes(receipt_path.read_bytes()))
            principal_path.write_bytes(ad132.canonical_json(principal))
            principal_path.chmod(0o600)
            principal_args = SimpleNamespace(plan=plan_path, plan_receipt=principal_path,
                                             container="1" * 64)
            ad132.load_plan_receipt(principal_args, release, plan_sha, plan)
            receipt_path.write_bytes(receipt_path.read_bytes() + b"\n")
            with self.assertRaisesRegex(ad132.Stop, "no deriva del clon"):
                ad132.load_plan_receipt(principal_args, release, plan_sha, plan)

    def test_base_sin_ct152_no_supera_anclas(self):
        expected = {"ct145_table": "1" * 64,
                    "functions": {"ct152_attested_read": "2" * 64}}
        actual = {"ct145_table": "1" * 64,
                  "functions": {"ct152_attested_read": None}}
        release = {"artifacts": {"anchors_query": b"SELECT 1;"},
                   "anchors_expected": expected}
        with mock.patch.object(ad132, "psql", return_value=json.dumps(actual)):
            with self.assertRaisesRegex(ad132.Stop, "no instaladas"):
                ad132.current_anchors(None, release)

    def test_manifest_recortado_y_sql_mutada_se_detienen(self):
        with tempfile.TemporaryDirectory(prefix="vec-ad132-release-") as folder:
            root = Path(folder)
            commit = "d" * 40
            (root / "COMMIT").write_text(commit + "\n")
            paths = [f"deploy/postgresql/prueba/migraciones/{i:06d}_fixture.up.sql"
                     for i in range(41)] + ad132.NEW_FUNCTIONAL
            sql = b"BEGIN;\nCOMMIT;\n"
            expected = {"ct145_table": "1" * 64,
                        "functions": {name: "2" * 64 for name in
                            ("ad125_core", "ad125_consumer", "ct145_register",
                             "ct145_read", "ct152_attested_read", "ct153_reincorporation")}}
            contents = {path: sql for path in paths}
            contents.update({path: b"fijo\n" for path in ad132.RELEASE_PATHS.values()})
            contents[ad132.RELEASE_PATHS["anchors_expected"]] = ad132.canonical_json(expected)
            contents["lista_sql_h6.txt"] = ("\n".join(paths) + "\n").encode()

            def prepare(selected):
                manifest = {"version": 1, "source_commit": commit,
                    "sql_list_sha256": ad132.sha_bytes(contents["lista_sql_h6.txt"]),
                    "functional_sql": [{"path": path, "sha256": ad132.sha_bytes(contents[path])}
                                       for path in selected],
                    "ad132": {key: {"path": path, "sha256": ad132.sha_bytes(contents[path])}
                              for key, path in ad132.RELEASE_PATHS.items()},
                    "installed_anchors": expected}
                data = ad132.canonical_json(manifest)
                (root / "h6-sql-release.json").write_bytes(data)
                (root / "h6-release.lock").write_text(
                    f"COMMIT {commit}\nSQL_LIST_SHA256 {manifest['sql_list_sha256']}\n"
                    f"SQL_RELEASE_SHA256 {ad132.sha_bytes(data)}\n"
                    f"PAQUETE_SHA256 {'9' * 64}\n")
                return SimpleNamespace(release_manifest=root / "h6-sql-release.json",
                                       release_lock=root / "h6-release.lock")

            args = prepare(paths)
            with mock.patch.object(ad132, "ROOT", root), \
                 mock.patch.object(ad132, "package_file", side_effect=lambda path: contents[path]):
                ad132.load_release(args)
                contents[paths[0]] = b"BEGIN;\nSELECT 1;\nCOMMIT;\n"
                with self.assertRaisesRegex(ad132.Stop, "SQL funcional distinta"):
                    ad132.load_release(args)
                contents[paths[0]] = sql
                for key in ("sql", "cli", "inventory"):
                    path = ad132.RELEASE_PATHS[key]
                    original = contents[path]
                    contents[path] = original + b"-- cambio posterior\n"
                    with self.assertRaisesRegex(ad132.Stop, f"artefacto AD3-132 {key} cambió"):
                        ad132.load_release(args)
                    contents[path] = original
                (root / "COMMIT").write_text("0" * 40 + "\n")
                with self.assertRaisesRegex(ad132.Stop, "COMMIT del paquete distinto"):
                    ad132.load_release(args)
                (root / "COMMIT").write_text(commit + "\n")
                contents["lista_sql_h6.txt"] = ("\n".join(paths[:41]) + "\n").encode()
                args = prepare(paths[:41])
                with self.assertRaisesRegex(ad132.Stop, "causal45 exacta"):
                    ad132.load_release(args)

    def test_ticket_y_usage_no_acreditan_servicio_extra(self):
        before = inventory(extra="vec_servicio_ajeno")
        release = {"sha256": "c" * 64, "source_commit": "d" * 40,
                   "sql_list_sha256": "e" * 64, "artifact_sha256": {"sql": "f" * 64}}
        anchors = {"ct145_table": "1" * 64, "functions": {}}
        approval = {"approved": True, "approved_by": "postgres",
                    "approval_ref": "H6-TEST-TICKET", "inventory": before,
                    "plan_sha256": "2" * 64, "plan_receipt_sha256": "4" * 64,
                    "release_sha256": release["sha256"],
                    "source_commit": release["source_commit"],
                    "sql_list_sha256": release["sql_list_sha256"],
                    "artifact_sha256": release["artifact_sha256"],
                    "installed_anchors": anchors,
                    "unexplained_connect_logins": ["vec_servicio_ajeno"],
                    "other_vec_services": [{"role": "vec_servicio_ajeno",
                        "private_provision_ref": "H6-TICKET-LIBRE"}]}
        with self.assertRaisesRegex(ad132.Stop, "fuera del plan"):
            ad132.validate(before, {("vec_app_ejercicio", "vec_contratacion_temporal_ejecutor")},
                           "2" * 64, "4" * 64, release, anchors, approval)

    def test_fallo_de_transporte_tras_commit_exige_conciliacion(self):
        before = inventory()
        after = dict(before)
        after["public_temp"] = False
        after["acl_sha256"] = "3" * 64
        after["logins"] = [dict(row, temp=False) if row["role"] != "postgres" else row
                           for row in before["logins"]]
        anchors = {"ct145_table": "1" * 64, "functions": {}}
        release = {"sha256": "c" * 64, "source_commit": "d" * 40,
                   "sql_list_sha256": "e" * 64,
                   "artifact_sha256": {"sql": "f" * 64},
                   "artifacts": {"sql": b"BEGIN;\nCOMMIT;\n"},
                   "anchors_expected": anchors}
        approval = {"approved": True, "approved_by": "postgres",
                    "approval_ref": "H6-TEST-TRANSPORTE",
                    "operation": "vec.h6.ad3_132.revoke_public_temp.v1",
                    "inventory": before, "plan_sha256": "2" * 64,
                    "plan_receipt_sha256": "4" * 64,
                    "release_sha256": release["sha256"],
                    "source_commit": release["source_commit"],
                    "sql_list_sha256": release["sql_list_sha256"],
                    "artifact_sha256": release["artifact_sha256"],
                    "installed_anchors": anchors,
                    "unexplained_connect_logins": []}
        with tempfile.TemporaryDirectory(prefix="vec-ad132-test-") as folder:
            approval_path = Path(folder) / "approval.json"
            approval_path.write_text(json.dumps(approval))
            approval_path.chmod(0o600)
            argv = ["ad132", "apply", "--engine", "docker", "--container", "a" * 64,
                    "--plan", str(Path(folder) / "plan.json"),
                    "--plan-receipt", str(Path(folder) / "plan-canonico-clon.json"),
                    "--release-manifest", str(Path(folder) / "release.json"),
                    "--release-lock", str(Path(folder) / "lock"),
                    "--approval", str(approval_path)]
            with mock.patch.object(sys, "argv", argv), \
                 mock.patch.object(ad132, "load_release", return_value=release), \
                 mock.patch.object(ad132, "plan_logins", return_value=(
                     {("vec_app_ejercicio", "vec_contratacion_temporal_ejecutor")}, "2" * 64,
                     {"huellas": {}})), \
                 mock.patch.object(ad132, "load_plan_receipt", return_value="4" * 64), \
                 mock.patch.object(ad132, "current_inventory", side_effect=[before, after]), \
                 mock.patch.object(ad132, "current_anchors", return_value=anchors), \
                 mock.patch.object(ad132, "psql", side_effect=ad132.PsqlFailure("desconexión")):
                with self.assertRaisesRegex(ad132.Stop,
                        "resultado de aplicación incierto \\(postimagen compatible observada\\).*conciliar"):
                    ad132.main()


if __name__ == "__main__":
    unittest.main()
