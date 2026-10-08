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
    def test_continuaciones_respetan_columnas_del_pdf(self) -> None:
        texto = "\n".join([
            "422 RESPONS. DE RR. HH. DE LA MILAGROSA Y SERVS.   1    S    F   C A2/C1        DG     AG                            22   RTDIP      19.201,98 €    111/SI   NO",
            "    GRALES. DE ARMILLA",
            "412 JEF. NEGOCIADO GESTIÓN DE JORNADA Y           1    S    F   C     C1        DG   AGAE                             21   RTDIP      17.194,38 €     10/SI   NO",
            "    HORARIOS DE TRABAJO",
            "217 INSPECTOR TERRITORIO                        2    N    F   C         B     DG      AE   149 BASE B SIN CATEGORÍA   18   RTDIP      19.125,26 €     9/SI    SI",
            "                                                                                                  ESPECÍFICA",
            "407 RESP. CALIDAD Y GESTION ASISTENCIAL       1    S    F   C A1/A2        DG      AE       37 ENFERMERO/A         25   RTDIP      21.521,08 €     12/SI   NO",
            "                                                                                             73 MÉDICO/ A",
        ])
        puestos = IMPORTADOR.parse_text(texto)
        self.assertEqual(len(puestos), 4)
        self.assertEqual(puestos[0]["name"], "RESPONS. DE RR. HH. DE LA MILAGROSA Y SERVS. GRALES. DE ARMILLA")
        self.assertNotIn("category_code", puestos[0])
        self.assertEqual(puestos[1]["name"], "JEF. NEGOCIADO GESTIÓN DE JORNADA Y HORARIOS DE TRABAJO")
        self.assertEqual(puestos[2]["category_code"], "149 BASE B SIN CATEGORÍA ESPECÍFICA")
        self.assertEqual(puestos[3]["category_code"], "37 ENFERMERO/A 73 MÉDICO/ A")

    def test_una_linea_amplia_nombre_y_categoria_por_separado(self) -> None:
        texto = "\n".join([
            "748 JEFATURA SECCIÓN DE ASISTENCIA TÉCNICA EN         1    S    F   C A1/A2        DG     AE       63 INGENIERO/ A DE     25   RTDIP      21.521,08 €     12/SI   SI",
            "    INGENIERÍA E INSTALACIONES                                                                    TELECOMUNICACIONES",
            "                                                                                               64 INGENIERO/ A INDUSTRIAL",
            "                                                                                                66 INGENIERO TÉCNICO/ A",
        ])
        puesto, = IMPORTADOR.parse_text(texto)
        self.assertEqual(
            puesto["name"],
            "JEFATURA SECCIÓN DE ASISTENCIA TÉCNICA EN INGENIERÍA E INSTALACIONES",
        )
        self.assertEqual(
            puesto["category_code"],
            "63 INGENIERO/ A DE TELECOMUNICACIONES 64 INGENIERO/ A INDUSTRIAL 66 INGENIERO TÉCNICO/ A",
        )

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
