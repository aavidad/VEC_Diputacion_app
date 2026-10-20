"""El manifiesto generado conserva aislamiento y recuperación de las etapas."""

import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from copia_local_firma import leer_manifiesto
from copia_local_firma_generacion import ETAPAS, generar, retirar


class CopiaPrueba:
    def __init__(self, raiz):
        self.raiz = raiz
        self.nombre = 'vec-copia-l-generacion-firma'

    def leer(self):
        return {'revision': 'a' * 40, 'puerto_pg': 55417, 'puerto_app': 18417,
                'puerto_smtp': 11017}


class GeneracionFirmaTest(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporal.cleanup)
        self.base = Path(self.temporal.name)
        self.copia = CopiaPrueba(self.base / 'instancia')
        self.copia.raiz.mkdir(mode=0o700)
        self.repo = self.base / 'repo'
        self.fuentes = self.repo / 'deploy/principal/firma_copia'
        self.fuentes.mkdir(parents=True, mode=0o700)
        (self.fuentes / 'ejecutar.py').write_text('raise SystemExit(0)\n')
        for nombre in ('comun', 'administracion', 'montajes', 'firmantes',
                       'publicaciones', 'plan', 'configuracion'):
            (self.fuentes / (nombre + '.py')).write_text('# fuente de la etapa\n')
        (self.fuentes / 'datos.json').write_text('{"version":1}\n')
        self.verificador = self.base / 'verificador'
        self.verificador.write_bytes(b'#!/bin/sh\nexit 0\n')
        self.verificador.chmod(0o700)

    def generar(self, **extra):
        return generar(self.copia, repo=self.repo,
                       verificador=extra.pop('verificador', self.verificador),
                       directorio=self.base / 'estado', **extra)

    def test_contrato_vigente_y_repeticion_conservan_fuentes(self):
        ruta = self.generar()
        contenido = ruta.read_bytes()
        inode = ruta.stat().st_ino
        manifiesto = leer_manifiesto(ruta, self.copia.leer())
        self.assertEqual(tuple(e['nombre'] for e in manifiesto['etapas']), ETAPAS)
        for etapa in manifiesto['etapas']:
            lanzador = Path(etapa['orden'][0])
            self.assertTrue(lanzador.is_relative_to(self.base / 'estado'))
            self.assertEqual(lanzador.stat().st_mode & 0o777, 0o700)
            self.assertIn('-I', lanzador.read_text())
        copia_verificador = Path(manifiesto['verificador']['binario'])
        self.assertEqual(copia_verificador.read_bytes(), self.verificador.read_bytes())
        self.assertEqual(self.generar(), ruta)
        self.assertEqual(ruta.read_bytes(), contenido)
        self.assertEqual(ruta.stat().st_ino, inode)
        fuentes = json.loads((ruta.parent / 'fuentes.json').read_text())
        self.assertEqual(fuentes['revision'], self.copia.leer()['revision'])
        self.assertEqual(fuentes['archivos']['datos.json'],
                         hashlib.sha256((self.fuentes / 'datos.json').read_bytes()).hexdigest())

    def test_no_sustituye_fuentes_ya_conservadas(self):
        ruta = self.generar()
        anterior = (ruta.parent / 'fuentes/datos.json').read_bytes()
        (self.fuentes / 'datos.json').write_text('{"version":2}\n')
        with self.assertRaisesRegex(ValueError, 'divergente'):
            self.generar()
        self.assertEqual((ruta.parent / 'fuentes/datos.json').read_bytes(), anterior)

    def test_rechaza_herramienta_enlazada_sin_manifestar_etapas(self):
        enlace = self.base / 'enlace'
        enlace.symlink_to(self.verificador)
        with self.assertRaisesRegex(ValueError, 'verificador_instalado_invalido'):
            self.generar(verificador=enlace)
        self.assertFalse((self.base / 'estado/manifiesto.json').exists())

    def test_rechaza_fuente_enlazada(self):
        (self.fuentes / 'externa.py').symlink_to(self.verificador)
        with self.assertRaisesRegex(ValueError, 'versionada_enlazada'):
            self.generar()

    def test_borrar_no_retiro_lanzadores_de_otra_copia(self):
        ruta = self.generar()
        self.copia.nombre = 'vec-copia-l-ajena'
        with self.assertRaisesRegex(ValueError, 'otra_copia'):
            retirar(self.copia, directorio=ruta.parent)
        self.assertTrue(ruta.exists())
        self.copia.nombre = 'vec-copia-l-generacion-firma'
        retirar(self.copia, directorio=ruta.parent)
        self.assertFalse(ruta.parent.exists())

    def test_puerto_verificador_de_la_instancia(self):
        manifiesto = leer_manifiesto(self.generar(), self.copia.leer())
        self.assertEqual(manifiesto['verificador']['puerto'], 58417)
        with self.assertRaisesRegex(ValueError, 'puerto_verificador_invalido'):
            self.generar(puerto=65536)

    def test_fuentes_incompletas_no_crean_lanzadores(self):
        (self.fuentes / 'administracion.py').unlink()
        with self.assertRaisesRegex(ValueError, 'fuente_versionada_ausente:administracion'):
            self.generar()
        self.assertFalse((self.base / 'estado').exists())


if __name__ == '__main__':
    unittest.main()
