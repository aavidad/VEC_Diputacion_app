"""Pruebas focales de la huella de cada SQL ensayado y confirmado."""

from __future__ import annotations

import hashlib
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import instalar_esquema as esquema


class EnsayoMismaFuenteTest(unittest.TestCase):
    def test_cuerpo_y_huella_proceden_de_una_sola_lectura(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            migration = source / "m" / "001.up.sql"
            migration.parent.mkdir()
            old = b"BEGIN;\nSELECT 'primera';\nCOMMIT;\n"
            new = b"BEGIN;\nSELECT 'segunda';\nCOMMIT;\n"
            migration.write_bytes(old)
            original = Path.read_bytes
            reads = 0

            def changed_after_first(path: Path) -> bytes:
                nonlocal reads
                if path == migration:
                    reads += 1
                    return old if reads == 1 else new
                return original(path)

            with patch.object(esquema, "SOURCE", source), \
                 patch.object(esquema, "DELTAS", ("m/001.up.sql",)), \
                 patch.object(Path, "read_bytes", changed_after_first):
                sql, hashes = esquema.build("ROLLBACK")

            self.assertEqual(reads, 1)
            self.assertIn("SELECT 'primera';", sql)
            self.assertNotIn("SELECT 'segunda';", sql)
            self.assertEqual(hashes["m/001.up.sql"], hashlib.sha256(old).hexdigest())

    def test_mutacion_durante_ensayo_impide_commit(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            migration = source / "m" / "001.up.sql"
            migration.parent.mkdir()
            migration.write_text("BEGIN;\nSELECT 'primera';\nCOMMIT;\n")
            calls: list[str] = []

            def execute(sql: str) -> None:
                calls.append(sql)
                migration.write_text("BEGIN;\nSELECT 'segunda';\nCOMMIT;\n")

            with patch.object(esquema, "SOURCE", source), \
                 patch.object(esquema, "DELTAS", ("m/001.up.sql",)):
                with self.assertRaisesRegex(RuntimeError, "cambió entre ensayo"):
                    esquema.ejecutar_con_ensayo(execute)

            self.assertEqual(len(calls), 1)
            self.assertIn("SELECT 'primera';", calls[0])
            self.assertTrue(calls[0].rstrip().endswith("ROLLBACK;"))


if __name__ == "__main__":
    unittest.main()
