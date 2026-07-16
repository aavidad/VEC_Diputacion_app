"""Pruebas de privacidad basicas del importador RPT."""

from __future__ import annotations

import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


RAIZ = Path(__file__).resolve().parents[2]
SCRIPT = RAIZ / "scripts" / "import_rpt_pdf.py"
ESPECIFICACION = importlib.util.spec_from_file_location("import_rpt_pdf", SCRIPT)
assert ESPECIFICACION is not None and ESPECIFICACION.loader is not None
IMPORTADOR = importlib.util.module_from_spec(ESPECIFICACION)
ESPECIFICACION.loader.exec_module(IMPORTADOR)


class PrivacidadImportadorRPTTests(unittest.TestCase):
    def test_payload_no_conserva_la_ruta_del_pdf(self) -> None:
        posiciones = [{"official_code": "1", "name": "Puesto de prueba"}]

        payload = IMPORTADOR.build_import(posiciones)
        serializado = json.dumps(payload, ensure_ascii=False)

        self.assertEqual(payload["source"], IMPORTADOR.SOURCE_URL)
        self.assertEqual(payload["positions"][0]["source"], IMPORTADOR.SOURCE_URL)
        self.assertNotIn("/home/", serializado)
        self.assertNotIn("\\\\", serializado)

    def test_cli_exige_indicar_el_pdf(self) -> None:
        with tempfile.TemporaryDirectory() as directorio:
            salida = Path(directorio) / "salida.json"
            proceso = subprocess.run(
                [sys.executable, str(SCRIPT), "--out", str(salida)],
                cwd=RAIZ,
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertFalse(salida.exists())

        self.assertEqual(proceso.returncode, 2)
        self.assertIn("--pdf", proceso.stderr)

    def test_fallo_de_lectura_no_revela_la_ruta_del_pdf(self) -> None:
        ruta_privada = "/home/usuario_prueba/ruta_privada_inexistente.pdf"
        with tempfile.TemporaryDirectory() as directorio:
            salida = Path(directorio) / "salida.json"
            proceso = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPT),
                    "--pdf",
                    ruta_privada,
                    "--out",
                    str(salida),
                ],
                cwd=RAIZ,
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertFalse(salida.exists())

        diagnostico = proceso.stdout + proceso.stderr
        self.assertEqual(proceso.returncode, 2)
        self.assertIn("no se pudo leer el PDF indicado", diagnostico)
        self.assertNotIn(ruta_privada, diagnostico)
        self.assertNotIn("/home/usuario_prueba", diagnostico)
        self.assertNotIn("Traceback", diagnostico)


if __name__ == "__main__":
    unittest.main()
