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


class ProvisionTests(unittest.TestCase):
    @staticmethod
    def ready_receipts(source, count):
        ready = {"commit": source, "sql_instaladas": count}
        journal = {"source_ref": module.SOURCE, "current_source_ref": source,
                   "installed": [{"position": i, "path": f"deploy/postgresql/fixture/{i}.sql",
                                  "sha256": "a" * 64} for i in range(1, count + 1)]}
        return ready, journal

    def test_each_authorized_source_requires_its_exact_receipt_count(self):
        for source, count in module.SOURCE_SQL_COUNTS.items():
            with self.subTest(count=count):
                ready, journal = self.ready_receipts(source, count)
                module._ready_receipts(ready, journal)
                ready["sql_instaladas"] = count + 1
                with self.assertRaisesRegex(module.ProvisionError, "clone_receipt_count_mismatch"):
                    module._ready_receipts(ready, journal)

    def test_source_36_rejects_missing_receipts_and_different_current_source(self):
        source = "e78687528d5725efd74e95c858d389f4437099ca"
        ready, journal = self.ready_receipts(source, 36)
        journal["installed"].pop()
        with self.assertRaisesRegex(module.ProvisionError, "clone_receipt_count_mismatch"):
            module._ready_receipts(ready, journal)
        ready, journal = self.ready_receipts(source, 36)
        journal["current_source_ref"] = module.SOURCE
        with self.assertRaisesRegex(module.ProvisionError, "clone_receipt_source_mismatch"):
            module._ready_receipts(ready, journal)

    def test_duplicate_receipt_and_invalid_hash_rejected(self):
        for failure in ("duplicate", "digest"):
            with self.subTest(failure=failure):
                ready, journal = self.ready_receipts(module.SOURCE, 33)
                if failure == "duplicate":
                    journal["installed"][-1] = copy.deepcopy(journal["installed"][0])
                else:
                    journal["installed"][-1]["sha256"] = "invalid"
                with self.assertRaises(module.ProvisionError):
                    module._ready_receipts(ready, journal)

    def test_creation_source_preserved_with_current_source_36(self):
        source = "e78687528d5725efd74e95c858d389f4437099ca"
        ready, journal = self.ready_receipts(source, 36)
        ready["commit"], ready["current_source_ref"] = module.SOURCE, source
        module._ready_receipts(ready, journal)

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
        clone = module.Clone("docker", module.CONTAINER, module.STATE)
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
