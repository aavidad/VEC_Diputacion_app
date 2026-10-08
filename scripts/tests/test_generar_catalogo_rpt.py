"""Pruebas del catálogo candidato derivado del PDF público de la RPT."""

from __future__ import annotations

import importlib.util
import hashlib
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


RAIZ = Path(__file__).resolve().parents[2]
SCRIPT = RAIZ / "scripts" / "generar_catalogo_rpt.py"
ESPECIFICACION = importlib.util.spec_from_file_location("generar_catalogo_rpt", SCRIPT)
assert ESPECIFICACION is not None and ESPECIFICACION.loader is not None
GENERADOR = importlib.util.module_from_spec(ESPECIFICACION)
ESPECIFICACION.loader.exec_module(GENERADOR)


class CatalogoRPTTests(unittest.TestCase):
    def test_alternativas_numeradas_conservan_el_grupo_compuesto_sin_asignarlo(self) -> None:
        self.assertEqual(
            GENERADOR.denominaciones_categoria("37 ENFERMERO/A 73 MÉDICO/ A"),
            ["ENFERMERO/A", "MÉDICO/A"],
        )
        self.assertEqual(
            GENERADOR.denominaciones_categoria(
                "63 INGENIERO/ A DE TELECOMUNICACIONES 64 INGENIERO/ A INDUSTRIAL 66 INGENIERO TÉCNICO/ A"
            ),
            ["INGENIERO/A DE TELECOMUNICACIONES", "INGENIERO/A INDUSTRIAL", "INGENIERO TÉCNICO/A"],
        )

        with tempfile.TemporaryDirectory() as directorio:
            entrada = Path(directorio) / "entrada.json"
            salida = Path(directorio) / "candidato.json"
            entrada.write_text(json.dumps({"version": "rpt-prueba", "positions": [
                {"code": "407-751-001", "name": "RESP. CALIDAD Y GESTION ASISTENCIAL", "dot": 1,
                 "type": "S", "group": "A1/A2", "category_code": "37 ENFERMERO/A 73 MÉDICO/ A",
                 "center_code": "751", "center_name": "CENTRO", "delegation": "DELEGACIÓN"},
                {"code": "638", "name": "ENFERMERO/A", "dot": 2, "type": "N", "group": "A2",
                 "category_code": "ENFERMERO/A", "center_code": "751", "center_name": "CENTRO", "delegation": "DELEGACIÓN"},
                {"code": "347", "name": "MÉDICO/A", "dot": 1, "type": "N", "group": "A1",
                 "category_code": "MÉDICO/ A", "center_code": "751", "center_name": "CENTRO", "delegation": "DELEGACIÓN"},
                {"code": "408", "name": "RESPONSABLE ASISTENCIAL", "dot": 1, "type": "S", "group": "A1/A2",
                 "category_code": "37 ENFERMERO/A 99 DESCONOCIDO/A", "center_code": "751",
                 "center_name": "CENTRO", "delegation": "DELEGACIÓN"},
                {"code": "88", "name": "AUXILIAR", "dot": 1, "type": "N", "group": "",
                 "category_code": "GENERALES", "center_code": "751",
                 "center_name": "CENTRO", "delegation": "DELEGACIÓN"},
            ]}), encoding="utf-8")
            comando = [sys.executable, str(SCRIPT), "--in", str(entrada), "--out", str(salida),
                       "--generated-on", "2026-09-17"]
            subprocess.run(comando, cwd=RAIZ, check=True, capture_output=True)
            primero = salida.read_bytes()
            subprocess.run(comando, cwd=RAIZ, check=True, capture_output=True)
            self.assertEqual(salida.read_bytes(), primero)
            catalogo = json.loads(primero)

        self.assertEqual(catalogo["resumen"]["puestos"], 5)
        self.assertEqual(catalogo["esquema"], GENERADOR.ESQUEMA_CANDIDATO)
        self.assertEqual(catalogo["estado"], "preparacion_no_autoritativa")
        self.assertEqual(catalogo["resumen"]["dotacion"], 6)
        self.assertEqual(catalogo["puestos"][0]["grupos"], ["A1", "A2"])
        self.assertEqual(catalogo["puestos"][0]["categoria_clave"], "")
        self.assertEqual(catalogo["puestos"][0]["categorias_claves"], ["enfermero-a", "medico-a"])
        categorias = {c["clave"]: c for c in catalogo["categorias"]}
        self.assertEqual(categorias["enfermero-a"]["grupos"], ["A2"])
        self.assertEqual(categorias["medico-a"]["grupos"], ["A1"])
        self.assertEqual(categorias["enfermero-a"]["dotacion"], 2)
        self.assertEqual(catalogo["puestos"][3]["categoria_clave"], "")
        self.assertEqual(catalogo["puestos"][3]["categorias_claves"], ["enfermero-a"])
        self.assertEqual(catalogo["puestos"][3]["categorias_pendientes"], [
            {"denominacion": "DESCONOCIDO/A", "origen": "categoria"},
        ])
        self.assertEqual(catalogo["puestos"][4]["categoria_clave"], "")
        self.assertEqual(catalogo["puestos"][4]["categorias_pendientes"], [
            {"denominacion": "GENERALES", "origen": "categoria"},
        ])
        claves_catalogo = set(categorias)
        for puesto in catalogo["puestos"]:
            self.assertTrue(set(puesto["categorias_claves"]) <= claves_catalogo)
            if puesto["categoria_clave"]:
                self.assertIn(puesto["categoria_clave"], claves_catalogo)

    def test_exige_salida_explicita_y_protege_catalogo_publicado(self) -> None:
        publicado = GENERADOR.SALIDA
        huella = hashlib.sha256(publicado.read_bytes()).hexdigest()
        with tempfile.TemporaryDirectory(dir=RAIZ) as directorio:
            entrada = Path(directorio) / "entrada.json"
            entrada.write_text('{"positions":[]}', encoding="utf-8")
            sin_salida = subprocess.run(
                [sys.executable, str(SCRIPT), "--in", str(entrada)],
                cwd=RAIZ, capture_output=True, text=True, check=False,
            )
            self.assertEqual(sin_salida.returncode, 2)
            self.assertIn("--out", sin_salida.stderr)

            for destino in (publicado, Path(directorio) / "enlace.json", Path(directorio) / "duro.json"):
                if destino.name == "enlace.json":
                    os.symlink(publicado, destino)
                elif destino.name == "duro.json":
                    os.link(publicado, destino)
                proceso = subprocess.run(
                    [sys.executable, str(SCRIPT), "--in", str(entrada), "--out", str(destino)],
                    cwd=RAIZ, capture_output=True, text=True, check=False,
                )
                self.assertEqual(proceso.returncode, 2)
                self.assertIn("no puede sobrescribir el catálogo RPT publicado", proceso.stderr)
            self.assertEqual(hashlib.sha256(publicado.read_bytes()).hexdigest(), huella)


if __name__ == "__main__":
    unittest.main()
