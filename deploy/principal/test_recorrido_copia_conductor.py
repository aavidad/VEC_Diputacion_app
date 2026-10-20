"""Orden de efectos y rechazo previo de la fábrica del recorrido."""
import argparse
import importlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import types
import unittest
from unittest.mock import Mock, patch

from recorrido_copia import conductor


class ConductorTest(unittest.TestCase):
    def test_fallo_en_acto_detiene_plan_firma_y_provision(self):
        eventos = []
        etapas = {'catalogo': 'preparar', 'roles_admin': 'preparar', 'corporativo': 'preparar', 'certificados_base': 'preparar',
                  'firmantes': 'preparar', 'montajes_firma': 'preparar',
                  'actos': 'preparar_actos', 'plan': 'preparar_plan', 'admisiones': 'preparar',
                  'configuracion_firma': 'preparar'}
        modulos = {}
        for nombre, funcion in etapas.items():
            modulo = types.ModuleType('recorrido_copia.' + nombre)
            def ejecutar(nombre=nombre):
                def llamada(c):
                    eventos.append(nombre)
                    if nombre == 'actos':
                        raise ValueError('Respuesta incierta; conservar recibo')
                    return {'confirmado': True}
                return llamada
            setattr(modulo, funcion, ejecutar())
            modulos[modulo.__name__] = modulo
        paquete = importlib.import_module('recorrido_copia')
        atributos = {n: modulos['recorrido_copia.' + n] for n in etapas}
        with patch.dict(sys.modules, modulos), patch.multiple(paquete, create=True, **atributos), \
             patch.object(conductor, 'Contexto') as contexto, \
             patch.object(conductor, '_parametros', return_value={}), \
             patch.object(conductor, '_organizacion', side_effect=lambda c: eventos.append('organizacion_historica')), \
             patch.object(conductor, 'provisionar_perfiles') as provisionar:
            copia = Mock(nombre='vec-copia-l-contrato')
            contexto.return_value.carpeta = Path('/tmp/recorrido')
            with self.assertRaisesRegex(ValueError, 'incierta'):
                conductor.preparar(copia, argparse.Namespace(puerto_verificador=19881, config_personal=None))
            self.assertEqual(eventos, ['catalogo', 'roles_admin', 'corporativo', 'organizacion_historica', 'certificados_base', 'firmantes', 'montajes_firma', 'actos'])
            copia.firma.assert_not_called()
            provisionar.assert_not_called()

    def test_no_aprueba_perfiles_sin_escucha_confirmada(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            (raiz / 'app.log').write_text('error al componer\n')
            copia = Mock(raiz=raiz)
            with self.assertRaisesRegex(ValueError, 'no confirmó'):
                conductor.provisionar_perfiles(copia)
            copia.parar.assert_not_called()

    def test_creacion_completa_rechaza_material_heredado_antes_de_efectos(self):
        guion = Path(__file__).with_name('copia_local.py')
        resultado = subprocess.run([sys.executable, str(guion), 'crear', '--equipo', 'l',
            '--instancia', 'rechazo', '--recorrido-completo', '--rrhh-original', '/tmp/ajeno'],
            capture_output=True, check=False)
        self.assertEqual(resultado.returncode, 2)
        self.assertIn(b'requiere crear con material nuevo', resultado.stderr)

    def test_puerto_verificador_no_puede_coincidir_con_administracion(self):
        guion = Path(__file__).with_name('copia_local.py')
        resultado = subprocess.run([sys.executable, str(guion), 'crear', '--equipo', 'l',
            '--instancia', 'rechazo', '--recorrido-completo', '--puerto-verificador', '19481'],
            capture_output=True, check=False)
        self.assertEqual(resultado.returncode, 2)
        self.assertIn(b'puerto del verificador debe ser distinto', resultado.stderr)


if __name__ == '__main__':
    unittest.main()
