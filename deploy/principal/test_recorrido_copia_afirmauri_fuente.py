"""Focales de afirmauri con ejecutables ficticios sin URI, GUI ni red."""

import hashlib
import io
import os
from pathlib import Path
import tempfile
import unittest
from contextlib import redirect_stderr
from unittest import mock

import copia_local
from recorrido_copia.afirmauri_fuente import validar


class AfirmauriFuenteTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(dir=os.environ.get('VEC_TEST_SCRATCH'))
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.base.chmod(0o700)
        self.binario = self.base / 'grxfirma-afirmauri'
        self.marca = self.base / 'argumentos'

    def crear(self, identidad='grxfirma-afirmauri dev'):
        self.binario.write_text(
            '#!/usr/bin/python3\n'
            'import pathlib,sys\n'
            f'pathlib.Path({str(self.marca)!r}).write_text(repr(sys.argv[1:]))\n'
            "if sys.argv[1:] != ['--version']: sys.exit(2)\n"
            f'print({identidad!r}, file=sys.stderr)\n')
        self.binario.chmod(0o700)
        return hashlib.sha256(self.binario.read_bytes()).hexdigest()

    def copia(self):
        copia = object.__new__(copia_local.Copia)
        copia.raiz = self.base / 'copia'
        copia.raiz.mkdir(mode=0o700)
        return copia

    def test_dev_775_con_sha_copia_propia_sin_cambiar_origen(self):
        sha = self.crear()
        self.binario.chmod(0o775)
        estado = self.copia().preparar_afirmauri(self.binario, sha)
        self.assertEqual(estado['version'], 'dev')
        self.assertEqual(estado['binario_sha256'], sha)
        self.assertEqual(Path(estado['binario']).stat().st_mode & 0o777, 0o700)
        self.assertEqual(self.binario.stat().st_mode & 0o777, 0o775)
        self.assertEqual(self.marca.read_text(), "['--version']")

    def test_dev_sin_sha_y_sha_distinta_no_publican_binario(self):
        self.crear()
        copia = self.copia()
        with self.assertRaisesRegex(ValueError, 'afirmauri_dev_sin_huella'):
            copia.preparar_afirmauri(self.binario, None)
        self.assertFalse((copia.raiz / 'software-afirmauri/grxfirma-afirmauri').exists())
        self.marca.unlink()
        with self.assertRaisesRegex(ValueError, 'Huella declarada del software divergente'):
            copia.preparar_afirmauri(self.binario, '0' * 64)
        self.assertFalse(self.marca.exists())

    def test_binario_verificador_no_suplanta_identidad_afirmauri(self):
        sha = self.crear('grxfirma dev')
        with self.assertRaisesRegex(ValueError, 'afirmauri_identidad_invalida'):
            validar(self.binario, sha)

    def test_flags_incompletas_cortan_antes_de_crear_copia(self):
        casos = (
            ['crear', '--equipo', 'l', '--instancia', 'focal', '--sha-afirmauri', 'a' * 64],
            ['crear', '--equipo', 'l', '--instancia', 'focal', '--binario-afirmauri', 'relativo'],
            ['crear', '--equipo', 'l', '--instancia', 'focal', '--binario-afirmauri', str(self.binario),
             '--sha-afirmauri', 'mal'],
            ['crear', '--equipo', 'l', '--instancia', 'focal', '--recorrido-completo',
             '--binario-verificador', '/ruta/verificador', '--binario-afirmauri', str(self.binario)],
            ['crear', '--equipo', 'l', '--instancia', 'focal', '--recorrido-completo',
             '--binario-verificador', '/ruta/verificador', '--binario-afirmauri', str(self.binario),
             '--sha-afirmauri', 'mal'],
        )
        with mock.patch.object(copia_local, 'Copia') as copia:
            for argv in casos:
                with self.subTest(argv=argv), mock.patch('sys.argv', ['copia_local.py', *argv]), \
                        redirect_stderr(io.StringIO()):
                    with self.assertRaises(SystemExit) as error:
                        copia_local.main()
                    self.assertEqual(error.exception.code, 2)
            copia.assert_not_called()
