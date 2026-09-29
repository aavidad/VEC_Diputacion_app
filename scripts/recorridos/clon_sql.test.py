"""Pruebas focales de integridad, recuperación y propiedad del clon."""

import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("clon_sql", Path(__file__).with_name("clon_sql.py"))
SQL = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SQL)
REPO = Path(__file__).resolve().parents[2]


class HelperTests(unittest.TestCase):
    def test_real_plan_pins_all_33_existing_files(self):
        rows = SQL.load_plan(REPO)
        self.assertEqual(len(rows), 33)
        self.assertEqual([len([r for r in rows if r["phase"] == phase])
                          for phase in ("H3", "H4", "MAIN")], [8, 9, 16])
        roles = [r["path"] for r in rows if "/roles" in r["path"]]
        self.assertEqual(len(roles), 4)

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
        with self.assertRaisesRegex(SQL.Refused, "hash main"):
            SQL.main(["--repo", str(REPO), "--source-ref", "b" * 40, "--plan"])

    def test_database_error_does_not_disclose_detail_or_rows(self):
        result = subprocess.CompletedProcess([], 1, "", "ERROR:  42501: denegado\n"
                                             "DETAIL: dato privado de una fila\n")
        with patch.object(SQL.subprocess, "run", return_value=result):
            with self.assertRaises(SQL.Refused) as error:
                SQL.DockerDB("vec-codexm-test").query("SELECT 1;")
        self.assertIn("42501", str(error.exception))
        self.assertNotIn("privado", str(error.exception))

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


if __name__ == "__main__":
    unittest.main()
