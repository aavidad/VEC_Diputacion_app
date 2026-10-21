"""Activación sobre una copia congelada y recuperación del entorno anterior."""

import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock

from copia_local_necesidades import activar, entorno_necesidades, VARIABLE


class NecesidadesTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.raiz = Path(self.tmp.name)
        self.repo = Path(__file__).resolve().parents[2]
        self.estado = {'preparada': True, 'recorrido_completo': True,
                       'revision': 'congelada', 'binario_sha256': 'original'}
        self.env = self.raiz / 'entorno_modulos.json'
        self.env.write_text(json.dumps({'VEC_RPT_CATALOGO_PATH': 'rpt-propia'}))
        self.copia = SimpleNamespace(raiz=self.raiz, estado=self.raiz / 'estado.json',
            leer=lambda: dict(self.estado), validar_binario=Mock(),
            entorno_modulos=lambda: json.loads(self.env.read_text()),
            parar=Mock(), arrancar=Mock(), comprobar=Mock())
        def guardar(ruta, datos):
            ruta.write_text(json.dumps(datos))
            if ruta == self.copia.estado:
                self.estado.update(datos)
        self.nativo = SimpleNamespace(REPO=self.repo, verificar_fuente=Mock(),
                                      guardar=guardar, sha256=lambda _: 'huella-real')

    def test_conserva_fuente_binario_y_otras_variables(self):
        activar(self.copia, self.nativo)
        entorno = json.loads(self.env.read_text())
        self.assertEqual(entorno['VEC_RPT_CATALOGO_PATH'], 'rpt-propia')
        self.assertEqual(entorno[VARIABLE], entorno_necesidades(self.repo)[VARIABLE])
        self.assertEqual(self.estado['revision'], 'congelada')
        self.assertEqual(self.estado['binario_sha256'], 'original')
        self.copia.arrancar.assert_called_once()
        self.copia.comprobar.assert_called_once()

    def test_fallo_de_arranque_recupera_entorno_y_devuelve_error(self):
        self.copia.arrancar.side_effect = [RuntimeError('arranque rechazado'), None]
        with self.assertRaisesRegex(RuntimeError, 'arranque rechazado'):
            activar(self.copia, self.nativo)
        self.assertEqual(json.loads(self.env.read_text()), {'VEC_RPT_CATALOGO_PATH': 'rpt-propia'})
        self.assertEqual(self.copia.arrancar.call_count, 2)
        self.copia.comprobar.assert_called_once()

    def test_no_sustituye_otra_fuente(self):
        self.env.write_text(json.dumps({VARIABLE: 'otra-fuente'}))
        with self.assertRaisesRegex(ValueError, 'otra fuente'):
            activar(self.copia, self.nativo)
        self.copia.parar.assert_not_called()

    def test_repeticion_solo_comprueba_sin_otro_reinicio(self):
        self.env.write_text(json.dumps(entorno_necesidades(self.repo)))
        activar(self.copia, self.nativo)
        self.copia.parar.assert_not_called()
        self.copia.comprobar.assert_called_once()


if __name__ == '__main__':
    unittest.main()
