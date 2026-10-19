"""La semilla de ámbito de bolsas solo traslada lo que declara el manifiesto."""
import importlib.util
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock

spec = importlib.util.spec_from_file_location(
    'ambito', Path(__file__).with_name('copia_local_ambito_bolsas.py'))
ambito = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ambito)

MANIFIESTO = {'unidad_ref': 'unidad:desarrollo:rrhh', 'ambito_ref': 'ambito:desarrollo:bolsa',
              'bolsas_ref': ['bolsa:educador:2026-09-17', 'bolsa:operario:2026-09-17']}


class AmbitoBolsasTest(unittest.TestCase):
    def test_declara_solo_las_bolsas_del_manifiesto_como_propietario(self):
        sql = ambito.sentencias(MANIFIESTO).decode()
        self.assertIn('SET LOCAL ROLE vec_bolsa_llamamientos_propietario;', sql)
        self.assertIn("ARRAY['bolsa:educador:2026-09-17','bolsa:operario:2026-09-17']", sql)
        self.assertIn("'ambito:desarrollo:bolsa','unidad:desarrollo:rrhh','semilla:copia-local:bolsa-bback'", sql)
        # Una bolsa del manifiesto que no está en la base no se declara.
        self.assertIn('WHERE EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida', sql)
        self.assertIn('\\if :b144', sql)

    def test_rechaza_referencias_que_romperian_el_sql(self):
        for campo, valor in (('ambito_ref', "x'); DROP TABLE t; --"), ('unidad_ref', ''),
                             ('bolsas_ref', ['bolsa:a', "b'c"]), ('bolsas_ref', [7])):
            with self.subTest(campo=campo), self.assertRaises(ValueError):
                ambito.sentencias({**MANIFIESTO, campo: valor})

    def test_sin_bolsas_declaradas_no_hay_semilla(self):
        self.assertIsNone(ambito.sentencias({**MANIFIESTO, 'bolsas_ref': []}))

    def test_sin_manifiesto_no_ejecuta_nada(self):
        with tempfile.TemporaryDirectory() as raiz:
            copia = SimpleNamespace(material=Path(raiz), sql_bytes=Mock())
            ambito.declarar(copia)
            copia.sql_bytes.assert_not_called()
            (Path(raiz) / 'identidad').mkdir()
            (Path(raiz) / 'identidad' / 'bolsa-bback.json').write_text(json.dumps(MANIFIESTO))
            ambito.declarar(copia)
            copia.sql_bytes.assert_called_once()
            self.assertEqual(copia.sql_bytes.call_args.args[1], 'ambito-bolsas.log')


if __name__ == '__main__':
    unittest.main()
