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
    def test_other_clone_rejected_before_any_process(self):
        with patch.object(module, "_run") as run:
            result = module.provision(Path("/irrelevant"), "principal", module.STATE,
                                      module.STATE / "material", 55531)
        self.assertEqual(result["blockers"], ["target_outside_authorized_clone"])
        run.assert_not_called()

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
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = root / "contract.go"
            path.write_text("changed")
            with patch.object(module, "CONTRACT_HASHES", {"contract.go": "unchanged digest"}):
                with self.assertRaisesRegex(module.ProvisionError, "source_contract_changed"):
                    module._source_contracts(root)


if __name__ == "__main__":
    unittest.main()
