import copy
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from generar import generar


RAIZ = Path(__file__).resolve().parent


class VersionCertificadaV2Test(unittest.TestCase):
    def test_fuente_historica_conserva_ambos_archivos_byte_a_byte(self):
        with tempfile.TemporaryDirectory() as temporal:
            salida = Path(temporal) / "salida"
            generar(RAIZ, salida)
            for nombre in ("cargar_version_certificada_v2.sql", "manifiesto.json"):
                self.assertEqual((salida / nombre).read_bytes(),
                                 (RAIZ / "preparado" / nombre).read_bytes(), nombre)

    def test_modalidades_cambian_solo_el_contenido_de_convocatoria(self):
        base = json.loads((RAIZ / "fuente_v2.json").read_text(encoding="utf-8"))
        resultados = {}
        with tempfile.TemporaryDirectory() as temporal:
            temporal = Path(temporal)
            for modalidad in ("separada", "conjunta"):
                fuente = copy.deepcopy(base)
                fuente["convocatoria"]["modalidad_inscripcion"] = modalidad
                ruta = temporal / (modalidad + ".json")
                ruta.write_text(json.dumps(fuente, ensure_ascii=False), encoding="utf-8")
                salida = temporal / modalidad
                subprocess.run((sys.executable, str(RAIZ / "generar.py"),
                                "--fuente", str(ruta), "--salida", str(salida)), check=True)
                sql = (salida / "cargar_version_certificada_v2.sql").read_text(encoding="utf-8")
                manifiesto = json.loads((salida / "manifiesto.json").read_text(encoding="utf-8"))
                self.assertIn('"modalidad_inscripcion":"' + modalidad + '"', sql)
                self.assertEqual(manifiesto["modalidad_inscripcion"], modalidad)
                resultados[modalidad] = manifiesto
        legado = json.loads((RAIZ / "preparado" / "manifiesto.json").read_text(encoding="utf-8"))
        self.assertEqual(resultados["separada"]["politica_v2_sha256"], legado["politica_v2_sha256"])
        self.assertEqual(resultados["conjunta"]["politica_v2_sha256"], legado["politica_v2_sha256"])
        self.assertNotEqual(resultados["separada"]["convocatoria_v2_sha256"],
                            resultados["conjunta"]["convocatoria_v2_sha256"])
        self.assertNotEqual(resultados["separada"]["convocatoria_v2_sha256"],
                            legado["convocatoria_v2_sha256"])

    def test_modalidad_invalida_y_preimagen_externa_fallan_antes_de_emitir(self):
        base = json.loads((RAIZ / "fuente_v2.json").read_text(encoding="utf-8"))
        for nombre, valor in (("desconocida", "otra"), ("nula", None),
                              ("vacia", ""), ("blanco", " ")):
            with self.subTest(nombre=nombre), tempfile.TemporaryDirectory() as temporal:
                temporal = Path(temporal)
                fuente = copy.deepcopy(base)
                fuente["convocatoria"]["modalidad_inscripcion"] = valor
                ruta = temporal / "fuente.json"
                ruta.write_text(json.dumps(fuente), encoding="utf-8")
                salida = temporal / "salida"
                with self.assertRaises(ValueError):
                    generar(RAIZ, salida, ruta)
                self.assertFalse(salida.exists())
        with tempfile.TemporaryDirectory() as temporal:
            temporal = Path(temporal)
            fuente = copy.deepcopy(base)
            fuente["politica"]["archivo_anterior"] = "../politica_v1_canonica.json"
            ruta = temporal / "fuente.json"
            ruta.write_text(json.dumps(fuente), encoding="utf-8")
            salida = temporal / "salida"
            with self.assertRaises(ValueError):
                generar(RAIZ, salida, ruta)
            self.assertFalse(salida.exists())

    def test_claves_duplicadas_en_convocatoria_o_raiz_no_emiten_archivos(self):
        base = json.dumps(json.loads((RAIZ / "fuente_v2.json").read_text(encoding="utf-8")))
        casos = (
            base.replace('"convocatoria": {',
                         '"convocatoria": {"modalidad_inscripcion":"separada",'
                         '"modalidad_inscripcion":"conjunta",', 1),
            '{"esquema":"otro",' + base[1:],
        )
        for texto in casos:
            with self.subTest(fuente=texto[:45]), tempfile.TemporaryDirectory() as temporal:
                temporal = Path(temporal)
                ruta = temporal / "fuente.json"
                ruta.write_text(texto, encoding="utf-8")
                salida = temporal / "salida"
                with self.assertRaisesRegex(ValueError, "clave duplicada"):
                    generar(RAIZ, salida, ruta)
                self.assertFalse(salida.exists())


if __name__ == "__main__":
    unittest.main()
