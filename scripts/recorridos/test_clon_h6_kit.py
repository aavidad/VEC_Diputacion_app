"""Guardas puras del proveedor H6; no prueba PostgreSQL ni RESTORE real."""

from dataclasses import replace
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

try:
    from . import clon_h6_kit as kit
except ImportError:
    import clon_h6_kit as kit


class PureReadOnly(kit.clon_sql.ReadOnlyDB):
    """Doble puro: jamás se conecta a un motor ni acredita sus garantías."""
    def __init__(self, receipt):
        self.receipt = dict(receipt)
        self.calls = []

    def system_identity(self):
        self.calls.append("identity")
        return {key: self.receipt[key] for key in kit.IDENTITY_FIELDS}

    def schema_digest(self, path, digest):
        self.calls.append("schema")
        return self.receipt["schema_sha"]

    def roles_digest(self, path, digest):
        self.calls.append("roles")
        return self.receipt["roles_sha"]

    def database_acl_digest(self):
        self.calls.append("acl")
        return self.receipt["datacl_sha"]


class KitGuards(unittest.TestCase):
    def setUp(self):
        scratch = Path(tempfile.gettempdir())
        if scratch.stat().st_mode & 0o022:
            runtime = Path(f"/run/user/{os.getuid()}")
            scratch = runtime if runtime.is_dir() else Path.home()
        self.directory = tempfile.TemporaryDirectory(dir=scratch)
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        os.chmod(self.root, 0o700)
        self.receipt = {"version": 1, "kind": "h1_restore_confirmed",
            "estado_h1_sha": "1" * 64, "system_identifier": "123456789",
            "database_oid": 5, "database_name": "postgres",
            "pg_container_id": "2" * 64, "pg_image": "postgres:18.4",
            "pg_image_id": "sha256:" + "f" * 64,
            "pg_volume": "/dev/shm/vec-recorridos-pure",
            "schema_sha": "3" * 64, "roles_sha": "4" * 64, "datacl_sha": "5" * 64}
        self.path = self.root / "h1-restore.json"
        self.path.write_bytes(kit.canonical(self.receipt))
        os.chmod(self.path, 0o600)
        normalizer = self.root / "h6_normalizar_pg_dump.py"
        normalizer.write_bytes(b"# pure fixture, never executed\n")
        normalizer_sha = kit.clon_sql.sha(normalizer.read_bytes())
        scripts = self.root / "h6-guiones.sha256"
        scripts.write_text(normalizer_sha + "  h6_normalizar_pg_dump.py\n")
        scripts_sha = kit.clon_sql.sha(scripts.read_bytes())
        lock = self.root / "release.lock"
        lock.write_text("KIT_GUIONES_SHA256 " + scripts_sha + "\n")
        lock_sha = kit.clon_sql.sha(lock.read_bytes())
        self.request = kit.Request(self.root / "package.tgz", self.root / "release.lock",
            "6" * 64, lock_sha, self.root / "h1.dump", "1" * 64,
            self.root, self.path, kit.clon_sql.sha(self.path.read_bytes()),
            normalizer, normalizer_sha, scripts, scripts_sha)
        self.entries = [{"phase": "H3" if n < 8 else "H4" if n < 17 else "H6",
            "path": f"deploy/postgresql/pure/{n:06d}.up.sql", "sha256": "8" * 64}
            for n in range(62)]
        self.plan = {"plan_family": kit.clon_sql.H6_PACKAGE_FAMILY,
            "source_ref": kit.clon_sql.H6_FIRMA_FINAL_REF,
            "approved_sql_ref": kit.clon_sql.H6_FIRMA_FINAL_REF,
            "entries": self.entries, "file_count": 62,
            "plan_sha": kit.clon_sql.plan_hash(self.entries), "inventory_sha": "9" * 64,
            "package_sha": "6" * 64, "lock_sha": lock_sha,
            "list_sha": "a" * 64, "release_sha": "b" * 64, "estado_h1_sha": "1" * 64}
        self.context = {key: self.plan[key] for key in (*kit.clon_sql.CONTEXT_KEYS[1:], "lock_sha")}
        self.context["identidad_clon"] = self.request.approved_restore_receipt_sha256
        self.provider = kit.H6Kit(self.request)
        self.db = PureReadOnly(self.receipt)

    def validate(self):
        with patch.object(kit.clon_sql, "preflight_h6_package", return_value=(self.plan, [])):
            self.assertIs(self.provider.validate(self.plan, self.context), True)

    def record(self):
        return {"version": 2, **self.context, "pending": None, "revisions": [],
            "source_commit": self.plan["source_ref"],
            "approved_sql_ref": self.plan["approved_sql_ref"],
            "plan_family": self.plan["plan_family"], "plan_sha": self.plan["plan_sha"],
            "inventory_sha": self.plan["inventory_sha"], "entries": self.entries,
            "file_count": 62, "phase": "ad132_confirmed",
            "installed": [{**entry, "position": n, "confirmed_at": "2026-09-30T10:00:00+00:00",
                "confirmation": "psql_exit0_observed"} for n, entry in enumerate(self.entries, 1)]}

    def test_fresh_preimage_guards_only(self):
        self.validate()
        self.assertEqual(self.provider.identity(self.db, self.context), self.context["identidad_clon"])
        self.assertIs(self.provider.confirm(self.db, None, 0, self.context), True)
        self.assertEqual(self.db.calls, ["identity", "identity", "schema", "roles", "acl"])

    def test_changed_live_identity_and_each_preimage_denied(self):
        self.validate()
        for field in ("pg_container_id", "pg_image_id", "schema_sha", "roles_sha", "datacl_sha"):
            with self.subTest(field=field):
                db = PureReadOnly({**self.receipt, field: "c" * 64})
                with self.assertRaises(kit.Refused):
                    self.provider.confirm(db, None, 0, self.context)

    def test_every_resume_denied_before_any_query(self):
        self.validate()
        for completed, entry in ((1, self.entries[0]), (62, self.entries[-1]),
                                 (True, None), (0, self.entries[0]), (-1, None)):
            with self.subTest(completed=completed), self.assertRaises(kit.Refused):
                self.provider.confirm(self.db, entry, completed, self.context)
        self.assertEqual(self.db.calls, [])

    def test_preflight_and_readonly_boundary_required(self):
        with self.assertRaises(kit.Refused):
            self.provider.identity(self.db, self.context)
        self.validate()
        with self.assertRaises(kit.Refused):
            self.provider.identity(object(), self.context)
        with self.assertRaises(kit.Refused):
            self.provider.identity(self.db, {**self.context, "lock_sha": "d" * 64})

    def test_changed_external_lock_and_plan_denied(self):
        for plan, context in (({**self.plan, "file_count": 45}, self.context),
                              (self.plan, {**self.context, "lock_sha": "d" * 64})):
            with patch.object(kit.clon_sql, "preflight_h6_package", return_value=(self.plan, [])):
                with self.assertRaises(kit.Refused):
                    self.provider.validate(plan, context)
        self.assertIsNone(self.provider._plan)

    def test_normalizer_and_guiones_remain_externally_pinned(self):
        self.request.normalizer_path.write_bytes(b"changed\n")
        with patch.object(kit.clon_sql, "preflight_h6_package", return_value=(self.plan, [])):
            with self.assertRaises(kit.Refused):
                self.provider.validate(self.plan, self.context)
        self.request.normalizer_path.write_bytes(b"# pure fixture, never executed\n")
        self.request.guiones_manifest.write_bytes(b"changed\n")
        with patch.object(kit.clon_sql, "preflight_h6_package", return_value=(self.plan, [])):
            with self.assertRaises(kit.Refused):
                self.provider.validate(self.plan, self.context)

    def test_restore_receipt_not_free_approval_json(self):
        for value in ({**self.receipt, "approved": True},
                      {**self.receipt, "version": True},
                      {**self.receipt, "estado_h1_sha": "d" * 64},
                      {**self.receipt, "pg_volume": "/dev/shm/../arbitrary"}):
            self.path.write_bytes(kit.canonical(value))
            request = replace(self.request, approved_restore_receipt_sha256=kit.clon_sql.sha(self.path.read_bytes()))
            with self.assertRaises(kit.Refused):
                kit.restore_receipt(request)

    def test_restore_receipt_changed_mode_symlink_and_hash_denied(self):
        self.path.write_bytes(b"{}\n")
        with self.assertRaises(kit.Refused):
            kit.restore_receipt(self.request)
        self.path.write_bytes(kit.canonical(self.receipt))
        os.chmod(self.path, 0o644)
        with self.assertRaises(kit.Refused):
            kit.restore_receipt(self.request)
        os.chmod(self.path, 0o600)
        alias = self.root / "alias.json"
        alias.symlink_to(self.path)
        with self.assertRaises(kit.Refused):
            kit.restore_receipt(replace(self.request, restore_receipt=alias))

    def test_receipt_replacement_and_mode_change_during_fd_read_denied(self):
        real_read = os.read
        for operation in ("replace", "chmod"):
            with self.subTest(operation=operation):
                self.path.write_bytes(kit.canonical(self.receipt))
                os.chmod(self.path, 0o600)
                changed = False

                def read_then_change(fd, limit):
                    nonlocal changed
                    data = real_read(fd, limit)
                    if data and not changed:
                        changed = True
                        if operation == "replace":
                            replacement = self.root / "replacement"
                            replacement.write_bytes(data)
                            os.chmod(replacement, 0o600)
                            os.replace(replacement, self.path)
                        else:
                            os.chmod(self.path, 0o644)
                    return data

                with patch.object(kit.os, "read", side_effect=read_then_change):
                    with self.assertRaises(kit.Refused):
                        kit.restore_receipt(self.request)
                self.assertTrue(changed)

    def test_untrusted_ancestor_rejected_before_receipt_bytes_read(self):
        os.chmod(self.root, 0o770)
        self.addCleanup(os.chmod, self.root, 0o700)
        with patch.object(kit.os, "read") as read:
            with self.assertRaisesRegex(kit.Refused, "ancestro"):
                kit.restore_receipt(self.request)
            read.assert_not_called()

    def test_final_requires_complete_journal_and_original_ad132_request(self):
        self.validate()
        record = self.record()
        with self.assertRaisesRegex(kit.Refused, "Request"):
            self.provider.verify(self.db, record, self.context)
        for change in ({"pending": {"position": 62}}, {"phase": "awaiting_ad132"},
                       {"installed": record["installed"][:-1]}):
            with self.subTest(change=change), self.assertRaises(kit.Refused):
                self.provider.verify(self.db, {**record, **change}, self.context)

    def test_no_callback_or_generic_request(self):
        with self.assertRaises(kit.Refused):
            kit.H6Kit({"approved": True})
        with self.assertRaises(kit.Refused):
            kit.H6Kit(lambda: True)
        with self.assertRaises(kit.Refused):
            kit.H6Kit(replace(self.request, ad132_request={"approved": True}))._bind_ad132(
                {"approved": True}, self.plan, self.receipt)

    def test_final_calls_original_revalidation_and_binds_its_receipt(self):
        request = kit.clon_ad132_recibo.Request(
            package_tar=self.request.package_tar, package_root=self.root / "package",
            approved_package_sha256=self.plan["package_sha"],
            release_lock=self.request.release_lock, approved_lock_sha256=self.plan["lock_sha"],
            source_commit=self.plan["source_ref"], approved_release_sha256=self.plan["release_sha"],
            plan=self.root / "plan.json", plan_receipt=self.root / "plan-receipt.json",
            approved_plan_sha256="c" * 64, approval=self.root / "approval.json",
            approval_sha256="d" * 64, container=self.receipt["pg_container_id"],
            receipt=self.root / "ad132.json", apply_output=self.root / "stdout",
            pending_path=self.root / "pending")
        self.provider = kit.H6Kit(replace(self.request, ad132_request=request))
        self.validate()
        confirmed = {"kind": "ad132_apply_confirmed", "package_sha256": self.plan["package_sha"],
            "lock_sha256": self.plan["lock_sha"], "release_sha256": self.plan["release_sha"],
            "source_commit": self.plan["source_ref"], "pg_container_id": self.receipt["pg_container_id"]}
        # Este doble comprueba delegación/guardas, nunca aplicación AD132 real.
        with patch.object(kit.clon_ad132_recibo, "revalidar", return_value=confirmed) as validate:
            self.assertIs(self.provider.verify(self.db, self.record(), self.context), True)
            validate.assert_called_once_with(request)
        with patch.object(kit.clon_ad132_recibo, "revalidar", side_effect=RuntimeError("pure failure")):
            with self.assertRaises(kit.Refused):
                self.provider.verify(self.db, self.record(), self.context)
        with patch.object(kit.clon_ad132_recibo, "revalidar",
                          return_value={**confirmed, "pg_container_id": "e" * 64}):
            with self.assertRaises(kit.Refused):
                self.provider.verify(self.db, self.record(), self.context)
        with self.assertRaises(kit.Refused):
            self.provider._bind_ad132(replace(request, approved_lock_sha256="e" * 64),
                                     self.plan, self.receipt)


if __name__ == "__main__":
    unittest.main()
