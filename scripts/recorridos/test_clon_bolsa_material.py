"""Focal failure checks for clone ownership and nominal PostgreSQL identities."""
import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("clon_bolsa_material", Path(__file__).with_name("clon_bolsa_material.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def snapshot(with_calculator=False):
    def role(name, login=False):
        return {"rolname": name, "rolcanlogin": login, "rolinherit": True,
                "rolsuper": False, "rolcreatedb": False, "rolcreaterole": False,
                "rolreplication": False, "rolbypassrls": False,
                "rolconnlimit": 4 if name == module.CALCULATOR else -1,
                "password_digest": "fixture"}
    value = {"roles": [], "memberships": []}
    for login, group in module.REQUIRED.items():
        value["roles"].append(role(group))
        if login != module.CALCULATOR or with_calculator:
            value["roles"].append(role(login, True))
            value["memberships"].append({"member": login, "role": group,
                "admin_option": False, "inherit_option": True,
                "set_option": login == module.AUDIT})
    return value


class FakeClone:
    def __init__(self):
        self.before = snapshot()
        self.after = snapshot(True)
        self.current = self.before
        self.statements = []

    def snapshot(self):
        return copy.deepcopy(self.current)

    def sql(self, statement):
        self.statements.append(statement)
        if statement.endswith("COMMIT;\n"):
            self.current = self.after
        return ""


class FakeSQLAuthority:
    def __init__(self):
        self.calls = []

    def validate_receipts(self, installed, plan, complete=True):
        self.calls.append((installed, plan, complete))
        actual = [(r.get("position"), r.get("path"), r.get("sha256")) for r in installed]
        expected = [(i, r["path"], r["sha256"]) for i, r in enumerate(plan["entries"], 1)]
        if actual != expected or not complete:
            raise RuntimeError("fixture receipt disagreement")


def convoca_inventory(connect=False):
    roles = []
    for name in (module.IMPORT_LOGIN, *module.IMPORT_GROUPS):
        roles.append({"rolname": name, "rolcanlogin": name == module.IMPORT_LOGIN,
                      "rolinherit": True, "rolsuper": False, "rolcreatedb": False,
                      "rolcreaterole": False, "rolreplication": False, "rolbypassrls": False,
                      "rolconfig": None if name == module.IMPORT_LOGIN else ["search_path=pg_catalog,pg_temp"]})
    return {"roles": roles,
            "memberships": [{"member": module.IMPORT_LOGIN, "role": g, "admin_option": False,
                             "inherit_option": True, "set_option": True} for g in module.IMPORT_GROUPS],
            "db_role_settings": 0, "owned_objects": 0, "direct_database_acl": 0, "connect": connect}


class FakeConvoca:
    def __init__(self, connect=False):
        self.inventory = convoca_inventory(connect)
        self.plan = {"source_ref": "8fc0b534dbdaa0a5b34d835810c4593ad823780b"}
        self.calls = []
        self.physical = module.IMPORT_LOGIN + "|true|true|true|true|TLSv1.3"

    def sql(self, statement, **kwargs):
        self.calls.append((statement, kwargs))
        return json.dumps(self.inventory) if "pg_db_role_setting" in statement else self.physical


class ProvisionTests(unittest.TestCase):
    @staticmethod
    def convoca_fixture(root):
        material = root / "material"
        (material / "kms").mkdir(parents=True, mode=0o700)
        key = material / "kms/clave-maestra.bin"
        key.write_bytes(b"d" * 32)
        key.chmod(0o600)
        return material

    def test_convoca_connect_pending_returns_nominal_url_without_accreditation(self):
        from urllib.parse import urlsplit
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            material = self.convoca_fixture(root)
            clone = FakeConvoca()
            uri, proof, blockers = module._prepare_convoca(clone, material, root / "ca.crt", 55577, "/private/ca.crt")
            self.assertEqual(urlsplit(uri).username, module.IMPORT_LOGIN)
            self.assertEqual(proof["status"], "metadata_only")
            self.assertEqual(proof["accreditation"], "not_accredited")
            self.assertFalse(proof["tls_probe_verified"])
            self.assertEqual(blockers, [{"profile": "bolsa", "code": "bolsa_importacion_convoca_connect_pendiente"}])
            self.assertEqual(len(clone.calls), 1)
            self.assertNotIn("user", clone.calls[0][1])
            self.assertEqual((material / "kms/clave-maestra.bin").read_bytes(), b"d" * 32)

    def test_convoca_connect_repaired_requires_physical_login_tls_probe(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            material = self.convoca_fixture(root)
            clone = FakeConvoca(connect=True)
            _, proof, blockers = module._prepare_convoca(clone, material, root / "ca.crt", 55577, "/private/ca.crt")
            self.assertEqual(blockers, [])
            self.assertTrue(proof["tls_probe_verified"])
            self.assertEqual(proof["source_ref"], clone.plan["source_ref"])
            self.assertEqual(proof["pending"], ["rrhh_http_read_and_recovery"])
            self.assertEqual(clone.calls[1][1], {"user": module.IMPORT_LOGIN, "tls_ca": "/private/ca.crt"})
            for statement, _ in clone.calls:
                self.assertTrue(statement.lstrip().startswith("SELECT"))
                self.assertFalse(any(token in statement.upper() for token in ("CREATE ", "GRANT ", "ALTER ", "REVOKE ")))

    def test_convoca_preserves_exact_two_historical_memberships(self):
        original = convoca_inventory(connect=True)
        self.assertTrue(module._convoca_nominal(original))
        self.assertEqual(original, convoca_inventory(connect=True))
        for failure in ("missing_login", "nologin", "extra_role", "admin", "settings", "owned"):
            with self.subTest(failure=failure):
                value = copy.deepcopy(original)
                if failure == "missing_login":
                    value["roles"] = value["roles"][1:]
                elif failure == "nologin":
                    value["roles"][0]["rolcanlogin"] = False
                elif failure == "extra_role":
                    value["memberships"].append({"member": module.IMPORT_LOGIN, "role": "unrelated_owner",
                                                 "admin_option": False, "inherit_option": True, "set_option": True})
                elif failure == "admin":
                    value["memberships"][0]["admin_option"] = True
                elif failure == "settings":
                    value["db_role_settings"] = 1
                else:
                    value["owned_objects"] = 1
                with self.assertRaises(module.ProvisionError):
                    module._convoca_nominal(value)

    def test_convoca_failed_physical_probe_does_not_claim_accreditation(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            material = self.convoca_fixture(root)
            clone = FakeConvoca(connect=True)
            clone.physical = module.IMPORT_LOGIN + "|true|true|true|false|"
            with self.assertRaisesRegex(module.ProvisionError, "bolsa_importacion_convoca_sonda_nominal_tls_fallida"):
                module._prepare_convoca(clone, material, root / "ca.crt", 55577, "/private/ca.crt")

    @staticmethod
    def target_fixture(state, container, port):
        inventory = {"propietario": "Codex-M", "estado": str(state),
                     "contenedor": container, "commit": module.SOURCE, "puerto_pg": port}
        ready = {**inventory, "sql_instaladas": 38}
        _, journal, plan = ProvisionTests.ready_receipts(module.SOURCE, 38)
        inspection = [{"Config": {"Labels": {"vec.recorridos.owner": "Codex-M",
                                                "vec.recorridos.state": str(state)}},
                       "State": {"Running": True},
                       "NetworkSettings": {"Ports": {"5432/tcp": [{"HostIp": "127.0.0.1", "HostPort": str(port)}]}}}]
        documents = {"clon.json": inventory, "DB_READY.json": ready, "sql-journal.json": journal}
        return documents, inspection, plan

    def test_owned_new_target_and_port_are_taken_from_persisted_inventory(self):
        state, container, port = Path("/synthetic-private/new-target"), "vec-codexm-new-target", 55577
        documents, inspection, plan = self.target_fixture(state, container, port)
        clone = module.Clone("docker", container, state, port, plan, FakeSQLAuthority())
        with patch.object(module, "_read_json", side_effect=lambda path: documents[path.name]), patch.object(module, "_run", return_value=json.dumps(inspection)):
            self.assertEqual(clone.guard(), documents["DB_READY.json"])
        self.assertIn(":55577/postgres?", module._dsn(module.CALCULATOR, port, state / "material/pg/ca.crt"))

    def test_generic_target_rejects_wrong_binding_state_or_inventory_port(self):
        state, container, port = Path("/synthetic-private/new-target"), "vec-codexm-new-target", 55577
        for failure in ("binding", "label", "inventory"):
            with self.subTest(failure=failure):
                documents, inspection, plan = self.target_fixture(state, container, port)
                if failure == "binding":
                    inspection[0]["NetworkSettings"]["Ports"]["5432/tcp"][0]["HostPort"] = "55531"
                elif failure == "label":
                    inspection[0]["Config"]["Labels"]["vec.recorridos.state"] = "/another-state"
                else:
                    documents["clon.json"]["puerto_pg"] = 55531
                clone = module.Clone("docker", container, state, port, plan, FakeSQLAuthority())
                with patch.object(module, "_read_json", side_effect=lambda path: documents[path.name]), patch.object(module, "_run", return_value=json.dumps(inspection)):
                    with self.assertRaises(module.ProvisionError):
                        clone.guard()

    @staticmethod
    def ready_receipts(source, count):
        ready = {"commit": source, "sql_instaladas": count}
        plan = {"source_ref": source, "approved_sql_ref": "a" * 40,
                "plan_sha": "b" * 64, "inventory_sha": "c" * 64,
                "file_count": count, "verified_main_ref": "d" * 40,
                "entries": [{"phase": "MAIN", "path": f"deploy/postgresql/fixture/{i}.sql",
                             "sha256": "a" * 64} for i in range(1, count + 1)]}
        journal = {"source_ref": module.SOURCE, "current_source_ref": source,
                   "verified_source_ref": source, "approved_sql_ref": plan["approved_sql_ref"],
                   "current_plan_sha": plan["plan_sha"], "inventory_sha": plan["inventory_sha"],
                   "installed": [{"position": i, "path": f"deploy/postgresql/fixture/{i}.sql",
                                  "sha256": "a" * 64} for i in range(1, count + 1)]}
        return ready, journal, plan

    def test_authority_plan_requires_exact_receipt_count_and_delegates_validation(self):
        ready, journal, plan = self.ready_receipts("e" * 40, 38)
        authority = FakeSQLAuthority()
        module._ready_receipts(ready, journal, plan, authority)
        self.assertEqual(authority.calls, [(journal["installed"], plan, True)])
        ready["sql_instaladas"] = 39
        with self.assertRaisesRegex(module.ProvisionError, "clone_receipt_count_mismatch"):
            module._ready_receipts(ready, journal, plan, authority)

    def test_approved_plan_rejects_missing_receipts_and_different_current_source(self):
        source = "e" * 40
        ready, journal, plan = self.ready_receipts(source, 38)
        journal["installed"].pop()
        with self.assertRaisesRegex(module.ProvisionError, "clone_receipt_count_mismatch"):
            module._ready_receipts(ready, journal, plan, FakeSQLAuthority())
        ready, journal, plan = self.ready_receipts(source, 38)
        journal["current_source_ref"] = module.SOURCE
        with self.assertRaisesRegex(module.ProvisionError, "clone_receipt_source_mismatch"):
            module._ready_receipts(ready, journal, plan, FakeSQLAuthority())

    def test_duplicate_receipt_and_invalid_hash_rejected(self):
        for failure in ("duplicate", "digest"):
            with self.subTest(failure=failure):
                ready, journal, plan = self.ready_receipts("e" * 40, 38)
                if failure == "duplicate":
                    journal["installed"][-1] = copy.deepcopy(journal["installed"][0])
                else:
                    journal["installed"][-1]["sha256"] = "invalid"
                with self.assertRaises(module.ProvisionError):
                    module._ready_receipts(ready, journal, plan, FakeSQLAuthority())

    def test_creation_source_preserved_with_verified_current_source(self):
        source = "e" * 40
        ready, journal, plan = self.ready_receipts(source, 38)
        ready["commit"], ready["current_source_ref"] = module.SOURCE, source
        module._ready_receipts(ready, journal, plan, FakeSQLAuthority())

    def test_legacy_journal_without_sql_approval_fails_closed(self):
        ready, journal, plan = self.ready_receipts("e" * 40, 38)
        del journal["approved_sql_ref"]
        with self.assertRaisesRegex(module.ProvisionError, "clone_receipt_source_mismatch"):
            module._ready_receipts(ready, journal, plan, FakeSQLAuthority())

    def test_new_sql_refusal_remains_a_fixed_structured_driver_blocker(self):
        from types import SimpleNamespace
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory)
            source = "e" * 40
            archive = state / ("source-" + source)
            archive.mkdir(mode=0o700)
            authority = SimpleNamespace(approved_source_plan=lambda *args, **kwargs: (_ for _ in ()).throw(RuntimeError("dummy SQL needs new approval")))
            with self.assertRaisesRegex(module.ProvisionError, "sql_source_needs_review"):
                module._approved_plan(state, state, {"commit": source}, None, authority)

    def test_missing_archive_never_falls_back_to_git_worktree(self):
        from unittest.mock import Mock
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory)
            authority = Mock()
            with self.assertRaisesRegex(module.ProvisionError, "source_archive_missing"):
                module._approved_plan(state, state, {"commit": "e" * 40}, None, authority)
            authority.approved_source_plan.assert_not_called()

    def test_explicit_context_passes_archive_git_control_and_current_source(self):
        from unittest.mock import Mock
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory)
            archive = state / "reviewed-archive"
            archive.mkdir(mode=0o700)
            source = "e" * 40
            _, _, plan = self.ready_receipts(source, 38)
            authority = Mock()
            authority.approved_source_plan.return_value = plan
            context = {"source_repo": archive, "git_repo": state, "source_ref": source}
            actual, git_repo = module._approved_plan(state, state, {"commit": source}, context, authority)
            self.assertEqual((actual, git_repo), (plan, state))
            authority.approved_source_plan.assert_called_once_with(archive, source, git_repo=state)

    def test_other_clone_rejected_before_any_process(self):
        with patch.object(module, "_run") as run:
            result = module.provision(Path("/irrelevant"), "principal", module.STATE,
                                      module.STATE / "material", 55531)
        self.assertEqual(result["blockers"], [{"profile": "bolsa", "code": "target_outside_authorized_clone"}])
        run.assert_not_called()

    def test_driver_can_filter_and_report_structured_blockers(self):
        result = module.provision(Path("/irrelevant"), "principal", module.STATE,
                                  module.STATE / "material", 55531)
        # The caller filters by code and later emits the same dictionaries.
        pending = [b for b in result["blockers"] if b["code"] != "optional_dependency"]
        encoded = json.dumps({"blockers": pending})
        self.assertEqual(json.loads(encoded)["blockers"][0]["profile"], "bolsa")
        self.assertEqual(pending[0]["code"], "target_outside_authorized_clone")

    def test_driver_gets_fixed_code_for_incidental_os_error(self):
        with patch.object(module, "_check_path", side_effect=OSError("private dummy detail")):
            result = module.provision(Path("/irrelevant"), module.CONTAINER, module.STATE,
                                      module.STATE / "material", 55531)
        self.assertEqual(result["blockers"], [{"profile": "bolsa", "code": "bolsa_material_invalid_or_unavailable"}])
        self.assertNotIn("private dummy detail", json.dumps(result))

    def test_sql_explicit_local_endpoint_ignores_ambient_host_and_port(self):
        clone = module.Clone("docker", module.CONTAINER, module.STATE, 55531)
        for tls_ca, expected_host in ((None, "/var/run/postgresql"), ("/var/lib/postgresql/ca.crt", "localhost")):
            with self.subTest(tls=bool(tls_ca)), patch.object(clone, "guard"), patch.object(module, "_run", return_value="") as run:
                clone.sql("SELECT 1;", tls_ca=tls_ca)
                arguments = run.call_args.args[0]
                self.assertEqual(arguments[arguments.index("-h") + 1], expected_host)
                self.assertEqual(arguments[arguments.index("-p") + 1], "5432")

    def test_revoked_login_not_reactivated(self):
        data = snapshot(True)
        next(r for r in data["roles"] if r["rolname"] == module.CALCULATOR)["rolcanlogin"] = False
        with self.assertRaises(module.ProvisionError):
            module._nominal(data, module.CALCULATOR, module.CALCULATOR_GROUP)

    def test_second_membership_and_group_inheritance_rejected(self):
        for change in ("login", "group"):
            with self.subTest(change=change):
                data = snapshot(True)
                data["memberships"].append({"member": module.CALCULATOR if change == "login" else module.CALCULATOR_GROUP,
                    "role": "unrelated_owner", "admin_option": False,
                    "inherit_option": True, "set_option": False})
                with self.assertRaises(module.ProvisionError):
                    module._nominal(data, module.CALCULATOR, module.CALCULATOR_GROUP)

    def test_historical_auditor_set_true_preserved(self):
        data = snapshot()
        self.assertTrue(module._nominal(data, module.AUDIT, module.REQUIRED[module.AUDIT]))
        self.assertTrue(next(m for m in data["memberships"] if m["member"] == module.AUDIT)["set_option"])

    def test_first_creation_rolls_back_then_commits_and_replays_without_ddl(self):
        clone = FakeClone()
        with tempfile.TemporaryDirectory() as directory:
            private = Path(directory)
            self.assertTrue(module._provision_calculator(clone, private))
            self.assertEqual(len(clone.statements), 2)
            self.assertTrue(clone.statements[0].endswith("ROLLBACK;\n"))
            self.assertTrue(clone.statements[1].endswith("COMMIT;\n"))
            self.assertFalse(module._provision_calculator(clone, private))
            self.assertEqual(len(clone.statements), 2)
            self.assertEqual((private / "calculator-intent.json").stat().st_mode & 0o777, 0o600)

    def test_rollback_mismatch_prevents_commit(self):
        clone = FakeClone()
        original = clone.sql
        def corrupt(statement):
            original(statement)
            clone.current = clone.after
        clone.sql = corrupt
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(module.ProvisionError):
                module._provision_calculator(clone, Path(directory))
        self.assertEqual(len(clone.statements), 1)
        self.assertTrue(clone.statements[0].endswith("ROLLBACK;\n"))

    def test_private_output_refuses_symlink_and_existing_changes(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "result.json"
            module._write_new(path, {"fixture": 1})
            with self.assertRaises(module.ProvisionError):
                module._write_new(path, {"fixture": 2})
            alias = Path(directory) / "alias.json"
            alias.symlink_to(path)
            with self.assertRaises(module.ProvisionError):
                module._write_new(alias, {"fixture": 1})

    def test_password_and_ca_are_url_encoded(self):
        value = module._dsn(module.CALCULATOR, 55531, Path("/private ca/cert.crt"), "dummy:@/ ?")
        self.assertNotIn("dummy:@/ ?", value)
        self.assertIn("sslmode=verify-full", value)
        self.assertNotIn("ssl_min_protocol_version", value)

    def test_sql_writer_lock_busy(self):
        import fcntl
        import os
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            fd = os.open(root / "sql.lock", os.O_CREAT | os.O_RDWR, 0o600)
            try:
                fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
                with self.assertRaisesRegex(module.ProvisionError, "clone_sql_lock_busy"):
                    with module._sql_lock(root):
                        self.fail("must not enter a concurrent writer window")
            finally:
                os.close(fd)

    def test_contract_change_rejected(self):
        from types import SimpleNamespace
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(module, "CONTRACT_HASHES", {"contract.go": "unchanged digest"}), patch.object(module.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=b"changed")):
                with self.assertRaisesRegex(module.ProvisionError, "source_contract_changed"):
                    module._source_contracts(root, module.SOURCE)

    def test_committed_source_accepted_despite_divergent_working_tree(self):
        import hashlib
        from types import SimpleNamespace
        committed = b"reviewed exact bytes\n\n"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "contract.go").write_bytes(b"unrelated WIP")
            hashes = {"contract.go": hashlib.sha256(committed).hexdigest()}
            with patch.object(module, "CONTRACT_HASHES", hashes), patch.object(module.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=committed)) as run:
                module._source_contracts(root, module.SOURCE)
                self.assertEqual(run.call_args.args[0], ["git", "-C", str(root), "show", module.SOURCE + ":contract.go"])

    def test_unknown_source_rejected_without_git_execution(self):
        with patch.object(module.subprocess, "run") as run:
            with self.assertRaisesRegex(module.ProvisionError, "source_commit_not_authorized"):
                module._source_contracts(Path("/irrelevant"), "unknown")
        run.assert_not_called()


if __name__ == "__main__":
    unittest.main()
