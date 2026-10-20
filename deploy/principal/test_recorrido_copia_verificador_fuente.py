"""Focales de preflight GrxFirma sin red, credenciales ni servicios."""

import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from types import SimpleNamespace

from copia_local import Copia
from recorrido_copia import conductor
from recorrido_copia.verificador_fuente import validar


class VerificadorFuenteTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(dir=os.environ.get('VEC_TEST_SCRATCH'))
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.base.chmod(0o700)
        self.binario = self.base / 'grxfirma'
        self.marca = self.base / 'ejecutado'

    def crear(self, version):
        script = ("#!/usr/bin/python3\n"
                  "import pathlib,sys\n"
                  f"pathlib.Path({str(self.marca)!r}).write_text('si')\n"
                  "if sys.argv[1:] == ['--version']:\n"
                  f" print('grxfirma {version}')\n"
                  "elif sys.argv[1:] == ['-rest-solo-verificacion', '--help']:\n"
                  " print('Uso: grxfirma -rest-solo-verificacion --help')\n"
                  "else: sys.exit(2)\n")
        self.binario.write_text(script)
        self.binario.chmod(0o700)
        return hashlib.sha256(self.binario.read_bytes()).hexdigest()

    def test_rechaza_105_antes_de_ayuda(self):
        self.crear('0.0.105')
        with self.assertRaisesRegex(ValueError, 'grxfirma_version_antigua'):
            validar(self.binario, None)

    def test_acepta_111_con_flag_y_sha_estable(self):
        sha = self.crear('0.0.111')
        self.assertEqual(validar(self.binario, sha), {
            'binario': 'grxfirma', 'binario_sha256': sha, 'version': '0.0.111'})
        self.assertTrue(self.marca.exists())

    def test_sha_distinta_corta_antes_de_ejecutar(self):
        self.crear('0.0.111')
        with self.assertRaisesRegex(ValueError, 'grxfirma_huella_distinta'):
            validar(self.binario, '0' * 64)
        self.assertFalse(self.marca.exists())

    def test_dev_exige_sha_explicita_y_conserva_version(self):
        sha = self.crear('dev')
        with self.assertRaisesRegex(ValueError, 'grxfirma_dev_sin_huella'):
            validar(self.binario, None)
        self.assertEqual(validar(self.binario, sha)['version'], 'dev')

    def test_fabrica_copia_fuente_grupo_escritura_con_sha_y_conductor_usa_propia(self):
        sha = self.crear('dev')
        self.binario.chmod(0o775)
        copia = object.__new__(Copia)
        copia.raiz = self.base / 'copia'
        copia.raiz.mkdir(mode=0o700)
        software = copia.preparar_verificador(self.binario, sha)
        self.assertEqual(software['version'], 'dev')
        self.assertEqual(software['binario_sha256'], sha)
        self.assertEqual(self.binario.stat().st_mode & 0o777, 0o775)
        self.assertEqual(Path(software['binario']).stat().st_mode & 0o777, 0o700)
        recorrido = copia.raiz / 'recorrido'
        recorrido.mkdir(mode=0o700)

        def guardar(path, data):
            fd = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
            with os.fdopen(fd, 'w') as out:
                json.dump(data, out)

        c = SimpleNamespace(raiz=copia.raiz, carpeta=recorrido,
                            copia=SimpleNamespace(leer=lambda: {'software_verificador': software}),
                            sha=lambda path: hashlib.sha256(Path(path).read_bytes()).hexdigest(),
                            guardar=guardar)
        manifest = conductor._verificador(c, 19881)
        doc = json.loads(manifest.read_text())
        self.assertEqual(doc['verificador']['binario_sha256'], sha)
        self.assertEqual(doc['verificador']['binario'], str(recorrido / 'grxfirma'))
        self.assertEqual((recorrido / 'grxfirma').stat().st_mode & 0o777, 0o700)

    def test_fabrica_rechaza_fuente_grupo_escritura_sin_sha(self):
        self.crear('dev')
        self.binario.chmod(0o775)
        copia = object.__new__(Copia)
        copia.raiz = self.base / 'copia'
        copia.raiz.mkdir(mode=0o700)
        with self.assertRaisesRegex(ValueError, 'sin huella declarada'):
            copia.preparar_verificador(self.binario, None)
        self.assertFalse((copia.raiz / 'software-verificador').exists())
