import hashlib
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("source", Path(__file__).with_name("fuente_sintetica_candidatos.py"))
source = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(source)
REPO = Path(__file__).resolve().parents[2]


class SyntheticSourceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name) / "private"
        self.outputs = source.prepare(REPO)

    def tearDown(self):
        self.temp.cleanup()

    def test_reproducible_private_source_and_manifest(self):
        self.assertEqual(self.outputs, source.prepare(REPO))
        first = source.write_proposal(self.root, self.outputs)
        inodes = {p.name: p.stat().st_ino for p in self.root.iterdir()}
        self.assertEqual(first, source.write_proposal(self.root, self.outputs))
        self.assertEqual(inodes, {p.name: p.stat().st_ino for p in self.root.iterdir()})
        self.assertEqual(self.root.stat().st_mode & 0o777, 0o700)
        self.assertTrue(all(p.stat().st_mode & 0o777 == 0o600 for p in self.root.iterdir()))
        manifest = json.loads(self.outputs["manifiesto.propuesta-v1.json"])
        for name, entry in manifest["archivos"].items():
            self.assertEqual(entry["sha256"], hashlib.sha256(self.outputs[name]).hexdigest())
            self.assertEqual(entry["bytes"], len(self.outputs[name]))

    def test_shared_person_is_pending_with_no_authority_or_cas(self):
        candidate = json.loads(self.outputs["candidato-fuente.propuesta-v1.json"])
        users = json.loads(self.outputs["usuarios-fuente.propuesta-v1.json"])
        master = json.loads(self.outputs["fuente-sintetica-v1.json"])
        self.assertIsNone(master["acreditacion"])
        self.assertEqual(len(master["personas"]), 1)
        self.assertEqual(len(master["cuentas"]), 1)
        for key in ("cuenta", "persona", "perfil", "contexto"):
            self.assertEqual(candidate["snapshot"][key], users["snapshot"][key])
            self.assertEqual(candidate["snapshot"][key]["procedencia_autoridad"], "propuesta")
        self.assertIsNone(candidate["preimagen"])
        self.assertIsNone(users["preimagen"])
        self.assertIsNone(users["snapshot"]["vinculo_candidato"])
        material = json.loads(self.outputs["bolsa-candidato.propuesta-v1.json"])
        self.assertEqual(set(material), {"version", "autoridad", "certificado", "identidad", "sujeto",
                        "cuenta_ref", "persona_ref", "perfil_ref", "candidato_ref"})
        for key in ("cuenta", "persona", "perfil"):
            self.assertEqual(material[key + "_ref"], candidate["snapshot"][key]["referencia"])
        self.assertEqual(material["candidato_ref"], candidate["snapshot"]["vinculo_candidato"]["candidato_ref"])
        self.assertNotIn(b"autoridad_maestra_acreditada", b"".join(self.outputs.values()))
        self.assertEqual(json.loads(self.outputs["manifiesto.propuesta-v1.json"])["fases_usuarios"], ["identidad"])

    def test_replay_cannot_replace_approved_source_or_emit_missing_files(self):
        source.write_proposal(self.root, self.outputs)
        target = self.root / "fuente-sintetica-v1.json"
        target.write_bytes(b"different")
        missing = self.root / "usuarios-fuente.propuesta-v1.json"
        missing.unlink()
        with self.assertRaisesRegex(source.SourceError, "preimage_changed"):
            source.write_proposal(self.root, self.outputs)
        self.assertEqual(target.read_bytes(), b"different")
        self.assertFalse(missing.exists())

    def test_symlink_and_hardlink_are_rejected(self):
        source.write_proposal(self.root, self.outputs)
        target = self.root / "fuente-sintetica-v1.json"
        preserved = self.root.parent / "preserved"
        target.rename(preserved)
        target.symlink_to(preserved)
        with self.assertRaises(OSError):
            source.write_proposal(self.root, self.outputs)
        target.unlink()
        os.link(preserved, target)
        with self.assertRaisesRegex(source.SourceError, "private_file_invalid"):
            source.write_proposal(self.root, self.outputs)
        self.assertEqual(preserved.read_bytes(), self.outputs[target.name])

    def test_unsafe_root_and_files_are_rejected(self):
        self.root.mkdir(mode=0o755)
        with self.assertRaisesRegex(source.SourceError, "permissions"):
            source.write_proposal(self.root, self.outputs)
        self.root.chmod(0o700)
        source.write_proposal(self.root, self.outputs)
        (self.root / "fuente-sintetica-v1.json").chmod(0o644)
        with self.assertRaisesRegex(source.SourceError, "private_file_invalid"):
            source.write_proposal(self.root, self.outputs)

    def test_git_and_symlinked_roots_are_rejected(self):
        (self.root.parent / ".git").write_text("gitdir: unrelated")
        with self.assertRaisesRegex(source.SourceError, "inside_git"):
            source.write_proposal(self.root, self.outputs)
        (self.root.parent / ".git").unlink()
        real = self.root.parent / "real"
        real.mkdir(mode=0o700)
        self.root.symlink_to(real, target_is_directory=True)
        with self.assertRaisesRegex(source.SourceError, "root_invalid"):
            source.write_proposal(self.root, self.outputs)

    def test_vocabulary_reuse_does_not_execute_generator(self):
        repo = self.root.parent / "repo"
        (repo / "scripts").mkdir(parents=True)
        path = repo / "scripts/generar_bolsa_demo.py"
        path.write_text("NOMBRES = ['Ana'] * 12\nAPELLIDOS = ['Ferrer'] * 12\nraise RuntimeError('must_not_execute')\n")
        with self.assertRaises(ValueError):
            # Only literal lists are admitted; computed expressions are closed.
            source.vocabularies(repo)
        path.write_text("NOMBRES = " + repr(["Ana"] * 12) + "\nAPELLIDOS = " + repr(["Ferrer"] * 12) +
                        "\nraise RuntimeError('must_not_execute')\n")
        self.assertEqual(source.vocabularies(repo), (["Ana"] * 12, ["Ferrer"] * 12))

    def test_ancestor_swap_cannot_redirect_to_git_after_descriptor_open(self):
        ancestor = self.root.parent / "ancestor"
        ancestor.mkdir(mode=0o700)
        root = ancestor / "private"
        preserved = self.root.parent / "preserved"
        git = self.root.parent / "git"
        git.mkdir(mode=0o700)
        (git / ".git").write_text("gitdir: unrelated")
        original_open = source.os.open
        changed = False

        def swap(name, flags, *args, **kwargs):
            nonlocal changed
            if name == "private" and kwargs.get("dir_fd") is not None and not changed:
                ancestor.rename(preserved)
                ancestor.symlink_to(git, target_is_directory=True)
                changed = True
            return original_open(name, flags, *args, **kwargs)

        with patch.object(source.os, "open", side_effect=swap):
            with self.assertRaisesRegex(source.SourceError, "private_root_invalid"):
                source.write_proposal(root, self.outputs)
        self.assertTrue(changed)
        self.assertFalse((git / "private").exists())
        self.assertEqual(list((preserved / "private").iterdir()), [])

    def test_directory_replacement_after_lock_is_rejected_before_publish(self):
        original_flock = source.fcntl.flock
        preserved = self.root.parent / "preserved"

        def swap(fd, operation):
            original_flock(fd, operation)
            self.root.rename(preserved)
            self.root.mkdir(mode=0o700)

        with patch.object(source.fcntl, "flock", side_effect=swap):
            with self.assertRaisesRegex(source.SourceError, "private_root_changed"):
                source.write_proposal(self.root, self.outputs)
        self.assertEqual(list(self.root.iterdir()), [])
        self.assertEqual([p.name for p in preserved.iterdir()], [".fuente.lock"])

    def test_writable_ancestor_is_rejected_before_creating_leaf(self):
        ancestor = self.root.parent / "ancestor"
        ancestor.mkdir(mode=0o700)
        ancestor.chmod(0o777)
        with self.assertRaisesRegex(source.SourceError, "private_root_permissions"):
            source.write_proposal(ancestor / "private", self.outputs)
        self.assertFalse((ancestor / "private").exists())


if __name__ == "__main__":
    unittest.main()
