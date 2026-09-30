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
