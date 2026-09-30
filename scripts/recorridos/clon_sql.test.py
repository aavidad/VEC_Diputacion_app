"""Pruebas focales de integridad, recuperación y propiedad del clon."""

import importlib.util
import contextlib
import json
import io
import tarfile
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
GIT_REPO = Path(__file__).resolve().parents[2]


class HelperTests(unittest.TestCase):
    def test_plan_45_preserva_44_y_coteja_ad132_fuera_del_lote(self):
        previous = SQL.load_plan(REPO, source_ref=SQL.H6_FIRMA_REF)
        rows = SQL.load_plan(REPO, source_ref=SQL.H6_FIRMA_FINAL_REF)
        self.assertEqual(rows[:44], previous)
        self.assertEqual({k: v for k, v in rows[44].items() if k != "sql"}, SQL.H6_CT153)
        self.assertEqual(SQL.plan_hash(rows),
                         "52240f99125ce4b83872bc41c75bfda233e7957b0b2978aa361c26fba708ac48")
        self.assertEqual(len(SQL.withheld_sql(SQL.H6_FIRMA_FINAL_REF)), 19 + 4)
        plan = SQL.validate_git_source(SQL.H6_FIRMA_FINAL_REF, GIT_REPO)
        self.assertEqual((plan["approved_sql_ref"], plan["file_count"], plan["plan_family"],
                          plan["status"], plan["execution_manifest"]),
                         (SQL.H6_FIRMA_FINAL_REF, 45, "h6_45", "proposed", "sql_main_h6_firma.txt"))
        self.assertEqual(plan["dba_excluded"],
                         [{"path": p, "sha256": h} for p, h in SQL.H6_DBA_EXCLUDED.items()])
        source = SQL.GitSource(GIT_REPO)
        before, _ = source.inventory(SQL.H6_FIRMA_REF)
        after, _ = source.inventory(SQL.H6_FIRMA_FINAL_REF)
        up_paths = lambda entries: {r["path"] for r in entries
                                    if r["path"].endswith((".up.sql", "_up.sql"))}
        self.assertEqual(up_paths(after) - up_paths(before),
                         {SQL.H6_CT153["path"]} | SQL.H6_DBA_EXCLUDED.keys())
        self.assertTrue((SQL.H6_FIRMA_WITHHELD.keys() | SQL.H6_DBA_EXCLUDED.keys())
                        .isdisjoint({r["path"] for r in rows}))
        self.assertEqual(SQL.plan_path(SQL.H6_FIRMA_FINAL_REF)[-3:],
                         (SQL.H6_REF, SQL.H6_FIRMA_REF, SQL.H6_FIRMA_FINAL_REF))
        self.assertEqual(len(SQL.plan_path(SQL.H6_FIRMA_FINAL_REF)), 8)

    def test_plan_45_rechaza_ad132_en_manifiesto_y_recibos(self):
        path, digest = next(iter(SQL.H6_DBA_EXCLUDED.items()))
        line = f"MAIN {digest} {path}\n"
        original = SQL.H6_FIRMA_MANIFEST.read_text()
        with tempfile.TemporaryDirectory() as scratch:
            manifest = Path(scratch) / "plan.txt"
            for content in (original + line, original.rsplit("MAIN\t", 1)[0] + line):
                manifest.write_text(content)
                with self.subTest(content=content[-160:]), self.assertRaisesRegex(SQL.Refused, "CLI DBA"):
                    SQL.load_plan(REPO, manifest, source_ref=SQL.H6_FIRMA_FINAL_REF)
        plan = SQL.validate_git_source(SQL.H6_FIRMA_FINAL_REF, GIT_REPO)
        installed = [{"position": i, "path": r["path"], "sha256": r["sha256"]}
                     for i, r in enumerate(plan["entries"], 1)]
        SQL.validate_receipts(installed, plan)
        receipt = {"position": 45, "path": path, "sha256": digest}
        with self.assertRaisesRegex(SQL.Refused, "incompatibles"):
            SQL.validate_receipts([*installed[:44], receipt], plan)
        with self.assertRaisesRegex(SQL.Refused, "ajenos"):
            SQL.validate_receipts([*installed, {**receipt, "position": 46}], plan)

    def test_plan_45_rechaza_huellas_de_plan_ct153_y_ad132_inesperadas(self):
        _, contents = SQL.GitSource(GIT_REPO).inventory(SQL.H6_FIRMA_FINAL_REF)
        with tempfile.TemporaryDirectory() as scratch:
            manifest = Path(scratch) / "plan.txt"
            manifest.write_text(SQL.H6_FIRMA_MANIFEST.read_text().replace(
                "MAIN\t" + SQL.H6_CT153["sha256"], "H4\t" + SQL.H6_CT153["sha256"]))
            with self.assertRaisesRegex(SQL.Refused, "manifiesto incompatible"):
                SQL.load_plan(None, manifest, SQL.H6_FIRMA_FINAL_REF, contents)
        for path, message in ((SQL.H6_CT153["path"], "SHA incompatible"),
                              (next(iter(SQL.H6_DBA_EXCLUDED)), "DBA excluida distinta")):
            with self.subTest(path=path), self.assertRaisesRegex(SQL.Refused, message):
                SQL.load_plan(None, source_ref=SQL.H6_FIRMA_FINAL_REF,
                              contents={**contents, path: b"bytes no revisados"})

    def test_revision_44_solo_tolera_la_cola_ct153_exacta(self):
        with tempfile.TemporaryDirectory() as scratch:
            manifest = Path(scratch) / "plan.txt"
            original = SQL.H6_FIRMA_MANIFEST.read_text()
            for content in (original + original.splitlines()[-1] + "\n",
                            original.replace(SQL.H6_CT153["sha256"], "a" * 64)):
                manifest.write_text(content)
                with self.assertRaisesRegex(SQL.Refused, "44 entradas"):
                    SQL.load_plan(REPO, manifest, source_ref=SQL.H6_FIRMA_REF)

    def test_preflight_45_rechaza_inventarios_y_archive_divergentes(self):
        inventory, contents = SQL.GitSource(GIT_REPO).inventory(SQL.H6_FIRMA_FINAL_REF)
        actual = [dict(r) for r in inventory]
        actual[-1]["sha256"] = "a" * 64
        with patch.object(SQL.GitSource, "inventory", side_effect=[(inventory, contents), (actual, contents)]):
            with self.assertRaisesRegex(SQL.Refused, "SQL distinto"):
                SQL.validate_git_source(SQL.H6_FIRMA_FINAL_REF, GIT_REPO)
        with tempfile.TemporaryDirectory() as scratch:
            root = Path(scratch)
            for path, data in contents.items():
                destination = root / path
                destination.parent.mkdir(parents=True, exist_ok=True)
                destination.write_bytes(data)
            SQL.approved_source_plan(root, SQL.H6_FIRMA_FINAL_REF, GIT_REPO)
            excluded = root / next(iter(SQL.H6_DBA_EXCLUDED))
            excluded.write_bytes(excluded.read_bytes() + b"-- archivo distinto\n")
            with self.assertRaisesRegex(SQL.Refused, "extraído|archive|inventario"):
                SQL.approved_source_plan(root, SQL.H6_FIRMA_FINAL_REF, GIT_REPO)


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
        source = SQL.GitSource(GIT_REPO)
        inventory_before, _ = source.inventory(SQL.H6_REF)
        inventory_after, _ = source.inventory(SQL.H6_FIRMA_REF)
        up_paths = lambda entries: {item["path"] for item in entries
                                    if item["path"].endswith((".up.sql", "_up.sql"))}
        self.assertEqual(up_paths(inventory_after) - up_paths(inventory_before),
                         {r["path"] for r in rows[41:]} |
                         (SQL.H6_FIRMA_WITHHELD.keys() - SQL.H6_WITHHELD.keys()))
        self.assertEqual(SQL.plan_path(SQL.H6_FIRMA_REF)[-2:], (SQL.H6_REF, SQL.H6_FIRMA_REF))
        plan = SQL.validate_git_source(SQL.H6_FIRMA_REF, GIT_REPO)
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


class ExternalJournalTests(unittest.TestCase):
    @contextlib.contextmanager
    def fixture(self, fail=None, postcheck=True):
        with tempfile.TemporaryDirectory() as temporary:
            state = Path(temporary)
            data = "BEGIN;\nSELECT 1;\nCOMMIT;\n"
            row = {"phase": "MAIN", "path": "deploy/postgresql/demo/000001.up.sql",
                   "sha256": SQL.sha(data.encode()), "sql": data}
            ref = "1" * 40
            context = {key: str(n) * 64 for n, key in enumerate(SQL.CONTEXT_KEYS, 1)}
            plan = {"source_ref": ref, "approved_sql_ref": ref, "plan_sha": SQL.plan_hash([row]),
                    "inventory_sha": "a" * 64, "file_count": 1, "plan_family": "fixture",
                    "execution_manifest": "fixture.txt", "entries": [{k: v for k, v in row.items() if k != "sql"}]}
            class DB:
                calls = []
                owner_checked = False
                def check_owner(self):
                    self.owner_checked = True
                def query(self, text):
                    self.calls.append(text)
                    raw = json.loads((state / "sql-journal.json").read_text())
                    if raw["pending"] is None or raw["installed"]:
                        raise AssertionError("SQL antes de pending durable")
                    if fail:
                        raise fail
                    return ""
            class Kit:
                def validate(self, actual, ctx): return actual == plan and ctx == context
                def identity(self, db, ctx): return context["identidad_clon"]
                def confirm(self, db, entry, position, ctx): return position == 0 or postcheck
                def verify(self, db, record, ctx): return True
            db = DB(); db.calls = []
            with patch.object(SQL, "REF_COUNTS", {ref: 1}), \
                    patch.object(SQL, "REF_PLAN_SHA", {ref: SQL.plan_hash([row])}):
                yield state, db, [row], ref, plan, context, Kit()

    def run_apply(self, fixture, kit=True):
        state, db, rows, ref, plan, context, provider = fixture
        return SQL.apply(db, rows, state, ref, plan, context, provider if kit else None)

    def test_only_original_sql_and_external_confirmation_after_commit(self):
        with self.fixture() as f:
            state, db, rows, _, _, _, _ = f
            record = self.run_apply(f)
            self.assertEqual(db.calls, [rows[0]["sql"]])
            self.assertNotIn("vec_recorridos_clon", Path(SQL.__file__).read_text())
            self.assertEqual(record["phase"], "awaiting_ad132")
            self.assertIsNone(record["pending"])
            self.assertEqual(record["installed"][0]["confirmation"], "commit_returned_and_postcheck")
            with SQL.Journal(state) as journal:
                self.assertEqual(journal.load()["installed"], record["installed"])
            self.run_apply(f)
            self.assertEqual(len(db.calls), 1)
            self.assertFalse((state / "READY.json").exists())

    def test_crash_before_or_after_commit_stops_forever_without_confirmation(self):
        for failure in (KeyboardInterrupt(), SQL.Refused("antes COMMIT"),
                        subprocess.TimeoutExpired("dummy", 1), RuntimeError("después COMMIT")):
            with self.subTest(failure=type(failure).__name__), self.fixture(fail=failure) as f:
                state, db, _, _, _, _, _ = f
                with self.assertRaises((SQL.Refused, KeyboardInterrupt)):
                    self.run_apply(f)
                raw = (state / "sql-journal.json").read_bytes()
                record = json.loads(raw)
                self.assertEqual(record["pending"]["position"], 1)
                self.assertEqual(record["installed"], [])
                with self.assertRaisesRegex(SQL.Refused, "reconstruir"):
                    self.run_apply(f)
                self.assertEqual(len(db.calls), 1)
                self.assertEqual((state / "sql-journal.json").read_bytes(), raw)

    def test_commit_return_then_postcheck_failure_keeps_pending(self):
        with self.fixture(postcheck=False) as f:
            with self.assertRaisesRegex(SQL.Refused, "incierta"):
                self.run_apply(f)
            state = f[0]
            record = json.loads((state / "sql-journal.json").read_text())
            self.assertEqual(record["installed"], [])
            self.assertIsNotNone(record["pending"])

    def test_crash_during_confirmation_write_keeps_durable_pending(self):
        with self.fixture() as f:
            store = SQL.Journal.store
            def crash(journal, record):
                if record["installed"]:
                    raise KeyboardInterrupt()
                return store(journal, record)
            with patch.object(SQL.Journal, "store", crash), self.assertRaises(KeyboardInterrupt):
                self.run_apply(f)
            with self.assertRaisesRegex(SQL.Refused, "pendiente"):
                self.run_apply(f)
            self.assertEqual(len(f[1].calls), 1)

    def test_pending_fsync_before_sql_and_confirmation_fsync_after_sql(self):
        with self.fixture() as f:
            events = []
            real_fsync = os.fsync
            original = f[1].query
            def query(text):
                events.append("sql")
                return original(text)
            def fsync(fd):
                events.append("fsync")
                return real_fsync(fd)
            with patch.object(f[1], "query", query), patch.object(SQL.os, "fsync", fsync):
                self.run_apply(f)
            self.assertEqual(events, ["fsync", "fsync", "fsync", "fsync", "sql", "fsync", "fsync"])

    def test_kit_missing_incomplete_or_wrong_denies_before_sql(self):
        for provider in (None, object()):
            with self.fixture() as f:
                with self.assertRaisesRegex(SQL.Refused, "kit D"):
                    SQL.apply(f[1], f[2], f[0], f[3], f[4], f[5], provider)
                self.assertEqual(f[1].calls, [])
                self.assertFalse(f[1].owner_checked)
                self.assertFalse((f[0] / "sql-journal.json").exists())
        with self.fixture() as f:
            with patch.object(f[6], "validate", return_value=False), self.assertRaises(SQL.Refused):
                self.run_apply(f)
            self.assertEqual(f[1].calls, [])

    def test_proposed_45_denies_before_kit_or_state(self):
        plan = {"source_ref": SQL.H6_FIRMA_FINAL_REF,
                "approved_sql_ref": SQL.H6_FIRMA_FINAL_REF, "plan_sha": SQL.plan_hash([])}
        with self.assertRaisesRegex(SQL.Refused, "propuesto"):
            SQL.apply(None, [], Path("/absent"), SQL.H6_FIRMA_FINAL_REF, plan)

    def test_journal_old_corrupt_symlink_hardlink_and_public_file_preserved(self):
        for kind in ("old", "corrupt", "symlink", "hardlink", "public", "foreign"):
            with self.subTest(kind=kind), self.fixture() as f:
                state = f[0]; journal = state / "sql-journal.json"
                if kind in ("public", "foreign"):
                    self.run_apply(f)
                    record = json.loads(journal.read_text())
                    if kind == "foreign":
                        record["identidad_clon"] = "f" * 64
                        record["journal_sha"] = SQL.record_hash(record)
                        journal.write_text(json.dumps(record))
                    else: journal.chmod(0o644)
                else:
                    content = '{"version":1}' if kind == "old" else '{bad json'
                    if kind == "symlink":
                        other = state / "evidence"; other.write_text(content); other.chmod(0o600)
                        journal.symlink_to(other)
                    elif kind == "hardlink":
                        other = state / "evidence"; other.write_text(content); other.chmod(0o600)
                        os.link(other, journal)
                    else: journal.write_text(content); journal.chmod(0o600)
                before = journal.read_bytes(); count = len(f[1].calls)
                with self.assertRaises(SQL.Refused): self.run_apply(f)
                self.assertEqual(journal.read_bytes(), before)
                self.assertEqual(len(f[1].calls), count)

    def test_mutated_sql_bytes_never_execute_even_with_claimed_hash(self):
        with self.fixture() as f:
            f[2][0]["sql"] += "-- bytes ajenos\n"
            with self.assertRaisesRegex(SQL.Refused, "bytes SQL"):
                self.run_apply(f)
            self.assertEqual(f[1].calls, [])
            self.assertFalse((f[0] / "sql-journal.json").exists())

    def test_plan_divergence_and_corruption_do_not_reapply(self):
        with self.fixture() as f:
            self.run_apply(f)
            p = f[0] / "sql-journal.json"
            for recompute in (False, True):
                record = json.loads(p.read_text())
                record["installed"][0]["sha256"] = "f" * 64
                if recompute: record["journal_sha"] = SQL.record_hash(record)
                p.write_text(json.dumps(record))
                with self.assertRaises(SQL.Refused): self.run_apply(f)
                self.assertEqual(len(f[1].calls), 1)

    def test_directory_and_lock_links_are_rejected(self):
        with self.fixture() as f:
            linked = f[0] / "linked"; linked.symlink_to(f[0], target_is_directory=True)
            with self.assertRaises(SQL.Refused):
                with SQL.Journal(linked): pass
            other = f[0] / "evidence"; other.write_text("private"); other.chmod(0o600)
            (f[0] / "sql.lock").symlink_to(other)
            with self.assertRaises((SQL.Refused, OSError)):
                self.run_apply(f)
            self.assertEqual(other.read_text(), "private")

    def test_verify_live_denies_missing_kit_pending_and_ad132_then_checks_ro(self):
        with self.fixture() as f:
            state, db, _, ref, plan, context, kit = f
            self.run_apply(f)
            with patch.object(SQL, "approved_source_plan", return_value=plan):
                with self.assertRaisesRegex(SQL.Refused, "kit D"):
                    SQL.verify_live(db, REPO, REPO, ref, state)
                with self.assertRaisesRegex(SQL.Refused, "AD132"):
                    SQL.verify_live(db, REPO, REPO, ref, state, context, kit)
                with SQL.Journal(state) as journal:
                    record = journal.load(); record["phase"] = "ad132_confirmed"; journal.store(record)
                self.assertEqual(SQL.verify_live(db, REPO, REPO, ref, state, context, kit)["phase"],
                                 "ad132_confirmed")
                with patch.object(kit, "verify", return_value=False), self.assertRaises(SQL.Refused):
                    SQL.verify_live(db, REPO, REPO, ref, state, context, kit)
                with SQL.Journal(state) as journal:
                    record = journal.load(); record["pending"] = {"kind": "ad132"}; journal.store(record)
                with self.assertRaisesRegex(SQL.Refused, "pendiente"):
                    SQL.verify_live(db, REPO, REPO, ref, state, context, kit)

    def test_external_extension_preserves_run_and_history_and_sends_only_tail(self):
        with self.fixture() as f:
            state, db, rows, ref, plan, context, kit = f
            original = self.run_apply(f)
            run_id = original["run_id"]
            installed = list(original["installed"])
            root_sha = original["original_plan_sha"]
            extra_sql = "BEGIN;\nSELECT 2;\nCOMMIT;\n"
            rows.append({"phase": "MAIN", "path": "deploy/postgresql/demo/000002.up.sql",
                         "sha256": SQL.sha(extra_sql.encode()), "sql": extra_sql})
            target = "2" * 40
            SQL.REF_COUNTS[target] = 2
            SQL.REF_PLAN_SHA[target] = SQL.plan_hash(rows)
            plan.update(source_ref=target, approved_sql_ref=target, file_count=2,
                        plan_sha=SQL.plan_hash(rows), inventory_sha="b" * 64,
                        entries=[{k: v for k, v in r.items() if k != "sql"} for r in rows])
            def query(text):
                raw = json.loads((state / "sql-journal.json").read_text())
                self.assertEqual(raw["installed"], installed)
                self.assertEqual(raw["pending"]["position"], 2)
                db.calls.append(text)
            with patch.dict(SQL.REF_PARENT, {target: ref}), patch.object(db, "query", query):
                record = SQL.apply(db, rows, state, target, plan, context, kit)
            self.assertEqual(db.calls, [rows[0]["sql"], extra_sql])
            self.assertEqual(record["run_id"], run_id)
            self.assertEqual(record["installed"][:1], installed)
            self.assertEqual(record["original_plan_sha"], root_sha)
            self.assertEqual(record["revisions"][0]["previous_ref"], ref)

    def test_steps_reject_old_or_pending_without_git_or_sql_effects(self):
        with self.fixture() as f:
            state, _, _, ref, plan, _, _ = f
            with patch.object(SQL, "approved_source_plan", return_value=plan), \
                    patch.dict(SQL.REF_PARENT, {ref: None}):
                self.assertEqual(SQL.etapas_requeridas(REPO, REPO, ref, state), [ref])
                self.run_apply(f)
                with patch.object(SQL, "validate_git_source", return_value=plan):
                    self.assertEqual(SQL.etapas_requeridas(REPO, REPO, ref, state), [])
                with SQL.Journal(state) as journal:
                    record = journal.load(); record["pending"] = {"position": 2}; journal.store(record)
                with patch.object(SQL, "validate_git_source") as lookup, self.assertRaises(SQL.Refused):
                    SQL.etapas_requeridas(REPO, REPO, ref, state)
                lookup.assert_not_called()

    def test_readonly_callbacks_have_no_write_connection(self):
        class DB:
            calls = []
            def query(self, text): self.calls.append(text); return "t"
        db = DB(); ro = SQL.ReadOnlyDB(db)
        self.assertEqual(ro.query("SELECT true;"), "t")
        self.assertEqual(db.calls, ["BEGIN READ ONLY;\nSELECT true;\nCOMMIT;"])
        with self.assertRaises(SQL.Refused): ro.query("INSERT INTO dummy VALUES (1);")


class StrictBoundaryTests(unittest.TestCase):
    def test_readonly_rejects_multiple_statements_literals_escapes_and_meta_before_db(self):
        denied = (
            "SELECT 1; COMMIT; CREATE SCHEMA escaped;",
            "SELECT 1; COMMIT; CREATE TABLE escaped(id integer);",
            "SELECT 1;;", "SELECT 1; SELECT 2;",
            "SELECT '; COMMIT; CREATE SCHEMA escaped;'",
            "SELECT E'\\x3b';", "SELECT U&'\\003b';",
            "SELECT 1\n\\gexec", "SELECT 1\n\\! harmless",
            "SELECT 1 -- comment", "SELECT 1 /* comment */",
            "SELECT $$; COMMIT; CREATE SCHEMA escaped;$$;",
            "SELECT 1\x00", "SELECT ", None, "COMMIT;",
        )
        for text in denied:
            with self.subTest(text=text), patch.object(SQL.DockerDB, "query") as query:
                ro = SQL.ReadOnlyDB(SQL.DockerDB("vec-fixture"))
                with self.assertRaisesRegex(SQL.Refused, "SELECT estricta"):
                    ro.query(text)
                query.assert_not_called()

    def test_readonly_accepts_only_single_select_with_optional_final_terminator(self):
        for text in ("SELECT 1", " SELECT 1; ", "select\n1;", "SELECT 'anchor' AS name;"):
            with self.subTest(text=text), patch.object(SQL.DockerDB, "query", return_value="t") as query:
                self.assertEqual(SQL.ReadOnlyDB(SQL.DockerDB("vec-fixture")).query(text), "t")
                sent = query.call_args.args[0]
                self.assertTrue(sent.startswith("BEGIN READ ONLY;\n"))
                self.assertTrue(sent.endswith(";\nCOMMIT;"))
                self.assertEqual(sent.count(";"), 3)

    def test_cli_rejects_original_final_or_ancestor_symlink_for_all_modes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            actual = root / "actual"; actual.mkdir(mode=0o700)
            linked = root / "linked"; linked.symlink_to(actual, target_is_directory=True)
            inputs = (linked, linked / "new-state")
            for mode in ([], ["--verify-live"], ["--plan"], ["--steps"], ["--installable"]):
                for option in ("--repo", "--git-repo", "--state-dir"):
                    for path in inputs:
                        with self.subTest(mode=mode, option=option, path=path), \
                                patch.object(SQL, "approved_source_plan") as source, \
                                patch.object(SQL, "validate_git_source") as git, \
                                patch.object(SQL, "apply") as apply, \
                                patch.object(SQL, "verify_live") as verify, \
                                patch.object(SQL.DockerDB, "check_owner") as docker:
                            args = ["--repo", str(actual), "--container", "vec-fixture"]
                            if option == "--repo": args[1] = str(path)
                            else: args.extend([option, str(path)])
                            args.extend(mode)
                            with self.assertRaisesRegex(SQL.Refused, "enlazado"):
                                SQL.main(args)
                            for callback in (source, git, apply, verify, docker):
                                callback.assert_not_called()
            self.assertFalse((actual / "sql-journal.json").exists())
            self.assertFalse((actual / "new-state").exists())

    def test_cli_detects_symlink_before_dotdot_resolution(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            actual = root / "actual"; actual.mkdir()
            linked = root / "linked"; linked.symlink_to(actual, target_is_directory=True)
            original = linked / ".." / "next"
            with patch.object(SQL.Path, "resolve") as resolve:
                with self.assertRaisesRegex(SQL.Refused, "enlazado"):
                    SQL.main(["--repo", str(root), "--state-dir", str(original), "--plan"])
                resolve.assert_not_called()

    def test_original_regular_or_missing_path_is_allowed_without_resolving(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for original in (root, root / "new" / "state"):
                with patch.object(SQL.Path, "resolve") as resolve:
                    self.assertEqual(SQL.validate_original_path(original), original)
                    resolve.assert_not_called()


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



class H6Package62Tests(unittest.TestCase):
    @contextlib.contextmanager
    def fixture(self, mutation=None):
        with tempfile.TemporaryDirectory() as scratch:
            root = Path(scratch)
            _, git_contents = SQL.GitSource(GIT_REPO).inventory(SQL.H6_FIRMA_FINAL_REF)
            prefix = []
            for line in SQL.H6_PREFIX_MANIFEST.read_text().splitlines():
                if line.startswith("#") or not line: continue
                phase, digest, path = line.split()
                prefix.append(SQL.h6_sql_row(phase, path, digest, git_contents[path]))
            contents = {r["path"]: r["sql"].encode() for r in prefix}
            files, functional = {}, []
            for index in range(45):
                path = f"deploy/postgresql/demo/{index:06d}.up.sql"
                data = f"BEGIN;\nSELECT {index};\nCOMMIT;\n".encode()
                contents[path] = files[path] = data
                functional.append({"path": path, "sha256": SQL.sha(data)})
            adroot = "deploy/postgresql/autorizacion_atestada_v3/"
            ad132 = {}
            adpaths = {"sql": next(iter(SQL.H6_DBA_EXCLUDED)),
                       "cli": adroot + "aplicar_000132_temp_public.py",
                       "inventory": adroot + "migraciones/000132_inventario_temp_public.sql",
                       "doc": adroot + "000132_TEMP_PUBLIC.md",
                       "anchors_query": adroot + "000132_anclas_h6.sql",
                       "anchors_expected": adroot + "000132_anclas_h6.json"}
            for name, path in adpaths.items():
                files[path] = b"synthetic " + name.encode()
                ad132[name] = {"path": path, "sha256": SQL.sha(files[path])}
            list_data = ("\n".join(item["path"] for item in functional) + "\n").encode()
            release = {"version": 1, "source_commit": SQL.H6_FIRMA_FINAL_REF,
                       "functional_sql": functional, "ad132": ad132,
                       "sql_list_sha256": SQL.sha(list_data)}
            files["lista_sql_h6.txt"] = list_data
            files["h6-sql-release.json"] = json.dumps(release, sort_keys=True).encode()
            if mutation: mutation(files, release)
            package = root / "kit.tgz"
            with tarfile.open(package, "w:gz") as archive:
                for path, data in files.items():
                    member = tarfile.TarInfo("./" + path); member.size = len(data)
                    archive.addfile(member, io.BytesIO(data))
            package_sha = SQL.sha(package.read_bytes())
            lock = {"COMMIT": SQL.H6_FIRMA_FINAL_REF, "PAQUETE_SHA256": package_sha,
                    "SQL_LIST_SHA256": SQL.sha(files["lista_sql_h6.txt"]),
                    "SQL_RELEASE_SHA256": SQL.sha(files["h6-sql-release.json"])}
            for name, key in {"sql": "AD132_SQL_SHA256", "cli": "AD132_CLI_SHA256",
                    "inventory": "AD132_INVENTARIO_SHA256", "doc": "AD132_DOC_SHA256",
                    "anchors_query": "AD132_ANCLAS_SQL_SHA256",
                    "anchors_expected": "AD132_ANCLAS_JSON_SHA256"}.items():
                lock[key] = ad132[name]["sha256"]
            lock_file = root / "lock"; lock_file.write_text("\n".join(f"{k} {v}" for k, v in lock.items()) + "\n")
            h1 = root / "h1.tgz"; h1.write_bytes(b"synthetic H1")
            args = [package, lock_file, package_sha, SQL.sha(lock_file.read_bytes()), h1, SQL.sha(h1.read_bytes())]
            with patch.object(SQL.GitSource, "inventory", return_value=([], contents)):
                yield root, args

    def test_17_plus_45_exact_list_source_and_separate_family(self):
        with self.fixture() as (_, args):
            plan, rows = SQL.preflight_h6_package(*args, git_repo=GIT_REPO)
            self.assertEqual(len(rows), 62)
            self.assertEqual([r["phase"] for r in rows], ["H3"] * 8 + ["H4"] * 9 + ["H6"] * 45)
            self.assertEqual([r["path"] for r in rows[17:]], [f"deploy/postgresql/demo/{n:06d}.up.sql" for n in range(45)])
            self.assertEqual(plan["plan_family"], "h6_package_62")
            self.assertNotEqual(plan["plan_sha"], SQL.REF_PLAN_SHA[SQL.H6_FIRMA_FINAL_REF])
            self.assertEqual(plan["ad132_verification"], "blocked_pending_reviewed_contract")
            self.assertNotIn(next(iter(SQL.H6_DBA_EXCLUDED)), [r["path"] for r in rows])
            with self.assertRaisesRegex(SQL.Refused, "fuente Git73e"):
                SQL.preflight_h6_package(*args, source_ref=SQL.H6_FIRMA_REF, git_repo=GIT_REPO)

    def test_external_approval_required_and_tar_lock_h1_tampering_denied(self):
        for index in (2, 3, 5):
            with self.fixture() as (_, args):
                for value in (None, "", "f" * 64):
                    bad = list(args); bad[index] = value
                    with self.subTest(index=index, value=value), self.assertRaises(SQL.Refused):
                        SQL.preflight_h6_package(*bad, git_repo=GIT_REPO)
        for index in (0, 1, 4):
            with self.fixture() as (_, args):
                args[index].write_bytes(args[index].read_bytes() + b"tampered")
                with self.subTest(index=index), self.assertRaisesRegex(SQL.Refused, "huella aprobada"):
                    SQL.preflight_h6_package(*args, git_repo=GIT_REPO)

    def test_mixed_sql_unknown_up_order_duplicate_and_unsafe_paths_denied(self):
        def mixed(files, release):
            path = release["functional_sql"][0]["path"]
            files[path] += b"-- mixed\n"
            release["functional_sql"][0]["sha256"] = SQL.sha(files[path])
            files["h6-sql-release.json"] = json.dumps(release).encode()
        def extra(files, _): files["deploy/postgresql/demo/extra.up.sql"] = b"BEGIN;\nCOMMIT;\n"
        def order(files, _): files["lista_sql_h6.txt"] = b"\n".join(reversed(files["lista_sql_h6.txt"].splitlines())) + b"\n"
        def duplicate(files, release):
            release["functional_sql"][-1] = release["functional_sql"][0]
            files["lista_sql_h6.txt"] = ("\n".join(r["path"] for r in release["functional_sql"]) + "\n").encode()
            release["sql_list_sha256"] = SQL.sha(files["lista_sql_h6.txt"])
            files["h6-sql-release.json"] = json.dumps(release).encode()
        def unsafe(files, _): files["../unsafe.sql"] = b"dummy"
        for mutation in (mixed, extra, order, duplicate, unsafe):
            with self.subTest(mutation=mutation.__name__), self.fixture(mutation) as (_, args):
                with self.assertRaises(SQL.Refused): SQL.preflight_h6_package(*args, git_repo=GIT_REPO)

    def test_tar_duplicate_link_absolute_and_psql_commands_denied(self):
        for name, kind in (("./one.sql", tarfile.REGTYPE), ("/one.sql", tarfile.REGTYPE),
                           ("./link", tarfile.SYMTYPE)):
            stream = io.BytesIO()
            with tarfile.open(fileobj=stream, mode="w:gz") as archive:
                member = tarfile.TarInfo(name); member.type = kind
                archive.addfile(member)
                if name == "./one.sql": archive.addfile(member)
            with self.subTest(name=name), self.assertRaises(SQL.Refused): SQL.package_members(stream.getvalue())
        for data in (b"SELECT 1;", b"BEGIN;\n\\! echo bad\nCOMMIT;\n", b"BEGIN;\nCOMMIT;\nSELECT 1;"):
            with self.assertRaises(SQL.Refused):
                SQL.h6_sql_row("H6", "deploy/postgresql/demo/one.up.sql", SQL.sha(data), data)

    def test_psql_exit0_only_pending_crash_and_historical_journal_rejected(self):
        for fail in (None, KeyboardInterrupt(), subprocess.TimeoutExpired("fixture", 1), "postrename"):
            with self.fixture() as (root, args):
                plan, rows = SQL.preflight_h6_package(*args, git_repo=GIT_REPO)
                state = root / "state"; state.mkdir(mode=0o700)
                context = {k: plan[k] for k in (*SQL.CONTEXT_KEYS[1:], "lock_sha")}
                context["identidad_clon"] = "a" * 64
                class DB:
                    calls = []
                    def check_owner(self): pass
                    def query(self, text):
                        raw = json.loads((state / "sql-journal.json").read_text())
                        self.assert_pending = raw["pending"] is not None
                        if not self.assert_pending: raise AssertionError("no pending")
                        self.calls.append(text)
                        if fail and fail != "postrename": raise fail
                class Kit:
                    calls = []
                    def validate(self, plan, context): return True
                    def identity(self, ro, context): return context["identidad_clon"]
                    def confirm(self, ro, entry, position, context):
                        self.calls.append(position)
                        return position == 0 or position == 62
                    def verify(self, ro, record, context): return True
                db, kit = DB(), Kit(); db.calls = []; kit.calls = []
                def apply(): return SQL.apply(db, rows, state, SQL.H6_FIRMA_FINAL_REF, plan, context, kit)
                if fail:
                    original_store = SQL.Journal.store
                    def uncertain(journal, record):
                        original_store(journal, record)
                        if record["installed"]: raise OSError("directory fsync uncertain")
                    with contextlib.ExitStack() as stack:
                        if fail == "postrename": stack.enter_context(patch.object(SQL.Journal, "store", uncertain))
                        with self.assertRaises((SQL.Refused, KeyboardInterrupt, OSError)): apply()
                    self.assertTrue((state / ".sql-confirming").exists())
                    with self.assertRaisesRegex(SQL.Refused, "pendiente"): apply()
                    self.assertEqual(len(db.calls), 1)
                else:
                    record = apply()
                    self.assertEqual(len(db.calls), 62)
                    self.assertEqual(kit.calls, [0])
                    self.assertEqual({r["confirmation"] for r in record["installed"]}, {"psql_exit0_observed"})
                    self.assertEqual(record["phase"], "awaiting_ad132")
                    apply(); self.assertEqual(len(db.calls), 62)
                    record["plan_family"] = "h6_45"
                    with SQL.Journal(state) as journal: journal.store(record)
                    with self.assertRaisesRegex(SQL.Refused, "histórico"): apply()
                self.assertFalse((state / "READY.json").exists())

    def test_h6_cli_plan_json_missing_kit_and_final_blocked_before_docker(self):
        with self.fixture() as (root, args):
            command = ["--repo", str(REPO), "--git-repo", str(GIT_REPO), "--source-ref", SQL.H6_FIRMA_FINAL_REF]
            for option, value in zip(("--h6-package", "--h6-lock", "--approved-package-sha256",
                    "--approved-lock-sha256", "--h1-state-file", "--estado-h1-sha"), args):
                command.extend([option, str(value)])
            with patch.object(SQL.DockerDB, "check_owner") as docker:
                with contextlib.redirect_stdout(io.StringIO()) as output:
                    SQL.main([*command, "--plan"])
                self.assertEqual(json.loads(output.getvalue())["file_count"], 62)
                with self.assertRaisesRegex(SQL.Refused, "contrato revisado"):
                    SQL.main([*command, "--verify-live"])
                with self.assertRaisesRegex(SQL.Refused, "kit D"):
                    SQL.main([*command, "--installable"])
                docker.assert_not_called()
                self.assertFalse((root / "state").exists())

class PreimageProbeTests(unittest.TestCase):
    """Sondas fijas con transportes dobles; nunca requieren Docker o PostgreSQL."""
    @staticmethod
    def metadata():
        return {"Id": "c" * 64, "Image": "sha256:" + "d" * 64, "Config": {"Image": "postgres:18.4", "Labels": {
                    SQL.OWNER_LABEL: SQL.OWNER, "vec.recorridos.state": "/private/fixture"}},
                "Running": True, "Mounts": [{"Type": "bind", "Destination": "/var/lib/postgresql",
                                               "Source": "/dev/shm/vec-fixture"}],
                "NetworkMode": "none", "Ports": {"5432/tcp": None}}

    @contextlib.contextmanager
    def normalizer(self):
        with tempfile.TemporaryDirectory() as scratch:
            path = Path(scratch) / "h6_normalizar_pg_dump.py"
            data = b"import sys\nsys.stdout.buffer.write(sys.stdin.buffer.read())\n"
            path.write_bytes(data)
            yield path, SQL.sha(data), data

    def test_readonly_exposes_only_fixed_preimage_methods(self):
        db = SQL.DockerDB("vec-fixture")
        ro = SQL.ReadOnlyDB(db)
        for name, args in (("schema_digest", ("path", "sha")), ("roles_digest", ("path", "sha")),
                           ("database_acl_digest", ()), ("system_identity", ())):
            with self.subTest(name=name), patch.object(db, name, return_value="fixture") as method:
                self.assertEqual(getattr(ro, name)(*args), "fixture")
                method.assert_called_once_with(*args)

    def test_dump_commands_pin_normalizer_bytes_and_preserve_acl_owners(self):
        with self.normalizer() as (path, digest, data):
            for name, arguments in (("schema_digest", ["pg_dump", "-s", "-U", "postgres", "-d", "postgres"]),
                                   ("roles_digest", ["pg_dumpall", "--globals-only", "--no-role-passwords",
                                                     "-U", "postgres"])):
                with self.subTest(name=name), patch.object(SQL, "_probe_command", side_effect=[
                        json.dumps(self.metadata()).encode(), b"raw dump", b"normalized dump\n"]) as runner:
                    value = getattr(SQL.DockerDB("vec-fixture", Path("/private/fixture")), name)(path, digest)
                    self.assertEqual(value, SQL.sha(b"normalized dump\n"))
                    calls = runner.call_args_list
                    self.assertEqual(calls[0].args[0], ["docker", "inspect", "--format",
                                                       SQL.PROBE_INSPECT_FORMAT, "vec-fixture"])
                    self.assertNotIn("Env", SQL.PROBE_INSPECT_FORMAT)
                    self.assertEqual(calls[1].args[0], ["docker", "exec", "-e",
                        "PGOPTIONS=-c default_transaction_read_only=on -c statement_timeout=120000 -c lock_timeout=15000", "c" * 64, *arguments])
                    self.assertEqual(calls[2].args, ([SQL.sys.executable, "-I", "-S", "-c", data.decode()], b"raw dump"))
                    self.assertNotIn(str(path), calls[2].args[0])

    def test_unapproved_unsafe_or_linked_normalizer_never_starts_probe(self):
        with self.normalizer() as (path, digest, data):
            linked = path.parent / "link"; linked.symlink_to(path.parent, target_is_directory=True)
            wrong_name = path.parent / "other.py"; wrong_name.write_bytes(data)
            for candidate, pin in ((path, "f" * 64), (path, None), (wrong_name, digest),
                                   (linked / path.name, digest), (Path(path.name), digest),
                                   (str(path.parent) + "/../" + path.parent.name + "/" + path.name, digest),
                                   (path.parent / "missing" / path.name, digest)):
                with self.subTest(candidate=candidate), patch.object(SQL, "_probe_command") as runner:
                    with self.assertRaises(SQL.Refused):
                        SQL.DockerDB("vec-fixture").schema_digest(candidate, pin)
                    runner.assert_not_called()
            hardlink = path.parent / "hard"; os.link(path, hardlink)
            with patch.object(SQL, "_probe_command") as runner, self.assertRaises(SQL.Refused):
                SQL.DockerDB("vec-fixture").schema_digest(path, digest)
            runner.assert_not_called()
            hardlink.unlink()
            path.write_bytes(b"x" * (64 * 1024 + 1))
            with patch.object(SQL, "_probe_command") as runner, self.assertRaises(SQL.Refused):
                SQL.DockerDB("vec-fixture").roles_digest(path, SQL.sha(path.read_bytes()))
            runner.assert_not_called()

    def test_sql_probes_are_read_only_fixed_and_return_exact_contract(self):
        db = SQL.DockerDB("vec-fixture")
        identity = {"system_identifier": "7533565316322819991", "database_name": "postgres", "database_oid": 5}
        for method, statement, value, expected in (
                (db.database_acl_digest, SQL.DATABASE_ACL_SQL, "a" * 64, "a" * 64),
                (db.system_identity, SQL.SYSTEM_IDENTITY_SQL, json.dumps(identity),
                 {**identity, "pg_container_id": "c" * 64, "pg_image": "postgres:18.4",
                  "pg_image_id": "sha256:" + "d" * 64, "pg_volume": "/dev/shm/vec-fixture"})):
            with self.subTest(method=method.__name__), patch.object(SQL, "_probe_command", side_effect=[
                    json.dumps(self.metadata()).encode(), (value + "\n").encode()]) as runner:
                self.assertEqual(method(), expected)
                call = runner.call_args_list[1]
                self.assertEqual(call.args[1], ("BEGIN READ ONLY;\n" + statement + ";\nCOMMIT;").encode())
                self.assertEqual(call.kwargs["limit"], 4096)
                self.assertEqual(call.args[0], ["docker", "exec", "-i", "c" * 64, "psql", "-h",
                    "/var/run/postgresql", "-p", "5432", "-U", "postgres", "-d", "postgres", "-X", "-q", "-A",
                    "-t", "-v", "ON_ERROR_STOP=1"])
        self.assertIn("'acl',datacl::text,'owner',datdba::regrole::text", SQL.DATABASE_ACL_SQL)
        self.assertIn("pg_catalog.pg_control_system()", SQL.SYSTEM_IDENTITY_SQL)

    def test_normalizer_path_mutation_after_pin_cannot_change_executed_bytes(self):
        with self.normalizer() as (path, digest, data):
            def fixture(command, payload=None, **kwargs):
                if command[:2] == ["docker", "inspect"]:
                    return json.dumps(self.metadata()).encode()
                if command[:2] == ["docker", "exec"]:
                    path.write_bytes(b"unapproved bytes")
                    return b"dump"
                self.assertEqual(command, [SQL.sys.executable, "-I", "-S", "-c", data.decode()])
                self.assertEqual(payload, b"dump")
                return b"normalized"
            with patch.object(SQL, "_probe_command", side_effect=fixture):
                self.assertEqual(SQL.DockerDB("vec-fixture").schema_digest(path, digest), SQL.sha(b"normalized"))

    def test_metadata_rejects_foreign_container_mount_image_ports_and_state(self):
        changes = [{"Id": "not-id"}, {"Image": "not-image-id"}, {"Running": False},
                   {"NetworkMode": "bridge"}, {"Config": {"Image": "postgres:latest"}},
                   {"Mounts": []}, {"Mounts": [{"Type": "volume", "Destination": "/var/lib/postgresql",
                                                "Source": "/dev/shm/fixture"}]},
                   {"Mounts": [{"Type": "bind", "Destination": "/var/lib/postgresql", "Source": "/private"}]},
                   {"Ports": {"5432/tcp": [{"HostIp": "0.0.0.0"}]}},
                   {"Ports": {"5432/tcp": [{"HostIp": "127.0.0.1"}]}},
                   {"Config": {"Image": "postgres:18.4", "Labels": {SQL.OWNER_LABEL: "foreign"}}}]
        with self.normalizer() as (path, digest, _):
            for change in changes:
                for method, args in (("system_identity", ()), ("database_acl_digest", ()),
                                     ("schema_digest", (path, digest)), ("roles_digest", (path, digest))):
                    with self.subTest(change=change, method=method), patch.object(SQL, "_probe_command",
                            return_value=json.dumps({**self.metadata(), **change}).encode()) as runner:
                        with self.assertRaises(SQL.Refused): getattr(SQL.DockerDB("vec-fixture"), method)(*args)
                        self.assertEqual(runner.call_count, 1)
            with patch.object(SQL, "_probe_command", return_value=json.dumps(self.metadata()).encode()):
                with self.assertRaises(SQL.Refused):
                    SQL.DockerDB("vec-fixture", Path("/wrong-state")).system_identity()

    def test_immutable_image_launch_requires_external_pin_and_returns_logical_family(self):
        for configured_image in ("postgres:18.4", "sha256:" + "d" * 64):
            metadata = self.metadata()
            metadata["Config"]["Image"] = configured_image
            identity = {"system_identifier": "7533565316322819991", "database_name": "postgres", "database_oid": 5}
            with self.subTest(image=configured_image), patch.object(SQL, "_probe_command", side_effect=[
                    json.dumps(metadata).encode(), json.dumps(identity).encode()]):
                result = SQL.DockerDB("vec-fixture", expected_image_id="sha256:" + "d" * 64).system_identity()
                self.assertEqual(result["pg_image"], "postgres:18.4")
                self.assertEqual(result["pg_image_id"], "sha256:" + "d" * 64)

    def test_image_pin_rejects_missing_foreign_and_invalid_approval_before_query(self):
        for configured_image, expected in (("sha256:" + "d" * 64, None),
                ("sha256:" + "d" * 64, "sha256:" + "e" * 64),
                ("postgres:18.4", "sha256:" + "e" * 64),
                ("postgres:other", "sha256:" + "d" * 64),
                (None, None), ("sha256:" + "e" * 64, "sha256:" + "d" * 64)):
            metadata = self.metadata()
            metadata["Config"]["Image"] = configured_image
            with self.subTest(image=configured_image, pin=expected), \
                    patch.object(SQL, "_probe_command", return_value=json.dumps(metadata).encode()) as runner:
                with self.assertRaises(SQL.Refused):
                    SQL.DockerDB("vec-fixture", expected_image_id=expected).system_identity()
                self.assertEqual(runner.call_count, 1)
        for expected in ("", "postgres:18.4", "SHA256:" + "d" * 64, True, []):
            with self.subTest(pin=expected), patch.object(SQL, "_probe_command") as runner:
                with self.assertRaises(SQL.Refused):
                    SQL.DockerDB("vec-fixture", expected_image_id=expected)
                runner.assert_not_called()

    def test_additional_mounts_cannot_replace_probed_data_or_socket(self):
        destinations = ("/var/run/postgresql", "/var/lib/postgresql/18/docker",
                        "/var/lib/postgresql", "/var/lib", "/", "/extra")
        with self.normalizer() as (path, digest, _):
            for kind in ("bind", "volume", "tmpfs"):
                for destination in destinations:
                    extra = {"Type": kind, "Destination": destination, "Source": "/dev/shm/other"}
                    for first in (False, True):
                        metadata = self.metadata()
                        metadata["Mounts"].insert(0 if first else 1, extra)
                        for method, args in (("system_identity", ()), ("database_acl_digest", ()),
                                             ("schema_digest", (path, digest)), ("roles_digest", (path, digest))):
                            with self.subTest(kind=kind, destination=destination, first=first, method=method), \
                                    patch.object(SQL, "_probe_command", return_value=json.dumps(metadata).encode()) as runner:
                                with self.assertRaises(SQL.Refused):
                                    getattr(SQL.DockerDB("vec-fixture"), method)(*args)
                                self.assertEqual(runner.call_count, 1)

    def test_only_exact_postgresql_bind_destination_is_accepted(self):
        for destination in ("/var/lib/postgresql/18/docker", "/var/lib", "/var/run/postgresql",
                            "/var/lib/postgresql/", "/var/lib/./postgresql", "/"):
            metadata = self.metadata()
            metadata["Mounts"][0]["Destination"] = destination
            with self.subTest(destination=destination), \
                    patch.object(SQL, "_probe_command", return_value=json.dumps(metadata).encode()) as runner:
                with self.assertRaises(SQL.Refused):
                    SQL.DockerDB("vec-fixture").system_identity()
                self.assertEqual(runner.call_count, 1)

    def test_mount_collection_must_be_one_complete_bind_record(self):
        for mounts in (None, {}, "mount", [], [None], [{}],
                       [{"Destination": "/var/lib/postgresql", "Type": "bind"}]):
            metadata = {**self.metadata(), "Mounts": mounts}
            with self.subTest(mounts=mounts), \
                    patch.object(SQL, "_probe_command", return_value=json.dumps(metadata).encode()) as runner:
                with self.assertRaises(SQL.Refused):
                    SQL.DockerDB("vec-fixture").system_identity()
                self.assertEqual(runner.call_count, 1)

    def test_malformed_sql_output_is_refused(self):
        identity = {"system_identifier": "7533565316322819991", "database_name": "postgres", "database_oid": 5}
        invalid = [b"not-json", b"[]", b"null", json.dumps({**identity, "database_name": "other"}).encode(),
                   json.dumps({**identity, "database_oid": True}).encode(),
                   json.dumps({**identity, "system_identifier": "0"}).encode(),
                   json.dumps({**identity, "system_identifier": str(2 ** 64)}).encode(),
                   json.dumps({**identity, "extra": "untrusted"}).encode()]
        for method, outputs in (("system_identity", invalid),
                                ("database_acl_digest", [b"", b"A" * 64, b"a" * 64 + b"\na", b"secret", b"\xff"])):
            for output in outputs:
                with self.subTest(method=method, output=output), patch.object(SQL, "_probe_command", side_effect=[
                        json.dumps(self.metadata()).encode(), output]):
                    with self.assertRaises(SQL.Refused): getattr(SQL.DockerDB("vec-fixture"), method)()

    class Process:
        """Tubos locales con salida fixture; no lanza comandos del sistema."""
        def __init__(self, stdout=b"", stderr=b"", code=0, **kwargs):
            self.code, self.returncode, self.killed = code, None, False
            self.stdout, self.stderr = self.pipe(stdout), self.pipe(stderr)
            if kwargs.get("stdin") == subprocess.PIPE:
                read, write = os.pipe()
                self.stdin_reader = os.fdopen(read, "rb")
                self.stdin = os.fdopen(write, "wb")
            else:
                self.stdin, self.stdin_reader = None, None
        @staticmethod
        def pipe(data):
            read, write = os.pipe()
            os.write(write, data); os.close(write)
            return os.fdopen(read, "rb")
        def wait(self, timeout=None): self.returncode = self.code; return self.code
        def poll(self): return self.returncode
        def kill(self): self.killed = True; self.code = -9
        def close(self):
            if self.stdin_reader: self.stdin_reader.close()

    def test_transport_bounds_output_and_errors_and_redacts_failures(self):
        for stdout, stderr, code, limit, error in (
                (b"result", b"", 0, 6, None), (b"secret-value", b"", 0, 6, "límite de salida"),
                (b"", b"secret-value", 0, 6, "límite de error"),
                (b"secret-value", b"secret-value", 1, 100, "falló")):
            with self.subTest(error=error):
                process = self.Process(stdout, stderr, code)
                try:
                    with patch.object(SQL.subprocess, "Popen", return_value=process) as popen, \
                            patch.object(SQL, "PROBE_STDERR_LIMIT", 6 if error == "límite de error" else 100):
                        if error:
                            with self.assertRaisesRegex(SQL.Refused, error) as raised:
                                SQL._probe_command(["fixed-fixture"], limit=limit)
                            self.assertNotIn("secret-value", str(raised.exception))
                        else:
                            self.assertEqual(SQL._probe_command(["fixed-fixture"], limit=limit), b"result")
                        self.assertEqual(popen.call_args.kwargs["env"], {"PATH": "/usr/bin:/bin", "LANG": "C.UTF-8"})
                        self.assertNotIn("shell", popen.call_args.kwargs)
                        self.assertTrue(process.stdout.closed and process.stderr.closed)
                        if error and code == 0: self.assertTrue(process.killed)
                finally: process.close()

    def test_transport_timeout_spawn_error_and_stdin_are_bounded(self):
        process = self.Process(b"result", stdin=subprocess.PIPE)
        try:
            with patch.object(SQL.subprocess, "Popen", return_value=process):
                self.assertEqual(SQL._probe_command(["fixed-fixture"], b"input"), b"result")
                self.assertEqual(process.stdin_reader.read(), b"input")
        finally: process.close()
        process = self.Process()
        with patch.object(SQL.subprocess, "Popen", return_value=process), \
                patch.object(SQL.time, "monotonic", side_effect=[0, 2]):
            with self.assertRaisesRegex(SQL.Refused, "tiempo"):
                SQL._probe_command(["fixed-fixture"], timeout=1)
            self.assertTrue(process.killed)
        with patch.object(SQL.subprocess, "Popen", side_effect=OSError("secret-value")):
            with self.assertRaises(SQL.Refused) as raised: SQL._probe_command(["fixed-fixture"])
            self.assertNotIn("secret-value", str(raised.exception))


if __name__ == "__main__":
    unittest.main()
