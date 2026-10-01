"""Repos Git sintéticos: integridad, cierre de rutas y ausencia de ejecución."""
from dataclasses import replace
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import clon_postmain_plan as planner


class PlanTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="postmain-fixture-")
        self.addCleanup(self.temporary.cleanup)
        self.repo = Path(self.temporary.name) / "repo"
        self.repo.mkdir()
        self.git("init", "-q")
        self.git("config", "user.name", "Synthetic")
        self.git("config", "user.email", "synthetic@example.invalid")
        self.installed = "deploy/postgresql/contratacion_temporal/migraciones/000153_installed.up.sql"
        self.write(self.installed, b"-- installed synthetic SQL\n")
        for path in planner.SQL_PATHS[:4]:
            self.write(path, ("-- original " + path + "\n").encode())
        self.companion = planner.CAT + "migraciones/000001_autoridad_categorias.down.sql"
        self.write(self.companion, b"-- synthetic old DOWN\n")
        self.source = self.commit()
        self.source_patch = patch.object(planner, "SOURCE", self.source)
        self.source_patch.start()
        self.addCleanup(self.source_patch.stop)
        for path in planner.SQL_PATHS[1:]:
            if path != planner.SQL_PATHS[3]:
                self.write(path, ("-- target " + path + "\n").encode())
        self.write(self.companion, b"-- synthetic target DOWN\n")
        self.write(planner.LIST, self.list_bytes())
        self.write(planner.LIST_B, self.list_bytes(planner.B_SQL_PATHS))
        self.write(planner.LIST_DEFERRED, self.list_bytes(planner.DEFERRED_SQL[:1]))
        for path in planner.B_SQL_PATHS + planner.B_COMPANIONS + planner.DEFERRED_SQL:
            self.write(path, ("-- synthetic target " + path + "\n").encode())
        self.target = self.commit()
        self.point_origin()

    def git(self, *args):
        result = subprocess.run(["/usr/bin/git", "-C", str(self.repo), *args],
            check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            stdin=subprocess.DEVNULL, timeout=10, env=planner.GIT_ENV)
        return result.stdout

    def write(self, path, data):
        target = self.repo / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)

    def list_bytes(self, paths=None):
        return ("# Lista causal sintética: después de H6 y AD132\n" + "\n".join(
            planner.SQL_PATHS if paths is None else paths) + "\n").encode()

    def commit(self):
        self.git("add", ".")
        self.git("commit", "-q", "-m", "synthetic")
        return self.git("rev-parse", "HEAD").decode().strip()

    def point_origin(self):
        self.git("update-ref", "refs/remotes/origin/main", self.target)

    def pin(self, path):
        return hashlib.sha256(self.git("show", self.target + ":" + path)).hexdigest()

    def request(self):
        return planner.Request(self.repo, self.target,
            self.pin(planner.LIST), tuple(self.pin(path) for path in planner.SQL_PATHS),
            self.pin(planner.LIST_B), tuple(self.pin(path) for path in planner.B_SQL_PATHS),
            tuple(self.pin(path) for path in planner.B_COMPANIONS), self.pin(planner.LIST_DEFERRED),
            tuple(self.pin(path) for path in planner.DEFERRED_SQL))

    def test_plan_uses_commit_bytes_and_classifies_modified_and_companions(self):
        request = self.request()
        self.write(planner.LIST, b"cwd is untrusted and ignored\n")
        self.write(planner.LIST_B, b"cwd second list is also ignored\n")
        self.write(planner.SQL_PATHS[1], b"cwd SQL must be ignored\n")
        value = planner.build_plan(request)
        self.assertEqual(value["plan"], "pending_approval")
        self.assertIs(value["executable"], False)
        self.assertFalse(value["sql_invoked"])
        self.assertEqual(value["blockers"], [])
        self.assertEqual(value["target"], {"commit": self.target,
            "tree": self.git("rev-parse", self.target + "^{tree}").decode().strip()})
        self.assertEqual([item["path"] for item in value["operations"]],
                         list(planner.SQL_PATHS + planner.B_SQL_PATHS))
        self.assertEqual([item["path"] for item in value["causal_lists"]], [planner.LIST, planner.LIST_B])
        self.assertEqual([item["sha256"] for item in value["causal_lists"]],
                         [request.expected_list_sha256, request.expected_b_list_sha256])
        self.assertEqual([item["path"] for item in value["companions"]], list(planner.B_COMPANIONS))
        self.assertTrue(all(item["linked_ups"] == list(planner.B_SQL_PATHS[4:]) and
                            item["executable"] is False for item in value["companions"]))
        changes = {item["path"]: item for item in value["sql_diff"]}
        self.assertEqual(changes[planner.SQL_PATHS[1]]["status"], "modified")
        self.assertEqual(changes[planner.SQL_PATHS[4]]["status"], "added")
        companion = changes[self.companion]
        self.assertEqual(companion["classification"], "companion_non_executable")
        self.assertEqual(companion["linked_up"], planner.SQL_PATHS[1])
        self.assertEqual(companion["after"]["sha256"], hashlib.sha256(
            b"-- synthetic target DOWN\n").hexdigest())
        self.assertEqual(value["receipt_requirements"]["status"], "not_read_not_validated")
        self.assertIn("h6_live_validation_before_transition",
                      value["receipt_requirements"]["future_gates"])
        self.assertEqual(planner.canonical(json.loads(planner.canonical(value))),
                         planner.canonical(value))

    def test_unknown_up_and_unlinked_test_sql_block(self):
        for path in ("deploy/postgresql/other/000999_new.up.sql",
                     planner.CAT + "pruebas_sql/000999_unlinked.sql"):
            self.write(path, b"-- unapproved synthetic SQL\n")
        self.target = self.commit()
        self.point_origin()
        value = planner.build_plan(self.request())
        self.assertEqual(len(value["blockers"]), 2)
        self.assertTrue(all(item["code"] == "unknown_sql_path" for item in value["blockers"]))
        self.assertEqual(value["plan"], "pending_approval")

    def test_installed_sql_modified_blocks(self):
        self.write(self.installed, b"-- installation cannot be rewritten\n")
        self.target = self.commit()
        self.point_origin()
        value = planner.build_plan(self.request())
        self.assertIn({"code": "previous_sql_modified", "path": self.installed}, value["blockers"])
        item = next(change for change in value["sql_diff"] if change["path"] == self.installed)
        self.assertNotEqual(item["before"]["sha256"], item["after"]["sha256"])

    def test_deleted_known_companion_blocks(self):
        (self.repo / self.companion).unlink()
        self.target = self.commit()
        self.point_origin()
        value = planner.build_plan(self.request())
        self.assertIn({"code": "deleted_or_type_changed_sql", "path": self.companion}, value["blockers"])
        item = next(change for change in value["sql_diff"] if change["path"] == self.companion)
        self.assertEqual(item["status"], "deleted")
        self.assertIsNone(item["after"])

    def test_sql_gitlinks_added_modified_deleted_cannot_be_hidden_by_config(self):
        paths = {"added": "deploy/postgresql/gitlink_added.sql",
                 "modified": "deploy/postgresql/gitlink_modified.sql",
                 "deleted": "deploy/postgresql/gitlink_deleted.sql"}
        self.git("read-tree", self.source)
        for status in ("modified", "deleted"):
            self.git("update-index", "--add", "--cacheinfo", "160000",
                     self.source, paths[status])
        tree = self.git("write-tree").decode().strip()
        linked_source = self.git("commit-tree", tree, "-p", self.source,
                                 "-m", "synthetic gitlink preimage").decode().strip()
        self.git("read-tree", self.target)
        self.git("update-index", "--add", "--cacheinfo", "160000", self.source, paths["added"])
        self.git("update-index", "--add", "--cacheinfo", "160000", linked_source, paths["modified"])
        tree = self.git("write-tree").decode().strip()
        self.target = self.git("commit-tree", tree, "-p", linked_source,
                               "-m", "synthetic gitlink postimage").decode().strip()
        self.point_origin()
        self.git("config", "diff.ignoreSubmodules", "all")
        hidden = self.git("diff", "--name-status", linked_source, self.target, "--", "*.sql")
        self.assertTrue(all(path.encode() not in hidden for path in paths.values()))
        with patch.object(planner, "SOURCE", linked_source):
            value = planner.build_plan(self.request())
        changes = {item["path"]: item for item in value["sql_diff"]}
        for status, path in paths.items():
            with self.subTest(status=status):
                self.assertEqual(changes[path]["status"], status)
                self.assertIn({"code": "non_regular_sql_blob", "path": path}, value["blockers"])
                self.assertIn({"code": "unknown_sql_path", "path": path}, value["blockers"])
        self.assertIn({"code": "deleted_or_type_changed_sql", "path": paths["deleted"]},
                      value["blockers"])
        self.assertEqual(value["plan"], "pending_approval")
        self.assertIs(value["executable"], False)

    def test_reordered_list_blocks(self):
        self.write(planner.LIST, self.list_bytes(tuple(reversed(planner.SQL_PATHS))))
        self.target = self.commit()
        self.point_origin()
        with self.assertRaisesRegex(planner.Refused, "causal_order_mismatch"):
            planner.build_plan(self.request())

    def test_second_causal_list_order_traversal_and_duplicate_block(self):
        for paths in (tuple(reversed(planner.B_SQL_PATHS)),
                      ("../escape.up.sql",) + planner.B_SQL_PATHS[1:],
                      planner.B_SQL_PATHS[:-1] + planner.B_SQL_PATHS[:1]):
            with self.subTest(paths=paths):
                self.write(planner.LIST_B, self.list_bytes(paths))
                self.target = self.commit()
                self.point_origin()
                with self.assertRaises(planner.Refused):
                    planner.build_plan(self.request())

    def test_u17_deferred_with_absent_u15_u16_never_becomes_operation(self):
        value = planner.build_plan(self.request())
        self.assertEqual([item["path"] for item in value["deferred"]], list(planner.DEFERRED_SQL))
        self.assertTrue(all(item["reason"] == "usuarios_15_16_absent_from_target" and
                            item["classification"] == "deferred" and item["executable"] is False
                            for item in value["deferred"]))
        self.assertTrue(all(item["absent_from_target"] for item in value["deferred"][0]["dependencies"]))
        self.assertFalse(set(planner.DEFERRED_SQL) & {item["path"] for item in value["operations"]})
        self.assertTrue(all(item["classification"] == "deferred" for item in value["sql_diff"]
                            if item["path"] in planner.DEFERRED_SQL))
        self.assertEqual(value["blockers"], [])

    def test_unapproved_u15_u16_presence_does_not_promote_u17(self):
        for number in ("000015", "000016"):
            self.write(planner.USERS + "migraciones/" + number + "_unapproved.up.sql", b"-- synthetic\n")
        self.target = self.commit()
        self.point_origin()
        value = planner.build_plan(self.request())
        self.assertTrue(all(item["classification"] == "deferred" and item["executable"] is False
                            for item in value["deferred"]))
        self.assertFalse(any(item["absent_from_target"] for item in value["deferred"][0]["dependencies"]))
        self.assertTrue(value["blockers"])
        self.assertEqual(value["plan"], "pending_approval")

    def test_main_non_sql_change_updates_commit_tree_keeps_sql_inventory(self):
        before = planner.build_plan(self.request())
        self.write("notes.txt", b"-- synthetic code-only main change\n")
        self.target = self.commit()
        self.point_origin()
        after = planner.build_plan(self.request())
        self.assertNotEqual(before["target"], after["target"])
        for key in ("causal_lists", "operations", "companions", "deferred", "sql_diff", "blockers"):
            self.assertEqual(before[key], after[key])
        self.assertEqual(after["plan"], "pending_approval")
        self.assertIs(after["executable"], False)

    def test_path_traversal_and_duplicate_list_block(self):
        for replacement in ("../escape.up.sql", "/absolute.sql", "a/../b.sql",
                            planner.SQL_PATHS[0]):
            with self.subTest(replacement=replacement):
                paths = list(planner.SQL_PATHS)
                paths[1] = replacement
                self.write(planner.LIST, self.list_bytes(paths))
                self.target = self.commit()
                self.point_origin()
                with self.assertRaises(planner.Refused):
                    planner.build_plan(self.request())

    def test_sha_must_name_commit_and_origin_main_exactly(self):
        request = self.request()
        bad_values = ["HEAD", self.target[:8], "-bad", "f" * 40,
                      self.git("rev-parse", self.target + "^{tree}").decode().strip(),
                      self.git("rev-parse", self.target + ":" + planner.SQL_PATHS[0]).decode().strip(),
                      self.source]
        for value in bad_values:
            with self.subTest(value=value), self.assertRaises(planner.Refused):
                planner.build_plan(replace(request, target_commit=value))

    def test_altered_external_hashes_block(self):
        request = self.request()
        for changed in (replace(request, expected_list_sha256="0" * 64),
                        replace(request, expected_sql_sha256=("0" * 64,) + request.expected_sql_sha256[1:]),
                        replace(request, expected_b_list_sha256="0" * 64),
                        replace(request, expected_b_sql_sha256=("0" * 64,) + request.expected_b_sql_sha256[1:]),
                        replace(request, expected_b_companion_sha256=("0" * 64,) + request.expected_b_companion_sha256[1:]),
                        replace(request, expected_deferred_list_sha256="0" * 64),
                        replace(request, expected_deferred_sql_sha256=("0" * 64,) + request.expected_deferred_sql_sha256[1:])):
            with self.assertRaisesRegex(planner.Refused, "expected_sha_mismatch"):
                planner.build_plan(changed)

    def test_show_bytes_must_match_git_blob(self):
        request = self.request()
        original = planner._git

        def changed(repo, *args):
            data = original(repo, *args)
            if args == ("show", self.target + ":" + planner.SQL_PATHS[1]):
                return bytes([data[0] ^ 1]) + data[1:]
            return data

        with patch.object(planner, "_git", side_effect=changed), self.assertRaisesRegex(
                planner.Refused, "original_bytes_mismatch"):
            planner.build_plan(request)

    def test_unknown_symlink_sql_blocks_without_following_it(self):
        path = self.repo / "deploy/postgresql/unknown.sql"
        path.symlink_to("/private-receipt-must-never-be-read")
        self.target = self.commit()
        self.point_origin()
        value = planner.build_plan(self.request())
        self.assertIn({"code": "non_regular_sql_blob", "path": "deploy/postgresql/unknown.sql"},
                      value["blockers"])

    def test_no_network_callbacks_services_or_receipt_reads(self):
        request = self.request()
        original = subprocess.run
        calls = []

        def record(command, **kwargs):
            calls.append((command, kwargs))
            self.assertEqual(command[0], "/usr/bin/git")
            self.assertEqual(kwargs["env"], planner.GIT_ENV)
            self.assertIn("protocol.allow=never", command)
            self.assertNotIn("fetch", command)
            self.assertFalse(kwargs.get("shell", False))
            return original(command, **kwargs)

        with patch.object(planner.subprocess, "run", side_effect=record), patch.object(
                Path, "read_bytes", side_effect=AssertionError("no receipt/filesystem reads")):
            value = planner.build_plan(request)
        self.assertTrue(calls)
        self.assertEqual(value["blockers"], [])
        self.assertEqual(planner.GIT_ENV["GIT_NO_LAZY_FETCH"], "1")

    def test_invalid_typed_inputs_block_before_git(self):
        request = self.request()
        for changed in (replace(request, repo=str(self.repo)),
                        replace(request, repo=Path("../repo")),
                        replace(request, expected_sql_sha256=list(request.expected_sql_sha256)),
                        replace(request, expected_sql_sha256=("0" * 64,)),
                        replace(request, expected_b_sql_sha256=("0" * 64,)),
                        replace(request, expected_b_companion_sha256=("0" * 64,)),
                        replace(request, expected_deferred_sql_sha256=("0" * 64,)),
                        replace(request, expected_b_list_sha256=None),
                        replace(request, expected_deferred_list_sha256=None),
                        replace(request, expected_list_sha256=None), {}):
            with self.subTest(changed=changed), patch.object(planner, "_git") as git:
                with self.assertRaises(planner.Refused):
                    planner.build_plan(changed)
                git.assert_not_called()


if __name__ == "__main__":
    unittest.main()
