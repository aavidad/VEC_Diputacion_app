#!/usr/bin/env python3
"""Pruebas focales sin dependencias del catálogo operativo."""

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

DIRECTORIO = Path(__file__).resolve().parent
CATALOGO = DIRECTORIO / "catalogo_operativo.json"
HERRAMIENTA = DIRECTORIO / "verificar_catalogo_operativo.py"


class CatalogoOperativoTest(unittest.TestCase):
    def ejecutar(self, catalogo: dict, *argumentos: str) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as temporal:
            ruta = Path(temporal) / "catalogo.json"
            ruta.write_text(json.dumps(catalogo), encoding="utf-8")
            return subprocess.run([sys.executable, str(HERRAMIENTA), "--catalogo", str(ruta), *argumentos],
                                  text=True, capture_output=True, check=False)

    def catalogo(self) -> dict:
        return json.loads(CATALOGO.read_text(encoding="utf-8"))

    def test_variable_faltante_no_supera_paridad(self) -> None:
        catalogo = self.catalogo()
        catalogo["variables"].pop()
        resultado = self.ejecutar(catalogo, "--validar")
        self.assertNotEqual(resultado.returncode, 0)
        self.assertIn("CAT005", resultado.stderr)

    def test_variable_duplicada_no_supera_unicidad(self) -> None:
        catalogo = self.catalogo()
        catalogo["variables"].append(catalogo["variables"][0].copy())
        resultado = self.ejecutar(catalogo, "--validar")
        self.assertNotEqual(resultado.returncode, 0)
        self.assertIn("CAT004", resultado.stderr)

    def test_secreto_con_valor_es_rechazado(self) -> None:
        catalogo = self.catalogo()
        secreto = next(v for v in catalogo["variables"] if v["secreto"])
        secreto["valor_predeterminado"] = "no-versionar"
        resultado = self.ejecutar(catalogo, "--validar")
        self.assertNotEqual(resultado.returncode, 0)
        self.assertIn("no puede tener valor predeterminado", resultado.stderr)

    def test_variable_prohibida_no_aparece_en_plantilla(self) -> None:
        resultado = subprocess.run([sys.executable, str(HERRAMIENTA), "--plantilla", "vec-emisor-capacidad-v4"],
                                  text=True, capture_output=True, check=False)
        self.assertEqual(resultado.returncode, 0, resultado.stderr)
        self.assertNotIn("VEC_V4_EJECUTOR_DATABASE_URL=", resultado.stdout)
        self.assertIn("VEC_V4_EMISOR_DATABASE_URL=", resultado.stdout)


if __name__ == "__main__":
    unittest.main()
