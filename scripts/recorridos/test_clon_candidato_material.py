import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location("candidate_material", Path(__file__).with_name("clon_candidato_material.py"))
candidate = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(candidate)


class CandidateMaterialTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.state = self.root / "state"
        self.repo = self.root / "repo"
        self.repo.mkdir()
        self.state.mkdir(mode=0o700)
        self.material = self.state / "material"
        self.material.mkdir(mode=0o700)
        for directory in ("identidad", "mtls", "ca"):
            (self.material / directory).mkdir(mode=0o700)
        self.der = b"synthetic-DER"
        self.fingerprint = hashlib.sha256(self.der).hexdigest()
        self.identity = {"version": 1, "autoridad": "no_autoritativo", "subject": candidate.SUBJECT,
                         "certificate_sha256": self.fingerprint, "display_name": "synthetic", "roles": ["candidato_bolsa"]}
        self.write(self.material / "identidad/candidato.json", self.identity)
        for name in ("mtls/candidato.crt", "mtls/candidato.key", "ca/ca.crt"):
            self.write(self.material / name, b"synthetic-PEM")
        self.write(self.state / "DB_READY.json", {"propietario": "Codex-M", "estado": str(self.state),
                   "contenedor": "vec-owned-clone", "puerto_pg": 55531, "commit": "a" * 40})
        self.info = {"Id": "container-owned", "Config": {"Labels": {
            "vec.recorridos.owner": "Codex-M", "vec.recorridos.state": str(self.state)}},
            "State": {"Running": True}, "NetworkSettings": {"Ports": {
            "5432/tcp": [{"HostIp": "127.0.0.1", "HostPort": "55531"}]}}}
        self.calls = []

    def tearDown(self):
        self.temp.cleanup()

    def write(self, path, value):
        path.write_bytes(value if isinstance(value, bytes) else json.dumps(value).encode())
        path.chmod(0o600)

    def fake_run(self, argv, data=None):
        self.calls.append((argv, data))
        if argv[0] == "git":
            return b""
        if argv[:2] == ["docker", "inspect"]:
            return json.dumps([self.info]).encode()
        if argv[:2] == ["docker", "exec"]:
            self.assertIn(b"BEGIN READ ONLY", data)
            self.assertIn(b"ROLLBACK", data)
            for forbidden in (b"INSERT", b"UPDATE", b"DELETE", b"GRANT", b"CREATE", b"COMMIT"):
                self.assertNotIn(forbidden, data)
            return b'{"bolsa_candidate_count":41,"candidate_context_count":3,"context_apis":[]}'
        if argv[:2] == ["openssl", "verify"]:
            return b"OK"
        if "DER" in argv:
            return self.der
        if "subjectAltName" in argv:
            return ("X509v3 Subject Alternative Name:\n    URI:urn:vec:" + candidate.SUBJECT + "\n").encode()
        return b"synthetic-public-key"

    def provision(self):
        with patch.object(candidate, "run", side_effect=self.fake_run):
            return candidate.provision(self.repo, "vec-owned-clone", self.state, self.material, 55531)

    def test_blocked_replay_preserves_certificate_and_no_activation(self):
        cert_before = (self.material / "mtls/candidato.crt").read_bytes()
        first = self.provision()
        second = self.provision()
        self.assertEqual(first, second)
        self.assertEqual(first["env"], {})
        self.assertEqual(first["status"], "blocked")
        self.assertFalse(first["sql_written"])
        self.assertFalse(first["runtime_activated"])
        self.assertFalse(first["candidate_ref_assigned"])
        self.assertNotIn("candidato_ref", first["profiles"]["candidato"])
        self.assertFalse((self.material / "identidad/bolsa-candidato.json").exists())
        self.assertEqual((self.material / "mtls/candidato.crt").read_bytes(), cert_before)
        result = self.state / "candidato-material/result.json"
        self.assertEqual(result.stat().st_mode & 0o777, 0o600)
        self.assertEqual(result.parent.stat().st_mode & 0o777, 0o700)

    def test_existing_external_identity_cannot_be_relabelled(self):
        self.identity["subject"] = "desarrollo:intervencion"
        self.write(self.material / "identidad/candidato.json", self.identity)
        with self.assertRaisesRegex(candidate.CandidateMaterialError, "not_nominal"):
            self.provision()
        self.assertFalse(any(argv[:2] == ["docker", "exec"] for argv, _ in self.calls))

    def test_wrong_certificate_rejected_before_database(self):
        self.identity["certificate_sha256"] = "b" * 64
        self.write(self.material / "identidad/candidato.json", self.identity)
        with self.assertRaisesRegex(candidate.CandidateMaterialError, "certificate_mismatch"):
            self.provision()
        self.assertFalse(any(argv[:2] == ["docker", "exec"] for argv, _ in self.calls))

    def test_foreign_clone_and_nonloopback_refused(self):
        self.info["Config"]["Labels"]["vec.recorridos.owner"] = "another-owner"
        with self.assertRaisesRegex(candidate.CandidateMaterialError, "ownership_mismatch"):
            self.provision()
        self.info["Config"]["Labels"]["vec.recorridos.owner"] = "Codex-M"
        self.info["NetworkSettings"]["Ports"]["5432/tcp"][0]["HostIp"] = "0.0.0.0"
        with self.assertRaisesRegex(candidate.CandidateMaterialError, "port_mismatch"):
            self.provision()

    def test_symlink_and_duplicate_keys_refused(self):
        linked = self.state / "linked.json"
        linked.symlink_to(self.material / "identidad/candidato.json")
        with self.assertRaisesRegex(candidate.CandidateMaterialError, "path_symlink"):
            candidate.read_json(linked)
        self.write(self.material / "identidad/candidato.json", b'{"version":1,"version":1}')
        with self.assertRaisesRegex(candidate.CandidateMaterialError, "duplicate_json_key"):
            self.provision()

    def test_preserved_result_rejects_new_identity(self):
        self.provision()
        path = self.state / "candidato-material/result.json"
        old = candidate.read_json(path)
        old["profiles"]["candidato"]["expected_persona_ref"] = "per_foreign"
        self.write(path, old)
        with self.assertRaisesRegex(candidate.CandidateMaterialError, "preimage_mismatch"):
            self.provision()

    def test_source_descendant_updates_only_metadata_atomically(self):
        first = self.provision()
        path = self.state / "candidato-material/result.json"
        before = path.stat().st_ino
        ready = candidate.read_json(self.state / "DB_READY.json")
        ready["commit"] = "b" * 40
        self.write(self.state / "DB_READY.json", ready)
        second = self.provision()
        self.assertEqual(second["target"]["source_commit"], "b" * 40)
        self.assertEqual(second["profiles"], first["profiles"])
        self.assertEqual(second["env"], {})
        self.assertEqual(candidate.read_json(path), second)
        self.assertEqual(path.stat().st_mode & 0o777, 0o600)
        self.assertNotEqual(path.stat().st_ino, before)
        ancestry = [argv for argv, _ in self.calls if "merge-base" in argv]
        self.assertEqual(ancestry, [["git", "-C", str(self.repo), "merge-base", "--is-ancestor", "a" * 40, "b" * 40]])
        self.assertFalse(list(path.parent.glob(".result-*")))

    def test_unrelated_source_preserves_previous_result(self):
        self.provision()
        path = self.state / "candidato-material/result.json"
        original = path.read_bytes()
        proposed = candidate.read_json(path)
        proposed["target"]["source_commit"] = "c" * 40
        with patch.object(candidate, "run", side_effect=candidate.CandidateMaterialError("subprocess_failed")):
            with self.assertRaisesRegex(candidate.CandidateMaterialError, "subprocess_failed"):
                candidate.write_result(path, proposed, self.repo)
        self.assertEqual(path.read_bytes(), original)

    def test_changed_clone_refused_even_when_source_advances(self):
        self.provision()
        path = self.state / "candidato-material/result.json"
        original = path.read_bytes()
        proposed = candidate.read_json(path)
        proposed["target"]["source_commit"] = "b" * 40
        for field, value in (("owner", "other"), ("container_id", "other"), ("pg_port", 55532)):
            changed = json.loads(json.dumps(proposed))
            changed["target"][field] = value
            with patch.object(candidate, "run", side_effect=self.fake_run):
                with self.assertRaisesRegex(candidate.CandidateMaterialError, "preimage_mismatch"):
                    candidate.write_result(path, changed, self.repo)
        self.assertEqual(path.read_bytes(), original)


if __name__ == "__main__":
    unittest.main()
