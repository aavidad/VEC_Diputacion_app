"""Pruebas focales de integridad, recuperación y propiedad del clon."""

import importlib.util
import contextlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("clon_sql", Path(__file__).with_name("clon_sql.py"))
SQL = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SQL)
REPO = Path(os.environ.get("VEC_CLON_SQL_TEST_REPO", Path(__file__).resolve().parents[2]))


class HelperTests(unittest.TestCase):
    def test_main_interno_firma_amplia_41_a_44_y_retiene_sql_exterior(self):
        previous = SQL.load_plan(REPO, source_ref=SQL.H6_REF)
        rows = SQL.load_plan(REPO, source_ref=SQL.H6_FIRMA_REF)
        self.assertEqual(rows[:41], previous)
        self.assertEqual([Path(row["path"]).name for row in rows[41:]], [
            "000145_enlace_firma_documento_custodiado.up.sql",
            "000125_consumidor_consulta_firmas_documento_ct.up.sql",
            "000152_consulta_firmas_documento_atestada.up.sql",
        ])
        self.assertEqual(SQL.plan_hash(rows), SQL.REF_PLAN_SHA[SQL.H6_FIRMA_REF])
        self.assertEqual(len(SQL.H6_FIRMA_WITHHELD), 23)
        self.assertTrue(SQL.H6_FIRMA_WITHHELD.keys().isdisjoint({r["path"] for r in rows}))
        source = SQL.GitSource(REPO)
        inventory_before, _ = source.inventory(SQL.H6_REF)
        inventory_after, _ = source.inventory(SQL.H6_FIRMA_REF)
        up_paths = lambda entries: {item["path"] for item in entries
                                    if item["path"].endswith((".up.sql", "_up.sql"))}
        self.assertEqual(up_paths(inventory_after) - up_paths(inventory_before),
                         {r["path"] for r in rows[41:]} |
                         (SQL.H6_FIRMA_WITHHELD.keys() - SQL.H6_WITHHELD.keys()))
        self.assertEqual(SQL.plan_path(SQL.H6_FIRMA_REF)[-2:], (SQL.H6_REF, SQL.H6_FIRMA_REF))
        plan = SQL.validate_git_source(SQL.H6_FIRMA_REF, REPO)
        self.assertEqual((plan["file_count"], plan["plan_family"], plan["execution_manifest"], plan["status"]),
                         (44, "h6_44", "sql_main_h6_firma.txt", "proposed"))
        for relative in SQL.H6_FIRMA_WITHHELD:
            with self.subTest(relative=relative):
                contents = {r["path"]: r["sql"].encode() for r in rows}
                contents.update({p: (REPO / p).read_bytes() for p in SQL.H6_FIRMA_WITHHELD})
                contents[relative] = b"cambio no aprobado"
                with self.assertRaisesRegex(SQL.Refused, "retenida"):
                    SQL.load_plan(None, source_ref=SQL.H6_FIRMA_REF, contents=contents)

    def test_h6_manifest_has_39_exact_prefix_and_only_ct150_ct151(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.H6_REF)
        old = SQL.load_plan(REPO)
        self.assertEqual(len(rows), 41)
        self.assertEqual(rows[:39], old[:39])
        for ref in SQL.plan_path(SQL.FIFTH_REF):
            self.assertEqual(rows[:SQL.REF_COUNTS[ref]], SQL.load_plan(REPO, source_ref=ref))
        self.assertEqual([Path(row["path"]).name for row in rows[39:]],
                         ["000150_entrega_peticion_perfil_fijo.up.sql", "000151_entrega_estado_http.up.sql"])
        self.assertEqual([row["sha256"] for row in rows[39:]],
                         ["cb3f114cd1b40afb3c044a14640cfa4f1d4f86053a689acb39a7cc122ca60249",
                          "68c8073c0b50c90674b3659f30d67bd972612c7b724ccacdd50fbad3475f1842"])
        self.assertEqual(SQL.plan_hash(rows), SQL.REF_PLAN_SHA[SQL.H6_REF])
        self.assertEqual({r["path"]: r["sha256"] for r in old[39:]}, SQL.H6_WITHHELD)
        self.assertEqual(len({r["path"] for r in rows} | SQL.H6_WITHHELD.keys()), 45)
        self.assertTrue(SQL.H6_WITHHELD.keys().isdisjoint({r["path"] for r in rows}))
        with self.assertRaisesRegex(SQL.Refused, "manifiesto"):
            SQL.load_plan(REPO, SQL.MANIFEST, source_ref=SQL.H6_REF)

    def test_h6_manifest_rejects_trailing_entry_and_retained_change(self):
        with tempfile.TemporaryDirectory() as scratch:
            manifest = Path(scratch) / "manifest.txt"
            manifest.write_text(SQL.H6_MANIFEST.read_text() + SQL.MANIFEST.read_text().splitlines()[-1] + "\n")
            with self.assertRaisesRegex(SQL.Refused, "41 entradas"):
                SQL.load_plan(REPO, manifest, source_ref=SQL.H6_REF)
        contents = {r["path"]: r["sql"].encode() for r in SQL.load_plan(REPO, source_ref=SQL.H6_REF)}
        contents.update({path: (REPO / path).read_bytes() for path in SQL.H6_WITHHELD})
        for path in SQL.H6_WITHHELD:
            with self.subTest(path=path), self.assertRaisesRegex(SQL.Refused, "retenida"):
                SQL.load_plan(None, source_ref=SQL.H6_REF, contents={**contents, path: b"modificado"})

    def test_real_plan_pins_all_43_existing_files_and_preserves_prefixes(self):
        rows = SQL.load_plan(REPO)
        self.assertEqual(len(rows), 43)
        self.assertEqual([len([r for r in rows if r["phase"] == phase])
                          for phase in ("H3", "H4", "MAIN")], [8, 9, 26])
        roles = [r["path"] for r in rows if "/roles" in r["path"]]
        self.assertEqual(len(roles), 5)
        self.assertEqual(rows[:33], SQL.load_plan(REPO, source_ref=SQL.BASE_REF))
        self.assertEqual(rows[:34], SQL.load_plan(REPO, source_ref=SQL.PREVIOUS_REF))
        self.assertEqual(rows[:36], SQL.load_plan(REPO, source_ref=SQL.THIRD_REF))
        self.assertEqual(rows[:38], SQL.load_plan(REPO, source_ref=SQL.FOURTH_REF))
        self.assertEqual(rows[:39], SQL.load_plan(REPO, source_ref=SQL.FIFTH_REF))
        self.assertEqual(SQL.plan_hash(rows[:33]), SQL.REF_PLAN_SHA[SQL.BASE_REF])

    def test_modified_sql_fails_before_database_access(self):
        with tempfile.TemporaryDirectory() as scratch:
            root = Path(scratch)
            row = SQL.load_plan(REPO)[0]
            source = root / row["path"]
            source.parent.mkdir(parents=True)
            source.write_text(row["sql"] + "-- modificación\n")
            manifest = root / "manifest.txt"
            manifest.write_text(f"H3 {row['sha256']} {row['path']}\n")
            with self.assertRaisesRegex(SQL.Refused, "SHA incompatible"):
                SQL.load_plan(root, manifest)

    def test_down_or_path_escape_cannot_enter_plan(self):
        with tempfile.TemporaryDirectory() as scratch:
            root = Path(scratch)
            manifest = root / "manifest.txt"
            for path in ("deploy/postgresql/mod/migraciones/000001.down.sql",
                         "deploy/postgresql/../../secret.up.sql"):
                manifest.write_text(f"MAIN {'a' * 64} {path}\n")
                with self.assertRaisesRegex(SQL.Refused, "ruta SQL"):
                    SQL.load_plan(root, manifest)

    def test_atomic_receipt_precedes_original_commit_for_all_sql(self):
        for position, row in enumerate(SQL.load_plan(REPO), 1):
            text = SQL.instrument(row, position)
            self.assertEqual(text.count("INSERT INTO vec_recorridos_clon.applied"), 1)
            self.assertTrue(text.endswith("COMMIT;\n"))
            self.assertIn(f"<> {position - 1}", text)
            # El marcador ejecuta como postgres, nunca como propietario del módulo.
            self.assertIn("RESET ROLE;\nINSERT INTO vec_recorridos_clon.applied", text)

    def test_receipt_hash_mismatch_and_gaps_are_rejected(self):
        rows = SQL.load_plan(REPO)
        class Database:
            def query(self, _):
                return json.dumps(installed)
        installed = [{"position": 1, "path": rows[0]["path"], "sha256": "a" * 64}]
        with self.assertRaises(SQL.Refused):
            SQL.receipts(Database(), rows)
        installed[0]["sha256"] = rows[0]["sha256"]
        installed[0]["position"] = 2
        with self.assertRaises(SQL.Refused):
            SQL.receipts(Database(), rows)

    def test_journal_recovered_from_database_but_never_ahead(self):
        with tempfile.TemporaryDirectory() as scratch:
            root = Path(scratch)
            meta = {"run_id": "clon1", "plan_sha": "plan1"}
            installed = [{"position": 1, "sha256": "a" * 64}]
            SQL.write_journal(root, meta, installed)
            self.assertEqual((root / "sql-journal.json").stat().st_mode & 0o777, 0o600)
            (root / "sql-journal.json").unlink()
            SQL.write_journal(root, meta, installed)
            with self.assertRaisesRegex(SQL.Refused, "ausente"):
                SQL.write_journal(root, meta, [])
            with self.assertRaisesRegex(SQL.Refused, "otro clon"):
                SQL.write_journal(root, {**meta, "run_id": "clon2"}, installed)

    def test_refuses_container_owned_by_someone_else(self):
        obj = {"Config": {"Labels": {SQL.OWNER_LABEL: "otra-persona"}},
               "State": {"Running": True}}
        result = subprocess.CompletedProcess([], 0, json.dumps([obj]), "")
        with patch.object(SQL.subprocess, "run", return_value=result):
            with self.assertRaisesRegex(SQL.Refused, "propiedad"):
                SQL.DockerDB("vec-codexm-test").check_owner()
        with self.assertRaises(SQL.Refused):
            SQL.DockerDB("clon-ajeno")

    def test_refuses_other_main_hash(self):
        with self.assertRaisesRegex(SQL.Refused, "Git no acredita"):
            SQL.main(["--repo", str(REPO), "--source-ref", "b" * 40, "--plan"])

    def test_database_error_does_not_disclose_detail_or_rows(self):
        result = subprocess.CompletedProcess([], 1, "", "ERROR:  42501: denegado\n"
                                             "DETAIL: dato privado de una fila\n")
        with patch.object(SQL.subprocess, "run", return_value=result):
            with self.assertRaises(SQL.Refused) as error:
                SQL.DockerDB("vec-codexm-test").query("SELECT 1;")
        self.assertIn("42501", str(error.exception))
        self.assertNotIn("privado", str(error.exception))

    def test_database_command_pins_local_socket_and_port(self):
        result = subprocess.CompletedProcess([], 0, "1\n", "")
        with patch.object(SQL.subprocess, "run", return_value=result) as run:
            self.assertEqual(SQL.DockerDB("vec-test").query("SELECT 1;"), "1")
        command = run.call_args.args[0]
        self.assertEqual(command[command.index("-h") + 1], "/var/run/postgresql")
        self.assertEqual(command[command.index("-p") + 1], "5432")

    def test_clon_exposed_outside_loopback_is_refused(self):
        obj = {"Config": {"Image": "postgres:18.4", "Labels": {SQL.OWNER_LABEL: SQL.OWNER}},
               "State": {"Running": True},
               "Mounts": [{"Type": "bind", "Destination": "/var/lib/postgresql",
                           "Source": "/dev/shm/clon"}],
               "NetworkSettings": {"Ports": {"5432/tcp": [{"HostIp": "0.0.0.0"}]}}}
        result = subprocess.CompletedProcess([], 0, json.dumps([obj]), "")
        with patch.object(SQL.subprocess, "run", return_value=result):
            with self.assertRaisesRegex(SQL.Refused, "loopback"):
                SQL.DockerDB("vec-codexm-test").check_owner()

    def test_generic_vec_name_is_accepted_and_state_label_is_checked(self):
        state = Path("/private/clon")
        obj = {"Config": {"Image": "postgres:18.4", "Labels": {
                   SQL.OWNER_LABEL: SQL.OWNER, "vec.recorridos.state": str(state)}},
               "State": {"Running": True},
               "Mounts": [{"Type": "bind", "Destination": "/var/lib/postgresql",
                           "Source": "/dev/shm/clon"}]}
        result = subprocess.CompletedProcess([], 0, json.dumps([obj]), "")
        with patch.object(SQL.subprocess, "run", return_value=result):
            SQL.DockerDB("vec-recorridos-local", state).check_owner()
            with self.assertRaisesRegex(SQL.Refused, "label de estado"):
                SQL.DockerDB("vec-recorridos-local", Path("/other/clon")).check_owner()

    def test_extension_preserves_original_metadata_and_acknowledges_new_ref(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.PREVIOUS_REF)
        original = {"run_id": "clon1", "source_ref": SQL.BASE_REF,
                    "plan_sha": SQL.plan_hash(rows[:33])}
        revision = {"revision": 2, "source_ref": SQL.PREVIOUS_REF,
                    "plan_sha": SQL.plan_hash(rows), "file_count": 34,
                    "acknowledged_at": "2026-09-30T01:00:00Z"}
        answers = ["f", json.dumps([{"position": n, "path": row["path"],
                    "sha256": row["sha256"]} for n, row in enumerate(rows[:33], 1)]),
                   "", "t", json.dumps([revision])]
        with patch.object(SQL.DockerDB, "query", side_effect=answers) as queries:
            result = SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.PREVIOUS_REF, original)
        for key, value in original.items():
            self.assertEqual(result[key], value)
        self.assertEqual(result["current_source_ref"], SQL.PREVIOUS_REF)
        self.assertEqual(result["revisions"], [revision])
        mutation = queries.call_args_list[2].args[0]
        self.assertNotIn("UPDATE", mutation)
        self.assertNotIn("DELETE", mutation)
        self.assertIn("CREATE TABLE vec_recorridos_clon.plan_revisions", mutation)

    def test_extension_refuses_changed_prefix_or_incomplete_base(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.PREVIOUS_REF)
        meta = {"run_id": "clon1", "source_ref": SQL.BASE_REF, "plan_sha": "a" * 64}
        with self.assertRaisesRegex(SQL.Refused, "prefijo"):
            SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.PREVIOUS_REF, meta)
        meta["plan_sha"] = SQL.plan_hash(rows[:33])
        with patch.object(SQL.DockerDB, "query", side_effect=["f", "[]"]):
            with self.assertRaisesRegex(SQL.Refused, "33 SQL"):
                SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.PREVIOUS_REF, meta)

    def test_extension_cannot_downgrade_or_lose_revision_from_journal(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.BASE_REF)
        original = {"run_id": "clon1", "source_ref": SQL.BASE_REF,
                    "plan_sha": SQL.plan_hash(rows)}
        rev = {"revision": 2, "source_ref": SQL.PREVIOUS_REF,
               "plan_sha": SQL.REF_PLAN_SHA[SQL.PREVIOUS_REF], "file_count": 34}
        with patch.object(SQL.DockerDB, "query", side_effect=["t", json.dumps([rev])]):
            with self.assertRaisesRegex(SQL.Refused, "volver a la base"):
                SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.BASE_REF, original)
        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            SQL.write_journal(state, {**original, "revisions": [rev]}, [])
            with self.assertRaisesRegex(SQL.Refused, "revisión"):
                SQL.write_journal(state, {**original, "revisions": []}, [])

    def test_third_revision_keeps_second_revision_and_receipt_prefix(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.THIRD_REF)
        original = {"run_id": "clon1", "source_ref": SQL.BASE_REF,
                    "plan_sha": SQL.plan_hash(rows[:33])}
        rev2 = {"revision": 2, "source_ref": SQL.PREVIOUS_REF,
                "plan_sha": SQL.plan_hash(rows[:34]), "file_count": 34,
                "acknowledged_at": "2026-09-30T01:00:00Z"}
        rev3 = {"revision": 3, "source_ref": SQL.THIRD_REF,
                "plan_sha": SQL.plan_hash(rows), "file_count": 36,
                "acknowledged_at": "2026-09-30T02:00:00Z"}
        installed = [{"position": n, "path": row["path"], "sha256": row["sha256"]}
                     for n, row in enumerate(rows[:34], 1)]
        answers = ["t", json.dumps([rev2]), json.dumps(installed), "",
                   "t", json.dumps([rev2, rev3])]
        with patch.object(SQL.DockerDB, "query", side_effect=answers) as queries:
            result = SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.THIRD_REF, original)
        self.assertEqual(result["revisions"], [rev2, rev3])
        self.assertEqual(result["current_source_ref"], SQL.THIRD_REF)
        for key, value in original.items():
            self.assertEqual(result[key], value)
        mutation = queries.call_args_list[3].args[0]
        self.assertIn("ALTER TABLE vec_recorridos_clon.plan_revisions", mutation)
        self.assertIn("(revision=2 AND file_count=34) OR (revision=3 AND file_count=36)", mutation)
        self.assertNotIn("UPDATE", mutation)
        self.assertNotIn("DELETE", mutation)
        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            SQL.write_journal(state, {**original, "revisions": [rev2]}, installed)
            SQL.write_journal(state, result, installed)
            self.assertEqual(json.loads((state / "sql-journal.json").read_text())["installed"], installed)

    def test_third_revision_refuses_gaps_in_history_or_incomplete_34(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.THIRD_REF)
        meta = {"run_id": "clon1", "source_ref": SQL.BASE_REF,
                "plan_sha": SQL.plan_hash(rows[:33])}
        rev2 = {"revision": 2, "source_ref": SQL.PREVIOUS_REF,
                "plan_sha": SQL.plan_hash(rows[:34]), "file_count": 34}
        with patch.object(SQL.DockerDB, "query", side_effect=["f"]):
            with self.assertRaisesRegex(SQL.Refused, "intermedia"):
                SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.THIRD_REF, meta)
        with patch.object(SQL.DockerDB, "query", side_effect=["t", json.dumps([rev2]), "[]"]):
            with self.assertRaisesRegex(SQL.Refused, "34 SQL"):
                SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.THIRD_REF, meta)
        bad = {**rev2, "revision": 3, "source_ref": SQL.THIRD_REF,
               "plan_sha": SQL.plan_hash(rows), "file_count": 36}
        with patch.object(SQL.DockerDB, "query", side_effect=["t", json.dumps([bad])]):
            with self.assertRaisesRegex(SQL.Refused, "incompatibles"):
                SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.THIRD_REF, meta)

    def test_complete_receipts_requires_all_38_and_hashes(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.FOURTH_REF)
        plan = {"file_count": 38, "entries": rows}
        installed = [{"position": n, "path": row["path"], "sha256": row["sha256"]}
                     for n, row in enumerate(rows, 1)]
        SQL.validate_receipts(installed, plan)
        with self.assertRaisesRegex(SQL.Refused, "incompleta"):
            SQL.validate_receipts(installed[:37], plan)
        installed[-1]["sha256"] = "a" * 64
        with self.assertRaisesRegex(SQL.Refused, "incompatibles"):
            SQL.validate_receipts(installed, plan)

    def test_fourth_revision_appends_to_36_without_changing_prior_history(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.FOURTH_REF)
        original = {"run_id": "clon1", "source_ref": SQL.BASE_REF,
                    "plan_sha": SQL.plan_hash(rows[:33])}
        revisions = [{"revision": n, "source_ref": ref, "plan_sha": SQL.plan_hash(rows[:count]),
                      "file_count": count, "acknowledged_at": f"2026-09-30T0{n}:00:00Z"}
                     for n, ref, count in ((2, SQL.PREVIOUS_REF, 34), (3, SQL.THIRD_REF, 36))]
        new = {"revision": 4, "source_ref": SQL.FOURTH_REF, "plan_sha": SQL.plan_hash(rows),
               "file_count": 38, "acknowledged_at": "2026-09-30T04:00:00Z"}
        installed = [{"position": n, "path": row["path"], "sha256": row["sha256"]}
                     for n, row in enumerate(rows[:36], 1)]
        answers = ["t", json.dumps(revisions), json.dumps(installed), "",
                   "t", json.dumps([*revisions, new])]
        with patch.object(SQL.DockerDB, "query", side_effect=answers) as queries:
            result = SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.FOURTH_REF, original)
        self.assertEqual(result["revisions"][:2], revisions)
        self.assertEqual(result["revisions"][-1], new)
        for key, value in original.items():
            self.assertEqual(result[key], value)
        mutation = queries.call_args_list[3].args[0]
        self.assertIn("(revision=4 AND file_count=38)", mutation)
        self.assertNotIn("UPDATE", mutation)
        self.assertNotIn("DELETE", mutation)

    def test_code_descendant_replay_changes_only_private_provenance(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.FOURTH_REF)
        original = {"run_id": "clon1", "source_ref": SQL.BASE_REF,
                    "plan_sha": SQL.plan_hash(rows[:33]), "revisions": [{"revision": 4}],
                    "current_source_ref": SQL.FOURTH_REF, "current_plan_sha": SQL.plan_hash(rows)}
        plan = {"approved_sql_ref": SQL.FOURTH_REF, "source_ref": "a" * 40,
                "verified_main_ref": "b" * 40, "inventory_sha": "c" * 64,
                "plan_sha": SQL.plan_hash(rows)}
        installed = [{"position": n, "path": row["path"], "sha256": row["sha256"],
                      "installed_at": "2026-09-30T04:00:00Z"}
                     for n, row in enumerate(rows, 1)]
        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            with patch.object(SQL.DockerDB, "check_owner"), patch.object(SQL.DockerDB, "query") as query:
                with patch.object(SQL, "initialize", return_value=original.copy()) as initialize:
                    with patch.object(SQL, "receipts", return_value=installed):
                        SQL.apply(SQL.DockerDB("vec-test"), rows, state, plan["source_ref"], plan)
                        first = (state / "sql-journal.json").read_bytes()
                        (state / "sql-journal.json").unlink()
                        SQL.apply(SQL.DockerDB("vec-test"), rows, state, plan["source_ref"], plan)
            query.assert_not_called()
            self.assertEqual(initialize.call_args.args[2], SQL.FOURTH_REF)
            self.assertEqual(first, (state / "sql-journal.json").read_bytes())
            journal = json.loads(first)
            self.assertEqual(journal["installed"], installed)
            self.assertEqual(journal["revisions"], original["revisions"])
            self.assertEqual(journal["source_ref"], SQL.BASE_REF)
            self.assertEqual(journal["current_source_ref"], plan["source_ref"])
            self.assertEqual(journal["approved_sql_ref"], SQL.FOURTH_REF)

    def test_fifth_revision_preserves_38_receipts_and_all_prior_revisions(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.FIFTH_REF)
        original = {"run_id": "clon1", "source_ref": SQL.BASE_REF,
                    "plan_sha": SQL.plan_hash(rows[:33])}
        revisions = [{"revision": n, "source_ref": ref, "plan_sha": SQL.plan_hash(rows[:count]),
                      "file_count": count, "acknowledged_at": f"2026-09-30T0{n}:00:00Z"}
                     for n, ref, count in ((2, SQL.PREVIOUS_REF, 34), (3, SQL.THIRD_REF, 36),
                                          (4, SQL.FOURTH_REF, 38))]
        new = {"revision": 5, "source_ref": SQL.FIFTH_REF, "plan_sha": SQL.plan_hash(rows),
               "file_count": 39, "acknowledged_at": "2026-09-30T05:00:00Z"}
        installed = [{"position": n, "path": row["path"], "sha256": row["sha256"]}
                     for n, row in enumerate(rows[:38], 1)]
        answers = ["t", json.dumps(revisions), json.dumps(installed), "",
                   "t", json.dumps([*revisions, new])]
        with patch.object(SQL.DockerDB, "query", side_effect=answers) as queries:
            result = SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.FIFTH_REF, original)
        self.assertEqual(result["revisions"][:3], revisions)
        self.assertEqual(result["revisions"][-1], new)
        for key, value in original.items():
            self.assertEqual(result[key], value)
        mutation = queries.call_args_list[3].args[0]
        self.assertIn("(revision=5 AND file_count=39)", mutation)
        self.assertNotIn("UPDATE", mutation)
        self.assertNotIn("DELETE", mutation)
        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            SQL.write_journal(state, {**original, "revisions": revisions}, installed)
            SQL.write_journal(state, result, installed)
            self.assertEqual(json.loads((state / "sql-journal.json").read_text())["installed"], installed)

    def planner_fixture(self, approved=SQL.FOURTH_REF, target=SQL.MAIN_REF):
        rows = SQL.load_plan(REPO, source_ref=approved)
        target_rows = SQL.load_plan(REPO, source_ref=target)
        count = SQL.REF_COUNTS[approved]
        target_count = SQL.REF_COUNTS[target]
        plan = {"source_ref": target, "approved_sql_ref": target,
                "file_count": target_count, "entries": target_rows,
                "plan_family": SQL.plan_family(target),
                "execution_manifest": SQL.execution_manifest(target).name}
        current = {"source_ref": approved, "approved_sql_ref": approved,
                   "file_count": count, "entries": rows[:count], "inventory_sha": "c" * 64,
                   "plan_family": SQL.plan_family(approved),
                   "execution_manifest": SQL.execution_manifest(approved).name}
        refs = SQL.plan_path(approved)[1:]
        record = {"run_id": "580a6b83-822d-4137-87d5-22d3b8c590e7", "source_ref": SQL.BASE_REF,
                  "plan_sha": SQL.REF_PLAN_SHA[SQL.BASE_REF], "current_source_ref": approved,
                  "approved_sql_ref": approved, "current_plan_sha": SQL.REF_PLAN_SHA[approved],
                  "inventory_sha": "c" * 64,
                  "revisions": [{"revision": len(SQL.plan_path(ref)), "source_ref": ref,
                     "plan_sha": SQL.REF_PLAN_SHA[ref], "file_count": SQL.REF_COUNTS[ref]}
                     for ref in refs],
                  "installed": [{"position": n, "path": row["path"], "sha256": row["sha256"],
                     "installed_at": "2026-09-30T04:00:00+00:00"} for n, row in enumerate(rows[:count], 1)]}
        return plan, current, record

    def test_steps_fresh_h1_lists_all_known_prefixes_without_docker(self):
        plan, _, _ = self.planner_fixture(target=SQL.FIFTH_REF)
        with tempfile.TemporaryDirectory() as scratch:
            with patch.object(SQL, "approved_source_plan", return_value=plan):
                with patch.object(SQL.DockerDB, "query") as query:
                    self.assertEqual(SQL.etapas_requeridas(REPO, REPO, SQL.FIFTH_REF, Path(scratch)),
                                     list(SQL.plan_path(SQL.FIFTH_REF)))
            query.assert_not_called()

    def test_steps_38_allows_safe_39_and_complete_43_requires_none(self):
        for ref, target, expected in ((SQL.FOURTH_REF, SQL.FIFTH_REF, [SQL.FIFTH_REF]),
                                     (SQL.MAIN_REF, SQL.MAIN_REF, [])):
            plan, current, record = self.planner_fixture(ref, target)
            with tempfile.TemporaryDirectory() as scratch:
                state = Path(scratch); journal = state / "sql-journal.json"
                original = json.dumps(record).encode(); journal.write_bytes(original)
                with patch.object(SQL, "approved_source_plan", return_value=plan):
                    with patch.object(SQL, "validate_git_source", return_value=current):
                        self.assertEqual(SQL.etapas_requeridas(REPO, REPO, target, state), expected)
                self.assertEqual(journal.read_bytes(), original)

    def test_steps_refuses_foreign_malformed_incomplete_or_gapped_journal(self):
        for invalid in ("ref", "hash", "gap", "date", "uuid", "source"):
            plan, current, record = self.planner_fixture()
            if invalid == "ref": record["source_ref"] = "a" * 40
            elif invalid == "hash": record["plan_sha"] = "a" * 64
            elif invalid == "gap": record["revisions"][0]["revision"] = 4
            elif invalid == "date": record["installed"][0]["installed_at"] = "fecha inválida"
            elif invalid == "uuid": record["run_id"] = "otro clon"
            else: current["approved_sql_ref"] = SQL.THIRD_REF
            with self.subTest(invalid=invalid), tempfile.TemporaryDirectory() as scratch:
                state = Path(scratch); (state / "sql-journal.json").write_text(json.dumps(record))
                with patch.object(SQL, "approved_source_plan", return_value=plan):
                    with patch.object(SQL, "validate_git_source", return_value=current):
                        with self.assertRaises(SQL.Refused):
                            SQL.etapas_requeridas(REPO, REPO, SQL.MAIN_REF, state)

    def test_h6_revision_requires_complete_39_and_preserves_five_prefixes(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.H6_REF)
        _, _, record = self.planner_fixture(SQL.FIFTH_REF)
        original = {k: record[k] for k in ("run_id", "source_ref", "plan_sha")}
        revisions = record["revisions"]
        new = {"revision": 6, "source_ref": SQL.H6_REF, "plan_sha": SQL.plan_hash(rows),
               "file_count": 41, "acknowledged_at": "2026-09-30T06:00:00Z"}
        answers = ["t", json.dumps(revisions), json.dumps(record["installed"]), "",
                   "t", json.dumps([*revisions, new])]
        with patch.object(SQL.DockerDB, "query", side_effect=answers) as queries:
            result = SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.H6_REF, original)
        self.assertEqual(result["revisions"][:-1], revisions)
        self.assertEqual(result["revisions"][-1], new)
        for key, value in original.items():
            self.assertEqual(result[key], value)
        self.assertEqual(result["plan_family"], "h6_41")
        self.assertEqual(result["execution_manifest"], "sql_main_h6.txt")
        mutation = queries.call_args_list[3].args[0]
        self.assertIn("(revision=6 AND file_count IN (41,43))", mutation)
        self.assertNotIn("UPDATE", mutation)
        self.assertNotIn("DELETE", mutation)
        with patch.object(SQL.DockerDB, "query", side_effect=["t", json.dumps(revisions), "[]"]):
            with self.assertRaisesRegex(SQL.Refused, "39 SQL"):
                SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.H6_REF, original)

    def test_revision_44_exige_recibos_41_y_rechaza_ledger_43(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.H6_FIRMA_REF)
        _, _, record = self.planner_fixture(SQL.H6_REF, SQL.H6_FIRMA_REF)
        original = {k: record[k] for k in ("run_id", "source_ref", "plan_sha")}
        revisions = record["revisions"]
        new = {"revision": 7, "source_ref": SQL.H6_FIRMA_REF,
               "plan_sha": SQL.plan_hash(rows), "file_count": 44,
               "acknowledged_at": "2026-09-30T16:00:00Z"}
        answers = ["t", json.dumps(revisions), json.dumps(record["installed"]), "",
                   "t", json.dumps([*revisions, new])]
        with patch.object(SQL.DockerDB, "query", side_effect=answers) as queries:
            result = SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.H6_FIRMA_REF, original)
        self.assertEqual(result["revisions"][-1], new)
        mutation = queries.call_args_list[3].args[0]
        self.assertIn("(revision=7 AND file_count=44)", mutation)
        self.assertNotIn("UPDATE", mutation)
        self.assertNotIn("DELETE", mutation)
        with patch.object(SQL.DockerDB, "query", side_effect=["t", json.dumps(revisions), "[]"]):
            with self.assertRaisesRegex(SQL.Refused, "41 SQL"):
                SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.H6_FIRMA_REF, original)
        plan, current, record43 = self.planner_fixture(SQL.MAIN_REF, SQL.H6_FIRMA_REF)
        with tempfile.TemporaryDirectory() as scratch:
            (Path(scratch) / "sql-journal.json").write_text(json.dumps(record43))
            with patch.object(SQL, "PROPOSED_REFS", set()), \
                    patch.object(SQL, "approved_source_plan", return_value=plan), \
                    patch.object(SQL, "validate_git_source", return_value=current):
                with self.assertRaisesRegex(SQL.Refused, "familia"):
                    SQL.etapas_requeridas(REPO, REPO, SQL.H6_FIRMA_REF, Path(scratch))

    def test_steps_reanuda_revision_44_con_41_recibos_tras_caida(self):
        plan, current, record = self.planner_fixture(SQL.H6_FIRMA_REF, SQL.H6_FIRMA_REF)
        record["installed"] = record["installed"][:41]
        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            (state / "sql-journal.json").write_text(json.dumps(record))
            with patch.object(SQL, "PROPOSED_REFS", set()), \
                    patch.object(SQL, "approved_source_plan", return_value=plan), \
                    patch.object(SQL, "validate_git_source", return_value=current):
                self.assertEqual(SQL.etapas_requeridas(REPO, REPO, SQL.H6_FIRMA_REF, state),
                                 [SQL.H6_FIRMA_REF])

    def test_plan_44_propuesto_no_llega_a_postgresql(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.H6_FIRMA_REF)
        plan = SQL.validate_git_source(SQL.H6_FIRMA_REF, REPO)
        with patch.object(SQL.DockerDB, "check_owner") as check_owner:
            with self.assertRaisesRegex(SQL.Refused, "propuesto"):
                SQL.apply(SQL.DockerDB("vec-test"), rows, Path("/irrelevant"),
                          SQL.H6_FIRMA_REF, plan)
        check_owner.assert_not_called()
        with self.assertRaisesRegex(SQL.Refused, "propuesto"):
            SQL.require_installable(SQL.H6_FIRMA_REF)

    def test_ready_exige_recibos_vivos_y_acl_actual(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.H6_REF)
        plan = {"approved_sql_ref": SQL.H6_REF, "plan_sha": SQL.REF_PLAN_SHA[SQL.H6_REF],
                "inventory_sha": "inventario", "file_count": len(rows), "entries": rows}
        installed = [{"position": i, "path": row["path"], "sha256": row["sha256"]}
                     for i, row in enumerate(rows, 1)]
        record = {"source_ref": SQL.BASE_REF, "plan_sha": SQL.REF_PLAN_SHA[SQL.BASE_REF],
                  "approved_sql_ref": SQL.H6_REF, "current_source_ref": SQL.H6_REF,
                  "current_plan_sha": plan["plan_sha"], "inventory_sha": "inventario",
                  "installed": installed}
        actual = {"base_ref": record["source_ref"], "base_sha": record["plan_sha"],
                  "last_ref": SQL.H6_REF, "last_sha": plan["plan_sha"]}

        class Database:
            def __init__(self, acl):
                self.acl = acl
                self.queries = 0
            def check_owner(self):
                return None
            def query(self, _):
                self.queries += 1
                return json.dumps(actual) if self.queries == 1 else self.acl

        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            (state / "sql-journal.json").write_text(json.dumps(record))
            with patch.object(SQL, "approved_source_plan", return_value=plan), \
                    patch.object(SQL, "load_plan", return_value=rows), \
                    patch.object(SQL, "receipts", return_value=installed):
                SQL.verify_live(Database("t"), REPO, REPO, SQL.H6_REF, state)
                with self.assertRaisesRegex(SQL.Refused, "ACL"):
                    SQL.verify_live(Database("f"), REPO, REPO, SQL.H6_REF, state)
                record["installed"] = installed[:-1]
                (state / "sql-journal.json").write_text(json.dumps(record))
                with self.assertRaisesRegex(SQL.Refused, "divergen"):
                    SQL.verify_live(Database("t"), REPO, REPO, SQL.H6_REF, state)

    def test_h6_steps_fresh_38_39_and_complete_41(self):
        plan, _, _ = self.planner_fixture(target=SQL.H6_REF)
        with tempfile.TemporaryDirectory() as scratch:
            with patch.object(SQL, "approved_source_plan", return_value=plan):
                self.assertEqual(SQL.etapas_requeridas(REPO, REPO, SQL.H6_REF, Path(scratch)),
                                 [SQL.BASE_REF, SQL.PREVIOUS_REF, SQL.THIRD_REF, SQL.FOURTH_REF,
                                  SQL.FIFTH_REF, SQL.H6_REF])
        for ref, expected in ((SQL.FOURTH_REF, [SQL.FIFTH_REF, SQL.H6_REF]),
                              (SQL.FIFTH_REF, [SQL.H6_REF]), (SQL.H6_REF, [])):
            plan, current, record = self.planner_fixture(ref, SQL.H6_REF)
            with self.subTest(ref=ref), tempfile.TemporaryDirectory() as scratch:
                state = Path(scratch); journal = state / "sql-journal.json"
                before = json.dumps(record).encode(); journal.write_bytes(before)
                with patch.object(SQL, "approved_source_plan", return_value=plan):
                    with patch.object(SQL, "validate_git_source", return_value=current):
                        self.assertEqual(SQL.etapas_requeridas(REPO, REPO, SQL.H6_REF, state), expected)
                self.assertEqual(journal.read_bytes(), before)

    def test_sibling_43_cannot_become_h6_and_41_cannot_become_43(self):
        for recognized, target in ((SQL.MAIN_REF, SQL.H6_REF), (SQL.H6_REF, SQL.MAIN_REF)):
            plan, current, record = self.planner_fixture(recognized, target)
            with self.subTest(target=target), tempfile.TemporaryDirectory() as scratch:
                state = Path(scratch); journal = state / "sql-journal.json"
                before = json.dumps(record).encode(); journal.write_bytes(before)
                with patch.object(SQL, "approved_source_plan", return_value=plan):
                    with patch.object(SQL, "validate_git_source", return_value=current):
                        with self.assertRaisesRegex(SQL.Refused, "familia"):
                            SQL.etapas_requeridas(REPO, REPO, target, state)
                self.assertEqual(journal.read_bytes(), before)
                original = {k: record[k] for k in ("run_id", "source_ref", "plan_sha")}
                rows = SQL.load_plan(REPO, source_ref=target)
                with patch.object(SQL.DockerDB, "query", side_effect=["t", json.dumps(record["revisions"])]) as query:
                    with self.assertRaisesRegex(SQL.Refused, "incompatibles"):
                        SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, target, original)
                self.assertTrue(all(c.args[0].lstrip().startswith("SELECT") for c in query.call_args_list))
                # El plan original también puede ser directamente la familia hermana.
                with patch.object(SQL.DockerDB, "query") as query:
                    with self.assertRaisesRegex(SQL.Refused, "otro plan"):
                        SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, target,
                            {"source_ref": recognized, "plan_sha": SQL.REF_PLAN_SHA[recognized]})
                query.assert_not_called()

    def test_recovery_only_43_never_creates_new_plan_revision(self):
        rows = SQL.load_plan(REPO)
        _, _, record = self.planner_fixture(SQL.FIFTH_REF)
        original = {k: record[k] for k in ("run_id", "source_ref", "plan_sha")}
        with patch.object(SQL.DockerDB, "query", side_effect=["t", json.dumps(record["revisions"])]) as query:
            with self.assertRaisesRegex(SQL.Refused, "retirado"):
                SQL.acknowledge_plan(SQL.DockerDB("vec-test"), rows, SQL.MAIN_REF, original)
        self.assertTrue(all(c.args[0].lstrip().startswith("SELECT") for c in query.call_args_list))

    def test_h6_replay_journal_preserves_root_metadata_and_physical_41(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.H6_REF)
        plan, _, record = self.planner_fixture(SQL.H6_REF, SQL.H6_REF)
        alias = "a" * 40
        plan.update(source_ref=alias, plan_sha=SQL.plan_hash(rows),
                    verified_main_ref="b" * 40, inventory_sha="c" * 64)
        original = {k: v for k, v in record.items() if k != "installed"}
        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            with patch.object(SQL.DockerDB, "check_owner"), patch.object(SQL.DockerDB, "query") as query:
                with patch.object(SQL, "initialize", return_value=original.copy()) as initialize:
                    with patch.object(SQL, "receipts", return_value=record["installed"]):
                        SQL.apply(SQL.DockerDB("vec-test"), rows, state, alias, plan)
            query.assert_not_called()
            self.assertEqual(initialize.call_args.args[2], SQL.H6_REF)
            journal = json.loads((state / "sql-journal.json").read_text())
            for key in ("run_id", "source_ref", "plan_sha", "revisions"):
                self.assertEqual(journal[key], record[key])
            self.assertEqual(journal["current_source_ref"], alias)
            self.assertEqual(journal["approved_sql_ref"], SQL.H6_REF)
            self.assertEqual(journal["current_plan_sha"], SQL.REF_PLAN_SHA[SQL.H6_REF])
            self.assertEqual(journal["file_count"], 41)
            self.assertEqual(journal["plan_family"], "h6_41")
            self.assertEqual(journal["execution_manifest"], "sql_main_h6.txt")
            self.assertEqual(journal["installed"], record["installed"])

    def test_recovery_only_43_refuses_fresh_or_38_before_initialize_or_up(self):
        rows = SQL.load_plan(REPO)
        installed = [{"position": n, "path": row["path"], "sha256": row["sha256"]}
                     for n, row in enumerate(rows[:38], 1)]
        for replies in (["f"], ["t", json.dumps(installed)]):
            with self.subTest(replies=len(replies)), tempfile.TemporaryDirectory() as scratch:
                with patch.object(SQL.DockerDB, "check_owner"):
                    with patch.object(SQL.DockerDB, "query", side_effect=replies) as query:
                        with patch.object(SQL, "initialize") as initialize:
                            with self.assertRaises(SQL.Refused):
                                SQL.apply(SQL.DockerDB("vec-test"), rows, Path(scratch), SQL.MAIN_REF)
                initialize.assert_not_called()
                self.assertTrue(all(call.args[0].lstrip().startswith("SELECT")
                                    for call in query.call_args_list))

    def test_recovery_only_43_recovers_exact_existing_ledger_without_sql_writes(self):
        rows = SQL.load_plan(REPO)
        _, _, record = self.planner_fixture(SQL.MAIN_REF)
        metadata = {k: v for k, v in record.items() if k != "installed"}
        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            with patch.object(SQL.DockerDB, "check_owner"):
                with patch.object(SQL.DockerDB, "query", return_value="t") as query:
                    with patch.object(SQL, "initialize", return_value=metadata):
                        with patch.object(SQL, "receipts", return_value=record["installed"]):
                            SQL.apply(SQL.DockerDB("vec-test"), rows, state, SQL.MAIN_REF)
                            first = (state / "sql-journal.json").read_bytes()
                            (state / "sql-journal.json").unlink()
                            SQL.apply(SQL.DockerDB("vec-test"), rows, state, SQL.MAIN_REF)
            self.assertEqual(first, (state / "sql-journal.json").read_bytes())
            self.assertTrue(all(call.args[0].lstrip().startswith("SELECT")
                                for call in query.call_args_list))
            self.assertEqual(json.loads(first)["installed"], record["installed"])

    def test_steps_recovery_only_43_refuses_missing_journal_or_complete_38(self):
        plan, current, record = self.planner_fixture()
        with tempfile.TemporaryDirectory() as scratch:
            state = Path(scratch)
            with patch.object(SQL, "approved_source_plan", return_value=plan):
                with self.assertRaisesRegex(SQL.Refused, "retirado"):
                    SQL.etapas_requeridas(REPO, REPO, SQL.MAIN_REF, state)
                (state / "sql-journal.json").write_text(json.dumps(record))
                with patch.object(SQL, "validate_git_source", return_value=current):
                    with self.assertRaisesRegex(SQL.Refused, "retirado"):
                        SQL.etapas_requeridas(REPO, REPO, SQL.MAIN_REF, state)

    def test_recovery_only_43_rejects_wrong_hash_even_with_43_rows(self):
        rows = SQL.load_plan(REPO)
        _, _, record = self.planner_fixture(SQL.MAIN_REF)
        record["installed"][-1]["sha256"] = "a" * 64
        with tempfile.TemporaryDirectory() as scratch:
            with patch.object(SQL.DockerDB, "check_owner"):
                with patch.object(SQL.DockerDB, "query", side_effect=["t", json.dumps(record["installed"])]):
                    with patch.object(SQL, "initialize") as initialize:
                        with self.assertRaisesRegex(SQL.Refused, "incompatibles"):
                            SQL.apply(SQL.DockerDB("vec-test"), rows, Path(scratch), SQL.MAIN_REF)
        initialize.assert_not_called()

    def test_safe_39_is_not_blocked_by_recovery_only_43(self):
        rows = SQL.load_plan(REPO, source_ref=SQL.FIFTH_REF)
        _, _, record = self.planner_fixture(SQL.FIFTH_REF)
        metadata = {k: v for k, v in record.items() if k != "installed"}
        with tempfile.TemporaryDirectory() as scratch:
            with patch.object(SQL.DockerDB, "check_owner"):
                with patch.object(SQL.DockerDB, "query") as query:
                    with patch.object(SQL, "initialize", return_value=metadata) as initialize:
                        with patch.object(SQL, "receipts", return_value=record["installed"]):
                            SQL.apply(SQL.DockerDB("vec-test"), rows, Path(scratch), SQL.FIFTH_REF)
        initialize.assert_called_once()
        query.assert_not_called()


class GitSourceTests(unittest.TestCase):
    @contextlib.contextmanager
    def fixture(self, mutation=None, unrelated=False, h6=False):
        with tempfile.TemporaryDirectory() as scratch:
            control = Path(scratch) / "control"
            archive = Path(scratch) / "archive"
            control.mkdir(); archive.mkdir()
            env = {"PATH": "/usr/bin:/bin", "HOME": scratch, "LC_ALL": "C",
                   "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null"}
            def git(*args):
                return subprocess.check_output(["/usr/bin/git", "-C", str(control), *args],
                                               env=env, stderr=subprocess.DEVNULL).decode().strip()
            git("init", "-q")
            git("config", "user.name", "aavidad")
            git("config", "user.email", "avidad@dipgra.es")
            relative = "deploy/postgresql/demo/migraciones/000001.up.sql"
            data = b"BEGIN;\nSELECT 1;\nCOMMIT;\n"
            path = control / relative
            path.parent.mkdir(parents=True); path.write_bytes(data)
            retained = {p: SQL.sha(data) for p in SQL.H6_WITHHELD} if h6 else {}
            for retained_path in retained:
                extra = control / retained_path
                extra.parent.mkdir(parents=True, exist_ok=True); extra.write_bytes(data)
            git("add", "deploy"); git("commit", "-q", "-m", "base SQL")
            base = git("rev-parse", "HEAD")
            (control / "app.py").write_text("version = 2\n")
            if mutation:
                mutation(control, relative)
            git("add", "-A"); git("commit", "-q", "-m", "fuente posterior")
            descendant = git("rev-parse", "HEAD")
            if unrelated:
                tree = git("rev-parse", "HEAD^{tree}")
                descendant = git("commit-tree", tree, "-m", "historia sin relación")
            git("update-ref", "refs/remotes/origin/main", descendant)
            copied = archive / relative
            copied.parent.mkdir(parents=True); copied.write_bytes(data); copied.chmod(0o600)
            for retained_path in retained:
                extra = archive / retained_path
                extra.parent.mkdir(parents=True, exist_ok=True); extra.write_bytes(data); extra.chmod(0o600)
            row = {"phase": "MAIN", "path": relative, "sha256": SQL.sha(data), "sql": data.decode()}
            manifest = Path(scratch) / "plan.txt"
            manifest.write_text(f"MAIN {row['sha256']} {relative}\n")
            original_load = SQL.load_plan
            def load(repo, source_ref, contents):
                return original_load(repo, manifest=manifest, source_ref=source_ref, contents=contents)
            with contextlib.ExitStack() as stack:
                if not h6:
                    stack.enter_context(patch.object(SQL, "MAIN_REF", base))
                if h6:
                    stack.enter_context(patch.object(SQL, "H6_REF", base))
                    stack.enter_context(patch.object(SQL, "H6_MANIFEST", manifest))
                    stack.enter_context(patch.object(SQL, "H6_WITHHELD", retained))
                stack.enter_context(patch.object(SQL, "REF_COUNTS", {base: 1}))
                stack.enter_context(patch.object(SQL, "REF_PLAN_SHA", {base: SQL.plan_hash([row])}))
                stack.enter_context(patch.object(SQL, "load_plan", side_effect=load))
                yield control, archive, base, descendant, relative

    def test_h6_total_inventory_keeps_all_four_retained_and_rejects_any_change(self):
        for relative in SQL.H6_WITHHELD:
            for change in ("bytes", "deleted", "mode"):
                def mutate(control, _, path=relative, change=change):
                    file = control / path
                    if change == "bytes": file.write_text("BEGIN;\nSELECT 9;\nCOMMIT;\n")
                    elif change == "deleted": file.unlink()
                    else: file.chmod(0o755)
                with self.subTest(path=relative, change=change), self.fixture(mutate, h6=True) as fixture:
                    control, archive, _, descendant, _ = fixture
                    with self.assertRaisesRegex(SQL.Refused, "SQL distinto"):
                        SQL.approved_source_plan(archive, descendant, control)
        def unknown(control, _):
            extra = control / "deploy/postgresql/catalogos_configurables/migraciones/000099_unknown.up.sql"
            extra.write_text("BEGIN;\nSELECT 2;\nCOMMIT;\n")
        with self.fixture(unknown, h6=True) as (control, archive, _, descendant, _):
            with self.assertRaisesRegex(SQL.Refused, "SQL distinto"):
                SQL.approved_source_plan(archive, descendant, control)

    def test_h6_identical_code_alias_keeps_family_and_execution_manifest(self):
        with self.fixture(h6=True) as (control, archive, base, descendant, _):
            plan = SQL.approved_source_plan(archive, descendant, control)
            self.assertEqual(plan["source_ref"], descendant)
            self.assertEqual(plan["approved_sql_ref"], base)
            self.assertEqual(plan["plan_family"], "h6_41")
            self.assertEqual(plan["execution_manifest"], "plan.txt")
            self.assertEqual(len(plan["sql_inventory"]), 5)
            self.assertEqual(plan["file_count"], 1)

    def test_identical_descendant_and_private_archive_are_accepted(self):
        with self.fixture() as (control, archive, base, descendant, _):
            plan = SQL.approved_source_plan(archive, descendant, control)
            self.assertEqual(plan["approved_sql_ref"], base)
            self.assertEqual(plan["source_ref"], descendant)
            self.assertEqual(plan["verified_main_ref"], descendant)
            self.assertEqual(plan["file_count"], 1)

    def test_unrelated_history_is_refused_even_with_same_sql(self):
        with self.fixture(unrelated=True) as (control, archive, _, descendant, _):
            with self.assertRaisesRegex(SQL.Refused, "historia aprobada"):
                SQL.approved_source_plan(archive, descendant, control)

    def test_added_modified_deleted_or_external_up_needs_review(self):
        def added(control, _):
            (control / "deploy/postgresql/demo/extra.up.sql").write_text("SELECT 2;\n")
        def added_test(control, _):
            (control / "deploy/postgresql/demo/prueba.SQL").write_text("SELECT 2;\n")
        def modified(control, path):
            (control / path).write_text("BEGIN;\nSELECT 2;\nCOMMIT;\n")
        def deleted(control, path):
            (control / path).unlink()
        def external(control, _):
            (control / "extra.UP.SQL").write_text("SELECT 3;\n")
        def external_role(control, _):
            (control / "roles_up.sql").write_text("SELECT 3;\n")
        for change in (added, added_test, modified, deleted, external, external_role):
            with self.subTest(change=change.__name__), self.fixture(change) as fixture:
                control, archive, _, descendant, _ = fixture
                with self.assertRaises(SQL.Refused):
                    SQL.approved_source_plan(archive, descendant, control)

    def test_archive_bytes_modes_links_and_extra_sql_are_refused(self):
        for change in ("bytes", "mode", "link", "extra", "external", "external_role"):
            with self.subTest(change=change), self.fixture() as fixture:
                control, archive, _, descendant, relative = fixture
                path = archive / relative
                if change == "bytes": path.write_text("SELECT 9;\n")
                elif change == "mode": path.chmod(0o700)
                elif change == "link": path.unlink(); path.symlink_to(control / relative)
                elif change == "extra": (path.parent / "extra.sql").write_text("SELECT 1;\n")
                elif change == "external": (archive / "extra.up.sql").write_text("SELECT 1;\n")
                else: (archive / "roles_up.sql").write_text("SELECT 1;\n")
                with self.assertRaises(SQL.Refused):
                    SQL.approved_source_plan(archive, descendant, control)

    def test_git_environment_redirection_is_not_inherited(self):
        with self.fixture() as (control, archive, base, descendant, _):
            with patch.dict(os.environ, {"GIT_DIR": "/unknown/git", "GIT_WORK_TREE": "/unknown/tree",
                                          "GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "core.worktree",
                                          "GIT_CONFIG_VALUE_0": "/unknown/worktree"}):
                self.assertEqual(SQL.approved_source_plan(archive, descendant, control)["approved_sql_ref"], base)

    def test_grafts_are_refused_and_replace_cannot_forge_lineage(self):
        with self.fixture(unrelated=True) as (control, archive, _, descendant, _):
            real = subprocess.check_output(["/usr/bin/git", "-C", str(control), "rev-parse", "HEAD"]).strip()
            subprocess.run(["/usr/bin/git", "-C", str(control), "update-ref",
                            "refs/replace/" + descendant, real.decode()], check=True)
            with self.assertRaisesRegex(SQL.Refused, "historia aprobada"):
                SQL.approved_source_plan(archive, descendant, control)
        with self.fixture() as (control, archive, base, descendant, _):
            (control / ".git/info/grafts").write_text(base + "\n")
            with self.assertRaisesRegex(SQL.Refused, "grafts"):
                SQL.approved_source_plan(archive, descendant, control)

    def test_includes_must_be_static_and_inside_inventory(self):
        for data in (b"\\i /outside.sql\n", b"\\ir ../../../outside.sql\n", b"\\ir :dynamic\n"):
            with self.assertRaisesRegex(SQL.Refused, "include SQL"):
                SQL.validate_includes({"deploy/postgresql/demo/source.sql": data})

    def test_old_descendant_remains_valid_after_a_later_sql_plan_is_approved(self):
        with self.fixture() as (control, archive, base, descendant, _):
            # El origen actual tiene otra aprobación posterior; la fuente
            # solicitada debe recuperar la aprobación ancestra correcta.
            newer = subprocess.check_output(["/usr/bin/git", "-C", str(control),
                       "commit-tree", "HEAD^{tree}", "-p", descendant, "-m", "aprobación posterior"]).decode().strip()
            subprocess.run(["/usr/bin/git", "-C", str(control), "update-ref",
                            "refs/remotes/origin/main", newer], check=True)
            with patch.object(SQL, "MAIN_REF", newer), patch.object(SQL, "REF_COUNTS", {base: 1, newer: 2}):
                plan = SQL.approved_source_plan(archive, descendant, control)
            self.assertEqual(plan["approved_sql_ref"], base)
            self.assertEqual(plan["source_ref"], descendant)


if __name__ == "__main__":
    unittest.main()
