"""La parada del buzón exige propiedad y la copia básica no lo necesita."""
import importlib.util
import json
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

spec = importlib.util.spec_from_file_location('correo', Path(__file__).with_name('copia_local_correo.py'))
correo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(correo)


class CorreoTest(unittest.TestCase):
    def setUp(self):
        self.copia = SimpleNamespace(nombre='vec-copia-l-focal', raiz=Path('/copia'),
                                     leer=Mock(return_value={'puerto_smtp': 11081}), docker=Mock())

    def test_no_detiene_un_buzon_con_marca_ajena(self):
        respuesta = SimpleNamespace(returncode=0, stdout=json.dumps([{
            'Config': {'Labels': {'vec.copia': '/otra', 'vec.funcion': 'correo-local'}}}]).encode())
        with patch.object(correo.subprocess, 'run', return_value=respuesta), self.assertRaises(ValueError):
            correo.parar(self.copia)
        self.copia.docker.assert_not_called()

    def test_parada_usa_solo_el_nombre_propio_validado(self):
        respuesta = SimpleNamespace(returncode=0, stdout=json.dumps([{
            'Config': {'Labels': {'vec.copia': '/copia', 'vec.funcion': 'correo-local'}}}]).encode())
        with patch.object(correo.subprocess, 'run', return_value=respuesta):
            correo.parar(self.copia)
        self.assertEqual(self.copia.docker.call_args.args,
                         ('stop', '--time', '10', 'vec-copia-l-focal-correo'))

    def test_copia_basica_no_crea_buzon(self):
        self.copia.leer.return_value = {}
        with patch.object(correo.subprocess, 'run') as run:
            self.assertEqual(correo.arrancar(self.copia), {})
            correo.parar(self.copia)
        run.assert_not_called()
        self.copia.docker.assert_not_called()
