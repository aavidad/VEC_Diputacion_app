from __future__ import annotations

import filecmp
import json
import re
import tempfile
import unittest
from pathlib import Path

from scripts.paquete_ejemplo import catalogos
from scripts.paquete_ejemplo.generar import CABECERAS_VEC, HOJAS, generar
from scripts.paquete_ejemplo.personas import documento_en_rango_reservado, documento_valido, enmascarar

try:
    import xlrd  # type: ignore
    import xlwt  # type: ignore  # noqa: F401
except ImportError:  # pragma: no cover - el ejecutor sin xlwt/xlrd omite estas pruebas
    xlrd = None

# Palabras completas: «comprueba» o «protesta» no delatan nada; «Usuario de prueba» sí.
PROHIBIDAS = re.compile(r"\b(?:pruebas?|test\w*|sint[eé]tic\w*|usuari[oa]s?|ejemplos?|demo\w*|fictici\w*|fake|dummy)\b",
                        re.IGNORECASE)


@unittest.skipIf(xlrd is None, "xlwt/xlrd no instalados")
class PaqueteEjemploTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls._tmp = tempfile.TemporaryDirectory()
        cls.raiz = Path(cls._tmp.name)
        cls.manifiesto = generar(cls.raiz / "a", bolsas=4, minimo=20, maximo=30, expedientes=6)
        cls.personas = json.loads((cls.raiz / "a" / "personas.json").read_text(encoding="utf-8"))["personas"]
        cls.identidades = json.loads((cls.raiz / "a" / "identidades.json").read_text(encoding="utf-8"))["identidades"]
        cls.expedientes = json.loads(
            (cls.raiz / "a" / "contratacion" / "expedientes.json").read_text(encoding="utf-8"))["expedientes"]

    @classmethod
    def tearDownClass(cls) -> None:
        cls._tmp.cleanup()

    def test_determinista(self) -> None:
        generar(self.raiz / "b", bolsas=4, minimo=20, maximo=30, expedientes=6)
        comparacion = filecmp.dircmp(self.raiz / "a", self.raiz / "b")
        pendientes = [comparacion]
        while pendientes:
            actual = pendientes.pop()
            _, distintos, errores = filecmp.cmpfiles(actual.left, actual.right, actual.common_files, shallow=False)
            self.assertEqual((distintos, errores, actual.left_only, actual.right_only), ([], [], [], []))
            pendientes.extend(actual.subdirs.values())

    def test_otra_semilla_cambia_los_datos(self) -> None:
        otro = generar(self.raiz / "c", semilla=7, bolsas=4, minimo=20, maximo=30, expedientes=6)
        self.assertNotEqual(otro["ficheros"], self.manifiesto["ficheros"])

    def test_documentos_validos_en_rango_reservado_y_unicos(self) -> None:
        documentos = [p["documento"] for p in self.personas]
        self.assertEqual(len(documentos), len(set(documentos)))
        mascaras = [p["documento_enmascarado"] for p in self.personas]
        self.assertEqual(len(mascaras), len(set(mascaras)))
        for p in self.personas:
            self.assertTrue(documento_valido(p["documento"]), p["documento"])
            self.assertTrue(documento_en_rango_reservado(p["documento"]), p["documento"])
            self.assertRegex(p["documento_enmascarado"], r"^\*\*\*\d{4}\*\*$")
            self.assertEqual(p["documento_enmascarado"], enmascarar(p["documento"]))
        self.assertTrue(any(p["tipo_documento"] == "NIE" for p in self.personas))

    def test_letra_de_control(self) -> None:
        self.assertTrue(documento_valido("12345678Z"))
        self.assertTrue(documento_valido("X1234567L"))
        self.assertFalse(documento_valido("12345678A"))
        self.assertFalse(documento_valido("99999999"))

    def test_sin_duplicados_en_nombres_correos_y_telefonos(self) -> None:
        for campo in ("nombre_completo", "correo", "telefono", "referencia"):
            valores = [p[campo] for p in self.personas]
            self.assertEqual(len(valores), len(set(valores)), campo)
        for p in self.personas:
            self.assertNotEqual(p["primer_apellido"], p["segundo_apellido"])
            self.assertTrue(p["correo"].endswith("@correo.internal"))
            self.assertRegex(p["telefono"], r"^79\d \d{3} \d{3}$")
            self.assertEqual(p["domicilio"]["provincia"], "Granada")
            self.assertTrue(1960 <= int(p["fecha_nacimiento"][:4]) <= 2005, p["fecha_nacimiento"])

    def test_listas_sin_repeticiones(self) -> None:
        for lista in (catalogos.NOMBRES_MUJER, catalogos.NOMBRES_HOMBRE, catalogos.APELLIDOS, catalogos.VIAS):
            self.assertEqual(len(lista), len(set(lista)))

    def _textos_visibles(self):
        for p in self.personas:
            yield from (p["nombre_completo"], p["correo"], *p["domicilio"].values())
        for i in self.identidades:
            yield from (i["display_name"], i["cn"], i["cargo"], i["correo"])
        for e in self.expedientes:
            yield from (e["detalle"], e["observaciones"], e["observaciones_analisis"])
        for xls in sorted((self.raiz / "a" / "convoca").rglob("*.xls")):
            hoja = xlrd.open_workbook(str(xls)).sheet_by_index(0)
            for fila in range(hoja.nrows):
                yield from (str(v) for v in hoja.row_values(fila))

    def test_sin_palabras_de_prueba_en_campos_visibles(self) -> None:
        for delator in ("Usuario de prueba", "Persona sintética 1", "test", "Candidato DEMO"):
            self.assertIsNotNone(PROHIBIDAS.search(delator), delator)
        for texto in self._textos_visibles():
            self.assertIsNone(PROHIBIDAS.search(texto), texto)

    def test_xls_con_formato_convoca_y_totales_coherentes(self) -> None:
        for bolsa in self.manifiesto["bolsas"]:
            carpeta = self.raiz / "a" / "convoca" / bolsa["categoria"]
            libro = xlrd.open_workbook(str(carpeta / "resumen.xls"))
            self.assertEqual(libro.sheet_names(), [HOJAS["resumen"]])
            resumen = libro.sheet_by_index(0)
            self.assertEqual(resumen.row_values(0), CABECERAS_VEC["resumen"])
            self.assertEqual(resumen.nrows - 1, bolsa["candidaturas"])
            totales = [resumen.row_values(f)[7] for f in range(1, resumen.nrows)]
            self.assertEqual(totales, sorted(totales, reverse=True))
            por_persona = {}
            for f in range(1, resumen.nrows):
                fila = resumen.row_values(f)
                self.assertAlmostEqual(fila[5] + fila[6], fila[7], places=2)
                por_persona[fila[0]] = fila
            detalle = xlrd.open_workbook(str(carpeta / "detalle.xls")).sheet_by_index(0)
            self.assertEqual(detalle.row_values(0), CABECERAS_VEC["detalle"])
            sumas = {}
            for f in range(1, detalle.nrows):
                fila = detalle.row_values(f)
                clave = (fila[0], fila[5])
                sumas[clave] = sumas.get(clave, 0.0) + fila[10]
            for documento, fila in por_persona.items():
                self.assertAlmostEqual(sumas[(documento, "EXP")], fila[5], places=2)
                self.assertAlmostEqual(sumas[(documento, "FOR")], fila[6], places=2)

    def test_manifiesto_marca_el_paquete(self) -> None:
        self.assertEqual(self.manifiesto["paquete"], "paquete:ejemplo:vec:v1")
        for bolsa in self.manifiesto["bolsas"]:
            self.assertTrue(bolsa["bolsa_ref"].endswith(":ejemplo-vec-v1"))
        for p in self.personas:
            self.assertTrue(p["referencia"].startswith("persona:ejemplo-vec-v1:"))
        for ruta, huella in self.manifiesto["ficheros"].items():
            self.assertRegex(huella, r"^[0-9a-f]{64}$", ruta)

    def test_identidades_con_cn_legible(self) -> None:
        roles = {i["rol"] for i in self.identidades}
        self.assertTrue({"tecnico_rrhh", "intervencion", "candidato_bolsa", "empleado"} <= roles)
        for i in self.identidades:
            self.assertRegex(i["cn"], r"^[A-Z ]+ - (99\d{6}|Z9\d{6})[A-Z]$")
        candidato = next(i for i in self.identidades if i["rol"] == "candidato_bolsa")
        inscritas = {r for b in self.manifiesto["bolsas"] for r in b["personas_ref"]}
        self.assertIn(candidato["persona_ref"], inscritas)


if __name__ == "__main__":
    unittest.main()
